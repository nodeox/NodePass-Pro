import { ArrowLeftOutlined, ToolOutlined } from '@ant-design/icons'
import { Button, Card, Result, Typography } from 'antd'
import { Link } from 'react-router-dom'

import BrandLogo from '../../components/common/BrandLogo'
import { usePageTitle } from '../../hooks/usePageTitle'

const ForgotPassword = () => {
  usePageTitle('忘记密码')

  return (
    <div className="flex min-h-screen items-center justify-center bg-gradient-to-br from-blue-50 to-indigo-100 px-4">
      <Card className="w-full max-w-md shadow-lg">
        <div style={{ textAlign: 'center', marginBottom: 16 }}>
          <BrandLogo subtitle="找回账号密码" />
        </div>

        <Result
          status="info"
          icon={<ToolOutlined />}
          title="忘记密码功能暂未开放"
          subTitle="请联系系统管理员重置密码，或使用已登录会话在个人中心修改密码。"
          extra={[
            <Button type="primary" key="back-login" icon={<ArrowLeftOutlined />}>
              <Link to="/login">返回登录</Link>
            </Button>,
          ]}
        />

        <Typography.Paragraph type="secondary" style={{ marginBottom: 0, textAlign: 'center' }}>
          出于安全考虑，当前页面不会模拟发送验证码或密码重置结果。
        </Typography.Paragraph>
      </Card>
    </div>
  )
}

export default ForgotPassword
