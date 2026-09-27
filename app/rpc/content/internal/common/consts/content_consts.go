package consts

// HotScoreUpdateBatchSize 热度分单次批量更新行数
const HotScoreUpdateBatchSize = 500

// ContentEventConsumerName content 域 outbox 事件流的消费方标识 同时是去重表的作用域键
// 消息消费与对账补跑必须用同一个值 否则同一条事件会被两条链路各处理一次
const ContentEventConsumerName = "content.feed_consumer"
