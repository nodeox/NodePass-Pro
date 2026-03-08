import { useState } from 'react'
import { useQuery, useMutation } from '@tanstack/react-query'
import { Card, Table, Button, Space, Tag, Modal, Form, Input, Select, Switch, message, Popconfirm } from 'antd'
import { PlusOutlined, EditOutlined, DeleteOutlined } from '@ant-design/icons'
import { webhookApi } from '@/api'
import type { WebhookConfig } from '@/types'
import dayjs from 'dayjs'

const eventOptions = [
  { label: '授权码创建', value: 'license.created' },
  { label: '授权码过期', value: 'license.expired' },
  { label: '授权码吊销', value: 'license.revoked' },
  { label: '授权码转移', value: 'license.transferred' },
  { label: '告警创建', value: 'alert.created' },
  { label: '所有事件', value: '*' },
]

export default function Webhooks() {
  const [modalOpen, setModalOpen] = useState(false)
  const [editingWebhook, setEditingWebhook] = useState<WebhookConfig | null>(null)
  const [form] = Form.useForm()

  const { data, isLoading, refetch } = useQuery({
    queryKey: ['webhooks'],
    queryFn: () => webhookApi.list(),
  })

  const createMutation = useMutation({
    mutationFn: webhookApi.create,
    onSuccess: () => {
      message.success('创建成功')
      setModalOpen(false)
      form.resetFields()
      refetch()
    },
  })

  const updateMutation = useMutation({
    mutationFn: ({ id, data }: any) => webhookApi.update(id, data),
    onSuccess: () => {
      message.success('更新成功')
      setModalOpen(false)
      setEditingWebhook(null)
      form.resetFields()
      refetch()
    },
  })

  const deleteMutation = useMutation({
    mutationFn: webhookApi.delete,
    onSuccess: () => {
      message.success('删除成功')
      refetch()
    },
  })

  const handleEdit = (record: WebhookConfig) => {
    setEditingWebhook(record)
    const events = JSON.parse(record.events)
    form.setFieldsValue({ ...record, events })
    setModalOpen(true)
  }

  const handleSubmit = (values: any) => {
    if (editingWebhook) {
      updateMutation.mutate({ id: editingWebhook.id, data: values })
    } else {
      createMutation.mutate(values)
    }
  }

  const columns = [
    {
      title: '名称',
      dataIndex: 'name',
      key: 'name',
    },
    {
      title: 'URL',
      dataIndex: 'url',
      key: 'url',
      ellipsis: true,
    },
    {
      title: '事件',
      dataIndex: 'events',
      key: 'events',
      render: (events: string) => {
        const eventList = JSON.parse(events)
        return (
          <Space wrap>
            {eventList.map((event: string) => (
              <Tag key={event}>{event}</Tag>
            ))}
          </Space>
        )
      },
    },
    {
      title: '状态',
      dataIndex: 'is_enabled',
      key: 'is_enabled',
      render: (enabled: boolean) => (
        <Tag color={enabled ? 'success' : 'default'}>
          {enabled ? '启用' : '禁用'}
        </Tag>
      ),
    },
    {
      title: '创建时间',
      dataIndex: 'created_at',
      key: 'created_at',
      render: (text: string) => dayjs(text).format('YYYY-MM-DD HH:mm'),
    },
    {
      title: '操作',
      key: 'action',
      render: (_: any, record: WebhookConfig) => (
        <Space>
          <Button
            type="link"
            size="small"
            icon={<EditOutlined />}
            onClick={() => handleEdit(record)}
          >
            编辑
          </Button>
          <Popconfirm
            title="确定要删除吗？"
            onConfirm={() => deleteMutation.mutate(record.id)}
          >
            <Button type="link" size="small" danger icon={<DeleteOutlined />}>
              删除
            </Button>
          </Popconfirm>
        </Space>
      ),
    },
  ]

  return (
    <Card
      title="Webhook 管理"
      extra={
        <Button
          type="primary"
          icon={<PlusOutlined />}
          onClick={() => {
            setEditingWebhook(null)
            form.resetFields()
            setModalOpen(true)
          }}
        >
          新建 Webhook
        </Button>
      }
    >
      <Table
        columns={columns}
        dataSource={data?.data || []}
        loading={isLoading}
        rowKey="id"
        pagination={false}
      />

      <Modal
        title={editingWebhook ? '编辑 Webhook' : '新建 Webhook'}
        open={modalOpen}
        onCancel={() => {
          setModalOpen(false)
          setEditingWebhook(null)
          form.resetFields()
        }}
        onOk={() => form.submit()}
        confirmLoading={createMutation.isPending || updateMutation.isPending}
        width={600}
      >
        <Form
          form={form}
          layout="vertical"
          onFinish={handleSubmit}
          initialValues={{
            is_enabled: true,
            events: ['*'],
          }}
        >
          <Form.Item
            name="name"
            label="名称"
            rules={[{ required: true, message: '请输入名称' }]}
          >
            <Input placeholder="例如：通知服务" />
          </Form.Item>

          <Form.Item
            name="url"
            label="URL"
            rules={[
              { required: true, message: '请输入 URL' },
              { type: 'url', message: '请输入有效的 URL' },
            ]}
          >
            <Input placeholder="https://your-webhook-url.com/notify" />
          </Form.Item>

          <Form.Item name="secret" label="密钥">
            <Input.Password placeholder="用于签名验证（可选）" />
          </Form.Item>

          <Form.Item
            name="events"
            label="监听事件"
            rules={[{ required: true, message: '请选择事件' }]}
          >
            <Select mode="multiple" options={eventOptions} placeholder="请选择事件" />
          </Form.Item>

          <Form.Item name="is_enabled" label="启用状态" valuePropName="checked">
            <Switch />
          </Form.Item>
        </Form>
      </Modal>
    </Card>
  )
}
