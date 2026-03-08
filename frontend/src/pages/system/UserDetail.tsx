import {
  ArrowLeftOutlined,
  ReloadOutlined,
  UserOutlined,
  SafetyOutlined,
  DashboardOutlined,
  HistoryOutlined,
} from '@ant-design/icons'
import {
  Button,
  Card,
  Descriptions,
  Space,
  Statistic,
  Tabs,
  Tag,
  Typography,
  Row,
  Col,
  message,
  Spin,
} from 'antd'
import { useCallback, useEffect, useState } from 'react'
import { useNavigate, useParams } from 'react-router-dom'

import PageContainer from '../../components/common/PageContainer'
import { usePageTitle } from '../../hooks/usePageTitle'
import { userAdminApi } from '../../services/api'
import type { AdminUserRecord } from '../../types'
import { getErrorMessage } from '../../utils/error'
import { formatTraffic, formatDateTime } from '../../utils/format'

const statusTagMap: Record<string, { color: string; label: string }> = {
  normal: { color: 'green', label: '正常' },
  paused: { color: 'orange', label: '暂停' },
  banned: { color: 'red', label: '封禁' },
  overlimit: { color: 'magenta', label: '超限' },
}

const UserDetail = () => {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  usePageTitle('用户详情')

  const [loading, setLoading] = useState<boolean>(false)
  const [user, setUser] = useState<AdminUserRecord | null>(null)

  const loadUser = useCallback(async () => {
    if (!id) return
    setLoading(true)
    try {
      const result = await userAdminApi.getUser(Number(id))
      setUser(result)
    } catch (error) {
      message.error(getErrorMessage(error, '加载用户详情失败'))
    } finally {
      setLoading(false)
    }
  }, [id])

  useEffect(() => {
    void loadUser()
  }, [loadUser])

  if (loading) {
    return (
      <PageContainer title="用户详情">
        <div style={{ textAlign: 'center', padding: '100px 0' }}>
          <Spin size="large" />
        </div>
      </PageContainer>
    )
  }

  if (!user) {
    return (
      <PageContainer title="用户详情">
        <Card>
          <Typography.Text type="secondary">用户不存在</Typography.Text>
        </Card>
      </PageContainer>
    )
  }

  const statusTag = statusTagMap[user.status] || { color: 'default', label: user.status }
  const trafficUsagePercent = user.traffic_quota > 0
    ? (user.traffic_used / user.traffic_quota) * 100
    : 0

  return (
    <PageContainer
      title="用户详情"
      extra={
        <Space>
          <Button
            icon={<ArrowLeftOutlined />}
            onClick={() => navigate('/system/users')}
          >
            返回
          </Button>
          <Button
            icon={<ReloadOutlined />}
            onClick={() => void loadUser()}
            loading={loading}
          >
            刷新
          </Button>
        </Space>
      }
    >
      <Space direction="vertical" size="large" style={{ width: '100%' }}>
        {/* 基本信息卡片 */}
        <Card title={<Space><UserOutlined /> 基本信息</Space>}>
          <Descriptions column={{ xs: 1, sm: 2, md: 3 }}>
            <Descriptions.Item label="用户 ID">{user.id}</Descriptions.Item>
            <Descriptions.Item label="用户名">{user.username}</Descriptions.Item>
            <Descriptions.Item label="邮箱">{user.email}</Descriptions.Item>
            <Descriptions.Item label="角色">
              <Tag color={user.role === 'admin' ? 'purple' : 'blue'}>
                {user.role === 'admin' ? '管理员' : '普通用户'}
              </Tag>
            </Descriptions.Item>
            <Descriptions.Item label="状态">
              <Tag color={statusTag.color}>{statusTag.label}</Tag>
            </Descriptions.Item>
            <Descriptions.Item label="VIP 等级">
              Lv.{user.vip_level}
            </Descriptions.Item>
            <Descriptions.Item label="VIP 到期时间">
              {user.vip_expires_at ? formatDateTime(user.vip_expires_at) : '-'}
            </Descriptions.Item>
            <Descriptions.Item label="注册时间">
              {formatDateTime(user.created_at)}
            </Descriptions.Item>
            <Descriptions.Item label="最后登录">
              {user.last_login_at ? formatDateTime(user.last_login_at) : '-'}
            </Descriptions.Item>
            <Descriptions.Item label="Telegram ID">
              {user.telegram_id || '-'}
            </Descriptions.Item>
            <Descriptions.Item label="Telegram 用户名">
              {user.telegram_username ? `@${user.telegram_username}` : '-'}
            </Descriptions.Item>
          </Descriptions>
        </Card>

        {/* 流量统计卡片 */}
        <Card title={<Space><DashboardOutlined /> 流量统计</Space>}>
          <Row gutter={16}>
            <Col xs={24} sm={12} md={6}>
              <Statistic
                title="流量配额"
                value={formatTraffic(user.traffic_quota)}
                valueStyle={{ color: '#1890ff' }}
              />
            </Col>
            <Col xs={24} sm={12} md={6}>
              <Statistic
                title="已用流量"
                value={formatTraffic(user.traffic_used)}
                valueStyle={{ color: '#52c41a' }}
              />
            </Col>
            <Col xs={24} sm={12} md={6}>
              <Statistic
                title="剩余流量"
                value={formatTraffic(Math.max(0, user.traffic_quota - user.traffic_used))}
                valueStyle={{ color: '#faad14' }}
              />
            </Col>
            <Col xs={24} sm={12} md={6}>
              <Statistic
                title="使用率"
                value={trafficUsagePercent.toFixed(2)}
                suffix="%"
                valueStyle={{
                  color: trafficUsagePercent > 90 ? '#ff4d4f' : trafficUsagePercent > 70 ? '#faad14' : '#52c41a'
                }}
              />
            </Col>
          </Row>
        </Card>

        {/* 权限配置卡片 */}
        <Card title={<Space><SafetyOutlined /> 权限配置</Space>}>
          <Descriptions column={{ xs: 1, sm: 2, md: 3 }}>
            <Descriptions.Item label="最大规则数">{user.max_rules ?? '-'}</Descriptions.Item>
            <Descriptions.Item label="最大带宽">{user.max_bandwidth ? `${user.max_bandwidth} Mbps` : '-'}</Descriptions.Item>
            <Descriptions.Item label="最大自建入口节点">{user.max_self_hosted_entry_nodes ?? '-'}</Descriptions.Item>
            <Descriptions.Item label="最大自建出口节点">{user.max_self_hosted_exit_nodes ?? '-'}</Descriptions.Item>
          </Descriptions>
        </Card>

        {/* 详细信息标签页 */}
        <Card>
          <Tabs
            items={[
              {
                key: 'activity',
                label: <Space><HistoryOutlined /> 活动记录</Space>,
                children: (
                  <Typography.Text type="secondary">
                    活动记录功能开发中...
                  </Typography.Text>
                ),
              },
              {
                key: 'nodes',
                label: '节点使用',
                children: (
                  <Typography.Text type="secondary">
                    节点使用统计功能开发中...
                  </Typography.Text>
                ),
              },
              {
                key: 'tunnels',
                label: '隧道列表',
                children: (
                  <Typography.Text type="secondary">
                    隧道列表功能开发中...
                  </Typography.Text>
                ),
              },
            ]}
          />
        </Card>
      </Space>
    </PageContainer>
  )
}

export default UserDetail
