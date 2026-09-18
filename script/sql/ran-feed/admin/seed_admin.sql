-- 超级管理员 帐号:admin 密码:admin 其余帐号的密码123456

INSERT INTO ran_feed_admin_role (code, name, remark)
VALUES ('super', '超级管理员', '拥有全部权限'),
       ('readonly_viewer', '访客', '可见全部模块列表与详情 无任何操作权限'),
       ('content_operator', '内容运营', '内容下架 恢复 审核'),
       ('user_operator', '用户运营', 'C 端用户封禁 恢复'),
       ('auditor', '审计员', '操作日志 登录日志查询')
ON DUPLICATE KEY UPDATE name = VALUES(name), remark = VALUES(remark);

INSERT INTO ran_feed_admin_permission (code, name, module)
VALUES ('admin:user:list', '管理员列表', 'admin'),
       ('admin:user:detail', '管理员详情', 'admin'),
       ('admin:user:create', '创建管理员', 'admin'),
       ('admin:user:update', '修改管理员昵称', 'admin'),
       ('admin:user:status', '启用禁用管理员', 'admin'),
       ('admin:user:reset-password', '重置管理员密码', 'admin'),
       ('admin:user:set-roles', '设置管理员角色', 'admin'),
       ('admin:role:list', '角色列表', 'admin'),
       ('admin:role:detail', '角色详情', 'admin'),
       ('admin:role:create', '创建角色', 'admin'),
       ('admin:role:update', '修改角色', 'admin'),
       ('admin:role:set-permissions', '设置角色权限', 'admin'),
       ('admin:role:delete', '删除角色', 'admin'),
       ('admin:permission:list', '权限点列表', 'admin'),
       ('admin:audit:list', '操作日志查询', 'admin'),
       ('admin:login-log:list', '登录日志查询', 'admin'),
       ('admin:content:list', '内容列表', 'content'),
       ('admin:content:detail', '内容详情', 'content'),
       ('admin:content:takedown', '内容下架/恢复', 'content'),
       ('admin:content:review', '内容审核', 'content'),
       ('admin:cuser:list', '用户列表', 'user'),
       ('admin:cuser:detail', '用户详情', 'user'),
       ('admin:cuser:ban', '用户封禁/恢复', 'user')
ON DUPLICATE KEY UPDATE name = VALUES(name);

INSERT INTO ran_feed_admin_role_permission (role_id, permission_id)
SELECT r.id, p.id
FROM ran_feed_admin_role r
         JOIN ran_feed_admin_permission p ON p.is_deleted = 0
WHERE r.code = 'super'
  AND r.is_deleted = 0
  AND p.module IN ('admin', 'content', 'user')
  AND NOT EXISTS (SELECT 1
                  FROM ran_feed_admin_role_permission rp
                  WHERE rp.role_id = r.id
                    AND rp.permission_id = p.id);

INSERT INTO ran_feed_admin_role_permission (role_id, permission_id)
SELECT r.id, p.id
FROM ran_feed_admin_role r
         JOIN ran_feed_admin_permission p ON p.is_deleted = 0
WHERE r.is_deleted = 0
  AND ((r.code = 'readonly_viewer' AND p.code IN ('admin:content:list', 'admin:content:detail',
                                                   'admin:cuser:list', 'admin:cuser:detail',
                                                   'admin:user:list', 'admin:user:detail',
                                                   'admin:role:list', 'admin:role:detail',
                                                   'admin:permission:list',
                                                   'admin:audit:list', 'admin:login-log:list'))
    OR (r.code = 'content_operator' AND p.code IN ('admin:content:list', 'admin:content:detail', 'admin:content:takedown', 'admin:content:review'))
    OR (r.code = 'user_operator' AND p.code IN ('admin:cuser:list', 'admin:cuser:detail', 'admin:cuser:ban'))
    OR (r.code = 'auditor' AND p.code IN ('admin:audit:list', 'admin:login-log:list')))
  AND NOT EXISTS (SELECT 1
                  FROM ran_feed_admin_role_permission rp
                  WHERE rp.role_id = r.id
                    AND rp.permission_id = p.id);

INSERT INTO ran_feed_admin_user (username, password_hash, nickname, status)
VALUES ('admin', '$2a$10$HUS8x3M9wg8XJ5I/XUaBReK3es7FHj4OacLNquQEWplMTOoPHPDA.', '超级管理员', 10),
       ('viewer', '$2a$10$Ws/iJ4n8RqXdyG8i3Svbkuug4lb.Up1u4ZB.IUqCK6NfPa.qHveRu', '访客', 10),
       ('content_op', '$2a$10$Ws/iJ4n8RqXdyG8i3Svbkuug4lb.Up1u4ZB.IUqCK6NfPa.qHveRu', '内容运营', 10),
       ('user_op', '$2a$10$Ws/iJ4n8RqXdyG8i3Svbkuug4lb.Up1u4ZB.IUqCK6NfPa.qHveRu', '用户运营', 10),
       ('auditor', '$2a$10$Ws/iJ4n8RqXdyG8i3Svbkuug4lb.Up1u4ZB.IUqCK6NfPa.qHveRu', '审计员', 10)
ON DUPLICATE KEY UPDATE nickname = VALUES(nickname);

INSERT INTO ran_feed_admin_user_role (admin_user_id, role_id)
SELECT u.id, r.id
FROM ran_feed_admin_user u
         JOIN ran_feed_admin_role r ON r.is_deleted = 0
WHERE u.is_deleted = 0
  AND ((u.username = 'admin' AND r.code = 'super')
    OR (u.username = 'viewer' AND r.code = 'readonly_viewer')
    OR (u.username = 'content_op' AND r.code = 'content_operator')
    OR (u.username = 'user_op' AND r.code = 'user_operator')
    OR (u.username = 'auditor' AND r.code = 'auditor'))
  AND NOT EXISTS (SELECT 1
                  FROM ran_feed_admin_user_role ur
                  WHERE ur.admin_user_id = u.id
                    AND ur.role_id = r.id);
