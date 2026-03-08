import { useState } from 'react'
import { Card, Tabs, Descriptions, Button, Modal, Form, Input, message, Progress, Space, Tag } from 'antd'
import { LockOutlined, MailOutlined, SafetyOutlined, BarChartOutlined } from '@ant-design/icons'

import PageContainer from '../../components/common/PageContainer'
import { usePageTitle } from '../../hooks/usePageTitle'
import { useAuthStore } from '../../store/auth'
import { formatBytes, formatDateTime } from '../../utils/format'
import { authApi } from '../../services/api'
import TrafficChart from './components/TrafficChart'
import SecuritySettings from './components/SecuritySettings'

const Profile = () => {
  usePageTitle('个人中心')

  const user = useAuthStore((state) => state.user)
  const [passwordModalVisible, setPasswordModalVisible] = useState(false)
  const [emailModalVisible, setEmailModalVisible] = useState(false)
  const [passwordForm] = Form.useForm()
  const [emailForm] = Form.useForm()
  const [loading, setLoading] = useState(false)

  const handleChangePassword = async (values: { old_password: string; new_password: string; confirm_password: string }) => {
    try {
      setLoading(true)
      await authApi.changePassword({
        old_password: values.old_password,
        new_password: values.new_password,
      })
      message.success('密码修改成功')
      setPasswordModalVisible(false)
      passwordForm.resetFields()
    } catch (error: any) {
      message.error(error.response?.data?.error?.message || '密码修改失败')
    } finally {
      setLoading(false)
    }
  }

  const handleChangeEmail = async (values: { new_email: string; password: string }) => {
    try {
      setLoading(true)
      // TODO: 实现修改邮箱 API
      console.log('修改邮箱:', values)
      message.info('修改邮箱功能开发中')
      setEmailModalVisible(false)
      emailForm.resetFields()
    } catch (error: any) {
      message.error(error.response?.data?.error?.message || '邮箱修改失败')
    } finally {
      setLoading(false)
    }
  }

  const trafficPercentage = user?.traffic_quota
    ? Math.min(Math.round((user.traffic_used / user.traffic_quota) * 100), 100)
    : 0

  const items = [
    {
      key: 'basic',
      label: '基本信息',
      children: (
        <Card>
          <Descriptions column={1} bordered>
            <Descriptions.Item label="用户名">{user?.username ?? '-'}</Descriptions.Item>
            <Descriptions.Item label="邮箱">
              <Space>
                {user?.email ?? '-'}
                <Button
                  type="link"
                  size="small"
                  icon={<MailOutlined />}
                  onClick={() => setEmailModalVisible(true)}
                >
                  修改
                </Button>
              </Space>
            </Descriptions.Item>
            <Descriptions.Item label="角色">
              <Tag color={user?.role === 'admin' ? 'red' : 'blue'}>
                {user?.role === 'admin' ? '管理员' : '普通用户'}
              </Tag>
            </Descriptions.Item>
            <Descriptions.Item label="账户状态">
              <Tag color={user?.status === 'normal' ? 'green' : 'red'}>
                {user?.status === 'normal' ? '正常' : user?.status}
              </Tag>
            </Descriptions.Item>
            <Descriptions.Item label="VIP 等级">
              <Tag color="gold">VIP {user?.vip_level ?? 0}</Tag>
            </Descriptions.Item>
            <Descriptions.Item label="VIP 到期">
              {user?.vip_expires_at ? formatDateTime(user.vip_expires_at) : '永久'}
            </Descriptions.Item>
            <Descriptions.Item label="注册时间">
              {formatDateTime(user?.created_at)}
            </Descriptions.Item>
            <Descriptions.Item label="最后登录">
              {formatDateTime(user?.lastLoginAt)}
            </Descriptions.Item>
          </Descriptions>

          <div style={{ marginTop: 24 }}>
            <h4>流量配额</h4>
            <Progress
              percent={trafficPercentage}
              status={trafficPercentage >= 90 ? 'exception' : 'normal'}
              format={() => `${formatBytes(user?.traffic_used ?? 0)} / ${formatBytes(user?.traffic_quota ?? 0)}`}
            />
            <p style={{ marginTop: 8, color: '#666' }}>
              剩余流量: {formatBytes((user?.traffic_quota ?? 0) - (user?.traffic_used ?? 0))}
            </p>
          </div>
        </Card>
      ),
    },
    {
      key: 'security',
      label: (
        <span>
          <SafetyOutlined />
          安全设置
        </span>
      ),
      children: (
        <Card>
          <Space direction="vertical" size="large" style={{ width: '100%' }}>
            <div>
              <h4>
                <LockOutlined /> 修改密码
              </h4>
              <p style={{ color: '#666', marginBottom: 12 }}>
                定期修改密码可以提高账户安全性
              </p>
              <Button type="primary" onClick={() => setPasswordModalVisible(true)}>
                修改密码
              </Button>
            </div>

            <SecuritySettings />
          </Space>
        </Card>
      ),
    },
    {
      key: 'traffic',
      label: (
        <span>
          <BarChartOutlined />
          流量统计
        </span>
      ),
      children: <TrafficChart />,
    },
  ]

  return (
    <PageContainer title="个人中心" description="管理您的账户信息、安全设置和流量使用情况">
      <Tabs defaultActiveKey="basic" items={items} />

      {/* 修改密码弹窗 */}
      <Modal
        title="修改密码"
        open={passwordModalVisible}
        onCancel={() => {
          setPasswordModalVisible(false)
          passwordForm.resetFields()
        }}
        footer={null}
      >
        <Form
          form={passwordForm}
          layout="vertical"
          onFinish={handleChangePassword}
        >
          <Form.Item
            label="原密码"
            name="old_password"
            rules={[{ required: true, message: '请输入原密码' }]}
          >
            <Input.Password placeholder="请输入原密码" />
          </Form.Item>

          <Form.Item
            label="新密码"
            name="new_password"
            rules={[
              { required: true, message: '请输入新密码' },
              { min: 8, message: '密码至少8个字符' },
            ]}
          >
            <Input.Password placeholder="请输入新密码（至少8个字符）" />
          </Form.Item>

          <Form.Item
            label="确认新密码"
            name="confirm_password"
            dependencies={['new_password']}
            rules={[
              { required: true, message: '请确认新密码' },
              ({ getFieldValue }) => ({
                validator(_, value) {
                  if (!value || getFieldValue('new_password') === value) {
                    return Promise.resolve()
                  }
                  return Promise.reject(new Error('两次输入的密码不一致'))
                },
              }),
            ]}
          >
            <Input.Password placeholder="请再次输入新密码" />
          </Form.Item>

          <Form.Item style={{ marginBottom: 0 }}>
            <Space>
              <Button type="primary" htmlType="submit" loading={loading}>
                确认修改
              </Button>
              <Button onClick={() => {
                setPasswordModalVisible(false)
                passwordForm.resetFields()
              }}>
                取消
              </Button>
            </Space>
          </Form.Item>
        </Form>
      </Modal>

      {/* 修改邮箱弹窗 */}
      <Modal
        title="修改邮箱"
        open={emailModalVisible}
        onCancel={() => {
          setEmailModalVisible(false)
          emailForm.resetFields()
        }}
        footer={null}
      >
        <Form
          form={emailForm}
          layout="vertical"
          onFinish={handleChangeEmail}
        >
          <Form.Item label="当前邮箱">
            <Input value={user?.email} disabled />
          </Form.Item>

          <Form.Item
            label="新邮箱"
            name="new_email"
            rules={[
              { required: true, message: '请输入新邮箱' },
              { type: 'email', message: '请输入有效的邮箱地址' },
            ]}
          >
            <Input placeholder="请输入新邮箱地址" />
          </Form.Item>

          <Form.Item
            label="确认密码"
            name="password"
            rules={[{ required: true, message: '请输入密码以确认身份' }]}
          >
            <Input.Password placeholder="请输入当前密码" />
          </Form.Item>

          <Form.Item style={{ marginBottom: 0 }}>
            <Space>
              <Button type="primary" htmlType="submit" loading={loading}>
                确认修改
              </Button>
              <Button onClick={() => {
                setEmailModalVisible(false)
                emailForm.resetFields()
              }}>
                取消
              </Button>
            </Space>
          </Form.Item>
        </Form>
      </Modal>
    </PageContainer>
  )
}

export default Profile
