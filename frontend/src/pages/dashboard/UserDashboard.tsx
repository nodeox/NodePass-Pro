import type { EChartsOption } from 'echarts'
import ReactECharts from 'echarts-for-react'
import dayjs from 'dayjs'
import {
  Card,
  Col,
  List,
  Progress,
  Row,
  Segmented,
  Skeleton,
  Space,
  Statistic,
  Tag,
  Typography,
  message,
} from 'antd'
import { useEffect, useMemo, useState } from 'react'

import { usePageTitle } from '../../hooks/usePageTitle'
import { announcementApi, trafficApi } from '../../services/api'
import { nodeGroupApi, tunnelApi } from '../../services/nodeGroupApi'
import { useAuthStore } from '../../store/auth'
import type { AnnouncementRecord } from '../../types'
import type { NodeGroup, NodeInstance } from '../../types/nodeGroup'
import { formatBytes, formatDateTime } from '../../utils/format'

type RangeType = '7d' | '30d'

type TrafficTrendPoint = {
  label: string
  up: number
  down: number
}

type UserDashboardStats = {
  totalNodes: number
  onlineNodes: number
  runningTunnels: number
  totalTunnels: number
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

const getRangeDays = (range: RangeType): number => (range === '7d' ? 7 : 30)

const collectNodeInstances = (groups: NodeGroup[]): NodeInstance[] =>
  groups.flatMap((group) => group.node_instances ?? [])

const normalizeNodeStatus = (nodes: NodeInstance[]): NodeStatusOverview => {
  return nodes.reduce<NodeStatusOverview>(
    (summary, node) => {
      if (node.status === 'online') {
        return { ...summary, online: summary.online + 1 }
      }
      if (node.status === 'maintain') {
        return { ...summary, maintain: summary.maintain + 1 }
      }
      return { ...summary, offline: summary.offline + 1 }
    },
    { online: 0, offline: 0, maintain: 0 },
  )
}

const announcementTypeMeta: Record<
  AnnouncementRecord['type'],
  { color: string; label: string }
> = {
  info: { color: 'blue', label: '通知' },
  warning: { color: 'orange', label: '警告' },
  error: { color: 'red', label: '错误' },
  success: { color: 'green', label: '成功' },
}

const UserDashboard = () => {
  usePageTitle('用户仪表盘')

  const user = useAuthStore((state) => state.user)

  const [range, setRange] = useState<RangeType>('7d')
  const [loading, setLoading] = useState<boolean>(true)
  const [stats, setStats] = useState<UserDashboardStats>({
    totalNodes: 0,
    onlineNodes: 0,
    runningTunnels: 0,
    totalTunnels: 0,
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

      return dayList.map((day, index) => ({
        label: day.format('MM-DD'),
        up: usageList[index]?.total_traffic_in ?? 0,
        down: usageList[index]?.total_traffic_out ?? 0,
      }))
    }

    const loadDashboard = async (): Promise<void> => {
      setLoading(true)
      try {
        const [groupResult, tunnelResult, quotaResult, announcementResult, trendResult] =
          await Promise.all([
            nodeGroupApi.list({ page: 1, page_size: 500 }),
            tunnelApi.list({ page: 1, page_size: 500 }),
            trafficApi.quota(),
            announcementApi.list(true),
            loadTrafficTrend(range),
          ])

        if (cancelled) {
          return
        }

        const groups = groupResult.items ?? []
        const nodes = collectNodeInstances(groups)
        const tunnels = tunnelResult.items ?? []
        const runningTunnels = tunnels.filter((item) => item.status === 'running').length
        const nodeSummary = normalizeNodeStatus(nodes)

        setStats({
          totalNodes: nodes.length,
          onlineNodes: nodeSummary.online,
          runningTunnels,
          totalTunnels: tunnels.length,
          monthlyTrafficUsed:
            quotaResult.traffic_used ?? quotaResult.trafficUsed ?? 0,
          monthlyTrafficQuota:
            quotaResult.traffic_quota ?? quotaResult.trafficQuota ?? 0,
          vipLevel: user?.vip_level ?? user?.vipLevel ?? 0,
          vipExpiresAt: user?.vip_expires_at ?? user?.vipExpiresAt ?? null,
        })
        setNodeStatus(nodeSummary)
        setTrafficTrend(trendResult)
        setAnnouncements((announcementResult.list ?? []).slice(0, 3))
      } catch {
        if (!cancelled) {
          message.error('用户仪表盘数据加载失败')
          setTrafficTrend([])
          setAnnouncements([])
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
  }, [range, user?.vipLevel, user?.vipExpiresAt, user?.vip_expires_at, user?.vip_level])

  const usagePercent = useMemo(() => {
    if (stats.monthlyTrafficQuota <= 0) {
      return 0
    }
    return Math.min(
      100,
      Math.round((stats.monthlyTrafficUsed / stats.monthlyTrafficQuota) * 10000) /
        100,
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
              <Typography.Text type="secondary">在线 {stats.onlineNodes}</Typography.Text>
            </Card>
          )}
        </Col>

        <Col xs={24} sm={12} lg={6}>
          {loading ? (
            renderSkeletonCard()
          ) : (
            <Card className="h-full">
              <Statistic title="运行中隧道" value={stats.runningTunnels} />
              <Typography.Text type="secondary">总计 {stats.totalTunnels}</Typography.Text>
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

      <Card title="最近公告">
        {loading ? (
          <Skeleton active paragraph={{ rows: 4 }} />
        ) : (
          <List
            dataSource={announcements}
            locale={{ emptyText: '暂无公告' }}
            renderItem={(item) => (
              <List.Item>
                <List.Item.Meta
                  title={
                    <Space size={8}>
                      <Tag color={announcementTypeMeta[item.type].color}>
                        {announcementTypeMeta[item.type].label}
                      </Tag>
                      <Typography.Text>{item.title}</Typography.Text>
                    </Space>
                  }
                  description={item.content}
                />
              </List.Item>
            )}
          />
        )}
      </Card>
    </Space>
  )
}

export default UserDashboard
