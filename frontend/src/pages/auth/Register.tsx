import { LockOutlined, MailOutlined, UserOutlined } from '@ant-design/icons'
import { Button, Card, Checkbox, Divider, Form, Input, Progress, Typography, message, Space } from 'antd'
import { useEffect, useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'

import BrandLogo from '../../components/common/BrandLogo'
import { usePageTitle } from '../../hooks/usePageTitle'
import { useAuthStore } from '../../store/auth'
import { getErrorMessage } from '../../utils/error'
import { getHomePathByRole } from '../../utils/route'

type RegisterFormValues = {
  username: string
  email: string
  password: string
  confirmPassword: string
  agreement: boolean
}

const Register = () => {
  usePageTitle('注册')

  const navigate = useNavigate()
  const [form] = Form.useForm()
  const register = useAuthStore((state) => state.register)
  const isLoading = useAuthStore((state) => state.isLoading)
  const isAuthenticated = useAuthStore((state) => state.isAuthenticated)
  const user = useAuthStore((state) => state.user)
  const [passwordStrength, setPasswordStrength] = useState(0)
  const [passwordStrengthText, setPasswordStrengthText] = useState('')

  useEffect(() => {
    if (isAuthenticated) {
      navigate(getHomePathByRole(user?.role), { replace: true })
    }
  }, [isAuthenticated, navigate, user?.role])

  const calculatePasswordStrength = (password: string) => {
    let strength = 0
    if (!password) {
      setPasswordStrength(0)
      setPasswordStrengthText('')
      return
    }

    // 长度
    if (password.length >= 8) strength += 20
    if (password.length >= 12) strength += 10

    // 包含小写字母
    if (/[a-z]/.test(password)) strength += 20

    // 包含大写字母
    if (/[A-Z]/.test(password)) strength += 20

    // 包含数字
    if (/\d/.test(password)) strength += 15

    // 包含特殊字符
    if (/[^a-zA-Z0-9]/.test(password)) strength += 15

    setPasswordStrength(strength)

    if (strength < 40) {
      setPasswordStrengthText('弱')
    } else if (strength < 70) {
      setPasswordStrengthText('中')
    } else {
      setPasswordStrengthText('强')
    }
  }

  const getPasswordStrengthColor = () => {
    if (passwordStrength < 40) return '#ff4d4f'
    if (passwordStrength < 70) return '#faad14'
    return '#52c41a'
  }

  const handleSubmit = async (values: RegisterFormValues) => {
    try {
      await register({
        username: values.username,
        email: values.email,
        password: values.password,
      })
      message.success('注册成功')
      navigate('/login', { replace: true })
    } catch (error) {
      message.error(getErrorMessage(error, '注册失败'))
    }
  }

  return (
    <div className="flex min-h-screen items-center justify-center bg-gradient-to-br from-blue-50 to-indigo-100 px-4 py-8">
      <Card className="w-full max-w-md shadow-lg">
        <div style={{ textAlign: 'center', marginBottom: 24 }}>
          <BrandLogo subtitle="创建账号，开启高效节点管理之旅" />
        </div>

        <Form<RegisterFormValues> form={form} layout="vertical" onFinish={handleSubmit}>
          <Form.Item
            label="用户名"
            name="username"
            rules={[
              { required: true, message: '请输入用户名' },
              { min: 3, message: '用户名至少 3 个字符' },
              { max: 20, message: '用户名最多 20 个字符' },
              { pattern: /^[a-zA-Z0-9_]+$/, message: '用户名只能包含字母、数字和下划线' },
            ]}
          >
            <Input
              prefix={<UserOutlined />}
              placeholder="请输入用户名"
              size="large"
            />
          </Form.Item>

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
            rules={[
              { required: true, message: '请输入密码' },
              { min: 8, message: '密码至少 8 位' },
              {
                pattern: /[a-z]/,
                message: '密码必须包含至少一个小写字母',
              },
              {
                pattern: /[A-Z]/,
                message: '密码必须包含至少一个大写字母',
              },
              {
                pattern: /\d/,
                message: '密码必须包含至少一个数字',
              },
              {
                pattern: /[^a-zA-Z0-9]/,
                message: '密码必须包含至少一个特殊字符',
              },
            ]}
          >
            <Input.Password
              prefix={<LockOutlined />}
              placeholder="请输入密码（至少8位，包含大小写字母、数字和特殊字符）"
              size="large"
              onChange={(e) => calculatePasswordStrength(e.target.value)}
            />
          </Form.Item>

          {passwordStrength > 0 && (
            <div style={{ marginTop: -16, marginBottom: 16 }}>
              <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 4 }}>
                <Typography.Text type="secondary" style={{ fontSize: 12 }}>
                  密码强度
                </Typography.Text>
                <Typography.Text style={{ fontSize: 12, color: getPasswordStrengthColor() }}>
                  {passwordStrengthText}
                </Typography.Text>
              </div>
              <Progress
                percent={passwordStrength}
                strokeColor={getPasswordStrengthColor()}
                showInfo={false}
                size="small"
              />
            </div>
          )}

          <Form.Item
            label="确认密码"
            name="confirmPassword"
            dependencies={['password']}
            rules={[
              { required: true, message: '请再次输入密码' },
              ({ getFieldValue }) => ({
                validator(_, value) {
                  if (!value || getFieldValue('password') === value) {
                    return Promise.resolve()
                  }
                  return Promise.reject(new Error('两次输入的密码不一致'))
                },
              }),
            ]}
          >
            <Input.Password
              prefix={<LockOutlined />}
              placeholder="请再次输入密码"
              size="large"
            />
          </Form.Item>

          <Form.Item
            name="agreement"
            valuePropName="checked"
            rules={[
              {
                validator: (_, value) =>
                  value ? Promise.resolve() : Promise.reject(new Error('请阅读并同意用户协议和隐私政策')),
              },
            ]}
          >
            <Checkbox>
              我已阅读并同意
              <a href="/terms" target="_blank" rel="noopener noreferrer"> 用户协议 </a>
              和
              <a href="/privacy" target="_blank" rel="noopener noreferrer"> 隐私政策</a>
            </Checkbox>
          </Form.Item>

          <Button type="primary" htmlType="submit" block size="large" loading={isLoading}>
            注册
          </Button>
        </Form>

        <Divider style={{ margin: '16px 0' }} />

        <div style={{ textAlign: 'center' }}>
          <Space direction="vertical" size={8}>
            <Typography.Text type="secondary">
              已有账号？<Link to="/login">立即登录</Link>
            </Typography.Text>
          </Space>
        </div>
      </Card>
    </div>
  )
}

export default Register
