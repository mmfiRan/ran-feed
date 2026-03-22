CREATE TABLE IF NOT EXISTS ran_feed_favorite
(
    id         BIGINT PRIMARY KEY COMMENT 'id',
    user_id    BIGINT   NOT NULL COMMENT '用户ID',
    status     INT      NOT NULL COMMENT '状态 10=正常 20=取消收藏',
    content_id BIGINT   NOT NULL COMMENT '内容ID',
    content_user_id BIGINT NOT NULL COMMENT '内容作者ID',
    created_by BIGINT   NOT NULL COMMENT '创建人',
    updated_by BIGINT   NOT NULL COMMENT '最后修改人',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
        ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    UNIQUE index uk_user_content (user_id, content_id),
    index idx_user_created (user_id, created_at DESC),
    index idx_content (content_id),
    index idx_content_user (content_user_id)
) ENGINE = InnoDB COMMENT ='用户收藏表';
