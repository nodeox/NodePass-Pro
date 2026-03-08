import {
  Button,
  Card,
  Col,
  Descriptions,
  Modal,
  Progress,
  Row,
  Select,
  Space,
  Statistic,
  Table,
  Tabs,
  Tag,
  Typography,
  message,
} from 'antd'
import type { TableProps, TabsProps } from 'antd'
import dayjs from 'dayjs'
import { useCallback, useEffect, useMemo, useState } from 'react'
import { useLocation, useNavigate, useParams } from 'react-router-dom'

import { usePageTitle } from '../../hooks/usePageTitle'
import { nodeGroupApi, nodeInstanceApi, tunnelApi } from '../../services/nodeGroupApi'
import type { NodeGroup, NodeGroupRelation, NodeInstance, Tunnel } from '../../types/nodeGroup'
import { getErrorMessage } from '../../utils/error'
import HealthCheck from './components/HealthCheck'
import MonitoringDashboard from './components/MonitoringDashboard'

const formatBytes = (bytes: number): string => {
  if (!Number.isFinite(bytes) || bytes <= 0) {
    return '0 B'
  }

  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  let value = bytes
  let index = 0

  while (value >= 1024 && index < units.length - 1) {
    value /= 1024
    index += 1
  }

  return `${value.toFixed(index === 0 ? 0 : 2)} ${units[index]}`
}

const clampPercent = (value: number): number => {
  if (!Number.isFinite(value)) {
    return 0
  }
  return Math.max(0, Math.min(100, Math.round(value)))
}

const renderNodeStatusTag = (status: NodeInstance['status']) => {
  if (status === 'online') {
    return <Tag color="green">在线</Tag>
  }
  if (status === 'maintain') {
    return <Tag color="orange">维护</Tag>
  }
  return <Tag color="red">离线</Tag>
}

const renderTunnelStatusTag = (status: Tunnel['status']) => {
  if (status === 'running') {
    return <Tag color="green">运行中</Tag>
  }
  if (status === 'error') {
    return <Tag color="red">异常</Tag>
  }
  return <Tag>已停止</Tag>
}

const NodeGroupDetail = () => {
  usePageTitle('节点组详情')

  const navigate = useNavigate()
  const location = useLocation()
  const { id } = useParams<{ id: string }>()
  const groupID = Number(id)

  const [loading, setLoading] = useState<boolean>(false)
  const [relationSubmitting, setRelationSubmitting] = useState<boolean>(false)
  const [relationModalOpen, setRelationModalOpen] = useState<boolean>(false)
  const [selectedExitGroupID, setSelectedExitGroupID] = useState<number | undefined>(undefined)
  const [group, setGroup] = useState<NodeGroup | null>(null)
  const [nodes, setNodes] = useState<NodeInstance[]>([])
  const [tunnels, setTunnels] = useState<Tunnel[]>([])
  const [relations, setRelations] = useState<NodeGroupRelation[]>([])
  const [exitGroups, setExitGroups] = useState<NodeGroup[]>([])

  const buildTunnelDetailPath = useCallback(
    (tunnelID: number) => {
      if (location.pathname.startsWith('/admin/')) {
        return `/admin/tunnels/${tunnelID}`
      }
      if (location.pathname.startsWith('/user/')) {
        return `/user/tunnels/${tunnelID}`
      }
      return `/tunnels/${tunnelID}`
    },
    [location.pathname],
  )

  const loadData = useCallback(
    async (silent = false) => {
      if (!Number.isFinite(groupID) || groupID <= 0) {
        return
      }

      if (!silent) {
        setLoading(true)
      }

      try {
        const [detail, stats, nodeList, tunnelList, relationList, exitGroupList] = await Promise.all([
          nodeGroupApi.get(groupID),
          nodeGroupApi.getStats(groupID),
          nodeGroupApi.listNodes(groupID),
          tunnelApi.list({ page: 1, page_size: 500 }),
          nodeGroupApi.listRelations(groupID),
          nodeGroupApi.list({ type: 'exit', enabled: true, page: 1, page_size: 500 }),
        ])

        const tunnelItems = tunnelList.items ?? []
        const relationItems = Array.isArray(relationList) ? relationList : []
        const exitItems = Array.isArray(exitGroupList.items) ? exitGroupList.items : []
        setGroup({ ...detail, stats: stats ?? detail.stats })
        setNodes(Array.isArray(nodeList) ? nodeList : [])
        setRelations(relationItems)
        setExitGroups(exitItems)
        setTunnels(
          tunnelItems.filter(
            (item) => item.entry_group_id === groupID || item.exit_group_id === groupID,
          ),
        )
      } catch (error) {
        message.error(getErrorMessage(error, '加载节点组详情失败'))
      } finally {
        if (!silent) {
          setLoading(false)
        }
      }
    },
    [groupID],
  )

  useEffect(() => {
    void loadData()

    const timer = window.setInterval(() => {
      void loadData(true)
    }, 30_000)

    return () => {
      window.clearInterval(timer)
    }
  }, [loadData])

  const handleToggleGroup = async () => {
    if (!group) {
      return
    }
    try {
      await nodeGroupApi.toggle(group.id)
      message.success(group.is_enabled ? '节点组已禁用' : '节点组已启用')
      await loadData(true)
    } catch (error) {
      message.error(getErrorMessage(error, '切换节点组状态失败'))
    }
  }

  const handleRestartNode = useCallback(async (instance: NodeInstance) => {
    try {
      await nodeInstanceApi.restart(instance.id)
      message.success(`节点 ${instance.name} 已重启`)
      await loadData(true)
    } catch (error) {
      message.error(getErrorMessage(error, '重启节点失败'))
    }
  }, [loadData])

  const handleToggleNode = useCallback(async (instance: NodeInstance) => {
    try {
      await nodeInstanceApi.update(instance.id, { is_enabled: !instance.is_enabled })
      message.success(instance.is_enabled ? '节点已禁用' : '节点已启用')
      await loadData(true)
    } catch (error) {
      message.error(getErrorMessage(error, '更新节点状态失败'))
    }
  }, [loadData])

  const handleDeleteNode = useCallback((instance: NodeInstance) => {
    Modal.confirm({
      title: '删除节点实例',
      content: `确认删除节点 ${instance.name} 吗？`,
      okText: '删除',
      okType: 'danger',
      cancelText: '取消',
      onOk: async () => {
        try {
          await nodeInstanceApi.delete(instance.id)
          message.success('节点实例已删除')
          await loadData(true)
        } catch (error) {
          message.error(getErrorMessage(error, '删除节点实例失败'))
        }
      },
    })
  }, [loadData])

  const handleTunnelToggle = useCallback(async (tunnel: Tunnel) => {
    try {
      if (tunnel.status === 'running') {
        await tunnelApi.stop(tunnel.id)
        message.success('隧道已停止')
      } else {
        await tunnelApi.start(tunnel.id)
        message.success('隧道已启动')
      }
      await loadData(true)
    } catch (error) {
      message.error(getErrorMessage(error, '更新隧道状态失败'))
    }
  }, [loadData])

  const relatedExitGroupIDs = useMemo(
    () =>
      new Set(
        relations
          .filter((item) => item.entry_group_id === groupID)
          .map((item) => item.exit_group_id),
      ),
    [relations, groupID],
  )

  const availableExitGroups = useMemo(() => {
    if (group?.type !== 'entry') {
      return []
    }
    return exitGroups.filter((item) => item.id !== groupID && !relatedExitGroupIDs.has(item.id))
  }, [exitGroups, groupID, group?.type, relatedExitGroupIDs])

  const handleCreateRelation = async () => {
    if (!group || group.type !== 'entry') {
      return
    }
    if (!selectedExitGroupID) {
      message.warning('请选择要关联的出口组')
      return
    }

    setRelationSubmitting(true)
    try {
      await nodeGroupApi.createRelation(group.id, { exit_group_id: selectedExitGroupID })
      message.success('出口组关联创建成功')
      setRelationModalOpen(false)
      setSelectedExitGroupID(undefined)
      await loadData(true)
    } catch (error) {
      message.error(getErrorMessage(error, '创建关联失败'))
    } finally {
      setRelationSubmitting(false)
    }
  }

  const handleToggleRelation = useCallback(async (relation: NodeGroupRelation) => {
    try {
      await nodeGroupApi.toggleRelation(relation.id)
      message.success(relation.is_enabled ? '关联已禁用' : '关联已启用')
      await loadData(true)
    } catch (error) {
      message.error(getErrorMessage(error, '切换关联状态失败'))
    }
  }, [loadData])

  const handleDeleteRelation = useCallback(async (relation: NodeGroupRelation) => {
    Modal.confirm({
      title: '删除关联',
      content: '删除后该入口组将不能再使用此出口组，确认继续吗？',
      okText: '删除',
      okType: 'danger',
      cancelText: '取消',
      onOk: async () => {
        try {
          await nodeGroupApi.deleteRelation(relation.id)
          message.success('关联已删除')
          await loadData(true)
        } catch (error) {
          message.error(getErrorMessage(error, '删除关联失败'))
        }
      },
    })
  }, [loadData])

  const summary = useMemo(() => {
    const totalNodes = group?.stats?.total_nodes ?? nodes.length
    const onlineNodes =
      group?.stats?.online_nodes ?? nodes.filter((item) => item.status === 'online').length
    const trafficIn = group?.stats?.total_traffic_in ?? 0
    const trafficOut = group?.stats?.total_traffic_out ?? 0
    const totalConnections = group?.stats?.total_connections ?? 0
    const onlineRate = totalNodes > 0 ? Math.round((onlineNodes / totalNodes) * 100) : 0

    return {
      totalNodes,
      onlineNodes,
      trafficIn,
      trafficOut,
      totalConnections,
      onlineRate,
    }
  }, [group, nodes])

  const nodeColumns = useMemo<TableProps<NodeInstance>['columns']>(
    () => [
      {
        title: '名称',
        dataIndex: 'name',
        key: 'name',
      },
      {
        title: 'Node ID',
        dataIndex: 'node_id',
        key: 'node_id',
      },
      {
        title: '主机:端口',
        key: 'endpoint',
        render: (_: unknown, record: NodeInstance) => {
          if (!record.host || !record.port) {
            return '-'
          }
          return `${record.host}:${record.port}`
        },
      },
      {
        title: '状态',
        dataIndex: 'status',
        key: 'status',
        width: 100,
        render: (status: NodeInstance['status']) => renderNodeStatusTag(status),
      },
      {
        title: 'CPU/内存',
        key: 'usage',
        width: 220,
        render: (_: unknown, record: NodeInstance) => {
          const cpu = record.system_info?.cpu_usage ?? 0
          const memory = record.system_info?.memory_usage ?? 0

          return (
            <Space direction="vertical" size={4} style={{ width: '100%' }}>
              <span>CPU</span>
              <Progress percent={clampPercent(cpu)} size="small" />
              <span>内存</span>
              <Progress percent={clampPercent(memory)} size="small" status="active" />
            </Space>
          )
        },
      },
      {
        title: '最后心跳',
        dataIndex: 'last_heartbeat_at',
        key: 'last_heartbeat_at',
        width: 180,
        render: (value: string | null) =>
          value ? dayjs(value).format('YYYY-MM-DD HH:mm:ss') : '-',
      },
      {
        title: '操作',
        key: 'actions',
        width: 220,
        render: (_: unknown, record: NodeInstance) => (
          <Space size="small" wrap>
            <Button type="link" onClick={() => void handleRestartNode(record)}>
              重启
            </Button>
            <Button type="link" onClick={() => void handleToggleNode(record)}>
              {record.is_enabled ? '禁用' : '启用'}
            </Button>
            <Button type="link" danger onClick={() => handleDeleteNode(record)}>
              删除
            </Button>
          </Space>
        ),
      },
    ],
    [handleDeleteNode, handleRestartNode, handleToggleNode],
  )

  const tunnelColumns = useMemo<TableProps<Tunnel>['columns']>(
    () => [
      {
        title: '隧道名称',
        dataIndex: 'name',
        key: 'name',
      },
      {
        title: '协议',
        dataIndex: 'protocol',
        key: 'protocol',
      },
      {
        title: '远程地址',
        key: 'remote',
        render: (_: unknown, record: Tunnel) => `${record.remote_host}:${record.remote_port}`,
      },
      {
        title: '状态',
        dataIndex: 'status',
        key: 'status',
        render: (status: Tunnel['status']) => renderTunnelStatusTag(status),
      },
      {
        title: '操作',
        key: 'actions',
        render: (_: unknown, record: Tunnel) => (
          <Space size="small">
            <Button type="link" onClick={() => void handleTunnelToggle(record)}>
              {record.status === 'running' ? '停止' : '启动'}
            </Button>
            <Button type="link" onClick={() => navigate(buildTunnelDetailPath(record.id))}>
              查看
            </Button>
          </Space>
        ),
      },
    ],
    [buildTunnelDetailPath, handleTunnelToggle, navigate],
  )

  const relationColumns = useMemo<TableProps<NodeGroupRelation>['columns']>(
    () => [
      {
        title: group?.type === 'entry' ? '出口组' : '入口组',
        key: 'linked_group',
        render: (_: unknown, record: NodeGroupRelation) => {
          if (group?.type === 'entry') {
            const linked = record.exit_group
            return linked?.name ?? `#${record.exit_group_id}`
          }
          const linked = record.entry_group
          return linked?.name ?? `#${record.entry_group_id}`
        },
      },
      {
        title: '状态',
        dataIndex: 'is_enabled',
        key: 'is_enabled',
        width: 110,
        render: (enabled: boolean) =>
          enabled ? <Tag color="green">已启用</Tag> : <Tag color="default">已禁用</Tag>,
      },
      {
        title: '创建时间',
        dataIndex: 'created_at',
        key: 'created_at',
        width: 180,
        render: (value: string) => dayjs(value).format('YYYY-MM-DD HH:mm:ss'),
      },
      {
        title: '操作',
        key: 'actions',
        width: 160,
        render: (_: unknown, record: NodeGroupRelation) =>
          group?.type === 'entry' ? (
            <Space size="small">
              <Button type="link" onClick={() => void handleToggleRelation(record)}>
                {record.is_enabled ? '禁用' : '启用'}
              </Button>
              <Button type="link" danger onClick={() => void handleDeleteRelation(record)}>
                删除
              </Button>
            </Space>
          ) : (
            '-'
          ),
      },
    ],
    [group?.type, handleDeleteRelation, handleToggleRelation],
  )

  const tabItems = useMemo<TabsProps['items']>(() => {
    const config = group?.config
    const descriptionRows = [
      {
        key: 'allowed_protocols',
        label: '允许协议',
        children:
          config?.allowed_protocols?.length && config.allowed_protocols.length > 0
            ? config.allowed_protocols.map((item) => item.toUpperCase()).join(', ')
            : '-',
      },
      {
        key: 'port_range',
        label: '端口范围',
        children:
          config?.port_range &&
          Number.isFinite(config.port_range.start) &&
          Number.isFinite(config.port_range.end)
            ? `${config.port_range.start} - ${config.port_range.end}`
            : '-',
      },
    ] as Array<{ key: string; label: string; children: string | number }>

    if (group?.type === 'entry') {
      descriptionRows.push(
        {
          key: 'require_exit_group',
          label: '要求出口组',
          children: config?.entry_config?.require_exit_group ? '是' : '否',
        },
        {
          key: 'traffic_multiplier',
          label: '流量倍率',
          children: config?.entry_config?.traffic_multiplier ?? '-',
        },
        {
          key: 'dns_load_balance',
          label: 'DNS 负载均衡',
          children: config?.entry_config?.dns_load_balance ? '开启' : '关闭',
        },
      )
    }

    if (group?.type === 'exit') {
      descriptionRows.push(
        {
          key: 'load_balance_strategy',
          label: '负载均衡策略',
          children: config?.exit_config?.load_balance_strategy ?? '-',
        },
        {
          key: 'health_check_interval',
          label: '健康检查间隔',
          children: config?.exit_config?.health_check_interval
            ? `${config.exit_config.health_check_interval} 秒`
            : '-',
        },
        {
          key: 'health_check_timeout',
          label: '健康检查超时',
          children: config?.exit_config?.health_check_timeout
            ? `${config.exit_config.health_check_timeout} 秒`
            : '-',
        },
      )
    }

    return [
      {
        key: 'overview',
        label: '概览',
        children: (
          <Row gutter={[16, 16]}>
            <Col xs={24} sm={12} md={6}>
              <Card>
                <Statistic title="总节点数" value={summary.totalNodes} />
              </Card>
            </Col>
            <Col xs={24} sm={12} md={6}>
              <Card>
                <Statistic title="在线节点数" value={summary.onlineNodes} />
              </Card>
            </Col>
            <Col xs={24} sm={12} md={6}>
              <Card>
                <Statistic title="总上行流量" value={formatBytes(summary.trafficIn)} />
              </Card>
            </Col>
            <Col xs={24} sm={12} md={6}>
              <Card>
                <Statistic title="总下行流量" value={formatBytes(summary.trafficOut)} />
              </Card>
            </Col>
          </Row>
        ),
      },
      {
        key: 'health',
        label: '健康检查',
        children: group ? <HealthCheck group={group} /> : null,
      },
      {
        key: 'monitoring',
        label: '实时监控',
        children: group ? <MonitoringDashboard group={group} /> : null,
      },
      {
        key: 'nodes',
        label: '节点列表',
        children: (
          <Table<NodeInstance>
            rowKey="id"
            dataSource={nodes}
            columns={nodeColumns}
            pagination={{ pageSize: 10, showSizeChanger: true }}
            scroll={{ x: 1100 }}
          />
        ),
      },
      {
        key: 'config',
        label: '配置信息',
        children: (
          <Descriptions bordered column={1} items={descriptionRows} />
        ),
      },
      {
        key: 'tunnels',
        label: '关联隧道',
        children: (
          <Table<Tunnel>
            rowKey="id"
            dataSource={tunnels}
            columns={tunnelColumns}
            pagination={{ pageSize: 10, showSizeChanger: true }}
          />
        ),
      },
      {
        key: 'relations',
        label: group?.type === 'entry' ? `关联出口组 (${relations.length})` : `关联入口组 (${relations.length})`,
        children: (
          <Space direction="vertical" size={12} style={{ width: '100%' }}>
            {group?.type === 'entry' ? (
              <Button
                type="primary"
                onClick={() => {
                  setSelectedExitGroupID(undefined)
                  setRelationModalOpen(true)
                }}
              >
                关联出口组
              </Button>
            ) : null}
            <Table<NodeGroupRelation>
              rowKey="id"
              dataSource={relations}
              columns={relationColumns}
              pagination={{ pageSize: 10, showSizeChanger: true }}
            />
          </Space>
        ),
      },
    ]
  }, [group, nodeColumns, tunnelColumns, tunnels, nodes, relations, relationColumns, summary])

  if (!Number.isFinite(groupID) || groupID <= 0) {
    return (
      <Card>
        <Space direction="vertical" size={12}>
          <Typography.Text type="danger">无效的节点组 ID</Typography.Text>
          <Button onClick={() => navigate('/node-groups')}>返回列表</Button>
        </Space>
      </Card>
    )
  }

  return (
    <Space direction="vertical" size={16} className="w-full">
      <Card loading={loading}>
        <Row justify="space-between" align="middle" gutter={[16, 16]}>
          <Col>
            <Space size={12} wrap>
              <Typography.Title level={4} style={{ margin: 0 }}>
                {group?.name ?? '-'}
              </Typography.Title>
              {group?.type === 'entry' ? <Tag color="blue">入口组</Tag> : <Tag color="green">出口组</Tag>}
              {group?.is_enabled ? (
                <Tag color="green">已启用</Tag>
              ) : (
                <Tag color="default">已禁用</Tag>
              )}
            </Space>
          </Col>
          <Col>
            <Space wrap>
              <Button onClick={() => navigate(`/node-groups/${groupID}/edit`)}>编辑</Button>
              <Button onClick={() => void handleToggleGroup()}>
                {group?.is_enabled ? '禁用' : '启用'}
              </Button>
              <Button type="primary" onClick={() => navigate(`/node-groups/${groupID}/deploy`)}>
                部署新节点
              </Button>
            </Space>
          </Col>
        </Row>
      </Card>

      <Card loading={loading}>
        <Tabs items={tabItems} />
      </Card>

      <Modal
        title="关联出口组"
        open={relationModalOpen}
        onCancel={() => {
          if (!relationSubmitting) {
            setRelationModalOpen(false)
          }
        }}
        onOk={() => void handleCreateRelation()}
        confirmLoading={relationSubmitting}
        okText="确认关联"
        cancelText="取消"
        destroyOnClose
      >
        <Space direction="vertical" size={12} style={{ width: '100%' }}>
          <Typography.Text type="secondary">
            仅显示已启用且未关联的出口组。
          </Typography.Text>
          <Select<number>
            value={selectedExitGroupID}
            onChange={(value) => setSelectedExitGroupID(value)}
            options={availableExitGroups.map((item) => ({
              label: `${item.name} (#${item.id})`,
              value: item.id,
            }))}
            placeholder={availableExitGroups.length > 0 ? '请选择出口组' : '暂无可关联的出口组'}
            style={{ width: '100%' }}
            showSearch
            optionFilterProp="label"
          />
        </Space>
      </Modal>
    </Space>
  )
}

export default NodeGroupDetail
