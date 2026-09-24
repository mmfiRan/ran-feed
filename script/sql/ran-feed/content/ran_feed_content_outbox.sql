CREATE TABLE IF NOT EXISTS ran_feed_content_outbox
(
    id           BIGINT PRIMARY KEY AUTO_INCREMENT COMMENT '主键id',
    event_id     VARCHAR(64) NOT NULL COMMENT '事件唯一id',
    event_type   TINYINT     NOT NULL COMMENT '事件类型 10=发布 20=删除 30=下架 40=拒绝 50=恢复',
    aggregate_id BIGINT      NOT NULL COMMENT '聚合id(content_id)',
    payload      TEXT        NOT NULL COMMENT '事件体json',
    created_at   DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) COMMENT '创建时间',
    UNIQUE KEY uk_event_id (event_id),
    KEY idx_aggregate (aggregate_id),
    KEY idx_created_at (created_at)
)
    ENGINE = InnoDB
    DEFAULT CHARSET = utf8mb4
    COLLATE = utf8mb4_bin
    COMMENT ='content本地消息表';
