import {
  CheckCircleOutlined,
  CrownOutlined,
  GiftOutlined,
} from '@ant-design/icons'
import {
  Alert,
  Button,
  Card,
  Col,
  Descriptions,
  Form,
  Input,
  List,
  Result,
  Row,
  Skeleton,
  Space,
  Tag,
  Typography,
  message,
} from 'antd'
import dayjs from 'dayjs'
import duration from 'dayjs/plugin/duration'
import { useEffect, useMemo, useState } from 'react'

import PageContainer from '../../components/common/PageContainer'
import { usePageTitle } from '../../hooks/usePageTitle'
import { benefitCodeApi, vipApi } from '../../services/api'
import type { BenefitCodeRedeemResult, VipLevelRecord, VipMyLevelResult } from '../../types'
import { getErrorMessage } from '../../utils/error'
import { formatDateTime, formatTraffic } from '../../utils/format'

dayjs.extend(duration)

type RedeemFormValues = {
  code: string
}

const formatBandwidth = (value?: number): string => {
  if (typeof value !== 'number') {
    return '-'
  }
  if (value < 0) {
    return '不限'
  }
  return `${value} Mbps`
}

const formatRuleLimit = (value?: number): string => {
  if (typeof value !== 'number') {
    return '-'
  }
  return value < 0 ? '不限' : `${value}`
}

const formatCountdown = (expiresAt?: string | null): string => {
  if (!expiresAt) {
    return '长期有效'
  }

  const now = dayjs()
  const expireAt = dayjs(expiresAt)
  if (!expireAt.isValid()) {
    return '-'
  }
  if (expireAt.isBefore(now)) {
    return '已过期'
  }

  const diff = dayjs.duration(expireAt.diff(now))
  const days = Math.floor(diff.asDays())
  const hours = diff.hours()
  return `${days} 天 ${hours} 小时`
}

const buildRights = (level?: VipLevelRecord | null): Array<{ label: string; value: string }> => {
  if (!level) {
    return []
  }

  return [
    { label: '流量配额', value: formatTraffic(level.traffic_quota) },
    { label: '最大规则数', value: formatRuleLimit(level.max_rules) },
    { label: '带宽限制', value: formatBandwidth(level.max_bandwidth) },
    {
      label: '自托管配额',
      value: `${level.max_self_hosted_entry_nodes}/${level.max_self_hosted_exit_nodes} (入口/出口)`,
    },
    { label: '流量倍率', value: `${(level.traffic_multiplier ?? 1).toFixed(2)}x` },
  ]
}

const VipCenter = () => {
  usePageTitle('VIP 中心')

  const [loading, setLoading] = useState<boolean>(true)
  const [redeeming, setRedeeming] = useState<boolean>(false)
  const [levels, setLevels] = useState<VipLevelRecord[]>([])
  const [myLevel, setMyLevel] = useState<VipMyLevelResult | null>(null)
  const [redeemResult, setRedeemResult] = useState<BenefitCodeRedeemResult | null>(null)
  const [form] = Form.useForm<RedeemFormValues>()

  const loadData = async (): Promise<void> => {
    setLoading(true)
    try {
      const [levelsResult, myLevelResult] = await Promise.all([
        vipApi.levels(),
        vipApi.myLevel(),
      ])
      setLevels(levelsResult.list ?? [])
      setMyLevel(myLevelResult)
    } catch (error) {
      message.error(getErrorMessage(error, 'VIP 信息加载失败'))
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    void loadData()
  }, [])

  const currentLevel = useMemo(
    () =>
      myLevel?.level_detail ??
      levels.find((level) => level.level === (myLevel?.vip_level ?? 0)) ??
      null,
    [levels, myLevel],
  )

  const currentRights = useMemo(() => buildRights(currentLevel), [currentLevel])

  const handleRedeem = async (values: RedeemFormValues): Promise<void> => {
    setRedeeming(true)
    try {
      const result = await benefitCodeApi.redeem(values.code)
      setRedeemResult(result)
      message.success('权益码兑换成功')
      form.resetFields()
      await loadData()
    } catch (error) {
      message.error(getErrorMessage(error, '权益码兑换失败'))
    } finally {
      setRedeeming(false)
    }
  }

  return (
    <PageContainer title="VIP 中心" description="查看等级权益并兑换权益码。">
      <Space direction="vertical" size={16} className="w-full">
        <Card>
          {loading ? (
            <Skeleton active paragraph={{ rows: 4 }} />
          ) : (
            <Row gutter={[16, 16]}>
              <Col xs={24} lg={12}>
                <Card
                  bordered={false}
                  style={{ background: 'linear-gradient(135deg, #0f172a, #1d4ed8)' }}
                >
                  <Space direction="vertical" size={8}>
                    <Tag color="gold" icon={<CrownOutlined />}>
                      当前 VIP
                    </Tag>
                    <Typography.Title level={4} style={{ margin: 0, color: '#fff' }}>
                      Lv.{myLevel?.vip_level ?? 0} {currentLevel?.name ?? '未命名等级'}
                    </Typography.Title>
                    <Typography.Text style={{ color: '#dbeafe' }}>
                      到期时间：{formatDateTime(myLevel?.vip_expires_at)}
                    </Typography.Text>
                    <Typography.Text style={{ color: '#dbeafe' }}>
                      倒计时：{formatCountdown(myLevel?.vip_expires_at)}
                    </Typography.Text>
                  </Space>
                </Card>
              </Col>

              <Col xs={24} lg={12}>
                <Descriptions title="权益列表" column={1} size="small">
                  {currentRights.map((item) => (
                    <Descriptions.Item key={item.label} label={item.label}>
                      {item.value}
                    </Descriptions.Item>
                  ))}
                </Descriptions>
              </Col>
            </Row>
          )}
        </Card>

        <Card title="可用 VIP 等级">
          {loading ? (
            <Skeleton active paragraph={{ rows: 6 }} />
          ) : (
            <Row gutter={[16, 16]}>
              {levels.map((level) => {
                const isCurrent = level.level === (myLevel?.vip_level ?? -1)
                const rights = buildRights(level)
                return (
                  <Col xs={24} md={12} xl={8} key={level.id}>
                    <Card
                      size="small"
                      style={
                        isCurrent
                          ? {
                              borderColor: '#1677ff',
                              boxShadow: '0 0 0 1px rgba(22,119,255,.25)',
                            }
                          : undefined
                      }
                    >
                      <Space direction="vertical" size={8} className="w-full">
                        <Space>
                          <Typography.Title level={5} style={{ margin: 0 }}>
                            Lv.{level.level} {level.name}
                          </Typography.Title>
                          {isCurrent ? <Tag color="blue">当前等级</Tag> : null}
                        </Space>

                        <Typography.Text type="secondary">
                          价格：{level.price == null ? '联系管理员' : `¥${level.price}`}
                        </Typography.Text>

                        <List
                          size="small"
                          dataSource={rights}
                          renderItem={(item) => (
                            <List.Item style={{ paddingInline: 0 }}>
                              <Space size={6}>
                                <CheckCircleOutlined style={{ color: '#16a34a' }} />
                                <Typography.Text>
                                  {item.label}: {item.value}
                                </Typography.Text>
                              </Space>
                            </List.Item>
                          )}
                        />
                      </Space>
                    </Card>
                  </Col>
                )
              })}
            </Row>
          )}
        </Card>

        <Card title="权益码兑换" extra={<GiftOutlined />}>
          <Space direction="vertical" size={16} className="w-full">
            <Alert
              showIcon
              type="info"
              message="输入权益码后将自动升级或延长你的 VIP 权益。"
            />

            <Form<RedeemFormValues>
              form={form}
              layout="inline"
              onFinish={(values) => void handleRedeem(values)}
            >
              <Form.Item
                name="code"
                rules={[
                  { required: true, message: '请输入权益码' },
                  { pattern: /^NP-[A-Z0-9]{4}-[A-Z0-9]{4}-[A-Z0-9]{4}$/, message: '权益码格式无效' },
                ]}
              >
                <Input
                  style={{ minWidth: 320 }}
                  placeholder="NP-XXXX-XXXX-XXXX"
                  onChange={(event) => {
                    form.setFieldValue('code', event.target.value.toUpperCase())
                  }}
                />
              </Form.Item>
              <Form.Item>
                <Button type="primary" htmlType="submit" loading={redeeming}>
                  兑换
                </Button>
              </Form.Item>
            </Form>

            {redeemResult ? (
              <Result
                status="success"
                title="兑换成功"
                subTitle={`已应用到 Lv.${redeemResult.applied_level}，到期时间：${formatDateTime(redeemResult.vip_expires_at)}`}
                extra={[
                  <Button key="close" onClick={() => setRedeemResult(null)}>
                    关闭
                  </Button>,
                ]}
              />
            ) : null}
          </Space>
        </Card>
      </Space>
    </PageContainer>
  )
}

export default VipCenter
