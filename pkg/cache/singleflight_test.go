package cache

import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDo_SingleCall 无并发场景下 fn 执行一次 结果正确
func TestDo_SingleCall(t *testing.T) {
	g := NewGroup()
	v, err, shared := Do(g, "k", func() (string, error) { return "hello", nil })
	require.NoError(t, err)
	assert.Equal(t, "hello", v)
	assert.False(t, shared, "无并发时 shared=false")
}

// TestDo_ConcurrentSameKey 同 key 并发 fn 仅执行一次 全部等待者拿到同一结果
func TestDo_ConcurrentSameKey(t *testing.T) {
	g := NewGroup()
	var execCount int32

	const n = 100
	results := make([]int, n)
	var wg sync.WaitGroup
	wg.Add(n)
	start := make(chan struct{})

	for i := 0; i < n; i++ {
		idx := i
		go func() {
			defer wg.Done()
			<-start
			v, err, _ := Do(g, "user:1", func() (int, error) {
				atomic.AddInt32(&execCount, 1)
				time.Sleep(20 * time.Millisecond)
				return 42, nil
			})
			require.NoError(t, err)
			results[idx] = v
		}()
	}
	close(start)
	wg.Wait()

	assert.Equal(t, int32(1), atomic.LoadInt32(&execCount), "fn 同 key 应只执行一次")
	for _, v := range results {
		assert.Equal(t, 42, v)
	}
}

// TestDo_ConcurrentDifferentKey 不同 key 并发互不阻塞 fn 各自执行
func TestDo_ConcurrentDifferentKey(t *testing.T) {
	g := NewGroup()
	var execCount int32

	const n = 10
	var wg sync.WaitGroup
	wg.Add(n)
	for i := 0; i < n; i++ {
		idx := i
		go func() {
			defer wg.Done()
			_, _, _ = Do(g, "user:"+itoa(idx), func() (int, error) {
				atomic.AddInt32(&execCount, 1)
				return idx, nil
			})
		}()
	}
	wg.Wait()
	assert.Equal(t, int32(n), atomic.LoadInt32(&execCount), "不同 key 各自执行")
}

// TestDo_ErrorPropagates fn 失败时所有等待者收到同一 err
func TestDo_ErrorPropagates(t *testing.T) {
	g := NewGroup()
	wantErr := errors.New("boom")

	const n = 20
	var wg sync.WaitGroup
	wg.Add(n)
	errs := make([]error, n)
	start := make(chan struct{})

	for i := 0; i < n; i++ {
		idx := i
		go func() {
			defer wg.Done()
			<-start
			_, err, _ := Do(g, "k", func() (int, error) {
				time.Sleep(10 * time.Millisecond)
				return 0, wantErr
			})
			errs[idx] = err
		}()
	}
	close(start)
	wg.Wait()

	for _, e := range errs {
		assert.ErrorIs(t, e, wantErr)
	}
}

// TestDo_ForgetAllowsImmediateRetry Forget 后立即重试 fn 重新执行
func TestDo_ForgetAllowsImmediateRetry(t *testing.T) {
	g := NewGroup()
	var execCount int32

	_, _, _ = Do(g, "k", func() (int, error) {
		atomic.AddInt32(&execCount, 1)
		return 1, nil
	})
	g.Forget("k")
	_, _, _ = Do(g, "k", func() (int, error) {
		atomic.AddInt32(&execCount, 1)
		return 2, nil
	})
	assert.Equal(t, int32(2), atomic.LoadInt32(&execCount))
}

// itoa 简易 int → string 避免引入 strconv 让测试更紧凑
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	digits := []byte{}
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	return string(digits)
}
