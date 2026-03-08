import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest'
import axios from 'axios'
import MockAdapter from 'axios-mock-adapter'

import apiClient, {
  clearAuthStorage,
  getStoredRefreshToken,
  getStoredToken,
  setAuthSession,
  setAuthToken,
} from '../services/api'

describe('apiClient', () => {
  let apiMock: MockAdapter
  let axiosMock: MockAdapter

  beforeEach(() => {
    apiMock = new MockAdapter(apiClient)
    axiosMock = new MockAdapter(axios)
    clearAuthStorage()
    sessionStorage.clear()
    localStorage.clear()
  })

  afterEach(() => {
    apiMock.restore()
    axiosMock.restore()
    clearAuthStorage()
    vi.restoreAllMocks()
  })

  it('请求自动带 Authorization 头', async () => {
    const token = 'access-token-1'
    setAuthToken(token)

    apiMock.onGet('/test').reply((config) => {
      expect(config.headers?.Authorization).toBe(`Bearer ${token}`)
      return [200, { success: true, data: { ok: true } }]
    })

    const response = await apiClient.get('/test')
    expect(response.status).toBe(200)
  })

  it('401 时使用 /auth/refresh/v2 刷新并重试原请求', async () => {
    setAuthSession({
      accessToken: 'access-old',
      refreshToken: 'refresh-old',
      user: {
        id: 1,
        username: 'tester',
        email: 'tester@example.com',
        role: 'user',
        status: 'active',
        traffic_quota: 0,
        traffic_used: 0,
        traffic_reset_at: null,
        created_at: '2026-01-01T00:00:00Z',
        updated_at: '2026-01-01T00:00:00Z',
      },
    })

    apiMock.onGet('/protected').replyOnce(401, {
      success: false,
      message: 'token expired',
    })

    axiosMock.onPost('/api/v1/auth/refresh/v2').reply(200, {
      success: true,
      data: {
        token: 'access-new',
        refresh_token: 'refresh-new',
        user: {
          id: 1,
          username: 'tester',
          email: 'tester@example.com',
          role: 'user',
          status: 'active',
          traffic_quota: 0,
          traffic_used: 0,
          traffic_reset_at: null,
          created_at: '2026-01-01T00:00:00Z',
          updated_at: '2026-01-01T00:00:00Z',
        },
      },
    })

    apiMock.onGet('/protected').reply(200, {
      success: true,
      data: {
        ok: true,
      },
    })

    const response = await apiClient.get('/protected')
    expect(response.status).toBe(200)
    expect(getStoredToken()).toBe('access-new')
    expect(getStoredRefreshToken()).toBe('refresh-new')
  })

  it('刷新失败时清空凭证并跳转登录页', async () => {
    window.history.replaceState({}, '', '/login')

    setAuthSession({
      accessToken: 'access-old',
      refreshToken: 'refresh-old',
    })

    apiMock.onGet('/protected').reply(401, {
      success: false,
      message: 'token expired',
    })

    axiosMock.onPost('/api/v1/auth/refresh/v2').reply(401, {
      success: false,
      message: 'refresh expired',
    })

    await expect(apiClient.get('/protected')).rejects.toBeTruthy()

    expect(getStoredToken()).toBeNull()
    expect(getStoredRefreshToken()).toBeNull()
  })
})
