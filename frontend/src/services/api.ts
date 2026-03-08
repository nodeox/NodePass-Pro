import axios, {
  type AxiosError,
  type AxiosResponse,
  type InternalAxiosRequestConfig,
} from 'axios'
import { message } from 'antd'

import {
  getStoredToken,
  setAuthToken,
  clearAuthStorage,
  migrateOldStorage,
} from '../utils/secureStorage'

// 重新导出供外部使用
export { setAuthToken, clearAuthStorage, getStoredToken }

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
  ChangeEmailPayload,
  ChangePasswordPayload,
  CreateVipLevelPayload,
  LoginPayload,
  LoginResult,
  PaginationQuery,
  PaginationResult,
  RegisterPayload,
  SendEmailChangeCodePayload,
  SendEmailChangeCodeResult,
  TrafficQuota,
  TrafficRecordItem,
  TrafficRecordsQuery,
  TrafficUsageQuery,
  TrafficUsageSummary,
  UpdateVipLevelPayload,
  User,
  VipMyLevelResult,
  VipLevelRecord,
} from '../types'

// 在模块加载时迁移旧数据
migrateOldStorage()

// 移除未使用的类型定义
// type PersistedAuthShape 已在 secureStorage 中定义

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
  withCredentials: true,
})

// 移除旧的存储函数，使用 secureStorage 模块
// parseAuthStorage, getStoredToken, setAuthToken, clearAuthStorage 已从 secureStorage 导入

const isAuthPath = (url: string): boolean =>
  ['/auth/login', '/auth/register', '/auth/refresh'].some((path) =>
    url.includes(path),
  )

const CSRF_HEADER = 'x-csrf-token'
let csrfToken: string | null = null
let csrfTokenPromise: Promise<string | null> | null = null
let loginRedirecting = false

const isUnsafeMethod = (method?: string): boolean => {
  const normalized = (method ?? 'get').toLowerCase()
  return ['post', 'put', 'patch', 'delete'].includes(normalized)
}

const pickCSRFTokenFromHeaders = (headers: AxiosResponse['headers']): string | null => {
  const token = headers?.[CSRF_HEADER]
  if (typeof token === 'string' && token.trim()) {
    return token.trim()
  }
  if (Array.isArray(token) && token.length > 0) {
    const first = token[0]
    if (typeof first === 'string' && first.trim()) {
      return first.trim()
    }
  }
  return null
}

const syncCSRFToken = (headers?: AxiosResponse['headers']): void => {
  if (!headers) {
    return
  }
  const token = pickCSRFTokenFromHeaders(headers)
  if (token) {
    csrfToken = token
  }
}

const ensureCSRFToken = async (): Promise<string | null> => {
  if (csrfToken) {
    return csrfToken
  }

  const authToken = getStoredToken()
  if (!authToken) {
    return null
  }

  if (csrfTokenPromise) {
    return csrfTokenPromise
  }

  csrfTokenPromise = axios
    .get<ApiSuccessResponse<User>>('/api/v1/auth/me', {
      timeout: 10_000,
      withCredentials: true,
      headers: {
        Authorization: `Bearer ${authToken}`,
      },
    })
    .then((response) => {
      syncCSRFToken(response.headers)
      return csrfToken
    })
    .catch(() => null)
    .finally(() => {
      csrfTokenPromise = null
    })

  return csrfTokenPromise
}

const redirectToLogin = (): void => {
  if (loginRedirecting) {
    return
  }
  if (window.location.pathname === '/login') {
    return
  }
  loginRedirecting = true
  window.location.replace('/login')
}

export const unwrapData = <T>(response: AxiosResponse<ApiSuccessResponse<T>>): T =>
  response.data.data

const normalizeLoginResult = (payload: unknown): LoginResult => {
  const source = (payload ?? {}) as Record<string, unknown>
  const token = String(source.token ?? source.access_token ?? '').trim()
  const user = source.user
  if (!token || !user || typeof user !== 'object') {
    throw new Error('登录响应缺少有效凭证，请稍后重试')
  }
  return {
    token,
    user: user as LoginResult['user'],
  }
}

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

  const csrf = await ensureCSRFToken()
  const headers: Record<string, string> = {
    Authorization: `Bearer ${token}`,
  }
  if (csrf) {
    headers['X-CSRF-Token'] = csrf
  }

  const response = await axios.post<ApiSuccessResponse<{ token: string }>>(
    '/api/v1/auth/refresh',
    {},
    {
      withCredentials: true,
      headers,
    },
  )
  syncCSRFToken(response.headers)

  const nextToken = response.data.data?.token
  if (!nextToken) {
    throw new Error('刷新 Token 失败')
  }

  setAuthToken(nextToken)
  return nextToken
}

apiClient.interceptors.request.use(async (requestConfig) => {
  const token = getStoredToken()
  if (token) {
    requestConfig.headers.Authorization = `Bearer ${token}`
  }

  requestConfig.withCredentials = true
  const requestURL = requestConfig.url ?? ''
  if (token && isUnsafeMethod(requestConfig.method) && !isAuthPath(requestURL)) {
    const csrf = (await ensureCSRFToken()) ?? csrfToken
    if (csrf) {
      requestConfig.headers['X-CSRF-Token'] = csrf
    }
  }

  return requestConfig
})

apiClient.interceptors.response.use(
  (response) => {
    syncCSRFToken(response.headers)
    return response
  },
  async (error: AxiosError<ApiErrorResponse>) => {
    syncCSRFToken(error.response?.headers)
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
      .then(unwrapData)
      .then(normalizeLoginResult),

  login: (payload: LoginPayload) =>
    apiClient
      .post<ApiSuccessResponse<LoginResult>>('/auth/login', payload)
      .then(unwrapData)
      .then(normalizeLoginResult),

  me: () =>
    apiClient.get<ApiSuccessResponse<User>>('/auth/me').then(unwrapData),

  changePassword: (payload: ChangePasswordPayload) =>
    apiClient
      .put<ApiSuccessResponse<null>>('/auth/password', payload)
      .then(unwrapData),

  sendEmailChangeCode: (payload: SendEmailChangeCodePayload) =>
    apiClient
      .post<ApiSuccessResponse<SendEmailChangeCodeResult>>('/auth/email/code', payload)
      .then(unwrapData),

  changeEmail: (payload: ChangeEmailPayload) =>
    apiClient
      .put<ApiSuccessResponse<null>>('/auth/email', payload)
      .then(unwrapData),

  refresh: () =>
    apiClient
      .post<ApiSuccessResponse<{ token: string }>>('/auth/refresh')
      .then(unwrapData),

  revokeAllTokens: () =>
    apiClient
      .post<ApiSuccessResponse<null>>('/auth/revoke-all')
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

  updateQuota: (targetUserID: number, trafficQuota: number) =>
    apiClient
      .put<ApiSuccessResponse<null>>(`/traffic/quota/${targetUserID}`, {
        traffic_quota: trafficQuota,
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

  getUser: (userID: number) =>
    apiClient
      .get<ApiSuccessResponse<User>>(`/users/${userID}`)
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

  updateConfigs: (payload: Array<{ key: string; value: string }>) =>
    apiClient
      .put<ApiSuccessResponse<null>>('/system/config', { items: payload })
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
