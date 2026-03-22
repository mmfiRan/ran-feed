CREATE TABLE IF NOT EXISTS ran_feed_video
(
    id               bigint primary key not null comment '主键id',
    content_id       BIGINT             NOT NULL COMMENT 'contentID',
    title            VARCHAR(255)       NOT NULL DEFAULT '' COMMENT '标题',
    media_id         BIGINT             NOT NULL COMMENT '媒体资源ID（原始视频）',
    origin_url       VARCHAR(512)       NOT NULL COMMENT '原始视频地址',
    hls_url          VARCHAR(512)       NOT NULL DEFAULT '' COMMENT 'HLS 播放地址',
    cover_url        VARCHAR(512)       NOT NULL DEFAULT '' COMMENT '封面图地址',
    duration         INT                NOT NULL DEFAULT 0 COMMENT '视频时长（秒）',
    transcode_status TINYINT            NOT NULL DEFAULT 0 COMMENT '转码状态 10=未开始 20=处理中 30=成功 40=失败',
    fail_reason      VARCHAR(255)       NOT NULL DEFAULT '' COMMENT '失败原因',
    version          INT                NOT NULL DEFAULT 1 COMMENT '版本号（乐观锁）',
    is_deleted       TINYINT            NOT NULL DEFAULT 0 COMMENT '逻辑删除 0=正常 1=删除',
    created_at       DATETIME           NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    updated_at       DATETIME           NOT NULL DEFAULT CURRENT_TIMESTAMP
        ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间'

) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4
  COLLATE = utf8mb4_bin
    COMMENT ='视频内容表';
