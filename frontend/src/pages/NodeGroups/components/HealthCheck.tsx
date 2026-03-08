import { Card, Progress, Row, Col, Statistic, Tag, Space, Alert } from 'antd'
import { CheckCircleOutlined, CloseCircleOutlined, WarningOutlined } from '@ant-design/icons'
import type { NodeGroup } from '../../../types/nodeGroup'

interface HealthCheckProps {
  group: NodeGroup
}

const HealthCheck = ({ group }: HealthCheckProps) => {
  const stats = group.stats
  const totalNodes = stats?.total_nodes ?? 0
  const onlineNodes = stats?.online_nodes ?? 0
  const offlineNodes = totalNodes - onlineNodes
  const onlineRate = totalNodes > 0 ? Math.round((onlineNodes / totalNodes) * 100) : 0

  // 健康状态判断
  const getHealthStatus = () => {
    if (onlineRate >= 80) {
      return { status: 'success', text: '健康', color: 'green', icon: <CheckCircleOutlined /> }
    }
    if (onlineRate >= 50) {
      return { status: 'warning', text: '警告', color: 'orange', icon: <WarningOutlined /> }
    }
    return { status: 'error', text: '异常', color: 'red', icon: <CloseCircleOutlined /> }
  }

  const health = getHealthStatus()

  return (
    <Space direction="vertical" size="large" style={{ width: '100%' }}>
      <Card>
        <Row gutter={16}>
          <Col span={6}>
            <Statistic
              title="健康状态"
              value={health.text}
              prefix={health.icon}
              valueStyle={{ color: health.color }}
            />
          </Col>
          <Col span={6}>
            <Statistic
              title="在线节点"
              value={onlineNodes}
              suffix={`/ ${totalNodes}`}
              valueStyle={{ color: onlineNodes > 0 ? '#52c41a' : '#999' }}
            />
          </Col>
          <Col span={6}>
            <Statistic
              title="离线节点"
              value={offlineNodes}
              valueStyle={{ color: offlineNodes > 0 ? '#ff4d4f' : '#999' }}
            />
          </Col>
          <Col span={6}>
            <Statistic
              title="在线率"
              value={onlineRate}
              suffix="%"
              valueStyle={{ color: onlineRate >= 80 ? '#52c41a' : onlineRate >= 50 ? '#faad14' : '#ff4d4f' }}
            />
          </Col>
        </Row>

        <div style={{ marginTop: 24 }}>
          <div style={{ marginBottom: 8 }}>节点在线率</div>
          <Progress
            percent={onlineRate}
            status={onlineRate >= 80 ? 'success' : onlineRate >= 50 ? 'normal' : 'exception'}
            strokeColor={onlineRate >= 80 ? '#52c41a' : onlineRate >= 50 ? '#faad14' : '#ff4d4f'}
          />
        </div>
      </Card>

      {onlineRate < 50 && (
        <Alert
          message="节点组健康状态异常"
          description={`当前在线率仅为 ${onlineRate}%，建议检查离线节点并及时处理。`}
          type="error"
          showIcon
        />
      )}

      {onlineRate >= 50 && onlineRate < 80 && (
        <Alert
          message="节点组健康状态警告"
          description={`当前在线率为 ${onlineRate}%，部分节点离线，建议关注节点状态。`}
          type="warning"
          showIcon
        />
      )}

      <Card title="节点状态分布">
        <Space size="large">
          <div>
            <Tag color="green" icon={<CheckCircleOutlined />}>
              在线: {onlineNodes}
            </Tag>
          </div>
          <div>
            <Tag color="red" icon={<CloseCircleOutlined />}>
              离线: {offlineNodes}
            </Tag>
          </div>
          <div>
            <Tag color="blue">
              总计: {totalNodes}
            </Tag>
          </div>
        </Space>
      </Card>
    </Space>
  )
}

export default HealthCheck
