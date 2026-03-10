import { create } from 'zustand'

import {
  authApi,
  clearAuthStorage,
  setAuthSession,
} from '../services/api'
import type { LoginPayload, RegisterPayload, User } from '../types'
import { getStoredToken, getStoredUser } from '../utils/secureStorage'

interface AuthState {
  token: string | null
  user: User | null
  isAuthenticated: boolean
  authChecked: boolean
  isLoading: boolean
  login: (payload: LoginPayload) => Promise<void>
  register: (payload: RegisterPayload) => Promise<void>
  logout: () => void
  fetchMe: () => Promise<void>
  refreshToken: () => Promise<string | null>
}

const initialToken = getStoredToken()
const initialUser = getStoredUser()
let bootstrapRefreshAttempted = false

export const useAuthStore = create<AuthState>()((set, get) => ({
  token: initialToken,
  user: initialUser,
  isAuthenticated: Boolean(initialToken),
  authChecked: false,
  isLoading: false,

  login: async (payload) => {
    set({ isLoading: true })
    try {
      const result = await authApi.login(payload)
      set({
        token: result.token,
        user: result.user,
        isAuthenticated: true,
        authChecked: true,
      })
      setAuthSession({
        accessToken: result.token,
        expiresIn: result.expiresIn,
        user: result.user,
      })
    } finally {
      set({ isLoading: false })
    }
  },

  register: async (payload) => {
    set({ isLoading: true })
    try {
      await authApi.register(payload)
    } finally {
      set({ isLoading: false })
    }
  },

  logout: () => {
    clearAuthStorage()
    set({
      token: null,
      user: null,
      isAuthenticated: false,
      authChecked: true,
      isLoading: false,
    })
  },

  fetchMe: async () => {
    const token = get().token
    if (!token) {
      if (bootstrapRefreshAttempted) {
        set({ authChecked: true })
        return
      }
      bootstrapRefreshAttempted = true
      set({ isLoading: true })
      try {
        const refreshed = await authApi.refresh()
        const nextToken = refreshed.token
        if (!nextToken) {
          set({ isAuthenticated: false, user: null, authChecked: true })
          return
        }

        setAuthSession({
          accessToken: nextToken,
          expiresIn: refreshed.expiresIn,
          user: refreshed.user,
        })
        set({
          token: nextToken,
          user: refreshed.user,
          isAuthenticated: true,
          authChecked: true,
        })
      } catch {
        set({ isAuthenticated: false, user: null, authChecked: true })
      } finally {
        set({ isLoading: false })
      }
      return
    }

    set({ isLoading: true })
    try {
      const user = await authApi.me()
      set({
        user,
        token,
        isAuthenticated: true,
        authChecked: true,
      })
    } catch {
      get().logout()
    } finally {
      set({ isLoading: false })
    }
  },

  refreshToken: async () => {
    if (!get().token) {
      return null
    }

    try {
      const result = await authApi.refresh()
      const nextToken = result.token
      if (!nextToken) {
        get().logout()
        return null
      }

      set({ token: nextToken, isAuthenticated: true, authChecked: true })
      setAuthSession({
        accessToken: nextToken,
        expiresIn: result.expiresIn,
        user: result.user,
      })
      return nextToken
    } catch {
      get().logout()
      return null
    }
  },
}))
