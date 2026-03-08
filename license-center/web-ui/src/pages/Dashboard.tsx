import { useQuery } from '@tanstack/react-query'
import { Row, Col, Card, Statistic, Table, Tag } from 'antd'
import {
  SafetyCertificateOutlined,
  CheckCircleOutlined,
  CloseCircleOutlined,
  StopOutlined,
  LinkOutlined,
  WarningOutlined,
} from '@ant-design/icons'
import { dashboardApi } from '@/api'
import { LineChart, Line, XAxis, YAxis, CartesianGrid, Tooltip, Legend, ResponsiveContainer } from 'recharts'

export default function Dashboard() {
  const { data: stats } = useQuery({
    queryKey: ['dashboard-stats'],
    queryFn: () => dashboardApi.getStats(),
  })

  const { data: trendData } = useQuery({
    queryKey: ['verify-trend', 7],
    queryFn: () => dashboardApi.getVerifyTrend(7),
  })

  const { data: topCustomers } = useQuery({
    queryKey: ['top-customers'],
    queryFn: () => dashboardApi.getTopCustomers(10),
  })

  const statsData = stats?.data

  const columns = [
    {
      title: '客户名称',
      dataIndex: 'customer',
      key: 'customer',
    },
    {
      title: '授权码数量',
      dataIndex: 'license_count',
      key: 'license_count',
    },
    {
      title: '激活机器数',
      dataIndex: 'activation_count',
      key: 'activation_count',
    },
  ]

  return (
    <div>
      <Row gutter={[16, 16]}>
        <Col xs={24} sm={12} lg={6}>
          <Card>
            <Statistic
              title="授权码总数"
              value={statsData?.license_total || 0}
              prefix={<SafetyCertificateOutlined />}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card>
            <Statistic
              title="活跃授权码"
              value={statsData?.license_active || 0}
              prefix={<CheckCircleOutlined />}
              valueStyle={{ color: '#3f8600' }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card>
            <Statistic
              title="已过期"
              value={statsData?.license_expired || 0}
              prefix={<CloseCircleOutlined />}
              valueStyle={{ color: '#cf1322' }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card>
            <Statistic
              title="已吊销"
              value={statsData?.license_revoked || 0}
              prefix={<StopOutlined />}
              valueStyle={{ color: '#999' }}
            />
          </Card>
        </Col>
      </Row>

      <Row gutter={[16, 16]} style={{ marginTop: 16 }}>
        <Col xs={24} sm={12} lg={6}>
          <Card>
            <Statistic
              title="激活机器数"
              value={statsData?.activation_total || 0}
              prefix={<LinkOutlined />}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card>
            <Statistic
              title="今日验证"
              value={statsData?.verify_today || 0}
              suffix={`/ ${statsData?.verify_success_today || 0} 成功`}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card>
            <Statistic
              title="本周验证"
              value={statsData?.verify_week || 0}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card>
            <Statistic
              title="即将过期"
              value={statsData?.expiring_count || 0}
              prefix={<WarningOutlined />}
              valueStyle={{ color: statsData?.expiring_count ? '#faad14' : undefined }}
            />
          </Card>
        </Col>
      </Row>

      <Row gutter={[16, 16]} style={{ marginTop: 16 }}>
        <Col xs={24} lg={16}>
          <Card title="验证趋势（最近 7 天）">
            <ResponsiveContainer width="100%" height={300}>
              <LineChart data={trendData?.data || []}>
                <CartesianGrid strokeDasharray="3 3" />
                <XAxis dataKey="date" />
                <YAxis />
                <Tooltip />
                <Legend />
                <Line type="monotone" dataKey="success" stroke="#52c41a" name="成功" />
                <Line type="monotone" dataKey="failed" stroke="#ff4d4f" name="失败" />
                <Line type="monotone" dataKey="total" stroke="#1890ff" name="总计" />
              </LineChart>
            </ResponsiveContainer>
          </Card>
        </Col>

        <Col xs={24} lg={8}>
          <Card title="Top 客户">
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
