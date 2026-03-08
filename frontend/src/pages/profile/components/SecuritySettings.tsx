import { useState } from 'react'
import { Button, Descriptions, Input, Modal, Typography, message, Space, Tag } from 'antd'
import { CopyOutlined, LogoutOutlined, SafetyCertificateOutlined } from '@ant-design/icons'
import { useNavigate } from 'react-router-dom'

import { authApi, telegramApi } from '../../../services/api'
import { useAuthStore } from '../../../store/auth'
import { getErrorMessage } from '../../../utils/error'

const SecuritySettings = () => {
  const [loading, setLoading] = useState(false)
  const [telegramLoading, setTelegramLoading] = useState(false)
  const [notifyLoading, setNotifyLoading] = useState(false)
  const [ssoLoading, setSSOLoading] = useState(false)
  const [bindModalOpen, setBindModalOpen] = useState(false)
  const [bindCommand, setBindCommand] = useState('')
  const [ssoURL, setSSOURL] = useState('')
  const navigate = useNavigate()
  const user = useAuthStore((state) => state.user)
  const logout = useAuthStore((state) => state.logout)
  const fetchMe = useAuthStore((state) => state.fetchMe)

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
          logout()
          message.success('已撤销所有登录会话，请重新登录')
          navigate('/login', { replace: true })
        } catch (error) {
          message.error(getErrorMessage(error, '操作失败'))
        } finally {
          setLoading(false)
        }
      },
    })
  }

  const handleTelegramBind = async () => {
    try {
      setTelegramLoading(true)
      const result = await telegramApi.bind()
      const command = String(result.command ?? '').trim()
      if (!command) {
        message.error('绑定指令生成失败，请稍后重试')
        return
      }
      setBindCommand(command)
      setBindModalOpen(true)
    } catch (error) {
      message.error(getErrorMessage(error, '生成绑定指令失败'))
    } finally {
      setTelegramLoading(false)
    }
  }

  const handleTelegramUnbind = async () => {
    Modal.confirm({
      title: '解绑 Telegram',
      content: '解绑后将无法接收 Telegram 通知，确定继续吗？',
      okText: '确定解绑',
      cancelText: '取消',
      okButtonProps: { danger: true },
      onOk: async () => {
        try {
          setTelegramLoading(true)
          await telegramApi.unbind()
          await fetchMe()
          message.success('Telegram 已解绑')
        } catch (error) {
          message.error(getErrorMessage(error, '解绑失败'))
        } finally {
          setTelegramLoading(false)
        }
      },
    })
  }

  const handleGenerateSSOURL = async () => {
    try {
      setSSOLoading(true)
      const result = await telegramApi.generateSSOURL()
      const url = String(result.login_url ?? '').trim()
      if (!url) {
        message.error('登录链接生成失败')
        return
      }
      setSSOURL(url)
      await navigator.clipboard.writeText(url)
      message.success('已生成并复制 SSO 登录链接')
    } catch (error) {
      message.error(getErrorMessage(error, '生成 SSO 登录链接失败'))
    } finally {
      setSSOLoading(false)
    }
  }

  const handleSendTestNotification = async () => {
    try {
      setNotifyLoading(true)
      await telegramApi.notify('这是一条 Telegram 测试通知，来自 NodePass 面板。')
      message.success('测试通知已发送')
    } catch (error) {
      message.error(getErrorMessage(error, '发送测试通知失败'))
    } finally {
      setNotifyLoading(false)
    }
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
            <Space>
              <Tag color="default">未绑定</Tag>
              <Button
                size="small"
                type="link"
                onClick={() => void handleTelegramBind()}
                loading={telegramLoading}
              >
                立即绑定
              </Button>
            </Space>
          )}
        </Descriptions.Item>
        <Descriptions.Item label="Telegram 操作">
          <Space wrap>
            <Button
              size="small"
              onClick={() => void handleTelegramBind()}
              loading={telegramLoading}
            >
              生成绑定指令
            </Button>
            <Button
              size="small"
              onClick={() => void handleGenerateSSOURL()}
              loading={ssoLoading}
              disabled={!user?.telegram_id}
            >
              生成 SSO 链接
            </Button>
            <Button
              size="small"
              onClick={() => void handleSendTestNotification()}
              loading={notifyLoading}
              disabled={!user?.telegram_id}
            >
              发送测试通知
            </Button>
            <Button
              size="small"
              danger
              onClick={() => void handleTelegramUnbind()}
              disabled={!user?.telegram_id}
            >
              解绑
            </Button>
          </Space>
          {ssoURL ? (
            <Typography.Paragraph copyable style={{ marginTop: 8, marginBottom: 0 }}>
              {ssoURL}
            </Typography.Paragraph>
          ) : null}
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

      <Modal
        title="Telegram 绑定指令"
        open={bindModalOpen}
        onCancel={() => {
          setBindModalOpen(false)
          setBindCommand('')
        }}
        footer={null}
      >
        <Typography.Paragraph>
          将下方指令发送给你的 Telegram Bot 完成绑定：
        </Typography.Paragraph>
        <Input.TextArea value={bindCommand} autoSize={{ minRows: 2, maxRows: 4 }} readOnly />
        <Space style={{ marginTop: 12 }}>
          <Button
            icon={<CopyOutlined />}
            onClick={async () => {
              await navigator.clipboard.writeText(bindCommand)
              message.success('绑定指令已复制')
            }}
          >
            复制指令
          </Button>
          <Button
            type="primary"
            onClick={() => {
              setBindModalOpen(false)
              setBindCommand('')
            }}
          >
            我已发送
          </Button>
        </Space>
      </Modal>
    </div>
  )
}

export default SecuritySettings
