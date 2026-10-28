SET NAMES utf8mb4;

CREATE TABLE IF NOT EXISTS ran_feed_im_message
(
    id              BIGINT PRIMARY KEY COMMENT '消息 ID',
    conversation_id BIGINT        NOT NULL COMMENT '会话 ID',
    seq             BIGINT        NOT NULL COMMENT '会话内序号 从1开始连续递增',
    client_msg_id   VARCHAR(64)   NOT NULL COMMENT '客户端生成的消息唯一标识,用于发送幂等',
    sender_id       BIGINT        NOT NULL COMMENT '发送者用户 ID',
    content_type    TINYINT       NOT NULL COMMENT '内容类型 10=文本 20=图片 30=视频',
    content         VARCHAR(4096) NOT NULL DEFAULT '' COMMENT '消息内容,文本为正文,图片/视频为 JSON 元数据',
    status          TINYINT       NOT NULL DEFAULT 10 COMMENT '消息状态 10=正常 20=已撤回',
    version         INT           NOT NULL DEFAULT 1 COMMENT '版本号,乐观锁',
    is_deleted      TINYINT       NOT NULL DEFAULT 0 COMMENT '逻辑删除 0=正常 1=删除',
    created_by      BIGINT        NOT NULL DEFAULT 0 COMMENT '创建人',
    updated_by      BIGINT        NOT NULL DEFAULT 0 COMMENT '最后修改人',
    created_at      DATETIME(3)   NOT NULL DEFAULT CURRENT_TIMESTAMP(3) COMMENT '创建时间',
    updated_at      DATETIME(3)   NOT NULL DEFAULT CURRENT_TIMESTAMP(3)
        ON UPDATE CURRENT_TIMESTAMP(3) COMMENT '更新时间',
    UNIQUE INDEX uk_conv_seq (conversation_id, seq),
    UNIQUE INDEX uk_conv_client_msg (conversation_id, client_msg_id)
)
    ENGINE = InnoDB
    DEFAULT CHARSET = utf8mb4
    COLLATE = utf8mb4_bin
    COMMENT ='IM 消息表';
