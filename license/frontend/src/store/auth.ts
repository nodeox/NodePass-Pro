import { create } from 'zustand'
import { persist } from 'zustand/middleware'
import type { CurrentUser } from '../types/api'

interface AuthState {
  token: string | null
  user: CurrentUser | null
  setAuth: (token: string, user: CurrentUser) => void
  clearAuth: () => void
}

export const useAuthStore = create<AuthState>()(
  persist(
    (set) => ({
      token: null,
      user: null,
      setAuth: (token, user) => set({ token, user }),
      clearAuth: () => set({ token: null, user: null })
    }),
    {
      name: 'license-unified-auth'
    }
  )
)
