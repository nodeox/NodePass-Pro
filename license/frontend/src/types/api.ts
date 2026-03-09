export interface ApiSuccess<T> {
  success: true
  data: T
  message: string
  timestamp: number
}

export interface ApiError {
  success: false
  error: {
    code: string
    message: string
  }
  timestamp: number
}

export interface CurrentUser {
  id: number
  username: string
  email: string
}

export interface LoginResponse {
  token: string
  user: CurrentUser
}

export type PlanStatus = 'active' | 'disabled'

export interface LicensePlan {
  id: number
  code: string
  name: string
  description: string
  max_machines: number
  duration_days: number
  status: PlanStatus | string
  license_count?: number
  active_license_count?: number
  activation_count?: number
  created_at: string
  updated_at: string
}

export interface LicenseActivation {
  id: number
  license_id: number
  machine_id: string
  hostname: string
  ip_address: string
  last_seen_at: string
  created_at: string
  updated_at: string
}

export interface License {
  id: number
  key: string
  plan_id: number
  plan: LicensePlan
  customer: string
  status: string
  expires_at?: string
  max_machines?: number
  metadata_json: string
  note: string
  created_by: number
  activations?: LicenseActivation[]
  created_at: string
  updated_at: string
}

export interface LicenseListResult {
  items: License[]
  total: number
  page: number
  page_size: number
}

export interface LicenseListParams {
  page: number
  page_size: number
  customer?: string
  status?: string
  plan_id?: number
  expire_from?: string
  expire_to?: string
  sort_by?: 'created_at' | 'expires_at' | 'status'
  sort_order?: 'asc' | 'desc'
}

export interface UpdateLicensePayload {
  key?: string
  plan_id?: number
  customer?: string
  status?: 'active' | 'revoked' | 'expired'
  expires_at?: string
  clear_expires_at?: boolean
  max_machines?: number
  clear_max_machines?: boolean
  metadata_json?: string
  note?: string
}

export interface BatchLicenseUpdateFields {
  plan_id?: number
  status?: 'active' | 'revoked' | 'expired'
  expires_at?: string | null
  max_machines?: number
  metadata_json?: string
  note?: string
}

export interface BatchActionResult {
  deleted_count?: number
  updated_count?: number
  revoked_count?: number
  restored_count?: number
}

export interface ProductRelease {
  id: number
  product: string
  version: string
  channel: string
  is_mandatory: boolean
  release_notes: string
  file_name?: string
  file_size?: number
  file_sha256?: string
  published_at: string
  is_active: boolean
  deleted_at?: string | null
  created_at: string
  updated_at: string
}

export interface VersionPolicy {
  id: number
  product: string
  channel: string
  min_supported_version: string
  recommended_version: string
  message: string
  is_active: boolean
  created_at: string
  updated_at: string
}

export interface VersionSyncConfig {
  id: number
  enabled: boolean
  auto_sync: boolean
  interval_minutes: number
  github_owner: string
  github_repo: string
  has_github_token: boolean
  product: string
  channel: string
  include_prerelease: boolean
  api_base_url: string
  last_sync_at?: string
  last_sync_status?: string
  last_sync_message?: string
  last_synced_count: number
  created_at: string
  updated_at: string
}

export interface VersionSyncResult {
  product: string
  fetched_count: number
  imported_count: number
  skipped_count: number
  synced_at: string
  status: string
  message: string
}

export interface VerifyLog {
  id: number
  license_id?: number
  license_key: string
  machine_id: string
  product: string
  client_version: string
  verified: boolean
  status: string
  reason: string
  client_ip: string
  user_agent: string
  created_at: string
}

export interface VerifyLogListResult {
  items: VerifyLog[]
  total: number
  page: number
  page_size: number
}

export interface DashboardStats {
  total_licenses: number
  active_licenses: number
  expiring_soon_30_days: number
  total_activations: number
  verify_requests_24h: number
  verify_success_rate_24h: number
}
