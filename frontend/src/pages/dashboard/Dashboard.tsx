import { message } from 'antd'
import axios from 'axios'
import type { EChartsOption } from 'echarts'
import ReactECharts from '../../components/charts/EChartsCore'
import dayjs from 'dayjs'
import { Card, Col, List, Progress, Row, Segmented, Skeleton, Space, Statistic, Tag, Typography } from 'antd'
import { useEffect, useMemo, useState } from 'react'

import { usePageTitle } from '../../hooks/usePageTitle'
import { announcementApi, auditApi, systemApi, trafficApi } from '../../services/api'
import { useAuthStore } from '../../store/auth'
import type { AnnouncementRecord, AuditLogRecord } from '../../types'
import { formatBytes, formatDateTime } from '../../utils/format'

type RangeType = '7d' | '30d'

type TrafficTrendPoint = {
  label: string
  up: number
  down: number
}

type DashboardStats = {
  totalNodes: number
  onlineNodes: number
  runningRules: number
  totalRules: number
  monthlyTrafficUsed: number
  monthlyTrafficQuota: number
  vipLevel: number
  vipExpiresAt: string | null
}

type NodeStatusOverview = {
  online: number
  offline: number
  maintain: number
}

const toNumber = (value: unknown): number => {
  if (typeof value === 'number' && Number.isFinite(value)) {
    return value
  }
  if (typeof value === 'string' && value.trim() !== '') {
    const parsed = Number(value)
    if (Number.isFinite(parsed)) {
      return parsed
    }
  }
  return 0
}

const pickNumber = (record: object, keys: string[]): number => {
  const source = record as Record<string, unknown>
  for (const key of keys) {
    if (key in source) {
      return toNumber(source[key])
    }
  }
  return 0
}

const getRangeDays = (range: RangeType): number => (range === '7d' ? 7 : 30)

const Dashboard = () => {
  usePageTitle('仪表盘')

  const user = useAuthStore((state) => state.user)

  const [range, setRange] = useState<RangeType>('7d')
  const [loading, setLoading] = useState<boolean>(true)
  const [stats, setStats] = useState<DashboardStats>({
    totalNodes: 0,
    onlineNodes: 0,
    runningRules: 0,
    totalRules: 0,
    monthlyTrafficUsed: 0,
    monthlyTrafficQuota: 0,
    vipLevel: 0,
    vipExpiresAt: null,
  })
  const [nodeStatus, setNodeStatus] = useState<NodeStatusOverview>({
    online: 0,
    offline: 0,
    maintain: 0,
  })
  const [trafficTrend, setTrafficTrend] = useState<TrafficTrendPoint[]>([])
  const [announcements, setAnnouncements] = useState<AnnouncementRecord[]>([])
  const [auditLogs, setAuditLogs] = useState<AuditLogRecord[]>([])
  const [auditForbidden, setAuditForbidden] = useState<boolean>(false)

  useEffect(() => {
    let cancelled = false

    const loadTrafficTrend = async (selectedRange: RangeType): Promise<TrafficTrendPoint[]> => {
      const days = getRangeDays(selectedRange)
      const dayList = Array.from({ length: days }, (_, index) =>
        dayjs()
          .subtract(days - 1 - index, 'day')
          .startOf('day'),
      )

      const usageList = await Promise.all(
        dayList.map((day) =>
          trafficApi.usage({
            start_time: day.startOf('day').toISOString(),
            end_time: day.endOf('day').toISOString(),
          }),
        ),
      )

      return dayList.map((day, index) => {
        const usage = usageList[index]
        return {
          label: day.format('MM-DD'),
          up: pickNumber(usage, ['total_traffic_in', 'traffic_in']),
          down: pickNumber(usage, ['total_traffic_out', 'traffic_out']),
        }
      })
    }

    const loadDashboard = async (): Promise<void> => {
      setLoading(true)
      try {
        const [systemStatsRaw, monthlyUsageRaw, announcementResult, trendResult] =
          await Promise.all([
            systemApi.stats(),
            trafficApi.usage({
              start_time: dayjs().startOf('month').toISOString(),
              end_time: dayjs().endOf('day').toISOString(),
            }),
            announcementApi.list(true),
            loadTrafficTrend(range),
          ])

        let auditLogItems: AuditLogRecord[] = []
        let isAuditForbidden = false
        try {
          const auditResult = await auditApi.list({
            page: 1,
            pageSize: 5,
          })
          auditLogItems = auditResult.list ?? []
        } catch (error) {
          if (axios.isAxiosError(error) && error.response?.status === 403) {
            isAuditForbidden = true
          } else {
            message.warning('最近操作日志加载失败')
          }
        }

        if (cancelled) {
          return
        }

        const onlineNodes = pickNumber(systemStatsRaw, ['online_nodes'])
        const totalNodes = pickNumber(systemStatsRaw, ['total_nodes', 'nodes_total']) || onlineNodes
        const runningRules = pickNumber(systemStatsRaw, ['running_rules'])
        const totalRules = pickNumber(systemStatsRaw, ['total_rules', 'rules_total']) || runningRules
        const offlineNodes = pickNumber(systemStatsRaw, ['offline_nodes'])
        const maintainNodes = pickNumber(systemStatsRaw, ['maintain_nodes', 'maintenance_nodes'])
        const monthlyUsage = pickNumber(monthlyUsageRaw, [
          'total_calculated_traffic',
          'calculated_traffic',
        ])

        setStats({
          totalNodes,
          onlineNodes,
          runningRules,
          totalRules,
          monthlyTrafficUsed: monthlyUsage,
          monthlyTrafficQuota: Math.max(user?.traffic_quota ?? 0, 0),
          vipLevel: user?.vip_level ?? 0,
          vipExpiresAt: user?.vip_expires_at ?? null,
        })
        setNodeStatus({
          online: onlineNodes,
          offline: offlineNodes,
          maintain: maintainNodes,
        })
        setTrafficTrend(trendResult)
        setAnnouncements((announcementResult.list ?? []).slice(0, 3))
        setAuditLogs(auditLogItems.slice(0, 5))
        setAuditForbidden(isAuditForbidden)
      } catch {
        if (!cancelled) {
          message.error('仪表盘数据加载失败')
          setAuditLogs([])
          setAnnouncements([])
          setTrafficTrend([])
        }
      } finally {
        if (!cancelled) {
          setLoading(false)
        }
      }
    }

    void loadDashboard()
    return () => {
      cancelled = true
    }
  }, [range, user?.traffic_quota, user?.vip_expires_at, user?.vip_level])

  const usagePercent = useMemo(() => {
    if (stats.monthlyTrafficQuota <= 0) {
      return 0
    }
    return Math.min(
      100,
      Math.round((stats.monthlyTrafficUsed / stats.monthlyTrafficQuota) * 10000) / 100,
    )
  }, [stats.monthlyTrafficQuota, stats.monthlyTrafficUsed])

  const trafficOption: EChartsOption = {
    tooltip: { trigger: 'axis' },
    xAxis: {
      type: 'category',
      data: trafficTrend.map((item) => item.label),
    },
    yAxis: {
      type: 'value',
      axisLabel: {
        formatter: (value: number) => formatBytes(value),
      },
    },
    series: [
      {
        name: '上行',
        type: 'line',
        smooth: true,
        data: trafficTrend.map((item) => item.up),
      },
      {
        name: '下行',
        type: 'line',
        smooth: true,
        data: trafficTrend.map((item) => item.down),
        areaStyle: {},
      },
    ],
  }

  const nodeStatusOption: EChartsOption = {
    tooltip: { trigger: 'item' },
    legend: { bottom: 0 },
    series: [
      {
        type: 'pie',
        radius: ['40%', '70%'],
        data: [
          { value: nodeStatus.online, name: '在线' },
          { value: nodeStatus.offline, name: '离线' },
          { value: nodeStatus.maintain, name: '维护' },
        ],
      },
    ],
  }

  const renderSkeletonCard = () => (
    <Card className="h-full">
      <Skeleton active paragraph={{ rows: 2 }} />
    </Card>
  )

  return (
    <Space direction="vertical" size={16} className="w-full">
      <Row gutter={16}>
        <Col xs={24} sm={12} lg={6}>
          {loading ? (
            renderSkeletonCard()
          ) : (
            <Card className="h-full">
              <Statistic title="总节点数" value={stats.totalNodes} />
              <Typography.Text type="secondary">
                在线 {stats.onlineNodes}
              </Typography.Text>
            </Card>
          )}
        </Col>

        <Col xs={24} sm={12} lg={6}>
          {loading ? (
            renderSkeletonCard()
          ) : (
            <Card className="h-full">
              <Statistic title="运行中规则" value={stats.runningRules} />
              <Typography.Text type="secondary">
                总计 {stats.totalRules}
              </Typography.Text>
            </Card>
          )}
        </Col>

        <Col xs={24} sm={12} lg={6}>
          {loading ? (
            renderSkeletonCard()
          ) : (
            <Card className="h-full">
              <Statistic
                title="本月流量使用"
                value={formatBytes(stats.monthlyTrafficUsed)}
              />
              <Typography.Text type="secondary">
                配额 {formatBytes(stats.monthlyTrafficQuota)}
              </Typography.Text>
              <Progress
                percent={usagePercent}
                status={usagePercent >= 100 ? 'exception' : 'active'}
                size="small"
                style={{ marginTop: 8 }}
              />
            </Card>
          )}
        </Col>

        <Col xs={24} sm={12} lg={6}>
          {loading ? (
            renderSkeletonCard()
          ) : (
            <Card className="h-full">
              <Statistic title="VIP 等级" value={`Lv.${stats.vipLevel}`} />
              <Typography.Text type="secondary">
                到期 {formatDateTime(stats.vipExpiresAt)}
              </Typography.Text>
            </Card>
          )}
        </Col>
      </Row>

      <Row gutter={16}>
        <Col xs={24} lg={16}>
          <Card
            title="流量趋势图"
            extra={
              <Segmented<RangeType>
                value={range}
                options={[
                  { label: '最近7天', value: '7d' },
                  { label: '最近30天', value: '30d' },
                ]}
                onChange={(value) => setRange(value)}
              />
            }
          >
            {loading ? (
              <Skeleton active paragraph={{ rows: 10 }} />
            ) : (
              <ReactECharts option={trafficOption} style={{ height: 320 }} />
            )}
          </Card>
        </Col>

        <Col xs={24} lg={8}>
          <Card title="节点状态概览">
            {loading ? (
              <Skeleton active paragraph={{ rows: 10 }} />
            ) : (
              <ReactECharts option={nodeStatusOption} style={{ height: 320 }} />
            )}
          </Card>
        </Col>
      </Row>

      <Row gutter={16}>
        <Col xs={24} lg={12}>
          <Card title="最近公告">
            {loading ? (
              <Skeleton active paragraph={{ rows: 6 }} />
            ) : announcements.length === 0 ? (
              <Typography.Text type="secondary">暂无公告</Typography.Text>
            ) : (
              <List
                dataSource={announcements}
                renderItem={(item) => (
                  <List.Item>
                    <List.Item.Meta
                      title={
                        <Space>
                          <Typography.Text>{item.title}</Typography.Text>
                          <Tag>{item.type}</Tag>
                        </Space>
                      }
                      description={formatDateTime(item.created_at)}
                    />
                  </List.Item>
                )}
              />
            )}
          </Card>
        </Col>

        <Col xs={24} lg={12}>
          <Card title="最近操作日志">
            {loading ? (
              <Skeleton active paragraph={{ rows: 8 }} />
            ) : auditForbidden ? (
              <Typography.Text type="secondary">
                当前账号无权限查看审计日志
              </Typography.Text>
            ) : auditLogs.length === 0 ? (
              <Typography.Text type="secondary">暂无操作日志</Typography.Text>
            ) : (
              <List
                dataSource={auditLogs}
                renderItem={(item) => (
                  <List.Item>
                    <List.Item.Meta
                      title={
                        <Typography.Text>
                          {item.action}
                          {item.resource_type ? ` / ${item.resource_type}` : ''}
                        </Typography.Text>
                      }
                      description={formatDateTime(item.created_at)}
                    />
                  </List.Item>
                )}
              />
            )}
          </Card>
        </Col>
      </Row>
    </Space>
  )
}

export default Dashboard
