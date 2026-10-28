SET NAMES utf8mb4;

CREATE TABLE IF NOT EXISTS ran_feed_im_group
(
    id           BIGINT PRIMARY KEY COMMENT '群ID',
    name         VARCHAR(64)  NOT NULL COMMENT '群名称',
    avatar       VARCHAR(512) NOT NULL DEFAULT '' COMMENT '群头像 URL',
    owner_id     BIGINT       NOT NULL COMMENT '群主用户 ID',
    member_count INT          NOT NULL DEFAULT 0 COMMENT '当前成员数',
    status       TINYINT      NOT NULL DEFAULT 10 COMMENT '状态 10=正常 20=已解散',
    version      INT          NOT NULL DEFAULT 1 COMMENT '版本号,乐观锁',
    is_deleted   TINYINT      NOT NULL DEFAULT 0 COMMENT '逻辑删除 0=正常 1=删除',
    created_by   BIGINT       NOT NULL DEFAULT 0 COMMENT '创建人',
    updated_by   BIGINT       NOT NULL DEFAULT 0 COMMENT '最后修改人',
    created_at   DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3) COMMENT '创建时间',
    updated_at   DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3)
        ON UPDATE CURRENT_TIMESTAMP(3) COMMENT '更新时间',
    INDEX idx_owner (owner_id, is_deleted)
)
    ENGINE = InnoDB
    DEFAULT CHARSET = utf8mb4
    COLLATE = utf8mb4_bin
    COMMENT ='IM 群组表';
