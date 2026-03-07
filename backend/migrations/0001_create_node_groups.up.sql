-- NodePass Pro 节点分组架构迁移（PostgreSQL）
--
-- 兼容性说明：
-- 1) 本文件使用 PostgreSQL 语法（BIGSERIAL、PL/pgSQL 函数、EXECUTE FUNCTION 触发器）。
-- 2) SQLite 不支持 CREATE FUNCTION ... LANGUAGE plpgsql。
--    如需 SQLite，请改为应用层更新统计，或使用 SQLite 触发器语法重写。

-- ======================
-- 1) node_groups
-- ======================
CREATE TABLE IF NOT EXISTS node_groups (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    type VARCHAR(20) NOT NULL CHECK (type IN ('entry', 'exit')),
    description TEXT,
    is_enabled BOOLEAN NOT NULL DEFAULT TRUE,
    config TEXT NOT NULL DEFAULT '{}',
    created_at TIMESTAMP WITHOUT TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITHOUT TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_node_groups_user_id ON node_groups(user_id);
CREATE INDEX IF NOT EXISTS idx_node_groups_type ON node_groups(type);
CREATE INDEX IF NOT EXISTS idx_node_groups_is_enabled ON node_groups(is_enabled);
CREATE INDEX IF NOT EXISTS idx_node_groups_user_type ON node_groups(user_id, type);

-- ======================
-- 2) node_instances
-- ======================
CREATE TABLE IF NOT EXISTS node_instances (
    id BIGSERIAL PRIMARY KEY,
    node_group_id BIGINT NOT NULL REFERENCES node_groups(id) ON DELETE CASCADE,
    node_id VARCHAR(100) NOT NULL UNIQUE,
    auth_token_hash VARCHAR(255),
    name VARCHAR(100) NOT NULL,
    host VARCHAR(255),
    port INTEGER,
    status VARCHAR(20) NOT NULL DEFAULT 'offline' CHECK (status IN ('online', 'offline', 'maintain')),
    is_enabled BOOLEAN NOT NULL DEFAULT TRUE,
    system_info TEXT,
    traffic_stats TEXT,
    config_version INTEGER NOT NULL DEFAULT 0,
    last_heartbeat_at TIMESTAMP WITHOUT TIME ZONE,
    created_at TIMESTAMP WITHOUT TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITHOUT TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_node_instances_node_group_id ON node_instances(node_group_id);
CREATE INDEX IF NOT EXISTS idx_node_instances_auth_token_hash ON node_instances(auth_token_hash);
CREATE INDEX IF NOT EXISTS idx_node_instances_status ON node_instances(status);
CREATE INDEX IF NOT EXISTS idx_node_instances_is_enabled ON node_instances(is_enabled);
CREATE INDEX IF NOT EXISTS idx_node_instances_group_status ON node_instances(node_group_id, status);

-- ======================
-- 3) node_group_relations
-- ======================
CREATE TABLE IF NOT EXISTS node_group_relations (
    id BIGSERIAL PRIMARY KEY,
    entry_group_id BIGINT NOT NULL REFERENCES node_groups(id) ON DELETE CASCADE,
    exit_group_id BIGINT NOT NULL REFERENCES node_groups(id) ON DELETE CASCADE,
    is_enabled BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMP WITHOUT TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT uk_node_group_relations_entry_exit UNIQUE (entry_group_id, exit_group_id)
);

CREATE INDEX IF NOT EXISTS idx_node_group_relations_entry_group_id ON node_group_relations(entry_group_id);
CREATE INDEX IF NOT EXISTS idx_node_group_relations_exit_group_id ON node_group_relations(exit_group_id);
CREATE INDEX IF NOT EXISTS idx_node_group_relations_is_enabled ON node_group_relations(is_enabled);

-- ======================
-- 4) node_group_stats
-- ======================
CREATE TABLE IF NOT EXISTS node_group_stats (
    id BIGSERIAL PRIMARY KEY,
    node_group_id BIGINT NOT NULL UNIQUE REFERENCES node_groups(id) ON DELETE CASCADE,
    total_nodes INTEGER NOT NULL DEFAULT 0,
    online_nodes INTEGER NOT NULL DEFAULT 0,
    total_traffic_in BIGINT NOT NULL DEFAULT 0,
    total_traffic_out BIGINT NOT NULL DEFAULT 0,
    total_connections INTEGER NOT NULL DEFAULT 0,
    updated_at TIMESTAMP WITHOUT TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_node_group_stats_node_group_id ON node_group_stats(node_group_id);
CREATE INDEX IF NOT EXISTS idx_node_group_stats_updated_at ON node_group_stats(updated_at);

-- ======================
-- 5) tunnels
-- ======================
CREATE TABLE IF NOT EXISTS tunnels (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    entry_group_id BIGINT NOT NULL REFERENCES node_groups(id) ON DELETE CASCADE,
    exit_group_id BIGINT NOT NULL REFERENCES node_groups(id) ON DELETE CASCADE,
    protocol VARCHAR(20) NOT NULL,
    remote_host VARCHAR(255) NOT NULL,
    remote_port INTEGER NOT NULL,
    listen_port INTEGER,
    status VARCHAR(20) NOT NULL DEFAULT 'stopped',
    traffic_in BIGINT NOT NULL DEFAULT 0,
    traffic_out BIGINT NOT NULL DEFAULT 0,
    config_json TEXT,
    created_at TIMESTAMP WITHOUT TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITHOUT TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_tunnels_user_id ON tunnels(user_id);
CREATE INDEX IF NOT EXISTS idx_tunnels_entry_group_id ON tunnels(entry_group_id);
CREATE INDEX IF NOT EXISTS idx_tunnels_exit_group_id ON tunnels(exit_group_id);
CREATE INDEX IF NOT EXISTS idx_tunnels_status ON tunnels(status);
CREATE INDEX IF NOT EXISTS idx_tunnels_protocol ON tunnels(protocol);

-- ========================================
-- 触发器函数：自动更新 node_group_stats
-- ========================================
CREATE OR REPLACE FUNCTION update_node_group_stats()
RETURNS TRIGGER AS $$
DECLARE
    old_group_id BIGINT;
    new_group_id BIGINT;
BEGIN
    IF TG_OP = 'DELETE' THEN
        old_group_id := OLD.node_group_id;
    ELSIF TG_OP = 'UPDATE' THEN
        old_group_id := OLD.node_group_id;
        new_group_id := NEW.node_group_id;
    ELSE
        new_group_id := NEW.node_group_id;
    END IF;

    IF old_group_id IS NOT NULL THEN
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
            old_group_id,
            COUNT(*)::INTEGER,
            COALESCE(SUM(CASE WHEN status = 'online' THEN 1 ELSE 0 END), 0)::INTEGER,
            0::BIGINT,
            0::BIGINT,
            0::INTEGER,
            CURRENT_TIMESTAMP
        FROM node_instances
        WHERE node_group_id = old_group_id
        ON CONFLICT (node_group_id)
        DO UPDATE SET
            total_nodes = EXCLUDED.total_nodes,
            online_nodes = EXCLUDED.online_nodes,
            total_traffic_in = EXCLUDED.total_traffic_in,
            total_traffic_out = EXCLUDED.total_traffic_out,
            total_connections = EXCLUDED.total_connections,
            updated_at = EXCLUDED.updated_at;
    END IF;

    IF new_group_id IS NOT NULL AND (old_group_id IS NULL OR new_group_id <> old_group_id) THEN
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
            new_group_id,
            COUNT(*)::INTEGER,
            COALESCE(SUM(CASE WHEN status = 'online' THEN 1 ELSE 0 END), 0)::INTEGER,
            0::BIGINT,
            0::BIGINT,
            0::INTEGER,
            CURRENT_TIMESTAMP
        FROM node_instances
        WHERE node_group_id = new_group_id
        ON CONFLICT (node_group_id)
        DO UPDATE SET
            total_nodes = EXCLUDED.total_nodes,
            online_nodes = EXCLUDED.online_nodes,
            total_traffic_in = EXCLUDED.total_traffic_in,
            total_traffic_out = EXCLUDED.total_traffic_out,
            total_connections = EXCLUDED.total_connections,
            updated_at = EXCLUDED.updated_at;
    END IF;

    IF TG_OP = 'DELETE' THEN
        RETURN OLD;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trigger_update_node_group_stats ON node_instances;
CREATE TRIGGER trigger_update_node_group_stats
AFTER INSERT OR UPDATE OR DELETE ON node_instances
FOR EACH ROW
EXECUTE FUNCTION update_node_group_stats();
