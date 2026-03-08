# 节点协议支持增强

## 概述

为 NodePass-Pro 添加了全面的协议支持增强，包括新增 WebSocket SSL (WSS) 和 QUIC 协议支持、可视化协议配置界面、协议统计分析等功能。

## 新增功能

### 1. 协议支持扩展

**新增协议**：
- **WebSocket (WS)** - 全双工通信协议
- **WebSocket SSL (WSS)** - 加密的 WebSocket 连接
- **QUIC** - 基于 UDP 的快速可靠传输协议

**已支持协议**：
- TCP - 可靠的面向连接传输
- UDP - 无连接的快速传输
- TLS - 加密传输层安全

### 2. 协议配置界面

**功能**：
- 可视化配置各协议的参数
- 协议说明和最佳实践提示
- 可折叠的高级配置选项

**TCP 配置**：
- TCP Keep-Alive 开关
- Keep-Alive 间隔（秒）
- 连接超时时间
- 读取超时时间

**UDP 配置**：
- 缓冲区大小（字节）
- 会话超时时间（秒）

**WebSocket 配置**：
- WebSocket 连接路径
- 心跳间隔（Ping 间隔）
- 最大消息大小（KB）
- 压缩开关

**TLS 配置**：
- TLS 版本选择（1.0 - 1.3）
- 证书验证开关
- SNI（Server Name Indication）

**QUIC 配置**：
- 最大并发流数量
- 初始流控窗口大小
- 连接空闲超时
- 0-RTT 快速握手开关

### 3. 协议统计分析

**功能**：
- 按协议类型统计隧道数量
- 按协议统计流量使用
- 可视化图表展示

**统计指标**：
- 各协议隧道数量
- 运行中/已停止隧道数
- 入站流量统计
- 出站流量统计
- 总流量统计

**可视化图表**：
- 协议分布饼图
- 协议流量对比柱状图
- 协议详细统计卡片

### 4. 协议优化建议

**TCP 优化**：
- 启用 Keep-Alive 检测死连接
- 合理设置超时时间
- 适用于可靠传输场景

**UDP 优化**：
- 调整缓冲区大小提升性能
- 设置合理的会话超时
- 适用于实时性要求高的场景

**WebSocket 优化**：
- 启用压缩减少带宽
- 合理设置心跳间隔
- 适用于实时双向通信

**QUIC 优化**：
- 启用 0-RTT 减少延迟
- 调整流控窗口提升吞吐
- 适用于低延迟高性能场景

## 技术实现

### 前端组件结构

```
frontend/src/pages/tunnels/components/
├── TunnelTrafficChart.tsx      # 流量统计图表
├── ProtocolConfig.tsx          # 协议配置组件（新增）
└── ProtocolStats.tsx           # 协议统计组件（新增）
```

### 类型定义更新

```typescript
// 支持的协议类型
protocol: 'tcp' | 'udp' | 'ws' | 'wss' | 'tls' | 'quic'

// 协议配置接口
interface ProtocolConfig {
  tcp_keepalive?: boolean
  keepalive_interval?: number
  connect_timeout?: number
  read_timeout?: number
  buffer_size?: number
  session_timeout?: number
  ws_path?: string
  ping_interval?: number
  max_message_size?: number
  compression?: boolean
  tls_version?: string
  verify_cert?: boolean
  sni?: string
  max_streams?: number
  initial_window?: number
  idle_timeout?: number
  enable_0rtt?: boolean
}
```

### 组件特性

**ProtocolConfig 组件**：
- 根据协议类型动态渲染配置项
- 表单验证和默认值
- 协议说明和提示信息
- 可折叠的高级选项

**ProtocolStats 组件**：
- ECharts 可视化图表
- 实时统计数据
- 协议图标和颜色区分
- 响应式布局

## 使用指南

### 创建隧道时配置协议

1. 选择协议类型（TCP/UDP/WS/WSS/TLS/QUIC）
2. 展开"协议配置"折叠面板
3. 根据需求配置协议参数
4. 查看协议说明和建议
5. 保存创建隧道

### 查看协议统计

1. 进入隧道管理页面
2. 切换到"协议统计"标签页
3. 查看协议分布饼图
4. 查看协议流量对比
5. 查看详细统计数据

### 协议选择建议

**TCP**：
- 适用场景：需要可靠传输的应用（HTTP、数据库、文件传输）
- 优点：可靠、有序、错误检测
- 缺点：延迟较高、连接开销大

**UDP**：
- 适用场景：实时性要求高的应用（视频、游戏、VoIP）
- 优点：低延迟、无连接开销
- 缺点：不可靠、可能丢包

**WebSocket (WS/WSS)**：
- 适用场景：Web 实时通信（聊天、推送、协作）
- 优点：全双工、低延迟、浏览器原生支持
- 缺点：需要 HTTP 升级握手

**TLS**：
- 适用场景：需要加密的安全传输
- 优点：端到端加密、身份验证
- 缺点：握手开销、性能损耗

**QUIC**：
- 适用场景：低延迟高性能应用（HTTP/3、流媒体）
- 优点：0-RTT、多路复用、连接迁移
- 缺点：较新协议、兼容性

## 界面优化

### 隧道列表页改进

- 协议选择增加 WSS 和 QUIC
- 新增"协议统计"标签页
- 协议标签颜色区分

### 创建隧道表单改进

- 协议配置折叠面板
- 协议说明和提示
- 表单验证和默认值

### 协议统计页面

- 协议分布饼图
- 流量对比柱状图
- 详细统计卡片
- 协议图标和颜色

## 性能优化

### 配置优化

- 可选配置项，不影响基本功能
- 默认值合理，开箱即用
- 高级选项折叠，界面简洁

### 统计优化

- 客户端计算，无需额外 API
- 图表按需渲染
- 数据缓存避免重复计算

## 注意事项

### 协议配置

- 配置项为可选，不配置使用默认值
- 不同协议配置项不同
- 配置错误可能导致连接失败

### 协议选择

- 根据实际需求选择合适协议
- 考虑性能、可靠性、安全性平衡
- 测试验证协议是否满足需求

### 协议统计

- 统计数据基于当前隧道列表
- 流量数据可能存在延迟
- 图表数据实时更新

## 后续优化建议

1. **协议转换功能**
   - 实现协议转换（TCP ↔ WebSocket）
   - 自动协议检测和适配
   - 协议降级和回退

2. **协议性能监控**
   - 实时监控协议性能指标
   - 延迟、丢包率、吞吐量
   - 性能告警和优化建议

3. **协议模板**
   - 预设常用协议配置模板
   - 一键应用最佳实践配置
   - 自定义模板保存

4. **协议测试工具**
   - 协议连通性测试
   - 性能基准测试
   - 配置验证工具

5. **更多协议支持**
   - HTTP/2 和 HTTP/3
   - gRPC
   - MQTT
   - 自定义协议

## 相关文件

### 修改的文件

- `frontend/src/types/nodeGroup.ts` - 添加 WSS 和 QUIC 协议类型
- `frontend/src/pages/tunnels/TunnelList.tsx` - 集成协议配置和统计

### 新增的文件

- `frontend/src/pages/tunnels/components/ProtocolConfig.tsx` - 协议配置组件
- `frontend/src/pages/tunnels/components/ProtocolStats.tsx` - 协议统计组件

## 兼容性

- 向后兼容，不影响现有隧道
- 新协议需要后端支持
- 配置项可选，不影响基本功能

## 测试建议

### 功能测试

- [ ] 创建各种协议的隧道
- [ ] 配置各协议参数
- [ ] 查看协议统计数据
- [ ] 协议图表显示正确
- [ ] 配置验证正常

### 协议测试

- [ ] TCP 隧道连接正常
- [ ] UDP 隧道传输正常
- [ ] WebSocket 握手成功
- [ ] TLS 加密正常
- [ ] QUIC 连接建立

### 性能测试

- [ ] 协议配置不影响性能
- [ ] 统计计算效率高
- [ ] 图表渲染流畅
- [ ] 大量隧道时性能正常
