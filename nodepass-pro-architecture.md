# NodePass 流量转发管理系统 - 完整架构设计方案

## 1. 项目背景与目标

### 1.1 背景
基于开源项目 NodePass (https://github.com/NodePassProject/nodepass) 构建一个完整的企业级流量转发管理系统。NodePass 是一个用 Go 语言开发的 TCP/UDP 网络隧道解决方案，支持多种传输协议（TCP、QUIC、WebSocket、HTTP/2），具有智能连接池、分层 TLS 加密等特性。

### 1.2 项目目标
从零开始实现一个功能完整的流量转发管理平台，包括：
- **后端 API (backend/)**：Go 语言开发的管理中心，负责用户管理、节点管理、规则管理、流量统计等
- **前端管理界面 (frontend/public/)**：React + TypeScript 开发的 Web 管理面板
- **节点客户端 (nodeclient/)**：Go 语言开发的节点端，集成 NodePass，支持配置下发和离线运行

### 1.3 技术选型

**前后端分离架构**

**后端 (backend/)**
- **语言**: Go 1.21+
- **框架**: Gin (HTTP 框架)
- **ORM**: Gorm
- **数据库**: SQLite / MySQL / PostgreSQL（可选配置）
- **认证**: JWT + bcrypt
- **实时通信**: WebSocket (gorilla/websocket)
- **定时任务**: cron

**前端 (frontend/public/)**
- **语言**: TypeScript
- **构建工具**: Vite
- **框架**: React 18+
- **UI 库**: Ant Design 5.x
- **状态管理**: Zustand
- **路由**: React Router
- **HTTP 客户端**: Axios
- **图表**: ECharts

**节点端 (nodeclient/)**
- **语言**: Go
- **核心**: 集成 NodePass 开源项目
- **部署**: 一键安装脚本
- **特性**:
  - 配置联网下发（出口跳板 IP 等由面板下发）
  - 离线容错（与面板失联不影响现有规则运行）
  - 本地配置缓存

### 1.4 项目目录结构

```
NodePass-Pro/
├── backend/                    # Go 后端 API
│   ├── cmd/
│   │   └── server/
│   │       └── main.go        # 入口文件
│   ├── internal/
│   │   ├── config/            # 配置管理
│   │   ├── models/            # 数据模型
│   │   ├── handlers/          # HTTP 处理器
│   │   ├── services/          # 业务逻辑
│   │   ├── middleware/        # 中间件
│   │   ├── database/          # 数据库连接
│   │   ├── websocket/         # WebSocket 服务
│   │   └── utils/             # 工具函数
│   ├── migrations/            # 数据库迁移
│   ├── go.mod
│   └── go.sum
├── frontend/                   # React 前端
│   ├── public/                # 管理界面
│   ├── src/
│   │   ├── components/        # 组件
│   │   ├── pages/             # 页面
│   │   ├── services/          # API 服务
│   │   ├── hooks/             # 自定义 Hooks
│   │   ├── store/             # 状态管理
│   │   ├── types/             # 类型定义
│   │   └── utils/             # 工具函数
│   ├── package.json
│   └── vite.config.ts
├── nodeclient/                 # Go 节点客户端
│   ├── cmd/
│   │   └── client/
│   │       └── main.go
│   ├── internal/
│   │   ├── agent/             # Agent 核心
│   │   ├── config/            # 配置管理
│   │   ├── nodepass/          # NodePass 集成
│   │   ├── heartbeat/         # 心跳服务
│   │   └── cache/             # 本地配置缓存
│   ├── scripts/
│   │   └── install.sh         # 一键安装脚本
│   ├── go.mod
│   └── go.sum
├── docs/                       # 文档
└── README.md
```

---

## 2. 系统架构设计

### 2.1 整体架构

```
┌─────────────────────────────────────────────────────────────┐
│                      用户浏览器                              │
│              (React + TypeScript + Ant Design)              │
└────────────────────┬────────────────────────────────────────┘
                     │ HTTP/WebSocket
                     ↓
┌─────────────────────────────────────────────────────────────┐
│                  Backend API Server (Go)                     │
│  ┌──────────────┬──────────────┬──────────────────────┐    │
│  │ 用户管理     │ 节点管理     │ 规则管理             │    │
│  │ VIP体系      │ 流量统计     │ 权益码系统           │    │
│  │ Telegram集成 │ 审计日志     │ 配置下发             │    │
│  └──────────────┴──────────────┴──────────────────────┘    │
│                                                              │
│  数据库: SQLite/MySQL/PostgreSQL                            │
└────────────────────┬────────────────────────────────────────┘
                     │ REST API (配置下发、心跳上报)
                     ↓
┌─────────────────────────────────────────────────────────────┐
│                  NodeClient Agent Nodes (Go)                 │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐     │
│  │ Agent Node 1 │  │ Agent Node 2 │  │ Agent Node N │     │
│  │              │  │              │  │              │     │
│  │ - 配置缓存   │  │ - 配置缓存   │  │ - 配置缓存   │     │
│  │ - 心跳上报   │  │ - 心跳上报   │  │ - 心跳上报   │     │
│  │ - 离线运行   │  │ - 离线运行   │  │ - 离线运行   │     │
│  │ - NodePass   │  │ - NodePass   │  │ - NodePass   │     │
│  └──────────────┘  └──────────────┘  └──────────────┘     │
└─────────────────────────────────────────────────────────────┘
                     │ TCP/UDP/QUIC/WebSocket
                     ↓
              ┌──────────────┐
              │  目标服务器   │
              └──────────────┘
```

### 2.2 核心特性

**配置下发机制**
- 节点启动时向面板注册，获取初始配置
- 面板下发规则配置（包括出口跳板 IP、端口等）
- 节点本地缓存配置文件
- 配置变更时面板主动推送或节点定期拉取

**离线容错机制**
- 节点与面板失联时，继续使用本地缓存的配置运行
- 现有规则不受影响，保持转发服务
- 重新连接后同步最新配置
- 心跳超时不影响业务运行

**三层架构**

1. **表现层 (Presentation Layer)**
   - React 前端应用
   - Ant Design 组件库
   - 响应式设计

2. **业务逻辑层 (Business Logic Layer)**
   - Gin REST API
   - WebSocket 实时推送
   - 业务规则处理
   - 权限控制

3. **数据访问层 (Data Access Layer)**
   - Gorm ORM
   - 数据库（SQLite/MySQL/PostgreSQL）
   - 数据模型定义

### 2.3 模块划分

**核心模块**
1. 用户管理模块 (User Management)
2. 节点管理模块 (Node Management)
3. 规则管理模块 (Rule Management)
4. 流量管理模块 (Traffic Management)
5. VIP 体系模块 (VIP System)
6. 权益码模块 (Benefit Code)
7. Telegram 集成模块 (Telegram Integration)
8. 系统管理模块 (System Management)
9. 配置下发模块 (Config Distribution)

---

## 3. 数据库设计

### 3.1 数据库选择

支持三种数据库，通过配置文件切换：

**SQLite**（默认，适合小规模部署）
- 无需额外安装
- 单文件存储
- 适合 < 1000 用户

**MySQL**（推荐，适合中大规模）
- 成熟稳定
- 性能优秀
- 适合 1000+ 用户

**PostgreSQL**（高级特性）
- JSONB 支持
- 高级查询
- 适合复杂场景

### 3.2 核心表结构

#### 3.2.1 用户相关表

**users (用户表)**
```sql
CREATE TABLE users (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  username VARCHAR(50) UNIQUE NOT NULL,
  email VARCHAR(100) UNIQUE NOT NULL,
  password_hash VARCHAR(255) NOT NULL,
  role VARCHAR(20) NOT NULL DEFAULT 'user', -- admin, user
  status VARCHAR(20) NOT NULL DEFAULT 'normal', -- normal, paused, banned, overlimit

  -- VIP 相关
  vip_level INTEGER DEFAULT 0,
  vip_expires_at DATETIME,

  -- 配额相关
  traffic_quota BIGINT DEFAULT 0, -- 字节
  traffic_used BIGINT DEFAULT 0,
  max_rules INTEGER DEFAULT 5,
  max_bandwidth INTEGER DEFAULT 100, -- Mbps
  max_self_hosted_entry_nodes INTEGER DEFAULT 0,
  max_self_hosted_exit_nodes INTEGER DEFAULT 0,

  -- Telegram 相关
  telegram_id VARCHAR(50) UNIQUE,
  telegram_username VARCHAR(100),

  created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
  last_login_at DATETIME
);

CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_username ON users(username);
CREATE INDEX idx_users_telegram_id ON users(telegram_id);
CREATE INDEX idx_users_status ON users(status);
```

**user_permissions (用户权限表)**
```sql
CREATE TABLE user_permissions (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  user_id INTEGER NOT NULL,
  permission VARCHAR(100) NOT NULL, -- 如: nodes.create, rules.delete
  created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX idx_user_permissions_user_id ON user_permissions(user_id);
```

#### 3.2.2 节点相关表

**nodes (节点表)**
```sql
CREATE TABLE nodes (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  user_id INTEGER NOT NULL,
  name VARCHAR(100) NOT NULL,
  type VARCHAR(20) NOT NULL, -- entry, exit, both
  status VARCHAR(20) NOT NULL DEFAULT 'offline', -- online, offline, maintain

  -- 节点信息
  host VARCHAR(255) NOT NULL,
  port INTEGER NOT NULL,
  region VARCHAR(50),
  node_level INTEGER DEFAULT 1, -- 节点等级，用于访问控制

  -- 自托管标识
  is_self_hosted BOOLEAN DEFAULT FALSE,

  -- 流量倍率
  traffic_multiplier DECIMAL(5,2) DEFAULT 1.0,

  -- 系统信息
  cpu_usage DECIMAL(5,2),
  memory_usage DECIMAL(5,2),
  disk_usage DECIMAL(5,2),
  bandwidth_in BIGINT DEFAULT 0,
  bandwidth_out BIGINT DEFAULT 0,
  connections INTEGER DEFAULT 0,

  -- 认证
  token_hash VARCHAR(255) UNIQUE,

  -- 配置版本（用于配置下发）
  config_version INTEGER DEFAULT 0,

  last_heartbeat_at DATETIME,
  created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,

  FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX idx_nodes_user_id ON nodes(user_id);
CREATE INDEX idx_nodes_status ON nodes(status);
CREATE INDEX idx_nodes_token_hash ON nodes(token_hash);
```

**node_configs (节点配置表 - 用于配置下发)**
```sql
CREATE TABLE node_configs (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  node_id INTEGER NOT NULL,
  config_version INTEGER NOT NULL,
  config_data TEXT NOT NULL, -- JSON 格式的配置数据
  created_at DATETIME DEFAULT CURRENT_TIMESTAMP,

  FOREIGN KEY (node_id) REFERENCES nodes(id) ON DELETE CASCADE
);

CREATE INDEX idx_node_configs_node_id ON node_configs(node_id);
CREATE INDEX idx_node_configs_version ON node_configs(node_id, config_version);
```


#### 3.2.3 规则相关表

**rules (转发规则表)**
```sql
CREATE TABLE rules (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  user_id INTEGER NOT NULL,
  name VARCHAR(100) NOT NULL,

  -- 规则类型
  mode VARCHAR(20) NOT NULL, -- single, tunnel
  type VARCHAR(50) NOT NULL, -- port_forward, tunnel, multi_hop, load_balance, reverse_proxy
  protocol VARCHAR(20) NOT NULL, -- tcp, udp, ws, tls, quic

  -- 单节点模式
  node_id INTEGER,
  target_host VARCHAR(255),
  target_port INTEGER,

  -- 隧道模式
  entry_node_id INTEGER,
  exit_node_id INTEGER,

  -- 监听配置
  listen_host VARCHAR(255),
  listen_port INTEGER,

  -- 状态
  status VARCHAR(20) NOT NULL DEFAULT 'stopped', -- running, stopped, paused
  sync_status VARCHAR(20) DEFAULT 'pending', -- pending, synced, failed

  -- 实例信息
  instance_id VARCHAR(100), -- NodePass 实例 ID
  instance_status TEXT, -- JSON 格式的实例状态

  -- 流量统计
  traffic_in BIGINT DEFAULT 0,
  traffic_out BIGINT DEFAULT 0,
  connections INTEGER DEFAULT 0,

  -- 配置
  config_json TEXT, -- JSON 格式的额外配置（TLS、超时、最大连接数等）

  -- 配置版本
  config_version INTEGER DEFAULT 0,

  created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,

  FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
  FOREIGN KEY (node_id) REFERENCES nodes(id) ON DELETE SET NULL,
  FOREIGN KEY (entry_node_id) REFERENCES nodes(id) ON DELETE SET NULL,
  FOREIGN KEY (exit_node_id) REFERENCES nodes(id) ON DELETE SET NULL
);

CREATE INDEX idx_rules_user_id ON rules(user_id);
CREATE INDEX idx_rules_node_id ON rules(node_id);
CREATE INDEX idx_rules_entry_node_id ON rules(entry_node_id);
CREATE INDEX idx_rules_exit_node_id ON rules(exit_node_id);
CREATE INDEX idx_rules_status ON rules(status);
```

#### 3.2.4 流量管理表

**traffic_records (流量记录表)**
```sql
CREATE TABLE traffic_records (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  user_id INTEGER NOT NULL,
  rule_id INTEGER,
  node_id INTEGER,

  -- 流量数据
  traffic_in BIGINT NOT NULL,
  traffic_out BIGINT NOT NULL,

  -- 倍率计算
  vip_multiplier DECIMAL(5,2) DEFAULT 1.0,
  node_multiplier DECIMAL(5,2) DEFAULT 1.0,
  final_multiplier DECIMAL(5,2) DEFAULT 1.0,
  calculated_traffic BIGINT, -- 应用倍率后的流量

  -- 时间
  hour DATETIME NOT NULL, -- 精确到小时
  created_at DATETIME DEFAULT CURRENT_TIMESTAMP,

  FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
  FOREIGN KEY (rule_id) REFERENCES rules(id) ON DELETE CASCADE,
  FOREIGN KEY (node_id) REFERENCES nodes(id) ON DELETE SET NULL
);

CREATE INDEX idx_traffic_records_user_id ON traffic_records(user_id);
CREATE INDEX idx_traffic_records_rule_id ON traffic_records(rule_id);
CREATE INDEX idx_traffic_records_hour ON traffic_records(hour);
CREATE INDEX idx_traffic_records_user_hour ON traffic_records(user_id, hour);
```

#### 3.2.5 VIP 体系表

**vip_levels (VIP 等级配置表)**
```sql
CREATE TABLE vip_levels (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  level INTEGER UNIQUE NOT NULL,
  name VARCHAR(50) NOT NULL,
  description TEXT,

  -- 权益配置
  traffic_quota BIGINT NOT NULL, -- 字节
  max_rules INTEGER NOT NULL,
  max_bandwidth INTEGER NOT NULL, -- Mbps
  max_self_hosted_entry_nodes INTEGER DEFAULT 0,
  max_self_hosted_exit_nodes INTEGER DEFAULT 0,
  accessible_node_level INTEGER DEFAULT 1, -- 可访问的节点等级
  traffic_multiplier DECIMAL(5,2) DEFAULT 1.0, -- 流量计算倍率

  -- 自定义功能
  custom_features TEXT, -- JSON 格式的自定义功能配置

  -- 价格（可选）
  price DECIMAL(10,2),
  duration_days INTEGER, -- 有效期天数

  created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_vip_levels_level ON vip_levels(level);
```

#### 3.2.6 权益码表

**benefit_codes (权益码表)**
```sql
CREATE TABLE benefit_codes (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  code VARCHAR(50) UNIQUE NOT NULL,
  vip_level INTEGER NOT NULL,
  duration_days INTEGER NOT NULL, -- 有效期天数

  -- 状态
  status VARCHAR(20) NOT NULL DEFAULT 'unused', -- unused, used
  is_enabled BOOLEAN DEFAULT TRUE,

  -- 使用信息
  used_by INTEGER,
  used_at DATETIME,

  -- 过期时间
  expires_at DATETIME,

  created_at DATETIME DEFAULT CURRENT_TIMESTAMP,

  FOREIGN KEY (used_by) REFERENCES users(id) ON DELETE SET NULL
);

CREATE INDEX idx_benefit_codes_code ON benefit_codes(code);
CREATE INDEX idx_benefit_codes_status ON benefit_codes(status);
```

#### 3.2.7 系统管理表

**system_config (系统配置表)**
```sql
CREATE TABLE system_config (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  key VARCHAR(100) UNIQUE NOT NULL,
  value TEXT,
  description TEXT,
  updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

**announcements (公告表)**
```sql
CREATE TABLE announcements (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  title VARCHAR(200) NOT NULL,
  content TEXT NOT NULL,
  type VARCHAR(20) NOT NULL, -- info, warning, error, success
  is_enabled BOOLEAN DEFAULT TRUE,
  start_time DATETIME,
  end_time DATETIME,
  created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

**audit_logs (审计日志表)**
```sql
CREATE TABLE audit_logs (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  user_id INTEGER,
  action VARCHAR(100) NOT NULL, -- 如: user.login, node.create, rule.delete
  resource_type VARCHAR(50), -- user, node, rule, etc.
  resource_id INTEGER,
  details TEXT, -- JSON 格式的详细信息
  ip_address VARCHAR(50),
  user_agent TEXT,
  created_at DATETIME DEFAULT CURRENT_TIMESTAMP,

  FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE SET NULL
);

CREATE INDEX idx_audit_logs_user_id ON audit_logs(user_id);
CREATE INDEX idx_audit_logs_action ON audit_logs(action);
CREATE INDEX idx_audit_logs_created_at ON audit_logs(created_at);
```

---

## 4. API 设计

### 4.1 API 规范

**基础 URL**: `http://localhost:8080/api/v1`

**认证方式**: JWT Bearer Token
```
Authorization: Bearer <token>
```

**响应格式**:
```json
{
  "success": true,
  "data": {},
  "message": "操作成功",
  "timestamp": "2026-03-06T12:00:00Z"
}
```

**错误响应**:
```json
{
  "success": false,
  "error": {
    "code": "UNAUTHORIZED",
    "message": "未授权访问"
  },
  "timestamp": "2026-03-06T12:00:00Z"
}
```

### 4.2 认证模块 API

**POST /auth/register** - 用户注册
**POST /auth/login** - 用户登录
**POST /auth/logout** - 用户登出
**GET /auth/me** - 获取当前用户信息
**POST /auth/refresh** - 刷新 Token

### 4.3 节点管理 API

**POST /nodes** - 创建节点
**GET /nodes** - 获取节点列表
**GET /nodes/:id** - 获取节点详情
**PUT /nodes/:id** - 更新节点信息
**DELETE /nodes/:id** - 删除节点
**POST /nodes/:id/test** - TCP 连通性测试

**核心：配置下发 API**

**POST /nodes/register** - 节点注册（nodeclient 调用）
```json
Request:
{
  "token": "node_token_xxx",
  "hostname": "node1.example.com",
  "version": "1.0.0"
}

Response:
{
  "success": true,
  "data": {
    "node_id": 1,
    "config_version": 1,
    "config": {
      "rules": [...],
      "exit_nodes": [...],
      "settings": {...}
    }
  }
}
```

**POST /nodes/heartbeat** - 节点心跳（nodeclient 定期调用）
```json
Request:
{
  "token": "node_token_xxx",
  "config_version": 1,
  "system_info": {
    "cpu_usage": 45.2,
    "memory_usage": 60.5,
    "disk_usage": 30.0,
    "bandwidth_in": 1024000,
    "bandwidth_out": 2048000,
    "connections": 10
  },
  "rules_status": [
    {"rule_id": 1, "status": "running", "connections": 5},
    {"rule_id": 2, "status": "running", "connections": 3}
  ]
}

Response:
{
  "success": true,
  "data": {
    "config_updated": true,  // 配置是否有更新
    "new_config_version": 2,
    "config": {
      // 如果有更新，返回新配置
    }
  }
}
```

**GET /nodes/:id/config** - 获取节点配置（nodeclient 拉取配置）
```json
Response:
{
  "success": true,
  "data": {
    "config_version": 2,
    "config": {
      "rules": [
        {
          "rule_id": 1,
          "mode": "tunnel",
          "entry_node": {
            "host": "entry.example.com",
            "port": 8080
          },
          "exit_node": {
            "host": "exit.example.com",  // 出口跳板 IP 由面板下发
            "port": 8081
          },
          "target": {
            "host": "192.168.1.100",
            "port": 3306
          },
          "listen": {
            "host": "0.0.0.0",
            "port": 13306
          },
          "protocol": "tcp"
        }
      ],
      "settings": {
        "heartbeat_interval": 30,
        "config_check_interval": 60
      }
    }
  }
}
```

**POST /nodes/traffic/report** - 流量上报（nodeclient 调用）
```json
Request:
{
  "token": "node_token_xxx",
  "records": [
    {
      "rule_id": 1,
      "traffic_in": 1024000,
      "traffic_out": 2048000,
      "timestamp": "2026-03-06T12:00:00Z"
    }
  ]
}
```

### 4.4 规则管理 API

**POST /rules** - 创建规则
**GET /rules** - 获取规则列表
**GET /rules/:id** - 获取规则详情
**PUT /rules/:id** - 更新规则
**DELETE /rules/:id** - 删除规则
**POST /rules/:id/start** - 启动规则实例
**POST /rules/:id/stop** - 停止规则实例
**POST /rules/:id/restart** - 重启规则实例

### 4.5 流量管理 API

**GET /traffic/quota** - 获取当前用户流量配额
**GET /traffic/usage** - 获取流量使用情况
**GET /traffic/records** - 获取流量记录
**POST /traffic/quota/reset** - 重置流量配额（管理员）

### 4.6 VIP 体系 API

**GET /vip/levels** - 获取 VIP 等级列表
**POST /vip/levels** - 创建 VIP 等级（管理员）
**PUT /vip/levels/:id** - 更新 VIP 等级（管理员）
**GET /vip/my-level** - 获取当前用户 VIP 等级

### 4.7 权益码 API

**POST /benefit-codes/generate** - 批量生成权益码（管理员）
**GET /benefit-codes** - 获取权益码列表（管理员）
**POST /benefit-codes/redeem** - 兑换权益码

### 4.8 Telegram 集成 API

**POST /telegram/bind** - 绑定 Telegram 账户
**POST /telegram/unbind** - 解绑 Telegram 账户
**POST /telegram/login** - Telegram Widget 登录

### 4.9 系统管理 API

**GET /system/config** - 获取系统配置
**PUT /system/config** - 更新系统配置（管理员）
**GET /announcements** - 获取公告列表
**POST /announcements** - 创建公告（管理员）
**GET /audit-logs** - 获取审计日志（管理员）

---

## 5. 后端实现方案 (Go)

### 5.1 项目结构

```
backend/
├── cmd/
│   └── server/
│       └── main.go                 # 入口文件
├── internal/
│   ├── config/
│   │   └── config.go              # 配置管理
│   ├── models/
│   │   ├── user.go
│   │   ├── node.go
│   │   ├── rule.go
│   │   ├── traffic.go
│   │   ├── vip.go
│   │   └── benefit_code.go
│   ├── handlers/
│   │   ├── auth_handler.go
│   │   ├── user_handler.go
│   │   ├── node_handler.go
│   │   ├── rule_handler.go
│   │   ├── traffic_handler.go
│   │   ├── vip_handler.go
│   │   └── system_handler.go
│   ├── services/
│   │   ├── auth_service.go
│   │   ├── user_service.go
│   │   ├── node_service.go
│   │   ├── rule_service.go
│   │   ├── traffic_service.go
│   │   ├── vip_service.go
│   │   ├── config_distribution.go  # 配置下发服务
│   │   └── telegram_service.go
│   ├── middleware/
│   │   ├── auth.go
│   │   ├── permission.go
│   │   ├── rate_limit.go
│   │   └── logger.go
│   ├── database/
│   │   └── db.go                  # 数据库连接
│   ├── websocket/
│   │   └── hub.go                 # WebSocket 服务
│   └── utils/
│       ├── jwt.go
│       ├── bcrypt.go
│       ├── validator.go
│       └── response.go
├── migrations/
│   └── 001_initial_schema.sql
├── go.mod
└── go.sum
```

### 5.2 核心代码示例

#### 5.2.1 配置管理

```go
// internal/config/config.go
package config

import (
    "github.com/spf13/viper"
)

type Config struct {
    Server   ServerConfig
    Database DatabaseConfig
    JWT      JWTConfig
    Telegram TelegramConfig
}

type ServerConfig struct {
    Port string
    Mode string // debug, release
}

type DatabaseConfig struct {
    Type     string // sqlite, mysql, postgres
    Host     string
    Port     int
    User     string
    Password string
    DBName   string
    DSN      string
}

type JWTConfig struct {
    Secret     string
    ExpireTime int // 小时
}

type TelegramConfig struct {
    BotToken string
    BotUsername string
}

func LoadConfig() (*Config, error) {
    viper.SetConfigName("config")
    viper.SetConfigType("yaml")
    viper.AddConfigPath("./configs")
    
    if err := viper.ReadInConfig(); err != nil {
        return nil, err
    }
    
    var config Config
    if err := viper.Unmarshal(&config); err != nil {
        return nil, err
    }
    
    return &config, nil
}
```

#### 5.2.2 数据库连接

```go
// internal/database/db.go
package database

import (
    "fmt"
    "gorm.io/driver/sqlite"
    "gorm.io/driver/mysql"
    "gorm.io/driver/postgres"
    "gorm.io/gorm"
)

func InitDB(config *config.DatabaseConfig) (*gorm.DB, error) {
    var dialector gorm.Dialector
    
    switch config.Type {
    case "sqlite":
        dialector = sqlite.Open(config.DSN)
    case "mysql":
        dialector = mysql.Open(config.DSN)
    case "postgres":
        dialector = postgres.Open(config.DSN)
    default:
        return nil, fmt.Errorf("unsupported database type: %s", config.Type)
    }
    
    db, err := gorm.Open(dialector, &gorm.Config{})
    if err != nil {
        return nil, err
    }
    
    // 自动迁移
    if err := autoMigrate(db); err != nil {
        return nil, err
    }
    
    return db, nil
}

func autoMigrate(db *gorm.DB) error {
    return db.AutoMigrate(
        &models.User{},
        &models.Node{},
        &models.Rule{},
        &models.TrafficRecord{},
        &models.VipLevel{},
        &models.BenefitCode{},
        // ... 其他模型
    )
}
```


#### 5.2.3 配置下发服务（核心）

```go
// internal/services/config_distribution.go
package services

import (
    "encoding/json"
    "time"
)

type ConfigDistributionService struct {
    db *gorm.DB
}

// NodeConfig 节点配置结构
type NodeConfig struct {
    ConfigVersion int         `json:"config_version"`
    Rules         []RuleConfig `json:"rules"`
    Settings      Settings     `json:"settings"`
}

type RuleConfig struct {
    RuleID    int      `json:"rule_id"`
    Mode      string   `json:"mode"`
    EntryNode *NodeInfo `json:"entry_node,omitempty"`
    ExitNode  *NodeInfo `json:"exit_node,omitempty"` // 出口跳板 IP 由面板下发
    Target    Target   `json:"target"`
    Listen    Listen   `json:"listen"`
    Protocol  string   `json:"protocol"`
}

// GenerateNodeConfig 生成节点配置
func (s *ConfigDistributionService) GenerateNodeConfig(nodeID int) (*NodeConfig, error) {
    var node models.Node
    if err := s.db.First(&node, nodeID).Error; err != nil {
        return nil, err
    }

    // 获取该节点的所有规则
    var rules []models.Rule
    if err := s.db.Where("node_id = ? OR entry_node_id = ?", nodeID, nodeID).
        Preload("EntryNode").
        Preload("ExitNode").
        Find(&rules).Error; err != nil {
        return nil, err
    }

    // 构建配置（包含出口跳板 IP 等信息）
    config := &NodeConfig{
        ConfigVersion: node.ConfigVersion,
        Rules:         make([]RuleConfig, 0),
    }

    for _, rule := range rules {
        ruleConfig := RuleConfig{
            RuleID:   rule.ID,
            Mode:     rule.Mode,
            Protocol: rule.Protocol,
        }

        // 隧道模式：下发出口节点信息
        if rule.Mode == "tunnel" && rule.ExitNode != nil {
            ruleConfig.ExitNode = &NodeInfo{
                Host: rule.ExitNode.Host, // 出口跳板 IP 由面板下发
                Port: rule.ExitNode.Port,
            }
        }

        config.Rules = append(config.Rules, ruleConfig)
    }

    return config, nil
}

// HandleHeartbeat 处理节点心跳
func (s *ConfigDistributionService) HandleHeartbeat(
    nodeToken string,
    currentVersion int,
) (bool, *NodeConfig, error) {
    // 验证节点并更新状态
    var node models.Node
    if err := s.db.Where("token_hash = ?", hashToken(nodeToken)).First(&node).Error; err != nil {
        return false, nil, err
    }

    // 更新心跳时间
    s.db.Model(&node).Update("last_heartbeat_at", time.Now())

    // 检查配置是否有更新
    if currentVersion < node.ConfigVersion {
        config, err := s.GenerateNodeConfig(node.ID)
        if err != nil {
            return false, nil, err
        }
        return true, config, nil
    }

    return false, nil, nil
}
```

---

## 6. 节点客户端实现方案 (nodeclient)

### 6.1 核心特性

**配置下发**
- 节点启动时从面板获取配置
- 配置包含出口跳板 IP、端口等信息
- 定期检查配置更新

**离线容错**
- 本地缓存配置文件
- 与面板失联时使用缓存配置
- 现有规则不受影响继续运行
- 重新连接后同步最新配置

### 6.2 核心代码示例

#### 6.2.1 配置缓存

```go
// internal/config/cache.go
package config

import (
    "encoding/json"
    "os"
)

type ConfigCache struct {
    cachePath string
}

// Save 保存配置到本地缓存
func (c *ConfigCache) Save(config *NodeConfig) error {
    data, err := json.MarshalIndent(config, "", "  ")
    if err != nil {
        return err
    }
    return os.WriteFile(c.cachePath, data, 0644)
}

// Load 从本地缓存加载配置
func (c *ConfigCache) Load() (*NodeConfig, error) {
    data, err := os.ReadFile(c.cachePath)
    if err != nil {
        return nil, err
    }

    var config NodeConfig
    if err := json.Unmarshal(data, &config); err != nil {
        return nil, err
    }

    return &config, nil
}
```

#### 6.2.2 Agent 核心（离线容错）

```go
// internal/agent/agent.go
package agent

import (
    "log"
    "time"
)

type Agent struct {
    apiClient   *api.Client
    configCache *config.ConfigCache
    nodePass    *nodepass.Integration
    isOnline    bool
}

// Start 启动 Agent
func (a *Agent) Start() error {
    log.Println("Starting NodePass Agent...")

    // 1. 尝试从面板获取配置
    config, err := a.fetchConfigFromHub()
    if err != nil {
        log.Printf("Failed to fetch config from hub: %v", err)
        
        // 2. 如果失败，尝试加载本地缓存
        if a.configCache.Exists() {
            log.Println("Loading config from local cache...")
            config, err = a.configCache.Load()
            if err != nil {
                return fmt.Errorf("failed to load cached config: %v", err)
            }
            a.isOnline = false
            log.Println("Running in OFFLINE mode with cached config")
        } else {
            return fmt.Errorf("no cached config available")
        }
    } else {
        a.isOnline = true
        log.Println("Running in ONLINE mode")
        
        // 3. 保存配置到本地缓存
        a.configCache.Save(config)
    }

    // 4. 应用配置，启动规则
    a.applyConfig(config)

    // 5. 启动心跳服务
    a.startHeartbeat()

    return nil
}

// sendHeartbeat 发送心跳
func (a *Agent) sendHeartbeat() {
    resp, err := a.apiClient.Heartbeat(...)

    if err != nil {
        log.Printf("Heartbeat failed: %v", err)
        a.isOnline = false
        // 心跳失败不影响现有规则运行
        return
    }

    a.isOnline = true

    // 检查配置更新
    if resp.ConfigUpdated {
        log.Printf("Config updated to version %d", resp.NewConfigVersion)
        
        // 保存新配置到缓存
        a.configCache.Save(resp.Config)

        // 应用新配置
        a.applyConfig(resp.Config)
    }
}
```

---

## 7. 前端实现方案

### 7.1 技术栈
- React 18 + TypeScript
- Vite (构建工具)
- Ant Design 5.x (UI 组件库)
- Zustand (状态管理)
- React Router (路由)
- Axios (HTTP 客户端)
- ECharts (图表)

### 7.2 核心页面
1. 仪表盘 - 统计卡片、流量趋势图、节点状态
2. 节点管理 - 节点列表、创建/编辑、实时状态
3. 规则管理 - 规则列表、创建/编辑、启动/停止
4. 流量统计 - 配额显示、使用趋势、按规则统计
5. VIP 中心 - 等级列表、当前 VIP、升级选项
6. 权益码管理 - 生成、列表、兑换
7. 系统管理 - 配置、公告、审计日志

---

## 8. 部署方案

### 8.1 开发环境

**后端**
```bash
git clone https://github.com/nodeox/NodePass-Pro.git
cd NodePass-Pro/backend
go mod init nodepass-pro/backend
go mod tidy
go run cmd/server/main.go
```

**前端**
```bash
cd ../frontend
npm install
npm run dev
```

**节点客户端**
```bash
cd ../nodeclient
go mod init nodepass-panel/nodeclient
go mod tidy
go run cmd/client/main.go
```

---

## 9. 总结

本架构设计提供了一个完整的基于 Go 的 NodePass 流量转发管理系统实现方案：

**核心特性**:
- Go 后端 + React 前端的前后端分离架构
- 支持 SQLite/MySQL/PostgreSQL 三种数据库
- 配置联网下发机制（出口跳板 IP 等由面板下发）
- 离线容错机制（节点失联不影响现有规则运行）
- 完整的用户管理、VIP 体系、权益码系统
- Telegram 集成
- 流量统计和配额管理

**项目仓库**: [https://github.com/nodeox/NodePass-Pro](https://github.com/nodeox/NodePass-Pro)

此方案为**架构设计文档**，提供了详细的数据库设计、API 设计和核心代码示例，开发者需要根据此设计自行完成完整实现。

---

## 10. 节点类型与自托管功能详解

### 10.1 节点类型

系统支持三种节点类型：

#### 10.1.1 入口节点 (Entry Node)
- **作用**: 接收用户连接请求
- **功能**: 
  - 监听用户的连接
  - 接收流量并转发到出口节点或目标服务器
  - 通常部署在用户网络附近
- **使用场景**: 
  - 单节点模式：直接转发到目标
  - 隧道模式：作为隧道的入口端

#### 10.1.2 出口节点 (Exit Node)
- **作用**: 连接目标服务器
- **功能**:
  - 接收入口节点转发的流量
  - 连接到最终目标服务器
  - 通常部署在目标服务器网络附近
- **使用场景**:
  - 隧道模式：作为隧道的出口端
  - 跨网络访问的跳板

#### 10.1.3 双功能节点 (Both)
- **作用**: 同时支持入口和出口功能
- **功能**: 可以作为入口节点或出口节点使用
- **使用场景**: 灵活部署，节省资源

### 10.2 节点自托管功能

#### 10.2.1 什么是自托管节点

**自托管节点** 是指用户自己部署和管理的节点，而非使用平台提供的公共节点。

**优势**:
- ✅ 完全控制节点资源
- ✅ 更好的隐私保护
- ✅ 自定义网络配置
- ✅ 无需与他人共享带宽

**限制**:
- 需要用户自己的服务器
- 需要一定的技术能力
- 受 VIP 等级限制数量

#### 10.2.2 自托管节点配额

不同 VIP 等级的自托管节点配额：

| VIP 等级 | 自托管入口节点 | 自托管出口节点 | 说明 |
|---------|--------------|--------------|------|
| 免费用户 | 0 | 0 | 只能使用公共节点 |
| VIP 1 | 1 | 1 | 可部署 1 个入口 + 1 个出口 |
| VIP 2 | 3 | 3 | 可部署 3 个入口 + 3 个出口 |
| VIP 3 | 5 | 5 | 可部署 5 个入口 + 5 个出口 |
| VIP 4+ | 10 | 10 | 可部署 10 个入口 + 10 个出口 |

#### 10.2.3 自托管节点部署流程

**1. 在面板创建节点**
```
用户操作：
1. 登录面板
2. 进入"节点管理"
3. 点击"创建节点"
4. 选择节点类型：入口/出口/双功能
5. 勾选"自托管节点"
6. 填写节点信息（名称、区域等）
7. 提交创建

系统响应：
- 生成节点 Token
- 显示安装命令
```

**2. 在服务器上部署节点客户端**
```bash
# 一键安装脚本
curl -fsSL https://your-panel.com/install.sh | bash -s -- \
  --hub-url "https://your-panel.com" \
  --token "node_token_xxx" \
  --type "entry"  # 或 exit, both
```

**3. 节点自动注册并上线**
```
节点客户端启动后：
1. 连接到面板
2. 使用 Token 认证
3. 获取配置
4. 上报状态
5. 开始接收规则
```

### 10.3 数据库设计更新

#### 10.3.1 nodes 表增强

```sql
CREATE TABLE nodes (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  user_id INTEGER NOT NULL,
  name VARCHAR(100) NOT NULL,
  
  -- 节点类型：entry(入口), exit(出口), both(双功能)
  type VARCHAR(20) NOT NULL,
  
  -- 节点状态
  status VARCHAR(20) NOT NULL DEFAULT 'offline', -- online, offline, maintain
  
  -- 节点信息
  host VARCHAR(255) NOT NULL,
  port INTEGER NOT NULL,
  region VARCHAR(50),
  node_level INTEGER DEFAULT 1,
  
  -- 自托管标识（重要）
  is_self_hosted BOOLEAN DEFAULT FALSE,
  
  -- 公共节点标识
  is_public BOOLEAN DEFAULT FALSE,
  
  -- 流量倍率
  traffic_multiplier DECIMAL(5,2) DEFAULT 1.0,
  
  -- 系统信息
  cpu_usage DECIMAL(5,2),
  memory_usage DECIMAL(5,2),
  disk_usage DECIMAL(5,2),
  bandwidth_in BIGINT DEFAULT 0,
  bandwidth_out BIGINT DEFAULT 0,
  connections INTEGER DEFAULT 0,
  
  -- 认证
  token_hash VARCHAR(255) UNIQUE,
  
  -- 配置版本
  config_version INTEGER DEFAULT 0,
  
  -- 节点描述
  description TEXT,
  
  last_heartbeat_at DATETIME,
  created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
  
  FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX idx_nodes_user_id ON nodes(user_id);
CREATE INDEX idx_nodes_type ON nodes(type);
CREATE INDEX idx_nodes_status ON nodes(status);
CREATE INDEX idx_nodes_is_self_hosted ON nodes(is_self_hosted);
CREATE INDEX idx_nodes_is_public ON nodes(is_public);
```

### 10.4 API 设计更新

#### 10.4.1 创建节点 API

**POST /api/v1/nodes**

```json
Request:
{
  "name": "我的入口节点",
  "type": "entry",           // entry, exit, both
  "is_self_hosted": true,    // 是否自托管
  "host": "node1.example.com",
  "port": 8080,
  "region": "us-west",
  "description": "部署在 AWS 的入口节点"
}

Response:
{
  "success": true,
  "data": {
    "node": {
      "id": 1,
      "name": "我的入口节点",
      "type": "entry",
      "is_self_hosted": true,
      "status": "offline",
      "token": "node_abc123xyz..."  // 用于节点认证
    },
    "install_command": "curl -fsSL https://panel.com/install.sh | bash -s -- --hub-url https://panel.com --token node_abc123xyz --type entry"
  }
}
```

#### 10.4.2 获取可用节点列表 API

**GET /api/v1/nodes/available**

用于创建规则时选择节点。

```json
Request Query:
?type=entry  // 筛选节点类型：entry, exit, both
&user_only=false  // 是否只显示用户自己的节点

Response:
{
  "success": true,
  "data": {
    "entry_nodes": [
      {
        "id": 1,
        "name": "我的入口节点",
        "type": "entry",
        "is_self_hosted": true,
        "is_public": false,
        "region": "us-west",
        "status": "online",
        "owner": "me"
      },
      {
        "id": 10,
        "name": "公共入口节点-美西",
        "type": "entry",
        "is_self_hosted": false,
        "is_public": true,
        "region": "us-west",
        "status": "online",
        "owner": "platform"
      }
    ],
    "exit_nodes": [
      {
        "id": 2,
        "name": "我的出口节点",
        "type": "exit",
        "is_self_hosted": true,
        "is_public": false,
        "region": "asia-east",
        "status": "online",
        "owner": "me"
      }
    ]
  }
}
```

#### 10.4.3 检查自托管配额 API

**GET /api/v1/nodes/quota**

```json
Response:
{
  "success": true,
  "data": {
    "vip_level": 2,
    "max_self_hosted_entry_nodes": 3,
    "max_self_hosted_exit_nodes": 3,
    "current_entry_nodes": 1,
    "current_exit_nodes": 1,
    "remaining_entry_nodes": 2,
    "remaining_exit_nodes": 2
  }
}
```

### 10.5 业务逻辑实现

#### 10.5.1 创建节点时的配额检查

```go
// internal/services/node_service.go
package services

func (s *NodeService) CreateNode(userID int, req *CreateNodeRequest) (*Node, error) {
    // 1. 获取用户信息
    user, err := s.userRepo.FindByID(userID)
    if err != nil {
        return nil, err
    }

    // 2. 如果是自托管节点，检查配额
    if req.IsSelfHosted {
        // 统计用户当前自托管节点数量
        var count int64
        
        if req.Type == "entry" || req.Type == "both" {
            s.db.Model(&Node{}).
                Where("user_id = ? AND is_self_hosted = ? AND (type = ? OR type = ?)", 
                    userID, true, "entry", "both").
                Count(&count)
            
            if count >= int64(user.MaxSelfHostedEntryNodes) {
                return nil, fmt.Errorf("已达到自托管入口节点配额上限 (%d)", 
                    user.MaxSelfHostedEntryNodes)
            }
        }
        
        if req.Type == "exit" || req.Type == "both" {
            s.db.Model(&Node{}).
                Where("user_id = ? AND is_self_hosted = ? AND (type = ? OR type = ?)", 
                    userID, true, "exit", "both").
                Count(&count)
            
            if count >= int64(user.MaxSelfHostedExitNodes) {
                return nil, fmt.Errorf("已达到自托管出口节点配额上限 (%d)", 
                    user.MaxSelfHostedExitNodes)
            }
        }
    }

    // 3. 生成节点 Token
    token := generateNodeToken()
    tokenHash := hashToken(token)

    // 4. 创建节点
    node := &Node{
        UserID:       userID,
        Name:         req.Name,
        Type:         req.Type,
        IsSelfHosted: req.IsSelfHosted,
        IsPublic:     false, // 自托管节点默认私有
        Host:         req.Host,
        Port:         req.Port,
        Region:       req.Region,
        Description:  req.Description,
        TokenHash:    tokenHash,
        Status:       "offline",
    }

    if err := s.db.Create(node).Error; err != nil {
        return nil, err
    }

    // 5. 返回节点信息和 Token（仅此一次）
    node.Token = token
    return node, nil
}
```

#### 10.5.2 获取可用节点列表

```go
// internal/services/node_service.go

func (s *NodeService) GetAvailableNodes(userID int, nodeType string) (*AvailableNodes, error) {
    var entryNodes, exitNodes []Node

    // 查询条件
    query := s.db.Where("status = ?", "online")

    // 1. 获取用户自己的节点
    userNodesQuery := query.Where("user_id = ? AND is_self_hosted = ?", userID, true)
    
    if nodeType == "entry" || nodeType == "" {
        userNodesQuery.Where("type IN (?)", []string{"entry", "both"}).Find(&entryNodes)
    }
    
    if nodeType == "exit" || nodeType == "" {
        userNodesQuery.Where("type IN (?)", []string{"exit", "both"}).Find(&exitNodes)
    }

    // 2. 获取公共节点
    publicNodesQuery := query.Where("is_public = ?", true)
    
    var publicEntryNodes, publicExitNodes []Node
    
    if nodeType == "entry" || nodeType == "" {
        publicNodesQuery.Where("type IN (?)", []string{"entry", "both"}).Find(&publicEntryNodes)
        entryNodes = append(entryNodes, publicEntryNodes...)
    }
    
    if nodeType == "exit" || nodeType == "" {
        publicNodesQuery.Where("type IN (?)", []string{"exit", "both"}).Find(&publicExitNodes)
        exitNodes = append(exitNodes, publicExitNodes...)
    }

    return &AvailableNodes{
        EntryNodes: entryNodes,
        ExitNodes:  exitNodes,
    }, nil
}
```

### 10.6 规则创建时的节点选择

#### 10.6.1 单节点模式

```json
POST /api/v1/rules

Request:
{
  "name": "MySQL 转发",
  "mode": "single",
  "type": "port_forward",
  "protocol": "tcp",
  "node_id": 1,              // 选择一个节点（入口或双功能）
  "target_host": "192.168.1.100",
  "target_port": 3306,
  "listen_host": "0.0.0.0",
  "listen_port": 13306
}
```

**流程**:
```
用户 → 入口节点 → 目标服务器
```

#### 10.6.2 隧道模式

```json
POST /api/v1/rules

Request:
{
  "name": "跨网络 MySQL 访问",
  "mode": "tunnel",
  "type": "tunnel",
  "protocol": "tcp",
  "entry_node_id": 1,        // 入口节点（自托管）
  "exit_node_id": 2,         // 出口节点（自托管或公共）
  "target_host": "192.168.1.100",
  "target_port": 3306,
  "listen_host": "0.0.0.0",
  "listen_port": 13306
}
```

**流程**:
```
用户 → 入口节点 → 出口节点 → 目标服务器
```

### 10.7 前端界面设计

#### 10.7.1 节点管理页面

**节点列表**
- 显示节点类型标签（入口/出口/双功能）
- 显示自托管标识
- 显示节点状态（在线/离线）
- 显示系统信息（CPU、内存等）

**创建节点表单**
```
┌─────────────────────────────────────┐
│ 创建节点                             │
├─────────────────────────────────────┤
│ 节点名称: [___________________]     │
│                                     │
│ 节点类型: ○ 入口节点               │
│          ○ 出口节点               │
│          ● 双功能节点             │
│                                     │
│ ☑ 自托管节点                       │
│   (当前配额: 入口 1/3, 出口 1/3)   │
│                                     │
│ 主机地址: [___________________]     │
│ 端口:     [____]                   │
│ 区域:     [下拉选择]               │
│ 描述:     [___________________]     │
│                                     │
│ [取消]  [创建节点]                 │
└─────────────────────────────────────┘
```

#### 10.7.2 规则创建页面

**节点选择**
```
┌─────────────────────────────────────┐
│ 创建转发规则                         │
├─────────────────────────────────────┤
│ 转发模式: ○ 单节点模式             │
│          ● 隧道模式               │
│                                     │
│ 入口节点: [下拉选择]               │
│   ├─ 我的入口节点 (自托管) ✓       │
│   ├─ 公共入口-美西                 │
│   └─ 公共入口-欧洲                 │
│                                     │
│ 出口节点: [下拉选择]               │
│   ├─ 我的出口节点 (自托管) ✓       │
│   ├─ 公共出口-亚洲                 │
│   └─ 公共出口-美东                 │
│                                     │
│ 目标地址: [___________________]     │
│ 目标端口: [____]                   │
│                                     │
│ [取消]  [创建规则]                 │
└─────────────────────────────────────┘
```

### 10.8 节点客户端更新

#### 10.8.1 启动参数

```bash
nodeclient start \
  --hub-url "https://panel.com" \
  --token "node_abc123xyz" \
  --type "entry"  # 或 exit, both
```

#### 10.8.2 配置文件

```yaml
# /etc/nodeclient/config.yaml
hub_url: "https://panel.com"
node_token: "node_abc123xyz"
node_type: "entry"  # entry, exit, both
cache_path: "/var/lib/nodeclient/config.json"
```

---

## 11. 总结

### 11.1 节点类型总结

| 节点类型 | 作用 | 部署位置 | 使用场景 |
|---------|------|---------|---------|
| 入口节点 | 接收用户连接 | 用户网络附近 | 单节点转发、隧道入口 |
| 出口节点 | 连接目标服务器 | 目标网络附近 | 隧道出口、跨网络访问 |
| 双功能节点 | 入口+出口 | 灵活部署 | 节省资源、灵活使用 |

### 11.2 自托管功能总结

**优势**:
- 完全控制节点
- 更好的隐私
- 自定义配置
- 独享带宽

**实现要点**:
- VIP 等级配额限制
- 节点 Token 认证
- 一键部署脚本
- 配置下发机制
- 离线容错支持

**部署流程**:
1. 面板创建节点 → 获取 Token
2. 服务器运行安装脚本
3. 节点自动注册上线
4. 开始接收规则配置


---

## 12. 节点管理与规则管理架构重构

### 12.1 核心概念澄清

#### 12.1.1 节点管理
- **职责**: 管理节点本身（创建、删除、监控、配对）
- **功能**: 
  - 创建/删除节点
  - 监控节点状态
  - 设置入口节点和出口节点的配对关系
  - 管理节点认证 Token

#### 12.1.2 规则管理
- **职责**: 管理转发规则（单节点转发、隧道转发）
- **功能**:
  - 创建转发规则
  - 选择入口节点和出口节点
  - 配置转发参数
  - 启动/停止规则

### 12.2 转发模式详解

#### 12.2.1 单节点转发（入口直出）

**定义**: 流量从入口节点直接转发到目标服务器，不经过出口节点。

**流程**:
```
用户 → 入口节点 → 目标服务器
```

**使用场景**:
- 目标服务器可以直接访问
- 不需要跨网络
- 简单的端口转发

**规则配置**:
```json
{
  "name": "MySQL 直连",
  "mode": "single",
  "entry_node_id": 1,        // 只需要入口节点
  "exit_node_id": null,      // 不需要出口节点
  "target_host": "192.168.1.100",
  "target_port": 3306,
  "listen_port": 13306,
  "protocol": "tcp"
}
```

#### 12.2.2 隧道转发

**定义**: 流量从入口节点经过出口节点，再到目标服务器。必须同时指定入口节点和出口节点。

**流程**:
```
用户 → 入口节点 → 出口节点 → 目标服务器
```

**使用场景**:
- 跨网络访问
- 目标服务器在内网
- 需要通过跳板访问

**规则配置**:
```json
{
  "name": "跨网络 MySQL 访问",
  "mode": "tunnel",
  "entry_node_id": 1,        // 必须：入口节点
  "exit_node_id": 2,         // 必须：出口节点
  "target_host": "192.168.1.100",
  "target_port": 3306,
  "listen_port": 13306,
  "protocol": "tcp"
}
```

### 12.3 数据库设计更新

#### 12.3.1 nodes 表（简化）

```sql
CREATE TABLE nodes (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  user_id INTEGER NOT NULL,
  name VARCHAR(100) NOT NULL,
  
  -- 节点状态
  status VARCHAR(20) NOT NULL DEFAULT 'offline', -- online, offline, maintain
  
  -- 节点信息
  host VARCHAR(255) NOT NULL,
  port INTEGER NOT NULL,
  region VARCHAR(50),
  
  -- 自托管标识
  is_self_hosted BOOLEAN DEFAULT FALSE,
  
  -- 公共节点标识
  is_public BOOLEAN DEFAULT FALSE,
  
  -- 流量倍率
  traffic_multiplier DECIMAL(5,2) DEFAULT 1.0,
  
  -- 系统信息
  cpu_usage DECIMAL(5,2),
  memory_usage DECIMAL(5,2),
  disk_usage DECIMAL(5,2),
  bandwidth_in BIGINT DEFAULT 0,
  bandwidth_out BIGINT DEFAULT 0,
  connections INTEGER DEFAULT 0,
  
  -- 认证
  token_hash VARCHAR(255) UNIQUE,
  
  -- 配置版本
  config_version INTEGER DEFAULT 0,
  
  -- 节点描述
  description TEXT,
  
  last_heartbeat_at DATETIME,
  created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
  
  FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX idx_nodes_user_id ON nodes(user_id);
CREATE INDEX idx_nodes_status ON nodes(status);
CREATE INDEX idx_nodes_is_self_hosted ON nodes(is_self_hosted);
CREATE INDEX idx_nodes_is_public ON nodes(is_public);
```

**注意**: 移除了 `type` 字段，节点不再区分入口/出口类型，而是在规则中指定用途。

#### 12.3.2 node_pairs 表（节点配对关系）

```sql
CREATE TABLE node_pairs (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  user_id INTEGER NOT NULL,
  entry_node_id INTEGER NOT NULL,    -- 入口节点
  exit_node_id INTEGER NOT NULL,     -- 出口节点
  name VARCHAR(100),                 -- 配对名称
  is_enabled BOOLEAN DEFAULT TRUE,   -- 是否启用
  description TEXT,
  created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
  
  FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
  FOREIGN KEY (entry_node_id) REFERENCES nodes(id) ON DELETE CASCADE,
  FOREIGN KEY (exit_node_id) REFERENCES nodes(id) ON DELETE CASCADE,
  
  UNIQUE(entry_node_id, exit_node_id)  -- 同一对节点只能配对一次
);

CREATE INDEX idx_node_pairs_user_id ON node_pairs(user_id);
CREATE INDEX idx_node_pairs_entry_node ON node_pairs(entry_node_id);
CREATE INDEX idx_node_pairs_exit_node ON node_pairs(exit_node_id);
```

#### 12.3.3 rules 表（更新）

```sql
CREATE TABLE rules (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  user_id INTEGER NOT NULL,
  name VARCHAR(100) NOT NULL,

  -- 转发模式：single(单节点转发/入口直出), tunnel(隧道转发)
  mode VARCHAR(20) NOT NULL,
  
  -- 协议
  protocol VARCHAR(20) NOT NULL, -- tcp, udp, ws, tls, quic

  -- 入口节点（必须）
  entry_node_id INTEGER NOT NULL,
  
  -- 出口节点（隧道模式必须，单节点模式为 NULL）
  exit_node_id INTEGER,

  -- 目标地址
  target_host VARCHAR(255) NOT NULL,
  target_port INTEGER NOT NULL,

  -- 监听配置
  listen_host VARCHAR(255) DEFAULT '0.0.0.0',
  listen_port INTEGER NOT NULL,

  -- 状态
  status VARCHAR(20) NOT NULL DEFAULT 'stopped', -- running, stopped, paused
  sync_status VARCHAR(20) DEFAULT 'pending', -- pending, synced, failed

  -- 实例信息
  instance_id VARCHAR(100),
  instance_status TEXT,

  -- 流量统计
  traffic_in BIGINT DEFAULT 0,
  traffic_out BIGINT DEFAULT 0,
  connections INTEGER DEFAULT 0,

  -- 配置
  config_json TEXT,

  -- 配置版本
  config_version INTEGER DEFAULT 0,

  created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,

  FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
  FOREIGN KEY (entry_node_id) REFERENCES nodes(id) ON DELETE CASCADE,
  FOREIGN KEY (exit_node_id) REFERENCES nodes(id) ON DELETE SET NULL,
  
  -- 约束：隧道模式必须有出口节点
  CHECK (
    (mode = 'single' AND exit_node_id IS NULL) OR
    (mode = 'tunnel' AND exit_node_id IS NOT NULL)
  )
);

CREATE INDEX idx_rules_user_id ON rules(user_id);
CREATE INDEX idx_rules_entry_node_id ON rules(entry_node_id);
CREATE INDEX idx_rules_exit_node_id ON rules(exit_node_id);
CREATE INDEX idx_rules_mode ON rules(mode);
CREATE INDEX idx_rules_status ON rules(status);
```

### 12.4 API 设计更新

#### 12.4.1 节点管理 API

**POST /api/v1/nodes** - 创建节点
```json
Request:
{
  "name": "我的节点1",
  "host": "node1.example.com",
  "port": 8080,
  "region": "us-west",
  "is_self_hosted": true,
  "description": "部署在 AWS 的节点"
}

Response:
{
  "success": true,
  "data": {
    "node": {
      "id": 1,
      "name": "我的节点1",
      "is_self_hosted": true,
      "status": "offline",
      "token": "node_abc123xyz..."
    },
    "install_command": "curl -fsSL https://panel.com/install.sh | bash -s -- --hub-url https://panel.com --token node_abc123xyz"
  }
}
```

**GET /api/v1/nodes** - 获取节点列表

**GET /api/v1/nodes/:id** - 获取节点详情

**PUT /api/v1/nodes/:id** - 更新节点信息

**DELETE /api/v1/nodes/:id** - 删除节点

#### 12.4.2 节点配对 API

**POST /api/v1/node-pairs** - 创建节点配对
```json
Request:
{
  "name": "美西到亚洲",
  "entry_node_id": 1,
  "exit_node_id": 2,
  "description": "从美西入口到亚洲出口"
}

Response:
{
  "success": true,
  "data": {
    "id": 1,
    "name": "美西到亚洲",
    "entry_node": {
      "id": 1,
      "name": "美西节点",
      "region": "us-west"
    },
    "exit_node": {
      "id": 2,
      "name": "亚洲节点",
      "region": "asia-east"
    },
    "is_enabled": true
  }
}
```

**GET /api/v1/node-pairs** - 获取节点配对列表
```json
Response:
{
  "success": true,
  "data": [
    {
      "id": 1,
      "name": "美西到亚洲",
      "entry_node": {
        "id": 1,
        "name": "美西节点",
        "status": "online"
      },
      "exit_node": {
        "id": 2,
        "name": "亚洲节点",
        "status": "online"
      },
      "is_enabled": true
    }
  ]
}
```

**PUT /api/v1/node-pairs/:id** - 更新节点配对

**DELETE /api/v1/node-pairs/:id** - 删除节点配对

**PUT /api/v1/node-pairs/:id/toggle** - 启用/禁用节点配对

#### 12.4.3 规则管理 API（更新）

**POST /api/v1/rules** - 创建规则

**单节点转发（入口直出）**:
```json
Request:
{
  "name": "MySQL 直连",
  "mode": "single",
  "entry_node_id": 1,        // 必须：入口节点
  "target_host": "192.168.1.100",
  "target_port": 3306,
  "listen_port": 13306,
  "protocol": "tcp"
}
```

**隧道转发**:
```json
Request:
{
  "name": "跨网络 MySQL",
  "mode": "tunnel",
  "entry_node_id": 1,        // 必须：入口节点
  "exit_node_id": 2,         // 必须：出口节点
  "target_host": "192.168.1.100",
  "target_port": 3306,
  "listen_port": 13306,
  "protocol": "tcp"
}
```

**GET /api/v1/rules** - 获取规则列表

**GET /api/v1/rules/:id** - 获取规则详情

**PUT /api/v1/rules/:id** - 更新规则

**DELETE /api/v1/rules/:id** - 删除规则

**POST /api/v1/rules/:id/start** - 启动规则

**POST /api/v1/rules/:id/stop** - 停止规则

### 12.5 业务逻辑实现

#### 12.5.1 创建规则时的验证

```go
// internal/services/rule_service.go
package services

func (s *RuleService) CreateRule(userID int, req *CreateRuleRequest) (*Rule, error) {
    // 1. 验证入口节点
    entryNode, err := s.nodeRepo.FindByID(req.EntryNodeID)
    if err != nil {
        return nil, fmt.Errorf("入口节点不存在")
    }
    
    if entryNode.Status != "online" {
        return nil, fmt.Errorf("入口节点离线")
    }

    // 2. 如果是隧道模式，验证出口节点
    if req.Mode == "tunnel" {
        if req.ExitNodeID == nil {
            return nil, fmt.Errorf("隧道模式必须指定出口节点")
        }
        
        exitNode, err := s.nodeRepo.FindByID(*req.ExitNodeID)
        if err != nil {
            return nil, fmt.Errorf("出口节点不存在")
        }
        
        if exitNode.Status != "online" {
            return nil, fmt.Errorf("出口节点离线")
        }
        
        // 检查节点配对是否存在且启用
        pair, err := s.nodePairRepo.FindByNodes(req.EntryNodeID, *req.ExitNodeID)
        if err != nil || !pair.IsEnabled {
            return nil, fmt.Errorf("入口节点和出口节点未配对或配对已禁用")
        }
    } else if req.Mode == "single" {
        // 单节点模式不能有出口节点
        if req.ExitNodeID != nil {
            return nil, fmt.Errorf("单节点模式不能指定出口节点")
        }
    }

    // 3. 检查端口冲突
    var existingRule Rule
    err = s.db.Where("entry_node_id = ? AND listen_port = ? AND status != ?", 
        req.EntryNodeID, req.ListenPort, "stopped").
        First(&existingRule).Error
    
    if err == nil {
        return nil, fmt.Errorf("端口 %d 已被规则 %s 占用", req.ListenPort, existingRule.Name)
    }

    // 4. 创建规则
    rule := &Rule{
        UserID:       userID,
        Name:         req.Name,
        Mode:         req.Mode,
        Protocol:     req.Protocol,
        EntryNodeID:  req.EntryNodeID,
        ExitNodeID:   req.ExitNodeID,
        TargetHost:   req.TargetHost,
        TargetPort:   req.TargetPort,
        ListenHost:   "0.0.0.0",
        ListenPort:   req.ListenPort,
        Status:       "stopped",
        SyncStatus:   "pending",
    }

    if err := s.db.Create(rule).Error; err != nil {
        return nil, err
    }

    return rule, nil
}
```

#### 12.5.2 节点配对管理

```go
// internal/services/node_pair_service.go
package services

type NodePairService struct {
    db *gorm.DB
}

func (s *NodePairService) CreatePair(userID int, req *CreatePairRequest) (*NodePair, error) {
    // 1. 验证节点存在且属于用户
    var entryNode, exitNode Node
    
    if err := s.db.Where("id = ? AND user_id = ?", req.EntryNodeID, userID).
        First(&entryNode).Error; err != nil {
        return nil, fmt.Errorf("入口节点不存在或无权限")
    }
    
    if err := s.db.Where("id = ? AND user_id = ?", req.ExitNodeID, userID).
        First(&exitNode).Error; err != nil {
        return nil, fmt.Errorf("出口节点不存在或无权限")
    }

    // 2. 检查是否已配对
    var existing NodePair
    err := s.db.Where("entry_node_id = ? AND exit_node_id = ?", 
        req.EntryNodeID, req.ExitNodeID).First(&existing).Error
    
    if err == nil {
        return nil, fmt.Errorf("该节点对已存在")
    }

    // 3. 创建配对
    pair := &NodePair{
        UserID:      userID,
        EntryNodeID: req.EntryNodeID,
        ExitNodeID:  req.ExitNodeID,
        Name:        req.Name,
        Description: req.Description,
        IsEnabled:   true,
    }

    if err := s.db.Create(pair).Error; err != nil {
        return nil, err
    }

    return pair, nil
}

func (s *NodePairService) GetPairs(userID int) ([]NodePair, error) {
    var pairs []NodePair
    
    err := s.db.Where("user_id = ?", userID).
        Preload("EntryNode").
        Preload("ExitNode").
        Find(&pairs).Error
    
    return pairs, err
}
```

### 12.6 前端界面设计

#### 12.6.1 节点管理页面

**节点列表**
```
┌─────────────────────────────────────────────────────────┐
│ 节点管理                          [+ 创建节点]           │
├─────────────────────────────────────────────────────────┤
│ ID │ 名称      │ 状态   │ 区域    │ 自托管 │ 操作      │
├────┼──────────┼────────┼─────────┼────────┼───────────┤
│ 1  │ 美西节点  │ 🟢在线 │ us-west │ ✓     │ 编辑 删除 │
│ 2  │ 亚洲节点  │ 🟢在线 │ asia    │ ✓     │ 编辑 删除 │
│ 3  │ 欧洲节点  │ 🔴离线 │ europe  │ ✓     │ 编辑 删除 │
└─────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────┐
│ 节点配对管理                      [+ 创建配对]           │
├─────────────────────────────────────────────────────────┤
│ 配对名称      │ 入口节点  │ 出口节点  │ 状态 │ 操作    │
├──────────────┼──────────┼──────────┼──────┼─────────┤
│ 美西到亚洲    │ 美西节点  │ 亚洲节点  │ 启用 │ 编辑 删除│
│ 美西到欧洲    │ 美西节点  │ 欧洲节点  │ 禁用 │ 编辑 删除│
└─────────────────────────────────────────────────────────┘
```

#### 12.6.2 规则创建页面

```
┌─────────────────────────────────────┐
│ 创建转发规则                         │
├─────────────────────────────────────┤
│ 规则名称: [___________________]     │
│                                     │
│ 转发模式: ● 单节点转发（入口直出）  │
│          ○ 隧道转发               │
│                                     │
│ 入口节点: [下拉选择]               │
│   ├─ 美西节点 (在线) ✓             │
│   ├─ 亚洲节点 (在线)               │
│   └─ 欧洲节点 (离线)               │
│                                     │
│ [隧道模式时显示]                    │
│ 出口节点: [下拉选择]               │
│   ├─ 亚洲节点 (在线) ✓             │
│   └─ 欧洲节点 (离线)               │
│                                     │
│ 提示: 入口节点"美西节点"与出口节点  │
│      "亚洲节点"已配对 ✓            │
│                                     │
│ 目标地址: [___________________]     │
│ 目标端口: [____]                   │
│ 监听端口: [____]                   │
│ 协议:     [TCP ▼]                  │
│                                     │
│ [取消]  [创建规则]                 │
└─────────────────────────────────────┘
```

**交互逻辑**:
1. 选择"单节点转发"时，隐藏"出口节点"选择
2. 选择"隧道转发"时，显示"出口节点"选择，并检查节点配对
3. 如果选择的入口和出口节点未配对，显示警告提示

### 12.7 配置下发更新

#### 12.7.1 单节点转发配置

```json
{
  "config_version": 1,
  "rules": [
    {
      "rule_id": 1,
      "mode": "single",
      "listen": {
        "host": "0.0.0.0",
        "port": 13306
      },
      "target": {
        "host": "192.168.1.100",
        "port": 3306
      },
      "protocol": "tcp"
    }
  ]
}
```

#### 12.7.2 隧道转发配置

```json
{
  "config_version": 1,
  "rules": [
    {
      "rule_id": 2,
      "mode": "tunnel",
      "listen": {
        "host": "0.0.0.0",
        "port": 13306
      },
      "exit_node": {
        "host": "exit.example.com",  // 出口节点 IP 由面板下发
        "port": 8081
      },
      "target": {
        "host": "192.168.1.100",
        "port": 3306
      },
      "protocol": "tcp"
    }
  ]
}
```

### 12.8 总结

**核心变更**:
1. ✅ 节点不再区分类型，统一管理
2. ✅ 在节点管理中设置入口-出口配对关系
3. ✅ 规则管理中选择转发模式和节点
4. ✅ 单节点转发 = 入口直出（不需要出口节点）
5. ✅ 隧道转发 = 入口 + 出口（必须配对）

**优势**:
- 更灵活的节点管理
- 清晰的职责划分
- 便于扩展和维护
