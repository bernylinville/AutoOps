-- 审计日志菜单 (parent_id=44 即 "操作审计")
-- 注意: sys_menu 表列名为 menu_name, value, menu_type, menu_status
-- 注意: sys_role_menu 表列名为 role_id, menu_id

-- 菜单页面 (使用 auto_increment 或手动指定未占用 ID)
INSERT INTO `sys_menu` (`parent_id`, `menu_name`, `icon`, `value`, `menu_type`, `url`, `menu_status`, `sort`, `create_time`)
VALUES (44, '审计日志', 'Memo', 'monitor:audit:list', 2, 'monitor/audit-log', 2, 5, NOW());

-- 获取刚插入的菜单 ID (假设为 @audit_menu_id)
SET @audit_menu_id = LAST_INSERT_ID();

-- 按钮权限
INSERT INTO `sys_menu` (`parent_id`, `menu_name`, `icon`, `value`, `menu_type`, `url`, `menu_status`, `sort`, `create_time`)
VALUES (@audit_menu_id, '查看审计日志', '', 'base:audit:view', 3, '', 2, 1, NOW()),
       (@audit_menu_id, '删除审计日志', '', 'base:audit:delete', 3, '', 2, 2, NOW()),
       (@audit_menu_id, '清空审计日志', '', 'base:audit:clean', 3, '', 2, 3, NOW());

-- admin 角色分配 (role_id=1)
INSERT INTO `sys_role_menu` (`role_id`, `menu_id`)
VALUES (1, @audit_menu_id),
       (1, @audit_menu_id + 1),
       (1, @audit_menu_id + 2),
       (1, @audit_menu_id + 3);
