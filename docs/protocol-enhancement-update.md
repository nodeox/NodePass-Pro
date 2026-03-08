# 隧道协议增强更新 - 2026-03-07

## 更新概述

本次更新完善了 NodePass-Pro 的隧道协议支持，新增了 WSS 协议支持，并为所有协议添加了详细的配置选项和验证逻辑。

## 主要更新内容

### 1. 后端更新

#### 1.1 协议支持扩展

**文件**: `backend/internal/services/tunnel_service.go`

- 新增 WSS (WebSocket SSL) 协议支持
- 更新协议标准化函数 `tunnelNormalizeProtocol`，支持 6 种协议：
  - TCP
  - UDP
  - WS (WebSocket)
  - WSS (WebSocket SSL) ✨ 新增
  - TLS
  - QUIC

**文件**: `backend/internal/services/node_group_service.go`

- 更新允许协议列表，添加 WSS 支持

#### 1.2 协议配置模型

**文件**: `backend/internal/models/node_group.go`

新增 `ProtocolConfig` 结构体，支持各协议的详细配置：

```go
type ProtocolConfig struct {
    // TCP 配置
    TCPKeepalive      *bool
    KeepaliveInterval *int  // 秒
    ConnectTimeout    *int  // 秒
    ReadTimeout       *int  // 秒

    // UDP 配置
    BufferSize     *int  // 字节
    SessionTimeout *int  // 秒

    // WebSocket 配置
    WSPath         *string
    PingInterval   *int  // 秒
    MaxMessageSize *int  // KB
    Compression    *bool

    // TLS 配置
    TLSVersion *string  // tls1.0, tls1.1, tls1.2, tls1.3
    VerifyCert *bool
    SNI        *string

    // QUIC 配置
    MaxStreams    *int
    InitialWindow *int  // KB
    IdleTimeout   *int  // 秒
    Enable0RTT    *bool
}
```

#### 1.3 配置验证

**文件**: `backend/internal/services/tunnel_service.go`

新增 `validateProtocolConfig` 函数，提供完整的协议配置验证：

- **TCP 配置验证**
  - keepalive_interval: 1-300 秒
  - connect_timeout: 1-60 秒
  - read_timeout: 1-300 秒

- **UDP 配置验证**
  - buffer_size: 1024-65536 字节
  - session_timeout: 10-600 秒

- **WebSocket 配置验证**
  - ws_path: 必须以 / 开头
  - ping_interval: 5-300 秒
  - max_message_size: 1-10240 KB

- **TLS 配置验证**
  - tls_version: tls1.0/tls1.1/tls1.2/tls1.3

- **QUIC 配置验证**
  - max_streams: 1-1000
  - initial_window: 16-1024 KB
  - idle_timeout: 10-600 秒

#### 1.4 测试覆盖

**文件**: `backend/internal/services/protocol_config_test.go` ✨ 新增

- 12 个协议配置验证测试用例
- 12 个协议标准化测试用例
- 所有测试通过 ✅

### 2. 前端更新

#### 2.1 类型定义

**文件**: `frontend/src/types/nodeGroup.ts`

新增 `ProtocolConfig` 接口，与后端保持一致：

```typescript
export interface ProtocolConfig {
  // TCP 配置
  tcp_keepalive?: boolean
  keepalive_interval?: number
  connect_timeout?: number
  read_timeout?: number

  // UDP 配置
  buffer_size?: number
  session_timeout?: number

  // WebSocket 配置
  ws_path?: string
  ping_interval?: number
  max_message_size?: number
  compression?: boolean

  // TLS 配置
  tls_version?: string
  verify_cert?: boolean
  sni?: string

  // QUIC 配置
  max_streams?: number
  initial_window?: number
  idle_timeout?: number
  enable_0rtt?: boolean
}
```

更新 `TunnelConfig` 接口，添加 `protocol_config` 字段。

#### 2.2 表单集成

**文件**: `frontend/src/pages/tunnels/TunnelList.tsx`

- 集成 `ProtocolConfig` 组件到隧道创建/编辑表单
- 更新表单提交逻辑，包含协议配置数据
- 更新编辑和复制功能，正确处理协议配置

#### 2.3 协议选择器

**文件**: `frontend/src/pages/tunnels/components/ProtocolSelector.tsx`

已支持的协议选择器，包含 WSS 协议：
- TCP
- UDP
- WebSocket (WS)
- WebSocket SSL (WSS) ✅
- TLS
- QUIC

#### 2.4 协议配置组件

**文件**: `frontend/src/pages/tunnels/components/ProtocolConfig.tsx`

可折叠的协议配置面板，根据选择的协议动态显示相应配置项。

#### 2.5 协议统计

**文件**: `frontend/src/pages/tunnels/components/ProtocolStats.tsx`

协议使用统计和可视化图表。

## 技术亮点

### 1. 类型安全

- 后端使用指针类型 (`*int`, `*bool`, `*string`) 区分未设置和零值
- 前端使用可选属性 (`?`) 保持类型一致性
- 完整的类型定义确保前后端数据结构一致

### 2. 验证完整性

- 后端提供详细的参数范围验证
- 前端表单提供输入限制和提示
- 双重验证确保数据安全性

### 3. 向后兼容

- 协议配置为可选字段，不影响现有隧道
- 默认值合理，开箱即用
- 渐进式增强，不破坏现有功能

### 4. 测试覆盖

- 单元测试覆盖所有验证逻辑
- 边界条件测试
- 协议标准化测试

## 使用示例

### 创建带 TCP 配置的隧道

```json
{
  "name": "TCP 隧道",
  "protocol": "tcp",
  "entry_group_id": 1,
  "exit_group_id": 2,
  "remote_host": "example.com",
  "remote_port": 8080,
  "config": {
    "load_balance_strategy": "round_robin",
    "ip_type": "auto",
    "protocol_config": {
      "tcp_keepalive": true,
      "keepalive_interval": 60,
      "connect_timeout": 10,
      "read_timeout": 30
    }
  }
}
```

### 创建带 WebSocket SSL 配置的隧道

```json
{
  "name": "WSS 隧道",
  "protocol": "wss",
  "entry_group_id": 1,
  "remote_host": "ws.example.com",
  "remote_port": 443,
  "config": {
    "protocol_config": {
      "ws_path": "/ws",
      "ping_interval": 30,
      "max_message_size": 1024,
      "compression": true,
      "tls_version": "tls1.3",
      "verify_cert": true,
      "sni": "ws.example.com"
    }
  }
}
```

### 创建带 QUIC 配置的隧道

```json
{
  "name": "QUIC 隧道",
  "protocol": "quic",
  "entry_group_id": 1,
  "exit_group_id": 2,
  "remote_host": "quic.example.com",
  "remote_port": 443,
  "config": {
    "protocol_config": {
      "max_streams": 100,
      "initial_window": 256,
      "idle_timeout": 30,
      "enable_0rtt": true
    }
  }
}
```

## 测试结果

### 后端测试

```bash
$ go test -v ./internal/services -run TestValidateProtocolConfig
=== RUN   TestValidateProtocolConfig
--- PASS: TestValidateProtocolConfig (0.00s)
PASS
ok      nodepass-pro/backend/internal/services  0.271s

$ go test -v ./internal/services -run TestTunnelNormalizeProtocol
=== RUN   TestTunnelNormalizeProtocol
--- PASS: TestTunnelNormalizeProtocol (0.00s)
PASS
ok      nodepass-pro/backend/internal/services  0.233s
```

### 前端编译

```bash
$ npm run build
✓ built in 4.38s
```

## 后续优化建议

### 1. 协议转换

- 实现协议转换功能（TCP ↔ WebSocket）
- 自动协议检测和适配
- 协议降级和回退机制

### 2. 性能监控

- 实时监控协议性能指标
- 延迟、丢包率、吞吐量统计
- 性能告警和优化建议

### 3. 协议模板

- 预设常用协议配置模板
- 一键应用最佳实践配置
- 自定义模板保存和分享

### 4. 协议测试工具

- 协议连通性测试
- 性能基准测试
- 配置验证工具

### 5. 更多协议支持

- HTTP/2 和 HTTP/3
- gRPC
- MQTT
- 自定义协议扩展

## 相关文件

### 后端文件

- `backend/internal/models/node_group.go` - 添加 ProtocolConfig 结构
- `backend/internal/services/tunnel_service.go` - 协议验证和处理
- `backend/internal/services/node_group_service.go` - 允许协议列表
- `backend/internal/services/protocol_config_test.go` - 测试文件 ✨

### 前端文件

- `frontend/src/types/nodeGroup.ts` - 类型定义
- `frontend/src/pages/tunnels/TunnelList.tsx` - 表单集成
- `frontend/src/pages/tunnels/components/ProtocolConfig.tsx` - 配置组件
- `frontend/src/pages/tunnels/components/ProtocolSelector.tsx` - 选择器
- `frontend/src/pages/tunnels/components/ProtocolStats.tsx` - 统计组件

## 兼容性说明

- ✅ 向后兼容，不影响现有隧道
- ✅ 协议配置为可选，不配置使用默认值
- ✅ 前后端类型定义一致
- ✅ 所有测试通过

## 总结

本次更新完善了 NodePass-Pro 的隧道协议支持体系，新增了 WSS 协议，并为所有协议提供了详细的配置选项和验证逻辑。通过完整的测试覆盖和类型安全设计，确保了功能的稳定性和可靠性。前端界面友好，后端验证严格，为用户提供了强大而灵活的隧道配置能力。
