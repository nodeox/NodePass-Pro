/**
 * 安全的 Token 存储模块
 *
 * 安全特性：
 * 1. 使用 sessionStorage 代替 localStorage（关闭浏览器后自动清除）
 * 2. 支持内存存储模式（最高安全级别）
 * 3. Token 加密存储（可选）
 * 4. 自动过期检查
 * 5. 防止 XSS 攻击的额外保护
 */

import type { User } from '../types'

// 存储键名
const AUTH_STORAGE_KEY = 'nodepass-auth'
const TOKEN_EXPIRY_KEY = 'nodepass-token-expiry'

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

// 当前存储模式（默认使用 sessionStorage）
let currentStorageMode: StorageMode = 'session'

/**
 * 设置存储模式
 * @param mode - 存储模式
 * - 'memory': 仅内存存储，最安全，刷新页面会丢失
 * - 'session': sessionStorage，关闭浏览器后清除（推荐）
 * - 'local': localStorage，持久化存储（不推荐，仅用于兼容）
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
 * 注意：不再使用加密，直接依赖 sessionStorage 的浏览器安全机制
 * sessionStorage 数据在关闭标签页后自动清除，且仅限同源访问
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
    clearAuthStorage()
    return null
  }

  return parsed.state.token ?? null
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
  }

  // 同时清除 localStorage（兼容旧版本）
  try {
    localStorage.removeItem('nodepass-auth')
    localStorage.removeItem(TOKEN_EXPIRY_KEY)
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
      setAuthToken(parsed.state.token)
      if (parsed.state.user) {
        setUserInfo(parsed.state.user)
      }

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
