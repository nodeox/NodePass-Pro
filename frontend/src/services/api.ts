import axios, {
  type AxiosError,
  type AxiosResponse,
  type InternalAxiosRequestConfig,
} from 'axios'
import { message } from 'antd'

import type {
  AdminUserListQuery,
  AdminUserListResult,
  AnnouncementRecord,
  ApiErrorResponse,
  ApiSuccessResponse,
  AuditLogRecord,
  BenefitCodeGeneratePayload,
  BenefitCodeListQuery,
  BenefitCodeRecord,
  BenefitCodeRedeemResult,
  ChangePasswordPayload,
  CreateNodePairPayload,
  CreateNodePayload,
  CreateNodeResult,
  CreateRulePayload,
  CreateVipLevelPayload,
  LoginPayload,
  LoginResult,
  NodeListQuery,
  NodePairListResult,
  NodePairRecord,
  NodeQuotaInfo,
  NodeRecord,
  PaginationQuery,
  PaginationResult,
  RegisterPayload,
  RuleListQuery,
  RuleRecord,
  TrafficQuota,
  TrafficRecordItem,
  TrafficRecordsQuery,
  TrafficUsageQuery,
  TrafficUsageSummary,
  UpdateNodePairPayload,
  UpdateNodePayload,
  UpdateRulePayload,
  UpdateVipLevelPayload,
  User,
  VipMyLevelResult,
  VipLevelRecord,
} from '../types'

export const AUTH_STORAGE_KEY = 'nodepass-auth'

type PersistedAuthShape = {
  state?: {
    token?: string | null
    user?: User | null
    isAuthenticated?: boolean
  }
  version?: number
}

type RetryableRequestConfig = InternalAxiosRequestConfig & {
  _retry?: boolean
}

type PendingRequest = {
  resolve: (token: string) => void
  reject: (error: unknown) => void
}

const apiClient = axios.create({
  baseURL: '/api/v1',
  timeout: 20_000,
})

const parseAuthStorage = (): PersistedAuthShape | null => {
  const raw = localStorage.getItem(AUTH_STORAGE_KEY)
  if (!raw) {
    return null
  }

  try {
    return JSON.parse(raw) as PersistedAuthShape
  } catch (_error) {
    return null
  }
}

export const getStoredToken = (): string | null => {
  const parsed = parseAuthStorage()
  return parsed?.state?.token ?? null
}

export const setAuthToken = (token: string | null): void => {
  if (!token) {
    localStorage.removeItem(AUTH_STORAGE_KEY)
    return
  }

  const current = parseAuthStorage()
  const next: PersistedAuthShape = {
    version: current?.version ?? 0,
    state: {
      ...(current?.state ?? {}),
      token,
      isAuthenticated: true,
    },
  }
  localStorage.setItem(AUTH_STORAGE_KEY, JSON.stringify(next))
}

export const clearAuthStorage = (): void => {
  localStorage.removeItem(AUTH_STORAGE_KEY)
}

const isAuthPath = (url: string): boolean =>
  ['/auth/login', '/auth/register', '/auth/refresh'].some((path) =>
    url.includes(path),
  )

const redirectToLogin = (): void => {
  if (window.location.pathname !== '/login') {
    window.location.href = '/login'
  }
}

const unwrapData = <T>(response: AxiosResponse<ApiSuccessResponse<T>>): T =>
  response.data.data

let isRefreshing = false
let pendingQueue: PendingRequest[] = []

const flushQueue = (error: unknown, token: string | null): void => {
  pendingQueue.forEach((request) => {
    if (error || !token) {
      request.reject(error)
      return
    }
    request.resolve(token)
  })
  pendingQueue = []
}

const requestRefreshToken = async (): Promise<string> => {
  const token = getStoredToken()
  if (!token) {
    throw new Error('缺少登录凭证')
  }

  const response = await axios.post<ApiSuccessResponse<{ token: string }>>(
    '/api/v1/auth/refresh',
    {},
    {
      headers: {
        Authorization: `Bearer ${token}`,
      },
    },
  )

  const nextToken = response.data.data?.token
  if (!nextToken) {
    throw new Error('刷新 Token 失败')
  }

  setAuthToken(nextToken)
  return nextToken
}

apiClient.interceptors.request.use((requestConfig) => {
  const token = getStoredToken()
  if (token) {
    requestConfig.headers.Authorization = `Bearer ${token}`
  }
  return requestConfig
})

apiClient.interceptors.response.use(
  (response) => response,
  async (error: AxiosError<ApiErrorResponse>) => {
    const status = error.response?.status
    const originalRequest = error.config as RetryableRequestConfig | undefined
    const requestURL = originalRequest?.url ?? ''

    if (
      status !== 401 ||
      !originalRequest ||
      originalRequest._retry ||
      isAuthPath(requestURL)
    ) {
      if (status === 401 && !isAuthPath(requestURL)) {
        clearAuthStorage()
        redirectToLogin()
      }
      return Promise.reject(error)
    }

    originalRequest._retry = true

    if (isRefreshing) {
      return new Promise<string>((resolve, reject) => {
        pendingQueue.push({ resolve, reject })
      }).then((token) => {
        originalRequest.headers.Authorization = `Bearer ${token}`
        return apiClient(originalRequest)
      })
    }

    isRefreshing = true
    try {
      const nextToken = await requestRefreshToken()
      flushQueue(null, nextToken)
      originalRequest.headers.Authorization = `Bearer ${nextToken}`
      return apiClient(originalRequest)
    } catch (refreshError) {
      flushQueue(refreshError, null)
      clearAuthStorage()
      message.error('登录已过期，请重新登录')
      redirectToLogin()
      return Promise.reject(refreshError)
    } finally {
      isRefreshing = false
    }
  },
)

export const authApi = {
  register: (payload: RegisterPayload) =>
    apiClient
      .post<ApiSuccessResponse<LoginResult>>('/auth/register', payload)
      .then(unwrapData),

  login: (payload: LoginPayload) =>
    apiClient
      .post<ApiSuccessResponse<LoginResult>>('/auth/login', payload)
      .then(unwrapData),

  me: () =>
    apiClient.get<ApiSuccessResponse<User>>('/auth/me').then(unwrapData),

  changePassword: (payload: ChangePasswordPayload) =>
    apiClient
      .put<ApiSuccessResponse<null>>('/auth/password', payload)
      .then(unwrapData),

  refresh: () =>
    apiClient
      .post<ApiSuccessResponse<{ token: string }>>('/auth/refresh')
      .then(unwrapData),
}

export const nodeApi = {
  list: (params?: NodeListQuery) =>
    apiClient
      .get<ApiSuccessResponse<PaginationResult<NodeRecord>>>('/nodes', {
        params,
      })
      .then(unwrapData),

  detail: (id: number) =>
    apiClient.get<ApiSuccessResponse<NodeRecord>>(`/nodes/${id}`).then(unwrapData),

  create: (payload: CreateNodePayload) =>
    apiClient
      .post<ApiSuccessResponse<CreateNodeResult>>('/nodes', payload)
      .then(unwrapData),

  update: (id: number, payload: UpdateNodePayload) =>
    apiClient
      .put<ApiSuccessResponse<NodeRecord>>(`/nodes/${id}`, payload)
      .then(unwrapData),

  remove: (id: number) =>
    apiClient.delete<ApiSuccessResponse<null>>(`/nodes/${id}`).then(unwrapData),

  quota: () =>
    apiClient
      .get<ApiSuccessResponse<NodeQuotaInfo>>('/nodes/quota')
      .then(unwrapData),
}

export const nodePairApi = {
  list: () =>
    apiClient
      .get<ApiSuccessResponse<NodePairListResult>>('/node-pairs')
      .then(unwrapData),

  create: (payload: CreateNodePairPayload) =>
    apiClient
      .post<ApiSuccessResponse<NodePairRecord>>('/node-pairs', payload)
      .then(unwrapData),

  update: (id: number, payload: UpdateNodePairPayload) =>
    apiClient
      .put<ApiSuccessResponse<NodePairRecord>>(`/node-pairs/${id}`, payload)
      .then(unwrapData),

  remove: (id: number) =>
    apiClient
      .delete<ApiSuccessResponse<null>>(`/node-pairs/${id}`)
      .then(unwrapData),

  toggle: (id: number) =>
    apiClient
      .put<ApiSuccessResponse<NodePairRecord>>(`/node-pairs/${id}/toggle`)
      .then(unwrapData),
}

export const ruleApi = {
  list: (params?: RuleListQuery) =>
    apiClient
      .get<ApiSuccessResponse<PaginationResult<RuleRecord>>>('/rules', { params })
      .then(unwrapData),

  detail: (id: number) =>
    apiClient.get<ApiSuccessResponse<RuleRecord>>(`/rules/${id}`).then(unwrapData),

  create: (payload: CreateRulePayload) =>
    apiClient.post<ApiSuccessResponse<RuleRecord>>('/rules', payload).then(unwrapData),

  update: (id: number, payload: UpdateRulePayload) =>
    apiClient
      .put<ApiSuccessResponse<RuleRecord>>(`/rules/${id}`, payload)
      .then(unwrapData),

  remove: (id: number) =>
    apiClient.delete<ApiSuccessResponse<null>>(`/rules/${id}`).then(unwrapData),

  start: (id: number) =>
    apiClient.post<ApiSuccessResponse<RuleRecord>>(`/rules/${id}/start`).then(unwrapData),

  stop: (id: number) =>
    apiClient.post<ApiSuccessResponse<RuleRecord>>(`/rules/${id}/stop`).then(unwrapData),

  restart: (id: number) =>
    apiClient
      .post<ApiSuccessResponse<RuleRecord>>(`/rules/${id}/restart`)
      .then(unwrapData),
}

export const trafficApi = {
  quota: () =>
    apiClient.get<ApiSuccessResponse<TrafficQuota>>('/traffic/quota').then(unwrapData),

  usage: (params: TrafficUsageQuery) =>
    apiClient
      .get<ApiSuccessResponse<TrafficUsageSummary>>('/traffic/usage', { params })
      .then(unwrapData),

  records: (params: TrafficRecordsQuery) =>
    apiClient
      .get<ApiSuccessResponse<PaginationResult<TrafficRecordItem>>>(
        '/traffic/records',
        { params },
      )
      .then(unwrapData),

  resetQuota: (targetUserID: number) =>
    apiClient
      .post<ApiSuccessResponse<null>>('/traffic/quota/reset', {
        target_user_id: targetUserID,
      })
      .then(unwrapData),
}

export const vipApi = {
  levels: () =>
    apiClient
      .get<ApiSuccessResponse<{ list: VipLevelRecord[]; total: number }>>('/vip/levels')
      .then(unwrapData),

  myLevel: () =>
    apiClient
      .get<ApiSuccessResponse<VipMyLevelResult>>('/vip/my-level')
      .then(unwrapData),

  createLevel: (payload: CreateVipLevelPayload) =>
    apiClient.post<ApiSuccessResponse<VipLevelRecord>>('/vip/levels', payload).then(unwrapData),

  updateLevel: (id: number, payload: UpdateVipLevelPayload) =>
    apiClient
      .put<ApiSuccessResponse<VipLevelRecord>>(`/vip/levels/${id}`, payload)
      .then(unwrapData),

  upgradeUser: (userID: number, payload: Record<string, unknown>) =>
    apiClient
      .post<ApiSuccessResponse<Record<string, unknown>>>(`/users/${userID}/vip/upgrade`, payload)
      .then(unwrapData),
}

export const userAdminApi = {
  list: (params?: AdminUserListQuery) =>
    apiClient
      .get<ApiSuccessResponse<AdminUserListResult>>('/users', { params })
      .then(unwrapData),

  updateRole: (userID: number, role: 'admin' | 'user') =>
    apiClient
      .put<ApiSuccessResponse<User>>(`/users/${userID}/role`, { role })
      .then(unwrapData),

  updateStatus: (userID: number, status: string) =>
    apiClient
      .put<ApiSuccessResponse<User>>(`/users/${userID}/status`, { status })
      .then(unwrapData),
}

export const benefitCodeApi = {
  list: (params?: BenefitCodeListQuery) =>
    apiClient
      .get<ApiSuccessResponse<PaginationResult<BenefitCodeRecord>>>('/benefit-codes', {
        params,
      })
      .then(unwrapData),

  generate: (payload: BenefitCodeGeneratePayload) =>
    apiClient
      .post<ApiSuccessResponse<{ list: BenefitCodeRecord[]; total: number }>>(
        '/benefit-codes/generate',
        payload,
      )
      .then(unwrapData),

  redeem: (code: string) =>
    apiClient
      .post<ApiSuccessResponse<BenefitCodeRedeemResult>>('/benefit-codes/redeem', {
        code,
      })
      .then(unwrapData),

  batchDelete: (ids: number[]) =>
    apiClient
      .post<ApiSuccessResponse<{ deleted: number }>>('/benefit-codes/batch-delete', { ids })
      .then(unwrapData),
}

export const systemApi = {
  config: () =>
    apiClient
      .get<ApiSuccessResponse<Record<string, string>>>('/system/config')
      .then(unwrapData),

  updateConfig: (payload: { key: string; value: string }) =>
    apiClient
      .put<ApiSuccessResponse<null>>('/system/config', payload)
      .then(unwrapData),

  stats: () =>
    apiClient
      .get<ApiSuccessResponse<Record<string, number>>>('/system/stats')
      .then(unwrapData),
}

export const announcementApi = {
  list: (onlyEnabled = true) =>
    apiClient
      .get<ApiSuccessResponse<{ list: AnnouncementRecord[]; total: number }>>(
        '/announcements',
        { params: { only_enabled: onlyEnabled } },
      )
      .then(unwrapData),

  create: (payload: Record<string, unknown>) =>
    apiClient
      .post<ApiSuccessResponse<AnnouncementRecord>>('/announcements', payload)
      .then(unwrapData),

  update: (id: number, payload: Record<string, unknown>) =>
    apiClient
      .put<ApiSuccessResponse<AnnouncementRecord>>(`/announcements/${id}`, payload)
      .then(unwrapData),

  remove: (id: number) =>
    apiClient
      .delete<ApiSuccessResponse<null>>(`/announcements/${id}`)
      .then(unwrapData),
}

export const auditApi = {
  list: (params?: PaginationQuery & Record<string, unknown>) =>
    apiClient
      .get<ApiSuccessResponse<PaginationResult<AuditLogRecord>>>('/audit-logs', {
        params,
      })
      .then(unwrapData),
}

export const telegramApi = {
  bind: () => apiClient.post<ApiSuccessResponse<Record<string, unknown>>>('/telegram/bind').then(unwrapData),

  unbind: () => apiClient.post<ApiSuccessResponse<null>>('/telegram/unbind').then(unwrapData),

  login: (payload: Record<string, unknown>) =>
    apiClient
      .post<ApiSuccessResponse<LoginResult>>('/telegram/login', payload)
      .then(unwrapData),
}

export const getApiErrorMessage = (error: unknown): string => {
  if (axios.isAxiosError(error)) {
    const axiosError = error as AxiosError<ApiErrorResponse>
    return axiosError.response?.data?.error?.message ?? axiosError.message
  }

  if (error instanceof Error) {
    return error.message
  }

  return '请求失败'
}

export default apiClient
