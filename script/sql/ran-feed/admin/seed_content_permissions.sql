-- Phase B 内容管理权限点播种（feat-admin-004 手动执行一次）
-- 幂等 可重复执行；super 角色补绑这批 content 权限点

-- 内容管理权限点
INSERT INTO ran_feed_admin_permission (code, name, module)
VALUES ('content:list', '内容列表', 'content'),
       ('content:detail', '内容详情', 'content'),
       ('content:takedown', '内容下架/恢复', 'content')
ON DUPLICATE KEY UPDATE name = VALUES(name);

-- super 补绑 content 模块全部权限点
INSERT INTO ran_feed_admin_role_permission (role_id, permission_id)
SELECT r.id, p.id
FROM ran_feed_admin_role r,
     ran_feed_admin_permission p
WHERE r.code = 'super'
  AND r.is_deleted = 0
  AND p.module = 'content'
  AND p.is_deleted = 0
  AND NOT EXISTS (SELECT 1
                  FROM ran_feed_admin_role_permission rp
                  WHERE rp.role_id = r.id
                    AND rp.permission_id = p.id);