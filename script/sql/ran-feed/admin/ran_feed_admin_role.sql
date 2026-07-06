CREATE TABLE IF NOT EXISTS ran_feed_admin_role
(
    id         BIGINT PRIMARY KEY AUTO_INCREMENT COMMENT '角色ID',
    code       VARCHAR(64)  NOT NULL COMMENT '角色码唯一 如 super/auditor/operator',
    name       VARCHAR(64)  NOT NULL COMMENT '角色名',
    remark     VARCHAR(255) NOT NULL DEFAULT '' COMMENT '备注',
    version    INT          NOT NULL DEFAULT 1 COMMENT '版本号（乐观锁）',
    is_deleted TINYINT      NOT NULL DEFAULT 0 COMMENT '逻辑删除 0=正常 1=删除',
    created_by BIGINT       NOT NULL DEFAULT 0 COMMENT '创建人',
    updated_by BIGINT       NOT NULL DEFAULT 0 COMMENT '最后修改人',
    created_at DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    updated_at DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP
        ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    UNIQUE INDEX uk_code (code)
)
    ENGINE = InnoDB
    DEFAULT CHARSET = utf8mb4
    COLLATE = utf8mb4_bin
    COMMENT ='后台角色表';