-- PostgreSQL 初始化脚本
-- 此脚本在 PostgreSQL 容器首次启动时执行
-- 表结构由 GORM AutoMigrate 自动创建，此处仅做扩展和基础设置

-- 启用常用扩展（为未来功能做准备）
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";      -- UUID 生成
CREATE EXTENSION IF NOT EXISTS "pg_trgm";         -- 模糊搜索（知识库全文搜索）

-- 设置时区
SET timezone = 'Asia/Shanghai';

-- 性能配置（开发环境）
ALTER SYSTEM SET shared_buffers = '256MB';
ALTER SYSTEM SET effective_cache_size = '512MB';
ALTER SYSTEM SET work_mem = '16MB';
