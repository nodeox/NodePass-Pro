import api from '@/utils/request'
import type {
  LoginRequest,
  LoginResponse,
  LicensePlan,
  LicenseKey,
  LicenseActivation,
  Alert,
  WebhookConfig,
  LicenseTag,
  VerifyLog,
  DashboardStats,
  PaginatedResponse,
  ApiResponse,
} from '@/types'

// 认证相关
export const authApi = {
  login: (data: LoginRequest) =>
    api.post<any, ApiResponse<LoginResponse>>('/auth/login', data),

  getMe: () =>
    api.get<any, ApiResponse>('/auth/me'),
}

// 仪表盘
export const dashboardApi = {
  getStats: () =>
    api.get<any, ApiResponse<DashboardStats>>('/dashboard'),

  getVerifyTrend: (days: number = 7) =>
    api.get<any, ApiResponse>(`/verify-trend?days=${days}`),

  getTopCustomers: (limit: number = 10) =>
    api.get<any, ApiResponse>(`/top-customers?limit=${limit}`),
}

// 套餐管理
export const planApi = {
  list: () =>
    api.get<any, ApiResponse<LicensePlan[]>>('/plans'),

  create: (data: Partial<LicensePlan>) =>
    api.post<any, ApiResponse<LicensePlan>>('/plans', data),

  update: (id: number, data: Partial<LicensePlan>) =>
    api.put<any, ApiResponse<LicensePlan>>(`/plans/${id}`, data),

  delete: (id: number) =>
    api.delete<any, ApiResponse>(`/plans/${id}`),
}

// 授权码管理
export const licenseApi = {
  list: (params: {
    status?: string
    customer?: string
    plan_id?: number
    page?: number
    page_size?: number
  }) =>
    api.get<any, ApiResponse<PaginatedResponse<LicenseKey>>>('/licenses', { params }),

  get: (id: number) =>
    api.get<any, ApiResponse<LicenseKey>>(`/licenses/${id}`),

  generate: (data: {
    plan_id: number
    customer: string
    count: number
    expires_at?: string
    max_machines?: number
    note?: string
    prefix?: string
  }) =>
    api.post<any, ApiResponse<LicenseKey[]>>('/licenses/generate', data),

  update: (id: number, data: Partial<LicenseKey>) =>
    api.put<any, ApiResponse<LicenseKey>>(`/licenses/${id}`, data),

  delete: (id: number) =>
    api.delete<any, ApiResponse>(`/licenses/${id}`),

  revoke: (id: number) =>
    api.post<any, ApiResponse>(`/licenses/${id}/revoke`),

  restore: (id: number) =>
    api.post<any, ApiResponse>(`/licenses/${id}/restore`),

  transfer: (id: number, data: { to_customer: string; reason?: string }) =>
    api.post<any, ApiResponse>(`/licenses/${id}/transfer`, data),

  getActivations: (id: number) =>
    api.get<any, ApiResponse<LicenseActivation[]>>(`/licenses/${id}/activations`),

  unbindActivation: (licenseId: number, activationId: number) =>
    api.delete<any, ApiResponse>(`/licenses/${licenseId}/activations/${activationId}`),

  // 批量操作
  batchRevoke: (license_ids: number[]) =>
    api.post<any, ApiResponse>('/licenses/batch/revoke', { license_ids }),

  batchRestore: (license_ids: number[]) =>
    api.post<any, ApiResponse>('/licenses/batch/restore', { license_ids }),

  batchDelete: (license_ids: number[]) =>
    api.post<any, ApiResponse>('/licenses/batch/delete', { license_ids }),

  // 标签
  getTags: (id: number) =>
    api.get<any, ApiResponse<LicenseTag[]>>(`/licenses/${id}/tags`),

  addTags: (id: number, tag_ids: number[]) =>
    api.post<any, ApiResponse>(`/licenses/${id}/tags`, { tag_ids }),

  removeTags: (id: number, tag_ids: number[]) =>
    api.delete<any, ApiResponse>(`/licenses/${id}/tags`, { data: { tag_ids } }),
}

// 告警管理
export const alertApi = {
  list: (params: {
    is_read?: boolean
    level?: string
    page?: number
    page_size?: number
  }) =>
    api.get<any, ApiResponse<PaginatedResponse<Alert>>>('/alerts', { params }),

  markRead: (id: number) =>
    api.post<any, ApiResponse>(`/alerts/${id}/read`),

  markAllRead: () =>
    api.post<any, ApiResponse>('/alerts/read-all'),

  delete: (id: number) =>
    api.delete<any, ApiResponse>(`/alerts/${id}`),

  getStats: () =>
    api.get<any, ApiResponse>('/alert-stats'),
}

// Webhook 管理
export const webhookApi = {
  list: () =>
    api.get<any, ApiResponse<WebhookConfig[]>>('/webhooks'),

  create: (data: {
    name: string
    url: string
    secret?: string
    events: string[]
    is_enabled: boolean
  }) =>
    api.post<any, ApiResponse<WebhookConfig>>('/webhooks', data),

  update: (id: number, data: Partial<WebhookConfig>) =>
    api.put<any, ApiResponse>(`/webhooks/${id}`, data),

  delete: (id: number) =>
    api.delete<any, ApiResponse>(`/webhooks/${id}`),

  getLogs: (params: {
    webhook_id?: number
    page?: number
    page_size?: number
  }) =>
    api.get<any, ApiResponse>('/webhook-logs', { params }),
}

// 标签管理
export const tagApi = {
  list: () =>
    api.get<any, ApiResponse<LicenseTag[]>>('/tags'),

  create: (data: { name: string; color?: string }) =>
    api.post<any, ApiResponse<LicenseTag>>('/tags', data),

  update: (id: number, data: { name?: string; color?: string }) =>
    api.put<any, ApiResponse>(`/tags/${id}`, data),

  delete: (id: number) =>
    api.delete<any, ApiResponse>(`/tags/${id}`),
}

// 日志查询
export const logApi = {
  listVerifyLogs: (params: {
    page?: number
    page_size?: number
  }) =>
    api.get<any, ApiResponse<PaginatedResponse<VerifyLog>>>('/verify-logs', { params }),
}

// 统计
export const statsApi = {
  get: () =>
    api.get<any, ApiResponse>('/stats'),
}

// 版本管理
export const versionApi = {
  getSystemInfo: () =>
    api.get<any, ApiResponse>('/versions/system'),

  getComponentVersion: (component: string) =>
    api.get<any, ApiResponse>(`/versions/components/${component}`),

  updateComponentVersion: (data: {
    component: string
    version: string
    build_time?: string
    git_commit?: string
    git_branch?: string
    description?: string
  }) =>
    api.post<any, ApiResponse>('/versions/components', data),

  getComponentHistory: (component: string, limit?: number) =>
    api.get<any, ApiResponse>(`/versions/components/${component}/history`, {
      params: { limit },
    }),

  checkCompatibility: (version: string) =>
    api.get<any, ApiResponse>(`/versions/compatibility/${version}`),

  listCompatibilityConfigs: () =>
    api.get<any, ApiResponse>('/versions/compatibility'),

  createCompatibilityConfig: (data: {
    backend_version: string
    min_frontend_version: string
    min_node_client_version: string
    min_license_center_version: string
    description?: string
  }) =>
    api.post<any, ApiResponse>('/versions/compatibility', data),
}
