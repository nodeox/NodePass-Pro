-- 测试数据插入脚本（与当前节点分组 schema 对齐）
-- 适用于 PostgreSQL

DO $$
DECLARE
    test_user_id BIGINT := 1;
    entry_group_id BIGINT;
    exit_group_id BIGINT;
BEGIN
    IF NOT EXISTS (SELECT 1 FROM users WHERE id = test_user_id) THEN
        RAISE NOTICE '用户 ID % 不存在，跳过测试数据插入', test_user_id;
        RETURN;
    END IF;

    RAISE NOTICE '开始插入节点分组测试数据...';

    -- 1) 入口节点组
    INSERT INTO node_groups (user_id, name, type, description, is_enabled, config)
    VALUES (
        test_user_id,
        '测试入口组-亚洲',
        'entry',
        '亚洲入口节点组（测试）',
        TRUE,
        jsonb_build_object(
            'allowed_protocols', jsonb_build_array('tcp', 'udp'),
            'port_range', jsonb_build_object('start', 10000, 'end', 20000),
            'entry_config', jsonb_build_object(
                'require_exit_group', TRUE,
                'traffic_multiplier', 1.0,
                'dns_load_balance', TRUE
            )
        )::text
    )
    RETURNING id INTO entry_group_id;

    -- 2) 出口节点组
    INSERT INTO node_groups (user_id, name, type, description, is_enabled, config)
    VALUES (
        test_user_id,
        '测试出口组-美国',
        'exit',
        '美国出口节点组（测试）',
        TRUE,
        jsonb_build_object(
            'allowed_protocols', jsonb_build_array('tcp', 'udp'),
            'port_range', jsonb_build_object('start', 10000, 'end', 20000),
            'exit_config', jsonb_build_object(
                'load_balance_strategy', 'round_robin',
                'health_check_interval', 30,
                'health_check_timeout', 5
            )
        )::text
    )
    RETURNING id INTO exit_group_id;

    -- 3) 入口组与出口组关联
    INSERT INTO node_group_relations (entry_group_id, exit_group_id, is_enabled)
    VALUES (entry_group_id, exit_group_id, TRUE);

    -- 4) 入口节点实例（在线）
    INSERT INTO node_instances (
        node_group_id, node_id, auth_token_hash, name, host, port, status, is_enabled,
        system_info, traffic_stats, config_version, last_heartbeat_at
    ) VALUES (
        entry_group_id,
        'entry-node-1-' || substring(md5(random()::text || clock_timestamp()::text) from 1 for 12),
        substring(md5(random()::text || clock_timestamp()::text) from 1 for 64),
        'nodepass-entry-001',
        '203.0.113.10',
        31001,
        'online',
        TRUE,
        jsonb_build_object(
            'cpu_usage', 25.5,
            'memory_usage', 45.2,
            'disk_usage', 60.0
        )::text,
        jsonb_build_object(
            'traffic_in', 1073741824,
            'traffic_out', 2147483648,
            'connections', 150
        )::text,
        3,
        CURRENT_TIMESTAMP
    );

    -- 5) 入口节点实例（在线，第二台）
    INSERT INTO node_instances (
        node_group_id, node_id, auth_token_hash, name, host, port, status, is_enabled,
        system_info, traffic_stats, config_version, last_heartbeat_at
    ) VALUES (
        entry_group_id,
        'entry-node-2-' || substring(md5(random()::text || clock_timestamp()::text) from 1 for 12),
        substring(md5(random()::text || clock_timestamp()::text) from 1 for 64),
        'nodepass-entry-002',
        '203.0.113.11',
        31002,
        'online',
        TRUE,
        jsonb_build_object(
            'cpu_usage', 30.2,
            'memory_usage', 50.8,
            'disk_usage', 55.0
        )::text,
        jsonb_build_object(
            'traffic_in', 2147483648,
            'traffic_out', 4294967296,
            'connections', 200
        )::text,
        3,
        CURRENT_TIMESTAMP
    );

    -- 6) 入口节点实例（离线）
    INSERT INTO node_instances (
        node_group_id, node_id, auth_token_hash, name, host, port, status, is_enabled,
        system_info, traffic_stats, config_version, last_heartbeat_at
    ) VALUES (
        entry_group_id,
        'entry-node-3-' || substring(md5(random()::text || clock_timestamp()::text) from 1 for 12),
        substring(md5(random()::text || clock_timestamp()::text) from 1 for 64),
        'nodepass-entry-003',
        '203.0.113.12',
        31003,
        'offline',
        TRUE,
        '{}'::jsonb::text,
        '{}'::jsonb::text,
        3,
        CURRENT_TIMESTAMP - INTERVAL '10 minutes'
    );

    -- 7) 出口节点实例（在线）
    INSERT INTO node_instances (
        node_group_id, node_id, auth_token_hash, name, host, port, status, is_enabled,
        system_info, traffic_stats, config_version, last_heartbeat_at
    ) VALUES (
        exit_group_id,
        'exit-node-1-' || substring(md5(random()::text || clock_timestamp()::text) from 1 for 12),
        substring(md5(random()::text || clock_timestamp()::text) from 1 for 64),
        'nodepass-exit-001',
        '198.51.100.10',
        32001,
        'online',
        TRUE,
        jsonb_build_object(
            'cpu_usage', 20.0,
            'memory_usage', 40.0,
            'disk_usage', 50.0
        )::text,
        jsonb_build_object(
            'traffic_in', 5368709120,
            'traffic_out', 10737418240,
            'connections', 300
        )::text,
        3,
        CURRENT_TIMESTAMP
    );

    -- 8) 创建两条测试隧道
    INSERT INTO tunnels (
        user_id, name, description, entry_group_id, exit_group_id,
        protocol, listen_host, listen_port, remote_host, remote_port,
        status, traffic_in, traffic_out, config_json
    ) VALUES (
        test_user_id,
        '测试隧道-SSH',
        'SSH 远程连接隧道',
        entry_group_id,
        exit_group_id,
        'tcp',
        '0.0.0.0',
        10022,
        '192.168.1.100',
        22,
        'running',
        123456789,
        987654321,
        jsonb_build_object(
            'load_balance_strategy', 'round_robin',
            'ip_type', 'auto',
            'enable_proxy_protocol', FALSE,
            'forward_targets', jsonb_build_array()
        )::text
    );

    INSERT INTO tunnels (
        user_id, name, description, entry_group_id, exit_group_id,
        protocol, listen_host, listen_port, remote_host, remote_port,
        status, traffic_in, traffic_out, config_json
    ) VALUES (
        test_user_id,
        '测试隧道-HTTP',
        'HTTP Web 服务隧道',
        entry_group_id,
        exit_group_id,
        'tcp',
        '0.0.0.0',
        10080,
        '192.168.1.200',
        80,
        'stopped',
        0,
        0,
        jsonb_build_object(
            'load_balance_strategy', 'least_connections',
            'ip_type', 'auto',
            'enable_proxy_protocol', TRUE,
            'forward_targets', jsonb_build_array()
        )::text
    );

    -- 9) 汇总统计（含流量与连接）
    WITH stats_source AS (
        SELECT
            ni.node_group_id,
            COUNT(*)::INTEGER AS total_nodes,
            COUNT(*) FILTER (WHERE ni.status = 'online')::INTEGER AS online_nodes,
            COALESCE(SUM((COALESCE(NULLIF(ni.traffic_stats, ''), '{}')::jsonb ->> 'traffic_in')::BIGINT), 0) AS total_traffic_in,
            COALESCE(SUM((COALESCE(NULLIF(ni.traffic_stats, ''), '{}')::jsonb ->> 'traffic_out')::BIGINT), 0) AS total_traffic_out,
            COALESCE(SUM((COALESCE(NULLIF(ni.traffic_stats, ''), '{}')::jsonb ->> 'connections')::INTEGER), 0) AS total_connections
        FROM node_instances ni
        WHERE ni.node_group_id IN (entry_group_id, exit_group_id)
        GROUP BY ni.node_group_id
    )
    INSERT INTO node_group_stats (
        node_group_id,
        total_nodes,
        online_nodes,
        total_traffic_in,
        total_traffic_out,
        total_connections,
        updated_at
    )
    SELECT
        ss.node_group_id,
        ss.total_nodes,
        ss.online_nodes,
        ss.total_traffic_in,
        ss.total_traffic_out,
        ss.total_connections,
        CURRENT_TIMESTAMP
    FROM stats_source ss
    ON CONFLICT (node_group_id)
    DO UPDATE SET
        total_nodes = EXCLUDED.total_nodes,
        online_nodes = EXCLUDED.online_nodes,
        total_traffic_in = EXCLUDED.total_traffic_in,
        total_traffic_out = EXCLUDED.total_traffic_out,
        total_connections = EXCLUDED.total_connections,
        updated_at = EXCLUDED.updated_at;

    RAISE NOTICE '测试数据插入完成。';
    RAISE NOTICE '入口组 ID: %, 出口组 ID: %', entry_group_id, exit_group_id;
    RAISE NOTICE '验证建议:';
    RAISE NOTICE '  SELECT id, name, type, is_enabled FROM node_groups WHERE id IN (%, %);', entry_group_id, exit_group_id;
    RAISE NOTICE '  SELECT node_group_id, name, status, host, port FROM node_instances WHERE node_group_id IN (%, %);', entry_group_id, exit_group_id;
    RAISE NOTICE '  SELECT id, name, status, listen_host, listen_port, remote_host, remote_port FROM tunnels WHERE entry_group_id = %;', entry_group_id;
    RAISE NOTICE '  SELECT * FROM node_group_stats WHERE node_group_id IN (%, %);', entry_group_id, exit_group_id;
END $$;
