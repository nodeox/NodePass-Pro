import {
  DeleteOutlined,
  EditOutlined,
  PlusOutlined,
  ReloadOutlined,
} from '@ant-design/icons'
import {
  Button,
  DatePicker,
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
import dayjs, { type Dayjs } from 'dayjs'
import { useCallback, useEffect, useState } from 'react'

import PageContainer from '../../components/common/PageContainer'
import { usePageTitle } from '../../hooks/usePageTitle'
import { announcementApi } from '../../services/api'
import type { AnnouncementRecord } from '../../types'
import { getErrorMessage } from '../../utils/error'
import { formatDateTime } from '../../utils/format'

type AnnouncementFormMode = 'create' | 'edit'

type AnnouncementFormValues = {
  title: string
  content: string
  type: AnnouncementRecord['type']
  is_enabled: boolean
  time_range?: [Dayjs, Dayjs]
}

const typeMap: Record<
  AnnouncementRecord['type'],
  { label: string; color: string }
> = {
  info: { label: '信息', color: 'blue' },
  warning: { label: '警告', color: 'orange' },
  error: { label: '错误', color: 'red' },
  success: { label: '成功', color: 'green' },
}

const Announcements = () => {
  usePageTitle('系统公告')

  const [form] = Form.useForm<AnnouncementFormValues>()
  const [loading, setLoading] = useState<boolean>(false)
  const [saving, setSaving] = useState<boolean>(false)
  const [records, setRecords] = useState<AnnouncementRecord[]>([])
  const [modalOpen, setModalOpen] = useState<boolean>(false)
  const [formMode, setFormMode] = useState<AnnouncementFormMode>('create')
  const [editingRecord, setEditingRecord] = useState<AnnouncementRecord | null>(null)

  const loadAnnouncements = useCallback(async (): Promise<void> => {
    setLoading(true)
    try {
      const result = await announcementApi.list(false)
      setRecords(result.list ?? [])
    } catch (error) {
      message.error(getErrorMessage(error, '公告列表加载失败'))
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    void loadAnnouncements()
  }, [loadAnnouncements])

  const openCreateModal = (): void => {
    setFormMode('create')
    setEditingRecord(null)
    form.setFieldsValue({
      title: '',
      content: '',
      type: 'info',
      is_enabled: true,
      time_range: undefined,
    })
    setModalOpen(true)
  }

  const openEditModal = (record: AnnouncementRecord): void => {
    setFormMode('edit')
    setEditingRecord(record)
    form.setFieldsValue({
      title: record.title,
      content: record.content,
      type: record.type,
      is_enabled: record.is_enabled,
      time_range:
        record.start_time && record.end_time
          ? [dayjs(record.start_time), dayjs(record.end_time)]
          : undefined,
    })
    setModalOpen(true)
  }

  const closeModal = (): void => {
    setModalOpen(false)
    setEditingRecord(null)
    form.resetFields()
  }

  const handleSubmit = async (values: AnnouncementFormValues): Promise<void> => {
    setSaving(true)
    try {
      const payload = {
        title: values.title.trim(),
        content: values.content.trim(),
        type: values.type,
        is_enabled: values.is_enabled,
        start_time: values.time_range?.[0]?.toISOString(),
        end_time: values.time_range?.[1]?.toISOString(),
      }

      if (formMode === 'create') {
        await announcementApi.create(payload)
        message.success('公告创建成功')
      } else if (editingRecord) {
        await announcementApi.update(editingRecord.id, payload)
        message.success('公告更新成功')
      }

      closeModal()
      await loadAnnouncements()
    } catch (error) {
      message.error(getErrorMessage(error, '公告保存失败'))
    } finally {
      setSaving(false)
    }
  }

  const handleDelete = async (id: number): Promise<void> => {
    try {
      await announcementApi.remove(id)
      message.success('公告已删除')
      await loadAnnouncements()
    } catch (error) {
      message.error(getErrorMessage(error, '删除公告失败'))
    }
  }

  const handleToggle = async (
    record: AnnouncementRecord,
    enabled: boolean,
  ): Promise<void> => {
    try {
      await announcementApi.update(record.id, {
        is_enabled: enabled,
      })
      message.success(enabled ? '公告已启用' : '公告已禁用')
      await loadAnnouncements()
    } catch (error) {
      message.error(getErrorMessage(error, '更新公告状态失败'))
    }
  }

  return (
    <PageContainer
      title="系统公告"
      description="管理系统公告和通知显示窗口。"
      extra={
        <Space>
          <Button
            icon={<ReloadOutlined />}
            onClick={() => void loadAnnouncements()}
            loading={loading}
          >
            刷新
          </Button>
          <Button type="primary" icon={<PlusOutlined />} onClick={openCreateModal}>
            创建公告
          </Button>
        </Space>
      }
    >
      <Table<AnnouncementRecord>
        rowKey="id"
        loading={loading}
        dataSource={records}
        pagination={false}
        columns={[
          {
            title: '标题',
            dataIndex: 'title',
            width: 220,
            render: (value: string, record) => (
              <Space direction="vertical" size={2}>
                <Typography.Text strong>{value}</Typography.Text>
                <Typography.Text type="secondary" ellipsis={{ tooltip: record.content }}>
                  {record.content}
                </Typography.Text>
              </Space>
            ),
          },
          {
            title: '类型',
            dataIndex: 'type',
            width: 100,
            render: (value: AnnouncementRecord['type']) => (
              <Tag color={typeMap[value]?.color ?? 'default'}>
                {typeMap[value]?.label ?? value}
              </Tag>
            ),
          },
          {
            title: '启用状态',
            dataIndex: 'is_enabled',
            width: 110,
            render: (enabled: boolean, record) => (
              <Switch
                checked={enabled}
                onChange={(checked) => void handleToggle(record, checked)}
              />
            ),
          },
          {
            title: '时间范围',
            width: 260,
            render: (_, record) =>
              `${formatDateTime(record.start_time)} ~ ${formatDateTime(record.end_time)}`,
          },
          {
            title: '操作',
            width: 140,
            render: (_, record) => (
              <Space>
                <Button
                  type="link"
                  icon={<EditOutlined />}
                  onClick={() => openEditModal(record)}
                >
                  编辑
                </Button>
                <Popconfirm
                  title="确定删除该公告吗？"
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
        title={formMode === 'create' ? '创建公告' : '编辑公告'}
        open={modalOpen}
        onCancel={closeModal}
        onOk={() => void form.submit()}
        okText={formMode === 'create' ? '创建' : '保存'}
        confirmLoading={saving}
        destroyOnClose
      >
        <Form<AnnouncementFormValues>
          form={form}
          layout="vertical"
          onFinish={(values) => void handleSubmit(values)}
          preserve={false}
        >
          <Form.Item
            label="标题"
            name="title"
            rules={[{ required: true, message: '请输入标题' }]}
          >
            <Input placeholder="请输入公告标题" />
          </Form.Item>

          <Form.Item
            label="内容"
            name="content"
            rules={[{ required: true, message: '请输入公告内容' }]}
          >
            <Input.TextArea rows={4} placeholder="请输入公告内容" />
          </Form.Item>

          <Form.Item label="类型" name="type" rules={[{ required: true }]}>
            <Select
              options={[
                { label: '信息', value: 'info' },
                { label: '警告', value: 'warning' },
                { label: '错误', value: 'error' },
                { label: '成功', value: 'success' },
              ]}
            />
          </Form.Item>

          <Form.Item label="启用" name="is_enabled" valuePropName="checked">
            <Switch />
          </Form.Item>

          <Form.Item label="开始/结束时间" name="time_range">
            <DatePicker.RangePicker className="w-full" showTime />
          </Form.Item>
        </Form>
      </Modal>
    </PageContainer>
  )
}

export default Announcements
