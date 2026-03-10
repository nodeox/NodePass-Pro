/**
 * 安全的 Token 存储模块
 *
 * 安全特性：
 * 1. 默认使用内存存储 access token（不落盘）
 * 2. 支持 session/local 兼容模式
 * 3. 自动过期检查
 * 4. 统一清理旧版持久化键
 */

import type { User } from '../types'

// 存储键名
const AUTH_STORAGE_KEY = 'nodepass-auth'
const TOKEN_EXPIRY_KEY = 'nodepass-token-expiry'
const AUTH_ZUSTAND_KEY = 'nodepass-auth-zustand'

// 存储模式
type StorageMode = 'memory' | 'session' | 'local'

// 认证数据结构
interface AuthData {
  token: string | null
  user: User | null
  isAuthenticated: boolean
  expiresAt?: number // Token 过期时间戳
}

interface PersistedAuthShape {
  state?: AuthData
  version?: number
}

// 内存存储（最安全，但刷新页面会丢失）
let memoryStorage: AuthData = {
  token: null,
  user: null,
  isAuthenticated: false,
}

// 当前存储模式（默认使用 memory，不落盘 access token）
let currentStorageMode: StorageMode = 'memory'

/**
 * 设置存储模式
 * @param mode - 存储模式
 * - 'memory': 仅内存存储（默认，最安全）
 * - 'session': sessionStorage（兼容模式）
 * - 'local': localStorage（兼容模式，不建议）
 */
export const setStorageMode = (mode: StorageMode): void => {
  currentStorageMode = mode
}

/**
 * 获取当前存储模式
 */
export const getStorageMode = (): StorageMode => {
  return currentStorageMode
}

/**
 * 获取存储对象
 */
const getStorage = (): Storage | null => {
  if (currentStorageMode === 'memory') {
    return null
  }

  try {
    const storage = currentStorageMode === 'session' ? sessionStorage : localStorage
    // 测试存储是否可用
    const testKey = '__storage_test__'
    storage.setItem(testKey, 'test')
    storage.removeItem(testKey)
    return storage
  } catch {
    // 如果存储不可用（如隐私模式），降级到内存存储
    console.warn('Storage not available, falling back to memory storage')
    currentStorageMode = 'memory'
    return null
  }
}

/**
 * 注意：默认内存模式不落盘；session/local 仅用于兼容或调试场景
 */

/**
 * 解析存储的认证数据
 */
const parseAuthStorage = (): PersistedAuthShape | null => {
  const storage = getStorage()

  if (!storage) {
    // 内存模式
    return {
      state: memoryStorage,
      version: 1,
    }
  }

  const raw = storage.getItem(AUTH_STORAGE_KEY)
  if (!raw) {
    return null
  }

  try {
    // 直接解析 JSON，不再使用加密
    return JSON.parse(raw) as PersistedAuthShape
  } catch {
    return null
  }
}

/**
 * 检查 Token 是否过期
 */
const isTokenExpired = (expiresAt?: number): boolean => {
  if (!expiresAt) {
    return false
  }
  return Date.now() > expiresAt
}

/**
 * 获取存储的 Token
 */
export const getStoredToken = (): string | null => {
  const parsed = parseAuthStorage()
  if (!parsed?.state) {
    return null
  }

  // 检查是否过期
  if (isTokenExpired(parsed.state.expiresAt)) {
    // Access Token 过期后返回 null，由拦截器通过 HttpOnly Cookie 刷新
    return null
  }

  return parsed.state.token ?? null
}

/**
 * 获取存储的 Refresh Token
 */
export const getStoredRefreshToken = (): string | null => {
  return null
}

/**
 * 获取存储的用户信息
 */
export const getStoredUser = (): User | null => {
  const parsed = parseAuthStorage()
  return parsed?.state?.user ?? null
}

/**
 * 获取认证状态
 */
export const getAuthState = (): AuthData => {
  const parsed = parseAuthStorage()
  return parsed?.state ?? {
    token: null,
    user: null,
    isAuthenticated: false,
  }
}

/**
 * 设置认证 Token
 * @param token - JWT Token
 * @param expiresIn - Token 有效期（秒），默认 7 天
 */
export const setAuthToken = (token: string | null, expiresIn: number = 7 * 24 * 60 * 60): void => {
  if (!token) {
    clearAuthStorage()
    return
  }

  const current = parseAuthStorage()
  const expiresAt = Date.now() + expiresIn * 1000

  const next: PersistedAuthShape = {
    version: current?.version ?? 1,
    state: {
      token,
      user: current?.state?.user ?? null,
      isAuthenticated: true,
      expiresAt,
    },
  }

  const storage = getStorage()
  if (!storage) {
    // 内存模式
    memoryStorage = next.state!
    return
  }

  try {
    // 直接存储 JSON，不再加密
    storage.setItem(AUTH_STORAGE_KEY, JSON.stringify(next))
  } catch (error) {
    console.error('Failed to store auth token:', error)
    // 降级到内存存储
    memoryStorage = next.state!
  }
}

/**
 * 设置 Access/Refresh Token 会话
 */
export const setAuthSession = (params: {
  accessToken: string
  expiresIn?: number
  user?: User | null
}): void => {
  const accessToken = String(params.accessToken ?? '').trim()
  if (!accessToken) {
    clearAuthStorage()
    return
  }

  const current = parseAuthStorage()
  const expiresIn = params.expiresIn && params.expiresIn > 0 ? params.expiresIn : 7 * 24 * 60 * 60
  const expiresAt = Date.now() + expiresIn * 1000
  const next: PersistedAuthShape = {
    version: current?.version ?? 1,
    state: {
      token: accessToken,
      user: params.user !== undefined ? (params.user ?? null) : (current?.state?.user ?? null),
      isAuthenticated: true,
      expiresAt,
    },
  }

  const storage = getStorage()
  if (!storage) {
    memoryStorage = next.state!
    return
  }

  try {
    storage.setItem(AUTH_STORAGE_KEY, JSON.stringify(next))
  } catch (error) {
    console.error('Failed to store auth session:', error)
    memoryStorage = next.state!
  }
}

/**
 * 设置用户信息
 */
export const setUserInfo = (user: User | null): void => {
  const current = parseAuthStorage()
  const next: PersistedAuthShape = {
    version: current?.version ?? 1,
    state: {
      token: current?.state?.token ?? null,
      user,
      isAuthenticated: !!user,
      expiresAt: current?.state?.expiresAt,
    },
  }

  const storage = getStorage()
  if (!storage) {
    memoryStorage = next.state!
    return
  }

  try {
    // 直接存储 JSON
    storage.setItem(AUTH_STORAGE_KEY, JSON.stringify(next))
  } catch (error) {
    console.error('Failed to store user info:', error)
    memoryStorage = next.state!
  }
}

/**
 * 清除认证存储
 */
export const clearAuthStorage = (): void => {
  // 清除内存
  memoryStorage = {
    token: null,
    user: null,
    isAuthenticated: false,
  }

  // 清除存储
  const storage = getStorage()
  if (storage) {
    storage.removeItem(AUTH_STORAGE_KEY)
    storage.removeItem(TOKEN_EXPIRY_KEY)
    storage.removeItem(AUTH_ZUSTAND_KEY)
  }

  // 同时清除 sessionStorage/localStorage（兼容旧版本与不同持久化策略）
  try {
    sessionStorage.removeItem(AUTH_STORAGE_KEY)
    sessionStorage.removeItem(TOKEN_EXPIRY_KEY)
    sessionStorage.removeItem(AUTH_ZUSTAND_KEY)
  } catch {
    // 忽略错误
  }

  try {
    localStorage.removeItem('nodepass-auth')
    localStorage.removeItem(TOKEN_EXPIRY_KEY)
    localStorage.removeItem(AUTH_ZUSTAND_KEY)
  } catch {
    // 忽略错误
  }
}

/**
 * 迁移旧的 localStorage 数据到新的存储方式
 */
export const migrateOldStorage = (): void => {
  try {
    const oldData = localStorage.getItem('nodepass-auth')
    if (!oldData) {
      return
    }

    const parsed = JSON.parse(oldData) as PersistedAuthShape
    if (parsed?.state?.token) {
      // 迁移到新的存储方式
      setAuthSession({
        accessToken: parsed.state.token,
        user: parsed.state.user ?? null,
      })

      // 清除旧数据
      localStorage.removeItem('nodepass-auth')
      console.info('Successfully migrated auth data from localStorage to sessionStorage')
    }
  } catch (error) {
    console.error('Failed to migrate old storage:', error)
  }
}

// 导出常量供外部使用
export const AUTH_STORAGE_KEY_EXPORT = AUTH_STORAGE_KEY
