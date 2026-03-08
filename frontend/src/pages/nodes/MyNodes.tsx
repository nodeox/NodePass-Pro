import {
  DeleteOutlined,
  MoreOutlined,
  PlusOutlined,
  PoweroffOutlined,
  ReloadOutlined,
  RetweetOutlined,
  UploadOutlined,
} from '@ant-design/icons'
import {
  Alert,
  Button,
  Dropdown,
  Form,
  Input,
  InputNumber,
  Modal,
  Select,
  Space,
  Switch,
  Table,
  Tabs,
  Tag,
  Typography,
  message,
} from 'antd'
import type { TableProps } from 'antd'
import { useCallback, useEffect, useMemo, useState } from 'react'

import PageContainer from '../../components/common/PageContainer'
import { usePageTitle } from '../../hooks/usePageTitle'
import { nodeGroupApi, nodeInstanceApi } from '../../services/nodeGroupApi'
import type {
  DeployCommandResponse,
  NodeGroup,
  NodeGroupConfig,
  NodeInstance,
} from '../../types/nodeGroup'
import { getErrorMessage } from '../../utils/error'
import { formatDateTime } from '../../utils/format'

type NodeRow = NodeInstance & {
  group_id: number
  group_name: string
  group_type: NodeGroup['type']
}

type GroupFormValues = {
  name: string
  type: NodeGroup['type']
  description?: string
  is_enabled: boolean
  allowed_protocols: string[]
  port_start: number
  port_end: number
}

type AddNodeFormValues = {
  name: string
  host: string
  port: number
}

type DeployFormValues = {
  group_id: number
  service_name: string
  debug_mode: boolean
}

type DeployNodeStatus =
  | 'pending'
  | 'online'
  | 'offline'
  | 'maintain'
  | 'not_found'
  | 'deleted_manual'
  | 'deleted_timeout'

const DEPLOY_TIMEOUT_MS = 10 * 60 * 1000

const buildServiceName = (groupName: string): string => {
  const safeGroupName = groupName
    .toLowerCase()
    .replace(/[^a-z0-9-]+/g, '-')
    .replace(/^-+|-+$/g, '')
    .slice(0, 24) || 'group'
  const suffix = Math.random().toString(36).slice(2, 6)
  return `nodepass-${safeGroupName}-${suffix}`
}

const getDefaultConfig = (type: NodeGroup['type']): NodeGroupConfig => {
  if (type === 'entry') {
    return {
      allowed_protocols: ['tcp', 'udp'],
      port_range: { start: 10000, end: 20000 },
      entry_config: {
        require_exit_group: false,
        traffic_multiplier: 1,
        dns_load_balance: false,
      },
    }
  }

  return {
    allowed_protocols: ['tcp', 'udp'],
    port_range: { start: 10000, end: 20000 },
    exit_config: {
      load_balance_strategy: 'round_robin',
      health_check_interval: 30,
      health_check_timeout: 5,
    },
  }
}

const toEndpoint = (record: NodeInstance): string => {
  if (!record.host || !record.port) {
    return '未上报'
  }
  return `${record.host}:${record.port}`
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

const MyNodes = () => {
  usePageTitle('我的节点')

  const [groupForm] = Form.useForm<GroupFormValues>()
  const [addNodeForm] = Form.useForm<AddNodeFormValues>()
  const [deployForm] = Form.useForm<DeployFormValues>()

  const [loading, setLoading] = useState(false)
  const [actionLoading, setActionLoading] = useState<string | null>(null)

  const [groups, setGroups] = useState<NodeGroup[]>([])
  const [nodes, setNodes] = useState<NodeRow[]>([])

  const [groupModalOpen, setGroupModalOpen] = useState(false)
  const [groupSubmitting, setGroupSubmitting] = useState(false)
  const [editingGroup, setEditingGroup] = useState<NodeGroup | null>(null)

  const [addNodeModalOpen, setAddNodeModalOpen] = useState(false)
  const [addNodeSubmitting, setAddNodeSubmitting] = useState(false)
  const [selectedGroup, setSelectedGroup] = useState<NodeGroup | null>(null)

  const [deployModalOpen, setDeployModalOpen] = useState(false)
  const [deploying, setDeploying] = useState(false)
  const [deployResult, setDeployResult] = useState<DeployCommandResponse | null>(null)
  const [deployWatchGroupID, setDeployWatchGroupID] = useState<number | null>(null)
  const [deployNodeStatus, setDeployNodeStatus] = useState<DeployNodeStatus>('pending')
  const [deployStatusLoading, setDeployStatusLoading] = useState<boolean>(false)
  const [deployLastHeartbeat, setDeployLastHeartbeat] = useState<string | null>(null)
  const [deployInstanceID, setDeployInstanceID] = useState<number | null>(null)
  const [deployWatchStartedAt, setDeployWatchStartedAt] = useState<number | null>(null)
  const [onlineNotified, setOnlineNotified] = useState<boolean>(false)

  const loadData = useCallback(async (silent = false) => {
    if (!silent) {
      setLoading(true)
    }
    try {
      const groupResult = await nodeGroupApi.list({ page: 1, page_size: 500 })
      const ownGroups = groupResult.items ?? []
      setGroups(ownGroups)

      const nodeChunks = await Promise.all(
        ownGroups.map(async (group) => {
          const list = await nodeGroupApi.listNodes(group.id)
          return list.map<NodeRow>((item) => ({
            ...item,
            group_id: group.id,
            group_name: group.name,
            group_type: group.type,
          }))
        }),
      )
      setNodes(nodeChunks.flat())
    } catch (error) {
      message.error(getErrorMessage(error, '自托管节点加载失败'))
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

  const openCreateGroupModal = () => {
    setEditingGroup(null)
    groupForm.resetFields()
    groupForm.setFieldsValue({
      name: '',
      type: 'entry',
      description: '',
      is_enabled: true,
      allowed_protocols: ['tcp', 'udp'],
      port_start: 10000,
      port_end: 20000,
    })
    setGroupModalOpen(true)
  }

  const openEditGroupModal = (group: NodeGroup) => {
    setEditingGroup(group)
    const cfg = group.config ?? getDefaultConfig(group.type)
    groupForm.setFieldsValue({
      name: group.name,
      type: group.type,
      description: group.description || '',
      is_enabled: group.is_enabled,
      allowed_protocols: cfg.allowed_protocols?.length ? cfg.allowed_protocols : ['tcp', 'udp'],
      port_start: cfg.port_range?.start || 10000,
      port_end: cfg.port_range?.end || 20000,
    })
    setGroupModalOpen(true)
  }

  const closeGroupModal = () => {
    setGroupModalOpen(false)
    setEditingGroup(null)
    groupForm.resetFields()
  }

  const submitGroup = async (values: GroupFormValues) => {
    const name = values.name?.trim()
    if (!name) {
      message.error('节点组名称不能为空')
      return
    }

    // 编辑时保留原有配置，只更新用户修改的字段
    // 创建时使用默认配置
    const baseConfig = editingGroup?.config ?? getDefaultConfig(values.type)
    const config: NodeGroupConfig = {
      ...baseConfig,
      allowed_protocols: values.allowed_protocols,
      port_range: {
        start: values.port_start,
        end: values.port_end,
      },
    }

    const payload = {
      name,
      description: values.description?.trim() || undefined,
      config,
    }

    setGroupSubmitting(true)
    try {
      if (editingGroup) {
        await nodeGroupApi.update(editingGroup.id, {
          ...payload,
          is_enabled: values.is_enabled,
        })
        message.success('节点组更新成功')
      } else {
        await nodeGroupApi.create({
          ...payload,
          type: values.type,
        })
        message.success('节点组创建成功')
      }
      closeGroupModal()
      await loadData(true)
    } catch (error) {
      message.error(getErrorMessage(error, editingGroup ? '更新节点组失败' : '创建节点组失败'))
    } finally {
      setGroupSubmitting(false)
    }
  }

  const toggleGroup = (group: NodeGroup) => {
    void runAction(
      `group-toggle-${group.id}`,
      async () => {
        await nodeGroupApi.toggle(group.id)
      },
      group.is_enabled ? '节点组已禁用' : '节点组已启用',
    )
  }

  const deleteGroup = (group: NodeGroup) => {
    Modal.confirm({
      title: '删除节点组',
      content: `确认删除节点组「${group.name}」吗？此操作不可恢复。`,
      okText: '删除',
      okType: 'danger',
      cancelText: '取消',
      onOk: async () => {
        await runAction(
          `group-delete-${group.id}`,
          async () => {
            await nodeGroupApi.delete(group.id)
          },
          '节点组已删除',
        )
      },
    })
  }

  const openAddNodeModal = (group: NodeGroup) => {
    setSelectedGroup(group)
    addNodeForm.resetFields()
    addNodeForm.setFieldsValue({
      name: `${group.name}-node`,
      host: '',
      port: 0,
    })
    setAddNodeModalOpen(true)
  }

  const closeAddNodeModal = () => {
    setAddNodeModalOpen(false)
    setSelectedGroup(null)
    addNodeForm.resetFields()
  }

  const submitAddNode = async (values: AddNodeFormValues) => {
    if (!selectedGroup) {
      return
    }
    setAddNodeSubmitting(true)
    try {
      await nodeGroupApi.addNode(selectedGroup.id, {
        name: values.name.trim(),
        host: values.host.trim(),
        port: values.port,
      })
      message.success('节点实例创建成功')
      closeAddNodeModal()
      await loadData(true)
    } catch (error) {
      message.error(getErrorMessage(error, '新增节点失败'))
    } finally {
      setAddNodeSubmitting(false)
    }
  }

  const openDeployModal = (group?: NodeGroup) => {
    const target = group ?? groups[0]
    if (!target) {
      message.warning('请先创建节点组')
      return
    }

    deployForm.resetFields()
    deployForm.setFieldsValue({
      group_id: target.id,
      service_name: buildServiceName(target.name),
      debug_mode: false,
    })
    setDeployResult(null)
    setDeployWatchGroupID(target.id)
    setDeployNodeStatus('pending')
    setDeployLastHeartbeat(null)
    setDeployInstanceID(null)
    setDeployWatchStartedAt(null)
    setOnlineNotified(false)
    setDeployModalOpen(true)
  }

  const onDeployGroupChange = (groupID: number) => {
    const target = groups.find((item) => item.id === groupID)
    if (!target) {
      return
    }
    deployForm.setFieldValue('service_name', buildServiceName(target.name))
  }

  const submitDeploy = async (values: DeployFormValues) => {
    setDeploying(true)
    try {
      const result = await nodeGroupApi.generateDeployCommand(values.group_id, {
        service_name: values.service_name.trim(),
        debug_mode: values.debug_mode,
      })
      setDeployResult(result)
      setDeployWatchGroupID(values.group_id)
      setDeployNodeStatus('pending')
      setDeployLastHeartbeat(null)
      setDeployInstanceID(null)
      setDeployWatchStartedAt(Date.now())
      setOnlineNotified(false)
      message.success('部署命令已生成')
      await loadData(true)
    } catch (error) {
      message.error(getErrorMessage(error, '生成部署命令失败'))
    } finally {
      setDeploying(false)
    }
  }

  const refreshDeployStatus = useCallback(
    async (silent = false): Promise<DeployNodeStatus | null> => {
      if (!deployResult || !deployWatchGroupID) {
        return null
      }

      if (!silent) {
        setDeployStatusLoading(true)
      }

      try {
        const list = await nodeGroupApi.listNodes(deployWatchGroupID)
        const target = list.find((item) => item.node_id === deployResult.node_id)

        setNodes((prev) => {
          const keep = prev.filter((item) => item.group_id !== deployWatchGroupID)
          const next = list.map<NodeRow>((item) => {
            const group = groups.find((g) => g.id === deployWatchGroupID)
            return {
              ...item,
              group_id: deployWatchGroupID,
              group_name: group?.name ?? `#${deployWatchGroupID}`,
              group_type: group?.type ?? 'entry',
            }
          })
          return [...keep, ...next]
        })

        if (!target) {
          setDeployNodeStatus('not_found')
          setDeployLastHeartbeat(null)
          setDeployInstanceID(null)
          return 'not_found'
        }

        setDeployInstanceID(target.id)
        setDeployNodeStatus(target.status ?? 'offline')
        setDeployLastHeartbeat(target.last_heartbeat_at ?? null)
        return target.status ?? 'offline'
      } catch (error) {
        if (!silent) {
          message.error(getErrorMessage(error, '检测节点状态失败'))
        }
        return null
      } finally {
        if (!silent) {
          setDeployStatusLoading(false)
        }
      }
    },
    [deployResult, deployWatchGroupID, groups],
  )

  const deleteDeployNode = useCallback(
    async (reason: 'manual' | 'timeout') => {
      if (!deployResult || !deployWatchGroupID) {
        return
      }

      let targetID = deployInstanceID
      if (!targetID) {
        const list = await nodeGroupApi.listNodes(deployWatchGroupID)
        const target = list.find((item) => item.node_id === deployResult.node_id)
        if (!target) {
          setDeployNodeStatus(reason === 'timeout' ? 'deleted_timeout' : 'deleted_manual')
          setDeployInstanceID(null)
          setDeployLastHeartbeat(null)
          return
        }
        targetID = target.id
      }

      await nodeInstanceApi.delete(targetID)
      setDeployNodeStatus(reason === 'timeout' ? 'deleted_timeout' : 'deleted_manual')
      setDeployInstanceID(null)
      setDeployLastHeartbeat(null)
      if (reason === 'timeout') {
        message.warning('超过 10 分钟未上线，节点已自动删除')
      } else {
        message.success('节点已手动删除')
      }
      await loadData(true)
    },
    [deployResult, deployWatchGroupID, deployInstanceID, loadData],
  )

  useEffect(() => {
    if (!deployModalOpen || !deployResult || !deployWatchGroupID) {
      return
    }

    void refreshDeployStatus()

    if (
      deployNodeStatus === 'online' ||
      deployNodeStatus === 'deleted_manual' ||
      deployNodeStatus === 'deleted_timeout'
    ) {
      return
    }

    const timer = window.setInterval(() => {
      void (async () => {
        const latestStatus = await refreshDeployStatus(true)
        if (
          latestStatus &&
          latestStatus !== 'online' &&
          deployWatchStartedAt &&
          Date.now() - deployWatchStartedAt >= DEPLOY_TIMEOUT_MS
        ) {
          try {
            await deleteDeployNode('timeout')
          } catch (error) {
            message.error(getErrorMessage(error, '自动删除超时节点失败'))
          }
        }
      })()
    }, 5000)

    return () => {
      window.clearInterval(timer)
    }
  }, [
    deployModalOpen,
    deployResult,
    deployWatchGroupID,
    deployNodeStatus,
    deployWatchStartedAt,
    refreshDeployStatus,
    deleteDeployNode,
  ])

  useEffect(() => {
    if (deployNodeStatus === 'online' && !onlineNotified) {
      message.success('节点已上线')
      setOnlineNotified(true)
    }
  }, [deployNodeStatus, onlineNotified])

  const deployStatusTag = useMemo(() => {
    if (deployNodeStatus === 'online') {
      return <Tag color="green">已上线</Tag>
    }
    if (deployNodeStatus === 'maintain') {
      return <Tag color="orange">维护中</Tag>
    }
    if (deployNodeStatus === 'offline') {
      return <Tag color="red">离线</Tag>
    }
    if (deployNodeStatus === 'deleted_manual') {
      return <Tag color="default">已手动删除</Tag>
    }
    if (deployNodeStatus === 'deleted_timeout') {
      return <Tag color="default">超时已删除</Tag>
    }
    if (deployNodeStatus === 'not_found') {
      return <Tag color="default">等待注册</Tag>
    }
    return <Tag color="processing">检测中</Tag>
  }, [deployNodeStatus])

  const restartNode = (record: NodeRow) => {
    void runAction(
      `node-restart-${record.id}`,
      async () => {
        await nodeInstanceApi.restart(record.id)
      },
      `节点 ${record.name} 已重启`,
    )
  }

  const toggleNode = (record: NodeRow) => {
    void runAction(
      `node-toggle-${record.id}`,
      async () => {
        await nodeInstanceApi.update(record.id, { is_enabled: !record.is_enabled })
      },
      record.is_enabled ? '节点已禁用' : '节点已启用',
    )
  }

  const deleteNode = (record: NodeRow) => {
    Modal.confirm({
      title: '删除节点实例',
      content: `确认删除节点「${record.name}」吗？`,
      okText: '删除',
      okType: 'danger',
      cancelText: '取消',
      onOk: async () => {
        await runAction(
          `node-delete-${record.id}`,
          async () => {
            await nodeInstanceApi.delete(record.id)
          },
          '节点实例已删除',
        )
      },
    })
  }

  const groupColumns: TableProps<NodeGroup>['columns'] = [
      {
        title: '名称',
        dataIndex: 'name',
        width: 180,
        render: (value: string) => (
          <Typography.Text ellipsis style={{ maxWidth: 160 }}>
            {value}
          </Typography.Text>
        ),
      },
      {
        title: '类型',
        dataIndex: 'type',
        width: 90,
        render: (value: NodeGroup['type']) =>
          value === 'entry' ? <Tag color="blue">入口组</Tag> : <Tag color="green">出口组</Tag>,
      },
      {
        title: '状态',
        dataIndex: 'is_enabled',
        width: 90,
        render: (enabled: boolean) =>
          enabled ? <Tag color="green">已启用</Tag> : <Tag color="default">已禁用</Tag>,
      },
      {
        title: '节点数',
        key: 'node_count',
        width: 110,
        render: (_: unknown, record: NodeGroup) => {
          const online = record.stats?.online_nodes ?? 0
          const total = record.stats?.total_nodes ?? 0
          return `${online} / ${total}`
        },
      },
      {
        title: '更新时间',
        dataIndex: 'updated_at',
        width: 168,
        render: (value: string) => formatDateTime(value),
      },
      {
        title: '操作',
        key: 'actions',
        fixed: 'right',
        width: 220,
        align: 'center',
        render: (_: unknown, record: NodeGroup) => (
          <Space size={6} style={{ whiteSpace: 'nowrap' }}>
            <Button
              type="primary"
              size="small"
              icon={<UploadOutlined />}
              onClick={() => openDeployModal(record)}
            >
              部署
            </Button>
            <Dropdown
              trigger={['click']}
              menu={{
                items: [
                  {
                    key: 'add-node',
                    icon: <PlusOutlined />,
                    label: '新增节点',
                  },
                  {
                    key: 'edit',
                    label: '编辑',
                  },
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
                  if (key === 'add-node') {
                    openAddNodeModal(record)
                    return
                  }
                  if (key === 'edit') {
                    openEditGroupModal(record)
                    return
                  }
                  if (key === 'toggle') {
                    toggleGroup(record)
                    return
                  }
                  if (key === 'delete') {
                    deleteGroup(record)
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
    ]

  const nodeColumns: TableProps<NodeRow>['columns'] = [
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
        title: 'Node ID',
        dataIndex: 'node_id',
        width: 210,
        render: (value: string) => (
          <Typography.Text copyable={{ text: value }} ellipsis style={{ maxWidth: 190 }}>
            {value}
          </Typography.Text>
        ),
      },
      {
        title: '所属组',
        key: 'group',
        width: 170,
        render: (_: unknown, record: NodeRow) => (
          <Space size={6}>
            <Typography.Text ellipsis style={{ maxWidth: 90 }}>
              {record.group_name}
            </Typography.Text>
            <Tag color={record.group_type === 'entry' ? 'blue' : 'green'}>
              {record.group_type === 'entry' ? '入口' : '出口'}
            </Tag>
          </Space>
        ),
      },
      {
        title: '连接地址',
        key: 'endpoint',
        width: 170,
        render: (_: unknown, record: NodeRow) => toEndpoint(record),
      },
      {
        title: '状态',
        dataIndex: 'status',
        width: 90,
        render: (status: NodeInstance['status']) => renderStatusTag(status),
      },
      {
        title: '启用',
        dataIndex: 'is_enabled',
        width: 80,
        render: (enabled: boolean) =>
          enabled ? <Tag color="green">是</Tag> : <Tag color="default">否</Tag>,
      },
      {
        title: '最后心跳',
        dataIndex: 'last_heartbeat_at',
        width: 168,
        render: (value: string | null) => (value ? formatDateTime(value) : '-'),
      },
      {
        title: '操作',
        key: 'actions',
        fixed: 'right',
        width: 220,
        align: 'center',
        render: (_: unknown, record: NodeRow) => (
          <Space size={6} style={{ whiteSpace: 'nowrap' }}>
            <Button
              size="small"
              icon={<RetweetOutlined />}
              loading={actionLoading === `node-restart-${record.id}`}
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
        ),
      },
    ]

  const groupOptions = useMemo(
    () =>
      groups.map((group) => ({
        value: group.id,
        label: `${group.name} (${group.type === 'entry' ? '入口组' : '出口组'})`,
      })),
    [groups],
  )

  return (
    <PageContainer
      title="我的节点"
      description="新建并管理你的自托管节点组、节点实例与部署命令。"
      extra={
        <Space>
          <Button icon={<ReloadOutlined />} onClick={() => void loadData()} loading={loading}>
            刷新
          </Button>
          <Button icon={<PlusOutlined />} onClick={openCreateGroupModal}>
            新建节点组
          </Button>
          <Button
            type="primary"
            icon={<UploadOutlined />}
            onClick={() => openDeployModal()}
            disabled={groups.length === 0}
          >
            部署新节点
          </Button>
        </Space>
      }
    >
      {groups.length === 0 ? (
        <Alert
          type="warning"
          showIcon
          style={{ marginBottom: 12 }}
          message="当前还没有自托管节点组，请先创建节点组。"
        />
      ) : null}

      <Tabs
        items={[
          {
            key: 'groups',
            label: `节点组 (${groups.length})`,
            children: (
              <Table<NodeGroup>
                rowKey="id"
                size="small"
                loading={loading}
                dataSource={groups}
                columns={groupColumns}
                scroll={{ x: 980 }}
                pagination={{
                  pageSize: 10,
                  showSizeChanger: true,
                  showTotal: (total) => `共 ${total} 条`,
                }}
              />
            ),
          },
          {
            key: 'nodes',
            label: `节点实例 (${nodes.length})`,
            children: (
              <Table<NodeRow>
                rowKey="id"
                size="small"
                loading={loading}
                dataSource={nodes}
                columns={nodeColumns}
                scroll={{ x: 1180 }}
                pagination={{
                  pageSize: 20,
                  showSizeChanger: true,
                  showTotal: (total) => `共 ${total} 条`,
                }}
              />
            ),
          },
        ]}
      />

      <Modal
        title={editingGroup ? '编辑节点组' : '新建节点组'}
        open={groupModalOpen}
        onCancel={closeGroupModal}
        onOk={() => void groupForm.submit()}
        okText={editingGroup ? '保存' : '创建'}
        confirmLoading={groupSubmitting}
        destroyOnClose
      >
        <Form<GroupFormValues>
          form={groupForm}
          layout="vertical"
          preserve={false}
          onFinish={(values) => void submitGroup(values)}
        >
          <Form.Item
            label="节点组名称"
            name="name"
            rules={[
              { required: true, message: '请输入节点组名称' },
              { max: 100, message: '节点组名称不能超过 100 个字符' },
            ]}
          >
            <Input maxLength={100} />
          </Form.Item>

          <Form.Item label="类型" name="type" rules={[{ required: true, message: '请选择类型' }]}>
            <Select
              disabled={Boolean(editingGroup)}
              options={[
                { label: '入口组', value: 'entry' },
                { label: '出口组', value: 'exit' },
              ]}
            />
          </Form.Item>

          <Form.Item label="描述" name="description">
            <Input.TextArea rows={2} maxLength={300} />
          </Form.Item>

          <Form.Item
            label="允许协议"
            name="allowed_protocols"
            rules={[{ required: true, message: '请选择至少一个协议' }]}
          >
            <Select
              mode="multiple"
              options={[
                { label: 'TCP', value: 'tcp' },
                { label: 'UDP', value: 'udp' },
                { label: 'WebSocket', value: 'ws' },
                { label: 'TLS', value: 'tls' },
                { label: 'QUIC', value: 'quic' },
              ]}
            />
          </Form.Item>

          <Space style={{ width: '100%' }}>
            <Form.Item
              label="端口起始"
              name="port_start"
              rules={[{ required: true, message: '请输入起始端口' }]}
            >
              <InputNumber min={1} max={65535} precision={0} />
            </Form.Item>
            <Form.Item
              label="端口结束"
              name="port_end"
              dependencies={['port_start']}
              rules={[
                { required: true, message: '请输入结束端口' },
                ({ getFieldValue }) => ({
                  validator(_, value) {
                    const start = Number(getFieldValue('port_start') || 0)
                    if (Number(value) > start) {
                      return Promise.resolve()
                    }
                    return Promise.reject(new Error('结束端口必须大于起始端口'))
                  },
                }),
              ]}
            >
              <InputNumber min={1} max={65535} precision={0} />
            </Form.Item>
          </Space>

          {editingGroup ? (
            <Form.Item label="启用" name="is_enabled" valuePropName="checked">
              <Switch />
            </Form.Item>
          ) : null}
        </Form>
      </Modal>

      <Modal
        title={selectedGroup ? `新增节点 · ${selectedGroup.name}` : '新增节点'}
        open={addNodeModalOpen}
        onCancel={closeAddNodeModal}
        onOk={() => void addNodeForm.submit()}
        okText="创建"
        confirmLoading={addNodeSubmitting}
        destroyOnClose
      >
        <Form<AddNodeFormValues>
          form={addNodeForm}
          layout="vertical"
          preserve={false}
          onFinish={(values) => void submitAddNode(values)}
        >
          <Form.Item
            label="节点名称"
            name="name"
            rules={[
              { required: true, message: '请输入节点名称' },
              { max: 100, message: '节点名称不能超过 100 个字符' },
            ]}
          >
            <Input maxLength={100} />
          </Form.Item>

          <Form.Item label="主机" name="host" rules={[{ required: true, message: '请输入主机或域名' }]}>
            <Input placeholder="例如 1.2.3.4 或 example.com" />
          </Form.Item>

          <Form.Item
            label="端口"
            name="port"
            rules={[{ required: true, message: '请输入端口' }]}
          >
            <InputNumber min={1} max={65535} precision={0} style={{ width: '100%' }} />
          </Form.Item>
        </Form>
      </Modal>

      <Modal
        title="部署新节点"
        open={deployModalOpen}
        onCancel={() => {
          setDeployModalOpen(false)
        }}
        onOk={() => void deployForm.submit()}
        okText="生成部署命令"
        confirmLoading={deploying}
        destroyOnClose
      >
        <Form<DeployFormValues>
          form={deployForm}
          layout="vertical"
          preserve={false}
          onFinish={(values) => void submitDeploy(values)}
        >
          <Form.Item
            label="节点组"
            name="group_id"
            rules={[{ required: true, message: '请选择节点组' }]}
          >
            <Select options={groupOptions} onChange={onDeployGroupChange} />
          </Form.Item>

          <Form.Item
            label="服务名称"
            name="service_name"
            rules={[
              { required: true, message: '请输入服务名称' },
              { max: 60, message: '服务名称长度不能超过 60 个字符' },
            ]}
          >
            <Input placeholder="例如：nodepass-entry-ab12" />
          </Form.Item>

          <Form.Item label="调试模式" name="debug_mode" valuePropName="checked">
            <Switch />
          </Form.Item>
        </Form>

        {deployResult ? (
          <Space direction="vertical" size={8} style={{ width: '100%', marginTop: 8 }}>
            <Alert type="success" showIcon message="部署命令已生成，可直接复制执行。" />
            <Typography.Text copyable={{ text: deployResult.node_id }}>
              Node ID: {deployResult.node_id}
            </Typography.Text>
            <Space>
              <Typography.Text strong>部署状态：</Typography.Text>
              {deployStatusTag}
              <Button size="small" loading={deployStatusLoading} onClick={() => void refreshDeployStatus()}>
                刷新状态
              </Button>
              <Button
                danger
                size="small"
                onClick={() => {
                  Modal.confirm({
                    title: '手动删除节点',
                    content: '确认删除当前部署节点吗？删除后需重新生成部署命令。',
                    okText: '删除',
                    okType: 'danger',
                    cancelText: '取消',
                    onOk: async () => {
                      try {
                        await deleteDeployNode('manual')
                      } catch (error) {
                        message.error(getErrorMessage(error, '删除节点失败'))
                      }
                    },
                  })
                }}
              >
                手动删除节点
              </Button>
              {deployLastHeartbeat ? (
                <Typography.Text type="secondary">
                  最后心跳：{formatDateTime(deployLastHeartbeat)}
                </Typography.Text>
              ) : null}
            </Space>
            <Typography.Paragraph copyable={{ text: deployResult.command }} style={{ marginBottom: 0 }}>
              <pre style={{ margin: 0, whiteSpace: 'pre-wrap', wordBreak: 'break-word' }}>
                {deployResult.command}
              </pre>
            </Typography.Paragraph>
          </Space>
        ) : null}
      </Modal>
    </PageContainer>
  )
}

export default MyNodes
