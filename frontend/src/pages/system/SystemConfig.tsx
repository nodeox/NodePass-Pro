import {
  Alert,
  Button,
  Card,
  Col,
  Descriptions,
  Divider,
  Form,
  Input,
  InputNumber,
  Row,
  Skeleton,
  Select,
  Space,
  Switch,
  Tag,
  Typography,
  message,
} from 'antd'
import { useCallback, useEffect, useState } from 'react'

import PageContainer from '../../components/common/PageContainer'
import { usePageTitle } from '../../hooks/usePageTitle'
import { licenseApi, systemApi } from '../../services/api'
import type { LicenseStatus } from '../../types'
import { getErrorMessage } from '../../utils/error'
import { formatDateTime } from '../../utils/format'

type SystemConfigFormValues = {
  site_name: string
  site_url: string
  register_enabled: boolean
  default_vip_level: number
  telegram_bot_token: string
  telegram_bot_username: string
  heartbeat_timeout_seconds: number
  traffic_stats_interval_seconds: number
  smtp_enabled: boolean
  smtp_host: string
  smtp_port: number
  smtp_username: string
  smtp_password: string
  smtp_from_email: string
  smtp_from_name: string
  smtp_reply_to: string
  smtp_encryption: 'none' | 'starttls' | 'ssl'
  smtp_skip_verify: boolean
}

type LicenseDomainFormValues = {
  domain: string
  site_url: string
}

const parseBoolean = (value: string | undefined): boolean => {
  if (!value) {
    return false
  }
  return ['1', 'true', 'yes', 'on'].includes(value.toLowerCase())
}

const parseNumber = (value: string | undefined, fallback: number): number => {
  if (!value) {
    return fallback
  }
  const parsed = Number(value)
  return Number.isFinite(parsed) ? parsed : fallback
}

const normalizeSMTPEncryption = (value: string | undefined): 'none' | 'starttls' | 'ssl' => {
  const normalized = (value ?? '').trim().toLowerCase()
  if (normalized === 'none' || normalized === 'ssl' || normalized === 'starttls') {
    return normalized
  }
  return 'starttls'
}

const SystemConfig = () => {
  usePageTitle('系统配置')

  const [form] = Form.useForm<SystemConfigFormValues>()
  const [licenseDomainForm] = Form.useForm<LicenseDomainFormValues>()
  const [loading, setLoading] = useState<boolean>(true)
  const [saving, setSaving] = useState<boolean>(false)
  const [licenseLoading, setLicenseLoading] = useState<boolean>(true)
  const [updatingLicenseDomain, setUpdatingLicenseDomain] = useState<boolean>(false)
  const [licenseStatus, setLicenseStatus] = useState<LicenseStatus | null>(null)

  const loadConfig = useCallback(async (): Promise<void> => {
    setLoading(true)
    try {
      const config = await systemApi.config()
      form.setFieldsValue({
        site_name: config.site_name ?? 'NodePass Panel',
        site_url: config.site_url ?? '',
        register_enabled: parseBoolean(config.register_enabled),
        default_vip_level: parseNumber(config.default_vip_level, 0),
        telegram_bot_token: config.telegram_bot_token ?? '',
        telegram_bot_username: config.telegram_bot_username ?? '',
        heartbeat_timeout_seconds: parseNumber(config.heartbeat_timeout_seconds, 180),
        traffic_stats_interval_seconds: parseNumber(
          config.traffic_stats_interval_seconds,
          300,
        ),
        smtp_enabled: parseBoolean(config.smtp_enabled),
        smtp_host: config.smtp_host ?? '',
        smtp_port: parseNumber(config.smtp_port, 587),
        smtp_username: config.smtp_username ?? '',
        smtp_password: config.smtp_password ?? '',
        smtp_from_email: config.smtp_from_email ?? '',
        smtp_from_name: config.smtp_from_name ?? 'NodePass',
        smtp_reply_to: config.smtp_reply_to ?? '',
        smtp_encryption: normalizeSMTPEncryption(config.smtp_encryption),
        smtp_skip_verify: parseBoolean(config.smtp_skip_verify),
      })
    } catch (error) {
      message.error(getErrorMessage(error, '系统配置加载失败'))
    } finally {
      setLoading(false)
    }
  }, [form])

  const loadLicenseStatus = useCallback(async (): Promise<void> => {
    setLicenseLoading(true)
    try {
      const status = await licenseApi.status()
      setLicenseStatus(status)
      licenseDomainForm.setFieldsValue({
        domain: status.domain ?? '',
        site_url: status.site_url ?? '',
      })
    } catch (error) {
      message.error(getErrorMessage(error, '授权状态加载失败'))
      setLicenseStatus(null)
    } finally {
      setLicenseLoading(false)
    }
  }, [licenseDomainForm])

  useEffect(() => {
    void Promise.all([loadConfig(), loadLicenseStatus()])
  }, [loadConfig, loadLicenseStatus])

  const handleSubmit = async (values: SystemConfigFormValues): Promise<void> => {
    setSaving(true)
    try {
      const payloadEntries: Array<{ key: string; value: string }> = [
        { key: 'site_name', value: values.site_name.trim() },
        { key: 'site_url', value: values.site_url.trim() },
        { key: 'register_enabled', value: String(values.register_enabled) },
        { key: 'default_vip_level', value: String(values.default_vip_level) },
        { key: 'telegram_bot_token', value: values.telegram_bot_token.trim() },
        { key: 'telegram_bot_username', value: values.telegram_bot_username.trim() },
        {
          key: 'heartbeat_timeout_seconds',
          value: String(values.heartbeat_timeout_seconds),
        },
        {
          key: 'traffic_stats_interval_seconds',
          value: String(values.traffic_stats_interval_seconds),
        },
        { key: 'smtp_enabled', value: String(values.smtp_enabled) },
        { key: 'smtp_host', value: values.smtp_host.trim() },
        { key: 'smtp_port', value: String(values.smtp_port) },
        { key: 'smtp_username', value: values.smtp_username.trim() },
        { key: 'smtp_password', value: values.smtp_password.trim() },
        { key: 'smtp_from_email', value: values.smtp_from_email.trim() },
        { key: 'smtp_from_name', value: values.smtp_from_name.trim() },
        { key: 'smtp_reply_to', value: values.smtp_reply_to.trim() },
        { key: 'smtp_encryption', value: values.smtp_encryption },
        { key: 'smtp_skip_verify', value: String(values.smtp_skip_verify) },
      ]

      await systemApi.updateConfigs(payloadEntries)

      message.success('系统配置保存成功')
      await loadConfig()
    } catch (error) {
      message.error(getErrorMessage(error, '系统配置保存失败'))
    } finally {
      setSaving(false)
    }
  }

  const handleUpdateLicenseDomain = async (values: LicenseDomainFormValues): Promise<void> => {
    if (!licenseStatus?.enabled) {
      message.warning('运行时授权未开启，无需更换域名')
      return
    }

    const domain = values.domain.trim()
    const siteURL = values.site_url.trim()
    if (!domain && !siteURL) {
      message.warning('授权域名和站点地址至少填写一个')
      return
    }

    setUpdatingLicenseDomain(true)
    try {
      const updatedStatus = await licenseApi.updateDomain({
        domain,
        site_url: siteURL,
      })
      setLicenseStatus(updatedStatus)
      licenseDomainForm.setFieldsValue({
        domain: updatedStatus.domain ?? domain,
        site_url: updatedStatus.site_url ?? siteURL,
      })
      message.success('授权域名更新成功')
      await loadLicenseStatus()
    } catch (error) {
      message.error(getErrorMessage(error, '授权域名更新失败'))
    } finally {
      setUpdatingLicenseDomain(false)
    }
  }

  return (
    <PageContainer title="系统配置" description="维护系统运行相关参数。">
      <Card style={{ marginBottom: 16 }}>
        <Typography.Title level={5} style={{ marginTop: 0 }}>
          运行时授权
        </Typography.Title>
        {licenseLoading ? (
          <Skeleton active paragraph={{ rows: 4 }} />
        ) : (
          <Space direction="vertical" size={16} className="w-full">
            <Descriptions
              bordered
              size="small"
              column={{ xs: 1, md: 2 }}
            >
              <Descriptions.Item label="授权状态">
                {licenseStatus?.enabled ? (
                  licenseStatus.valid ? (
                    <Tag color="success">已授权</Tag>
                  ) : (
                    <Tag color="error">未授权</Tag>
                  )
                ) : (
                  <Tag>未启用</Tag>
                )}
              </Descriptions.Item>
              <Descriptions.Item label="授权码">
                {licenseStatus?.license_key || '-'}
              </Descriptions.Item>
              <Descriptions.Item label="授权日期">
                {formatDateTime(licenseStatus?.authorized_at ?? licenseStatus?.last_success_at)}
              </Descriptions.Item>
              <Descriptions.Item label="授权到期">
                {formatDateTime(licenseStatus?.expires_at)}
              </Descriptions.Item>
              <Descriptions.Item label="授权域名">
                {licenseStatus?.domain || '-'}
              </Descriptions.Item>
              <Descriptions.Item label="站点地址">
                {licenseStatus?.site_url || '-'}
              </Descriptions.Item>
            </Descriptions>

            <Alert
              type={licenseStatus?.valid ? 'success' : 'warning'}
              showIcon
              message={licenseStatus?.message || '授权状态未知'}
            />

            <Form<LicenseDomainFormValues>
              form={licenseDomainForm}
              layout="vertical"
              onFinish={(values) => void handleUpdateLicenseDomain(values)}
            >
              <Row gutter={16}>
                <Col xs={24} md={10}>
                  <Form.Item label="更换授权域名" name="domain">
                    <Input
                      placeholder="panel.example.com"
                      disabled={!licenseStatus?.enabled}
                    />
                  </Form.Item>
                </Col>
                <Col xs={24} md={10}>
                  <Form.Item label="站点地址（可选）" name="site_url">
                    <Input
                      placeholder="https://panel.example.com"
                      disabled={!licenseStatus?.enabled}
                    />
                  </Form.Item>
                </Col>
                <Col xs={24} md={4}>
                  <Form.Item label=" ">
                    <Button
                      type="primary"
                      htmlType="submit"
                      loading={updatingLicenseDomain}
                      disabled={!licenseStatus?.enabled}
                      className="w-full"
                    >
                      更换域名
                    </Button>
                  </Form.Item>
                </Col>
              </Row>
            </Form>
          </Space>
        )}
      </Card>

      <Card>
        {loading ? (
          <Skeleton active paragraph={{ rows: 14 }} />
        ) : (
          <Form<SystemConfigFormValues>
            form={form}
            layout="vertical"
            onFinish={(values) => void handleSubmit(values)}
          >
            <Typography.Title level={5} style={{ marginTop: 0 }}>
              基础配置
            </Typography.Title>
            <Row gutter={16}>
              <Col xs={24} md={12}>
                <Form.Item
                  label="站点名称"
                  name="site_name"
                  rules={[{ required: true, message: '请输入站点名称' }]}
                >
                  <Input placeholder="NodePass Panel" />
                </Form.Item>
              </Col>
              <Col xs={24} md={12}>
                <Form.Item
                  label="站点地址 (用于 Telegram SSO)"
                  name="site_url"
                  rules={[
                    {
                      validator(_, value) {
                        const normalized = String(value ?? '').trim()
                        if (!normalized) {
                          return Promise.resolve()
                        }
                        try {
                          const target = normalized.includes('://') ? normalized : `https://${normalized}`
                          const parsed = new URL(target)
                          if (!parsed.hostname) {
                            return Promise.reject(new Error('站点地址格式不正确'))
                          }
                          return Promise.resolve()
                        } catch {
                          return Promise.reject(new Error('站点地址格式不正确'))
                        }
                      },
                    },
                  ]}
                >
                  <Input placeholder="https://panel.example.com" />
                </Form.Item>
              </Col>
              <Col xs={24} md={12}>
                <Form.Item
                  label="默认 VIP 等级"
                  name="default_vip_level"
                  rules={[{ required: true, message: '请输入默认 VIP 等级' }]}
                >
                  <InputNumber min={0} precision={0} className="w-full" />
                </Form.Item>
              </Col>
            </Row>
            <Form.Item
              label="允许注册"
              name="register_enabled"
              valuePropName="checked"
            >
              <Switch />
            </Form.Item>

            <Divider />
            <Typography.Title level={5}>Telegram 配置</Typography.Title>
            <Row gutter={16}>
              <Col xs={24} md={12}>
                <Form.Item label="Telegram Bot Token" name="telegram_bot_token">
                  <Input.Password placeholder="可选" />
                </Form.Item>
              </Col>
              <Col xs={24} md={12}>
                <Form.Item label="Telegram Bot Username" name="telegram_bot_username">
                  <Input placeholder="nodepass_bot" />
                </Form.Item>
              </Col>
            </Row>

            <Divider />
            <Typography.Title level={5}>SMTP 邮件配置</Typography.Title>
            <Form.Item
              label="启用 SMTP"
              name="smtp_enabled"
              valuePropName="checked"
            >
              <Switch />
            </Form.Item>
            <Row gutter={16}>
              <Col xs={24} md={12}>
                <Form.Item
                  label="SMTP 主机"
                  name="smtp_host"
                  rules={[
                    ({ getFieldValue }) => ({
                      validator(_, value) {
                        if (!getFieldValue('smtp_enabled')) {
                          return Promise.resolve()
                        }
                        if (String(value ?? '').trim()) {
                          return Promise.resolve()
                        }
                        return Promise.reject(new Error('启用 SMTP 时必须填写主机'))
                      },
                    }),
                  ]}
                >
                  <Input placeholder="smtp.example.com" />
                </Form.Item>
              </Col>
              <Col xs={24} md={6}>
                <Form.Item
                  label="SMTP 端口"
                  name="smtp_port"
                  rules={[
                    ({ getFieldValue }) => ({
                      validator(_, value) {
                        if (!getFieldValue('smtp_enabled')) {
                          return Promise.resolve()
                        }
                        if (typeof value === 'number' && value >= 1 && value <= 65535) {
                          return Promise.resolve()
                        }
                        return Promise.reject(new Error('请输入 1-65535 的端口'))
                      },
                    }),
                  ]}
                >
                  <InputNumber min={1} max={65535} precision={0} className="w-full" />
                </Form.Item>
              </Col>
              <Col xs={24} md={6}>
                <Form.Item label="加密方式" name="smtp_encryption">
                  <Select
                    options={[
                      { label: 'STARTTLS', value: 'starttls' },
                      { label: 'SSL/TLS', value: 'ssl' },
                      { label: '无加密', value: 'none' },
                    ]}
                  />
                </Form.Item>
              </Col>
            </Row>
            <Row gutter={16}>
              <Col xs={24} md={12}>
                <Form.Item label="SMTP 用户名" name="smtp_username">
                  <Input placeholder="可选" />
                </Form.Item>
              </Col>
              <Col xs={24} md={12}>
                <Form.Item label="SMTP 密码" name="smtp_password">
                  <Input.Password placeholder="可选" />
                </Form.Item>
              </Col>
            </Row>
            <Row gutter={16}>
              <Col xs={24} md={12}>
                <Form.Item
                  label="发件邮箱"
                  name="smtp_from_email"
                  rules={[
                    ({ getFieldValue }) => ({
                      validator(_, value) {
                        if (!getFieldValue('smtp_enabled')) {
                          return Promise.resolve()
                        }
                        const normalized = String(value ?? '').trim()
                        if (!normalized) {
                          return Promise.reject(new Error('启用 SMTP 时必须填写发件邮箱'))
                        }
                        const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/
                        if (!emailRegex.test(normalized)) {
                          return Promise.reject(new Error('发件邮箱格式不正确'))
                        }
                        return Promise.resolve()
                      },
                    }),
                  ]}
                >
                  <Input placeholder="no-reply@example.com" />
                </Form.Item>
              </Col>
              <Col xs={24} md={12}>
                <Form.Item label="发件人名称" name="smtp_from_name">
                  <Input placeholder="NodePass" />
                </Form.Item>
              </Col>
            </Row>
            <Row gutter={16}>
              <Col xs={24} md={12}>
                <Form.Item
                  label="Reply-To（可选）"
                  name="smtp_reply_to"
                  rules={[
                    {
                      type: 'email',
                      message: 'Reply-To 邮箱格式不正确',
                    },
                  ]}
                >
                  <Input placeholder="support@example.com" />
                </Form.Item>
              </Col>
              <Col xs={24} md={12}>
                <Form.Item label="跳过证书校验" name="smtp_skip_verify" valuePropName="checked">
                  <Switch />
                </Form.Item>
              </Col>
            </Row>

            <Divider />
            <Typography.Title level={5}>运行参数</Typography.Title>
            <Row gutter={16}>
              <Col xs={24} md={12}>
                <Form.Item
                  label="心跳超时时间 (秒)"
                  name="heartbeat_timeout_seconds"
                  rules={[{ required: true, message: '请输入心跳超时秒数' }]}
                >
                  <InputNumber min={30} precision={0} className="w-full" />
                </Form.Item>
              </Col>
              <Col xs={24} md={12}>
                <Form.Item
                  label="流量统计间隔 (秒)"
                  name="traffic_stats_interval_seconds"
                  rules={[{ required: true, message: '请输入统计间隔秒数' }]}
                >
                  <InputNumber min={10} precision={0} className="w-full" />
                </Form.Item>
              </Col>
            </Row>

            <Space>
              <Button type="primary" htmlType="submit" loading={saving}>
                保存
              </Button>
              <Button onClick={() => void loadConfig()}>重置</Button>
            </Space>

            <Typography.Paragraph type="secondary" style={{ marginTop: 12 }}>
              配置会统一批量写入 `system_config`，SMTP 启用后将用于发送邮箱验证码。
            </Typography.Paragraph>
          </Form>
        )}
      </Card>
    </PageContainer>
  )
}

export default SystemConfig
