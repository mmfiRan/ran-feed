-- auto-generated definition
CREATE TABLE IF NOT EXISTS ran_feed_article
(
    id          bigint unsigned auto_increment comment '文章ID'
        primary key,
    content_id  bigint                                 not null comment 'content 表主键ID',
    title       varchar(255) default ''                not null comment '标题',
    description varchar(255)                           null comment '摘要描述',
    cover       varchar(512) default ''                not null comment '封面图URL',
    content     mediumtext                             not null comment '文章正文内容',
    version     int          default 1                 not null comment '版本号（乐观锁）',
    is_deleted  tinyint      default 0                 not null comment '逻辑删除 0=正常 1=删除',
    created_by  bigint       default 0                 not null comment '创建人',
    updated_by  bigint       default 0                 not null comment '最后修改人',
    created_at  datetime     default CURRENT_TIMESTAMP not null comment '创建时间',
    updated_at  datetime     default CURRENT_TIMESTAMP not null on update CURRENT_TIMESTAMP comment '更新时间',
    constraint uk_content_id
        unique (content_id)
)
    comment '文章内容表' collate = utf8mb4_bin;

