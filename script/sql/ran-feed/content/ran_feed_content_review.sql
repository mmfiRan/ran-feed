CREATE TABLE IF NOT EXISTS ran_feed_content_review
(
    id          BIGINT PRIMARY KEY NOT NULL COMMENT '主键id',
    content_id  BIGINT             NOT NULL COMMENT '被审内容ID',
    decision    TINYINT            NOT NULL COMMENT '审核决策 10=通过 20=拒绝 30=管理端下架 40=管理端恢复',
    reason      VARCHAR(512)       NOT NULL DEFAULT '' COMMENT '原因 通过与恢复为空 拒绝为拒绝理由 下架为下架原因',
    version     INT                NOT NULL DEFAULT 1 COMMENT '版本号（乐观锁）',
    is_deleted  TINYINT            NOT NULL DEFAULT 0 COMMENT '逻辑删除 0=正常 1=删除',
    created_by  BIGINT             NOT NULL COMMENT '创建人（审核管理员 admin_user.id）',
    updated_by  BIGINT             NOT NULL COMMENT '最后修改人',
    created_at  DATETIME           NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    updated_at  DATETIME           NOT NULL DEFAULT CURRENT_TIMESTAMP
        ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    KEY idx_content (content_id, created_at)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4
  COLLATE = utf8mb4_bin
    COMMENT ='内容审核记录表';