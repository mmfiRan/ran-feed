package xxljob

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"

	"github.com/zeromicro/go-zero/core/threading"
)

type Executor struct {
	cfg        Config
	admin      *AdminClient
	runner     *taskRunner
	server     *http.Server
	logHandler func(param LogParam) (LogResult, error)
	logger     Logger
	ctxBuilder func(r *http.Request, param *TriggerParam) context.Context
	mu         sync.Mutex
	started    bool
}

func NewExecutor(cfg Config) *Executor {
	cfg.normalize()
	return &Executor{
		cfg:    cfg,
		admin:  NewAdminClient(cfg.AdminAddresses, cfg.HTTPTimeout, cfg.AccessToken),
		runner: newTaskRunner(),
	}
}

type Routes struct {
	Run      http.HandlerFunc
	Beat     http.HandlerFunc
	IdleBeat http.HandlerFunc
	Kill     http.HandlerFunc
	Log      http.HandlerFunc
}

func (e *Executor) RegisterTask(name string, handler TaskHandler) {
	e.runner.register(name, handler)
}

func (e *Executor) Use(mw TaskMiddleWare) {
	e.runner.use(mw)
}

func (e *Executor) SetLogHandler(h func(param LogParam) (LogResult, error)) {
	e.logHandler = h
}

// SetLogger 设置日志实现（默认使用 go-zero 的 logx）。
func (e *Executor) SetLogger(l Logger) {
	e.logger = l
}

// SetContextBuilder 注入上下文构造器，用于绑定 trace 等信息
// 传 nil 表示使用请求的原始 context
func (e *Executor) SetContextBuilder(builder func(r *http.Request, param *TriggerParam) context.Context) {
	e.ctxBuilder = builder
}

func (e *Executor) Routes() Routes {
	return Routes{
		Run:      e.handleRun,
		Beat:     e.handleBeat,
		IdleBeat: e.handleIdle,
		Kill:     e.handleKill,
		Log:      e.handleLog,
	}
}

func (e *Executor) Handler() http.Handler {
	mux := http.NewServeMux()
	r := e.Routes()
	mux.HandleFunc("/run", r.Run)
	mux.HandleFunc("/beat", r.Beat)
	mux.HandleFunc("/idleBeat", r.IdleBeat)
	mux.HandleFunc("/kill", r.Kill)
	mux.HandleFunc("/log", r.Log)
	return mux
}

func (e *Executor) Start(ctx context.Context) error {
	e.mu.Lock()
	if e.started {
		e.mu.Unlock()
		return nil
	}
	e.started = true
	e.mu.Unlock()

	listenAddr := e.listenAddr()
	e.server = &http.Server{
		Addr:    listenAddr,
		Handler: e.Handler(),
	}

	if len(e.cfg.AdminAddresses) > 0 {
		threading.GoSafe(func() {
			e.registryLoop(ctx)
		})
	}

	ln, err := net.Listen("tcp", listenAddr)
	if err != nil {
		return err
	}
	threading.GoSafe(func() {
		<-ctx.Done()
		e.Shutdown(context.Background())
	})
	return e.server.Serve(ln)
}

func (e *Executor) Shutdown(ctx context.Context) error {
	e.mu.Lock()
	if !e.started {
		e.mu.Unlock()
		return nil
	}
	e.started = false
	e.mu.Unlock()
	if e.server != nil {
		return e.server.Shutdown(ctx)
	}
	return nil
}

func (e *Executor) registryLoop(ctx context.Context) {
	param := RegistryParam{
		RegistryGroup: "EXECUTOR",
		RegistryKey:   e.cfg.AppName,
		RegistryValue: e.registryValue(),
	}
	fmt.Println(param)
	_ = e.admin.Register(ctx, param)
	ticker := timeTicker(e.cfg.RegistryInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			_ = e.admin.Unregister(context.Background(), param)
			return
		case <-ticker.C:
			_ = e.admin.Register(context.Background(), param)
		}
	}
}

func (e *Executor) registryValue() string {
	if e.cfg.IP != "" {
		return "http://" + e.cfg.Address
	}
	return "http://" + net.JoinHostPort(e.cfg.IP, itoa(e.cfg.Port))
}

func (e *Executor) listenAddr() string {
	addr := strings.TrimSpace(e.cfg.Address)
	if addr == "" {
		return net.JoinHostPort(e.cfg.IP, itoa(e.cfg.Port))
	}
	if strings.Contains(addr, "://") {
		u, err := url.Parse(addr)
		if err == nil && u.Host != "" {
			return u.Host
		}
	}
	return addr
}

func (e *Executor) authorize(r *http.Request) bool {
	if e.cfg.AccessToken == "" {
		return true
	}
	return r.Header.Get(HeaderAccessToken) == e.cfg.AccessToken || r.Header.Get(AltHeaderToken) == e.cfg.AccessToken
}

func (e *Executor) handleRun(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if !e.authorize(r) {
		logError(ctx, e.logger, "xxljob: unauthorized run request")
		writeJSON(w, Fail("invalid access token"))
		return
	}
	payload, err := io.ReadAll(r.Body)
	if err != nil {
		logError(ctx, e.logger, "xxljob: read body error: %v", err)
		writeJSON(w, Fail(err.Error()))
		return
	}
	param, err := decodeTriggerParam(payload)
	if err != nil {
		logError(ctx, e.logger, "xxljob: decode trigger error: %v", err)
		writeJSON(w, Fail(err.Error()))
		return
	}
	logInfo(
		ctx,
		e.logger,
		"xxljob: received run request remote=%s handler=%s jobId=%d logId=%d block=%s timeout=%ds glue=%s broadcast=%d/%d params=%q raw=%s",
		r.RemoteAddr,
		param.ExecutorHandler,
		param.JobID,
		param.LogID,
		param.ExecutorBlockStrategy,
		param.ExecutorTimeout,
		param.GlueType,
		param.BroadcastIndex,
		param.BroadcastTotal,
		shortenForLog(param.ExecutorParams, 256),
		shortenForLog(string(payload), 512),
	)
	if strings.TrimSpace(param.ExecutorHandler) == "" {
		logError(ctx, e.logger, "xxljob: empty executorHandler")
		writeJSON(w, Fail("executorHandler required"))
		return
	}
	execCtx := context.WithoutCancel(ctx)
	if e.ctxBuilder != nil {
		execCtx = e.ctxBuilder(r, &param)
		if execCtx == nil {
			execCtx = context.Background()
		} else {
			execCtx = context.WithoutCancel(execCtx)
		}
	}
	if err := e.runner.start(execCtx, param, e.admin, e.logger); err != nil {
		logError(execCtx, e.logger, "xxljob: start handler=%s jobId=%d logId=%d error=%v", param.ExecutorHandler, param.JobID, param.LogID, err)
		writeJSON(w, Fail(err.Error()))
		return
	}
	logInfo(execCtx, e.logger, "xxljob: started handler=%s jobId=%d logId=%d", param.ExecutorHandler, param.JobID, param.LogID)
	writeJSON(w, Ok("success"))
}

func (e *Executor) handleBeat(w http.ResponseWriter, r *http.Request) {
	if !e.authorize(r) {
		writeJSON(w, Fail("invalid access token"))
		return
	}
	writeJSON(w, Ok("success"))
}

func (e *Executor) handleIdle(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if !e.authorize(r) {
		logError(ctx, e.logger, "xxljob: unauthorized idleBeat request")
		writeJSON(w, Fail("invalid access token"))
		return
	}
	var param IdleBeatParam
	if err := decodeJSON(r, &param); err != nil {
		logError(ctx, e.logger, "xxljob: decode idleBeat error: %v", err)
		writeJSON(w, Fail(err.Error()))
		return
	}
	if e.runner.idle(param.ExecutorHandler) {
		writeJSON(w, Ok("success"))
		return
	}
	logInfo(ctx, e.logger, "xxljob: idleBeat rejected handler=%s running", param.ExecutorHandler)
	writeJSON(w, Fail("job running"))
}

func (e *Executor) handleKill(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if !e.authorize(r) {
		logError(ctx, e.logger, "xxljob: unauthorized kill request")
		writeJSON(w, Fail("invalid access token"))
		return
	}
	var param KillParam
	if err := decodeJSON(r, &param); err != nil {
		logError(ctx, e.logger, "xxljob: decode kill error: %v", err)
		writeJSON(w, Fail(err.Error()))
		return
	}
	if e.runner.kill(param.ExecutorHandler) {
		logInfo(ctx, e.logger, "xxljob: killed handler=%s", param.ExecutorHandler)
		writeJSON(w, Ok("success"))
		return
	}
	logInfo(ctx, e.logger, "xxljob: kill ignored handler=%s no running task", param.ExecutorHandler)
	writeJSON(w, Fail("no running task"))
}

func (e *Executor) handleLog(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if !e.authorize(r) {
		logError(ctx, e.logger, "xxljob: unauthorized log request")
		writeJSON(w, Fail("invalid access token"))
		return
	}
	var param LogParam
	if err := decodeLogParam(r, &param); err != nil {
		logError(ctx, e.logger, "xxljob: decode log error: %v", err)
		writeJSON(w, Fail(err.Error()))
		return
	}
	if e.logHandler == nil {
		writeJSON(w, ReturnT{Code: SuccessCode, Msg: "success", Content: LogResult{}})
		return
	}
	res, err := e.logHandler(param)
	if err != nil {
		logError(ctx, e.logger, "xxljob: log handler error: %v", err)
		writeJSON(w, Fail(err.Error()))
		return
	}
	writeJSON(w, ReturnT{Code: SuccessCode, Msg: "success", Content: res})
}

func writeJSON(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	enc := json.NewEncoder(w)
	_ = enc.Encode(v)
}

func decodeJSON(r *http.Request, v interface{}) error {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return err
	}
	if len(body) == 0 {
		return nil
	}
	return json.Unmarshal(body, v)
}

func decodeTriggerParam(payload []byte) (TriggerParam, error) {
	var tmp struct {
		TriggerParam
		LogDateTim int64 `json:"logDateTim"`
	}
	if err := json.Unmarshal(payload, &tmp); err != nil {
		return TriggerParam{}, err
	}
	if tmp.TriggerParam.LogDateTime == 0 {
		tmp.TriggerParam.LogDateTime = tmp.LogDateTim
	}
	tmp.TriggerParam.Raw = payload
	return tmp.TriggerParam, nil
}

func decodeLogParam(r *http.Request, param *LogParam) error {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return err
	}
	if len(body) == 0 {
		return nil
	}
	var tmp struct {
		LogDateTim  int64 `json:"logDateTim"`
		LogDateTime int64 `json:"logDateTime"`
		LogID       int64 `json:"logId"`
		FromLineNum int   `json:"fromLineNum"`
	}
	if err := json.Unmarshal(body, &tmp); err != nil {
		return err
	}
	param.LogID = tmp.LogID
	param.FromLineNum = tmp.FromLineNum
	if tmp.LogDateTim != 0 {
		param.LogDateTime = tmp.LogDateTim
	} else {
		param.LogDateTime = tmp.LogDateTime
	}
	return nil
}
