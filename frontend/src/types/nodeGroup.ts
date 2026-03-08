// 节点分组类型
export type NodeGroupType = 'entry' | 'exit'
export type LoadBalanceStrategy =
  | 'round_robin'
  | 'least_connections'
  | 'random'

// 节点组配置
export interface NodeGroupConfig {
  allowed_protocols: string[]
  port_range: { start: number; end: number }
  entry_config?: EntryGroupConfig
  exit_config?: ExitGroupConfig
}

export interface EntryGroupConfig {
  require_exit_group: boolean
  traffic_multiplier: number
  dns_load_balance: boolean
}

export interface ExitGroupConfig {
  load_balance_strategy: LoadBalanceStrategy
  health_check_interval: number
  health_check_timeout: number
}

// 统计
export interface NodeGroupStats {
  id: number
  node_group_id: number
  total_nodes: number
  online_nodes: number
  total_traffic_in: number
  total_traffic_out: number
  total_connections: number
  updated_at: string
}

// 节点实例
export interface SystemInfo {
  cpu_usage: number
  memory_usage: number
  disk_usage: number
  bandwidth_in: number
  bandwidth_out: number
  connections: number
}

export interface TrafficStats {
  traffic_in: number
  traffic_out: number
  active_connections: number
}

export interface NodeInstance {
  id: number
  node_group_id: number
  node_id: string
  name: string
  host: string | null
  port: number | null
  status: 'online' | 'offline' | 'maintain'
  is_enabled: boolean
  system_info?: SystemInfo
  traffic_stats?: TrafficStats
  config_version: number
  last_heartbeat_at: string | null
  created_at: string
  updated_at: string
}

// 节点组
export interface NodeGroup {
  id: number
  user_id: number
  name: string
  type: 'entry' | 'exit'
  description: string
  is_enabled: boolean
  config: NodeGroupConfig
  stats?: NodeGroupStats
  node_instances?: NodeInstance[]
  created_at: string
  updated_at: string
}

export interface NodeGroupRelation {
  id: number
  entry_group_id: number
  exit_group_id: number
  is_enabled: boolean
  created_at: string
  entry_group?: NodeGroup
  exit_group?: NodeGroup
}

export interface AccessibleNodeGroup {
  group: NodeGroup
  nodes: NodeInstance[]
  editable: boolean
  is_public: boolean
}

// 转发目标
export interface ForwardTarget {
  host: string
  port: number
  weight: number
}

// 协议配置
export interface ProtocolConfig {
  // TCP 配置
  tcp_keepalive?: boolean
  keepalive_interval?: number // 秒
  connect_timeout?: number // 秒
  read_timeout?: number // 秒

  // UDP 配置
  buffer_size?: number // 字节
  session_timeout?: number // 秒

  // WebSocket 配置
  ws_path?: string
  ping_interval?: number // 秒
  max_message_size?: number // KB
  compression?: boolean

  // TLS 配置
  tls_version?: string // tls1.0, tls1.1, tls1.2, tls1.3
  verify_cert?: boolean
  sni?: string

  // QUIC 配置
  max_streams?: number
  initial_window?: number // KB
  idle_timeout?: number // 秒
  enable_0rtt?: boolean
}

// 隧道配置
export interface TunnelConfig {
  load_balance_strategy: LoadBalanceStrategy
  ip_type: 'ipv4' | 'ipv6' | 'auto'
  enable_proxy_protocol: boolean
  forward_targets: ForwardTarget[]
  health_check_interval?: number
  health_check_timeout?: number
  protocol_config?: ProtocolConfig
}

// 隧道
export interface Tunnel {
  id: number
  user_id: number
  name: string
  description?: string | null
  entry_group_id: number
  exit_group_id?: number | null
  entry_group?: NodeGroup
  exit_group?: NodeGroup
  protocol: 'tcp' | 'udp' | 'ws' | 'wss' | 'tls' | 'quic'
  listen_host: string
  listen_port: number
  remote_host: string
  remote_port: number
  status: 'running' | 'stopped' | 'error'
  traffic_in: number
  traffic_out: number
  config?: TunnelConfig
  config_json?: string
  created_at: string
  updated_at: string
}

// 请求类型
export interface CreateNodeGroupPayload {
  name: string
  type: 'entry' | 'exit'
  description?: string
  config: NodeGroupConfig
}

export interface UpdateNodeGroupPayload {
  name?: string
  description?: string
  config?: NodeGroupConfig
  is_enabled?: boolean
}

export interface DeployNodePayload {
  service_name: string
  debug_mode?: boolean
}

export interface DeployCommandResponse {
  node_id: string
  command: string
  service_name: string
}

export interface CreateTunnelPayload {
  name: string
  description?: string
  entry_group_id: number
  exit_group_id?: number | null
  protocol: string
  listen_host?: string
  listen_port?: number
  remote_host: string
  remote_port: number
  config?: TunnelConfig
}

// 通用分页
export interface PaginationResult<T> {
  items: T[]
  total: number
  page: number
  page_size: number
}
