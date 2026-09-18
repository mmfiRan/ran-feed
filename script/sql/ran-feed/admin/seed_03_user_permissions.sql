-- 用户管理权限点
INSERT INTO ran_feed_admin_permission (code, name, module)
VALUES ('admin:cuser:list', '用户列表', 'user'),
       ('admin:cuser:detail', '用户详情', 'user'),
       ('admin:cuser:ban', '用户封禁/恢复', 'user')
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