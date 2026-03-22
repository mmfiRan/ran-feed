-- auto-generated definition
CREATE TABLE IF NOT EXISTS ran_feed_count_value
(
    id          bigint auto_increment
        primary key,
    biz_type    int              not null comment '计数业务类型：10=like,20=favorite,30=comment,40=followed,41=following',
    target_type int              not null comment '计数对象类型：10=content,20=user,...',
    target_id   bigint           not null comment '对象ID（如 content_id）',
    value       bigint default 0 not null comment '计数值',
    version     bigint default 0 not null comment '版本号（可用于乐观锁/审计）',
    created_at  datetime(3)      not null,
    updated_at  datetime(3)      not null,
    owner_id    bigint default 0 not null comment '内容作者ID（仅target_type=CONTENT时有效）',
    constraint uk_biz_target
        unique (biz_type, target_type, target_id)
)
    charset = utf8mb4;

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

CALL create_index_if_missing('ran_feed_count_value', 'idx_owner',
    'CREATE INDEX idx_owner ON ran_feed_count_value (owner_id)');
CALL create_index_if_missing('ran_feed_count_value', 'idx_target',
    'CREATE INDEX idx_target ON ran_feed_count_value (target_type, target_id)');

DROP PROCEDURE IF EXISTS create_index_if_missing;


