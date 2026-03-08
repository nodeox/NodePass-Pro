import { useState, useEffect } from 'react'
import { Card, Row, Col, Statistic, Space, Typography } from 'antd'
import { ArrowUpOutlined, ArrowDownOutlined, LinkOutlined, ClusterOutlined } from '@ant-design/icons'
import ReactECharts from 'echarts-for-react'
import dayjs from 'dayjs'
import type { NodeGroup } from '../../../types/nodeGroup'
import { formatBytes } from '../../../utils/format'

interface MonitoringDashboardProps {
  group: NodeGroup
}

const MonitoringDashboard = ({ group }: MonitoringDashboardProps) => {
  const [trafficData, setTrafficData] = useState<any[]>([])

  const stats = group.stats

  useEffect(() => {
    // 模拟实时流量数据
    const generateMockData = () => {
      const data: any[] = []
      const now = dayjs()
      for (let i = 23; i >= 0; i--) {
        const time = now.subtract(i, 'hour').format('HH:00')
        const inTraffic = Math.random() * 1024 * 1024 * 50 // 0-50MB
        const outTraffic = Math.random() * 1024 * 1024 * 80 // 0-80MB
        data.push(
          { time, type: '入站流量', value: inTraffic },
          { time, type: '出站流量', value: outTraffic }
        )
      }
      setTrafficData(data)
    }

    generateMockData()
    // 每30秒更新一次数据
    const interval = setInterval(generateMockData, 30000)
    return () => clearInterval(interval)
  }, [group.id])

  // 准备 ECharts 数据
  const times = Array.from(new Set(trafficData.map((d) => d.time))).sort()
  const inData = times.map((time) => {
    const item = trafficData.find((d) => d.time === time && d.type === '入站流量')
    return item ? item.value : 0
  })
  const outData = times.map((time) => {
    const item = trafficData.find((d) => d.time === time && d.type === '出站流量')
    return item ? item.value : 0
  })

  const option = {
    tooltip: {
      trigger: 'axis',
      formatter: (params: any) => {
        let result = `${params[0].axisValue}<br/>`
        params.forEach((param: any) => {
          result += `${param.marker}${param.seriesName}: ${formatBytes(param.value)}<br/>`
        })
        return result
      },
    },
    legend: {
      data: ['入站流量', '出站流量'],
      top: 0,
    },
    grid: {
      left: '3%',
      right: '4%',
      bottom: '3%',
      containLabel: true,
    },
    xAxis: {
      type: 'category',
      boundaryGap: false,
      data: times,
    },
    yAxis: {
      type: 'value',
      axisLabel: {
        formatter: (value: number) => formatBytes(value),
      },
    },
    series: [
      {
        name: '入站流量',
        type: 'line',
        smooth: true,
        data: inData,
        itemStyle: { color: '#52c41a' },
        areaStyle: { opacity: 0.3 },
      },
      {
        name: '出站流量',
        type: 'line',
        smooth: true,
        data: outData,
        itemStyle: { color: '#1890ff' },
        areaStyle: { opacity: 0.3 },
      },
    ],
  }

  return (
    <Space direction="vertical" size="large" style={{ width: '100%' }}>
      <Row gutter={16}>
        <Col span={6}>
          <Card>
            <Statistic
              title="入站流量"
              value={stats?.total_traffic_in ?? 0}
              formatter={(value) => formatBytes(Number(value))}
              prefix={<ArrowDownOutlined style={{ color: '#52c41a' }} />}
              valueStyle={{ color: '#52c41a' }}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic
              title="出站流量"
              value={stats?.total_traffic_out ?? 0}
              formatter={(value) => formatBytes(Number(value))}
              prefix={<ArrowUpOutlined style={{ color: '#1890ff' }} />}
              valueStyle={{ color: '#1890ff' }}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic
              title="总连接数"
              value={stats?.total_connections ?? 0}
              prefix={<LinkOutlined style={{ color: '#faad14' }} />}
              valueStyle={{ color: '#faad14' }}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic
              title="在线节点"
              value={stats?.online_nodes ?? 0}
              suffix={`/ ${stats?.total_nodes ?? 0}`}
              prefix={<ClusterOutlined style={{ color: '#722ed1' }} />}
              valueStyle={{ color: '#722ed1' }}
            />
          </Card>
        </Col>
      </Row>

      <Card>
        <Typography.Title level={5} style={{ marginBottom: 16 }}>
          流量趋势（最近24小时）
        </Typography.Title>
        <ReactECharts option={option} style={{ height: 350 }} />
      </Card>

      <Card title="实时统计">
        <Row gutter={16}>
          <Col span={8}>
            <Statistic
              title="总流量"
              value={(stats?.total_traffic_in ?? 0) + (stats?.total_traffic_out ?? 0)}
              formatter={(value) => formatBytes(Number(value))}
            />
          </Col>
          <Col span={8}>
            <Statistic
              title="平均流量"
              value={
                stats?.total_nodes && stats.total_nodes > 0
                  ? ((stats.total_traffic_in ?? 0) + (stats.total_traffic_out ?? 0)) / stats.total_nodes
                  : 0
              }
              formatter={(value) => formatBytes(Number(value))}
            />
          </Col>
          <Col span={8}>
            <Statistic
              title="平均连接数"
              value={
                stats?.total_nodes && stats.total_nodes > 0
                  ? Math.round((stats.total_connections ?? 0) / stats.total_nodes)
                  : 0
              }
            />
          </Col>
        </Row>
      </Card>
    </Space>
  )
}

export default MonitoringDashboard
