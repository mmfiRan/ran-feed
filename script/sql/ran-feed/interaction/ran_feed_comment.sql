CREATE TABLE IF NOT EXISTS ran_feed_comment
(
    id         BIGINT primary key AUTO_INCREMENT COMMENT '评论ID',
    content_id BIGINT   NOT NULL COMMENT '内容ID',
    content_user_id BIGINT NOT NULL COMMENT '内容作者ID',
    user_id    BIGINT   NOT NULL COMMENT '评论用户',
    reply_to_user_id BIGINT NOT NULL DEFAULT 0 COMMENT '被回复用户ID',
    parent_id  BIGINT   NOT NULL COMMENT '父评论ID，0表示一级评论',
    root_id    BIGINT   NOT NULL COMMENT '根评论ID',
    comment    TEXT     NOT NULL COMMENT '评论内容',
    status     TINYINT  NOT NULL COMMENT '状态 10=正常 20=删除',
    version    INT      NOT NULL DEFAULT 1 COMMENT '版本号（乐观锁）',
    is_deleted TINYINT  NOT NULL DEFAULT 0 COMMENT '逻辑删除 0=正常 1=删除',
    created_by BIGINT   NOT NULL COMMENT '创建人',
    updated_by BIGINT   NOT NULL COMMENT '最后修改人',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
        ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    index idx_content (content_id),
    index idx_root (root_id),
    index idx_parent (parent_id),
    index idx_content_user (content_user_id)
)
    ENGINE = InnoDB
    DEFAULT CHARSET = utf8mb4
    COLLATE = utf8mb4_bin
    COMMENT ='内容评论表';
