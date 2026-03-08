import { LockOutlined, MailOutlined } from '@ant-design/icons'
import { Button, Card, Form, Input, Result, Steps, Typography, message } from 'antd'
import { useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'

import BrandLogo from '../../components/common/BrandLogo'
import { usePageTitle } from '../../hooks/usePageTitle'

type ForgotPasswordFormValues = {
  email: string
}

type ResetPasswordFormValues = {
  code: string
  password: string
  confirmPassword: string
}

const ForgotPassword = () => {
  usePageTitle('忘记密码')

  const navigate = useNavigate()
  const [currentStep, setCurrentStep] = useState(0)
  const [email, setEmail] = useState('')
  const [loading, setLoading] = useState(false)

  const handleSendCode = async (values: ForgotPasswordFormValues) => {
    setLoading(true)
    try {
      // TODO: 调用发送验证码 API
      await new Promise((resolve) => setTimeout(resolve, 1000))
      setEmail(values.email)
      setCurrentStep(1)
      message.success('验证码已发送到您的邮箱')
    } catch (error) {
      message.error('发送验证码失败，请稍后重试')
    } finally {
      setLoading(false)
    }
  }

  const handleResetPassword = async (values: ResetPasswordFormValues) => {
    setLoading(true)
    try {
      // TODO: 调用重置密码 API
      console.log('Reset password with:', { email, ...values })
      await new Promise((resolve) => setTimeout(resolve, 1000))
      setCurrentStep(2)
      message.success('密码重置成功')
    } catch (error) {
      message.error('密码重置失败，请检查验证码是否正确')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="flex min-h-screen items-center justify-center bg-gradient-to-br from-blue-50 to-indigo-100 px-4">
      <Card className="w-full max-w-md shadow-lg">
        <div style={{ textAlign: 'center', marginBottom: 24 }}>
          <BrandLogo subtitle="重置您的账号密码" />
        </div>

        <Steps
          current={currentStep}
          items={[
            { title: '验证邮箱' },
            { title: '重置密码' },
            { title: '完成' },
          ]}
          style={{ marginBottom: 32 }}
        />

        {currentStep === 0 && (
          <Form<ForgotPasswordFormValues> layout="vertical" onFinish={handleSendCode}>
            <Typography.Paragraph type="secondary" style={{ marginBottom: 24 }}>
              请输入您注册时使用的邮箱地址，我们将向您发送验证码。
            </Typography.Paragraph>

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
                placeholder="请输入注册邮箱"
                size="large"
              />
            </Form.Item>

            <Button type="primary" htmlType="submit" block size="large" loading={loading}>
              发送验证码
            </Button>

            <div style={{ textAlign: 'center', marginTop: 16 }}>
              <Link to="/login">返回登录</Link>
            </div>
          </Form>
        )}

        {currentStep === 1 && (
          <Form<ResetPasswordFormValues> layout="vertical" onFinish={handleResetPassword}>
            <Typography.Paragraph type="secondary" style={{ marginBottom: 24 }}>
              验证码已发送到 <Typography.Text strong>{email}</Typography.Text>，请查收邮件。
            </Typography.Paragraph>

            <Form.Item
              label="验证码"
              name="code"
              rules={[
                { required: true, message: '请输入验证码' },
                { len: 6, message: '验证码为6位数字' },
              ]}
            >
              <Input
                placeholder="请输入6位验证码"
                size="large"
                maxLength={6}
              />
            </Form.Item>

            <Form.Item
              label="新密码"
              name="password"
              rules={[
                { required: true, message: '请输入新密码' },
                { min: 8, message: '密码至少 8 位' },
              ]}
            >
              <Input.Password
                prefix={<LockOutlined />}
                placeholder="请输入新密码"
                size="large"
              />
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
              <Input.Password
                prefix={<LockOutlined />}
                placeholder="请再次输入新密码"
                size="large"
              />
            </Form.Item>

            <Button type="primary" htmlType="submit" block size="large" loading={loading}>
              重置密码
            </Button>

            <div style={{ textAlign: 'center', marginTop: 16 }}>
              <Typography.Link onClick={() => setCurrentStep(0)}>
                重新发送验证码
              </Typography.Link>
            </div>
          </Form>
        )}

        {currentStep === 2 && (
          <Result
            status="success"
            title="密码重置成功！"
            subTitle="您的密码已成功重置，请使用新密码登录。"
            extra={[
              <Button type="primary" key="login" onClick={() => navigate('/login')}>
                去登录
              </Button>,
            ]}
          />
        )}
      </Card>
    </div>
  )
}

export default ForgotPassword
