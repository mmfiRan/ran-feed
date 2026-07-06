CREATE TABLE IF NOT EXISTS ran_feed_operation_log
(
    id          BIGINT PRIMARY KEY AUTO_INCREMENT COMMENT '日志ID',
    admin_id    BIGINT       NOT NULL COMMENT '操作人管理员ID',
    action      VARCHAR(128) NOT NULL COMMENT '操作动作 如 content:takedown',
    target_type VARCHAR(64)  NOT NULL DEFAULT '' COMMENT '目标类型 如 content/user',
    target_id   BIGINT       NOT NULL DEFAULT 0 COMMENT '目标ID',
    payload     TEXT COMMENT '入参摘要 JSON',
    result      VARCHAR(255) NOT NULL DEFAULT '' COMMENT '操作结果',
    ip          VARCHAR(64)  NOT NULL DEFAULT '' COMMENT '来源IP',
    version     INT          NOT NULL DEFAULT 1 COMMENT '版本号（乐观锁）',
    is_deleted  TINYINT      NOT NULL DEFAULT 0 COMMENT '逻辑删除 0=正常 1=删除',
    created_by  BIGINT       NOT NULL DEFAULT 0 COMMENT '创建人',
    updated_by  BIGINT       NOT NULL DEFAULT 0 COMMENT '最后修改人',
    created_at  DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    updated_at  DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP
        ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    INDEX idx_admin (admin_id),
    INDEX idx_target (target_type, target_id),
    INDEX idx_created_at (created_at)
)
    ENGINE = InnoDB
    DEFAULT CHARSET = utf8mb4
    COLLATE = utf8mb4_bin
    COMMENT ='后台操作审计日志表';