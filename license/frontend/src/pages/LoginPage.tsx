import { Alert, Button, Card, Form, Input, Typography } from 'antd'
import { useState } from 'react'
import { authApi } from '../utils/api'
import { extractErrorMessage } from '../utils/request'

interface Props {
  onSuccess: (token: string, user: { id: number; username: string; email: string }) => void
}

export default function LoginPage({ onSuccess }: Props) {
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string>('')

  const onFinish = async (values: { username: string; password: string }) => {
    setLoading(true)
    setError('')
    try {
      const res = await authApi.login(values)
      onSuccess(res.token, res.user)
    } catch (err) {
      setError(extractErrorMessage(err))
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="login-shell">
      <Card className="login-card" bordered={false}>
        <Typography.Title level={3} style={{ marginBottom: 4 }}>
          NodePass 授权与版本中心
        </Typography.Title>
        <Typography.Paragraph type="secondary" style={{ marginBottom: 24 }}>
          授权系统与版本系统合一管理
        </Typography.Paragraph>

        {error ? <Alert type="error" showIcon message={error} style={{ marginBottom: 16 }} /> : null}

        <Form layout="vertical" onFinish={onFinish} initialValues={{ username: 'admin' }}>
          <Form.Item label="用户名" name="username" rules={[{ required: true, message: '请输入用户名' }]}>
            <Input placeholder="admin" />
          </Form.Item>
          <Form.Item label="密码" name="password" rules={[{ required: true, message: '请输入密码' }]}>
            <Input.Password placeholder="请输入密码" />
          </Form.Item>
          <Button htmlType="submit" type="primary" block loading={loading}>
            登录
          </Button>
        </Form>
      </Card>
    </div>
  )
}
