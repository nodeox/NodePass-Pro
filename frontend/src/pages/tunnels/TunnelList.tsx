import {
  DeleteOutlined,
  PauseCircleOutlined,
  PlayCircleOutlined,
  PlusOutlined,
  ReloadOutlined,
} from '@ant-design/icons'
import {
  Button,
  Card,
  Checkbox,
  Form,
  Input,
  InputNumber,
  Modal,
  Popconfirm,
  Select,
  Space,
  Table,
  Tag,
  Typography,
  message,
} from 'antd'
import { useCallback, useEffect, useMemo, useState } from 'react'

import PageContainer from '../../components/common/PageContainer'
import { usePageTitle } from '../../hooks/usePageTitle'
import { useAuthStore } from '../../store/auth'
import { nodeGroupApi, tunnelApi } from '../../services/nodeGroupApi'
import type {
  ForwardTarget,
  LoadBalanceStrategy,
  NodeGroup,
  Tunnel,
  TunnelConfig,
} from '../../types/nodeGroup'
import { getErrorMessage } from '../../utils/error'
import { formatDateTime } from '../../utils/format'

type TunnelFormValues = {
  name: string
  description?: string
  protocol: 'tcp' | 'udp' | 'ws' | 'tls'
  entry_group_id: number
  exit_group_id?: number
  listen_host: string
  listen_port?: number
  remote_host: string
  remote_port: number
  load_balance_strategy: LoadBalanceStrategy
  ip_type: 'ipv4' | 'ipv6' | 'auto'
  enable_proxy_protocol: boolean
  forward_targets: ForwardTarget[]
}

const TunnelList = () => {
  const user = useAuthStore((state) => state.user)
  const isAdmin = user?.role === 'admin'

  usePageTitle(isAdmin ? '隧道管理' : '我的隧道')

  const [list, setList] = useState<Tunnel[]>([])
  const [loading, setLoading] = useState<boolean>(false)
  const [groups, setGroups] = useState<NodeGroup[]>([])
  const [open, setOpen] = useState<boolean>(false)
  const [submitting, setSubmitting] = useState<boolean>(false)
  const [form] = Form.useForm<TunnelFormValues>()

  const loadData = useCallback(async () => {
    setLoading(true)
    try {
      const [tunnelResult, groupResult] = await Promise.all([
        tunnelApi.list({ page: 1, page_size: 200 }),
        nodeGroupApi.list({ page: 1, page_size: 200 }),
      ])
      setList(tunnelResult.items ?? [])
      setGroups(groupResult.items ?? [])
    } catch (error) {
      message.error(getErrorMessage(error, '隧道数据加载失败'))
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    void loadData()
  }, [loadData])

  const entryGroups = useMemo(
    () =>
      groups
        .filter((group) => group.type === 'entry' && group.is_enabled)
        .map((group) => ({
          label: `${group.name} (#${group.id})`,
          value: group.id,
        })),
    [groups],
  )

  const exitGroups = useMemo(
    () =>
      groups
        .filter((group) => group.type === 'exit' && group.is_enabled)
        .map((group) => ({
          label: `${group.name} (#${group.id})`,
          value: group.id,
        })),
    [groups],
  )

  const exitGroupId = Form.useWatch('exit_group_id', form)

  const handleCreate = async (values: TunnelFormValues) => {
    setSubmitting(true)
    try {
      const config: TunnelConfig = {
        load_balance_strategy: values.load_balance_strategy,
        ip_type: values.ip_type,
        enable_proxy_protocol: values.enable_proxy_protocol,
        forward_targets: values.forward_targets || [],
      }

      await tunnelApi.create({
        name: values.name.trim(),
        description: values.description?.trim() || undefined,
        protocol: values.protocol,
        entry_group_id: values.entry_group_id,
        exit_group_id: values.exit_group_id || undefined,
        listen_host: values.listen_host?.trim() || '0.0.0.0',
        listen_port: values.listen_port || undefined,
        remote_host: values.remote_host.trim(),
        remote_port: values.remote_port,
        config,
      })
      message.success('隧道创建成功')
      setOpen(false)
      form.resetFields()
      await loadData()
    } catch (error) {
      message.error(getErrorMessage(error, '创建隧道失败'))
    } finally {
      setSubmitting(false)
    }
  }

  const changeStatus = async (
    id: number,
    action: 'start' | 'stop',
    successText: string,
  ) => {
    try {
      if (action === 'start') {
        await tunnelApi.start(id)
      } else {
        await tunnelApi.stop(id)
      }
      message.success(successText)
      await loadData()
    } catch (error) {
      message.error(getErrorMessage(error, '操作失败'))
    }
  }

  const handleDelete = async (id: number) => {
    try {
      await tunnelApi.delete(id)
      message.success('隧道删除成功')
      await loadData()
    } catch (error) {
      message.error(getErrorMessage(error, '删除隧道失败'))
    }
  }

  const openCreateModal = () => {
    form.resetFields()
    form.setFieldsValue({
      protocol: 'tcp',
      listen_host: '0.0.0.0',
      load_balance_strategy: 'round_robin',
      ip_type: 'auto',
      enable_proxy_protocol: false,
      forward_targets: [],
    })
    setOpen(true)
  }

  return (
    <PageContainer
      title={isAdmin ? '隧道管理' : '我的隧道'}
      description={
        isAdmin
          ? '管理所有用户的隧道，支持多种协议和负载均衡策略。'
          : '管理您创建的隧道，支持多种协议和负载均衡策略。'
      }
      extra={
        <Space>
          <Typography.Text type="secondary">共 {list.length} 条</Typography.Text>
          <Button icon={<ReloadOutlined />} onClick={() => void loadData()} loading={loading}>
            刷新
          </Button>
          <Button type="primary" icon={<PlusOutlined />} onClick={openCreateModal}>
            创建隧道
          </Button>
        </Space>
      }
    >
      <Table<Tunnel>
        rowKey="id"
        loading={loading}
        dataSource={list}
        pagination={{ pageSize: 20, showSizeChanger: true, showTotal: (t) => `共 ${t} 条` }}
        scroll={{ x: 1400 }}
        columns={[
          { title: 'ID', dataIndex: 'id', width: 70, fixed: 'left' },
          { title: '名称', dataIndex: 'name', width: 160, fixed: 'left' },
          {
            title: '协议',
            dataIndex: 'protocol',
            width: 90,
            render: (v: string) => <Tag color="blue">{v.toUpperCase()}</Tag>,
          },
          {
            title: '入口组',
            width: 140,
            render: (_, record) => record.entry_group?.name || `#${record.entry_group_id}`,
          },
          {
            title: '出口组',
            width: 140,
            render: (_, record) =>
              record.exit_group_id
                ? record.exit_group?.name || `#${record.exit_group_id}`
                : <Tag>直连</Tag>,
          },
          {
            title: '监听',
            width: 180,
            render: (_, record) => `${record.listen_host}:${record.listen_port || '自动'}`,
          },
          {
            title: '目标',
            width: 180,
            render: (_, record) => `${record.remote_host}:${record.remote_port}`,
          },
          {
            title: '状态',
            dataIndex: 'status',
            width: 100,
            render: (v: string) => {
              const colors: Record<string, string> = {
                running: 'green',
                stopped: 'default',
                error: 'red',
              }
              return <Tag color={colors[v] || 'default'}>{v}</Tag>
            },
          },
          {
            title: '创建时间',
            dataIndex: 'created_at',
            width: 170,
            render: (v: string) => formatDateTime(v),
          },
          {
            title: '操作',
            fixed: 'right',
            width: 240,
            render: (_, record) => (
              <Space size="small">
                {record.status === 'running' ? (
                  <Button
                    type="link"
                    size="small"
                    icon={<PauseCircleOutlined />}
                    onClick={() => void changeStatus(record.id, 'stop', '隧道已停止')}
                  >
                    停止
                  </Button>
                ) : (
                  <Button
                    type="link"
                    size="small"
                    icon={<PlayCircleOutlined />}
                    onClick={() => void changeStatus(record.id, 'start', '隧道已启动')}
                  >
                    启动
                  </Button>
                )}
                <Popconfirm
                  title="确定删除该隧道吗？"
                  okText="删除"
                  cancelText="取消"
                  onConfirm={() => void handleDelete(record.id)}
                >
                  <Button type="link" size="small" danger icon={<DeleteOutlined />}>
                    删除
                  </Button>
                </Popconfirm>
              </Space>
            ),
          },
        ]}
      />

      <Modal
        title="创建隧道"
        open={open}
        onCancel={() => setOpen(false)}
        onOk={() => void form.submit()}
        confirmLoading={submitting}
        width={900}
        destroyOnClose
      >
        <Form<TunnelFormValues>
          form={form}
          layout="vertical"
          onFinish={(v) => void handleCreate(v)}
        >
          <Form.Item label="隧道名称" name="name" rules={[{ required: true, message: '请输入隧道名称' }]}>
            <Input placeholder="例如：美国入口到日本出口" maxLength={100} />
          </Form.Item>

          <Form.Item label="描述" name="description">
            <Input.TextArea rows={2} placeholder="可选，隧道的用途说明" maxLength={500} />
          </Form.Item>

          <Space wrap className="w-full">
            <Form.Item label="协议" name="protocol" rules={[{ required: true }]}>
              <Select
                style={{ width: 140 }}
                options={[
                  { label: 'TCP', value: 'tcp' },
                  { label: 'UDP', value: 'udp' },
                  { label: 'WS 加密', value: 'ws' },
                  { label: 'TLS 加密', value: 'tls' },
                ]}
              />
            </Form.Item>

            <Form.Item label="IP类型" name="ip_type">
              <Select
                style={{ width: 120 }}
                options={[
                  { label: '自动', value: 'auto' },
                  { label: 'IPv4', value: 'ipv4' },
                  { label: 'IPv6', value: 'ipv6' },
                ]}
              />
            </Form.Item>

            <Form.Item label="负载均衡" name="load_balance_strategy">
              <Select
                style={{ width: 180 }}
                options={[
                  { label: '轮询', value: 'round_robin' },
                  { label: '最少连接数', value: 'least_connections' },
                  { label: '随机', value: 'random' },
                  { label: '主备', value: 'failover' },
                  { label: '哈希', value: 'hash' },
                  { label: '最小延迟', value: 'latency' },
                ]}
              />
            </Form.Item>

            <Form.Item label="Proxy Protocol" name="enable_proxy_protocol" valuePropName="checked">
              <Checkbox>启用</Checkbox>
            </Form.Item>
          </Space>

          <Space wrap className="w-full">
            <Form.Item label="入口节点组" name="entry_group_id" rules={[{ required: true, message: '请选择入口节点组' }]}>
              <Select
                style={{ width: 280 }}
                options={entryGroups}
                showSearch
                optionFilterProp="label"
                placeholder="选择入口节点组"
              />
            </Form.Item>

            <Form.Item label="出口节点组" name="exit_group_id" tooltip="可选，不选择则为直连模式">
              <Select
                style={{ width: 280 }}
                options={exitGroups}
                showSearch
                optionFilterProp="label"
                placeholder="可选，不选择则直连"
                allowClear
              />
            </Form.Item>
          </Space>

          <Space wrap className="w-full">
            <Form.Item label="监听地址" name="listen_host" tooltip="为空则监听所有地址">
              <Input style={{ width: 180 }} placeholder="0.0.0.0" />
            </Form.Item>

            <Form.Item label="监听端口" name="listen_port" tooltip="为空则自动分配">
              <InputNumber style={{ width: 140 }} min={1} max={65535} precision={0} placeholder="自动分配" />
            </Form.Item>

            <Form.Item label="目标地址" name="remote_host" rules={[{ required: true, message: '请输入目标地址' }]}>
              <Input style={{ width: 220 }} placeholder="目标主机或IP" />
            </Form.Item>

            <Form.Item label="目标端口" name="remote_port" rules={[{ required: true, message: '请输入目标端口' }]}>
              <InputNumber style={{ width: 140 }} min={1} max={65535} precision={0} />
            </Form.Item>
          </Space>

          {exitGroupId && (
            <Card title="转发地址配置" size="small" style={{ marginTop: 16 }}>
              <Typography.Paragraph type="secondary" style={{ marginBottom: 12 }}>
                可配置多个转发地址，权重仅对随机负载均衡有效
              </Typography.Paragraph>
              <Form.List name="forward_targets">
                {(fields, { add, remove }) => (
                  <>
                    {fields.map((field) => (
                      <Space key={field.key} align="baseline" style={{ marginBottom: 8 }}>
                        <Form.Item
                          {...field}
                          name={[field.name, 'host']}
                          rules={[{ required: true, message: '请输入地址' }]}
                          style={{ marginBottom: 0 }}
                        >
                          <Input placeholder="主机或IP" style={{ width: 200 }} />
                        </Form.Item>
                        <Form.Item
                          {...field}
                          name={[field.name, 'port']}
                          rules={[{ required: true, message: '请输入端口' }]}
                          style={{ marginBottom: 0 }}
                        >
                          <InputNumber placeholder="端口" min={1} max={65535} style={{ width: 120 }} />
                        </Form.Item>
                        <Form.Item
                          {...field}
                          name={[field.name, 'weight']}
                          initialValue={1}
                          style={{ marginBottom: 0 }}
                        >
                          <InputNumber placeholder="权重" min={0} max={100} style={{ width: 100 }} />
                        </Form.Item>
                        <Button type="link" danger onClick={() => remove(field.name)}>
                          删除
                        </Button>
                      </Space>
                    ))}
                    <Button type="dashed" onClick={() => add()} block>
                      + 添加转发地址
                    </Button>
                  </>
                )}
              </Form.List>
            </Card>
          )}
        </Form>
      </Modal>
    </PageContainer>
  )
}

export default TunnelList
