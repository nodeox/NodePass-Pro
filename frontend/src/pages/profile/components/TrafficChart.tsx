import { useState, useEffect, useCallback } from 'react'
import { Card, DatePicker, Space, Spin, Empty, Statistic, Row, Col } from 'antd'
import { ArrowUpOutlined, ArrowDownOutlined, CloudOutlined } from '@ant-design/icons'
import dayjs, { Dayjs } from 'dayjs'
import ReactECharts from '../../../components/charts/EChartsCore'

import { trafficApi } from '../../../services/api'
import { formatBytes } from '../../../utils/format'

const { RangePicker } = DatePicker

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

const TrafficChart = () => {
  const [loading, setLoading] = useState(false)
  const [dateRange, setDateRange] = useState<[Dayjs, Dayjs]>([
    dayjs().subtract(7, 'days'),
    dayjs(),
  ])
  const [trafficData, setTrafficData] = useState<TrafficPoint[]>([])
  const [summary, setSummary] = useState({
    total_traffic_in: 0,
    total_traffic_out: 0,
    total_calculated_traffic: 0,
  })

  const fetchTrafficData = useCallback(async () => {
    try {
      setLoading(true)
      const result = await trafficApi.usage({
        start_time: dateRange[0].format('YYYY-MM-DD'),
        end_time: dateRange[1].format('YYYY-MM-DD'),
      })

      setSummary({
        total_traffic_in: result.total_traffic_in || 0,
        total_traffic_out: result.total_traffic_out || 0,
        total_calculated_traffic: result.total_calculated_traffic || 0,
      })

      // 获取详细记录用于图表展示
      const records = await trafficApi.records({
        start_time: dateRange[0].format('YYYY-MM-DD'),
        end_time: dateRange[1].format('YYYY-MM-DD'),
        page: 1,
        pageSize: 1000,
      })

      // 按小时聚合数据
      const hourlyData: Record<string, { in: number; out: number }> = {}
      records.list.forEach((record) => {
        const hour = record.hour || record.created_at
        if (!hourlyData[hour]) {
          hourlyData[hour] = { in: 0, out: 0 }
        }
        hourlyData[hour].in += record.traffic_in
        hourlyData[hour].out += record.traffic_out
      })

      // 转换为图表数据格式
      const chartData: TrafficPoint[] = []
      Object.entries(hourlyData).forEach(([hour, data]) => {
        chartData.push(
          { time: hour, type: '入站流量', value: data.in },
          { time: hour, type: '出站流量', value: data.out }
        )
      })

      // 按时间排序
      chartData.sort((a, b) => a.time.localeCompare(b.time))
      setTrafficData(chartData)
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
          <h3>流量使用趋势</h3>
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
                title="计费流量"
                value={summary.total_calculated_traffic}
                formatter={(value) => formatBytes(Number(value))}
                prefix={<CloudOutlined style={{ color: '#faad14' }} />}
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

export default TrafficChart
