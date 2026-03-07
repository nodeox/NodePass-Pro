import {
  DeleteOutlined,
  PlusOutlined,
  ReloadOutlined,
} from '@ant-design/icons'
import {
  Button,
  DatePicker,
  Form,
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
import { type Dayjs } from 'dayjs'
import { useCallback, useEffect, useMemo, useState } from 'react'

import PageContainer from '../../components/common/PageContainer'
import { usePageTitle } from '../../hooks/usePageTitle'
import { benefitCodeApi, vipApi } from '../../services/api'
import type { BenefitCodeRecord, VipLevelRecord } from '../../types'
import { getErrorMessage } from '../../utils/error'
import { formatDateTime } from '../../utils/format'

type GenerateFormValues = {
  vip_level: number
  duration_days: number
  count: number
  expires_at?: Dayjs
}

const statusOptions = [
  { label: '全部', value: 'all' },
  { label: '未使用', value: 'unused' },
  { label: '已使用', value: 'used' },
]

const renderStatusTag = (status: string) => {
  if (status === 'unused') {
    return <Tag color="green">未使用</Tag>
  }
  if (status === 'used') {
    return <Tag color="default">已使用</Tag>
  }
  return <Tag>{status}</Tag>
}

const BenefitCodeManage = () => {
  usePageTitle('权益码管理')

  const [form] = Form.useForm<GenerateFormValues>()
  const [loading, setLoading] = useState<boolean>(false)
  const [saving, setSaving] = useState<boolean>(false)

  const [levels, setLevels] = useState<VipLevelRecord[]>([])
  const [records, setRecords] = useState<BenefitCodeRecord[]>([])
  const [selectedRowKeys, setSelectedRowKeys] = useState<number[]>([])
  const [modalOpen, setModalOpen] = useState<boolean>(false)

  const [statusFilter, setStatusFilter] = useState<'all' | 'unused' | 'used'>('all')
  const [vipLevelFilter, setVipLevelFilter] = useState<number | undefined>()

  const [page, setPage] = useState<number>(1)
  const [pageSize, setPageSize] = useState<number>(10)
  const [total, setTotal] = useState<number>(0)

  const loadLevels = async (): Promise<void> => {
    try {
      const result = await vipApi.levels()
      setLevels(result.list ?? [])
    } catch (error) {
      message.error(getErrorMessage(error, 'VIP 等级数据加载失败'))
    }
  }

  const loadBenefitCodes = useCallback(
    async (targetPage = page, targetPageSize = pageSize): Promise<void> => {
      setLoading(true)
      try {
        const result = await benefitCodeApi.list({
          page: targetPage,
          pageSize: targetPageSize,
          status: statusFilter === 'all' ? undefined : statusFilter,
          vip_level: vipLevelFilter,
        })
        setRecords(result.list ?? [])
        setTotal(result.total ?? 0)
        setPage(result.page || targetPage)
        setPageSize(result.page_size || targetPageSize)
      } catch (error) {
        message.error(getErrorMessage(error, '权益码列表加载失败'))
      } finally {
        setLoading(false)
      }
    },
    [page, pageSize, statusFilter, vipLevelFilter],
  )

  useEffect(() => {
    void loadLevels()
  }, [])

  useEffect(() => {
    void loadBenefitCodes()
  }, [loadBenefitCodes])

  const levelOptions = useMemo(
    () =>
      levels.map((level) => ({
        label: `Lv.${level.level} ${level.name}`,
        value: level.level,
      })),
    [levels],
  )

  const levelMap = useMemo(() => {
    const map = new Map<number, VipLevelRecord>()
    levels.forEach((level) => map.set(level.level, level))
    return map
  }, [levels])

  const openGenerateModal = (): void => {
    form.setFieldsValue({
      vip_level: levelOptions[0]?.value ?? 0,
      duration_days: 30,
      count: 10,
      expires_at: undefined,
    })
    setModalOpen(true)
  }

  const closeGenerateModal = (): void => {
    setModalOpen(false)
    form.resetFields()
  }

  const handleGenerate = async (values: GenerateFormValues): Promise<void> => {
    setSaving(true)
    try {
      await benefitCodeApi.generate({
        vip_level: values.vip_level,
        duration_days: values.duration_days,
        count: values.count,
        expires_at: values.expires_at ? values.expires_at.toISOString() : undefined,
      })
      message.success('权益码生成成功')
      closeGenerateModal()
      await loadBenefitCodes()
    } catch (error) {
      message.error(getErrorMessage(error, '权益码生成失败'))
    } finally {
      setSaving(false)
    }
  }

  const handleBatchDelete = async (ids: number[]): Promise<void> => {
    if (ids.length === 0) {
      message.warning('请先选择要删除的权益码')
      return
    }

    try {
      const result = await benefitCodeApi.batchDelete(ids)
      message.success(`已删除 ${result.deleted} 条权益码`)
      setSelectedRowKeys([])
      if (records.length === ids.length && page > 1) {
        setPage(page - 1)
        return
      }
      await loadBenefitCodes()
    } catch (error) {
      message.error(getErrorMessage(error, '批量删除失败'))
    }
  }

  return (
    <PageContainer
      title="权益码管理"
      description="支持批量生成、筛选查询和批量删除。"
      extra={
        <Space>
          <Button
            icon={<ReloadOutlined />}
            onClick={() => void loadBenefitCodes()}
            loading={loading}
          >
            刷新
          </Button>
          <Popconfirm
            title={`确定删除已选中的 ${selectedRowKeys.length} 个权益码吗？`}
            okText="删除"
            cancelText="取消"
              onConfirm={() => void handleBatchDelete(selectedRowKeys)}
            disabled={selectedRowKeys.length === 0}
          >
            <Button
              danger
              icon={<DeleteOutlined />}
              disabled={selectedRowKeys.length === 0}
            >
              批量删除
            </Button>
          </Popconfirm>
          <Button type="primary" icon={<PlusOutlined />} onClick={openGenerateModal}>
            批量生成
          </Button>
        </Space>
      }
    >
      <Space wrap style={{ marginBottom: 16 }}>
        <Select
          style={{ minWidth: 160 }}
          value={statusFilter}
          options={statusOptions}
          onChange={(value) => {
            setStatusFilter(value)
            setPage(1)
          }}
        />
        <Select
          allowClear
          placeholder="按 VIP 等级筛选"
          style={{ minWidth: 180 }}
          options={levelOptions}
          value={vipLevelFilter}
          onChange={(value) => {
            setVipLevelFilter(value)
            setPage(1)
          }}
        />
      </Space>

      <Table<BenefitCodeRecord>
        rowKey="id"
        loading={loading}
        dataSource={records}
        rowSelection={{
          selectedRowKeys,
          onChange: (keys) => setSelectedRowKeys(keys as number[]),
        }}
        pagination={{
          current: page,
          pageSize,
          total,
          showSizeChanger: true,
          showTotal: (recordTotal) => `共 ${recordTotal} 条`,
          onChange: (nextPage, nextPageSize) => {
            setPage(nextPage)
            setPageSize(nextPageSize)
          },
        }}
        columns={[
          {
            title: '编码',
            dataIndex: 'code',
            width: 220,
            render: (value: string) => (
              <Typography.Text copyable={{ text: value }}>{value}</Typography.Text>
            ),
          },
          {
            title: 'VIP 等级',
            dataIndex: 'vip_level',
            width: 120,
            render: (value: number) => {
              const level = levelMap.get(value)
              return level ? `Lv.${value} ${level.name}` : `Lv.${value}`
            },
          },
          {
            title: '天数',
            dataIndex: 'duration_days',
            width: 100,
            render: (value: number) => `${value} 天`,
          },
          {
            title: '状态',
            dataIndex: 'status',
            width: 100,
            render: (value: string) => renderStatusTag(value),
          },
          {
            title: '使用者',
            dataIndex: 'used_by',
            width: 110,
            render: (value?: number | null) =>
              value ? `用户 #${value}` : <Typography.Text type="secondary">-</Typography.Text>,
          },
          {
            title: '使用时间',
            dataIndex: 'used_at',
            width: 180,
            render: (value?: string | null) => formatDateTime(value),
          },
          {
            title: '过期时间',
            dataIndex: 'expires_at',
            width: 180,
            render: (value?: string | null) => formatDateTime(value),
          },
          {
            title: '操作',
            width: 100,
            render: (_, record) => (
              <Popconfirm
                title="确定删除该权益码吗？"
                okText="删除"
                cancelText="取消"
                onConfirm={() => void handleBatchDelete([record.id])}
              >
                <Button type="link" danger icon={<DeleteOutlined />}>
                  删除
                </Button>
              </Popconfirm>
            ),
          },
        ]}
      />

      <Modal
        title="批量生成权益码"
        open={modalOpen}
        onCancel={closeGenerateModal}
        onOk={() => void form.submit()}
        okText="生成"
        confirmLoading={saving}
        destroyOnClose
      >
        <Form<GenerateFormValues>
          form={form}
          layout="vertical"
          onFinish={(values) => void handleGenerate(values)}
          preserve={false}
        >
          <Form.Item
            label="VIP 等级"
            name="vip_level"
            rules={[{ required: true, message: '请选择 VIP 等级' }]}
          >
            <Select options={levelOptions} />
          </Form.Item>

          <Form.Item
            label="有效天数"
            name="duration_days"
            rules={[{ required: true, message: '请输入有效天数' }]}
          >
            <InputNumber min={1} precision={0} className="w-full" />
          </Form.Item>

          <Form.Item
            label="生成数量"
            name="count"
            rules={[{ required: true, message: '请输入生成数量' }]}
          >
            <InputNumber min={1} max={1000} precision={0} className="w-full" />
          </Form.Item>

          <Form.Item label="过期时间 (可选)" name="expires_at">
            <DatePicker className="w-full" showTime />
          </Form.Item>
        </Form>
      </Modal>
    </PageContainer>
  )
}

export default BenefitCodeManage
