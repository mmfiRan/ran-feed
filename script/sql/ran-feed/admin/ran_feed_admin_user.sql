CREATE TABLE IF NOT EXISTS ran_feed_admin_user
(
    id            BIGINT PRIMARY KEY AUTO_INCREMENT COMMENT '管理员ID',
    username      VARCHAR(64)  NOT NULL COMMENT '登录用户名唯一',
    password_hash VARCHAR(255) NOT NULL COMMENT '密码哈希',
    password_salt VARCHAR(64)  NOT NULL COMMENT '密码盐',
    nickname      VARCHAR(64)  NOT NULL DEFAULT '' COMMENT '昵称',
    status        TINYINT      NOT NULL DEFAULT 10 COMMENT '状态 10=启用 20=禁用',
    version       INT          NOT NULL DEFAULT 1 COMMENT '版本号（乐观锁）',
    is_deleted    TINYINT      NOT NULL DEFAULT 0 COMMENT '逻辑删除 0=正常 1=删除',
    created_by    BIGINT       NOT NULL DEFAULT 0 COMMENT '创建人',
    updated_by    BIGINT       NOT NULL DEFAULT 0 COMMENT '最后修改人',
    created_at    DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    updated_at    DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP
        ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    UNIQUE INDEX uk_username (username),
    INDEX idx_status (status)
)
    ENGINE = InnoDB
    DEFAULT CHARSET = utf8mb4
    COLLATE = utf8mb4_bin
    COMMENT ='后台管理员账号表';