# 监控和告警系统

## 功能概述

实现了完整的监控和告警系统，包括告警管理、通知渠道、节点健康监控等功能。

---

## 已实现功能

### 1. 告警系统核心

#### 数据模型
- **Alert（告警记录）**
  - 支持多种告警类型：节点离线、CPU/内存/磁盘过高、流量配额、高延迟等
  - 4 个告警级别：info、warning、error、critical
  - 5 种告警状态：pending、firing、resolved、silenced、acknowledged
  - 告警去重（基于指纹）
  - 告警聚合和统计

- **AlertRule（告警规则）**
  - 自定义告警条件
  - 阈值配置
  - 评估间隔和持续时间
  - 通知间隔和最大通知次数
  - 静默时间段配置

- **NotificationChannel（通知渠道）**
  - 支持多种通知方式：邮件、Telegram、Webhook、Slack
  - 渠道配置管理
  - 发送统计和错误追踪

#### 服务层
- **AlertService** - 告警管理
  - 创建/解决/确认/静默告警
  - 告警列表和统计
  - 自动清理已解决告警

- **AlertRuleService** - 规则管理
  - CRUD 操作
  - 启用/禁用规则

- **NotificationChannelService** - 渠道管理
  - CRUD 操作
  - 测试通知
  - 发送统计

---

## 告警类型

### 节点相关
- `node_offline` - 节点离线
- `node_high_cpu` - CPU 使用率过高
- `node_high_memory` - 内存使用率过高
- `node_high_disk` - 磁盘使用率过高

### 流量相关
- `traffic_quota` - 流量配额告警

### 性能相关
- `high_latency` - 高延迟
- `high_packet_loss` - 高丢包率

### 系统相关
- `system_error` - 系统错误

---

## 告警级别

| 级别 | 说明 | 颜色 | 示例 |
|------|------|------|------|
| info | 信息 | 蓝色 | 节点上线通知 |
| warning | 警告 | 黄色 | CPU 使用率 > 70% |
| error | 错误 | 橙色 | 节点离线 |
| critical | 严重 | 红色 | 所有节点离线 |

---

## 告警状态流转

```
pending → firing → resolved
              ↓
         acknowledged
              ↓
          silenced
```

---

## API 接口（待实现）

### 告警管理

#### 1. 获取告警列表
**GET** `/api/v1/alerts`

**查询参数**:
- `status` - 状态过滤（可多选）
- `level` - 级别过滤（可多选）
- `resource_type` - 资源类型
- `page` - 页码
- `page_size` - 每页数量

#### 2. 获取告警详情
**GET** `/api/v1/alerts/:id`

#### 3. 确认告警
**POST** `/api/v1/alerts/:id/acknowledge`

#### 4. 解决告警
**POST** `/api/v1/alerts/:id/resolve`

**请求**:
```json
{
  "notes": "问题已修复"
}
```

#### 5. 静默告警
**POST** `/api/v1/alerts/:id/silence`

**请求**:
```json
{
  "duration": 3600  // 秒
}
```

#### 6. 获取告警统计
**GET** `/api/v1/alerts/stats`

**响应**:
```json
{
  "by_status": [
    {"status": "firing", "count": 5},
    {"status": "resolved", "count": 10}
  ],
  "by_level": [
    {"level": "critical", "count": 2},
    {"level": "warning", "count": 3}
  ],
  "by_type": [
    {"type": "node_offline", "count": 2},
    {"type": "node_high_cpu", "count": 3}
  ]
}
```

### 告警规则管理

#### 1. 创建告警规则
**POST** `/api/v1/alert-rules`

**请求**:
```json
{
  "name": "节点 CPU 过高",
  "description": "当节点 CPU 使用率超过 80% 持续 5 分钟时触发",
  "type": "node_high_cpu",
  "level": "warning",
  "condition": "cpu_usage > 80",
  "threshold": "80",
  "duration": 300,
  "eval_interval": 60,
  "notify_channels": ["1", "2"],
  "notify_interval": 300,
  "max_notifications": 10
}
```

#### 2. 更新告警规则
**PUT** `/api/v1/alert-rules/:id`

#### 3. 删除告警规则
**DELETE** `/api/v1/alert-rules/:id`

#### 4. 获取告警规则列表
**GET** `/api/v1/alert-rules`

#### 5. 启用/禁用规则
**POST** `/api/v1/alert-rules/:id/toggle`

### 通知渠道管理

#### 1. 创建通知渠道
**POST** `/api/v1/notification-channels`

**邮件渠道示例**:
```json
{
  "name": "管理员邮箱",
  "type": "email",
  "description": "发送到管理员邮箱",
  "config": {
    "smtp_host": "smtp.example.com",
    "smtp_port": 587,
    "smtp_user": "alert@example.com",
    "smtp_password": "password",
    "from": "alert@example.com",
    "to": ["admin@example.com"]
  }
}
```

**Telegram 渠道示例**:
```json
{
  "name": "Telegram 通知",
  "type": "telegram",
  "config": {
    "bot_token": "your-bot-token",
    "chat_id": "your-chat-id"
  }
}
```

**Webhook 渠道示例**:
```json
{
  "name": "Webhook 通知",
  "type": "webhook",
  "config": {
    "url": "https://example.com/webhook",
    "method": "POST",
    "headers": {
      "Authorization": "Bearer token"
    }
  }
}
```

#### 2. 测试通知渠道
**POST** `/api/v1/notification-channels/:id/test`

#### 3. 获取通知渠道列表
**GET** `/api/v1/notification-channels`

---

## 节点健康监控（待实现）

### 监控指标

#### 系统资源
- CPU 使用率
- 内存使用率
- 磁盘使用率
- 网络带宽

#### 性能指标
- 延迟（ping）
- 丢包率
- 连接数
- QPS

#### 状态指标
- 在线/离线状态
- 最后心跳时间
- 健康评分

### 健康评分算法

```
健康评分 = 100 - (CPU权重 * CPU使用率 + 内存权重 * 内存使用率 + ...)

权重分配：
- CPU: 30%
- 内存: 25%
- 磁盘: 15%
- 延迟: 20%
- 丢包率: 10%

评分等级：
- 90-100: 优秀（绿色）
- 70-89: 良好（蓝色）
- 50-69: 一般（黄色）
- 30-49: 较差（橙色）
- 0-29: 危险（红色）
```

---

## 监控 Dashboard（待实现）

### 实时监控
- 节点状态总览
- 实时告警列表
- 关键指标图表
- 节点拓扑图

### 历史趋势
- CPU/内存/磁盘使用趋势
- 流量趋势
- 延迟趋势
- 告警趋势

### 统计报表
- 节点可用性统计
- 告警统计
- 流量统计
- 性能统计

---

## 告警规则示例

### 1. 节点离线告警
```json
{
  "name": "节点离线",
  "type": "node_offline",
  "level": "error",
  "condition": "last_heartbeat > 5m",
  "duration": 300,
  "notify_channels": ["telegram", "email"]
}
```

### 2. CPU 过高告警
```json
{
  "name": "CPU 使用率过高",
  "type": "node_high_cpu",
  "level": "warning",
  "condition": "cpu_usage > 80",
  "threshold": "80",
  "duration": 300,
  "notify_interval": 600
}
```

### 3. 流量配额告警
```json
{
  "name": "流量配额即将用尽",
  "type": "traffic_quota",
  "level": "warning",
  "condition": "traffic_usage > 90%",
  "threshold": "90",
  "notify_channels": ["email"]
}
```

---

## 通知模板

### 邮件模板
```
主题：[{level}] {title}

告警详情：
- 类型：{type}
- 级别：{level}
- 资源：{resource_name}
- 触发值：{value}
- 阈值：{threshold}
- 触发时间：{first_fired_at}

消息：
{message}

查看详情：{alert_url}
```

### Telegram 模板
```
🚨 [{level}] {title}

📊 {resource_name}
⚠️ {value} / {threshold}
🕐 {first_fired_at}

{message}
```

---

## 后续开发计划

### Phase 1: 核心功能（当前）
- [x] 告警数据模型
- [x] 告警服务层
- [ ] 告警 API 接口
- [ ] 告警规则评估引擎

### Phase 2: 通知系统
- [ ] 邮件通知实现
- [ ] Telegram 通知实现
- [ ] Webhook 通知实现
- [ ] 通知模板系统

### Phase 3: 节点监控
- [ ] 节点健康检查
- [ ] 性能指标收集
- [ ] 健康评分计算
- [ ] 历史数据存储

### Phase 4: Dashboard
- [ ] 实时监控页面
- [ ] 告警管理页面
- [ ] 规则配置页面
- [ ] 统计报表页面

### Phase 5: 高级功能
- [ ] 告警聚合和关联
- [ ] 智能告警降噪
- [ ] 告警预测
- [ ] 自动修复建议

---

## 数据库表结构

### alerts 表
```sql
CREATE TABLE alerts (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    created_at TIMESTAMP,
    updated_at TIMESTAMP,
    deleted_at TIMESTAMP,

    type VARCHAR(50) NOT NULL,
    level VARCHAR(20) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    title VARCHAR(255) NOT NULL,
    message TEXT,
    fingerprint VARCHAR(64) UNIQUE,

    resource_type VARCHAR(50),
    resource_id BIGINT,
    resource_name VARCHAR(255),

    labels JSON,
    annotations JSON,
    value VARCHAR(255),
    threshold VARCHAR(255),

    first_fired_at TIMESTAMP,
    last_fired_at TIMESTAMP,
    resolved_at TIMESTAMP,
    acknowledged_at TIMESTAMP,
    silenced_until TIMESTAMP,

    notification_sent BOOLEAN DEFAULT FALSE,
    notification_count INT DEFAULT 0,
    last_notified_at TIMESTAMP,

    acknowledged_by BIGINT,
    resolved_by BIGINT,
    notes TEXT,

    INDEX idx_type (type),
    INDEX idx_level (level),
    INDEX idx_status (status),
    INDEX idx_resource (resource_type, resource_id),
    INDEX idx_fingerprint (fingerprint)
);
```

---

## 相关文件

- `backend/internal/models/alert.go` - 告警数据模型
- `backend/internal/services/alert_service.go` - 告警服务
- `docs/feature-monitoring-alerts.md` - 本文档
