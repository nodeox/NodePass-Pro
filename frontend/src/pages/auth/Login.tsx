import { LockOutlined, MailOutlined } from '@ant-design/icons'
import { Button, Card, Checkbox, Divider, Form, Input, Typography, message, Space } from 'antd'
import { useEffect, useMemo, useRef } from 'react'
import { Link, useNavigate } from 'react-router-dom'

import BrandLogo from '../../components/common/BrandLogo'
import { usePageTitle } from '../../hooks/usePageTitle'
import { setAuthToken, telegramApi } from '../../services/api'
import { useAuthStore } from '../../store/auth'
import { getErrorMessage } from '../../utils/error'
import { getHomePathByRole } from '../../utils/route'
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
  const token = useAuthStore((state) => state.token)
  const isLoading = useAuthStore((state) => state.isLoading)
  const isAuthenticated = useAuthStore((state) => state.isAuthenticated)
  const user = useAuthStore((state) => state.user)

  const telegramBotUsername = useMemo(
    () => import.meta.env.VITE_TELEGRAM_BOT_USERNAME?.trim() ?? '',
    [],
  )

  useEffect(() => {
    if (isAuthenticated && token && user) {
      navigate(getHomePathByRole(user?.role), { replace: true })
    }
  }, [isAuthenticated, navigate, token, user])

  const handleSubmit = async (values: LoginFormValues) => {
    try {
      await login({
        account: values.email,
        password: values.password,
      })

      message.success('登录成功')
      const currentUser = useAuthStore.getState().user
      navigate(getHomePathByRole(currentUser?.role), { replace: true })
    } catch (error) {
      message.error(getErrorMessage(error, '登录失败'))
    }
  }

  useEffect(() => {
    if (!telegramBotUsername || !widgetContainerRef.current) {
      return
    }

    const widgetContainer = widgetContainerRef.current
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
        navigate(getHomePathByRole(result.user?.role), { replace: true })
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

    widgetContainer.replaceChildren()
    widgetContainer.appendChild(script)

    return () => {
      widgetContainer.replaceChildren()
      delete window[callbackName]
    }
  }, [navigate, telegramBotUsername])

  return (
    <div className="flex min-h-screen items-center justify-center bg-gradient-to-br from-blue-50 to-indigo-100 px-4">
      <Card className="w-full max-w-md shadow-lg">
        <div style={{ textAlign: 'center', marginBottom: 24 }}>
          <BrandLogo subtitle="安全、快速、可靠的节点管理平台" />
        </div>

        <Form<LoginFormValues> layout="vertical" onFinish={handleSubmit}>
          <Form.Item
            label="邮箱"
            name="email"
            rules={[
              { required: true, message: '请输入邮箱' },
              { type: 'email', message: '邮箱格式不正确' },
            ]}
          >
            <Input
              prefix={<MailOutlined />}
              placeholder="请输入邮箱"
              size="large"
            />
          </Form.Item>

          <Form.Item
            label="密码"
            name="password"
            rules={[{ required: true, message: '请输入密码' }]}
          >
            <Input.Password
              prefix={<LockOutlined />}
              placeholder="请输入密码"
              size="large"
            />
          </Form.Item>

          <Form.Item>
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
              <Form.Item name="remember" valuePropName="checked" initialValue={true} noStyle>
                <Checkbox>记住我</Checkbox>
              </Form.Item>
              <Link to="/forgot-password">忘记密码？</Link>
            </div>
          </Form.Item>

          <Button type="primary" htmlType="submit" block size="large" loading={isLoading}>
            登录
          </Button>
        </Form>

        {telegramBotUsername ? (
          <>
            <Divider>或使用第三方登录</Divider>
            <div
              ref={widgetContainerRef}
              className="flex w-full justify-center pb-2"
            />
          </>
        ) : null}

        <Divider style={{ margin: '16px 0' }} />

        <div style={{ textAlign: 'center' }}>
          <Space direction="vertical" size={8}>
            <Typography.Text type="secondary">
              还没有账号？<Link to="/register">立即注册</Link>
            </Typography.Text>
            <Typography.Text type="secondary" style={{ fontSize: 12 }}>
              登录即表示您同意我们的服务条款和隐私政策
            </Typography.Text>
          </Space>
        </div>
      </Card>
    </div>
  )
}

export default Login
