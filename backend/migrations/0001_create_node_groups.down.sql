-- NodePass Pro 节点分组架构回滚（PostgreSQL）
-- 按依赖倒序删除：触发器/函数 -> tunnels -> stats -> relations -> instances -> groups

DROP TRIGGER IF EXISTS trigger_update_node_group_stats ON node_instances;
DROP FUNCTION IF EXISTS update_node_group_stats();

DROP TABLE IF EXISTS tunnels;
DROP TABLE IF EXISTS node_group_stats;
DROP TABLE IF EXISTS node_group_relations;
DROP TABLE IF EXISTS node_instances;
DROP TABLE IF EXISTS node_groups;
