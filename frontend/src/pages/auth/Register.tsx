import { Button, Card, Form, Input, Typography, message } from 'antd'
import { useEffect } from 'react'
import { Link, useNavigate } from 'react-router-dom'

import BrandLogo from '../../components/common/BrandLogo'
import { usePageTitle } from '../../hooks/usePageTitle'
import { useAuthStore } from '../../store/auth'
import { getErrorMessage } from '../../utils/error'

type RegisterFormValues = {
  username: string
  email: string
  password: string
  confirmPassword: string
}

const Register = () => {
  usePageTitle('注册')

  const navigate = useNavigate()
  const register = useAuthStore((state) => state.register)
  const isLoading = useAuthStore((state) => state.isLoading)
  const isAuthenticated = useAuthStore((state) => state.isAuthenticated)

  useEffect(() => {
    if (isAuthenticated) {
      navigate('/dashboard', { replace: true })
    }
  }, [isAuthenticated, navigate])

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
    <div className="flex min-h-screen items-center justify-center bg-slate-100 px-4">
      <Card className="w-full max-w-md shadow-sm">
        <BrandLogo subtitle="创建你的 NodePass Panel 账号" />

        <Form<RegisterFormValues> layout="vertical" onFinish={handleSubmit}>
          <Form.Item
            label="用户名"
            name="username"
            rules={[
              { required: true, message: '请输入用户名' },
              { min: 3, message: '用户名至少 3 个字符' },
              { max: 20, message: '用户名最多 20 个字符' },
            ]}
          >
            <Input placeholder="请输入用户名" />
          </Form.Item>

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
            rules={[
              { required: true, message: '请输入密码' },
              { min: 8, message: '密码至少 8 位' },
            ]}
          >
            <Input.Password placeholder="请输入密码" />
          </Form.Item>

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
            <Input.Password placeholder="请再次输入密码" />
          </Form.Item>

          <Button type="primary" htmlType="submit" block loading={isLoading}>
            注册
          </Button>
        </Form>

        <Typography.Paragraph style={{ marginTop: 16, marginBottom: 0 }}>
          已有账号？<Link to="/login">去登录</Link>
        </Typography.Paragraph>
      </Card>
    </div>
  )
}

export default Register
