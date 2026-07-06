-- 后台超管播种（手动执行一次）
-- 账号 admin 密码 Admin@123456（password_hash = bcrypt(password + salt) salt=rf_admin_salt_v1）
-- 幂等 可重复执行

-- super 角色
INSERT INTO ran_feed_admin_role (code, name, remark)
VALUES ('super', '超级管理员', '拥有全部权限')
ON DUPLICATE KEY UPDATE name = VALUES(name);

-- Phase A 权限点（管理自身：账号/角色/权限）
INSERT INTO ran_feed_admin_permission (code, name, module)
VALUES ('admin:user:list', '管理员列表', 'admin'),
       ('admin:user:manage', '管理员管理', 'admin'),
       ('admin:role:list', '角色列表', 'admin'),
       ('admin:role:manage', '角色管理', 'admin'),
       ('admin:permission:list', '权限点列表', 'admin')
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
INSERT INTO ran_feed_admin_user (username, password_hash, password_salt, nickname, status)
VALUES ('admin', '$2a$10$zU8zm2jcSvup8p0y.nlu7ON/PKmX/NHciZDi6EV6SOtrY2jMC15Im', 'rf_admin_salt_v1', '超级管理员', 10)
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
