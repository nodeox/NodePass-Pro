-- 初始化数据库脚本

-- 启用 TimescaleDB 扩展
CREATE EXTENSION IF NOT EXISTS timescaledb;

-- 创建索引优化函数
CREATE OR REPLACE FUNCTION create_optimized_indexes() RETURNS void AS $$
BEGIN
    -- 用户表索引
    CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
    CREATE INDEX IF NOT EXISTS idx_users_username ON users(username);
    CREATE INDEX IF NOT EXISTS idx_users_status ON users(status) WHERE status = 'normal';
    CREATE INDEX IF NOT EXISTS idx_users_vip_level ON users(vip_level) WHERE vip_level > 0;
    
    -- 节点表索引
    CREATE INDEX IF NOT EXISTS idx_node_instances_group_id ON node_instances(group_id);
    CREATE INDEX IF NOT EXISTS idx_node_instances_status ON node_instances(status);
    CREATE INDEX IF NOT EXISTS idx_node_instances_heartbeat ON node_instances(last_heartbeat_at DESC);
    CREATE INDEX IF NOT EXISTS idx_node_instances_group_status ON node_instances(group_id, status);
    
    -- 隧道表索引
    CREATE INDEX IF NOT EXISTS idx_tunnels_user_id ON tunnels(user_id);
    CREATE INDEX IF NOT EXISTS idx_tunnels_status ON tunnels(status);
    CREATE INDEX IF NOT EXISTS idx_tunnels_user_status ON tunnels(user_id, status);
    
    RAISE NOTICE '索引创建完成';
END;
$$ LANGUAGE plpgsql;

-- 创建时序表转换函数
CREATE OR REPLACE FUNCTION convert_to_hypertable() RETURNS void AS $$
BEGIN
    -- 检查表是否存在
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'traffic_records') THEN
        -- 转换为超表
        PERFORM create_hypertable('traffic_records', 'created_at', 
            chunk_time_interval => INTERVAL '1 day',
            if_not_exists => TRUE);
        
        -- 启用压缩
        ALTER TABLE traffic_records SET (
            timescaledb.compress,
            timescaledb.compress_segmentby = 'user_id'
        );
        
        -- 添加压缩策略（压缩 7 天前的数据）
        SELECT add_compression_policy('traffic_records', INTERVAL '7 days', if_not_exists => TRUE);
        
        -- 添加保留策略（删除 90 天前的数据）
        SELECT add_retention_policy('traffic_records', INTERVAL '90 days', if_not_exists => TRUE);
        
        RAISE NOTICE 'traffic_records 已转换为超表';
    END IF;
    
    -- 性能指标表
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'node_performance_metrics') THEN
        PERFORM create_hypertable('node_performance_metrics', 'created_at',
            chunk_time_interval => INTERVAL '1 day',
            if_not_exists => TRUE);
        
        ALTER TABLE node_performance_metrics SET (
            timescaledb.compress,
            timescaledb.compress_segmentby = 'node_instance_id'
        );
        
        SELECT add_compression_policy('node_performance_metrics', INTERVAL '7 days', if_not_exists => TRUE);
        SELECT add_retention_policy('node_performance_metrics', INTERVAL '30 days', if_not_exists => TRUE);
        
        RAISE NOTICE 'node_performance_metrics 已转换为超表';
    END IF;
    
    -- 健康检查记录表
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'node_health_records') THEN
        PERFORM create_hypertable('node_health_records', 'created_at',
            chunk_time_interval => INTERVAL '1 day',
            if_not_exists => TRUE);
        
        ALTER TABLE node_health_records SET (
            timescaledb.compress,
            timescaledb.compress_segmentby = 'node_instance_id'
        );
        
        SELECT add_compression_policy('node_health_records', INTERVAL '7 days', if_not_exists => TRUE);
        SELECT add_retention_policy('node_health_records', INTERVAL '30 days', if_not_exists => TRUE);
        
        RAISE NOTICE 'node_health_records 已转换为超表';
    END IF;
END;
$$ LANGUAGE plpgsql;

-- 创建统计视图
CREATE OR REPLACE VIEW v_user_traffic_stats AS
SELECT 
    user_id,
    DATE_TRUNC('day', created_at) AS date,
    SUM(traffic_in) AS total_in,
    SUM(traffic_out) AS total_out,
    SUM(traffic_in + traffic_out) AS total_traffic,
    COUNT(*) AS record_count
FROM traffic_records
GROUP BY user_id, DATE_TRUNC('day', created_at);

-- 创建节点健康度视图
CREATE OR REPLACE VIEW v_node_health_summary AS
SELECT 
    node_instance_id,
    DATE_TRUNC('hour', created_at) AS hour,
    AVG(response_time) AS avg_response_time,
    AVG(CASE WHEN success THEN 1 ELSE 0 END) * 100 AS success_rate,
    COUNT(*) AS check_count
FROM node_health_records
GROUP BY node_instance_id, DATE_TRUNC('hour', created_at);

-- 输出提示信息
DO $$
BEGIN
    RAISE NOTICE '===========================================';
    RAISE NOTICE 'NodePass Pro 数据库初始化完成';
    RAISE NOTICE '===========================================';
    RAISE NOTICE '提示：';
    RAISE NOTICE '1. 应用启动后会自动执行表迁移';
    RAISE NOTICE '2. 执行 SELECT create_optimized_indexes(); 创建索引';
    RAISE NOTICE '3. 执行 SELECT convert_to_hypertable(); 转换时序表';
    RAISE NOTICE '===========================================';
END $$;
