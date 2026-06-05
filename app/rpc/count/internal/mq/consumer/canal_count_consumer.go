package consumer

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/zeromicro/go-zero/core/logc"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/threading"

	"ran-feed/app/rpc/count/count"
	rediskey "ran-feed/app/rpc/count/internal/common/consts/redis"
	"ran-feed/app/rpc/count/internal/entity/query"
	counterservicelogic "ran-feed/app/rpc/count/internal/logic/counterservice"
	"ran-feed/app/rpc/count/internal/mq/consumer/strategy"
	"ran-feed/app/rpc/count/internal/repositories"
	"ran-feed/app/rpc/count/internal/svc"
)

type canalMessage struct {
	ID    interface{}              `json:"id"`
	Table string                   `json:"table"`
	Type  string                   `json:"type"`
	Ts    int64                    `json:"ts"`
	Data  []map[string]interface{} `json:"data"`
	Old   []map[string]interface{} `json:"old"`
}

type CanalCountConsumer struct {
	ctx        context.Context
	svcContext *svc.ServiceContext
	logx.Logger
	countRepo     repositories.CountValueRepository
	dedupRepo     repositories.MqConsumeDedupRepository
	deltaOperator *counterservicelogic.CountDeltaOperator
	consumerName  string
	strategies    *strategy.Registry
}

const (
	userProfileCacheInvalidateDelay   = 200 * time.Millisecond
	userProfileCacheInvalidateTimeout = 5 * time.Second
)

func NewCanalCountConsumer(ctx context.Context, svcContext *svc.ServiceContext) *CanalCountConsumer {
	return &CanalCountConsumer{
		ctx:           ctx,
		svcContext:    svcContext,
		Logger:        logx.WithContext(ctx),
		countRepo:     repositories.NewCountValueRepository(ctx, svcContext.MysqlDb),
		dedupRepo:     repositories.NewMqConsumeDedupRepository(ctx, svcContext.MysqlDb),
		deltaOperator: counterservicelogic.NewCountDeltaOperator(ctx, svcContext),
		consumerName:  "count.canal_consumer",
		strategies:    strategy.NewDefaultRegistry(),
	}
}

func (c *CanalCountConsumer) Consume(ctx context.Context, key, val string) error {
	var msg canalMessage
	logc.Infof(ctx, "收到canal消息: key=%s, val=%s", key, val)
	if err := json.Unmarshal([]byte(val), &msg); err != nil {
		logc.Errorf(ctx, "解析canal消息失败: %v, val=%s", err, val)
		return err
	}

	table := strings.ToLower(strings.TrimSpace(msg.Table))
	tableStrategy, ok := c.strategies.Get(table)
	if !ok {
		logc.Infof(ctx, "跳过未监听表消息: table=%s", msg.Table)
		return nil
	}

	eventID := buildEventID(msg, val)
	if eventID == "" {
		logc.Errorf(ctx, "canal消息event_id为空: table=%s", msg.Table)
		return nil
	}

	op := strings.ToUpper(strings.TrimSpace(msg.Type))
	updatedAt := canalTsToTime(msg.Ts)

	changedKeys := make(map[string]struct{})
	changedUserIDs := make(map[int64]struct{})
	dirtyContentIDs := make(map[int64]struct{})
	err := query.Q.Transaction(func(tx *query.Query) error {
		for i, row := range msg.Data {
			if row == nil {
				continue
			}
			oldRow := getOldRow(msg.Old, i)
			rowEventID := buildRowEventID(eventID, table, op, row, i)
			inserted, err := c.dedupRepo.WithTx(tx).InsertIfAbsent(c.consumerName, rowEventID)
			if err != nil {
				return err
			}
			if !inserted {
				logc.Infof(ctx, "canal消息行已处理，跳过: eventId=%s, rowEventId=%s, table=%s", eventID, rowEventID, table)
				continue
			}

			updates := tableStrategy.ExtractUpdates(ctx, op, row, oldRow)
			for _, u := range updates {
				if u.TargetID <= 0 {
					continue
				}
				if u.Action == strategy.UpdateActionResetToZero {
					rowVal, gerr := c.countRepo.WithTx(tx).Get(int32(u.BizType), int32(u.TargetType), u.TargetID)
					if gerr != nil {
						return gerr
					}
					if rowVal == nil || rowVal.Value <= 0 {
						continue
					}
					ownerID := u.OwnerID
					if ownerID <= 0 {
						ownerID = rowVal.OwnerID
					}
					if err = c.deltaOperator.UpdateDeltaOnlyWithRepoAndOwner(
						c.countRepo.WithTx(tx),
						u.BizType,
						u.TargetType,
						u.TargetID,
						ownerID,
						-rowVal.Value,
						updatedAt,
					); err != nil {
						return err
					}
					if u.TargetType == count.TargetType_CONTENT && ownerID > 0 {
						changedUserIDs[ownerID] = struct{}{}
					}
				} else {
					if u.Delta == 0 {
						continue
					}
					if err = c.deltaOperator.UpdateDeltaOnlyWithRepoAndOwner(
						c.countRepo.WithTx(tx),
						u.BizType,
						u.TargetType,
						u.TargetID,
						u.OwnerID,
						u.Delta,
						updatedAt,
					); err != nil {
						return err
					}
				}
				changedKeys[fmt.Sprintf("%d:%d:%d", u.BizType, u.TargetType, u.TargetID)] = struct{}{}
				if u.TargetType == count.TargetType_CONTENT && u.OwnerID > 0 {
					changedUserIDs[u.OwnerID] = struct{}{}
				}
				if u.TargetType == count.TargetType_USER && u.TargetID > 0 {
					changedUserIDs[u.TargetID] = struct{}{}
				}
				// 内容互动 点赞取消 收藏取消 评论相关 只登记谁脏了
				// 算分由 content 快更任务回查计数总量批量完成 见 HOT_FEED_DESIGN 第0节和第4节
				if u.TargetType == count.TargetType_CONTENT && u.TargetID > 0 {
					dirtyContentIDs[u.TargetID] = struct{}{}
				}
			}
		}
		return nil
	})
	if err != nil {
		return err
	}

	for key := range changedKeys {
		parts := strings.Split(key, ":")
		if len(parts) != 3 {
			continue
		}
		bizType, err1 := strconv.ParseInt(parts[0], 10, 32)
		targetType, err2 := strconv.ParseInt(parts[1], 10, 32)
		targetID, err3 := strconv.ParseInt(parts[2], 10, 64)
		if err1 != nil || err2 != nil || err3 != nil || targetID <= 0 {
			continue
		}
		c.deltaOperator.InvalidateCountCache(count.BizType(bizType), count.TargetType(targetType), targetID)
	}
	c.invalidateUserProfileCaches(changedUserIDs)
	if err = c.markHotDirty(ctx, dirtyContentIDs); err != nil {
		return err
	}

	return nil
}
func (c *CanalCountConsumer) extractCountUpdates(
	ctx context.Context,
	s strategy.TableStrategy,
	op string,
	data []map[string]interface{},
	old []map[string]interface{},
) []strategy.Update {
	if len(data) == 0 {
		return nil
	}

	updates := make([]strategy.Update, 0, len(data))
	for i, row := range data {
		if row == nil {
			continue
		}
		oldRow := getOldRow(old, i)
		updates = append(updates, s.ExtractUpdates(ctx, op, row, oldRow)...)
	}

	return updates
}

func getOldRow(old []map[string]interface{}, idx int) map[string]interface{} {
	if idx < 0 || idx >= len(old) {
		return nil
	}
	return old[idx]
}

func buildEventID(msg canalMessage, raw string) string {
	id := strings.TrimSpace(fmt.Sprint(msg.ID))
	if id != "" && id != "<nil>" {
		if len(id) > 64 {
			return id[:64]
		}
		return id
	}

	h := sha1.Sum([]byte(raw))
	return hex.EncodeToString(h[:])
}

func buildRowEventID(eventID, table, op string, row map[string]interface{}, idx int) string {
	if eventID == "" {
		eventID = "unknown"
	}
	rowID, _ := getInt64Value(row["id"])
	if rowID > 0 {
		raw := fmt.Sprintf("%s|%s|%s|%d", eventID, table, op, rowID)
		h := sha1.Sum([]byte(raw))
		return hex.EncodeToString(h[:])
	}

	rowJSON, _ := json.Marshal(row)
	raw := fmt.Sprintf("%s|%s|%s|%d|%s", eventID, table, op, idx, string(rowJSON))
	h := sha1.Sum([]byte(raw))
	return hex.EncodeToString(h[:])
}

func getInt64Value(v interface{}) (int64, bool) {
	switch n := v.(type) {
	case nil:
		return 0, false
	case int:
		return int64(n), true
	case int32:
		return int64(n), true
	case int64:
		return n, true
	case uint:
		return int64(n), true
	case uint32:
		return int64(n), true
	case uint64:
		return int64(n), true
	case float64:
		return int64(n), true
	case json.Number:
		val, err := n.Int64()
		if err != nil {
			return 0, false
		}
		return val, true
	case string:
		s := strings.TrimSpace(n)
		if s == "" {
			return 0, false
		}
		val, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			return 0, false
		}
		return val, true
	default:
		return 0, false
	}
}

func canalTsToTime(ts int64) time.Time {
	if ts <= 0 {
		return time.Now()
	}
	if ts > 1_000_000_000_000 {
		return time.UnixMilli(ts)
	}
	return time.Unix(ts, 0)
}

func (c *CanalCountConsumer) invalidateUserProfileCaches(userIDs map[int64]struct{}) {
	if len(userIDs) == 0 {
		return
	}
	for userID := range userIDs {
		if userID <= 0 {
			continue
		}
		cacheKey := rediskey.GetRedisPrefixKey(rediskey.RedisUserProfileCountsPrefix, strconv.FormatInt(userID, 10))
		if _, err := c.svcContext.Redis.DelCtx(c.ctx, cacheKey); err != nil {
			c.Errorf("删除用户主页计数缓存失败: key=%s, user_id=%d, err=%v", cacheKey, userID, err)
		}
		threading.GoSafe(func() {
			func(cacheKey string, userID int64) {
				time.Sleep(userProfileCacheInvalidateDelay)
				// 用 bg ctx + timeout：consumer 关停时 c.ctx 被 cancel 会让残留延迟清理直接失败，
				// 进一步加大缓存与 DB 漂移。bg ctx 保证清理一定有机会跑完。
				ctx, cancel := context.WithTimeout(context.Background(), userProfileCacheInvalidateTimeout)
				defer cancel()
				if _, err := c.svcContext.Redis.DelCtx(ctx, cacheKey); err != nil {
					c.Errorf("延迟删除用户主页计数缓存失败: key=%s, user_id=%d, err=%v", cacheKey, userID, err)
				}
			}(cacheKey, userID)
		})
	}
}

func hotDirtyShard(contentID int64) int {
	if contentID <= 0 {
		return 0
	}
	return int(contentID % int64(rediskey.RedisFeedHotIncDefaultShards))
}

// markHotDirty 把发生互动的内容登记进热榜脏集合 Set 去重 按取模分片
// 只记谁脏了 不算分 算分由 content 快更任务回查计数总量批量完成
func (c *CanalCountConsumer) markHotDirty(ctx context.Context, contentIDs map[int64]struct{}) error {
	if len(contentIDs) == 0 {
		return nil
	}
	// 按分片聚合 每片一次 SADD 批量写入 减少往返
	byShard := make(map[int][]any, rediskey.RedisFeedHotIncDefaultShards)
	for contentID := range contentIDs {
		if contentID <= 0 {
			continue
		}
		shard := hotDirtyShard(contentID)
		byShard[shard] = append(byShard[shard], strconv.FormatInt(contentID, 10))
	}
	for shard, members := range byShard {
		dirtyKey := rediskey.BuildHotFeedDirtyKey(shard)
		if _, err := c.svcContext.Redis.SaddCtx(ctx, dirtyKey, members...); err != nil {
			return err
		}
	}
	return nil
}
