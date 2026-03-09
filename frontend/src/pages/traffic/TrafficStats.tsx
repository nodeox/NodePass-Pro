import {
  DownloadOutlined,
  ReloadOutlined,
} from '@ant-design/icons'
import {
  Button,
  Card,
  Col,
  DatePicker,
  Progress,
  Row,
  Segmented,
  Select,
  Skeleton,
  Space,
  Statistic,
  Table,
  Typography,
  message,
} from 'antd'
import type { EChartsOption } from 'echarts'
import ReactECharts from '../../components/charts/EChartsCore'
import dayjs, { type Dayjs } from 'dayjs'
import { useCallback, useEffect, useMemo, useState } from 'react'

import PageContainer from '../../components/common/PageContainer'
import { usePageTitle } from '../../hooks/usePageTitle'
import { trafficApi } from '../../services/api'
import { nodeGroupApi, tunnelApi } from '../../services/nodeGroupApi'
import type { TrafficQuota, TrafficRecordItem } from '../../types'
import type { NodeInstance, Tunnel } from '../../types/nodeGroup'
import { getErrorMessage } from '../../utils/error'
import { formatDateTime, formatTraffic } from '../../utils/format'

type TrendRangeType = '7d' | '30d' | 'custom'
type ChartGroupType = 'all' | 'rule' | 'node'

type TrendPoint = {
  date: string
  up: number
  down: number
}

const getDefaultRecentRange = (days: number): [Dayjs, Dayjs] => [
  dayjs().subtract(days - 1, 'day').startOf('day'),
  dayjs().endOf('day'),
]

const toISORange = (range: [Dayjs, Dayjs]): { start: string; end: string } => ({
  start: range[0].toISOString(),
  end: range[1].toISOString(),
})

const escapeCSVCell = (value: string): string => {
  const normalized = value.replaceAll('"', '""')
  return `"${normalized}"`
}

const toNumber = (value: unknown, fallback = 0): number => {
  if (typeof value === 'number' && Number.isFinite(value)) {
    return value
  }
  if (typeof value === 'string' && value.trim() !== '') {
    const parsed = Number(value)
    if (Number.isFinite(parsed)) {
      return parsed
    }
  }
  return fallback
}

const buildTrendPoints = (
  records: TrafficRecordItem[],
  range: [Dayjs, Dayjs],
): TrendPoint[] => {
  const start = range[0].startOf('day')
  const end = range[1].startOf('day')
  const totalDays = Math.max(end.diff(start, 'day') + 1, 1)

  const map = new Map<string, TrendPoint>()
  for (let index = 0; index < totalDays; index += 1) {
    const current = start.add(index, 'day')
    const key = current.format('YYYY-MM-DD')
    map.set(key, {
      date: current.format('MM-DD'),
      up: 0,
      down: 0,
    })
  }

  records.forEach((record) => {
    const key = dayjs(record.hour).format('YYYY-MM-DD')
    const point = map.get(key)
    if (!point) {
      return
    }
    point.up += record.traffic_in ?? 0
    point.down += record.traffic_out ?? 0
  })

  return Array.from(map.values())
}

const pickMultiplier = (
  records: TrafficRecordItem[],
): { vip: number; node: number; final: number } => {
  if (records.length === 0) {
    return { vip: 1, node: 1, final: 1 }
  }

  let vipTotal = 0
  let nodeTotal = 0
  let finalTotal = 0

  records.forEach((record) => {
    vipTotal += toNumber(record.vip_multiplier, 1)
    nodeTotal += toNumber(record.node_multiplier, 1)
    finalTotal += toNumber(record.final_multiplier, 1)
  })

  return {
    vip: vipTotal / records.length,
    node: nodeTotal / records.length,
    final: finalTotal / records.length,
  }
}

const TrafficStats = () => {
  usePageTitle('流量统计')

  const [bootLoading, setBootLoading] = useState<boolean>(true)
  const [chartLoading, setChartLoading] = useState<boolean>(false)
  const [tableLoading, setTableLoading] = useState<boolean>(false)
  const [exportLoading, setExportLoading] = useState<boolean>(false)

  const [quota, setQuota] = useState<TrafficQuota | null>(null)
  const [tunnels, setTunnels] = useState<Tunnel[]>([])
  const [nodes, setNodes] = useState<NodeInstance[]>([])

  const [trendRangeType, setTrendRangeType] = useState<TrendRangeType>('7d')
  const [trendCustomRange, setTrendCustomRange] = useState<[Dayjs, Dayjs] | null>(
    null,
  )
  const [chartGroupType, setChartGroupType] = useState<ChartGroupType>('all')
  const [chartRuleID, setChartRuleID] = useState<number | undefined>()
  const [chartNodeID, setChartNodeID] = useState<number | undefined>()
  const [trendPoints, setTrendPoints] = useState<TrendPoint[]>([])
  const [multiplier, setMultiplier] = useState<{ vip: number; node: number; final: number }>({
    vip: 1,
    node: 1,
    final: 1,
  })

  const [tableRange, setTableRange] = useState<[Dayjs, Dayjs]>(() =>
    getDefaultRecentRange(7),
  )
  const [tableRuleID, setTableRuleID] = useState<number | undefined>()
  const [tableNodeID, setTableNodeID] = useState<number | undefined>()
  const [tablePage, setTablePage] = useState<number>(1)
  const [tablePageSize, setTablePageSize] = useState<number>(10)
  const [tableTotal, setTableTotal] = useState<number>(0)
  const [tableRecords, setTableRecords] = useState<TrafficRecordItem[]>([])

  const loadAllTunnels = useCallback(async (): Promise<Tunnel[]> => {
    const merged: Tunnel[] = []
    let page = 1
    const pageSize = 200

    while (page <= 20) {
      const result = await tunnelApi.list({ page, page_size: pageSize })
      merged.push(...(result.items ?? []))
      if (merged.length >= (result.total ?? 0)) {
        break
      }
      page += 1
    }

    return merged
  }, [])

  const loadAllNodes = useCallback(async (): Promise<NodeInstance[]> => {
    const merged: NodeInstance[] = []
    let page = 1
    const pageSize = 200

    while (page <= 20) {
      const result = await nodeGroupApi.list({ page, page_size: pageSize })
      const list = result.items ?? []
      list.forEach((group) => {
        merged.push(...(group.node_instances ?? []))
      })
      if (list.length < pageSize) {
        break
      }
      page += 1
    }

    return merged
  }, [])

  const fetchAllTrafficRecords = useCallback(
    async (params: {
      start_time: string
      end_time: string
      rule_id?: number
      node_id?: number
    }): Promise<TrafficRecordItem[]> => {
      const merged: TrafficRecordItem[] = []
      let page = 1
      const pageSize = 200

      while (page <= 50) {
        const result = await trafficApi.records({
          ...params,
          page,
          pageSize,
        })
        merged.push(...(result.list ?? []))
        if (merged.length >= (result.total ?? 0)) {
          break
        }
        page += 1
      }

      return merged
    },
    [],
  )

  const loadBaseData = useCallback(async (): Promise<void> => {
    setBootLoading(true)
    try {
      const [quotaResult, allTunnels, allNodes] = await Promise.all([
        trafficApi.quota(),
        loadAllTunnels(),
        loadAllNodes(),
      ])
      setQuota(quotaResult)
      setTunnels(allTunnels)
      setNodes(allNodes)
    } catch (error) {
      message.error(getErrorMessage(error, '流量统计基础数据加载失败'))
    } finally {
      setBootLoading(false)
    }
  }, [loadAllNodes, loadAllTunnels])

  useEffect(() => {
    void loadBaseData()
  }, [loadBaseData])

  useEffect(() => {
    if (chartGroupType !== 'rule') {
      setChartRuleID(undefined)
    }
    if (chartGroupType !== 'node') {
      setChartNodeID(undefined)
    }
  }, [chartGroupType])

  const trendRange = useMemo<[Dayjs, Dayjs]>(() => {
    if (trendRangeType === 'custom' && trendCustomRange) {
      return trendCustomRange
    }
    return getDefaultRecentRange(trendRangeType === '7d' ? 7 : 30)
  }, [trendCustomRange, trendRangeType])

  const loadTrend = useCallback(async (): Promise<void> => {
    const { start, end } = toISORange(trendRange)
    const params = {
      start_time: start,
      end_time: end,
      rule_id: chartGroupType === 'rule' ? chartRuleID : undefined,
      node_id: chartGroupType === 'node' ? chartNodeID : undefined,
    }

    setChartLoading(true)
    try {
      const records = await fetchAllTrafficRecords(params)
      setTrendPoints(buildTrendPoints(records, trendRange))
      setMultiplier(pickMultiplier(records))
    } catch (error) {
      message.error(getErrorMessage(error, '流量趋势加载失败'))
      setTrendPoints(buildTrendPoints([], trendRange))
      setMultiplier({ vip: 1, node: 1, final: 1 })
    } finally {
      setChartLoading(false)
    }
  }, [chartGroupType, chartNodeID, chartRuleID, fetchAllTrafficRecords, trendRange])

  useEffect(() => {
    if (bootLoading) {
      return
    }
    void loadTrend()
  }, [bootLoading, loadTrend])

  const loadTableRecords = useCallback(
    async (targetPage = tablePage, targetPageSize = tablePageSize): Promise<void> => {
      const { start, end } = toISORange(tableRange)
      setTableLoading(true)
      try {
        const result = await trafficApi.records({
          page: targetPage,
          pageSize: targetPageSize,
          start_time: start,
          end_time: end,
          rule_id: tableRuleID,
          node_id: tableNodeID,
        })

        setTableRecords(result.list ?? [])
        setTableTotal(result.total ?? 0)
        setTablePage(result.page || targetPage)
        setTablePageSize(result.page_size || targetPageSize)
      } catch (error) {
        message.error(getErrorMessage(error, '流量明细加载失败'))
      } finally {
        setTableLoading(false)
      }
    },
    [tableNodeID, tablePage, tablePageSize, tableRange, tableRuleID],
  )

  useEffect(() => {
    if (bootLoading) {
      return
    }
    void loadTableRecords()
  }, [bootLoading, loadTableRecords])

  const usagePercent = useMemo(() => {
    if (!quota || quota.traffic_quota <= 0) {
      return 0
    }
    return Math.max(
      0,
      Math.min(100, Number(((quota.traffic_used / quota.traffic_quota) * 100).toFixed(2))),
    )
  }, [quota])

  const trendOption = useMemo<EChartsOption>(() => {
    const upColor = '#1677ff'
    const downColor = '#52c41a'

    return {
      tooltip: {
        trigger: 'axis',
        formatter: (params) => {
          const list = Array.isArray(params) ? params : [params]
          const date =
            typeof list[0]?.name === 'string' ? list[0].name : ''
          const upValue = toNumber(list.find((item) => item.seriesName === '上行流量')?.data)
          const downValue = toNumber(list.find((item) => item.seriesName === '下行流量')?.data)
          return `${date}<br/>上行流量: ${formatTraffic(upValue)}<br/>下行流量: ${formatTraffic(downValue)}`
        },
      },
      grid: { left: 48, right: 48, top: 36, bottom: 36 },
      legend: {
        data: ['上行流量', '下行流量'],
      },
      xAxis: {
        type: 'category',
        data: trendPoints.map((item) => item.date),
      },
      yAxis: [
        {
          type: 'value',
          name: '上行',
          axisLabel: {
            formatter: (value: number) => formatTraffic(value),
          },
        },
        {
          type: 'value',
          name: '下行',
          axisLabel: {
            formatter: (value: number) => formatTraffic(value),
          },
        },
      ],
      series: [
        {
          name: '上行流量',
          type: 'line',
          smooth: true,
          yAxisIndex: 0,
          data: trendPoints.map((item) => item.up),
          itemStyle: { color: upColor },
          lineStyle: { color: upColor },
        },
        {
          name: '下行流量',
          type: 'line',
          smooth: true,
          yAxisIndex: 1,
          data: trendPoints.map((item) => item.down),
          itemStyle: { color: downColor },
          lineStyle: { color: downColor },
        },
      ],
    }
  }, [trendPoints])

  const ruleOptions = useMemo(
    () =>
      tunnels.map((tunnel) => ({
        value: tunnel.id,
        label: `${tunnel.name} (#${tunnel.id})`,
      })),
    [tunnels],
  )

  const nodeOptions = useMemo(
    () =>
      nodes.map((node) => ({
        value: node.id,
        label: `${node.name} (${node.node_id})`,
      })),
    [nodes],
  )

  const exportRecords = async (): Promise<void> => {
    const { start, end } = toISORange(tableRange)

    setExportLoading(true)
    try {
      const allRecords = await fetchAllTrafficRecords({
        start_time: start,
        end_time: end,
        rule_id: tableRuleID,
        node_id: tableNodeID,
      })

      if (allRecords.length === 0) {
        message.warning('当前筛选条件下无可导出数据')
        return
      }

      const header = ['时间', '隧道名称', '节点', '上行', '下行', '倍率', '计费流量']
      const rows = allRecords.map((record) => [
        formatDateTime(record.hour),
        record.rule?.name ?? `隧道 #${record.rule_id ?? '-'}`,
        record.node?.name ?? `节点 #${record.node_id ?? '-'}`,
        formatTraffic(record.traffic_in),
        formatTraffic(record.traffic_out),
        `${toNumber(record.final_multiplier, 1).toFixed(2)}x`,
        formatTraffic(record.calculated_traffic),
      ])

      const content = [header, ...rows]
        .map((line) => line.map((cell) => escapeCSVCell(String(cell))).join(','))
        .join('\n')

      const blob = new Blob([`\uFEFF${content}`], {
        type: 'text/csv;charset=utf-8;',
      })
      const url = URL.createObjectURL(blob)
      const link = document.createElement('a')
      link.href = url
      link.download = `traffic-records-${dayjs().format('YYYYMMDD-HHmmss')}.csv`
      link.click()
      URL.revokeObjectURL(url)
      message.success('流量明细导出成功')
    } catch (error) {
      message.error(getErrorMessage(error, '导出失败'))
    } finally {
      setExportLoading(false)
    }
  }

  return (
    <PageContainer title="流量统计" description="查看配额、趋势和流量明细。">
      <Space direction="vertical" size={16} className="w-full">
        <Card>
          {bootLoading ? (
            <Skeleton active paragraph={{ rows: 2 }} />
          ) : (
            <Row gutter={[16, 16]} align="middle">
              <Col xs={24} md={8}>
                <Progress
                  type="dashboard"
                  percent={usagePercent}
                  status={usagePercent >= 100 ? 'exception' : 'active'}
                  size={180}
                />
              </Col>
              <Col xs={24} md={16}>
                <Space direction="vertical" size={8}>
                  <Statistic
                    title="已用流量 / 总配额"
                    value={`${formatTraffic(quota?.traffic_used ?? 0)} / ${formatTraffic(
                      quota?.traffic_quota ?? 0,
                    )}`}
                  />
                  <Typography.Text type="secondary">
                    流量倍率: VIP {multiplier.vip.toFixed(2)} × 节点 {multiplier.node.toFixed(2)} ={' '}
                    {multiplier.final.toFixed(2)}
                  </Typography.Text>
                </Space>
              </Col>
            </Row>
          )}
        </Card>

        <Card title="流量趋势">
          <Space wrap style={{ marginBottom: 16 }}>
            <Segmented<TrendRangeType>
              value={trendRangeType}
              onChange={(value) => setTrendRangeType(value)}
              options={[
                { label: '最近 7 天', value: '7d' },
                { label: '最近 30 天', value: '30d' },
                { label: '自定义', value: 'custom' },
              ]}
            />

            {trendRangeType === 'custom' ? (
              <DatePicker.RangePicker
                value={trendCustomRange}
                onChange={(values) => {
                  if (values?.[0] && values[1]) {
                    setTrendCustomRange([
                      values[0].startOf('day'),
                      values[1].endOf('day'),
                    ])
                  } else {
                    setTrendCustomRange(null)
                  }
                }}
              />
            ) : null}

            <Segmented<ChartGroupType>
              value={chartGroupType}
              onChange={(value) => setChartGroupType(value)}
              options={[
                { label: '总览', value: 'all' },
                { label: '按隧道', value: 'rule' },
                { label: '按节点', value: 'node' },
              ]}
            />

            {chartGroupType === 'rule' ? (
              <Select
                allowClear
                placeholder="选择隧道"
                style={{ minWidth: 220 }}
                options={ruleOptions}
                value={chartRuleID}
                onChange={(value) => setChartRuleID(value)}
              />
            ) : null}

            {chartGroupType === 'node' ? (
              <Select
                allowClear
                placeholder="选择节点"
                style={{ minWidth: 220 }}
                options={nodeOptions}
                value={chartNodeID}
                onChange={(value) => setChartNodeID(value)}
              />
            ) : null}

            <Button
              icon={<ReloadOutlined />}
              loading={chartLoading}
              onClick={() => void loadTrend()}
            >
              刷新趋势
            </Button>
          </Space>

          {chartLoading && trendPoints.length === 0 ? (
            <Skeleton active paragraph={{ rows: 8 }} />
          ) : (
            <ReactECharts option={trendOption} style={{ height: 360 }} />
          )}
        </Card>

        <Card title="流量明细">
          <Space wrap style={{ marginBottom: 16 }}>
            <DatePicker.RangePicker
              value={tableRange}
              onChange={(values) => {
                if (values?.[0] && values[1]) {
                  setTableRange([values[0].startOf('day'), values[1].endOf('day')])
                  setTablePage(1)
                }
              }}
            />

            <Select
              allowClear
              placeholder="按隧道过滤"
              style={{ minWidth: 220 }}
              options={ruleOptions}
              value={tableRuleID}
              onChange={(value) => {
                setTableRuleID(value)
                setTablePage(1)
              }}
            />

            <Select
              allowClear
              placeholder="按节点过滤"
              style={{ minWidth: 220 }}
              options={nodeOptions}
              value={tableNodeID}
              onChange={(value) => {
                setTableNodeID(value)
                setTablePage(1)
              }}
            />

            <Button
              icon={<ReloadOutlined />}
              onClick={() => void loadTableRecords()}
              loading={tableLoading}
            >
              刷新明细
            </Button>

            <Button
              icon={<DownloadOutlined />}
              onClick={() => void exportRecords()}
              loading={exportLoading}
            >
              导出
            </Button>
          </Space>

          <Table<TrafficRecordItem>
            rowKey="id"
            loading={tableLoading}
            dataSource={tableRecords}
            pagination={{
              current: tablePage,
              pageSize: tablePageSize,
              total: tableTotal,
              showSizeChanger: true,
              showTotal: (totalValue) => `共 ${totalValue} 条`,
              onChange: (nextPage, nextPageSize) => {
                setTablePage(nextPage)
                setTablePageSize(nextPageSize)
              },
            }}
            columns={[
              {
                title: '时间',
                dataIndex: 'hour',
                width: 180,
                render: (value: string) => formatDateTime(value),
              },
              {
                title: '隧道名称',
                width: 220,
                render: (_, record) =>
                  record.rule?.name ?? `隧道 #${record.rule_id ?? '-'}`,
              },
              {
                title: '节点',
                width: 220,
                render: (_, record) =>
                  record.node?.name ?? `节点 #${record.node_id ?? '-'}`,
              },
              {
                title: '上行',
                dataIndex: 'traffic_in',
                width: 130,
                render: (value: number) => formatTraffic(value),
              },
              {
                title: '下行',
                dataIndex: 'traffic_out',
                width: 130,
                render: (value: number) => formatTraffic(value),
              },
              {
                title: '倍率',
                width: 180,
                render: (_, record) =>
                  `${toNumber(record.vip_multiplier, 1).toFixed(2)} × ${toNumber(
                    record.node_multiplier,
                    1,
                  ).toFixed(2)} = ${toNumber(record.final_multiplier, 1).toFixed(2)}`,
              },
              {
                title: '计费流量',
                dataIndex: 'calculated_traffic',
                width: 150,
                render: (value: number) => formatTraffic(value),
              },
            ]}
          />
        </Card>
      </Space>
    </PageContainer>
  )
}

export default TrafficStats
