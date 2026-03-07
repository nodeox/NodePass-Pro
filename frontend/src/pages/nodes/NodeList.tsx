import {
  DeleteOutlined,
  EditOutlined,
  EyeOutlined,
  PlusOutlined,
  ReloadOutlined,
} from '@ant-design/icons'
import {
  Button,
  Descriptions,
  Modal,
  Popconfirm,
  Progress,
  Select,
  Space,
  Table,
  Tabs,
  Tag,
  Typography,
  message,
} from 'antd'
import { useCallback, useEffect, useMemo, useState } from 'react'

import PageContainer from '../../components/common/PageContainer'
import { usePageTitle } from '../../hooks/usePageTitle'
import { nodeApi } from '../../services/api'
import { useAppStore } from '../../store/app'
import type { NodeQuotaInfo, NodeRecord } from '../../types'
import { getErrorMessage } from '../../utils/error'
import { formatDateTime } from '../../utils/format'
import NodeForm from './NodeForm'
import NodePairList from './NodePairList'

type TabKey = 'nodes' | 'pairs'
type NodeFormMode = 'create' | 'edit'
type SelfHostedFilter = 'all' | 'true' | 'false'

const statusMap: Record<
  string,
  { label: string; color: 'green' | 'red' | 'orange' | 'default' }
> = {
  online: { label: '在线', color: 'green' },
  offline: { label: '离线', color: 'red' },
  maintain: { label: '维护', color: 'orange' },
}

const regionOptions = [
  { label: '美国西部', value: 'us-west' },
  { label: '美国东部', value: 'us-east' },
  { label: '欧洲', value: 'europe' },
  { label: '亚洲东部', value: 'asia-east' },
  { label: '亚洲东南', value: 'asia-se' },
  { label: '中国香港', value: 'hk' },
]

const normalizePercent = (value?: number | null): number => {
  if (typeof value !== 'number' || Number.isNaN(value)) {
    return 0
  }
  if (value < 0) {
    return 0
  }
  if (value > 100) {
    return 100
  }
  return Number(value.toFixed(2))
}

const renderStatus = (status: string) => {
  const mapped = statusMap[status]
  if (!mapped) {
    return <Tag>{status}</Tag>
  }
  return <Tag color={mapped.color}>{mapped.label}</Tag>
}

const NodeList = () => {
  usePageTitle('节点管理')

  const [activeTab, setActiveTab] = useState<TabKey>('nodes')

  const [nodes, setNodes] = useState<NodeRecord[]>([])
  const [quota, setQuota] = useState<NodeQuotaInfo | null>(null)
  const [loading, setLoading] = useState<boolean>(false)

  const [statusFilter, setStatusFilter] = useState<string | undefined>()
  const [regionFilter, setRegionFilter] = useState<string | undefined>()
  const [selfHostedFilter, setSelfHostedFilter] =
    useState<SelfHostedFilter>('all')

  const [page, setPage] = useState<number>(1)
  const [pageSize, setPageSize] = useState<number>(10)
  const [total, setTotal] = useState<number>(0)

  const [formOpen, setFormOpen] = useState<boolean>(false)
  const [formMode, setFormMode] = useState<NodeFormMode>('create')
  const [editingNode, setEditingNode] = useState<NodeRecord | null>(null)

  const [detailNode, setDetailNode] = useState<NodeRecord | null>(null)
  const nodeStatusMap = useAppStore((state) => state.nodeStatusMap)

  const loadNodes = useCallback(
    async (targetPage = page, targetPageSize = pageSize): Promise<void> => {
      setLoading(true)
      try {
        const isSelfHosted =
          selfHostedFilter === 'all'
            ? undefined
            : selfHostedFilter === 'true'

        const [nodeResult, quotaResult] = await Promise.all([
          nodeApi.list({
            page: targetPage,
            pageSize: targetPageSize,
            status: statusFilter,
            region: regionFilter,
            is_self_hosted: isSelfHosted,
          }),
          nodeApi.quota(),
        ])

        setNodes(nodeResult.list ?? [])
        setTotal(nodeResult.total ?? 0)
        setPage(nodeResult.page || targetPage)
        setPageSize(nodeResult.page_size || targetPageSize)
        setQuota(quotaResult)
      } catch (error) {
        message.error(getErrorMessage(error, '节点数据加载失败'))
      } finally {
        setLoading(false)
      }
    },
    [page, pageSize, regionFilter, selfHostedFilter, statusFilter],
  )

  useEffect(() => {
    void loadNodes()
  }, [loadNodes])

  useEffect(() => {
    const ids = Object.keys(nodeStatusMap)
    if (ids.length === 0) {
      return
    }

    setNodes((current) => {
      let changed = false
      const next = current.map((node) => {
        const nextStatus = nodeStatusMap[node.id]
        if (!nextStatus || nextStatus === node.status) {
          return node
        }
        changed = true
        return {
          ...node,
          status: nextStatus,
        }
      })
      return changed ? next : current
    })

    setDetailNode((current) => {
      if (!current) {
        return current
      }
      const nextStatus = nodeStatusMap[current.id]
      if (!nextStatus || nextStatus === current.status) {
        return current
      }
      return {
        ...current,
        status: nextStatus,
      }
    })
  }, [nodeStatusMap])

  const quotaSummary = useMemo(() => {
    if (!quota) {
      return '配额加载中...'
    }
    const used = quota.used_self_hosted_nodes ?? 0
    return `入口: ${used}/${quota.max_self_hosted_entry_nodes}，出口: ${used}/${quota.max_self_hosted_exit_nodes}`
  }, [quota])

  const openCreateModal = (): void => {
    setFormMode('create')
    setEditingNode(null)
    setFormOpen(true)
  }

  const openEditModal = (record: NodeRecord): void => {
    setFormMode('edit')
    setEditingNode(record)
    setFormOpen(true)
  }

  const closeFormModal = (): void => {
    setFormOpen(false)
    setEditingNode(null)
  }

  const handleDelete = async (nodeID: number): Promise<void> => {
    try {
      await nodeApi.remove(nodeID)
      message.success('节点删除成功')
      if (nodes.length === 1 && page > 1) {
        setPage(page - 1)
        return
      }
      await loadNodes()
    } catch (error) {
      message.error(getErrorMessage(error, '删除节点失败'))
    }
  }

  const handleFormSuccess = async (): Promise<void> => {
    if (formMode === 'create' && page !== 1) {
      setPage(1)
      return
    }
    await loadNodes()
  }

  const tableRegionOptions = useMemo(() => {
    const dynamicOptions = nodes
      .map((item) => item.region)
      .filter((item): item is string => Boolean(item))
      .filter(
        (item, index, array) => array.findIndex((entry) => entry === item) === index,
      )
      .map((value) => ({ label: value, value }))

    const merged = [...regionOptions]
    dynamicOptions.forEach((item) => {
      if (!merged.some((region) => region.value === item.value)) {
        merged.push(item)
      }
    })
    return merged
  }, [nodes])

  return (
    <PageContainer
      title="节点管理"
      description="统一管理节点与入口/出口配对关系。"
      extra={
        <Space wrap>
          <Typography.Text type="secondary">{quotaSummary}</Typography.Text>
          <Button
            icon={<ReloadOutlined />}
            onClick={() => void loadNodes()}
            loading={loading}
          >
            刷新
          </Button>
          <Button type="primary" icon={<PlusOutlined />} onClick={openCreateModal}>
            创建节点
          </Button>
        </Space>
      }
    >
      <Tabs
        activeKey={activeTab}
        onChange={(key) => setActiveTab(key as TabKey)}
        items={[
          {
            key: 'nodes',
            label: '节点列表',
            children: (
              <>
                <Space wrap style={{ marginBottom: 16 }}>
                  <Select
                    allowClear
                    style={{ minWidth: 140 }}
                    placeholder="按状态筛选"
                    value={statusFilter}
                    options={[
                      { label: '在线', value: 'online' },
                      { label: '离线', value: 'offline' },
                      { label: '维护', value: 'maintain' },
                    ]}
                    onChange={(value) => {
                      setStatusFilter(value)
                      setPage(1)
                    }}
                  />

                  <Select
                    allowClear
                    style={{ minWidth: 160 }}
                    placeholder="按区域筛选"
                    value={regionFilter}
                    options={tableRegionOptions}
                    onChange={(value) => {
                      setRegionFilter(value)
                      setPage(1)
                    }}
                  />

                  <Select
                    style={{ minWidth: 160 }}
                    value={selfHostedFilter}
                    options={[
                      { label: '全部节点', value: 'all' },
                      { label: '仅自托管', value: 'true' },
                      { label: '仅平台节点', value: 'false' },
                    ]}
                    onChange={(value) => {
                      setSelfHostedFilter(value)
                      setPage(1)
                    }}
                  />
                </Space>

                <Table<NodeRecord>
                  rowKey="id"
                  loading={loading}
                  dataSource={nodes}
                  pagination={{
                    current: page,
                    pageSize,
                    total,
                    showSizeChanger: true,
                    showTotal: (recordTotal) => `共 ${recordTotal} 条`,
                    onChange: (nextPage, nextPageSize) => {
                      setPage(nextPage)
                      setPageSize(nextPageSize)
                    },
                  }}
                  columns={[
                    {
                      title: 'ID',
                      dataIndex: 'id',
                      width: 80,
                    },
                    {
                      title: '名称',
                      dataIndex: 'name',
                      width: 180,
                    },
                    {
                      title: '状态',
                      dataIndex: 'status',
                      width: 120,
                      render: (status: string) => renderStatus(status),
                    },
                    {
                      title: '区域',
                      dataIndex: 'region',
                      width: 120,
                      render: (region?: string | null) => region || '-',
                    },
                    {
                      title: '自托管',
                      dataIndex: 'is_self_hosted',
                      width: 110,
                      render: (isSelfHosted: boolean) =>
                        isSelfHosted ? (
                          <Tag color="blue">自托管</Tag>
                        ) : (
                          <Tag>平台</Tag>
                        ),
                    },
                    {
                      title: '流量倍率',
                      dataIndex: 'traffic_multiplier',
                      width: 110,
                      render: (value: number) => `${(value ?? 1).toFixed(2)}x`,
                    },
                    {
                      title: 'CPU / 内存',
                      width: 220,
                      render: (_, record) => (
                        <Space direction="vertical" size={4} className="w-full">
                          <div>
                            <Typography.Text type="secondary">CPU</Typography.Text>
                            <Progress
                              percent={normalizePercent(record.cpu_usage)}
                              showInfo={false}
                              size="small"
                            />
                          </div>
                          <div>
                            <Typography.Text type="secondary">内存</Typography.Text>
                            <Progress
                              percent={normalizePercent(record.memory_usage)}
                              showInfo={false}
                              size="small"
                            />
                          </div>
                        </Space>
                      ),
                    },
                    {
                      title: '最后心跳',
                      dataIndex: 'last_heartbeat_at',
                      width: 180,
                      render: (value?: string | null) => formatDateTime(value),
                    },
                    {
                      title: '操作',
                      fixed: 'right',
                      width: 230,
                      render: (_, record) => (
                        <Space>
                          <Button
                            type="link"
                            icon={<EyeOutlined />}
                            onClick={() => setDetailNode(record)}
                          >
                            详情
                          </Button>
                          <Button
                            type="link"
                            icon={<EditOutlined />}
                            onClick={() => openEditModal(record)}
                          >
                            编辑
                          </Button>
                          <Popconfirm
                            title="确定删除该节点吗？"
                            okText="删除"
                            cancelText="取消"
                            onConfirm={() => void handleDelete(record.id)}
                          >
                            <Button type="link" danger icon={<DeleteOutlined />}>
                              删除
                            </Button>
                          </Popconfirm>
                        </Space>
                      ),
                    },
                  ]}
                />
              </>
            ),
          },
          {
            key: 'pairs',
            label: '节点配对',
            children: <NodePairList />,
          },
        ]}
      />

      <NodeForm
        open={formOpen}
        mode={formMode}
        initialData={editingNode}
        quota={quota}
        onClose={closeFormModal}
        onSuccess={handleFormSuccess}
      />

      <Modal
        title={detailNode ? `节点详情 #${detailNode.id}` : '节点详情'}
        open={Boolean(detailNode)}
        onCancel={() => setDetailNode(null)}
        footer={[
          <Button key="close" type="primary" onClick={() => setDetailNode(null)}>
            关闭
          </Button>,
        ]}
      >
        {detailNode ? (
          <Descriptions size="small" column={1}>
            <Descriptions.Item label="名称">{detailNode.name}</Descriptions.Item>
            <Descriptions.Item label="状态">
              {renderStatus(detailNode.status)}
            </Descriptions.Item>
            <Descriptions.Item label="地址">
              {detailNode.host}:{detailNode.port}
            </Descriptions.Item>
            <Descriptions.Item label="区域">
              {detailNode.region ?? '-'}
            </Descriptions.Item>
            <Descriptions.Item label="自托管">
              {detailNode.is_self_hosted ? '是' : '否'}
            </Descriptions.Item>
            <Descriptions.Item label="公开节点">
              {detailNode.is_public ? '是' : '否'}
            </Descriptions.Item>
            <Descriptions.Item label="流量倍率">
              {`${(detailNode.traffic_multiplier ?? 1).toFixed(2)}x`}
            </Descriptions.Item>
            <Descriptions.Item label="最后心跳">
              {formatDateTime(detailNode.last_heartbeat_at)}
            </Descriptions.Item>
            <Descriptions.Item label="描述">
              {detailNode.description ?? '-'}
            </Descriptions.Item>
          </Descriptions>
        ) : null}
      </Modal>
    </PageContainer>
  )
}

export default NodeList
