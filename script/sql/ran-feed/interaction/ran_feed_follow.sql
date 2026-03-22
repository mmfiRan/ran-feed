CREATE TABLE IF NOT EXISTS ran_feed_follow
(
    id             BIGINT PRIMARY KEY AUTO_INCREMENT COMMENT '关注关系ID',
    user_id        BIGINT   NOT NULL COMMENT '关注者用户ID（follower）',
    follow_user_id BIGINT   NOT NULL COMMENT '被关注用户ID（followee）',
    status         INT      NOT NULL COMMENT '状态 10=正常 20=取消关注',
    version        INT      NOT NULL DEFAULT 1 COMMENT '版本号（乐观锁）',
    is_deleted     TINYINT  NOT NULL DEFAULT 0 COMMENT '逻辑删除 0=正常 1=删除',
    created_by     BIGINT   NOT NULL COMMENT '创建人',
    updated_by     BIGINT   NOT NULL COMMENT '最后修改人',
    created_at     DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    updated_at     DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
        ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    UNIQUE INDEX uk_user_follow_user (user_id, follow_user_id),
    INDEX idx_user (user_id),
    INDEX idx_follow_user (follow_user_id)
)
    ENGINE = InnoDB
    DEFAULT CHARSET = utf8mb4
    COLLATE = utf8mb4_bin
    COMMENT ='用户关注关系表';
