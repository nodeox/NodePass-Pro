import {
  Button,
  Card,
  Form,
  Input,
  InputNumber,
  Skeleton,
  Space,
  Switch,
  Typography,
  message,
} from 'antd'
import { useEffect, useState } from 'react'

import PageContainer from '../../components/common/PageContainer'
import { usePageTitle } from '../../hooks/usePageTitle'
import { systemApi } from '../../services/api'
import { getErrorMessage } from '../../utils/error'

type SystemConfigFormValues = {
  site_name: string
  register_enabled: boolean
  default_vip_level: number
  telegram_bot_token: string
  telegram_bot_username: string
  heartbeat_timeout_seconds: number
  traffic_stats_interval_seconds: number
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

const SystemConfig = () => {
  usePageTitle('系统配置')

  const [form] = Form.useForm<SystemConfigFormValues>()
  const [loading, setLoading] = useState<boolean>(true)
  const [saving, setSaving] = useState<boolean>(false)

  const loadConfig = async (): Promise<void> => {
    setLoading(true)
    try {
      const config = await systemApi.config()
      form.setFieldsValue({
        site_name: config.site_name ?? 'NodePass Panel',
        register_enabled: parseBoolean(config.register_enabled),
        default_vip_level: parseNumber(config.default_vip_level, 0),
        telegram_bot_token: config.telegram_bot_token ?? '',
        telegram_bot_username: config.telegram_bot_username ?? '',
        heartbeat_timeout_seconds: parseNumber(config.heartbeat_timeout_seconds, 180),
        traffic_stats_interval_seconds: parseNumber(
          config.traffic_stats_interval_seconds,
          300,
        ),
      })
    } catch (error) {
      message.error(getErrorMessage(error, '系统配置加载失败'))
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    void loadConfig()
  }, [])

  const handleSubmit = async (values: SystemConfigFormValues): Promise<void> => {
    setSaving(true)
    try {
      const payloadEntries: Array<{ key: string; value: string }> = [
        { key: 'site_name', value: values.site_name.trim() },
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
      ]

      for (const item of payloadEntries) {
        await systemApi.updateConfig(item)
      }

      message.success('系统配置保存成功')
      await loadConfig()
    } catch (error) {
      message.error(getErrorMessage(error, '系统配置保存失败'))
    } finally {
      setSaving(false)
    }
  }

  return (
    <PageContainer title="系统配置" description="维护系统运行相关参数。">
      <Card>
        {loading ? (
          <Skeleton active paragraph={{ rows: 8 }} />
        ) : (
          <Form<SystemConfigFormValues>
            form={form}
            layout="vertical"
            onFinish={(values) => void handleSubmit(values)}
          >
            <Form.Item
              label="站点名称"
              name="site_name"
              rules={[{ required: true, message: '请输入站点名称' }]}
            >
              <Input placeholder="NodePass Panel" />
            </Form.Item>

            <Form.Item
              label="注册开关"
              name="register_enabled"
              valuePropName="checked"
            >
              <Switch />
            </Form.Item>

            <Form.Item
              label="默认 VIP 等级"
              name="default_vip_level"
              rules={[{ required: true, message: '请输入默认 VIP 等级' }]}
            >
              <InputNumber min={0} precision={0} className="w-full" />
            </Form.Item>

            <Form.Item label="Telegram Bot Token" name="telegram_bot_token">
              <Input.Password placeholder="可选" />
            </Form.Item>

            <Form.Item label="Telegram Bot Username" name="telegram_bot_username">
              <Input placeholder="@nodepass_bot" />
            </Form.Item>

            <Form.Item
              label="心跳超时时间 (秒)"
              name="heartbeat_timeout_seconds"
              rules={[{ required: true, message: '请输入心跳超时秒数' }]}
            >
              <InputNumber min={30} precision={0} className="w-full" />
            </Form.Item>

            <Form.Item
              label="流量统计间隔 (秒)"
              name="traffic_stats_interval_seconds"
              rules={[{ required: true, message: '请输入统计间隔秒数' }]}
            >
              <InputNumber min={10} precision={0} className="w-full" />
            </Form.Item>

            <Space>
              <Button type="primary" htmlType="submit" loading={saving}>
                保存
              </Button>
              <Button onClick={() => void loadConfig()}>重置</Button>
            </Space>

            <Typography.Paragraph type="secondary" style={{ marginTop: 12 }}>
              配置以键值形式存储，点击保存会逐项写入 `system_config`。
            </Typography.Paragraph>
          </Form>
        )}
      </Card>
    </PageContainer>
  )
}

export default SystemConfig
