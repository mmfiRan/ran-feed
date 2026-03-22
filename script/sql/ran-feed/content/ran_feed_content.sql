CREATE TABLE IF NOT EXISTS ran_feed_content
(
    id           BIGINT PRIMARY KEY AUTO_INCREMENT COMMENT '内容ID',
    user_id      BIGINT   NOT NULL COMMENT '发布者（内容作者）',
    content_type TINYINT  NOT NULL COMMENT '内容类型 10=文章 20=视频',
    status       TINYINT  NOT NULL COMMENT '状态 10=草稿 20=处理中 30=已发布 40=失败',
    visibility   TINYINT  NOT NULL DEFAULT 1 COMMENT '可见性 10=公开 20=私密',
    like_count   BIGINT  NOT NULL DEFAULT 0 COMMENT '点赞数',
    favorite_count BIGINT  NOT NULL DEFAULT 0 COMMENT '收藏数',
    comment_count BIGINT  NOT NULL DEFAULT 0 COMMENT '评论数',
    hot_score    DOUBLE  NOT NULL DEFAULT 0 COMMENT '热度分',
    last_hot_score_at DATETIME NULL COMMENT '热度分最后更新时间',
    version      INT      NOT NULL DEFAULT 1 COMMENT '版本号（乐观锁）',
    is_deleted   TINYINT  NOT NULL DEFAULT 0 COMMENT '逻辑删除 0=正常 1=删除',
    created_by   BIGINT   NOT NULL COMMENT '创建人',
    updated_by   BIGINT   NOT NULL COMMENT '最后修改人',
    created_at   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    published_at DATETIME NULL COMMENT '发布时间',
    updated_at   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
        ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间'
)
    ENGINE = InnoDB
    DEFAULT CHARSET = utf8mb4
    COLLATE = utf8mb4_bin
    COMMENT ='统一内容主表';

-- Idempotent index creation: skip if the index already exists.
DELIMITER $$
DROP PROCEDURE IF EXISTS create_index_if_missing $$
CREATE PROCEDURE create_index_if_missing(IN p_table VARCHAR(64), IN p_index VARCHAR(64), IN p_ddl TEXT)
BEGIN
    IF (
        SELECT COUNT(*)
        FROM information_schema.statistics
        WHERE table_schema = DATABASE()
          AND table_name = p_table
          AND index_name = p_index
    ) = 0 THEN
        SET @s = p_ddl;
        PREPARE stmt FROM @s;
        EXECUTE stmt;
        DEALLOCATE PREPARE stmt;
    END IF;
END $$
DELIMITER ;

CALL create_index_if_missing('ran_feed_content', 'idx_ran_feed_content_hot_score',
    'CREATE INDEX idx_ran_feed_content_hot_score ON ran_feed_content (hot_score, id)');
CALL create_index_if_missing('ran_feed_content', 'idx_ran_feed_content_user_created',
    'CREATE INDEX idx_ran_feed_content_user_created ON ran_feed_content (user_id, created_at, id)');
CALL create_index_if_missing('ran_feed_content', 'idx_ran_feed_content_published',
    'CREATE INDEX idx_ran_feed_content_published ON ran_feed_content (published_at, id)');
CALL create_index_if_missing('ran_feed_content', 'idx_ran_feed_content_user_published',
    'CREATE INDEX idx_ran_feed_content_user_published ON ran_feed_content (user_id, published_at, id)');

DROP PROCEDURE IF EXISTS create_index_if_missing;
