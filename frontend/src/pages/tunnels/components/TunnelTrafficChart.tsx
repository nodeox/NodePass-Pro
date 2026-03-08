import { useState, useEffect, useCallback } from 'react'
import { Card, DatePicker, Space, Spin, Empty, Statistic, Row, Col, Typography } from 'antd'
import { ArrowUpOutlined, ArrowDownOutlined, SwapOutlined } from '@ant-design/icons'
import dayjs, { Dayjs } from 'dayjs'
import ReactECharts from 'echarts-for-react'

import { formatBytes } from '../../../utils/format'

const { RangePicker } = DatePicker

interface TunnelTrafficChartProps {
  tunnelId: number
}

type TrafficPoint = {
  time: string
  type: '入站流量' | '出站流量'
  value: number
}

type AxisTooltipParam = {
  axisValue: string
  marker: string
  seriesName: string
  value: number
}

const TunnelTrafficChart = ({ tunnelId }: TunnelTrafficChartProps) => {
  void tunnelId
  const [loading, setLoading] = useState(false)
  const [dateRange, setDateRange] = useState<[Dayjs, Dayjs]>([
    dayjs().subtract(7, 'days'),
    dayjs(),
  ])
  const [trafficData, setTrafficData] = useState<TrafficPoint[]>([])
  const [summary, setSummary] = useState({
    total_traffic_in: 0,
    total_traffic_out: 0,
    total_traffic: 0,
  })

  const fetchTrafficData = useCallback(async () => {
    try {
      setLoading(true)
      // TODO: 实现隧道流量统计 API
      // const result = await tunnelApi.getTraffic(tunnelId, {
      //   start_time: dateRange[0].format('YYYY-MM-DD'),
      //   end_time: dateRange[1].format('YYYY-MM-DD'),
      // })

      // 模拟数据
      const mockData: TrafficPoint[] = []
      const days = dateRange[1].diff(dateRange[0], 'days')
      for (let i = 0; i <= days; i++) {
        const date = dateRange[0].add(i, 'days').format('YYYY-MM-DD')
        const inTraffic = Math.random() * 1024 * 1024 * 100 // 0-100MB
        const outTraffic = Math.random() * 1024 * 1024 * 150 // 0-150MB
        mockData.push(
          { time: date, type: '入站流量', value: inTraffic },
          { time: date, type: '出站流量', value: outTraffic }
        )
      }

      setTrafficData(mockData)

      // 计算总计
      const totalIn = mockData
        .filter((d) => d.type === '入站流量')
        .reduce((sum, d) => sum + d.value, 0)
      const totalOut = mockData
        .filter((d) => d.type === '出站流量')
        .reduce((sum, d) => sum + d.value, 0)

      setSummary({
        total_traffic_in: totalIn,
        total_traffic_out: totalOut,
        total_traffic: totalIn + totalOut,
      })
    } catch (error) {
      console.error('获取流量数据失败:', error)
    } finally {
      setLoading(false)
    }
  }, [dateRange])

  useEffect(() => {
    void fetchTrafficData()
  }, [fetchTrafficData])

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
      formatter: (params: AxisTooltipParam[]) => {
        let result = `${params[0].axisValue}<br/>`
        params.forEach((param) => {
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
    <Card>
      <Space direction="vertical" size="large" style={{ width: '100%' }}>
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
          <Typography.Title level={5} style={{ margin: 0 }}>
            流量使用趋势
          </Typography.Title>
          <RangePicker
            value={dateRange}
            onChange={(dates) => {
              if (dates && dates[0] && dates[1]) {
                setDateRange([dates[0], dates[1]])
              }
            }}
            format="YYYY-MM-DD"
            allowClear={false}
          />
        </div>

        <Row gutter={16}>
          <Col span={8}>
            <Card>
              <Statistic
                title="入站流量"
                value={summary.total_traffic_in}
                formatter={(value) => formatBytes(Number(value))}
                prefix={<ArrowDownOutlined style={{ color: '#52c41a' }} />}
                valueStyle={{ color: '#52c41a' }}
              />
            </Card>
          </Col>
          <Col span={8}>
            <Card>
              <Statistic
                title="出站流量"
                value={summary.total_traffic_out}
                formatter={(value) => formatBytes(Number(value))}
                prefix={<ArrowUpOutlined style={{ color: '#1890ff' }} />}
                valueStyle={{ color: '#1890ff' }}
              />
            </Card>
          </Col>
          <Col span={8}>
            <Card>
              <Statistic
                title="总流量"
                value={summary.total_traffic}
                formatter={(value) => formatBytes(Number(value))}
                prefix={<SwapOutlined style={{ color: '#faad14' }} />}
                valueStyle={{ color: '#faad14' }}
              />
            </Card>
          </Col>
        </Row>

        <Spin spinning={loading}>
          {trafficData.length > 0 ? (
            <ReactECharts option={option} style={{ height: 400 }} />
          ) : (
            <Empty description="暂无流量数据" />
          )}
        </Spin>
      </Space>
    </Card>
  )
}

export default TunnelTrafficChart
