-- Phase C 用户管理权限点播种（feat-admin-012 手动执行一次）
-- 幂等 可重复执行；super 角色补绑这批 user 权限点

-- 用户管理权限点
INSERT INTO ran_feed_admin_permission (code, name, module)
VALUES ('user:list', '用户列表/详情', 'user'),
       ('user:ban', '用户封禁/恢复', 'user')
ON DUPLICATE KEY UPDATE name = VALUES(name);

-- super 补绑 user 模块全部权限点
INSERT INTO ran_feed_admin_role_permission (role_id, permission_id)
SELECT r.id, p.id
FROM ran_feed_admin_role r,
     ran_feed_admin_permission p
WHERE r.code = 'super'
  AND r.is_deleted = 0
  AND p.module = 'user'
  AND p.is_deleted = 0
  AND NOT EXISTS (SELECT 1
                  FROM ran_feed_admin_role_permission rp
                  WHERE rp.role_id = r.id
                    AND rp.permission_id = p.id);