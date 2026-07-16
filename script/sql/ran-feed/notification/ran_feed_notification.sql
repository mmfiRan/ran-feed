CREATE TABLE IF NOT EXISTS ran_feed_notification
(
    id           BIGINT primary key AUTO_INCREMENT COMMENT '通知ID',
    recipient_id BIGINT       NOT NULL COMMENT '收件人(被通知用户)ID',
    actor_id     BIGINT       NOT NULL COMMENT '触发者ID 聚合行为记最新触发者',
    notify_type  TINYINT      NOT NULL COMMENT '通知类型 10=赞或收藏 20=评论或回复 30=关注',
    agg_key      VARCHAR(64)  NOT NULL COMMENT '聚合或去重键 如 LF:{content_id} CR:{comment_id} FO:{actor_id}',
    agg_count    INT          NOT NULL DEFAULT 1 COMMENT '聚合触发次数 单条恒为1',
    content_id   BIGINT       NOT NULL DEFAULT 0 COMMENT '关联内容ID 关注类为0',
    comment_id   BIGINT       NOT NULL DEFAULT 0 COMMENT '关联评论ID 评论或回复类有值',
    snippet      VARCHAR(140) NOT NULL DEFAULT '' COMMENT '评论或回复文本摘要',
    is_read      TINYINT      NOT NULL DEFAULT 0 COMMENT '已读状态 0=未读 1=已读',
    read_at      DATETIME(3)  NULL COMMENT '已读时间',
    version      INT          NOT NULL DEFAULT 1 COMMENT '版本号(乐观锁)',
    is_deleted   TINYINT      NOT NULL DEFAULT 0 COMMENT '逻辑删除 0=正常 1=删除',
    created_by   BIGINT       NOT NULL DEFAULT 0 COMMENT '创建人',
    updated_by   BIGINT       NOT NULL DEFAULT 0 COMMENT '最后修改人',
    created_at   DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3) COMMENT '创建时间',
    updated_at   DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3)
        ON UPDATE CURRENT_TIMESTAMP(3) COMMENT '更新时间',
    constraint uk_recipient_aggkey
        unique (recipient_id, agg_key),
    index idx_recipient_updated (recipient_id, is_deleted, updated_at, id),
    index idx_recipient_unread (recipient_id, is_deleted, is_read)
)
    ENGINE = InnoDB
    DEFAULT CHARSET = utf8mb4
    COLLATE = utf8mb4_bin
    COMMENT ='互动通知表';