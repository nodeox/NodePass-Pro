import {
  CopyOutlined,
  DeleteOutlined,
  EditOutlined,
  EyeOutlined,
  MoreOutlined,
  PoweroffOutlined,
  ReloadOutlined,
  UploadOutlined,
} from '@ant-design/icons'
import {
  Button,
  Card,
  Dropdown,
  Modal,
  Space,
  Spin,
  Table,
  Tabs,
  Tag,
  Tooltip,
  Typography,
  message,
} from 'antd'
import type { TabsProps, TableProps } from 'antd'
import dayjs from 'dayjs'
import { useCallback, useEffect, useMemo, useState } from 'react'
import { useNavigate } from 'react-router-dom'

import { usePageTitle } from '../../hooks/usePageTitle'
import { nodeGroupApi } from '../../services/nodeGroupApi'
import type { NodeGroup } from '../../types/nodeGroup'
import { getErrorMessage } from '../../utils/error'

type TabKey = 'all' | 'entry' | 'exit'

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

  const precision = index === 0 ? 0 : 2
  return `${value.toFixed(precision)} ${units[index]}`
}

const NodeGroupsPage = () => {
  usePageTitle('节点组管理')

  const navigate = useNavigate()

  const [activeTab, setActiveTab] = useState<TabKey>('all')
  const [loading, setLoading] = useState<boolean>(false)
  const [items, setItems] = useState<NodeGroup[]>([])
  const [page, setPage] = useState<number>(1)
  const [pageSize, setPageSize] = useState<number>(10)
  const [total, setTotal] = useState<number>(0)
  const [selectedRowKeys, setSelectedRowKeys] = useState<React.Key[]>([])
  const [batchLoading, setBatchLoading] = useState<boolean>(false)

  const loadData = useCallback(async () => {
    setLoading(true)

    try {
      const params: {
        type?: string
        page?: number
        page_size?: number
      } = {
        page,
        page_size: pageSize,
      }

      if (activeTab !== 'all') {
        params.type = activeTab
      }

      const res = await nodeGroupApi.list(params)
      const rows = res.items ?? []

      setItems(rows)
      setTotal(res.total ?? rows.length)
      setPage(res.page ?? page)
      setPageSize(res.page_size ?? pageSize)
    } catch (error) {
      message.error(getErrorMessage(error, '节点组加载失败'))
    } finally {
      setLoading(false)
    }
  }, [activeTab, page, pageSize])

  useEffect(() => {
    void loadData()
  }, [loadData])

  const handleToggle = async (record: NodeGroup) => {
    try {
      await nodeGroupApi.toggle(record.id)
      message.success(record.is_enabled ? '已禁用节点组' : '已启用节点组')
      await loadData()
    } catch (error) {
      message.error(getErrorMessage(error, '切换节点组状态失败'))
    }
  }

  const handleDelete = (record: NodeGroup) => {
    Modal.confirm({
      title: '删除节点组',
      content: `确认删除节点组「${record.name}」吗？此操作不可恢复。`,
      okText: '删除',
      okType: 'danger',
      cancelText: '取消',
      onOk: async () => {
        try {
          await nodeGroupApi.delete(record.id)
          message.success('节点组已删除')
          await loadData()
        } catch (error) {
          message.error(getErrorMessage(error, '删除节点组失败'))
        }
      },
    })
  }

  const handleCopy = (record: NodeGroup) => {
    navigate('/node-groups/create', {
      state: {
        copyFrom: record,
      },
    })
  }

  const handleBatchEnable = async () => {
    if (selectedRowKeys.length === 0) {
      message.warning('请先选择要启用的节点组')
      return
    }
    setBatchLoading(true)
    try {
      const promises = selectedRowKeys.map((id) => {
        const item = items.find((g) => g.id === id)
        if (item && !item.is_enabled) {
          return nodeGroupApi.toggle(Number(id))
        }
        return Promise.resolve()
      })
      await Promise.all(promises)
      message.success(`已启用 ${selectedRowKeys.length} 个节点组`)
      setSelectedRowKeys([])
      await loadData()
    } catch (error) {
      message.error(getErrorMessage(error, '批量启用失败'))
    } finally {
      setBatchLoading(false)
    }
  }

  const handleBatchDisable = async () => {
    if (selectedRowKeys.length === 0) {
      message.warning('请先选择要禁用的节点组')
      return
    }
    setBatchLoading(true)
    try {
      const promises = selectedRowKeys.map((id) => {
        const item = items.find((g) => g.id === id)
        if (item && item.is_enabled) {
          return nodeGroupApi.toggle(Number(id))
        }
        return Promise.resolve()
      })
      await Promise.all(promises)
      message.success(`已禁用 ${selectedRowKeys.length} 个节点组`)
      setSelectedRowKeys([])
      await loadData()
    } catch (error) {
      message.error(getErrorMessage(error, '批量禁用失败'))
    } finally {
      setBatchLoading(false)
    }
  }

  const handleBatchDelete = () => {
    if (selectedRowKeys.length === 0) {
      message.warning('请先选择要删除的节点组')
      return
    }
    Modal.confirm({
      title: '批量删除节点组',
      content: `确认删除选中的 ${selectedRowKeys.length} 个节点组吗？此操作不可恢复。`,
      okText: '删除',
      okType: 'danger',
      cancelText: '取消',
      onOk: async () => {
        setBatchLoading(true)
        try {
          await Promise.all(
            selectedRowKeys.map((id) => nodeGroupApi.delete(Number(id)))
          )
          message.success(`已删除 ${selectedRowKeys.length} 个节点组`)
          setSelectedRowKeys([])
          await loadData()
        } catch (error) {
          message.error(getErrorMessage(error, '批量删除失败'))
        } finally {
          setBatchLoading(false)
        }
      },
    })
  }

  const tabItems = useMemo<TabsProps['items']>(
    () => [
      { key: 'all', label: '全部' },
      { key: 'entry', label: '入口组' },
      { key: 'exit', label: '出口组' },
    ],
    [],
  )

  const columns = useMemo<TableProps<NodeGroup>['columns']>(
    () => [
      {
        title: '名称',
        dataIndex: 'name',
        key: 'name',
        width: 220,
        render: (_: string, record: NodeGroup) => (
          <Button
            type="link"
            style={{ paddingInline: 0, maxWidth: '100%', justifyContent: 'flex-start' }}
            onClick={() => navigate(`/node-groups/${record.id}`)}
          >
            <Tooltip title={record.name}>
              <Typography.Text ellipsis style={{ maxWidth: 190 }}>
                {record.name}
              </Typography.Text>
            </Tooltip>
          </Button>
        ),
      },
      {
        title: '类型',
        dataIndex: 'type',
        key: 'type',
        width: 96,
        render: (value: NodeGroup['type']) =>
          value === 'entry' ? <Tag color="blue">入口组</Tag> : <Tag color="green">出口组</Tag>,
      },
      {
        title: '状态',
        dataIndex: 'is_enabled',
        key: 'is_enabled',
        width: 96,
        render: (enabled: boolean) =>
          enabled ? <Tag color="green">已启用</Tag> : <Tag color="default">已禁用</Tag>,
      },
      {
        title: '节点数',
        key: 'node_count',
        width: 110,
        render: (_: unknown, record: NodeGroup) => {
          const online = record.stats?.online_nodes ?? 0
          const totalNodes = record.stats?.total_nodes ?? 0
          return `${online} / ${totalNodes}`
        },
      },
      {
        title: '总流量',
        key: 'total_traffic',
        width: 130,
        render: (_: unknown, record: NodeGroup) => {
          const trafficIn = record.stats?.total_traffic_in ?? 0
          const trafficOut = record.stats?.total_traffic_out ?? 0
          return formatBytes(trafficIn + trafficOut)
        },
      },
      {
        title: '创建时间',
        dataIndex: 'created_at',
        key: 'created_at',
        width: 168,
        render: (value: string) => dayjs(value).format('YYYY-MM-DD HH:mm:ss'),
      },
      {
        title: '操作',
        key: 'actions',
        fixed: 'right',
        width: 240,
        align: 'center',
        render: (_: unknown, record: NodeGroup) => (
          <Space size={6} style={{ whiteSpace: 'nowrap' }}>
            <Button
              size="small"
              type="primary"
              icon={<UploadOutlined />}
              onClick={() => navigate(`/node-groups/${record.id}/deploy`)}
            >
              部署节点
            </Button>
            <Button
              size="small"
              icon={<EyeOutlined />}
              onClick={() => navigate(`/node-groups/${record.id}`)}
            >
              详情
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
                    key: 'edit',
                    icon: <EditOutlined />,
                    label: '编辑',
                  },
                  {
                    key: 'copy',
                    icon: <CopyOutlined />,
                    label: '复制',
                  },
                  {
                    type: 'divider',
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
                    void handleToggle(record)
                    return
                  }
                  if (key === 'edit') {
                    navigate(`/node-groups/${record.id}/edit`)
                    return
                  }
                  if (key === 'copy') {
                    handleCopy(record)
                    return
                  }
                  if (key === 'delete') {
                    handleDelete(record)
                  }
                },
              }}
            >
              <Button size="small" icon={<MoreOutlined />}>
                更多
              </Button>
            </Dropdown>
          </Space>
        ),
      },
    ],
    [navigate, handleToggle, handleDelete],
  )

  return (
    <Card
      className="shadow-sm"
      title={<Typography.Title level={4} style={{ margin: 0 }}>节点组管理</Typography.Title>}
      extra={
        <Space>
          {selectedRowKeys.length > 0 && (
            <>
              <Typography.Text type="secondary">
                已选 {selectedRowKeys.length} 项
              </Typography.Text>
              <Button
                icon={<PoweroffOutlined />}
                onClick={() => void handleBatchEnable()}
                loading={batchLoading}
              >
                批量启用
              </Button>
              <Button
                icon={<PoweroffOutlined />}
                onClick={() => void handleBatchDisable()}
                loading={batchLoading}
              >
                批量禁用
              </Button>
              <Button
                danger
                icon={<DeleteOutlined />}
                onClick={handleBatchDelete}
                loading={batchLoading}
              >
                批量删除
              </Button>
            </>
          )}
          <Button icon={<ReloadOutlined />} onClick={() => void loadData()} loading={loading}>
            刷新
          </Button>
          <Button type="primary" onClick={() => navigate('/node-groups/create')}>
            创建节点组
          </Button>
        </Space>
      }
    >
      <Tabs
        activeKey={activeTab}
        items={tabItems}
        onChange={(key) => {
          setActiveTab(key as TabKey)
          setPage(1)
        }}
      />

      <Spin spinning={loading}>
        <Table<NodeGroup>
          rowKey="id"
          dataSource={items}
          columns={columns}
          size="small"
          scroll={{ x: 980 }}
          rowSelection={{
            selectedRowKeys,
            onChange: setSelectedRowKeys,
          }}
          pagination={{
            current: page,
            pageSize,
            total,
            showSizeChanger: true,
            size: 'small',
            showTotal: (count) => `共 ${count} 条`,
            onChange: (nextPage, nextPageSize) => {
              setPage(nextPage)
              setPageSize(nextPageSize)
            },
          }}
        />
      </Spin>
    </Card>
  )
}

export default NodeGroupsPage
