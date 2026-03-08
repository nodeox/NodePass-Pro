import {
  DeleteOutlined,
  MoreOutlined,
  PoweroffOutlined,
  ReloadOutlined,
  RetweetOutlined,
} from '@ant-design/icons'
import {
  Alert,
  Button,
  Dropdown,
  Segmented,
  Space,
  Table,
  Tag,
  Typography,
  message,
} from 'antd'
import type { TableProps } from 'antd'
import { useCallback, useEffect, useMemo, useState } from 'react'

import PageContainer from '../../components/common/PageContainer'
import { usePageTitle } from '../../hooks/usePageTitle'
import { nodeGroupApi, nodeInstanceApi } from '../../services/nodeGroupApi'
import type { AccessibleNodeGroup, NodeInstance } from '../../types/nodeGroup'
import { getErrorMessage } from '../../utils/error'
import { formatDateTime } from '../../utils/format'

type FilterKey = 'all' | 'own' | 'public'

type NodeStatusRow = NodeInstance & {
  group_id: number
  group_name: string
  group_type: 'entry' | 'exit'
  editable: boolean
  is_public: boolean
}

const renderStatusTag = (status: NodeInstance['status']) => {
  if (status === 'online') {
    return <Tag color="green">在线</Tag>
  }
  if (status === 'maintain') {
    return <Tag color="orange">维护</Tag>
  }
  return <Tag color="red">离线</Tag>
}

const endpointText = (node: NodeInstance): string => {
  if (!node.host || !node.port) {
    return '未上报'
  }
  return `${node.host}:${node.port}`
}

const NodeStatus = () => {
  usePageTitle('节点状态')

  const [loading, setLoading] = useState(false)
  const [actionLoading, setActionLoading] = useState<string | null>(null)
  const [filter, setFilter] = useState<FilterKey>('all')
  const [source, setSource] = useState<AccessibleNodeGroup[]>([])

  const loadData = useCallback(async (silent = false) => {
    if (!silent) {
      setLoading(true)
    }
    try {
      const result = await nodeGroupApi.accessibleNodes()
      setSource(result.items ?? [])
    } catch (error) {
      message.error(getErrorMessage(error, '节点状态加载失败'))
    } finally {
      if (!silent) {
        setLoading(false)
      }
    }
  }, [])

  useEffect(() => {
    void loadData()
    const timer = window.setInterval(() => {
      void loadData(true)
    }, 30_000)
    return () => {
      window.clearInterval(timer)
    }
  }, [loadData])

  const rows = useMemo<NodeStatusRow[]>(() => {
    const merged: NodeStatusRow[] = []

    source.forEach((item) => {
      if (filter === 'own' && item.is_public) {
        return
      }
      if (filter === 'public' && !item.is_public) {
        return
      }

      item.nodes.forEach((node) => {
        merged.push({
          ...node,
          group_id: item.group.id,
          group_name: item.group.name,
          group_type: item.group.type,
          editable: item.editable,
          is_public: item.is_public,
        })
      })
    })

    return merged
  }, [filter, source])

  const stats = useMemo(() => {
    const ownGroups = source.filter((item) => !item.is_public)
    const publicGroups = source.filter((item) => item.is_public)

    return {
      ownGroups: ownGroups.length,
      publicGroups: publicGroups.length,
      ownNodes: ownGroups.reduce((acc, item) => acc + item.nodes.length, 0),
      publicNodes: publicGroups.reduce((acc, item) => acc + item.nodes.length, 0),
    }
  }, [source])

  const runAction = async (key: string, action: () => Promise<void>, successText: string) => {
    setActionLoading(key)
    try {
      await action()
      message.success(successText)
      await loadData(true)
    } catch (error) {
      message.error(getErrorMessage(error, '操作失败'))
    } finally {
      setActionLoading(null)
    }
  }

  const restartNode = (record: NodeStatusRow) => {
    if (!record.editable) {
      return
    }
    void runAction(
      `restart-${record.id}`,
      async () => {
        await nodeInstanceApi.restart(record.id)
      },
      `节点 ${record.name} 已重启`,
    )
  }

  const toggleNode = (record: NodeStatusRow) => {
    if (!record.editable) {
      return
    }
    void runAction(
      `toggle-${record.id}`,
      async () => {
        await nodeInstanceApi.update(record.id, { is_enabled: !record.is_enabled })
      },
      record.is_enabled ? '节点已禁用' : '节点已启用',
    )
  }

  const deleteNode = (record: NodeStatusRow) => {
    if (!record.editable) {
      return
    }
    void runAction(
      `delete-${record.id}`,
      async () => {
        await nodeInstanceApi.delete(record.id)
      },
      '节点实例已删除',
    )
  }

  const columns = useMemo<TableProps<NodeStatusRow>['columns']>(
    () => [
      {
        title: '节点名称',
        dataIndex: 'name',
        width: 170,
        render: (value: string) => (
          <Typography.Text ellipsis style={{ maxWidth: 150 }}>
            {value}
          </Typography.Text>
        ),
      },
      {
        title: '节点组',
        width: 190,
        render: (_, record) => (
          <Space size={6}>
            <Typography.Text ellipsis style={{ maxWidth: 90 }}>
              {record.group_name}
            </Typography.Text>
            <Tag color={record.group_type === 'entry' ? 'blue' : 'green'}>
              {record.group_type === 'entry' ? '入口' : '出口'}
            </Tag>
            {record.is_public ? <Tag color="default">公共</Tag> : <Tag color="cyan">自托管</Tag>}
          </Space>
        ),
      },
      {
        title: '连接地址',
        key: 'endpoint',
        width: 170,
        render: (_, record) => endpointText(record),
      },
      {
        title: '状态',
        dataIndex: 'status',
        width: 90,
        render: (status: NodeInstance['status']) => renderStatusTag(status),
      },
      {
        title: '最后心跳',
        dataIndex: 'last_heartbeat_at',
        width: 168,
        render: (value: string | null) => (value ? formatDateTime(value) : '-'),
      },
      {
        title: '权限',
        key: 'permission',
        width: 90,
        render: (_, record) =>
          record.editable ? <Tag color="green">可编辑</Tag> : <Tag color="default">只读</Tag>,
      },
      {
        title: '操作',
        key: 'actions',
        fixed: 'right',
        width: 220,
        align: 'center',
        render: (_, record) => {
          if (!record.editable) {
            return <Typography.Text type="secondary">公共节点仅查看</Typography.Text>
          }

          return (
            <Space size={6} style={{ whiteSpace: 'nowrap' }}>
              <Button
                size="small"
                icon={<RetweetOutlined />}
                loading={actionLoading === `restart-${record.id}`}
                onClick={() => restartNode(record)}
              >
                重启
              </Button>
              <Dropdown
                trigger={['click']}
                menu={{
                  items: [
                    {
                      key: 'toggle',
                      icon: <PoweroffOutlined />,
                      label: record.is_enabled ? '禁用' : '启用',
                    },
                    {
                      key: 'delete',
                      icon: <DeleteOutlined />,
                      label: '删除',
                      danger: true,
                    },
                  ],
                  onClick: ({ key }) => {
                    if (key === 'toggle') {
                      toggleNode(record)
                      return
                    }
                    if (key === 'delete') {
                      deleteNode(record)
                    }
                  },
                }}
              >
                <Button size="small" icon={<MoreOutlined />}>
                  更多
                </Button>
              </Dropdown>
            </Space>
          )
        },
      },
    ],
    [actionLoading],
  )

  return (
    <PageContainer
      title="节点状态"
      description="查看你有权访问的节点详情。公共节点仅允许查看，不允许编辑。"
      extra={
        <Button icon={<ReloadOutlined />} onClick={() => void loadData()} loading={loading}>
          刷新
        </Button>
      }
    >
      <Space style={{ marginBottom: 12 }} wrap>
        <Tag color="cyan">自托管节点组 {stats.ownGroups}</Tag>
        <Tag color="default">公共节点组 {stats.publicGroups}</Tag>
        <Tag color="cyan">自托管节点 {stats.ownNodes}</Tag>
        <Tag color="default">公共节点 {stats.publicNodes}</Tag>
      </Space>

      <Alert
        type="info"
        showIcon
        style={{ marginBottom: 12 }}
        message="公共节点为只读数据，不能进行重启、禁用、删除等操作。"
      />

      <Space style={{ marginBottom: 12 }}>
        <Segmented<FilterKey>
          value={filter}
          options={[
            { label: '全部', value: 'all' },
            { label: '自托管', value: 'own' },
            { label: '公共节点', value: 'public' },
          ]}
          onChange={(value) => setFilter(value)}
        />
      </Space>

      <Table<NodeStatusRow>
        rowKey="id"
        size="small"
        loading={loading}
        dataSource={rows}
        columns={columns}
        scroll={{ x: 1320 }}
        pagination={{
          pageSize: 20,
          showSizeChanger: true,
          showTotal: (total) => `共 ${total} 条`,
        }}
      />
    </PageContainer>
  )
}

export default NodeStatus
