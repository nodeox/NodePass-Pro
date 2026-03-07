import { Button, Card, Checkbox, Divider, Form, Input, Typography, message } from 'antd'
import { useEffect, useMemo, useRef } from 'react'
import { Link, useNavigate } from 'react-router-dom'

import BrandLogo from '../../components/common/BrandLogo'
import { usePageTitle } from '../../hooks/usePageTitle'
import { AUTH_STORAGE_KEY, setAuthToken, telegramApi } from '../../services/api'
import { useAuthStore } from '../../store/auth'
import { getErrorMessage } from '../../utils/error'
import type { User } from '../../types'

type LoginFormValues = {
  email: string
  password: string
  remember: boolean
}

type TelegramWidgetUser = Record<string, unknown>

type TelegramLoginResult = {
  token: string
  user: User
}

declare global {
  interface Window {
    [key: string]: ((user: TelegramWidgetUser) => void) | unknown
  }
}

const Login = () => {
  usePageTitle('登录')

  const navigate = useNavigate()
  const widgetContainerRef = useRef<HTMLDivElement | null>(null)

  const login = useAuthStore((state) => state.login)
  const isLoading = useAuthStore((state) => state.isLoading)
  const isAuthenticated = useAuthStore((state) => state.isAuthenticated)

  const telegramBotUsername = useMemo(
    () => import.meta.env.VITE_TELEGRAM_BOT_USERNAME?.trim() ?? '',
    [],
  )

  useEffect(() => {
    if (isAuthenticated) {
      navigate('/dashboard', { replace: true })
    }
  }, [isAuthenticated, navigate])

  const handleSubmit = async (values: LoginFormValues) => {
    try {
      await login({
        account: values.email,
        password: values.password,
      })

      if (!values.remember) {
        localStorage.removeItem(AUTH_STORAGE_KEY)
      }

      message.success('登录成功')
      navigate('/dashboard', { replace: true })
    } catch (error) {
      message.error(getErrorMessage(error, '登录失败'))
    }
  }

  useEffect(() => {
    if (!telegramBotUsername || !widgetContainerRef.current) {
      return
    }

    const callbackName = `onTelegramAuth_${Date.now()}`
    const handleTelegramAuth = async (telegramUser: TelegramWidgetUser) => {
      try {
        const result = (await telegramApi.login(
          telegramUser,
        )) as TelegramLoginResult
        useAuthStore.setState({
          token: result.token,
          user: result.user,
          isAuthenticated: true,
        })
        setAuthToken(result.token)
        message.success('Telegram 登录成功')
        navigate('/dashboard', { replace: true })
      } catch (error) {
        message.error(getErrorMessage(error, 'Telegram 登录失败'))
      }
    }

    window[callbackName] = handleTelegramAuth

    const script = document.createElement('script')
    script.src = 'https://telegram.org/js/telegram-widget.js?22'
    script.async = true
    script.setAttribute('data-telegram-login', telegramBotUsername)
    script.setAttribute('data-size', 'large')
    script.setAttribute('data-userpic', 'false')
    script.setAttribute('data-request-access', 'write')
    script.setAttribute('data-onauth', `${callbackName}(user)`)

    widgetContainerRef.current.innerHTML = ''
    widgetContainerRef.current.appendChild(script)

    return () => {
      if (widgetContainerRef.current) {
        widgetContainerRef.current.innerHTML = ''
      }
      delete window[callbackName]
    }
  }, [navigate, telegramBotUsername])

  return (
    <div className="flex min-h-screen items-center justify-center bg-slate-100 px-4">
      <Card className="w-full max-w-md shadow-sm">
        <BrandLogo subtitle="登录后可管理节点、规则与流量统计" />

        <Form<LoginFormValues> layout="vertical" onFinish={handleSubmit}>
          <Form.Item
            label="邮箱"
            name="email"
            rules={[
              { required: true, message: '请输入邮箱' },
              { type: 'email', message: '邮箱格式不正确' },
            ]}
          >
            <Input placeholder="请输入邮箱" />
          </Form.Item>

          <Form.Item
            label="密码"
            name="password"
            rules={[{ required: true, message: '请输入密码' }]}
          >
            <Input.Password placeholder="请输入密码" />
          </Form.Item>

          <Form.Item name="remember" valuePropName="checked" initialValue={true}>
            <Checkbox>记住我</Checkbox>
          </Form.Item>

          <Button type="primary" htmlType="submit" block loading={isLoading}>
            登录
          </Button>
        </Form>

        {telegramBotUsername ? (
          <>
            <Divider>或使用 Telegram 登录</Divider>
            <div
              ref={widgetContainerRef}
              className="flex w-full justify-center pb-2"
            />
          </>
        ) : null}

        <Typography.Paragraph style={{ marginTop: 16, marginBottom: 0 }}>
          还没有账号？<Link to="/register">立即注册</Link>
        </Typography.Paragraph>
      </Card>
    </div>
  )
}

export default Login
