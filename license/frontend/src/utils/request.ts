import axios, { AxiosError } from 'axios'
import type { ApiError, ApiSuccess } from '../types/api'

const request = axios.create({
  baseURL: import.meta.env.VITE_API_BASE ?? '/api/v1',
  timeout: 15000
})

request.interceptors.request.use((config) => {
  const storage = window.localStorage.getItem('license-unified-auth')
  if (storage) {
    const parsed = JSON.parse(storage) as { state?: { token?: string } }
    const token = parsed?.state?.token
    if (token) {
      config.headers.Authorization = `Bearer ${token}`
    }
  }
  return config
})

export function unwrap<T>(payload: ApiSuccess<T>): T {
  return payload.data
}

export function extractErrorMessage(error: unknown): string {
  if (axios.isAxiosError(error)) {
    const axiosError = error as AxiosError<ApiError>
    if (axiosError.response?.data?.error?.message) {
      return axiosError.response.data.error.message
    }
    if (axiosError.message) {
      return axiosError.message
    }
  }
  return '请求失败'
}

export default request
