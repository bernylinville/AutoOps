-- N9E (夜莺监控) 集成迁移脚本
-- 执行前请备份数据库

-- 1. N9E 连接配置表
CREATE TABLE IF NOT EXISTS `n9e_config` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `endpoint` varchar(500) NOT NULL COMMENT 'N9E API 地址',
  `token` varchar(500) NOT NULL COMMENT 'X-User-Token',
  `timeout` int DEFAULT 30 COMMENT '请求超时(秒)',
  `sync_cron` varchar(50) DEFAULT '' COMMENT '自动同步 Cron 表达式',
  `enabled` tinyint DEFAULT 0 COMMENT '是否启用',
  `last_sync_time` datetime(3) DEFAULT NULL COMMENT '最后同步时间',
  `last_sync_result` text COMMENT '最后同步结果 JSON',
  `create_time` datetime(3) NOT NULL,
  `update_time` datetime(3) NOT NULL,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='N9E 连接配置';

-- 1.1 N9E 同步日志表
CREATE TABLE IF NOT EXISTS `n9e_sync_log` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `sync_type` varchar(20) NOT NULL DEFAULT 'full' COMMENT '同步类型(full/hosts/busi_groups/datasources)',
  `status` varchar(20) NOT NULL DEFAULT 'success' COMMENT '状态(success/failed)',
  `result_json` text COMMENT '同步结果 JSON',
  `error_msg` text COMMENT '错误信息',
  `duration_ms` int DEFAULT 0 COMMENT '耗时(毫秒)',
  `trigger_by` varchar(20) DEFAULT 'manual' COMMENT '触发方式(manual/cron)',
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  KEY `idx_sync_log_created` (`created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='N9E 同步日志';

-- 2. N9E 业务组表
CREATE TABLE IF NOT EXISTS `n9e_busi_group` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `n9e_group_id` bigint NOT NULL COMMENT 'N9E 业务组 ID',
  `name` varchar(200) NOT NULL COMMENT '业务组名称',
  `create_time` datetime(3) NOT NULL,
  `update_time` datetime(3) NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_n9e_group_id` (`n9e_group_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='N9E 业务组';

-- 3. N9E 数据源表
CREATE TABLE IF NOT EXISTS `n9e_datasource` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `n9e_source_id` bigint NOT NULL COMMENT 'N9E 数据源 ID',
  `name` varchar(200) NOT NULL COMMENT '数据源名称',
  `plugin_type` varchar(50) DEFAULT '' COMMENT '插件类型(prometheus/victoriametrics)',
  `category` varchar(50) DEFAULT '' COMMENT '分类',
  `url` varchar(500) DEFAULT '' COMMENT 'HTTP URL',
  `status` varchar(20) DEFAULT '' COMMENT '状态',
  `create_time` datetime(3) NOT NULL,
  `update_time` datetime(3) NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_n9e_source_id` (`n9e_source_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='N9E 数据源';

-- 4. 扩展 cmdb_host 表（添加 N9E 相关字段）
-- MySQL 8.0 不支持 ADD COLUMN IF NOT EXISTS，使用存储过程实现
DELIMITER //
CREATE PROCEDURE add_n9e_columns_to_cmdb_host()
BEGIN
    -- 添加 source_type 字段
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_schema = DATABASE() AND table_name = 'cmdb_host' AND column_name = 'source_type') THEN
        ALTER TABLE `cmdb_host` ADD COLUMN `source_type` varchar(20) DEFAULT 'manual' COMMENT '数据来源(manual/aliyun/tencent/n9e)';
    END IF;

    -- 添加 n9e_id 字段
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_schema = DATABASE() AND table_name = 'cmdb_host' AND column_name = 'n9e_id') THEN
        ALTER TABLE `cmdb_host` ADD COLUMN `n9e_id` bigint DEFAULT NULL COMMENT 'N9E Target ID';
    END IF;

    -- 添加 n9e_ident 字段
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_schema = DATABASE() AND table_name = 'cmdb_host' AND column_name = 'n9e_ident') THEN
        ALTER TABLE `cmdb_host` ADD COLUMN `n9e_ident` varchar(128) DEFAULT NULL COMMENT 'N9E Target 标识';
    END IF;

    -- 添加索引 (忽略已存在的错误)
    IF NOT EXISTS (SELECT 1 FROM information_schema.statistics WHERE table_schema = DATABASE() AND table_name = 'cmdb_host' AND index_name = 'idx_cmdb_host_source_type') THEN
        ALTER TABLE `cmdb_host` ADD INDEX `idx_cmdb_host_source_type` (`source_type`);
    END IF;

    IF NOT EXISTS (SELECT 1 FROM information_schema.statistics WHERE table_schema = DATABASE() AND table_name = 'cmdb_host' AND index_name = 'idx_cmdb_host_n9e_ident') THEN
        ALTER TABLE `cmdb_host` ADD INDEX `idx_cmdb_host_n9e_ident` (`n9e_ident`);
    END IF;
END //
DELIMITER ;

CALL add_n9e_columns_to_cmdb_host();
DROP PROCEDURE IF EXISTS add_n9e_columns_to_cmdb_host;

-- 5. 添加 N9E 菜单项
-- 系统管理(id=4) 下添加 N9E 配置
INSERT INTO `sys_menu` (`id`, `parent_id`, `menu_name`, `icon`, `value`, `menu_type`, `url`, `menu_status`, `sort`, `create_time`)
VALUES (260, 4, 'N9E 配置', 'Connection', '', 2, 'system/n9e', 2, 10, NOW())
ON DUPLICATE KEY UPDATE menu_name=VALUES(menu_name), url=VALUES(url);

-- 监控中心(id=212) 下添加 N9E 监控
INSERT INTO `sys_menu` (`id`, `parent_id`, `menu_name`, `icon`, `value`, `menu_type`, `url`, `menu_status`, `sort`, `create_time`)
VALUES (261, 212, 'N9E 监控', 'Monitor', '', 2, 'monitor/n9e', 2, 3, NOW())
ON DUPLICATE KEY UPDATE menu_name=VALUES(menu_name), url=VALUES(url);

-- 监控中心(id=212) 下添加 数据源管理
INSERT INTO `sys_menu` (`id`, `parent_id`, `menu_name`, `icon`, `value`, `menu_type`, `url`, `menu_status`, `sort`, `create_time`)
VALUES (262, 212, '数据源管理', 'Coin', '', 2, 'monitor/datasource', 2, 4, NOW())
ON DUPLICATE KEY UPDATE menu_name=VALUES(menu_name), url=VALUES(url);

-- 6. 为管理员角色(role_id=1)关联 N9E 菜单
INSERT IGNORE INTO `sys_role_menu` (`role_id`, `menu_id`) VALUES (1, 260), (1, 261), (1, 262);

-- 7. 监控中心(id=212) 下添加 CMDB 总览
INSERT INTO `sys_menu` (`id`, `parent_id`, `menu_name`, `icon`, `value`, `menu_type`, `url`, `menu_status`, `sort`, `create_time`)
VALUES (263, 212, 'CMDB 总览', 'DataAnalysis', '', 2, 'monitor/n9e-overview', 2, 1, NOW())
ON DUPLICATE KEY UPDATE menu_name=VALUES(menu_name), url=VALUES(url);

INSERT IGNORE INTO `sys_role_menu` (`role_id`, `menu_id`) VALUES (1, 263);

