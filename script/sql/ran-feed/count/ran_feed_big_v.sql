CREATE TABLE IF NOT EXISTS ran_feed_big_v
(
    id             bigint auto_increment
        primary key,
    user_id        bigint           not null comment '大 V 用户ID',
    follower_count bigint default 0 not null comment '晋升时粉丝数快照（审计用）',
    version        bigint default 0 not null comment '版本号（乐观锁/审计）',
    created_at     datetime(3)      not null default CURRENT_TIMESTAMP(3) comment '晋升大 V 时间',
    updated_at     datetime(3)      not null default CURRENT_TIMESTAMP(3) on update CURRENT_TIMESTAMP(3) comment '更新时间',
    constraint uk_user
        unique (user_id)
)
    charset = utf8mb4 comment ='大 V 注册表 粉丝数跨阈值晋升 只增不删 关注流推拉判定真相源';
