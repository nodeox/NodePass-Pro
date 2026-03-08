import { useQuery } from '@tanstack/react-query'
import { Row, Col, Card, Statistic, Table, Tag, Spin } from 'antd'
import {
  SafetyCertificateOutlined,
  CheckCircleOutlined,
  CloseCircleOutlined,
  StopOutlined,
  LinkOutlined,
  WarningOutlined,
  ArrowUpOutlined,
  ArrowDownOutlined,
} from '@ant-design/icons'
import { dashboardApi } from '@/api'
import { LineChart, Line, XAxis, YAxis, CartesianGrid, Tooltip, Legend, ResponsiveContainer, AreaChart, Area } from 'recharts'

export default function Dashboard() {
  const { data: stats, isLoading: statsLoading } = useQuery({
    queryKey: ['dashboard-stats'],
    queryFn: () => dashboardApi.getStats(),
  })

  const { data: trendData, isLoading: trendLoading } = useQuery({
    queryKey: ['verify-trend', 7],
    queryFn: () => dashboardApi.getVerifyTrend(7),
  })

  const { data: topCustomers, isLoading: customersLoading } = useQuery({
    queryKey: ['top-customers'],
    queryFn: () => dashboardApi.getTopCustomers(10),
  })

  const statsData = stats?.data

  // 计算增长率
  const calculateGrowth = (current: number, previous: number) => {
    if (previous === 0) return 0
    return ((current - previous) / previous * 100).toFixed(1)
  }

  const columns = [
    {
      title: '排名',
      key: 'rank',
      width: 60,
      render: (_: any, __: any, index: number) => (
        <span style={{
          display: 'inline-flex',
          alignItems: 'center',
          justifyContent: 'center',
          width: 24,
          height: 24,
          borderRadius: '50%',
          background: index < 3 ? 'linear-gradient(135deg, #667eea 0%, #764ba2 100%)' : '#f0f0f0',
          color: index < 3 ? '#fff' : '#595959',
          fontSize: 12,
          fontWeight: 600,
        }}>
          {index + 1}
        </span>
      ),
    },
    {
      title: '客户名称',
      dataIndex: 'customer',
      key: 'customer',
    },
    {
      title: '授权码',
      dataIndex: 'license_count',
      key: 'license_count',
      render: (count: number) => <Tag color="blue">{count}</Tag>,
    },
    {
      title: '激活机器',
      dataIndex: 'activation_count',
      key: 'activation_count',
      render: (count: number) => <Tag color="green">{count}</Tag>,
    },
  ]

  if (statsLoading) {
    return (
      <div style={{ display: 'flex', justifyContent: 'center', alignItems: 'center', minHeight: 400 }}>
        <Spin size="large" />
      </div>
    )
  }

  return (
    <div>
      {/* 核心指标 */}
      <Row gutter={[16, 16]}>
        <Col xs={24} sm={12} lg={6}>
          <Card bordered={false} style={{ borderRadius: 8 }}>
            <Statistic
              title="授权码总数"
              value={statsData?.license_total || 0}
              prefix={<SafetyCertificateOutlined />}
              valueStyle={{ color: '#1890ff' }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card bordered={false} style={{ borderRadius: 8 }}>
            <Statistic
              title="活跃授权码"
              value={statsData?.license_active || 0}
              prefix={<CheckCircleOutlined />}
              valueStyle={{ color: '#52c41a' }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card bordered={false} style={{ borderRadius: 8 }}>
            <Statistic
              title="已过期"
              value={statsData?.license_expired || 0}
              prefix={<CloseCircleOutlined />}
              valueStyle={{ color: '#ff4d4f' }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card bordered={false} style={{ borderRadius: 8 }}>
            <Statistic
              title="已吊销"
              value={statsData?.license_revoked || 0}
              prefix={<StopOutlined />}
              valueStyle={{ color: '#8c8c8c' }}
            />
          </Card>
        </Col>
      </Row>

      {/* 使用统计 */}
      <Row gutter={[16, 16]} style={{ marginTop: 16 }}>
        <Col xs={24} sm={12} lg={6}>
          <Card bordered={false} style={{ borderRadius: 8 }}>
            <Statistic
              title="激活机器数"
              value={statsData?.activation_total || 0}
              prefix={<LinkOutlined />}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card bordered={false} style={{ borderRadius: 8 }}>
            <Statistic
              title="今日验证"
              value={statsData?.verify_today || 0}
              suffix={
                <span style={{ fontSize: 14, color: '#52c41a' }}>
                  / {statsData?.verify_success_today || 0} 成功
                </span>
              }
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card bordered={false} style={{ borderRadius: 8 }}>
            <Statistic
              title="本周验证"
              value={statsData?.verify_week || 0}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card bordered={false} style={{ borderRadius: 8 }}>
            <Statistic
              title="即将过期"
              value={statsData?.expiring_count || 0}
              prefix={<WarningOutlined />}
              valueStyle={{ color: statsData?.expiring_count ? '#faad14' : undefined }}
            />
          </Card>
        </Col>
      </Row>

      {/* 趋势图表和 Top 客户 */}
      <Row gutter={[16, 16]} style={{ marginTop: 16 }}>
        <Col xs={24} lg={16}>
          <Card
            title="验证趋势（最近 7 天）"
            bordered={false}
            style={{ borderRadius: 8 }}
            loading={trendLoading}
          >
            <ResponsiveContainer width="100%" height={300}>
              <AreaChart data={trendData?.data || []}>
                <defs>
                  <linearGradient id="colorSuccess" x1="0" y1="0" x2="0" y2="1">
                    <stop offset="5%" stopColor="#52c41a" stopOpacity={0.3}/>
                    <stop offset="95%" stopColor="#52c41a" stopOpacity={0}/>
                  </linearGradient>
                  <linearGradient id="colorFailed" x1="0" y1="0" x2="0" y2="1">
                    <stop offset="5%" stopColor="#ff4d4f" stopOpacity={0.3}/>
                    <stop offset="95%" stopColor="#ff4d4f" stopOpacity={0}/>
                  </linearGradient>
                </defs>
                <CartesianGrid strokeDasharray="3 3" stroke="#f0f0f0" />
                <XAxis dataKey="date" stroke="#8c8c8c" />
                <YAxis stroke="#8c8c8c" />
                <Tooltip />
                <Legend />
                <Area
                  type="monotone"
                  dataKey="success"
                  stroke="#52c41a"
                  fillOpacity={1}
                  fill="url(#colorSuccess)"
                  name="成功"
                />
                <Area
                  type="monotone"
                  dataKey="failed"
                  stroke="#ff4d4f"
                  fillOpacity={1}
                  fill="url(#colorFailed)"
                  name="失败"
                />
              </AreaChart>
            </ResponsiveContainer>
          </Card>
        </Col>

        <Col xs={24} lg={8}>
          <Card
            title="Top 客户"
            bordered={false}
            style={{ borderRadius: 8 }}
            loading={customersLoading}
          >
            <Table
              dataSource={topCustomers?.data || []}
              columns={columns}
              pagination={false}
              size="small"
              rowKey="customer"
            />
          </Card>
        </Col>
      </Row>
    </div>
  )
}
