import { Spin, message } from 'antd'
import { useEffect } from 'react'
import { useNavigate, useSearchParams } from 'react-router-dom'

import { setAuthSession, telegramApi } from '../../services/api'
import { useAuthStore } from '../../store/auth'
import { getErrorMessage } from '../../utils/error'
import { getHomePathByRole } from '../../utils/route'

const resolveSafeRedirectPath = (target: string): string | null => {
  const raw = target.trim()
  if (!raw) {
    return null
  }
  try {
    const parsed = new URL(raw, window.location.origin)
    if (parsed.origin !== window.location.origin) {
      return null
    }
    if (!parsed.pathname.startsWith('/')) {
      return null
    }
    return `${parsed.pathname}${parsed.search}${parsed.hash}`
  } catch {
    return null
  }
}

const TelegramCallback = () => {
  const [searchParams] = useSearchParams()
  const navigate = useNavigate()

  useEffect(() => {
    const ticket = (searchParams.get('ticket') ?? '').trim()
    if (!ticket) {
      message.error('缺少 SSO 票据，请重新获取登录链接')
      navigate('/login', { replace: true })
      return
    }

    const run = async () => {
      try {
        const result = await telegramApi.ssoLogin(ticket)
        useAuthStore.setState({
          token: result.token,
          user: result.user,
          isAuthenticated: true,
          isLoading: false,
        })
        setAuthSession({
          accessToken: result.token,
          expiresIn: result.expiresIn,
          user: result.user,
        })
        const safeRedirectPath = resolveSafeRedirectPath(result.redirect_uri ?? '')
        if (safeRedirectPath) {
          navigate(safeRedirectPath, { replace: true })
          return
        }
        navigate(getHomePathByRole(result.user?.role), { replace: true })
      } catch (error) {
        message.error(getErrorMessage(error, 'Telegram SSO 登录失败'))
        navigate('/login', { replace: true })
      }
    }

    void run()
  }, [navigate, searchParams])

  return (
    <div className="flex min-h-screen items-center justify-center bg-slate-100">
      <Spin size="large" tip="正在完成 Telegram 单点登录..." />
    </div>
  )
}

export default TelegramCallback
