import { useState } from 'react'
import { Button, Descriptions, Modal, message, Space, Tag } from 'antd'
import { LogoutOutlined, SafetyCertificateOutlined } from '@ant-design/icons'

import { authApi } from '../../../services/api'
import { useAuthStore } from '../../../store/auth'

const SecuritySettings = () => {
  const [loading, setLoading] = useState(false)
  const user = useAuthStore((state) => state.user)

  const handleRevokeAllSessions = () => {
    Modal.confirm({
      title: '撤销所有登录会话',
      content: '此操作将登出所有设备上的登录会话，您需要重新登录。确定要继续吗？',
      okText: '确定',
      cancelText: '取消',
      okButtonProps: { danger: true },
      onOk: async () => {
        try {
          setLoading(true)
          await authApi.revokeAllTokens()
          message.success('已撤销所有登录会话，请重新登录')
          // 清除本地存储并跳转到登录页
          setTimeout(() => {
            window.location.href = '/login'
          }, 1500)
        } catch (error: any) {
          message.error(error.response?.data?.error?.message || '操作失败')
        } finally {
          setLoading(false)
        }
      },
    })
  }

  return (
    <div>
      <h4>
        <SafetyCertificateOutlined /> 账户安全
      </h4>

      <Descriptions column={1} bordered style={{ marginTop: 12, marginBottom: 24 }}>
        <Descriptions.Item label="Telegram 绑定">
          {user?.telegram_id ? (
            <Space>
              <Tag color="green">已绑定</Tag>
              <span>@{user.telegram_username || user.telegram_id}</span>
            </Space>
          ) : (
            <Tag color="default">未绑定</Tag>
          )}
        </Descriptions.Item>
        <Descriptions.Item label="两步验证">
          <Tag color="default">未启用</Tag>
          <span style={{ marginLeft: 8, color: '#999' }}>（功能开发中）</span>
        </Descriptions.Item>
      </Descriptions>

      <div>
        <h4>会话管理</h4>
        <p style={{ color: '#666', marginBottom: 12 }}>
          撤销所有登录会话可以确保其他设备上的登录失效，提高账户安全性
        </p>
        <Button
          danger
          icon={<LogoutOutlined />}
          onClick={handleRevokeAllSessions}
          loading={loading}
        >
          撤销所有登录会话
        </Button>
      </div>
    </div>
  )
}

export default SecuritySettings
