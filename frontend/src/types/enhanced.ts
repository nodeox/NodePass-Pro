// 用户相关类型定义
export enum UserRole {
  Admin = 'admin',
  User = 'user',
}

export enum UserStatus {
  Normal = 'normal',
  Suspended = 'suspended',
  Banned = 'banned',
}

export interface User {
  id: number
  username: string
  email: string
  role: UserRole
  status: UserStatus
  vip_level: number
  vip_expires_at: string | null
  traffic_quota: number
  traffic_used: number
  max_rules: number
  max_bandwidth: number
  max_self_hosted_entry_nodes: number
  max_self_hosted_exit_nodes: number
  telegram_id: string | null
  telegram_username: string | null
  created_at: string
  updated_at: string
  last_login_at: string | null
}

// 节点组类型
export enum NodeGroupType {
  Entry = 'entry',
  Exit = 'exit',
}

export enum NodeGroupStatus {
  Enabled = 'enabled',
  Disabled = 'disabled',
}

export interface NodeGroup {
  id: number
  user_id: number
  name: string
  type: NodeGroupType
  description: string | null
  is_enabled: boolean
  config: NodeGroupConfig | null
  created_at: string
  updated_at: string
}

export interface NodeGroupConfig {
  allowed_protocols: string[]
  port_range: {
    start: number
    end: number
  }
  entry_config?: EntryGroupConfig
  exit_config?: ExitGroupConfig
}

export interface EntryGroupConfig {
  require_exit_group: boolean
  default_listen_host: string
}

export interface ExitGroupConfig {
  load_balance_strategy: LoadBalanceStrategy
  health_check_enabled: boolean
  health_check_interval: number
}

export enum LoadBalanceStrategy {
  RoundRobin = 'round_robin',
  Random = 'random',
  LeastConnections = 'least_connections',
  IPHash = 'ip_hash',
}

// 隧道类型
export enum TunnelStatus {
  Running = 'running',
  Stopped = 'stopped',
  Error = 'error',
}

export enum TunnelProtocol {
  TCP = 'tcp',
  UDP = 'udp',
  WS = 'ws',
  WSS = 'wss',
  TLS = 'tls',
  QUIC = 'quic',
}

export interface Tunnel {
  id: number
  user_id: number
  name: string
  description: string | null
  entry_group_id: number
  exit_group_id: number | null
  protocol: TunnelProtocol
  listen_host: string
  listen_port: number
  remote_host: string
  remote_port: number
  status: TunnelStatus
  config: TunnelConfig | null
  created_at: string
  updated_at: string
}

export interface TunnelConfig {
  load_balance_strategy: LoadBalanceStrategy
  ip_type: 'auto' | 'ipv4' | 'ipv6'
  forward_targets: ForwardTarget[]
  timeout: number
  buffer_size: number
}

export interface ForwardTarget {
  host: string
  port: number
  weight: number
}

// VIP 类型
export interface VIPLevel {
  id: number
  level: number
  name: string
  description: string | null
  traffic_quota: number
  max_rules: number
  max_bandwidth: number
  max_self_hosted_entry_nodes: number
  max_self_hosted_exit_nodes: number
  accessible_node_level: number
  traffic_multiplier: number
  custom_features: string | null
  price: number | null
  duration_days: number | null
  created_at: string
  updated_at: string
}

// 权益码类型
export enum BenefitCodeStatus {
  Unused = 'unused',
  Used = 'used',
  Expired = 'expired',
}

export interface BenefitCode {
  id: number
  code: string
  vip_level: number
  duration_days: number
  status: BenefitCodeStatus
  is_enabled: boolean
  expires_at: string | null
  used_by_user_id: number | null
  used_at: string | null
  created_at: string
  updated_at: string
}

// API 响应类型
export interface ApiSuccessResponse<T = unknown> {
  success: true
  data: T
  message?: string
}

export interface ApiErrorResponse {
  success: false
  code: string
  message: string
  timestamp?: string
}

export type ApiResponse<T = unknown> = ApiSuccessResponse<T> | ApiErrorResponse

// 分页类型
export interface PaginationQuery {
  page: number
  page_size: number
}

export interface PaginationResult<T> {
  list: T[]
  total: number
  page: number
  page_size: number
}

// 认证类型
export interface LoginPayload {
  account: string
  password: string
}

export interface RegisterPayload {
  username: string
  email: string
  password: string
}

export interface LoginResult {
  access_token: string
  refresh_token: string
  expires_in: number
  token_type: 'Bearer'
  user: User
}

export interface ChangePasswordPayload {
  old_password: string
  new_password: string
}

export interface ChangeEmailPayload {
  password: string
  new_email: string
  code: string
}

// 流量统计类型
export interface TrafficQuota {
  traffic_quota: number
  traffic_used: number
  usage_percent: number
  is_over_limit: boolean
}

export interface TrafficUsageSummary {
  start_time: string
  end_time: string
  total_traffic_in: number
  total_traffic_out: number
  total_calculated_traffic: number
  record_count: number
}

export interface TrafficRecordItem {
  id: number
  user_id: number
  rule_id: number | null
  node_id: number | null
  hour: string
  traffic_in: number
  traffic_out: number
  calculated_traffic: number
  created_at: string
}

export interface TrafficRecordsQuery extends PaginationQuery {
  rule_id?: number
  node_id?: number
  start_time?: string
  end_time?: string
}

// 公告类型
export enum AnnouncementType {
  Info = 'info',
  Warning = 'warning',
  Error = 'error',
  Success = 'success',
}

export interface AnnouncementRecord {
  id: number
  title: string
  content: string
  type: AnnouncementType
  is_pinned: boolean
  is_enabled: boolean
  created_at: string
  updated_at: string
}

// 审计日志类型
export interface AuditLogRecord {
  id: number
  user_id: number
  action: string
  resource_type: string
  resource_id: string | null
  details: string | null
  ip_address: string
  user_agent: string
  created_at: string
}

// 管理员用户列表类型
export interface AdminUserListQuery extends PaginationQuery {
  role?: UserRole
  status?: UserStatus
  keyword?: string
}

export type AdminUserListResult = PaginationResult<User>

// 权益码类型
export interface BenefitCodeGeneratePayload {
  vip_level: number
  duration_days: number
  count: number
  expires_at?: string
}

export interface BenefitCodeListQuery extends PaginationQuery {
  status?: BenefitCodeStatus
  vip_level?: number
}

export interface BenefitCodeRedeemResult {
  code: string
  applied_level: number
  vip_expires_at: string | null
}

// VIP 类型
export interface VipMyLevelResult {
  user_id: number
  vip_level: number
  vip_expires_at: string | null
  level_detail: VIPLevel | null
}

export interface CreateVipLevelPayload {
  level: number
  name: string
  description?: string
  traffic_quota: number
  max_rules: number
  max_bandwidth: number
  max_self_hosted_entry_nodes: number
  max_self_hosted_exit_nodes: number
  accessible_node_level: number
  traffic_multiplier: number
  custom_features?: string
  price?: number
  duration_days?: number
}

export interface UpdateVipLevelPayload {
  name?: string
  description?: string
  traffic_quota?: number
  max_rules?: number
  max_bandwidth?: number
  max_self_hosted_entry_nodes?: number
  max_self_hosted_exit_nodes?: number
  accessible_node_level?: number
  traffic_multiplier?: number
  custom_features?: string
  price?: number
  duration_days?: number
}

// Telegram 类型
export interface TelegramSSOURLResult {
  sso_url: string
  expires_at: string
}

export interface TelegramSSOLoginResult {
  access_token: string
  refresh_token: string
  expires_in: number
  token_type: 'Bearer'
  user: User
}

export interface SendEmailChangeCodePayload {
  password: string
  new_email: string
}

export interface SendEmailChangeCodeResult {
  expires_at: string
  debug_code?: string
  sent: boolean
}

// 类型守卫
export function isApiSuccessResponse<T>(
  response: ApiResponse<T>
): response is ApiSuccessResponse<T> {
  return response.success === true
}

export function isApiErrorResponse(
  response: ApiResponse
): response is ApiErrorResponse {
  return response.success === false
}

// 类型转换辅助函数
export function parseUserRole(role: string): UserRole {
  if (role === 'admin') return UserRole.Admin
  return UserRole.User
}

export function parseUserStatus(status: string): UserStatus {
  switch (status) {
    case 'suspended':
      return UserStatus.Suspended
    case 'banned':
      return UserStatus.Banned
    default:
      return UserStatus.Normal
  }
}

export function parseTunnelStatus(status: string): TunnelStatus {
  switch (status) {
    case 'running':
      return TunnelStatus.Running
    case 'error':
      return TunnelStatus.Error
    default:
      return TunnelStatus.Stopped
  }
}

export function parseTunnelProtocol(protocol: string): TunnelProtocol {
  switch (protocol.toLowerCase()) {
    case 'tcp':
      return TunnelProtocol.TCP
    case 'udp':
      return TunnelProtocol.UDP
    case 'ws':
      return TunnelProtocol.WS
    case 'wss':
      return TunnelProtocol.WSS
    case 'tls':
      return TunnelProtocol.TLS
    case 'quic':
      return TunnelProtocol.QUIC
    default:
      return TunnelProtocol.TCP
  }
}
