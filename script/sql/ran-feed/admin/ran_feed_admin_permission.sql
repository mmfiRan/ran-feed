CREATE TABLE IF NOT EXISTS ran_feed_admin_permission
(
    id         BIGINT PRIMARY KEY AUTO_INCREMENT COMMENT '权限点ID',
    code       VARCHAR(64)  NOT NULL COMMENT '权限点码唯一 如 content:takedown',
    name       VARCHAR(64)  NOT NULL COMMENT '权限点名',
    module     VARCHAR(64)  NOT NULL DEFAULT '' COMMENT '所属模块',
    version    INT          NOT NULL DEFAULT 1 COMMENT '版本号（乐观锁）',
    is_deleted TINYINT      NOT NULL DEFAULT 0 COMMENT '逻辑删除 0=正常 1=删除',
    created_by BIGINT       NOT NULL DEFAULT 0 COMMENT '创建人',
    updated_by BIGINT       NOT NULL DEFAULT 0 COMMENT '最后修改人',
    created_at DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    updated_at DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP
        ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    UNIQUE INDEX uk_code (code),
    INDEX idx_module (module)
)
    ENGINE = InnoDB
    DEFAULT CHARSET = utf8mb4
    COLLATE = utf8mb4_bin
    COMMENT ='后台权限点表';