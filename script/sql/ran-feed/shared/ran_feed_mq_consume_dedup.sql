-- 跨服务共享表 content/count/interaction/notification/search 各消费者共用
-- consumer 维度隔离 同一条消息可被多个服务各处理一次
CREATE TABLE IF NOT EXISTS ran_feed_mq_consume_dedup
(
    id         BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键id',
    consumer   VARCHAR(64)     NOT NULL COMMENT '消费者标识 如 count.canal_consumer',
    event_id   VARCHAR(64)     NOT NULL COMMENT '行级幂等键 见 pkg/event/canal.RowEventID',
    created_at DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3) COMMENT '创建时间',
    PRIMARY KEY (id),
    UNIQUE KEY uniq_consumer_event (consumer, event_id),
    KEY idx_created_at (created_at)
)
    ENGINE = InnoDB
    DEFAULT CHARSET = utf8mb4
    COMMENT ='消息消费幂等去重表';
