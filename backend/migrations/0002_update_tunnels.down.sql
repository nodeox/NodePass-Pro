-- 回滚 tunnels 表更新

-- 删除 listen_host 字段
ALTER TABLE tunnels DROP COLUMN IF EXISTS listen_host;

-- 删除 description 字段
ALTER TABLE tunnels DROP COLUMN IF EXISTS description;

-- 恢复 exit_group_id 为非空（注意：这可能会失败如果有NULL值）
-- ALTER TABLE tunnels ALTER COLUMN exit_group_id SET NOT NULL;
