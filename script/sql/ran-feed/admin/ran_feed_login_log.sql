CREATE TABLE IF NOT EXISTS ran_feed_login_log
(
    id          BIGINT PRIMARY KEY AUTO_INCREMENT COMMENT '登录日志ID',
    admin_id    BIGINT       NOT NULL DEFAULT 0 COMMENT '管理员ID 成功时填 失败为0',
    username    VARCHAR(64)  NOT NULL DEFAULT '' COMMENT '登录账号 未认证也记 供失败排查',
    ip          VARCHAR(64)  NOT NULL DEFAULT '' COMMENT '来源IP',
    user_agent  VARCHAR(255) NOT NULL DEFAULT '' COMMENT 'UA 原始串',
    status      TINYINT      NOT NULL DEFAULT 0 COMMENT '登录结果 1=成功 2=失败 ',
    msg         VARCHAR(255) NOT NULL DEFAULT '' COMMENT '提示消息 成功或失败原因',
    version     INT          NOT NULL DEFAULT 1 COMMENT '版本号（乐观锁）',
    is_deleted  TINYINT      NOT NULL DEFAULT 0 COMMENT '逻辑删除 0=正常 1=删除',
    created_by  BIGINT       NOT NULL DEFAULT 0 COMMENT '创建人',
    updated_by  BIGINT       NOT NULL DEFAULT 0 COMMENT '最后修改人',
    created_at  DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    updated_at  DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP
        ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    INDEX idx_admin (admin_id),
    INDEX idx_username (username),
    INDEX idx_status (status),
    INDEX idx_created_at (created_at)
)
    ENGINE = InnoDB
    DEFAULT CHARSET = utf8mb4
    COLLATE = utf8mb4_bin
    COMMENT ='后台登录日志表';
