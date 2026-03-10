import {
  ArrowLeftOutlined,
  ReloadOutlined,
  UserOutlined,
  SafetyOutlined,
  DashboardOutlined,
  HistoryOutlined,
  LinkOutlined,
  ClusterOutlined,
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
  Table,
  Empty,
  Tooltip,
} from 'antd'
import { useCallback, useEffect, useState } from 'react'
import { useNavigate, useParams } from 'react-router-dom'

import PageContainer from '../../components/common/PageContainer'
import { usePageTitle } from '../../hooks/usePageTitle'
import { userAdminApi } from '../../services/api'
import type { AdminUserDetailResult } from '../../types'
import { getErrorMessage } from '../../utils/error'
import { formatTraffic, formatDateTime } from '../../utils/format'

const statusTagMap: Record<string, { color: string; label: string }> = {
  normal: { color: 'green', label: '正常' },
  paused: { color: 'orange', label: '暂停' },
  banned: { color: 'red', label: '封禁' },
  overlimit: { color: 'magenta', label: '超限' },
}

const permissionLabelMap: Record<string, string> = {
  'users.read': '查看用户',
  'users.write': '管理用户',
  'roles.read': '查看角色',
  'roles.write': '管理角色',
  'node_groups.read': '查看节点组',
  'node_groups.write': '管理节点组',
  'node_instances.read': '查看节点实例',
  'node_instances.write': '管理节点实例',
  'tunnels.read': '查看隧道',
  'tunnels.write': '管理隧道',
  'traffic.read': '查看流量',
  'traffic.write': '管理流量',
  'vip.read': '查看 VIP',
  'vip.write': '管理 VIP',
  'benefit_codes.read': '查看权益码',
  'benefit_codes.write': '管理权益码',
  'announcements.read': '查看公告',
  'announcements.write': '管理公告',
  'system.config.read': '查看系统配置',
  'system.config.write': '管理系统配置',
  'audit_logs.read': '查看审计日志',
  'alerts.read': '查看告警',
  'alerts.write': '管理告警',
  'notification_channels.read': '查看通知渠道',
  'notification_channels.write': '管理通知渠道',
}

const getPermissionLabel = (permission: string): string =>
  permissionLabelMap[permission] ?? permission

const UserDetail = () => {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  usePageTitle('用户详情')

  const [loading, setLoading] = useState<boolean>(false)
  const [detail, setDetail] = useState<AdminUserDetailResult | null>(null)

  const loadUserDetail = useCallback(async () => {
    if (!id) return
    setLoading(true)
    try {
      const result = await userAdminApi.getUserDetail(Number(id))
      setDetail(result)
    } catch (error) {
      message.error(getErrorMessage(error, '加载用户详情失败'))
    } finally {
      setLoading(false)
    }
  }, [id])

  useEffect(() => {
    void loadUserDetail()
  }, [loadUserDetail])

  if (loading) {
    return (
      <PageContainer title="用户详情">
        <div style={{ textAlign: 'center', padding: '100px 0' }}>
          <Spin size="large" />
        </div>
      </PageContainer>
    )
  }

  if (!detail?.user) {
    return (
      <PageContainer title="用户详情">
        <Card>
          <Typography.Text type="secondary">用户不存在</Typography.Text>
        </Card>
      </PageContainer>
    )
  }

  const user = detail.user
  const statusTag = statusTagMap[user.status] || { color: 'default', label: user.status }
  const roleLabel = detail.role?.name || user.role
  const stats = detail.stats ?? {
    tunnel_count: 0,
    running_tunnel_count: 0,
    node_group_count: 0,
    node_instance_count: 0,
    active_session_count: 0,
    total_traffic_in: 0,
    total_traffic_out: 0,
  }
  const sessions = detail.sessions ?? []
  const activities = detail.recent_activities ?? []
  const recentTunnels = detail.recent_tunnels ?? []
  const recentNodeGroups = detail.recent_node_groups ?? []
  const trafficUsagePercent = user.traffic_quota > 0
    ? Math.min(100, (user.traffic_used / user.traffic_quota) * 100)
    : 0

  const permissionTags = !detail.permissions || detail.permissions.length === 0
    ? <Typography.Text type="secondary">无显式权限（依赖角色默认能力）</Typography.Text>
    : (
      <Space size={[6, 8]} wrap>
        {detail.permissions.map((permission) => (
          <Tooltip key={permission} title={permission}>
            <Tag color="blue">{getPermissionLabel(permission)}</Tag>
          </Tooltip>
        ))}
      </Space>
    )

  return (
    <PageContainer
      title="用户详情"
      description="查看用户基础信息、授权状态、会话与行为摘要。"
      extra={
        <Space>
          <Button icon={<ArrowLeftOutlined />} onClick={() => navigate('/admin/system/users')}>
            返回
          </Button>
          <Button icon={<ReloadOutlined />} onClick={() => void loadUserDetail()} loading={loading}>
            刷新
          </Button>
        </Space>
      }
    >
      <Space direction="vertical" size="large" style={{ width: '100%' }}>
        <Card title={<Space><UserOutlined /> 基本信息</Space>}>
          <Descriptions column={{ xs: 1, sm: 2, md: 3 }}>
            <Descriptions.Item label="用户 ID">{user.id}</Descriptions.Item>
            <Descriptions.Item label="用户名">{user.username}</Descriptions.Item>
            <Descriptions.Item label="邮箱">{user.email}</Descriptions.Item>
            <Descriptions.Item label="角色">
              <Tag color={user.role === 'admin' ? 'purple' : 'blue'}>{roleLabel}</Tag>
              {detail.role?.is_system ? <Tag color="gold">系统角色</Tag> : null}
              {detail.role && !detail.role.is_enabled ? <Tag color="red">角色已禁用</Tag> : null}
            </Descriptions.Item>
            <Descriptions.Item label="状态">
              <Tag color={statusTag.color}>{statusTag.label}</Tag>
            </Descriptions.Item>
            <Descriptions.Item label="VIP 等级">Lv.{user.vip_level}</Descriptions.Item>
            <Descriptions.Item label="VIP 到期时间">
              {user.vip_expires_at ? formatDateTime(user.vip_expires_at) : '-'}
            </Descriptions.Item>
            <Descriptions.Item label="注册时间">
              {user.created_at ? formatDateTime(user.created_at) : '-'}
            </Descriptions.Item>
            <Descriptions.Item label="最后登录">
              {user.last_login_at ? formatDateTime(user.last_login_at) : '-'}
            </Descriptions.Item>
            <Descriptions.Item label="Telegram">
              {user.telegram_username ? `@${user.telegram_username}` : user.telegram_id || '-'}
            </Descriptions.Item>
          </Descriptions>
        </Card>

        <Card title={<Space><DashboardOutlined /> 统计概览</Space>}>
          <Row gutter={[16, 16]}>
            <Col xs={24} sm={12} md={6}>
              <Statistic title="隧道总数" value={stats.tunnel_count} />
            </Col>
            <Col xs={24} sm={12} md={6}>
              <Statistic title="运行中隧道" value={stats.running_tunnel_count} />
            </Col>
            <Col xs={24} sm={12} md={6}>
              <Statistic title="节点组总数" value={stats.node_group_count} />
            </Col>
            <Col xs={24} sm={12} md={6}>
              <Statistic title="节点实例数" value={stats.node_instance_count} />
            </Col>
            <Col xs={24} sm={12} md={6}>
              <Statistic title="活跃会话" value={stats.active_session_count} />
            </Col>
            <Col xs={24} sm={12} md={6}>
              <Statistic title="总上行" value={formatTraffic(stats.total_traffic_in)} />
            </Col>
            <Col xs={24} sm={12} md={6}>
              <Statistic title="总下行" value={formatTraffic(stats.total_traffic_out)} />
            </Col>
            <Col xs={24} sm={12} md={6}>
              <Statistic title="配额使用率" value={trafficUsagePercent.toFixed(2)} suffix="%" />
            </Col>
          </Row>
        </Card>

        <Card title={<Space><SafetyOutlined /> 权限信息</Space>}>
          {permissionTags}
        </Card>

        <Card>
          <Tabs
            items={[
              {
                key: 'sessions',
                label: <Space><HistoryOutlined /> 会话记录</Space>,
                children: sessions.length > 0 ? (
                  <Table
                    size="small"
                    rowKey="id"
                    pagination={{ pageSize: 5 }}
                    dataSource={sessions}
                    columns={[
                      { title: '会话 ID', dataIndex: 'id', width: 90 },
                      {
                        title: '状态',
                        dataIndex: 'is_revoked',
                        width: 90,
                        render: (revoked: boolean) =>
                          revoked ? <Tag color="red">已撤销</Tag> : <Tag color="green">有效</Tag>,
                      },
                      {
                        title: 'IP',
                        dataIndex: 'ip_address',
                        width: 140,
                        render: (value: string) => value || '-',
                      },
                      {
                        title: '最后使用',
                        dataIndex: 'last_used_at',
                        width: 180,
                        render: (value?: string | null) => (value ? formatDateTime(value) : '-'),
                      },
                      {
                        title: '到期时间',
                        dataIndex: 'expires_at',
                        width: 180,
                        render: (value: string) => formatDateTime(value),
                      },
                      {
                        title: '客户端',
                        dataIndex: 'user_agent',
                        ellipsis: true,
                        render: (value: string) => value || '-',
                      },
                    ]}
                  />
                ) : (
                  <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="暂无会话记录" />
                ),
              },
              {
                key: 'activities',
                label: <Space><HistoryOutlined /> 最近活动</Space>,
                children: activities.length > 0 ? (
                  <Table
                    size="small"
                    rowKey="id"
                    pagination={{ pageSize: 8 }}
                    dataSource={activities}
                    columns={[
                      { title: '时间', dataIndex: 'created_at', width: 180, render: (value: string) => formatDateTime(value) },
                      { title: '动作', dataIndex: 'action', width: 180 },
                      {
                        title: '资源',
                        width: 140,
                        render: (_, record) => record.resource_type || '-',
                      },
                      {
                        title: '详情',
                        dataIndex: 'details',
                        ellipsis: true,
                        render: (value?: string | null) => value || '-',
                      },
                      {
                        title: 'IP',
                        dataIndex: 'ip_address',
                        width: 140,
                        render: (value?: string | null) => value || '-',
                      },
                    ]}
                  />
                ) : (
                  <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="暂无活动记录" />
                ),
              },
              {
                key: 'tunnels',
                label: <Space><LinkOutlined /> 最近隧道</Space>,
                children: recentTunnels.length > 0 ? (
                  <Table
                    size="small"
                    rowKey="id"
                    pagination={{ pageSize: 8 }}
                    dataSource={recentTunnels}
                    columns={[
                      { title: 'ID', dataIndex: 'id', width: 80 },
                      { title: '名称', dataIndex: 'name', width: 180 },
                      { title: '协议', dataIndex: 'protocol', width: 120 },
                      {
                        title: '状态',
                        dataIndex: 'status',
                        width: 100,
                        render: (value: string) => <Tag color={value === 'running' ? 'green' : 'default'}>{value}</Tag>,
                      },
                      {
                        title: '更新时间',
                        dataIndex: 'updated_at',
                        width: 180,
                        render: (value: string) => formatDateTime(value),
                      },
                    ]}
                  />
                ) : (
                  <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="暂无隧道数据" />
                ),
              },
              {
                key: 'node-groups',
                label: <Space><ClusterOutlined /> 最近节点组</Space>,
                children: recentNodeGroups.length > 0 ? (
                  <Table
                    size="small"
                    rowKey="id"
                    pagination={{ pageSize: 8 }}
                    dataSource={recentNodeGroups}
                    columns={[
                      { title: 'ID', dataIndex: 'id', width: 80 },
                      { title: '名称', dataIndex: 'name', width: 220 },
                      {
                        title: '类型',
                        dataIndex: 'type',
                        width: 120,
                        render: (value: string) => <Tag color={value === 'entry' ? 'blue' : 'purple'}>{value}</Tag>,
                      },
                      {
                        title: '状态',
                        dataIndex: 'is_enabled',
                        width: 100,
                        render: (value: boolean) =>
                          value ? <Tag color="green">启用</Tag> : <Tag color="red">禁用</Tag>,
                      },
                      {
                        title: '更新时间',
                        dataIndex: 'updated_at',
                        width: 180,
                        render: (value: string) => formatDateTime(value),
                      },
                    ]}
                  />
                ) : (
                  <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="暂无节点组数据" />
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
