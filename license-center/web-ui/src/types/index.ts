export interface LoginRequest {
  username: string
  password: string
}

export interface LoginResponse {
  token: string
  user: AdminUser
}

export interface AdminUser {
  id: number
  username: string
  email: string
  role: string
}

export interface LicensePlan {
  id: number
  name: string
  code: string
  description: string
  is_enabled: boolean
  max_machines: number
  duration_days: number
  min_panel_version: string
  max_panel_version: string
  min_backend_version: string
  max_backend_version: string
  min_frontend_version: string
  max_frontend_version: string
  min_nodeclient_version: string
  max_nodeclient_version: string
  created_at: string
  updated_at: string
}

export interface LicenseKey {
  id: number
  key: string
  plan_id: number
  plan: LicensePlan
  customer: string
  status: 'active' | 'expired' | 'revoked'
  expires_at: string | null
  max_machines: number | null
  note: string
  metadata_json: string
  created_by: number
  created_at: string
  updated_at: string
}

export interface LicenseActivation {
  id: number
  license_id: number
  machine_id: string
  machine_name: string
  ip_address: string
  first_verified_at: string
  last_verified_at: string
  verify_count: number
  is_active: boolean
}

export interface Alert {
  id: number
  type: string
  level: 'info' | 'warning' | 'error' | 'critical'
  title: string
  message: string
  license_id: number | null
  is_read: boolean
  is_sent: boolean
  metadata_json: string
  created_at: string
  updated_at: string
}

export interface WebhookConfig {
  id: number
  name: string
  url: string
  secret: string
  events: string
  is_enabled: boolean
  created_at: string
  updated_at: string
}

export interface LicenseTag {
  id: number
  name: string
  color: string
  created_at: string
  updated_at: string
}

export interface VerifyLog {
  id: number
  license_id: number | null
  license_key: string
  machine_id: string
  action: string
  result: 'success' | 'failed'
  reason: string
  panel_version: string
  backend_version: string
  frontend_version: string
  nodeclient_version: string
  branch: string
  commit: string
  ip_address: string
  user_agent: string
  created_at: string
}

export interface DashboardStats {
  license_total: number
  license_active: number
  license_expired: number
  license_revoked: number
  activation_total: number
  verify_today: number
  verify_week: number
  verify_month: number
  verify_success_today: number
  verify_failed_today: number
  expiring_count: number
}

export interface PaginatedResponse<T> {
  items: T[]
  total: number
  page: number
  page_size: number
}

export interface ApiResponse<T = any> {
  success: boolean
  data?: T
  message: string
  timestamp: string
  error?: {
    code: string
    message: string
  }
}
