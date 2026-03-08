import { useEffect, useMemo, useState } from 'react'
import {
  BarChartOutlined,
  LockOutlined,
  ReloadOutlined,
  SafetyOutlined,
  UserOutlined,
} from '@ant-design/icons'
import {
  Avatar,
  Button,
  Card,
  Col,
  Descriptions,
  Form,
  Input,
  Modal,
  Progress,
  Row,
  Space,
  Statistic,
  Tabs,
  Tag,
  Typography,
  message,
} from 'antd'
import type { TabsProps } from 'antd'

import PageContainer from '../../components/common/PageContainer'
import { usePageTitle } from '../../hooks/usePageTitle'
import { useAuthStore } from '../../store/auth'
import { formatBytes, formatDateTime } from '../../utils/format'
import { authApi } from '../../services/api'
import type { ChangeEmailPayload, ChangePasswordPayload } from '../../types'
import { getErrorMessage } from '../../utils/error'
import TrafficChart from './components/TrafficChart'
import SecuritySettings from './components/SecuritySettings'

type PasswordFormValues = ChangePasswordPayload & {
  confirm_password: string
}

type EmailFormValues = ChangeEmailPayload & {
  password: string
}

const roleMeta: Record<string, { color: string; text: string }> = {
  admin: { color: 'red', text: '管理员' },
  user: { color: 'blue', text: '普通用户' },
}

const statusMeta: Record<string, { color: string; text: string }> = {
  normal: { color: 'green', text: '正常' },
  paused: { color: 'orange', text: '暂停' },
  banned: { color: 'red', text: '已封禁' },
  overlimit: { color: 'gold', text: '超额限制' },
}

const Profile = () => {
  usePageTitle('个人中心')

  const user = useAuthStore((state) => state.user)
  const fetchMe = useAuthStore((state) => state.fetchMe)

  const [passwordModalVisible, setPasswordModalVisible] = useState(false)
  const [emailModalVisible, setEmailModalVisible] = useState(false)
  const [refreshLoading, setRefreshLoading] = useState(false)
  const [passwordLoading, setPasswordLoading] = useState(false)
  const [emailLoading, setEmailLoading] = useState(false)
  const [sendCodeLoading, setSendCodeLoading] = useState(false)
  const [codeCountdown, setCodeCountdown] = useState(0)
  const [passwordForm] = Form.useForm<PasswordFormValues>()
  const [emailForm] = Form.useForm<EmailFormValues>()

  useEffect(() => {
    if (codeCountdown <= 0) {
      return
    }
    const timer = window.setInterval(() => {
      setCodeCountdown((prev) => {
        if (prev <= 1) {
          window.clearInterval(timer)
          return 0
        }
        return prev - 1
      })
    }, 1000)
    return () => window.clearInterval(timer)
  }, [codeCountdown])

  const summary = useMemo(() => {
    const userRecord = (user ?? {}) as Record<string, unknown>
    const trafficUsed = Number(user?.traffic_used ?? user?.trafficUsed ?? 0)
    const trafficQuota = Number(user?.traffic_quota ?? user?.trafficQuota ?? 0)
    const vipLevel = Number(user?.vip_level ?? user?.vipLevel ?? 0)
    const maxRules = Number(user?.maxRules ?? userRecord.max_rules ?? 0)
    const maxBandwidth = Number(user?.maxBandwidth ?? userRecord.max_bandwidth ?? 0)
    const usedPercent = trafficQuota > 0
      ? Math.min(Math.round((trafficUsed / trafficQuota) * 100), 100)
      : 0
    const remaining = Math.max(trafficQuota - trafficUsed, 0)

    return {
      trafficUsed,
      trafficQuota,
      vipLevel,
      maxRules,
      maxBandwidth,
      usedPercent,
      remaining,
    }
  }, [user])

  const handleRefreshProfile = async () => {
    try {
      setRefreshLoading(true)
      await fetchMe()
      message.success('资料已刷新')
    } catch (error) {
      message.error(getErrorMessage(error, '刷新资料失败'))
    } finally {
      setRefreshLoading(false)
    }
  }

  const handleChangePassword = async (values: PasswordFormValues) => {
    try {
      setPasswordLoading(true)
      await authApi.changePassword({
        old_password: values.old_password,
        new_password: values.new_password,
      })
      message.success('密码修改成功')
      setPasswordModalVisible(false)
      passwordForm.resetFields()
    } catch (error) {
      message.error(getErrorMessage(error, '密码修改失败'))
    } finally {
      setPasswordLoading(false)
    }
  }

  const handleOpenEmailModal = () => {
    setEmailModalVisible(true)
  }

  const handleSendEmailCode = async () => {
    try {
      const values = await emailForm.validateFields(['new_email', 'password'])
      const normalizedEmail = String(values.new_email ?? '').trim()
      const normalizedPassword = String(values.password ?? '').trim()
      if (!normalizedEmail || !normalizedPassword) {
        message.error('请填写新邮箱和当前密码')
        return
      }
      setSendCodeLoading(true)
      const result = await authApi.sendEmailChangeCode({
        new_email: normalizedEmail,
        password: normalizedPassword,
      })
      setCodeCountdown(60)
      if (result.debug_code) {
        message.info(`开发验证码：${result.debug_code}`, 6)
      }
      if (result.sent) {
        message.success('验证码已发送，请查收邮箱')
      } else {
        message.success('验证码已生成（当前为调试模式）')
      }
    } catch (error) {
      if (typeof error === 'object' && error !== null && 'errorFields' in error) {
        return
      }
      message.error(getErrorMessage(error, '发送验证码失败'))
    } finally {
      setSendCodeLoading(false)
    }
  }

  const handleChangeEmail = async (values: EmailFormValues) => {
    try {
      const normalizedEmail = String(values.new_email ?? '').trim()
      const normalizedCode = String(values.code ?? '').trim()
      if (!normalizedEmail || !normalizedCode) {
        message.error('请填写新邮箱和验证码')
        return
      }
      setEmailLoading(true)
      await authApi.changeEmail({
        new_email: normalizedEmail,
        code: normalizedCode,
      })
      message.success('邮箱修改成功，请使用新邮箱登录')
      setEmailModalVisible(false)
      setCodeCountdown(0)
      emailForm.resetFields()
      await fetchMe()
    } catch (error) {
      message.error(getErrorMessage(error, '修改邮箱失败'))
    } finally {
      setEmailLoading(false)
    }
  }

  const currentRole = roleMeta[user?.role ?? 'user'] ?? roleMeta.user
  const currentStatus = statusMeta[user?.status ?? 'normal'] ?? {
    color: 'default',
    text: user?.status || '未知',
  }

  const items: TabsProps['items'] = [
    {
      key: 'overview',
      label: '账户概览',
      children: (
        <Space direction="vertical" size={16} style={{ width: '100%' }}>
          <Card>
            <Row gutter={[16, 16]} align="middle">
              <Col xs={24} md={16}>
                <Space size={16}>
                  <Avatar size={64} icon={<UserOutlined />} />
                  <Space direction="vertical" size={4}>
                    <Space size={8}>
                      <Typography.Title level={4} style={{ margin: 0 }}>
                        {user?.username || '-'}
                      </Typography.Title>
                      <Tag color={currentRole.color}>{currentRole.text}</Tag>
                      <Tag color={currentStatus.color}>{currentStatus.text}</Tag>
                    </Space>
                    <Typography.Text type="secondary">
                      {user?.email || '-'}
                    </Typography.Text>
                    <Typography.Text type="secondary">
                      用户 ID: {user?.id ?? '-'}
                    </Typography.Text>
                  </Space>
                </Space>
              </Col>
              <Col xs={24} md={8}>
                <Space direction="vertical" style={{ width: '100%' }}>
                  <Button
                    icon={<ReloadOutlined />}
                    onClick={() => void handleRefreshProfile()}
                    loading={refreshLoading}
                  >
                    刷新资料
                  </Button>
                  <Button onClick={handleOpenEmailModal}>修改邮箱</Button>
                </Space>
              </Col>
            </Row>
          </Card>

          <Row gutter={[16, 16]}>
            <Col xs={24} sm={12} lg={6}>
              <Card>
                <Statistic title="VIP 等级" value={`VIP ${summary.vipLevel}`} />
              </Card>
            </Col>
            <Col xs={24} sm={12} lg={6}>
              <Card>
                <Statistic title="最大规则数" value={summary.maxRules} />
              </Card>
            </Col>
            <Col xs={24} sm={12} lg={6}>
              <Card>
                <Statistic title="带宽上限" value={summary.maxBandwidth} suffix="Mbps" />
              </Card>
            </Col>
            <Col xs={24} sm={12} lg={6}>
              <Card>
                <Statistic title="剩余流量" value={formatBytes(summary.remaining)} />
              </Card>
            </Col>
          </Row>

          <Card>
            <Descriptions
              bordered
              column={1}
              items={[
                {
                  key: 'vip_expires',
                  label: 'VIP 到期时间',
                  children: user?.vip_expires_at
                    ? formatDateTime(user.vip_expires_at)
                    : '永久',
                },
                {
                  key: 'created_at',
                  label: '注册时间',
                  children: formatDateTime(user?.created_at ?? user?.createdAt),
                },
                {
                  key: 'last_login',
                  label: '最后登录',
                  children: formatDateTime(user?.lastLoginAt),
                },
              ]}
            />

            <div style={{ marginTop: 20 }}>
              <Typography.Text strong>流量配额使用</Typography.Text>
              <Progress
                style={{ marginTop: 10 }}
                percent={summary.usedPercent}
                status={summary.usedPercent >= 90 ? 'exception' : 'active'}
                format={() => `${formatBytes(summary.trafficUsed)} / ${formatBytes(summary.trafficQuota)}`}
              />
            </div>
          </Card>
        </Space>
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
        <Space direction="vertical" size={16} style={{ width: '100%' }}>
          <Card>
            <Space direction="vertical" size={8} style={{ width: '100%' }}>
              <Typography.Text strong>
                <LockOutlined /> 密码管理
              </Typography.Text>
              <Typography.Text type="secondary">
                建议定期更新密码，避免与其他平台重复使用。
              </Typography.Text>
              <Button type="primary" onClick={() => setPasswordModalVisible(true)}>
                修改密码
              </Button>
            </Space>
          </Card>
          <Card>
            <SecuritySettings />
          </Card>
        </Space>
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
      <Tabs defaultActiveKey="overview" items={items} />

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
              <Button type="primary" htmlType="submit" loading={passwordLoading}>
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
          setCodeCountdown(0)
          emailForm.resetFields()
        }}
        footer={null}
      >
        <Form form={emailForm} layout="vertical" onFinish={handleChangeEmail}>
          <Form.Item
            label="新邮箱"
            name="new_email"
            rules={[
              { required: true, message: '请输入新邮箱' },
              { type: 'email', message: '邮箱格式不正确' },
            ]}
          >
            <Input placeholder="请输入新邮箱地址" />
          </Form.Item>

          <Form.Item
            label="当前密码"
            name="password"
            rules={[{ required: true, message: '请输入当前密码' }]}
          >
            <Input.Password placeholder="请输入当前登录密码" />
          </Form.Item>

          <Form.Item
            label="验证码"
            name="code"
            rules={[
              { required: true, message: '请输入验证码' },
              { len: 6, message: '验证码为 6 位数字' },
            ]}
          >
            <Input
              placeholder="请输入 6 位验证码"
              maxLength={6}
              addonAfter={(
                <Button
                  type="link"
                  size="small"
                  disabled={codeCountdown > 0}
                  loading={sendCodeLoading}
                  onClick={() => void handleSendEmailCode()}
                >
                  {codeCountdown > 0 ? `${codeCountdown}s 后重发` : '发送验证码'}
                </Button>
              )}
            />
          </Form.Item>

          <Form.Item style={{ marginBottom: 0 }}>
            <Space>
              <Button type="primary" htmlType="submit" loading={emailLoading}>
                确认修改
              </Button>
              <Button
                onClick={() => {
                  setEmailModalVisible(false)
                  setCodeCountdown(0)
                  emailForm.resetFields()
                }}
              >
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
