# 会话交接（Session Handoff）

> 每个会话结束时更新本文件，下一会话启动时阅读。只描述**最新一次交接的当前面**。

最后更新：2026-06-11

---

## 当前目标

- **目标**：refactor-001 count canal 消费者重构与策略重抽象
- **当前状态**：done ./init.sh 通过 待提交本地
- **分支 / 提交**：feature/feat

## 本会话已完成

- [x] Consume 拆五步管道 消息/累积器/副作用各成文件 落库收口 applyUpdate
- [x] strategy 重抽象 presence 与 reset 两族 拆子包 删重复实现
- [x] 新增服务级 RecordStatus 枚举 复用 pkg/enum.IsDeleted
- [x] 补单测 各表增量翻转 级联清零 去重幂等

## 验证证据

| 检查 | 命令 | 结果 | 备注 |
|------|------|------|------|
| 编译 静态分析 测试 | `./init.sh` | 通过 ✓ | build vet test 全绿 |
| count 服务测试 | `go test ./app/rpc/count/...` | 通过 ✓ | presence reset consumer enum 用例全过 |

## 改动文件

- 新增 app/rpc/count/internal/common/enum/record_status.go(+test)
- 新增 strategy/presence/*(presence like favorite comment follow + test)
- 新增 strategy/reset/reset.go(+test)
- 重写 strategy/registry.go 核心加 ParseInt64
- 重写 canal_count_consumer.go 加 canal_message.go change_set.go effects.go(+test)
- 改 logic/counterservice/count_delta_operator.go 延迟双删改 GoSafe
- 删 strategy 下 like/favorite/comment/follow/content_delete/helpers 旧文件

## 决策记录

- 策略抽象改为判活谓词加目标映射两轴 四张表共用 presenceCounterStrategy
- 子包按实现拆 presence reset 空导入触发注册
- applyUpdate 留 consumer 不下沉 operator 避免 logic 反向依赖 mq

## 风险 / 阻塞

- 坏 JSON 消息无限重试 无死信 后续考虑加上限

## 下一会话启动步骤

1. 阅读 `CLAUDE.md`
2. `./init.sh` 验证环境
3. 阅读 `feature_list.json` + `progress.md` + 本文件

## 推荐下一步

- 从 feat-014~018 选一条新功能开工
