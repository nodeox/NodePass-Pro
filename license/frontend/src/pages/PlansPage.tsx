import {
  Button,
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
  message
} from 'antd'
import dayjs from 'dayjs'
import { useEffect, useMemo, useState } from 'react'
import type { LicensePlan, PlanStatus } from '../types/api'
import { planApi } from '../utils/api'
import { extractErrorMessage } from '../utils/request'

type PlanFormValues = {
  code: string
  name: string
  description?: string
  max_machines: number
  duration_days: number
  status: PlanStatus
}

type CloneFormValues = {
  code?: string
  name?: string
  description?: string
  status?: PlanStatus
}

const toPlanStatus = (value: string | undefined): PlanStatus => (value === 'disabled' ? 'disabled' : 'active')

export default function PlansPage() {
  const [items, setItems] = useState<LicensePlan[]>([])
  const [loading, setLoading] = useState(false)
  const [actionLoadingId, setActionLoadingId] = useState<number>()
  const [open, setOpen] = useState(false)
  const [submitting, setSubmitting] = useState(false)
  const [editingPlan, setEditingPlan] = useState<LicensePlan | null>(null)
  const [cloneOpen, setCloneOpen] = useState(false)
  const [cloneSubmitting, setCloneSubmitting] = useState(false)
  const [cloningPlan, setCloningPlan] = useState<LicensePlan | null>(null)
  const [keyword, setKeyword] = useState('')
  const [statusFilter, setStatusFilter] = useState<PlanStatus | undefined>()
  const [form] = Form.useForm<PlanFormValues>()
  const [cloneForm] = Form.useForm<CloneFormValues>()

  const load = async () => {
    setLoading(true)
    try {
      setItems(await planApi.list())
    } catch (err) {
      message.error(extractErrorMessage(err))
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    void load()
  }, [])

  const filteredItems = useMemo(() => {
    const text = keyword.trim().toLowerCase()
    return items.filter((item) => {
      if (statusFilter && toPlanStatus(item.status) !== statusFilter) {
        return false
      }
      if (!text) return true
      return (
        item.code.toLowerCase().includes(text) ||
        item.name.toLowerCase().includes(text) ||
        (item.description || '').toLowerCase().includes(text)
      )
    })
  }, [items, keyword, statusFilter])

  const openCreateModal = () => {
    setEditingPlan(null)
    form.setFieldsValue({
      code: '',
      name: '',
      description: '',
      max_machines: 3,
      duration_days: 365,
      status: 'active'
    })
    setOpen(true)
  }

  const openEditModal = (record: LicensePlan) => {
    setEditingPlan(record)
    form.setFieldsValue({
      code: record.code,
      name: record.name,
      description: record.description,
      max_machines: record.max_machines,
      duration_days: record.duration_days,
      status: toPlanStatus(record.status)
    })
    setOpen(true)
  }

  const buildPlanPayload = (values: PlanFormValues) => ({
    code: values.code.trim(),
    name: values.name.trim(),
    description: values.description?.trim() ?? '',
    max_machines: values.max_machines,
    duration_days: values.duration_days,
    status: values.status
  })

  const onSubmit = async () => {
    setSubmitting(true)
    try {
      const values = await form.validateFields()
      const payload = buildPlanPayload(values)
      if (editingPlan) {
        await planApi.update(editingPlan.id, payload)
        message.success('套餐更新成功')
      } else {
        await planApi.create(payload)
        message.success('套餐创建成功')
      }
      setOpen(false)
      setEditingPlan(null)
      form.resetFields()
      await load()
    } catch (err) {
      message.error(extractErrorMessage(err))
    } finally {
      setSubmitting(false)
    }
  }

  const toggleStatus = async (record: LicensePlan) => {
    const nextStatus: PlanStatus = toPlanStatus(record.status) === 'active' ? 'disabled' : 'active'
    try {
      setActionLoadingId(record.id)
      await planApi.update(record.id, {
        code: record.code,
        name: record.name,
        description: record.description || '',
        max_machines: record.max_machines,
        duration_days: record.duration_days,
        status: nextStatus
      })
      message.success(nextStatus === 'active' ? '套餐已启用' : '套餐已停用')
      await load()
    } catch (err) {
      message.error(extractErrorMessage(err))
    } finally {
      setActionLoadingId(undefined)
    }
  }

  const openCloneModal = (record: LicensePlan) => {
    setCloningPlan(record)
    cloneForm.setFieldsValue({
      code: `${record.code}-COPY`,
      name: `${record.name} 副本`,
      description: record.description || '',
      status: toPlanStatus(record.status)
    })
    setCloneOpen(true)
  }

  const closeCloneModal = () => {
    setCloneOpen(false)
    setCloningPlan(null)
    cloneForm.resetFields()
  }

  const clonePlan = async () => {
    if (!cloningPlan) return
    setCloneSubmitting(true)
    try {
      const values = await cloneForm.validateFields()
      const payload = {
        code: values.code?.trim() || undefined,
        name: values.name?.trim() || undefined,
        description: values.description === undefined ? undefined : values.description.trim(),
        status: values.status
      }
      const created = await planApi.clone(cloningPlan.id, payload)
      message.success(`套餐克隆成功：${created.code}`)
      closeCloneModal()
      await load()
    } catch (err) {
      message.error(extractErrorMessage(err))
    } finally {
      setCloneSubmitting(false)
    }
  }

  const removePlan = async (record: LicensePlan, force = false) => {
    try {
      setActionLoadingId(record.id)
      await planApi.remove(record.id, force)
      message.success(force ? '套餐及关联授权已删除' : '套餐删除成功')
      await load()
    } catch (err) {
      message.error(extractErrorMessage(err))
    } finally {
      setActionLoadingId(undefined)
    }
  }

  return (
    <>
      <Space style={{ marginBottom: 16, width: '100%', justifyContent: 'space-between' }} wrap>
        <Space wrap>
          <Input
            allowClear
            placeholder="搜索编码/名称/描述"
            style={{ width: 280 }}
            value={keyword}
            onChange={(event) => setKeyword(event.target.value)}
          />
          <Select
            allowClear
            placeholder="状态"
            style={{ width: 140 }}
            value={statusFilter}
            onChange={(value) => setStatusFilter(value as PlanStatus | undefined)}
            options={[
              { label: 'active', value: 'active' },
              { label: 'disabled', value: 'disabled' }
            ]}
          />
          <Button onClick={() => void load()}>刷新</Button>
        </Space>

        <Space>
          <Typography.Text type="secondary">共 {filteredItems.length} 个套餐</Typography.Text>
          <Button type="primary" onClick={openCreateModal}>
            新建套餐
          </Button>
        </Space>
      </Space>

      <Table
        rowKey="id"
        loading={loading}
        dataSource={filteredItems}
        pagination={false}
        scroll={{ x: 1200 }}
        columns={[
          { title: '编码', dataIndex: 'code', width: 160 },
          { title: '名称', dataIndex: 'name', width: 200 },
          {
            title: '描述',
            dataIndex: 'description',
            width: 260,
            render: (value?: string) => value || '-'
          },
          { title: '最大设备', dataIndex: 'max_machines', width: 110 },
          { title: '有效期(天)', dataIndex: 'duration_days', width: 120 },
          {
            title: '关联授权',
            dataIndex: 'license_count',
            width: 110,
            render: (value?: number) => value ?? 0
          },
          {
            title: '活跃授权',
            dataIndex: 'active_license_count',
            width: 110,
            render: (value?: number) => value ?? 0
          },
          {
            title: '绑定设备',
            dataIndex: 'activation_count',
            width: 110,
            render: (value?: number) => value ?? 0
          },
          {
            title: '状态',
            dataIndex: 'status',
            width: 120,
            render: (value: string) => {
              const status = toPlanStatus(value)
              const color = status === 'active' ? 'green' : 'orange'
              return <Tag color={color}>{status}</Tag>
            }
          },
          {
            title: '更新时间',
            dataIndex: 'updated_at',
            width: 170,
            render: (value: string) => dayjs(value).format('YYYY-MM-DD HH:mm')
          },
          {
            title: '操作',
            fixed: 'right',
            width: 360,
            render: (_, record) => {
              const status = toPlanStatus(record.status)
              const hasLicense = (record.license_count ?? 0) > 0
              const isActionLoading = actionLoadingId === record.id
              return (
                <Space>
                  <Button size="small" onClick={() => openEditModal(record)} loading={isActionLoading}>
                    编辑
                  </Button>
                  <Button size="small" onClick={() => openCloneModal(record)} loading={isActionLoading}>
                    克隆
                  </Button>
                  <Popconfirm
                    title={status === 'active' ? '确认停用该套餐？' : '确认启用该套餐？'}
                    onConfirm={() => toggleStatus(record)}
                    okText="确认"
                    cancelText="取消"
                  >
                    <Button size="small" loading={isActionLoading}>
                      {status === 'active' ? '停用' : '启用'}
                    </Button>
                  </Popconfirm>
                  <Popconfirm
                    title="确认删除该套餐？"
                    description={hasLicense ? '将同时永久删除该套餐下所有授权和绑定记录，删除后不可恢复。' : '删除后不可恢复。'}
                    onConfirm={() => removePlan(record, hasLicense)}
                    okText="确认"
                    cancelText="取消"
                  >
                    <Button danger size="small" loading={isActionLoading}>
                      {hasLicense ? '强制删除' : '删除'}
                    </Button>
                  </Popconfirm>
                </Space>
              )
            }
          }
        ]}
      />

      <Modal
        title={editingPlan ? `编辑套餐 #${editingPlan.id}` : '新建套餐'}
        open={open}
        onCancel={() => {
          setOpen(false)
          setEditingPlan(null)
          form.resetFields()
        }}
        onOk={() => void onSubmit()}
        confirmLoading={submitting}
        destroyOnClose
      >
        <Form form={form} layout="vertical" initialValues={{ max_machines: 3, duration_days: 365, status: 'active' }}>
          <Form.Item label="编码" name="code" rules={[{ required: true, message: '请输入编码' }]}>
            <Input placeholder="如: NP-PRO" />
          </Form.Item>
          <Form.Item label="名称" name="name" rules={[{ required: true, message: '请输入名称' }]}>
            <Input placeholder="套餐名称" />
          </Form.Item>
          <Form.Item label="描述" name="description">
            <Input.TextArea rows={3} />
          </Form.Item>
          <Form.Item label="最大设备" name="max_machines" rules={[{ required: true, message: '请输入最大设备数' }]}>
            <InputNumber min={1} style={{ width: '100%' }} />
          </Form.Item>
          <Form.Item label="有效期(天)" name="duration_days" rules={[{ required: true, message: '请输入有效天数' }]}>
            <InputNumber min={1} style={{ width: '100%' }} />
          </Form.Item>
          <Form.Item label="状态" name="status" rules={[{ required: true, message: '请选择状态' }]}>
            <Select
              options={[
                { label: 'active', value: 'active' },
                { label: 'disabled', value: 'disabled' }
              ]}
            />
          </Form.Item>
        </Form>
      </Modal>

      <Modal
        title={cloningPlan ? `克隆套餐 #${cloningPlan.id}` : '克隆套餐'}
        open={cloneOpen}
        onCancel={closeCloneModal}
        onOk={() => void clonePlan()}
        confirmLoading={cloneSubmitting}
        destroyOnClose
      >
        <Form form={cloneForm} layout="vertical" initialValues={{ status: 'active' }}>
          <Form.Item
            label="新编码"
            name="code"
            extra="可编辑；留空时后端自动生成。"
            rules={[{ max: 64, message: '编码长度不能超过 64' }]}
          >
            <Input placeholder="如: NP-PRO-COPY" />
          </Form.Item>
          <Form.Item label="新名称" name="name" extra="可编辑；留空时默认使用“原名称 + 副本”。">
            <Input placeholder="如: Pro 套餐副本" />
          </Form.Item>
          <Form.Item label="描述" name="description" extra="可编辑，默认继承原套餐描述。">
            <Input.TextArea rows={3} />
          </Form.Item>
          <Form.Item label="状态" name="status" extra="默认继承原套餐状态。">
            <Select
              allowClear
              options={[
                { label: 'active', value: 'active' },
                { label: 'disabled', value: 'disabled' }
              ]}
            />
          </Form.Item>
        </Form>
      </Modal>
    </>
  )
}
