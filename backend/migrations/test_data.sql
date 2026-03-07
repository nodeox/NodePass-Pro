-- 测试数据插入脚本
-- 用于测试节点分组功能

-- 假设已有用户 ID 为 1
DO $$
DECLARE
    test_user_id INTEGER := 1;
    entry_group_id INTEGER;
    exit_group_id INTEGER;
    node_instance_id INTEGER;
BEGIN
    -- 检查用户是否存在
    IF NOT EXISTS (SELECT 1 FROM users WHERE id = test_user_id) THEN
        RAISE NOTICE '用户 ID % 不存在，跳过测试数据插入', test_user_id;
        RETURN;
    END IF;

    RAISE NOTICE '开始插入测试数据...';

    -- 1. 创建入口节点组
    INSERT INTO node_groups (user_id, name, description, type, config, enabled)
    VALUES (
        test_user_id,
        '测试入口组-亚洲',
        '亚洲地区入口节点组，用于测试',
        'entry',
        jsonb_build_object(
            'listen_address', '0.0.0.0',
            'port_range', jsonb_build_object('start', 10000, 'end', 20000),
            'allowed_protocols', jsonb_build_array('tcp', 'udp', 'both'),
            'entry_config', jsonb_build_object(
                'require_exit_group', true,
                'allowed_exit_groups', jsonb_build_array(),
                'traffic_multiplier', 1.0,
                'display_address', 'asia.nodepass.example.com',
                'dns_load_balance', true
            )
        ),
        true
    )
    RETURNING id INTO entry_group_id;

    RAISE NOTICE '✓ 创建入口节点组: ID=%', entry_group_id;

    -- 2. 创建出口节点组
    INSERT INTO node_groups (user_id, name, description, type, config, enabled)
    VALUES (
        test_user_id,
        '测试出口组-美国',
        '美国地区出口节点组，用于测试',
        'exit',
        jsonb_build_object(
            'allowed_protocols', jsonb_build_array('tcp', 'udp', 'both'),
            'exit_config', jsonb_build_object(
                'allowed_entry_groups', jsonb_build_array(),
                'load_balance_strategy', 'round_robin',
                'health_check_interval', 30,
                'health_check_timeout', 5
            )
        ),
        true
    )
    RETURNING id INTO exit_group_id;

    RAISE NOTICE '✓ 创建出口节点组: ID=%', exit_group_id;

    -- 3. 创建节点组关联
    INSERT INTO node_group_relations (entry_group_id, exit_group_id, priority, enabled)
    VALUES (entry_group_id, exit_group_id, 10, true);

    RAISE NOTICE '✓ 创建节点组关联';

    -- 4. 更新入口组的允许出口组列表
    UPDATE node_groups
    SET config = jsonb_set(
        config,
        '{entry_config,allowed_exit_groups}',
        jsonb_build_array(exit_group_id)
    )
    WHERE id = entry_group_id;

    -- 5. 更新出口组的允许入口组列表
    UPDATE node_groups
    SET config = jsonb_set(
        config,
        '{exit_config,allowed_entry_groups}',
        jsonb_build_array(entry_group_id)
    )
    WHERE id = exit_group_id;

    RAISE NOTICE '✓ 更新节点组关联配置';

    -- 6. 创建入口节点实例
    INSERT INTO node_instances (
        group_id, user_id, node_id, service_name,
        connection_address, ipv4, ipv6,
        debug_mode, auto_start, status
    )
    VALUES (
        entry_group_id,
        test_user_id,
        'entry-node-001-' || gen_random_uuid()::text,
        'nodepass-entry-001',
        'entry1.asia.nodepass.example.com',
        '203.0.113.10',
        '2001:db8::1',
        false,
        true,
        'online'
    )
    RETURNING id INTO node_instance_id;

    -- 更新心跳时间
    UPDATE node_instances
    SET last_heartbeat = CURRENT_TIMESTAMP,
        system_info = jsonb_build_object(
            'cpu_usage', 25.5,
            'memory_usage', 45.2,
            'disk_usage', 60.0,
            'network_in', 1024000,
            'network_out', 2048000,
            'uptime', 86400
        ),
        traffic_stats = jsonb_build_object(
            'total_in', 1073741824,
            'total_out', 2147483648,
            'connections', 150
        )
    WHERE id = node_instance_id;

    RAISE NOTICE '✓ 创建入口节点实例: ID=%', node_instance_id;

    -- 7. 创建第二个入口节点实例（用于测试负载均衡）
    INSERT INTO node_instances (
        group_id, user_id, node_id, service_name,
        connection_address, ipv4,
        debug_mode, auto_start, status,
        last_heartbeat,
        system_info,
        traffic_stats
    )
    VALUES (
        entry_group_id,
        test_user_id,
        'entry-node-002-' || gen_random_uuid()::text,
        'nodepass-entry-002',
        'entry2.asia.nodepass.example.com',
        '203.0.113.11',
        false,
        true,
        'online',
        CURRENT_TIMESTAMP,
        jsonb_build_object(
            'cpu_usage', 30.2,
            'memory_usage', 50.8,
            'disk_usage', 55.0,
            'network_in', 2048000,
            'network_out', 4096000,
            'uptime', 172800
        ),
        jsonb_build_object(
            'total_in', 2147483648,
            'total_out', 4294967296,
            'connections', 200
        )
    );

    RAISE NOTICE '✓ 创建第二个入口节点实例';

    -- 8. 创建出口节点实例
    INSERT INTO node_instances (
        group_id, user_id, node_id, service_name,
        connection_address, ipv4,
        exit_network,
        debug_mode, auto_start, status,
        last_heartbeat,
        system_info,
        traffic_stats
    )
    VALUES (
        exit_group_id,
        test_user_id,
        'exit-node-001-' || gen_random_uuid()::text,
        'nodepass-exit-001',
        'exit1.us.nodepass.example.com',
        '198.51.100.10',
        'eth0',
        false,
        true,
        'online',
        CURRENT_TIMESTAMP,
        jsonb_build_object(
            'cpu_usage', 20.0,
            'memory_usage', 40.0,
            'disk_usage', 50.0,
            'network_in', 5120000,
            'network_out', 10240000,
            'uptime', 259200
        ),
        jsonb_build_object(
            'total_in', 5368709120,
            'total_out', 10737418240,
            'connections', 300
        )
    );

    RAISE NOTICE '✓ 创建出口节点实例';

    -- 9. 创建测试隧道
    INSERT INTO tunnels (
        user_id, name, description,
        entry_group_id, exit_group_id,
        protocol, local_port, remote_port, remote_host,
        bandwidth_limit, timeout, keepalive,
        status, enabled
    )
    VALUES (
        test_user_id,
        '测试隧道-SSH',
        'SSH 远程连接隧道',
        entry_group_id,
        exit_group_id,
        'tcp',
        10022,
        22,
        '192.168.1.100',
        100,
        300,
        true,
        'running',
        true
    );

    RAISE NOTICE '✓ 创建测试隧道: SSH';

    -- 10. 创建第二个测试隧道
    INSERT INTO tunnels (
        user_id, name, description,
        entry_group_id, exit_group_id,
        protocol, local_port, remote_port, remote_host,
        keepalive,
        status, enabled
    )
    VALUES (
        test_user_id,
        '测试隧道-HTTP',
        'HTTP Web 服务隧道',
        entry_group_id,
        exit_group_id,
        'tcp',
        10080,
        80,
        '192.168.1.200',
        true,
        'stopped',
        true
    );

    RAISE NOTICE '✓ 创建测试隧道: HTTP';

    -- 11. 创建一个离线节点（用于测试）
    INSERT INTO node_instances (
        group_id, user_id, node_id, service_name,
        connection_address, ipv4,
        debug_mode, auto_start, status,
        last_heartbeat
    )
    VALUES (
        entry_group_id,
        test_user_id,
        'entry-node-003-' || gen_random_uuid()::text,
        'nodepass-entry-003',
        'entry3.asia.nodepass.example.com',
        '203.0.113.12',
        false,
        true,
        'offline',
        CURRENT_TIMESTAMP - INTERVAL '10 minutes'
    );

    RAISE NOTICE '✓ 创建离线节点实例（用于测试）';

    RAISE NOTICE '';
    RAISE NOTICE '========================================';
    RAISE NOTICE '测试数据插入完成！';
    RAISE NOTICE '========================================';
    RAISE NOTICE '';
    RAISE NOTICE '创建的资源:';
    RAISE NOTICE '  - 入口节点组: % (ID: %)', '测试入口组-亚洲', entry_group_id;
    RAISE NOTICE '  - 出口节点组: % (ID: %)', '测试出口组-美国', exit_group_id;
    RAISE NOTICE '  - 入口节点实例: 3 个 (2 在线, 1 离线)';
    RAISE NOTICE '  - 出口节点实例: 1 个 (在线)';
    RAISE NOTICE '  - 隧道: 2 个 (1 运行中, 1 已停止)';
    RAISE NOTICE '';
    RAISE NOTICE '查询命令:';
    RAISE NOTICE '  - 查看节点组: SELECT * FROM node_groups_with_stats;';
    RAISE NOTICE '  - 查看节点实例: SELECT * FROM node_instances WHERE group_id IN (%, %);', entry_group_id, exit_group_id;
    RAISE NOTICE '  - 查看隧道: SELECT * FROM tunnels_with_groups;';
    RAISE NOTICE '  - 查看统计: SELECT * FROM node_group_stats;';

END $$;
