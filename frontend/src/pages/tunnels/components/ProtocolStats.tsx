import { Card, Row, Col, Statistic, Space, Tag } from 'antd'
import { ApiOutlined, ThunderboltOutlined, SafetyOutlined, RocketOutlined } from '@ant-design/icons'
import ReactECharts from 'echarts-for-react'
import type { Tunnel } from '../../../types/nodeGroup'
import { formatBytes } from '../../../utils/format'

interface ProtocolStatsProps {
  tunnels: Tunnel[]
}

const ProtocolStats = ({ tunnels }: ProtocolStatsProps) => {
  // 按协议统计
  const protocolStats = tunnels.reduce((acc, tunnel) => {
    const protocol = tunnel.protocol.toUpperCase()
    if (!acc[protocol]) {
      acc[protocol] = {
        count: 0,
        traffic_in: 0,
        traffic_out: 0,
        running: 0,
      }
    }
    acc[protocol].count++
    acc[protocol].traffic_in += tunnel.traffic_in || 0
    acc[protocol].traffic_out += tunnel.traffic_out || 0
    if (tunnel.status === 'running') {
      acc[protocol].running++
    }
    return acc
  }, {} as Record<string, { count: number; traffic_in: number; traffic_out: number; running: number }>)

  // 协议分布饼图
  const protocolDistribution = Object.entries(protocolStats).map(([protocol, stats]) => ({
    name: protocol,
    value: stats.count,
  }))

  const pieOption = {
    tooltip: {
      trigger: 'item',
      formatter: '{b}: {c} ({d}%)',
    },
    legend: {
      orient: 'vertical',
      left: 'left',
    },
    series: [
      {
        name: '协议分布',
        type: 'pie',
        radius: '50%',
        data: protocolDistribution,
        emphasis: {
          itemStyle: {
            shadowBlur: 10,
            shadowOffsetX: 0,
            shadowColor: 'rgba(0, 0, 0, 0.5)',
          },
        },
      },
    ],
  }

  // 协议流量对比柱状图
  const protocolTraffic = Object.entries(protocolStats).map(([protocol, stats]) => ({
    protocol,
    inbound: stats.traffic_in,
    outbound: stats.traffic_out,
  }))

  const barOption = {
    tooltip: {
      trigger: 'axis',
      axisPointer: {
        type: 'shadow',
      },
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
    },
    xAxis: {
      type: 'category',
      data: protocolTraffic.map((item) => item.protocol),
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
        type: 'bar',
        data: protocolTraffic.map((item) => item.inbound),
        itemStyle: { color: '#52c41a' },
      },
      {
        name: '出站流量',
        type: 'bar',
        data: protocolTraffic.map((item) => item.outbound),
        itemStyle: { color: '#1890ff' },
      },
    ],
  }

  const getProtocolIcon = (protocol: string) => {
    switch (protocol.toLowerCase()) {
      case 'tcp':
        return <ApiOutlined style={{ color: '#1890ff' }} />
      case 'udp':
        return <ThunderboltOutlined style={{ color: '#52c41a' }} />
      case 'ws':
      case 'wss':
        return <RocketOutlined style={{ color: '#722ed1' }} />
      case 'tls':
        return <SafetyOutlined style={{ color: '#fa8c16' }} />
      case 'quic':
        return <ThunderboltOutlined style={{ color: '#eb2f96' }} />
      default:
        return <ApiOutlined />
    }
  }

  return (
    <Space direction="vertical" size="large" style={{ width: '100%' }}>
      <Row gutter={16}>
        {Object.entries(protocolStats).map(([protocol, stats]) => (
          <Col span={6} key={protocol}>
            <Card>
              <Statistic
                title={
                  <Space>
                    {getProtocolIcon(protocol)}
                    <span>{protocol}</span>
                  </Space>
                }
                value={stats.count}
                suffix="个隧道"
              />
              <div style={{ marginTop: 12 }}>
                <Tag color="green">{stats.running} 运行中</Tag>
                <Tag color="default">{stats.count - stats.running} 已停止</Tag>
              </div>
            </Card>
          </Col>
        ))}
      </Row>

      <Row gutter={16}>
        <Col span={12}>
          <Card title="协议分布">
            <ReactECharts option={pieOption} style={{ height: 300 }} />
          </Card>
        </Col>
        <Col span={12}>
          <Card title="协议流量对比">
            <ReactECharts option={barOption} style={{ height: 300 }} />
          </Card>
        </Col>
      </Row>

      <Card title="协议详细统计">
        <Row gutter={16}>
          {Object.entries(protocolStats).map(([protocol, stats]) => (
            <Col span={8} key={protocol}>
              <Card size="small">
                <Space direction="vertical" style={{ width: '100%' }}>
                  <div style={{ fontSize: 16, fontWeight: 'bold' }}>
                    {getProtocolIcon(protocol)} {protocol}
                  </div>
                  <Statistic
                    title="隧道数量"
                    value={stats.count}
                    valueStyle={{ fontSize: 20 }}
                  />
                  <Statistic
                    title="入站流量"
                    value={formatBytes(stats.traffic_in)}
                    valueStyle={{ fontSize: 16, color: '#52c41a' }}
                  />
                  <Statistic
                    title="出站流量"
                    value={formatBytes(stats.traffic_out)}
                    valueStyle={{ fontSize: 16, color: '#1890ff' }}
                  />
                  <Statistic
                    title="总流量"
                    value={formatBytes(stats.traffic_in + stats.traffic_out)}
                    valueStyle={{ fontSize: 16, color: '#faad14' }}
                  />
                </Space>
              </Card>
            </Col>
          ))}
        </Row>
      </Card>
    </Space>
  )
}

export default ProtocolStats
