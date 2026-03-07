import { create } from 'zustand'
import { persist, createJSONStorage } from 'zustand/middleware'

import {
  authApi,
  clearAuthStorage,
  setAuthToken,
} from '../services/api'
import type { LoginPayload, RegisterPayload, User } from '../types'

const AUTH_STORE_PERSIST_KEY = 'nodepass-auth-zustand'

interface AuthState {
  token: string | null
  user: User | null
  isAuthenticated: boolean
  isLoading: boolean
  login: (payload: LoginPayload) => Promise<void>
  register: (payload: RegisterPayload) => Promise<void>
  logout: () => void
  fetchMe: () => Promise<void>
  refreshToken: () => Promise<string | null>
}

export const useAuthStore = create<AuthState>()(
  persist(
    (set, get) => ({
      token: null,
      user: null,
      isAuthenticated: false,
      isLoading: false,

      login: async (payload) => {
        set({ isLoading: true })
        try {
          const result = await authApi.login(payload)
          set({
            token: result.token,
            user: result.user,
            isAuthenticated: true,
          })
          setAuthToken(result.token)
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
          isLoading: false,
        })
      },

      fetchMe: async () => {
        const token = get().token
        if (!token) {
          set({ isAuthenticated: false, user: null })
          return
        }

        set({ isLoading: true })
        try {
          const user = await authApi.me()
          set({
            user,
            token,
            isAuthenticated: true,
          })
        } catch (_error) {
          get().logout()
        } finally {
          set({ isLoading: false })
        }
      },

      refreshToken: async () => {
        const token = get().token
        if (!token) {
          return null
        }

        try {
          const result = await authApi.refresh()
          const nextToken = result.token
          if (!nextToken) {
            get().logout()
            return null
          }

          set({ token: nextToken, isAuthenticated: true })
          setAuthToken(nextToken)
          return nextToken
        } catch (_error) {
          get().logout()
          return null
        }
      },
    }),
    {
      name: AUTH_STORE_PERSIST_KEY,
      // 使用 sessionStorage 代替 localStorage（更安全）
      storage: createJSONStorage(() => sessionStorage),
      partialize: (state) => ({
        token: state.token,
        user: state.user,
        isAuthenticated: state.isAuthenticated,
      }),
    },
  ),
)
