SET NAMES utf8mb4;

CREATE TABLE IF NOT EXISTS ran_feed_im_conversation_member
(
    id              BIGINT PRIMARY KEY COMMENT '成员记录 ID',
    conversation_id BIGINT      NOT NULL COMMENT '会话 ID',
    user_id         BIGINT      NOT NULL COMMENT '成员用户 ID',
    read_seq        BIGINT      NOT NULL DEFAULT 0 COMMENT '已读水位,只增不减',
    min_seq         BIGINT      NOT NULL DEFAULT 0 COMMENT '可见起点,小于等于该值的消息不可见',
    muted           TINYINT     NOT NULL DEFAULT 0 COMMENT '免打扰 0=否 1=是',
    version         INT         NOT NULL DEFAULT 1 COMMENT '版本号,乐观锁',
    is_deleted      TINYINT     NOT NULL DEFAULT 0 COMMENT '逻辑删除 0=正常 1=删除,退出会话即删除',
    created_by      BIGINT      NOT NULL DEFAULT 0 COMMENT '创建人',
    updated_by      BIGINT      NOT NULL DEFAULT 0 COMMENT '最后修改人',
    created_at      DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) COMMENT '创建时间',
    updated_at      DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3)
        ON UPDATE CURRENT_TIMESTAMP(3) COMMENT '更新时间',
    UNIQUE INDEX uk_conv_user (conversation_id, user_id),
    INDEX idx_user_active (user_id, is_deleted, updated_at)
)
    ENGINE = InnoDB
    DEFAULT CHARSET = utf8mb4
    COLLATE = utf8mb4_bin
    COMMENT ='IM 会话成员表';
