import { useState } from 'react'
import { useQuery, useMutation } from '@tanstack/react-query'
import { Card, Table, Button, Space, Tag, Modal, Form, Input, InputNumber, Switch, message, Popconfirm } from 'antd'
import { PlusOutlined, EditOutlined, DeleteOutlined } from '@ant-design/icons'
import { planApi } from '@/api'
import type { LicensePlan } from '@/types'

export default function Plans() {
  const [modalOpen, setModalOpen] = useState(false)
  const [editingPlan, setEditingPlan] = useState<LicensePlan | null>(null)
  const [form] = Form.useForm()

  const { data, isLoading, refetch } = useQuery({
    queryKey: ['plans'],
    queryFn: () => planApi.list(),
  })

  const createMutation = useMutation({
    mutationFn: planApi.create,
    onSuccess: () => {
      message.success('创建成功')
      setModalOpen(false)
      form.resetFields()
      refetch()
    },
  })

  const updateMutation = useMutation({
    mutationFn: ({ id, data }: any) => planApi.update(id, data),
    onSuccess: () => {
      message.success('更新成功')
      setModalOpen(false)
      setEditingPlan(null)
      form.resetFields()
      refetch()
    },
  })

  const deleteMutation = useMutation({
    mutationFn: planApi.delete,
    onSuccess: () => {
      message.success('删除成功')
      refetch()
    },
  })

  const handleEdit = (record: LicensePlan) => {
    setEditingPlan(record)
    form.setFieldsValue(record)
    setModalOpen(true)
  }

  const handleSubmit = (values: any) => {
    if (editingPlan) {
      updateMutation.mutate({ id: editingPlan.id, data: values })
    } else {
      createMutation.mutate(values)
    }
  }

  const columns = [
    {
      title: '套餐名称',
      dataIndex: 'name',
      key: 'name',
    },
    {
      title: '套餐代码',
      dataIndex: 'code',
      key: 'code',
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
      title: '最大机器数',
      dataIndex: 'max_machines',
      key: 'max_machines',
    },
    {
      title: '有效期（天）',
      dataIndex: 'duration_days',
      key: 'duration_days',
    },
    {
      title: '描述',
      dataIndex: 'description',
      key: 'description',
      ellipsis: true,
    },
    {
      title: '操作',
      key: 'action',
      render: (_: any, record: LicensePlan) => (
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
      title="套餐管理"
      extra={
        <Button
          type="primary"
          icon={<PlusOutlined />}
          onClick={() => {
            setEditingPlan(null)
            form.resetFields()
            setModalOpen(true)
          }}
        >
          新建套餐
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
        title={editingPlan ? '编辑套餐' : '新建套餐'}
        open={modalOpen}
        onCancel={() => {
          setModalOpen(false)
          setEditingPlan(null)
          form.resetFields()
        }}
        onOk={() => form.submit()}
        confirmLoading={createMutation.isPending || updateMutation.isPending}
        width={800}
      >
        <Form
          form={form}
          layout="vertical"
          onFinish={handleSubmit}
          initialValues={{
            is_enabled: true,
            max_machines: 1,
            duration_days: 365,
            min_panel_version: '0.1.0',
            max_panel_version: '9.9.9',
            min_backend_version: '0.1.0',
            max_backend_version: '9.9.9',
            min_frontend_version: '0.1.0',
            max_frontend_version: '9.9.9',
            min_nodeclient_version: '0.1.0',
            max_nodeclient_version: '9.9.9',
          }}
        >
          <Form.Item
            name="name"
            label="套餐名称"
            rules={[{ required: true, message: '请输入套餐名称' }]}
          >
            <Input placeholder="例如：标准版" />
          </Form.Item>

          <Form.Item
            name="code"
            label="套餐代码"
            rules={[{ required: true, message: '请输入套餐代码' }]}
          >
            <Input placeholder="例如：standard" />
          </Form.Item>

          <Form.Item name="description" label="描述">
            <Input.TextArea rows={2} placeholder="套餐描述" />
          </Form.Item>

          <Space size="large">
            <Form.Item name="is_enabled" label="启用状态" valuePropName="checked">
              <Switch />
            </Form.Item>

            <Form.Item
              name="max_machines"
              label="最大机器数"
              rules={[{ required: true, message: '请输入最大机器数' }]}
            >
              <InputNumber min={1} />
            </Form.Item>

            <Form.Item
              name="duration_days"
              label="有效期（天）"
              rules={[{ required: true, message: '请输入有效期' }]}
            >
              <InputNumber min={1} />
            </Form.Item>
          </Space>

          <div style={{ marginTop: 16, marginBottom: 8, fontWeight: 600 }}>版本限制</div>

          <Space.Compact block>
            <Form.Item name="min_panel_version" label="Panel 最小版本" style={{ flex: 1 }}>
              <Input placeholder="0.1.0" />
            </Form.Item>
            <Form.Item name="max_panel_version" label="Panel 最大版本" style={{ flex: 1 }}>
              <Input placeholder="9.9.9" />
            </Form.Item>
          </Space.Compact>

          <Space.Compact block>
            <Form.Item name="min_backend_version" label="Backend 最小版本" style={{ flex: 1 }}>
              <Input placeholder="0.1.0" />
            </Form.Item>
            <Form.Item name="max_backend_version" label="Backend 最大版本" style={{ flex: 1 }}>
              <Input placeholder="9.9.9" />
            </Form.Item>
          </Space.Compact>

          <Space.Compact block>
            <Form.Item name="min_frontend_version" label="Frontend 最小版本" style={{ flex: 1 }}>
              <Input placeholder="0.1.0" />
            </Form.Item>
            <Form.Item name="max_frontend_version" label="Frontend 最大版本" style={{ flex: 1 }}>
              <Input placeholder="9.9.9" />
            </Form.Item>
          </Space.Compact>

          <Space.Compact block>
            <Form.Item name="min_nodeclient_version" label="Nodeclient 最小版本" style={{ flex: 1 }}>
              <Input placeholder="0.1.0" />
            </Form.Item>
            <Form.Item name="max_nodeclient_version" label="Nodeclient 最大版本" style={{ flex: 1 }}>
              <Input placeholder="9.9.9" />
            </Form.Item>
          </Space.Compact>
        </Form>
      </Modal>
    </Card>
  )
}
