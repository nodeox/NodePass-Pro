-- 更新 tunnels 表，添加新字段

-- 添加 description 字段
ALTER TABLE tunnels ADD COLUMN IF NOT EXISTS description TEXT;

-- 添加 listen_host 字段
ALTER TABLE tunnels ADD COLUMN IF NOT EXISTS listen_host VARCHAR(255) DEFAULT '0.0.0.0';

-- 修改 exit_group_id 为可空（支持不带出口节点组模式）
ALTER TABLE tunnels ALTER COLUMN exit_group_id DROP NOT NULL;

-- 更新现有记录的 listen_host
UPDATE tunnels SET listen_host = '0.0.0.0' WHERE listen_host IS NULL OR listen_host = '';
