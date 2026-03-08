import { useState } from 'react'
import { useQuery, useMutation } from '@tanstack/react-query'
import {
  Card,
  Table,
  Button,
  Space,
  Tag,
  Modal,
  Form,
  Input,
  Select,
  DatePicker,
  InputNumber,
  message,
  Popconfirm,
  Drawer,
  Descriptions,
  Badge,
} from 'antd'
import {
  PlusOutlined,
  ReloadOutlined,
  DeleteOutlined,
  StopOutlined,
  CheckOutlined,
  SwapOutlined,
} from '@ant-design/icons'
import { licenseApi, planApi } from '@/api'
import type { LicenseKey, LicensePlan } from '@/types'
import dayjs from 'dayjs'

export default function Licenses() {
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(20)
  const [filters, setFilters] = useState<any>({})
  const [generateModalOpen, setGenerateModalOpen] = useState(false)
  const [transferModalOpen, setTransferModalOpen] = useState(false)
  const [detailDrawerOpen, setDetailDrawerOpen] = useState(false)
  const [selectedLicense, setSelectedLicense] = useState<LicenseKey | null>(null)
  const [selectedRowKeys, setSelectedRowKeys] = useState<number[]>([])

  const [generateForm] = Form.useForm()
  const [transferForm] = Form.useForm()

  const { data, isLoading, refetch } = useQuery({
    queryKey: ['licenses', page, pageSize, filters],
    queryFn: () => licenseApi.list({ ...filters, page, page_size: pageSize }),
  })

  const { data: plansData } = useQuery({
    queryKey: ['plans'],
    queryFn: () => planApi.list(),
  })

  const generateMutation = useMutation({
    mutationFn: licenseApi.generate,
    onSuccess: () => {
      message.success('生成成功')
      setGenerateModalOpen(false)
      generateForm.resetFields()
      refetch()
    },
  })

  const revokeMutation = useMutation({
    mutationFn: licenseApi.revoke,
    onSuccess: () => {
      message.success('吊销成功')
      refetch()
    },
  })

  const restoreMutation = useMutation({
    mutationFn: licenseApi.restore,
    onSuccess: () => {
      message.success('恢复成功')
      refetch()
    },
  })

  const deleteMutation = useMutation({
    mutationFn: licenseApi.delete,
    onSuccess: () => {
      message.success('删除成功')
      refetch()
    },
  })

  const transferMutation = useMutation({
    mutationFn: ({ id, data }: any) => licenseApi.transfer(id, data),
    onSuccess: () => {
      message.success('转移成功')
      setTransferModalOpen(false)
      transferForm.resetFields()
      refetch()
    },
  })

  const batchRevokeMutation = useMutation({
    mutationFn: licenseApi.batchRevoke,
    onSuccess: () => {
      message.success('批量吊销成功')
      setSelectedRowKeys([])
      refetch()
    },
  })

  const batchRestoreMutation = useMutation({
    mutationFn: licenseApi.batchRestore,
    onSuccess: () => {
      message.success('批量恢复成功')
      setSelectedRowKeys([])
      refetch()
    },
  })

  const batchDeleteMutation = useMutation({
    mutationFn: licenseApi.batchDelete,
    onSuccess: () => {
      message.success('批量删除成功')
      setSelectedRowKeys([])
      refetch()
    },
  })

  const columns = [
    {
      title: '授权码',
      dataIndex: 'key',
      key: 'key',
      width: 200,
      render: (text: string) => <code>{text}</code>,
    },
    {
      title: '客户',
      dataIndex: 'customer',
      key: 'customer',
    },
    {
      title: '套餐',
      dataIndex: ['plan', 'name'],
      key: 'plan',
    },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      render: (status: string) => {
        const colorMap: any = {
          active: 'success',
          expired: 'error',
          revoked: 'default',
        }
        const textMap: any = {
          active: '活跃',
          expired: '已过期',
          revoked: '已吊销',
        }
        return <Tag color={colorMap[status]}>{textMap[status]}</Tag>
      },
    },
    {
      title: '过期时间',
      dataIndex: 'expires_at',
      key: 'expires_at',
      render: (text: string) => text ? dayjs(text).format('YYYY-MM-DD HH:mm') : '-',
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
      width: 280,
      render: (_: any, record: LicenseKey) => (
        <Space size="small">
          <Button
            type="link"
            size="small"
            onClick={() => {
              setSelectedLicense(record)
              setDetailDrawerOpen(true)
            }}
          >
            详情
          </Button>
          {record.status === 'active' && (
            <Popconfirm
              title="确定要吊销吗？"
              onConfirm={() => revokeMutation.mutate(record.id)}
            >
              <Button type="link" size="small" danger icon={<StopOutlined />}>
                吊销
              </Button>
            </Popconfirm>
          )}
          {record.status === 'revoked' && (
            <Button
              type="link"
              size="small"
              icon={<CheckOutlined />}
              onClick={() => restoreMutation.mutate(record.id)}
            >
              恢复
            </Button>
          )}
          <Button
            type="link"
            size="small"
            icon={<SwapOutlined />}
            onClick={() => {
              setSelectedLicense(record)
              setTransferModalOpen(true)
            }}
          >
            转移
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
      title="授权码管理"
      extra={
        <Space>
          <Button icon={<ReloadOutlined />} onClick={() => refetch()}>
            刷新
          </Button>
          <Button
            type="primary"
            icon={<PlusOutlined />}
            onClick={() => setGenerateModalOpen(true)}
          >
            生成授权码
          </Button>
        </Space>
      }
    >
      <Space style={{ marginBottom: 16 }}>
        <Select
          placeholder="状态"
          style={{ width: 120 }}
          allowClear
          onChange={(value) => setFilters({ ...filters, status: value })}
        >
          <Select.Option value="active">活跃</Select.Option>
          <Select.Option value="expired">已过期</Select.Option>
          <Select.Option value="revoked">已吊销</Select.Option>
        </Select>
        <Input
          placeholder="客户名称"
          style={{ width: 200 }}
          allowClear
          onChange={(e) => setFilters({ ...filters, customer: e.target.value })}
        />
        {selectedRowKeys.length > 0 && (
          <>
            <Popconfirm
              title="确定要批量吊销吗？"
              onConfirm={() => batchRevokeMutation.mutate(selectedRowKeys)}
            >
              <Button danger>批量吊销</Button>
            </Popconfirm>
            <Button onClick={() => batchRestoreMutation.mutate(selectedRowKeys)}>
              批量恢复
            </Button>
            <Popconfirm
              title="确定要批量删除吗？"
              onConfirm={() => batchDeleteMutation.mutate(selectedRowKeys)}
            >
              <Button danger>批量删除</Button>
            </Popconfirm>
          </>
        )}
      </Space>

      <Table
        rowSelection={{
          selectedRowKeys,
          onChange: (keys) =>
            setSelectedRowKeys(
              keys
                .map((key) => Number(key))
                .filter((id) => Number.isFinite(id)),
            ),
        }}
        columns={columns}
        dataSource={data?.data?.items || []}
        loading={isLoading}
        rowKey="id"
        pagination={{
          current: page,
          pageSize,
          total: data?.data?.total || 0,
          showSizeChanger: true,
          showTotal: (total) => `共 ${total} 条`,
          onChange: (p, ps) => {
            setPage(p)
            setPageSize(ps)
          },
        }}
      />

      <Modal
        title="生成授权码"
        open={generateModalOpen}
        onCancel={() => setGenerateModalOpen(false)}
        onOk={() => generateForm.submit()}
        confirmLoading={generateMutation.isPending}
      >
        <Form
          form={generateForm}
          layout="vertical"
          onFinish={(values) => {
            generateMutation.mutate({
              ...values,
              expires_at: values.expires_at?.toISOString(),
            })
          }}
        >
          <Form.Item
            name="plan_id"
            label="套餐"
            rules={[{ required: true, message: '请选择套餐' }]}
          >
            <Select placeholder="请选择套餐">
              {plansData?.data?.map((plan: LicensePlan) => (
                <Select.Option key={plan.id} value={plan.id}>
                  {plan.name}
                </Select.Option>
              ))}
            </Select>
          </Form.Item>
          <Form.Item
            name="customer"
            label="客户名称"
            rules={[{ required: true, message: '请输入客户名称' }]}
          >
            <Input placeholder="请输入客户名称" />
          </Form.Item>
          <Form.Item
            name="count"
            label="生成数量"
            rules={[{ required: true, message: '请输入生成数量' }]}
            initialValue={1}
          >
            <InputNumber min={1} max={200} style={{ width: '100%' }} />
          </Form.Item>
          <Form.Item name="expires_at" label="过期时间">
            <DatePicker showTime style={{ width: '100%' }} />
          </Form.Item>
          <Form.Item name="max_machines" label="最大机器数">
            <InputNumber min={1} style={{ width: '100%' }} />
          </Form.Item>
          <Form.Item name="note" label="备注">
            <Input.TextArea rows={3} />
          </Form.Item>
        </Form>
      </Modal>

      <Modal
        title="转移授权码"
        open={transferModalOpen}
        onCancel={() => setTransferModalOpen(false)}
        onOk={() => transferForm.submit()}
        confirmLoading={transferMutation.isPending}
      >
        <Form
          form={transferForm}
          layout="vertical"
          onFinish={(values) => {
            transferMutation.mutate({
              id: selectedLicense?.id,
              data: values,
            })
          }}
        >
          <Form.Item label="当前客户">
            <Input value={selectedLicense?.customer} disabled />
          </Form.Item>
          <Form.Item
            name="to_customer"
            label="目标客户"
            rules={[{ required: true, message: '请输入目标客户' }]}
          >
            <Input placeholder="请输入目标客户名称" />
          </Form.Item>
          <Form.Item name="reason" label="转移原因">
            <Input.TextArea rows={3} placeholder="请输入转移原因" />
          </Form.Item>
        </Form>
      </Modal>

      <Drawer
        title="授权码详情"
        open={detailDrawerOpen}
        onClose={() => setDetailDrawerOpen(false)}
        width={600}
      >
        {selectedLicense && (
          <Descriptions column={1} bordered>
            <Descriptions.Item label="授权码">
              <code>{selectedLicense.key}</code>
            </Descriptions.Item>
            <Descriptions.Item label="客户">{selectedLicense.customer}</Descriptions.Item>
            <Descriptions.Item label="套餐">{selectedLicense.plan.name}</Descriptions.Item>
            <Descriptions.Item label="状态">
              <Badge
                status={selectedLicense.status === 'active' ? 'success' : 'error'}
                text={selectedLicense.status}
              />
            </Descriptions.Item>
            <Descriptions.Item label="过期时间">
              {selectedLicense.expires_at
                ? dayjs(selectedLicense.expires_at).format('YYYY-MM-DD HH:mm:ss')
                : '-'}
            </Descriptions.Item>
            <Descriptions.Item label="最大机器数">
              {selectedLicense.max_machines || selectedLicense.plan.max_machines}
            </Descriptions.Item>
            <Descriptions.Item label="备注">{selectedLicense.note || '-'}</Descriptions.Item>
            <Descriptions.Item label="创建时间">
              {dayjs(selectedLicense.created_at).format('YYYY-MM-DD HH:mm:ss')}
            </Descriptions.Item>
          </Descriptions>
        )}
      </Drawer>
    </Card>
  )
}
