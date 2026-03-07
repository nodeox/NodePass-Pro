import {
  EditOutlined,
  PlusOutlined,
  ReloadOutlined,
} from '@ant-design/icons'
import {
  Button,
  Form,
  Input,
  InputNumber,
  Modal,
  Space,
  Table,
  Tag,
  Typography,
  message,
} from 'antd'
import { useEffect, useState } from 'react'

import PageContainer from '../../components/common/PageContainer'
import { usePageTitle } from '../../hooks/usePageTitle'
import { vipApi } from '../../services/api'
import type { CreateVipLevelPayload, UpdateVipLevelPayload, VipLevelRecord } from '../../types'
import { getErrorMessage } from '../../utils/error'
import { formatTraffic } from '../../utils/format'

type VipFormMode = 'create' | 'edit'

type VipFormValues = {
  level: number
  name: string
  description?: string
  traffic_quota_gb: number
  max_rules: number
  max_bandwidth: number
  max_self_hosted_entry_nodes: number
  max_self_hosted_exit_nodes: number
  traffic_multiplier: number
  price?: number
}

const gbBytes = 1024 * 1024 * 1024

const formatLimit = (value: number, suffix = ''): string => {
  if (value < 0) {
    return `不限${suffix}`
  }
  return `${value}${suffix}`
}

const normalizeText = (value?: string): string | undefined => {
  if (!value) {
    return undefined
  }
  const trimmed = value.trim()
  return trimmed === '' ? undefined : trimmed
}

const VipLevelManage = () => {
  usePageTitle('VIP 等级管理')

  const [form] = Form.useForm<VipFormValues>()
  const [loading, setLoading] = useState<boolean>(false)
  const [saving, setSaving] = useState<boolean>(false)
  const [levels, setLevels] = useState<VipLevelRecord[]>([])
  const [modalOpen, setModalOpen] = useState<boolean>(false)
  const [formMode, setFormMode] = useState<VipFormMode>('create')
  const [editingLevel, setEditingLevel] = useState<VipLevelRecord | null>(null)

  const loadLevels = async (): Promise<void> => {
    setLoading(true)
    try {
      const result = await vipApi.levels()
      setLevels(result.list ?? [])
    } catch (error) {
      message.error(getErrorMessage(error, 'VIP 等级加载失败'))
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    void loadLevels()
  }, [])

  const openCreateModal = (): void => {
    setFormMode('create')
    setEditingLevel(null)
    form.setFieldsValue({
      level: 0,
      name: '',
      description: '',
      traffic_quota_gb: 10,
      max_rules: 5,
      max_bandwidth: 100,
      max_self_hosted_entry_nodes: 0,
      max_self_hosted_exit_nodes: 0,
      traffic_multiplier: 1,
      price: undefined,
    })
    setModalOpen(true)
  }

  const openEditModal = (level: VipLevelRecord): void => {
    setFormMode('edit')
    setEditingLevel(level)
    form.setFieldsValue({
      level: level.level,
      name: level.name,
      description: level.description ?? '',
      traffic_quota_gb: Number((level.traffic_quota / gbBytes).toFixed(2)),
      max_rules: level.max_rules,
      max_bandwidth: level.max_bandwidth,
      max_self_hosted_entry_nodes: level.max_self_hosted_entry_nodes,
      max_self_hosted_exit_nodes: level.max_self_hosted_exit_nodes,
      traffic_multiplier: level.traffic_multiplier ?? 1,
      price: level.price ?? undefined,
    })
    setModalOpen(true)
  }

  const closeModal = (): void => {
    setModalOpen(false)
    setEditingLevel(null)
    form.resetFields()
  }

  const handleSubmit = async (values: VipFormValues): Promise<void> => {
    const quotaBytes = Math.round((values.traffic_quota_gb ?? 0) * gbBytes)

    setSaving(true)
    try {
      if (formMode === 'create') {
        const payload: CreateVipLevelPayload = {
          level: values.level,
          name: values.name.trim(),
          description: normalizeText(values.description),
          traffic_quota: quotaBytes,
          max_rules: values.max_rules,
          max_bandwidth: values.max_bandwidth,
          max_self_hosted_entry_nodes: values.max_self_hosted_entry_nodes,
          max_self_hosted_exit_nodes: values.max_self_hosted_exit_nodes,
          traffic_multiplier: values.traffic_multiplier,
          price: values.price,
        }
        await vipApi.createLevel(payload)
        message.success('VIP 等级创建成功')
      } else if (editingLevel) {
        const payload: UpdateVipLevelPayload = {
          name: values.name.trim(),
          description: normalizeText(values.description),
          traffic_quota: quotaBytes,
          max_rules: values.max_rules,
          max_bandwidth: values.max_bandwidth,
          max_self_hosted_entry_nodes: values.max_self_hosted_entry_nodes,
          max_self_hosted_exit_nodes: values.max_self_hosted_exit_nodes,
          traffic_multiplier: values.traffic_multiplier,
          price: values.price,
        }
        await vipApi.updateLevel(editingLevel.id, payload)
        message.success('VIP 等级更新成功')
      }

      closeModal()
      await loadLevels()
    } catch (error) {
      message.error(getErrorMessage(error, '保存 VIP 等级失败'))
    } finally {
      setSaving(false)
    }
  }

  return (
    <PageContainer
      title="VIP 等级管理"
      description="管理员可创建和调整 VIP 权益。"
      extra={
        <Space>
          <Button
            icon={<ReloadOutlined />}
            onClick={() => void loadLevels()}
            loading={loading}
          >
            刷新
          </Button>
          <Button type="primary" icon={<PlusOutlined />} onClick={openCreateModal}>
            创建等级
          </Button>
        </Space>
      }
    >
      <Table<VipLevelRecord>
        rowKey="id"
        loading={loading}
        dataSource={levels}
        pagination={false}
        columns={[
          {
            title: '等级',
            dataIndex: 'level',
            width: 80,
            render: (value: number) => <Tag color="blue">Lv.{value}</Tag>,
          },
          {
            title: '名称',
            dataIndex: 'name',
            width: 120,
          },
          {
            title: '流量配额',
            dataIndex: 'traffic_quota',
            width: 130,
            render: (value: number) => formatTraffic(value),
          },
          {
            title: '规则数',
            dataIndex: 'max_rules',
            width: 100,
            render: (value: number) => formatLimit(value),
          },
          {
            title: '带宽',
            dataIndex: 'max_bandwidth',
            width: 120,
            render: (value: number) => formatLimit(value, ' Mbps'),
          },
          {
            title: '自托管',
            width: 140,
            render: (_, record) =>
              `${record.max_self_hosted_entry_nodes}/${record.max_self_hosted_exit_nodes}`,
          },
          {
            title: '倍率',
            dataIndex: 'traffic_multiplier',
            width: 100,
            render: (value: number) => `${(value ?? 1).toFixed(2)}x`,
          },
          {
            title: '价格',
            dataIndex: 'price',
            width: 120,
            render: (value?: number | null) =>
              value == null ? (
                <Typography.Text type="secondary">-</Typography.Text>
              ) : (
                `¥${value}`
              ),
          },
          {
            title: '操作',
            width: 100,
            render: (_, record) => (
              <Button
                type="link"
                icon={<EditOutlined />}
                onClick={() => openEditModal(record)}
              >
                编辑
              </Button>
            ),
          },
        ]}
      />

      <Modal
        title={formMode === 'create' ? '创建 VIP 等级' : '编辑 VIP 等级'}
        open={modalOpen}
        onCancel={closeModal}
        onOk={() => void form.submit()}
        okText={formMode === 'create' ? '创建' : '保存'}
        confirmLoading={saving}
        destroyOnClose
      >
        <Form<VipFormValues>
          form={form}
          layout="vertical"
          onFinish={(values) => void handleSubmit(values)}
          preserve={false}
        >
          <Form.Item
            label="等级"
            name="level"
            rules={[{ required: true, message: '请输入等级' }]}
          >
            <InputNumber min={0} precision={0} className="w-full" disabled={formMode === 'edit'} />
          </Form.Item>

          <Form.Item
            label="名称"
            name="name"
            rules={[{ required: true, message: '请输入名称' }]}
          >
            <Input placeholder="例如：专业版" />
          </Form.Item>

          <Form.Item label="描述" name="description">
            <Input.TextArea rows={2} placeholder="可选描述" />
          </Form.Item>

          <Form.Item
            label="流量配额 (GB)"
            name="traffic_quota_gb"
            rules={[{ required: true, message: '请输入流量配额' }]}
          >
            <InputNumber min={0} className="w-full" />
          </Form.Item>

          <Form.Item label="规则数 (-1 表示不限)" name="max_rules" rules={[{ required: true }]}>
            <InputNumber min={-1} precision={0} className="w-full" />
          </Form.Item>

          <Form.Item
            label="带宽限制 Mbps (-1 表示不限)"
            name="max_bandwidth"
            rules={[{ required: true }]}
          >
            <InputNumber min={-1} precision={0} className="w-full" />
          </Form.Item>

          <Form.Item
            label="自托管入口配额"
            name="max_self_hosted_entry_nodes"
            rules={[{ required: true }]}
          >
            <InputNumber min={0} precision={0} className="w-full" />
          </Form.Item>

          <Form.Item
            label="自托管出口配额"
            name="max_self_hosted_exit_nodes"
            rules={[{ required: true }]}
          >
            <InputNumber min={0} precision={0} className="w-full" />
          </Form.Item>

          <Form.Item
            label="流量倍率"
            name="traffic_multiplier"
            rules={[{ required: true, message: '请输入倍率' }]}
          >
            <InputNumber min={0.1} step={0.1} className="w-full" />
          </Form.Item>

          <Form.Item label="价格 (¥)" name="price">
            <InputNumber min={0} step={0.01} className="w-full" />
          </Form.Item>
        </Form>
      </Modal>
    </PageContainer>
  )
}

export default VipLevelManage
