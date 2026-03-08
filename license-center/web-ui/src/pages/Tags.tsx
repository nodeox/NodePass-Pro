import { useState } from 'react'
import { useQuery, useMutation } from '@tantml:react-query'
import { Card, Table, Button, Space, Tag, Modal, Form, Input, message, Popconfirm } from 'antd'
import { PlusOutlined, EditOutlined, DeleteOutlined } from '@ant-design/icons'
import { tagApi } from '@/api'
import type { LicenseTag } from '@/types'

const colorOptions = [
  '#f50', '#2db7f5', '#87d068', '#108ee9', '#ff4d4f',
  '#52c41a', '#faad14', '#722ed1', '#eb2f96', '#13c2c2',
]

export default function Tags() {
  const [modalOpen, setModalOpen] = useState(false)
  const [editingTag, setEditingTag] = useState<LicenseTag | null>(null)
  const [form] = Form.useForm()

  const { data, isLoading, refetch } = useQuery({
    queryKey: ['tags'],
    queryFn: () => tagApi.list(),
  })

  const createMutation = useMutation({
    mutationFn: tagApi.create,
    onSuccess: () => {
      message.success('创建成功')
      setModalOpen(false)
      form.resetFields()
      refetch()
    },
  })

  const updateMutation = useMutation({
    mutationFn: ({ id, data }: any) => tagApi.update(id, data),
    onSuccess: () => {
      message.success('更新成功')
      setModalOpen(false)
      setEditingTag(null)
      form.resetFields()
      refetch()
    },
  })

  const deleteMutation = useMutation({
    mutationFn: tagApi.delete,
    onSuccess: () => {
      message.success('删除成功')
      refetch()
    },
  })

  const handleEdit = (record: LicenseTag) => {
    setEditingTag(record)
    form.setFieldsValue(record)
    setModalOpen(true)
  }

  const handleSubmit = (values: any) => {
    if (editingTag) {
      updateMutation.mutate({ id: editingTag.id, data: values })
    } else {
      createMutation.mutate(values)
    }
  }

  const columns = [
    {
      title: '标签名称',
      dataIndex: 'name',
      key: 'name',
      render: (text: string, record: LicenseTag) => (
        <Tag color={record.color}>{text}</Tag>
      ),
    },
    {
      title: '颜色',
      dataIndex: 'color',
      key: 'color',
      render: (color: string) => (
        <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
          <div
            style={{
              width: 20,
              height: 20,
              backgroundColor: color,
              borderRadius: 4,
              border: '1px solid #d9d9d9',
            }}
          />
          <code>{color}</code>
        </div>
      ),
    },
    {
      title: '操作',
      key: 'action',
      render: (_: any, record: LicenseTag) => (
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
      title="标签管理"
      extra={
        <Button
          type="primary"
          icon={<PlusOutlined />}
          onClick={() => {
            setEditingTag(null)
            form.resetFields()
            setModalOpen(true)
          }}
        >
          新建标签
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
        title={editingTag ? '编辑标签' : '新建标签'}
        open={modalOpen}
        onCancel={() => {
          setModalOpen(false)
          setEditingTag(null)
          form.resetFields()
        }}
        onOk={() => form.submit()}
        confirmLoading={createMutation.isPending || updateMutation.isPending}
      >
        <Form
          form={form}
          layout="vertical"
          onFinish={handleSubmit}
          initialValues={{ color: colorOptions[0] }}
        >
          <Form.Item
            name="name"
            label="标签名称"
            rules={[{ required: true, message: '请输入标签名称' }]}
          >
            <Input placeholder="例如：VIP客户" />
          </Form.Item>

          <Form.Item name="color" label="颜色">
            <Space wrap>
              {colorOptions.map((color) => (
                <div
                  key={color}
                  style={{
                    width: 32,
                    height: 32,
                    backgroundColor: color,
                    borderRadius: 4,
                    cursor: 'pointer',
                    border: form.getFieldValue('color') === color ? '2px solid #1890ff' : '1px solid #d9d9d9',
                  }}
                  onClick={() => form.setFieldValue('color', color)}
                />
              ))}
            </Space>
          </Form.Item>
        </Form>
      </Modal>
    </Card>
  )
}
