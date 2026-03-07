import {
  DeleteOutlined,
  EditOutlined,
  PlusOutlined,
  RetweetOutlined,
} from '@ant-design/icons'
import {
  Button,
  Form,
  Input,
  Modal,
  Popconfirm,
  Select,
  Space,
  Switch,
  Table,
  Tag,
  Typography,
  message,
} from 'antd'
import { useEffect, useMemo, useState } from 'react'

import { nodeApi, nodePairApi } from '../../services/api'
import { useAppStore } from '../../store/app'
import type {
  CreateNodePairPayload,
  NodePairRecord,
  NodeRecord,
  UpdateNodePairPayload,
} from '../../types'
import { getErrorMessage } from '../../utils/error'

type NodePairFormValues = {
  name?: string
  entry_node_id: number
  exit_node_id: number
  description?: string
}

type NodePairFormMode = 'create' | 'edit'

const statusMap: Record<
  string,
  { label: string; color: 'green' | 'red' | 'orange' | 'default' }
> = {
  online: { label: '在线', color: 'green' },
  offline: { label: '离线', color: 'red' },
  maintain: { label: '维护', color: 'orange' },
}

const normalizeText = (value?: string): string | undefined => {
  if (!value) {
    return undefined
  }
  const trimmed = value.trim()
  return trimmed === '' ? undefined : trimmed
}

const renderNodeStatus = (status?: string) => {
  const mapped = status ? statusMap[status] : undefined
  if (!mapped) {
    return <Tag>{status ?? '未知'}</Tag>
  }
  return <Tag color={mapped.color}>{mapped.label}</Tag>
}

const NodePairList = () => {
  const [form] = Form.useForm<NodePairFormValues>()
  const selectedEntryNodeID = Form.useWatch('entry_node_id', form)

  const [loading, setLoading] = useState<boolean>(false)
  const [pairs, setPairs] = useState<NodePairRecord[]>([])
  const [nodes, setNodes] = useState<NodeRecord[]>([])
  const nodeStatusMap = useAppStore((state) => state.nodeStatusMap)

  const [formOpen, setFormOpen] = useState<boolean>(false)
  const [formMode, setFormMode] = useState<NodePairFormMode>('create')
  const [editingPair, setEditingPair] = useState<NodePairRecord | null>(null)
  const [saving, setSaving] = useState<boolean>(false)
  const [togglingID, setTogglingID] = useState<number | null>(null)

  const loadData = async (): Promise<void> => {
    setLoading(true)
    try {
      const [pairResult, nodeResult] = await Promise.all([
        nodePairApi.list(),
        nodeApi.list({
          page: 1,
          pageSize: 200,
        }),
      ])
      setPairs(pairResult.list ?? [])
      setNodes(nodeResult.list ?? [])
    } catch (error) {
      message.error(getErrorMessage(error, '节点配对数据加载失败'))
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    void loadData()
  }, [])

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

    setPairs((current) => {
      let changed = false
      const next = current.map((pair) => {
        const entryStatus = pair.entry_node ? nodeStatusMap[pair.entry_node.id] : undefined
        const exitStatus = pair.exit_node ? nodeStatusMap[pair.exit_node.id] : undefined

        if (!entryStatus && !exitStatus) {
          return pair
        }

        const nextPair: NodePairRecord = {
          ...pair,
          entry_node:
            entryStatus && pair.entry_node
              ? {
                  ...pair.entry_node,
                  status: entryStatus,
                }
              : pair.entry_node,
          exit_node:
            exitStatus && pair.exit_node
              ? {
                  ...pair.exit_node,
                  status: exitStatus,
                }
              : pair.exit_node,
        }

        if (
          nextPair.entry_node?.status !== pair.entry_node?.status ||
          nextPair.exit_node?.status !== pair.exit_node?.status
        ) {
          changed = true
          return nextPair
        }
        return pair
      })
      return changed ? next : current
    })
  }, [nodeStatusMap])

  const nodeOptions = useMemo(
    () =>
      nodes.map((node) => ({
        label: `${node.name} (${statusMap[node.status]?.label ?? node.status})`,
        value: node.id,
      })),
    [nodes],
  )

  const exitNodeOptions = useMemo(
    () =>
      nodeOptions.filter((option) => {
        if (!selectedEntryNodeID) {
          return true
        }
        return option.value !== selectedEntryNodeID
      }),
    [nodeOptions, selectedEntryNodeID],
  )

  const openCreateModal = (): void => {
    setFormMode('create')
    setEditingPair(null)
    form.resetFields()
    setFormOpen(true)
  }

  const openEditModal = (pair: NodePairRecord): void => {
    setFormMode('edit')
    setEditingPair(pair)
    form.setFieldsValue({
      name: pair.name ?? undefined,
      entry_node_id: pair.entry_node_id,
      exit_node_id: pair.exit_node_id,
      description: pair.description ?? undefined,
    })
    setFormOpen(true)
  }

  const closeModal = (): void => {
    setFormOpen(false)
    setEditingPair(null)
    form.resetFields()
  }

  const handleSubmit = async (values: NodePairFormValues): Promise<void> => {
    setSaving(true)
    try {
      const payload: CreateNodePairPayload = {
        entry_node_id: values.entry_node_id,
        exit_node_id: values.exit_node_id,
        name: normalizeText(values.name),
        description: normalizeText(values.description),
      }

      if (formMode === 'create') {
        await nodePairApi.create(payload)
        message.success('节点配对创建成功')
      } else {
        if (!editingPair) {
          return
        }
        const updatePayload: UpdateNodePairPayload = {
          ...payload,
        }
        await nodePairApi.update(editingPair.id, updatePayload)
        message.success('节点配对更新成功')
      }

      closeModal()
      await loadData()
    } catch (error) {
      message.error(getErrorMessage(error, '保存节点配对失败'))
    } finally {
      setSaving(false)
    }
  }

  const handleDelete = async (pairID: number): Promise<void> => {
    try {
      await nodePairApi.remove(pairID)
      message.success('节点配对已删除')
      await loadData()
    } catch (error) {
      message.error(getErrorMessage(error, '删除节点配对失败'))
    }
  }

  const handleToggle = async (pair: NodePairRecord): Promise<void> => {
    setTogglingID(pair.id)
    try {
      await nodePairApi.toggle(pair.id)
      message.success(pair.is_enabled ? '节点配对已禁用' : '节点配对已启用')
      await loadData()
    } catch (error) {
      message.error(getErrorMessage(error, '切换节点配对状态失败'))
    } finally {
      setTogglingID(null)
    }
  }

  return (
    <>
      <div className="mb-4 flex items-center justify-between">
        <Typography.Text type="secondary">
          维护入口节点与出口节点绑定关系，用于隧道转发模式。
        </Typography.Text>
        <Button
          type="primary"
          icon={<PlusOutlined />}
          onClick={openCreateModal}
        >
          创建配对
        </Button>
      </div>

      <Table<NodePairRecord>
        rowKey="id"
        loading={loading}
        dataSource={pairs}
        pagination={{
          pageSize: 10,
          showSizeChanger: false,
        }}
        columns={[
          {
            title: '配对名称',
            dataIndex: 'name',
            render: (_, record) =>
              record.name ??
              `#${record.id} ${record.entry_node?.name ?? record.entry_node_id} → ${
                record.exit_node?.name ?? record.exit_node_id
              }`,
          },
          {
            title: '入口节点',
            render: (_, record) => (
              <Space>
                <span>{record.entry_node?.name ?? `ID:${record.entry_node_id}`}</span>
                {renderNodeStatus(record.entry_node?.status)}
              </Space>
            ),
          },
          {
            title: '出口节点',
            render: (_, record) => (
              <Space>
                <span>{record.exit_node?.name ?? `ID:${record.exit_node_id}`}</span>
                {renderNodeStatus(record.exit_node?.status)}
              </Space>
            ),
          },
          {
            title: '是否启用',
            dataIndex: 'is_enabled',
            width: 120,
            render: (_enabled, record) => (
              <Switch
                checked={record.is_enabled}
                loading={togglingID === record.id}
                onChange={() => void handleToggle(record)}
              />
            ),
          },
          {
            title: '操作',
            width: 240,
            render: (_, record) => (
              <Space>
                <Button
                  type="link"
                  icon={<EditOutlined />}
                  onClick={() => openEditModal(record)}
                >
                  编辑
                </Button>
                <Button
                  type="link"
                  icon={<RetweetOutlined />}
                  loading={togglingID === record.id}
                  onClick={() => void handleToggle(record)}
                >
                  {record.is_enabled ? '禁用' : '启用'}
                </Button>
                <Popconfirm
                  title="确定删除该节点配对吗？"
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

      <Modal
        title={formMode === 'create' ? '创建节点配对' : '编辑节点配对'}
        open={formOpen}
        onCancel={closeModal}
        onOk={() => void form.submit()}
        okText={formMode === 'create' ? '创建' : '保存'}
        confirmLoading={saving}
        destroyOnClose
      >
        <Form<NodePairFormValues>
          form={form}
          layout="vertical"
          onFinish={(values) => void handleSubmit(values)}
          preserve={false}
        >
          <Form.Item
            label="配对名称"
            name="name"
            rules={[{ max: 100, message: '名称长度不能超过 100 个字符' }]}
          >
            <Input placeholder="例如：美国入口 → 日本出口" />
          </Form.Item>

          <Form.Item
            label="入口节点"
            name="entry_node_id"
            rules={[{ required: true, message: '请选择入口节点' }]}
          >
            <Select placeholder="请选择入口节点" options={nodeOptions} />
          </Form.Item>

          <Form.Item
            label="出口节点"
            name="exit_node_id"
            dependencies={['entry_node_id']}
            rules={[
              { required: true, message: '请选择出口节点' },
              ({ getFieldValue }) => ({
                validator: async (_, value) => {
                  if (!value || value !== getFieldValue('entry_node_id')) {
                    return
                  }
                  throw new Error('出口节点不能与入口节点相同')
                },
              }),
            ]}
          >
            <Select placeholder="请选择出口节点" options={exitNodeOptions} />
          </Form.Item>

          <Form.Item label="描述" name="description">
            <Input.TextArea rows={3} placeholder="可选：记录配对用途" />
          </Form.Item>
        </Form>
      </Modal>
    </>
  )
}

export default NodePairList
