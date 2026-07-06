CREATE TABLE IF NOT EXISTS ran_feed_admin_role_permission
(
    id            BIGINT PRIMARY KEY AUTO_INCREMENT COMMENT '主键',
    role_id       BIGINT   NOT NULL COMMENT '角色ID',
    permission_id BIGINT   NOT NULL COMMENT '权限点ID',
    version       INT      NOT NULL DEFAULT 1 COMMENT '版本号（乐观锁）',
    is_deleted    TINYINT  NOT NULL DEFAULT 0 COMMENT '逻辑删除 0=正常 1=删除',
    created_by    BIGINT   NOT NULL DEFAULT 0 COMMENT '创建人',
    updated_by    BIGINT   NOT NULL DEFAULT 0 COMMENT '最后修改人',
    created_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    updated_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
        ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    UNIQUE INDEX uk_role_permission (role_id, permission_id),
    INDEX idx_permission (permission_id)
)
    ENGINE = InnoDB
    DEFAULT CHARSET = utf8mb4
    COLLATE = utf8mb4_bin
    COMMENT ='后台角色-权限点关系表';