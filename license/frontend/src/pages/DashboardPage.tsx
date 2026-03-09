import { Card, Col, Row, Statistic } from 'antd'
import { useEffect, useState } from 'react'
import type { DashboardStats } from '../types/api'
import { dashboardApi } from '../utils/api'

export default function DashboardPage() {
  const [stats, setStats] = useState<DashboardStats | null>(null)

  useEffect(() => {
    dashboardApi.stats().then(setStats).catch(() => setStats(null))
  }, [])

  return (
    <Row gutter={[16, 16]}>
      <Col xs={24} md={8}>
        <Card>
          <Statistic title="授权总数" value={stats?.total_licenses ?? 0} />
        </Card>
      </Col>
      <Col xs={24} md={8}>
        <Card>
          <Statistic title="激活授权" value={stats?.active_licenses ?? 0} />
        </Card>
      </Col>
      <Col xs={24} md={8}>
        <Card>
          <Statistic title="30天内到期" value={stats?.expiring_soon_30_days ?? 0} />
        </Card>
      </Col>
      <Col xs={24} md={8}>
        <Card>
          <Statistic title="设备绑定总数" value={stats?.total_activations ?? 0} />
        </Card>
      </Col>
      <Col xs={24} md={8}>
        <Card>
          <Statistic title="24h 校验请求" value={stats?.verify_requests_24h ?? 0} />
        </Card>
      </Col>
      <Col xs={24} md={8}>
        <Card>
          <Statistic
            title="24h 校验成功率"
            precision={2}
            value={stats?.verify_success_rate_24h ?? 0}
            suffix="%"
          />
        </Card>
      </Col>
    </Row>
  )
}
