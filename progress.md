# 会话进度日志

## 当前状态

**最后更新：** 2026-06-11
**当前功能：** refactor-001 count canal 消费者重构与策略重抽象（done）

---

## 已完成

- refactor-001 count canal 消费者重构
  - Consume 拆为解析 路由 去重 落库 派发 五步管道 主文件瘦身
  - 新增 canal_message.go 消息值对象 change_set.go 类型化 key 累积器 effects.go 副作用派发
  - 落库分支收口 consumer.applyUpdate operator 延迟双删裸 go func 改 threading.GoSafe
  - strategy 按判活谓词加目标映射重抽象 presence 与 reset 两族 拆子包 删 followCountTableStrategy 等重复
  - 新增服务级 enum.RecordStatus 复用 pkg/enum.IsDeleted 替换 status 魔法数
  - 删死代码 extractCountUpdates getInt64Value 统一 strategy.ParseInt64
  - 补单测 presence reset registry change_set consumer record_status

## 进行中

- [ ]

## 下一步

1. 从 feature_list.json 选 feat-014~018 中一条新功能开工

---

## 决策记录

- 策略抽象线由 deltaByOpFn 改为判活谓词加目标映射两轴 like/favorite/comment/follow 共用 presenceCounterStrategy
- 子包按实现而非按表拆 presence 一族 reset 一族 空导入触发 init 注册
- applyUpdate 留在 consumer 而非下沉 operator 避免 logic 层反向依赖 mq 层

## 遗留风险

- 坏 JSON 消息 Consume 返回 err 会无限重试 暂无死信处理 后续可加上限或丢弃
