SET NAMES utf8mb4;

CREATE TABLE IF NOT EXISTS ran_feed_im_conversation
(
    id          BIGINT PRIMARY KEY COMMENT '会话 ID',
    conv_type   TINYINT      NOT NULL COMMENT '会话类型 10=单聊 20=群聊',
    biz_key     VARCHAR(128) NOT NULL COMMENT '业务幂等键,单聊=S:{较小 userId}_{较大 userId},群聊=G:{groupId}',
    owner_id    BIGINT       NOT NULL DEFAULT 0 COMMENT '群聊关联的群 ID,单聊为 0',
    max_seq     BIGINT       NOT NULL DEFAULT 0 COMMENT '当前最大消息序号,定序水位',
    last_msg_id BIGINT       NOT NULL DEFAULT 0 COMMENT '最后一条消息 ID,用于列表预览',
    last_msg_at DATETIME(3)  NULL COMMENT '最后活跃时间,用于会话列表排序',
    version     INT          NOT NULL DEFAULT 1 COMMENT '版本号,乐观锁',
    is_deleted  TINYINT      NOT NULL DEFAULT 0 COMMENT '逻辑删除 0=正常 1=删除',
    created_by  BIGINT       NOT NULL DEFAULT 0 COMMENT '创建人',
    updated_by  BIGINT       NOT NULL DEFAULT 0 COMMENT '最后修改人',
    created_at  DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3) COMMENT '创建时间',
    updated_at  DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3)
        ON UPDATE CURRENT_TIMESTAMP(3) COMMENT '更新时间',
    UNIQUE INDEX uk_biz_key (biz_key)
)
    ENGINE = InnoDB
    DEFAULT CHARSET = utf8mb4
    COLLATE = utf8mb4_bin
    COMMENT ='IM 会话表';
