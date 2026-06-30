CREATE TABLE IF NOT EXISTS ran_feed_search_history
(
    id         BIGINT PRIMARY KEY AUTO_INCREMENT COMMENT '主键',
    user_id    BIGINT      NOT NULL COMMENT '用户ID 仅登录用户',
    keyword    VARCHAR(64) NOT NULL COMMENT '搜索词',
    status     INT         NOT NULL DEFAULT 10 COMMENT '状态 10=正常',
    version    INT         NOT NULL DEFAULT 1 COMMENT '版本号（乐观锁）',
    is_deleted TINYINT     NOT NULL DEFAULT 0 COMMENT '逻辑删除 0=正常 1=删除',
    created_by BIGINT      NOT NULL DEFAULT 0 COMMENT '创建人',
    updated_by BIGINT      NOT NULL DEFAULT 0 COMMENT '最后修改人',
    created_at DATETIME    NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    updated_at DATETIME    NOT NULL DEFAULT CURRENT_TIMESTAMP
        ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    UNIQUE INDEX uk_user_keyword (user_id, keyword),
    INDEX idx_user (user_id)
)
    ENGINE = InnoDB
    DEFAULT CHARSET = utf8mb4
    COLLATE = utf8mb4_bin
    COMMENT ='用户搜索历史表';
