CREATE TABLE IF NOT EXISTS ran_feed_user
(
    id            BIGINT PRIMARY KEY AUTO_INCREMENT COMMENT '用户ID',
    username      VARCHAR(64)  NOT NULL COMMENT '用户名唯一',
    nickname      VARCHAR(64)  NOT NULL COMMENT '昵称',
    avatar        VARCHAR(255) NOT NULL DEFAULT '' COMMENT '头像地址',
    bio           VARCHAR(255) NOT NULL DEFAULT '' COMMENT '个人简介',
    mobile        VARCHAR(20)  NOT NULL DEFAULT '' COMMENT '手机号',
    email         VARCHAR(128) NOT NULL DEFAULT '' COMMENT '邮箱',
    password_hash VARCHAR(255) NOT NULL COMMENT '密码哈希',
    password_salt VARCHAR(64)  NOT NULL COMMENT '密码盐',
    gender        TINYINT      NOT NULL DEFAULT 0 COMMENT '性别 0=未知 1=男 2=女',
    birthday      DATE                  DEFAULT NULL COMMENT '生日',
    status        TINYINT      NOT NULL DEFAULT 10 COMMENT '状态 10=正常 20=禁用 30=注销',
    version       INT          NOT NULL DEFAULT 1 COMMENT '版本号（乐观锁）',
    is_deleted    TINYINT      NOT NULL DEFAULT 0 COMMENT '逻辑删除 0=正常 1=删除',
    created_by    BIGINT       NOT NULL DEFAULT 0 COMMENT '创建人',
    updated_by    BIGINT       NOT NULL DEFAULT 0 COMMENT '最后修改人',
    created_at    DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    updated_at    DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP
        ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    UNIQUE INDEX uk_username (username),
    UNIQUE INDEX uk_mobile (mobile),
    UNIQUE INDEX uk_email (email),
    INDEX idx_status (status)
)
    ENGINE = InnoDB
    DEFAULT CHARSET = utf8mb4
    COLLATE = utf8mb4_bin
    COMMENT ='用户基础信息表';

