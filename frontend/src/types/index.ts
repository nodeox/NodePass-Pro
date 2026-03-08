export type UserRole = 'admin' | 'user'

export interface User {
  id: number
  username: string
  email: string
  role: UserRole
  status: string
  vipLevel: number
  vipExpiresAt: string | null
  trafficQuota: number
  trafficUsed: number
  maxRules: number
  maxBandwidth: number
  maxSelfHostedEntryNodes: number
  maxSelfHostedExitNodes: number
  telegramId: string | null
  telegramUsername: string | null
  createdAt: string
  lastLoginAt: string | null
  vip_level: number
  vip_expires_at?: string | null
  traffic_quota: number
  traffic_used: number
  telegram_id?: string | null
  telegram_username?: string | null
  created_at?: string
  updated_at?: string
}

export interface ApiResponse<T> {
  success: boolean
  data: T
  message: string
  timestamp: string
}

export interface ApiSuccessResponse<T> {
  success: true
  data: T
  message?: string
  timestamp: string
}

export interface ApiErrorBody {
  code: string
  message: string
}

export interface ApiErrorResponse {
  success: false
  error: ApiErrorBody
  timestamp: string
}

export interface LoginPayload {
  account: string
  password: string
}

export interface RegisterPayload {
  username: string
  email: string
  password: string
}

export interface ChangePasswordPayload {
  old_password: string
  new_password: string
}

export interface SendEmailChangeCodePayload {
  password: string
  new_email: string
}

export interface ChangeEmailPayload {
  new_email: string
  code: string
}

export interface SendEmailChangeCodeResult {
  expires_at: string
  debug_code?: string
  sent?: boolean
}

export interface TelegramSSOURLResult {
  login_url: string
  expires_at: string
  expires_in: number
  redirect_uri?: string
}

export interface TelegramSSOLoginResult extends LoginResult {
  redirect_uri?: string
}

export interface LoginResult {
  token: string
  user: User
}

export interface PaginationQuery {
  page?: number
  pageSize?: number
}

export interface PaginationResult<T> {
  list: T[]
  total: number
  page: number
  page_size: number
}

export interface PaginatedData<T> {
  items: T[]
  total: number
  page: number
  pageSize: number
}

export type NodeStatus = 'online' | 'offline' | 'maintain'

export interface NodeRecord {
  id: number
  user_id: number
  name: string
  status: NodeStatus | string
  host: string
  port: number
  region?: string | null
  is_self_hosted: boolean
  is_public: boolean
  traffic_multiplier: number
  cpu_usage?: number | null
  memory_usage?: number | null
  disk_usage?: number | null
  bandwidth_in: number
  bandwidth_out: number
  connections: number
  last_heartbeat_at?: string | null
  description?: string | null
  config_version: number
  created_at: string
  updated_at: string
}

export interface RuleRecord {
  id: number
  user_id: number
  name: string
  mode: 'single' | 'tunnel'
  protocol: 'tcp' | 'udp' | 'ws' | 'tls' | 'quic'
  entry_node_id: number
  exit_node_id?: number | null
  entry_node?: NodeRecord | null
  exit_node?: NodeRecord | null
  listen_host: string
  listen_port: number
  target_host: string
  target_port: number
  status: string
  sync_status: string
  traffic_in: number
  traffic_out: number
  connections: number
  config_version?: number
  created_at: string
  updated_at: string
}

export interface TrafficQuota {
  trafficQuota: number
  trafficUsed: number
  percentage: number
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

export interface TrafficUsageQuery {
  start_time: string
  end_time: string
}

export interface TrafficRecordItem {
  id: number
  user_id: number
  rule_id?: number | null
  node_id?: number | null
  traffic_in: number
  traffic_out: number
  vip_multiplier: number
  node_multiplier: number
  final_multiplier: number
  calculated_traffic: number
  hour: string
  created_at: string
  rule?: RuleRecord | null
  node?: NodeRecord | null
}

export interface TrafficRecordsQuery extends PaginationQuery {
  rule_id?: number
  node_id?: number
  start_time?: string
  end_time?: string
}

export interface AnnouncementRecord {
  id: number
  title: string
  content: string
  type: 'info' | 'warning' | 'error' | 'success'
  is_enabled: boolean
  start_time?: string | null
  end_time?: string | null
  created_at: string
  updated_at: string
}

export interface AuditLogRecord {
  id: number
  user_id?: number | null
  action: string
  resource_type?: string | null
  resource_id?: number | null
  details?: string | null
  ip_address?: string | null
  user_agent?: string | null
  created_at: string
  user?: User | null
}

export interface VipLevelRecord {
  id: number
  level: number
  name: string
  description?: string | null
  traffic_quota: number
  max_rules: number
  max_bandwidth: number
  max_self_hosted_entry_nodes: number
  max_self_hosted_exit_nodes: number
  accessible_node_level: number
  traffic_multiplier: number
  custom_features?: string | null
  price?: number | null
  duration_days?: number | null
  created_at?: string
  updated_at?: string
}

export interface VipMyLevelResult {
  user_id: number
  vip_level: number
  vip_expires_at?: string | null
  level_detail?: VipLevelRecord | null
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
  accessible_node_level?: number
  traffic_multiplier?: number
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

export interface BenefitCodeRecord {
  id: number
  code: string
  vip_level: number
  duration_days: number
  status: 'unused' | 'used' | string
  is_enabled: boolean
  used_by?: number | null
  used_at?: string | null
  expires_at?: string | null
  created_at: string
}

export interface BenefitCodeListQuery extends PaginationQuery {
  status?: 'unused' | 'used'
  vip_level?: number
}

export interface BenefitCodeGeneratePayload {
  vip_level: number
  duration_days: number
  count: number
  expires_at?: string
}

export interface BenefitCodeRedeemResult {
  code: string
  applied_level: number
  vip_expires_at?: string | null
}

export interface AdminUserRecord {
  id: number
  username: string
  email: string
  role: UserRole
  vip_level: number
  status: string
  traffic_used: number
  traffic_quota: number
  telegram_id?: string | null
  telegram_username?: string | null
  created_at?: string
  updated_at?: string
}

export interface AdminUserListQuery extends PaginationQuery {
  role?: UserRole
  status?: string
  vip_level?: number
  keyword?: string
}

export interface AdminUserListResult extends PaginationResult<AdminUserRecord> {}

export type WebSocketEventType =
  | 'node_status_changed'
  | 'rule_status_changed'
  | 'traffic_alert'
  | 'announcement'
  | 'config_updated'
  | 'ping'
  | 'pong'

export interface WebSocketEventMessage<
  T extends Record<string, unknown> = Record<string, unknown>,
> {
  type: WebSocketEventType | string
  data?: T
  message?: string
  timestamp?: string
}

export interface AppNotification {
  id: string
  type: WebSocketEventType | string
  title: string
  content: string
  created_at: string
  read: boolean
  payload?: Record<string, unknown>
}

export interface Node {
  id: number
  userId: number
  name: string
  status: 'online' | 'offline' | 'maintain'
  host: string
  port: number
  region: string
  isSelfHosted: boolean
  isPublic: boolean
  trafficMultiplier: number
  cpuUsage: number
  memoryUsage: number
  diskUsage: number
  bandwidthIn: number
  bandwidthOut: number
  connections: number
  configVersion: number
  description: string
  lastHeartbeatAt: string | null
  createdAt: string
}

export interface Rule {
  id: number
  userId: number
  name: string
  mode: 'single' | 'tunnel'
  protocol: string
  entryNodeId: number
  exitNodeId: number | null
  entryNode: Node
  exitNode: Node | null
  targetHost: string
  targetPort: number
  listenHost: string
  listenPort: number
  status: 'running' | 'stopped' | 'paused'
  syncStatus: string
  trafficIn: number
  trafficOut: number
  connections: number
  createdAt: string
}

export interface TrafficRecord {
  id: number
  userId: number
  ruleId: number
  nodeId: number
  trafficIn: number
  trafficOut: number
  vipMultiplier: number
  nodeMultiplier: number
  finalMultiplier: number
  calculatedTraffic: number
  hour: string
}

export interface VipLevel {
  id: number
  level: number
  name: string
  description: string
  trafficQuota: number
  maxRules: number
  maxBandwidth: number
  maxSelfHostedEntryNodes: number
  maxSelfHostedExitNodes: number
  accessibleNodeLevel: number
  trafficMultiplier: number
  price: number
  durationDays: number
}

export interface BenefitCode {
  id: number
  code: string
  vipLevel: number
  durationDays: number
  status: 'unused' | 'used'
  isEnabled: boolean
  usedBy: number | null
  usedAt: string | null
  expiresAt: string | null
  createdAt: string
}

export interface Announcement {
  id: number
  title: string
  content: string
  type: 'info' | 'warning' | 'error' | 'success'
  isEnabled: boolean
  startTime: string | null
  endTime: string | null
  createdAt: string
}

export interface AuditLog {
  id: number
  userId: number
  action: string
  resourceType: string
  resourceId: number
  details: string
  ipAddress: string
  userAgent: string
  createdAt: string
}

export interface WsMessage {
  type: string
  data: any
  timestamp: string
}

export interface NodeQuota {
  vipLevel: number
  maxSelfHostedEntryNodes: number
  maxSelfHostedExitNodes: number
  currentEntryNodes: number
  currentExitNodes: number
  remainingEntryNodes: number
  remainingExitNodes: number
}
