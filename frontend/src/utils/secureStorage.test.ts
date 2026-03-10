import { describe, it, expect, beforeEach } from 'vitest'

import {
  AUTH_STORAGE_KEY_EXPORT,
  clearAuthStorage,
  getStoredRefreshToken,
  getStoredToken,
  migrateOldStorage,
  setAuthSession,
  setStorageMode,
} from '../utils/secureStorage'

describe('secureStorage', () => {
  beforeEach(() => {
    setStorageMode('session')
    clearAuthStorage()
    sessionStorage.clear()
    localStorage.clear()
  })

  it('默认使用 sessionStorage 保存 nodepass-auth', () => {
    setAuthSession({
      accessToken: 'access-1',
    })

    const raw = sessionStorage.getItem(AUTH_STORAGE_KEY_EXPORT)
    expect(raw).not.toBeNull()
    expect(getStoredToken()).toBe('access-1')
    expect(getStoredRefreshToken()).toBeNull()
  })

  it('access token 过期时返回 null，refresh token 始终不暴露给前端', () => {
    sessionStorage.setItem(
      AUTH_STORAGE_KEY_EXPORT,
      JSON.stringify({
        version: 1,
        state: {
          token: 'access-expired',
          refreshToken: 'refresh-keep',
          user: null,
          isAuthenticated: true,
          expiresAt: Date.now() - 1_000,
        },
      }),
    )

    expect(getStoredToken()).toBeNull()
    expect(getStoredRefreshToken()).toBeNull()
  })

  it('migrateOldStorage 会把 localStorage 旧数据迁移到 sessionStorage', () => {
    localStorage.setItem(
      AUTH_STORAGE_KEY_EXPORT,
      JSON.stringify({
        version: 1,
        state: {
          token: 'access-old',
          refreshToken: 'refresh-old',
          user: null,
          isAuthenticated: true,
        },
      }),
    )

    migrateOldStorage()

    expect(localStorage.getItem(AUTH_STORAGE_KEY_EXPORT)).toBeNull()
    expect(sessionStorage.getItem(AUTH_STORAGE_KEY_EXPORT)).not.toBeNull()
    expect(getStoredToken()).toBe('access-old')
    expect(getStoredRefreshToken()).toBeNull()
  })

  it('clearAuthStorage 会清理 session/local 与 zustand key', () => {
    sessionStorage.setItem(AUTH_STORAGE_KEY_EXPORT, '{}')
    sessionStorage.setItem('nodepass-auth-zustand', '{}')
    localStorage.setItem(AUTH_STORAGE_KEY_EXPORT, '{}')
    localStorage.setItem('nodepass-auth-zustand', '{}')

    clearAuthStorage()

    expect(sessionStorage.getItem(AUTH_STORAGE_KEY_EXPORT)).toBeNull()
    expect(sessionStorage.getItem('nodepass-auth-zustand')).toBeNull()
    expect(localStorage.getItem(AUTH_STORAGE_KEY_EXPORT)).toBeNull()
    expect(localStorage.getItem('nodepass-auth-zustand')).toBeNull()
    expect(getStoredToken()).toBeNull()
    expect(getStoredRefreshToken()).toBeNull()
  })
})
