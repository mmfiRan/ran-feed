-- super 角色
INSERT INTO ran_feed_admin_role (code, name, remark)
VALUES ('super', '超级管理员', '拥有全部权限')
ON DUPLICATE KEY UPDATE name = VALUES(name);

-- Phase A 权限点（管理自身：账号/角色/权限）
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
       ('admin:audit:list', '审计日志查询', 'admin')
ON DUPLICATE KEY UPDATE name = VALUES(name);

-- super 绑定全部权限点
INSERT INTO ran_feed_admin_role_permission (role_id, permission_id)
SELECT r.id, p.id
FROM ran_feed_admin_role r,
     ran_feed_admin_permission p
WHERE r.code = 'super'
  AND r.is_deleted = 0
  AND p.module = 'admin'
  AND p.is_deleted = 0
  AND NOT EXISTS (SELECT 1
                  FROM ran_feed_admin_role_permission rp
                  WHERE rp.role_id = r.id
                    AND rp.permission_id = p.id);

-- 超管账号
INSERT INTO ran_feed_admin_user (username, password_hash, nickname, status)
VALUES ('admin', '$2a$10$My4IMCvnAmNuPN70wIEYu.dzPy4MmSPnHXpQtgdnjDbVhlzolG6/q', '超级管理员', 10)
ON DUPLICATE KEY UPDATE nickname = VALUES(nickname);

-- 超管绑定 super 角色
INSERT INTO ran_feed_admin_user_role (admin_user_id, role_id)
SELECT u.id, r.id
FROM ran_feed_admin_user u,
     ran_feed_admin_role r
WHERE u.username = 'admin'
  AND u.is_deleted = 0
  AND r.code = 'super'
  AND r.is_deleted = 0
  AND NOT EXISTS (SELECT 1
                  FROM ran_feed_admin_user_role ur
                  WHERE ur.admin_user_id = u.id
                    AND ur.role_id = r.id);
