import axios, { AxiosError } from 'axios'
import { message } from 'antd'
import { useAuthStore } from '@/store/auth'

const api = axios.create({
  baseURL: '/api/v1',
  timeout: 30000,
  headers: {
    'Content-Type': 'application/json',
  },
})

// 请求拦截器
api.interceptors.request.use(
  (config) => {
    const token = useAuthStore.getState().token
    if (token) {
      config.headers.Authorization = `Bearer ${token}`
    }
    return config
  },
  (error) => {
    return Promise.reject(error)
  }
)

// 响应拦截器
api.interceptors.response.use(
  (response) => {
    return response.data
  },
  (error: AxiosError<any>) => {
    if (error.response) {
      const { status, data } = error.response

      if (status === 401) {
        useAuthStore.getState().logout()
        window.location.href = '/login'
        message.error('登录已过期，请重新登录')
      } else if (status === 403) {
        message.error('没有权限访问')
      } else if (status === 429) {
        message.error('请求过于频繁，请稍后再试')
      } else {
        const errorMsg = data?.error?.message || data?.message || '请求失败'
        message.error(errorMsg)
      }
    } else if (error.request) {
      message.error('网络错误，请检查网络连接')
    } else {
      message.error('请求配置错误')
    }

    return Promise.reject(error)
  }
)

export default api
