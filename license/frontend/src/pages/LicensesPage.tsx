import {
  Button,
  DatePicker,
  Descriptions,
  Drawer,
  Form,
  Input,
  InputNumber,
  Popconfirm,
  Select,
  Space,
  Spin,
  Switch,
  Table,
  Tabs,
  Tag,
  Typography,
  message
} from 'antd'
import dayjs, { type Dayjs } from 'dayjs'
import { useEffect, useMemo, useState, type Key } from 'react'
import type { SorterResult, TablePaginationConfig, TableRowSelection } from 'antd/es/table/interface'
import type { License, LicenseActivation, LicenseListParams, LicensePlan, UpdateLicensePayload } from '../types/api'
import { licenseApi, planApi } from '../utils/api'
import { extractErrorMessage } from '../utils/request'

type EditFormValues = {
  key: string
  plan_id: number
  customer: string
  status: 'active' | 'revoked' | 'expired'
  expires_at?: Dayjs
  clear_expires_at?: boolean
  max_machines?: number
  clear_max_machines?: boolean
  metadata_json?: string
  note?: string
}

type SearchFormValues = {
  customer?: string
  status?: 'active' | 'revoked' | 'expired'
  plan_id?: number
  expire_range?: [Dayjs, Dayjs]
}

type LicenseQuery = Omit<LicenseListParams, 'page' | 'page_size'>
type SortField = NonNullable<LicenseListParams['sort_by']>
type SortDirection = NonNullable<LicenseListParams['sort_order']>

type InitialQueryState = {
  page: number
  pageSize: number
  query: LicenseQuery
  searchFormValues: SearchFormValues
}

const DEFAULT_PAGE_SIZE = 20
const MAX_PAGE_SIZE = 200
const SORTABLE_FIELDS: SortField[] = ['created_at', 'expires_at', 'status']

const parsePositiveInt = (value: string | null, fallback: number, max = MAX_PAGE_SIZE) => {
  if (!value) return fallback
  const parsed = Number.parseInt(value, 10)
  if (!Number.isFinite(parsed) || parsed <= 0) return fallback
  return Math.min(parsed, max)
}

const isStatusValue = (value?: string | null): value is SearchFormValues['status'] =>
  value === 'active' || value === 'revoked' || value === 'expired'

const isSortField = (value?: string | null): value is SortField =>
  value === 'created_at' || value === 'expires_at' || value === 'status'

const isSortDirection = (value?: string | null): value is SortDirection => value === 'asc' || value === 'desc'

const parseInitialQueryState = (): InitialQueryState => {
  const params = new URLSearchParams(window.location.search)

  const customer = params.get('customer')?.trim() || undefined
  const rawStatus = params.get('status')?.trim()
  const status = isStatusValue(rawStatus) ? rawStatus : undefined
  const rawPlanID = params.get('plan_id')?.trim()
  const parsedPlanID = rawPlanID ? Number.parseInt(rawPlanID, 10) : NaN
  const planID = Number.isInteger(parsedPlanID) && parsedPlanID > 0 ? parsedPlanID : undefined

  const rawExpireFrom = params.get('expire_from')?.trim()
  const rawExpireTo = params.get('expire_to')?.trim()
  const expireFrom = rawExpireFrom && dayjs(rawExpireFrom).isValid() ? dayjs(rawExpireFrom).startOf('day') : undefined
  const expireTo = rawExpireTo && dayjs(rawExpireTo).isValid() ? dayjs(rawExpireTo).endOf('day') : undefined

  const rawSortBy = params.get('sort_by')?.trim()
  const rawSortOrder = params.get('sort_order')?.trim()
  const sortBy = isSortField(rawSortBy) ? rawSortBy : undefined
  const sortOrder = isSortDirection(rawSortOrder) ? rawSortOrder : undefined

  const query: LicenseQuery = {
    customer,
    status,
    plan_id: planID,
    expire_from: expireFrom?.toISOString(),
    expire_to: expireTo?.toISOString()
  }
  if (sortBy && sortOrder) {
    query.sort_by = sortBy
    query.sort_order = sortOrder
  }

  const searchFormValues: SearchFormValues = {
    customer,
    status,
    plan_id: planID
  }
  if (expireFrom && expireTo) {
    searchFormValues.expire_range = [expireFrom, expireTo]
  }

  return {
    page: parsePositiveInt(params.get('page'), 1, Number.MAX_SAFE_INTEGER),
    pageSize: parsePositiveInt(params.get('page_size'), DEFAULT_PAGE_SIZE),
    query,
    searchFormValues
  }
}

const toAntSortOrder = (query: LicenseQuery, field: SortField): 'ascend' | 'descend' | undefined => {
  if (query.sort_by !== field) return undefined
  return query.sort_order === 'asc' ? 'ascend' : query.sort_order === 'desc' ? 'descend' : undefined
}

const resolveSortQuery = (
  sorter: SorterResult<License> | SorterResult<License>[]
): Pick<LicenseQuery, 'sort_by' | 'sort_order'> => {
  const target = Array.isArray(sorter) ? sorter.find((item) => !!item.order) : sorter
  if (!target || !target.order || typeof target.field !== 'string') {
    return {}
  }
  if (!SORTABLE_FIELDS.includes(target.field as SortField)) {
    return {}
  }
  return {
    sort_by: target.field as SortField,
    sort_order: target.order === 'ascend' ? 'asc' : 'desc'
  }
}

const syncQueryToURL = (page: number, pageSize: number, query: LicenseQuery) => {
  const params = new URLSearchParams()
  params.set('page', String(page))
  params.set('page_size', String(pageSize))
  if (query.customer) params.set('customer', query.customer)
  if (query.status) params.set('status', query.status)
  if (query.plan_id) params.set('plan_id', String(query.plan_id))
  if (query.expire_from) params.set('expire_from', query.expire_from)
  if (query.expire_to) params.set('expire_to', query.expire_to)
  if (query.sort_by) params.set('sort_by', query.sort_by)
  if (query.sort_order) params.set('sort_order', query.sort_order)
  const search = params.toString()
  const target = search ? `${window.location.pathname}?${search}` : window.location.pathname
  window.history.replaceState(null, '', target)
}

export default function LicensesPage() {
  const initialQueryState = useMemo(parseInitialQueryState, [])

  const [items, setItems] = useState<License[]>([])
  const [plans, setPlans] = useState<LicensePlan[]>([])
  const [loading, setLoading] = useState(false)
  const [page, setPage] = useState(initialQueryState.page)
  const [pageSize, setPageSize] = useState(initialQueryState.pageSize)
  const [total, setTotal] = useState(0)
  const [query, setQuery] = useState<LicenseQuery>(initialQueryState.query)
  const [selectedRowKeys, setSelectedRowKeys] = useState<Key[]>([])
  const [batchPlanID, setBatchPlanID] = useState<number | undefined>()
  const [batchActionLoading, setBatchActionLoading] = useState(false)

  const [generateOpen, setGenerateOpen] = useState(false)
  const [detailOpen, setDetailOpen] = useState(false)
  const [detailLoading, setDetailLoading] = useState(false)
  const [detailLicense, setDetailLicense] = useState<License | null>(null)
  const [detailTab, setDetailTab] = useState<'overview' | 'edit' | 'activations'>('overview')
  const [activationLoading, setActivationLoading] = useState(false)
  const [activationItems, setActivationItems] = useState<LicenseActivation[]>([])

  const [generateForm] = Form.useForm()
  const [editForm] = Form.useForm<EditFormValues>()
  const [searchForm] = Form.useForm<SearchFormValues>()

  const planOptions = useMemo(
    () => plans.map((p) => ({ label: `${p.name} (${p.code})`, value: p.id })),
    [plans]
  )
  const selectedLicenseIDs = useMemo(
    () =>
      selectedRowKeys
        .map((key) => Number(key))
        .filter((id): id is number => Number.isInteger(id) && id > 0),
    [selectedRowKeys]
  )
  const hasSelectedLicenses = selectedLicenseIDs.length > 0

  const loadPlans = async () => {
    try {
      const data = await planApi.list()
      setPlans(data)
    } catch (err) {
      message.error(extractErrorMessage(err))
    }
  }

  const loadLicenses = async (nextPage = page, nextPageSize = pageSize, nextQuery = query) => {
    setLoading(true)
    try {
      const result = await licenseApi.list({
        page: nextPage,
        page_size: nextPageSize,
        ...nextQuery
      })
      setItems(result.items)
      setTotal(result.total)
    } catch (err) {
      message.error(extractErrorMessage(err))
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    void loadPlans()
  }, [])

  useEffect(() => {
    searchForm.setFieldsValue(initialQueryState.searchFormValues)
  }, [initialQueryState.searchFormValues, searchForm])

  useEffect(() => {
    void loadLicenses(page, pageSize, query)
  }, [page, pageSize, query])

  useEffect(() => {
    syncQueryToURL(page, pageSize, query)
  }, [page, pageSize, query])

  const buildQueryFromSearch = (values: SearchFormValues): LicenseQuery => {
    const customer = values.customer?.trim()
    const expireFrom = values.expire_range?.[0]
    const expireTo = values.expire_range?.[1]

    return {
      customer: customer || undefined,
      status: values.status || undefined,
      plan_id: values.plan_id || undefined,
      expire_from: expireFrom ? expireFrom.startOf('day').toISOString() : undefined,
      expire_to: expireTo ? expireTo.endOf('day').toISOString() : undefined
    }
  }

  const onSearch = async () => {
    const values = searchForm.getFieldsValue()
    setSelectedRowKeys([])
    setPage(1)
    setQuery((prev) => ({
      ...buildQueryFromSearch(values),
      sort_by: prev.sort_by,
      sort_order: prev.sort_order
    }))
  }

  const onResetSearch = () => {
    searchForm.resetFields()
    setSelectedRowKeys([])
    setPage(1)
    setQuery({})
  }

  const onGenerate = async () => {
    try {
      const values = await generateForm.validateFields()
      await licenseApi.generate(values)
      message.success('授权码生成成功')
      setGenerateOpen(false)
      generateForm.resetFields()
      await loadLicenses(page, pageSize, query)
    } catch (err) {
      if (err instanceof Error) {
        message.error(err.message)
      }
    }
  }

  const fillEditForm = (record: License) => {
    editForm.setFieldsValue({
      key: record.key,
      plan_id: record.plan_id,
      customer: record.customer,
      status: (record.status as EditFormValues['status']) || 'active',
      expires_at: record.expires_at ? dayjs(record.expires_at) : undefined,
      clear_expires_at: false,
      max_machines: record.max_machines ?? undefined,
      clear_max_machines: false,
      metadata_json: record.metadata_json ?? '',
      note: record.note ?? ''
    })
  }

  const loadActivations = async (licenseId: number) => {
    setActivationLoading(true)
    try {
      const data = await licenseApi.listActivations(licenseId)
      setActivationItems(data)
    } catch (err) {
      message.error(extractErrorMessage(err))
    } finally {
      setActivationLoading(false)
    }
  }

  const refreshDetail = async (licenseId: number) => {
    setDetailLoading(true)
    try {
      const [fresh, activations] = await Promise.all([licenseApi.get(licenseId), licenseApi.listActivations(licenseId)])
      setDetailLicense(fresh)
      setActivationItems(activations)
      fillEditForm(fresh)
    } catch (err) {
      message.error(extractErrorMessage(err))
    } finally {
      setDetailLoading(false)
    }
  }

  const openDetailDrawer = async (record: License) => {
    setDetailOpen(true)
    setDetailTab('overview')
    await refreshDetail(record.id)
  }

  const onUpdateLicense = async () => {
    if (!detailLicense) return

    try {
      const values = await editForm.validateFields()
      const payload: UpdateLicensePayload = {
        key: values.key?.trim(),
        plan_id: values.plan_id,
        customer: values.customer?.trim(),
        status: values.status,
        clear_expires_at: !!values.clear_expires_at,
        clear_max_machines: !!values.clear_max_machines,
        metadata_json: values.metadata_json?.trim() ?? '',
        note: values.note?.trim() ?? ''
      }

      if (!values.clear_expires_at && values.expires_at) {
        payload.expires_at = values.expires_at.toISOString()
      }
      if (!values.clear_max_machines && typeof values.max_machines === 'number') {
        payload.max_machines = values.max_machines
      }

      await licenseApi.update(detailLicense.id, payload)
      message.success('授权编辑成功')
      await refreshDetail(detailLicense.id)
      await loadLicenses(page, pageSize, query)
    } catch (err) {
      message.error(extractErrorMessage(err))
    }
  }

  const unbindActivation = async (activationId: number) => {
    if (!detailLicense) return
    try {
      await licenseApi.unbindActivation(detailLicense.id, activationId)
      message.success('解绑成功')
      await loadActivations(detailLicense.id)
      await loadLicenses(page, pageSize, query)
    } catch (err) {
      message.error(extractErrorMessage(err))
    }
  }

  const clearAllActivations = async () => {
    if (!detailLicense) return
    try {
      const result = await licenseApi.clearActivations(detailLicense.id)
      message.success(`已清空 ${result.cleared_count} 条绑定`)
      await loadActivations(detailLicense.id)
      await loadLicenses(page, pageSize, query)
    } catch (err) {
      message.error(extractErrorMessage(err))
    }
  }

  const changeStatus = async (id: number, status: 'revoke' | 'restore') => {
    try {
      if (status === 'revoke') {
        await licenseApi.revoke(id)
      } else {
        await licenseApi.restore(id)
      }
      await loadLicenses(page, pageSize, query)
      if (detailOpen && detailLicense?.id === id) {
        await refreshDetail(id)
      }
      message.success('操作成功')
    } catch (err) {
      message.error(extractErrorMessage(err))
    }
  }

  const refreshAfterBatchAction = async (affectedIDs: number[]) => {
    await loadLicenses(page, pageSize, query)
    setSelectedRowKeys([])
    if (detailOpen && detailLicense && affectedIDs.includes(detailLicense.id)) {
      await refreshDetail(detailLicense.id)
    }
  }

  const onBatchRevoke = async () => {
    if (!hasSelectedLicenses) return
    const ids = [...selectedLicenseIDs]
    setBatchActionLoading(true)
    try {
      const result = await licenseApi.batchRevoke(ids)
      message.success(`批量吊销成功（${result.revoked_count ?? ids.length} 条）`)
      await refreshAfterBatchAction(ids)
    } catch (err) {
      message.error(extractErrorMessage(err))
    } finally {
      setBatchActionLoading(false)
    }
  }

  const onBatchRestore = async () => {
    if (!hasSelectedLicenses) return
    const ids = [...selectedLicenseIDs]
    setBatchActionLoading(true)
    try {
      const result = await licenseApi.batchRestore(ids)
      message.success(`批量恢复成功（${result.restored_count ?? ids.length} 条）`)
      await refreshAfterBatchAction(ids)
    } catch (err) {
      message.error(extractErrorMessage(err))
    } finally {
      setBatchActionLoading(false)
    }
  }

  const onBatchChangePlan = async () => {
    if (!hasSelectedLicenses || !batchPlanID) return
    const ids = [...selectedLicenseIDs]
    setBatchActionLoading(true)
    try {
      const result = await licenseApi.batchUpdate(ids, { plan_id: batchPlanID })
      message.success(`批量改套餐成功（${result.updated_count ?? ids.length} 条）`)
      await refreshAfterBatchAction(ids)
    } catch (err) {
      message.error(extractErrorMessage(err))
    } finally {
      setBatchActionLoading(false)
    }
  }

  const closeDetailDrawer = () => {
    setDetailOpen(false)
    setDetailLicense(null)
    setActivationItems([])
    editForm.resetFields()
    setDetailTab('overview')
  }

  const removeLicense = async (id: number) => {
    try {
      await licenseApi.remove(id)
      message.success('授权已删除')
      if (detailOpen && detailLicense?.id === id) {
        closeDetailDrawer()
      }
      setSelectedRowKeys((prev) => prev.filter((key) => Number(key) !== id))
      await loadLicenses(page, pageSize, query)
    } catch (err) {
      message.error(extractErrorMessage(err))
    }
  }

  const onBatchDelete = async () => {
    if (!hasSelectedLicenses) return
    const ids = [...selectedLicenseIDs]
    setBatchActionLoading(true)
    try {
      const result = await licenseApi.batchDelete(ids)
      message.success(`批量删除成功（${result.deleted_count ?? ids.length} 条）`)
      if (detailOpen && detailLicense && ids.includes(detailLicense.id)) {
        closeDetailDrawer()
      }
      setSelectedRowKeys([])
      await loadLicenses(page, pageSize, query)
    } catch (err) {
      message.error(extractErrorMessage(err))
    } finally {
      setBatchActionLoading(false)
    }
  }

  const rowSelection: TableRowSelection<License> = {
    selectedRowKeys,
    preserveSelectedRowKeys: true,
    onChange: (keys) => setSelectedRowKeys(keys)
  }

  const onTableChange = (
    pagination: TablePaginationConfig,
    _filters: Record<string, unknown>,
    sorter: SorterResult<License> | SorterResult<License>[],
    extra: { action: 'paginate' | 'sort' | 'filter' }
  ) => {
    const nextPageSize = pagination.pageSize ?? pageSize
    const nextPage = extra.action === 'sort' ? 1 : pagination.current ?? 1

    setPage(nextPage)
    setPageSize(nextPageSize)
    if (extra.action !== 'sort') {
      return
    }

    const sortQuery = resolveSortQuery(sorter)
    setQuery((prev) => {
      const next: LicenseQuery = { ...prev }
      if (sortQuery.sort_by && sortQuery.sort_order) {
        next.sort_by = sortQuery.sort_by
        next.sort_order = sortQuery.sort_order
      } else {
        delete next.sort_by
        delete next.sort_order
      }
      if (next.sort_by === prev.sort_by && next.sort_order === prev.sort_order) {
        return prev
      }
      return next
    })
  }

  return (
    <>
      <Form form={searchForm} layout="inline" onFinish={onSearch} style={{ marginBottom: 16, rowGap: 12 }}>
        <Form.Item label="客户" name="customer">
          <Input allowClear placeholder="客户名称" style={{ width: 180 }} />
        </Form.Item>
        <Form.Item label="状态" name="status">
          <Select
            allowClear
            placeholder="全部"
            style={{ width: 130 }}
            options={[
              { label: 'active', value: 'active' },
              { label: 'revoked', value: 'revoked' },
              { label: 'expired', value: 'expired' }
            ]}
          />
        </Form.Item>
        <Form.Item label="套餐" name="plan_id">
          <Select allowClear placeholder="全部套餐" options={planOptions} style={{ width: 220 }} />
        </Form.Item>
        <Form.Item label="到期区间" name="expire_range">
          <DatePicker.RangePicker />
        </Form.Item>
        <Form.Item>
          <Space>
            <Button type="primary" htmlType="submit">
              查询
            </Button>
            <Button onClick={onResetSearch}>重置</Button>
            <Button type="dashed" onClick={() => setGenerateOpen(true)}>
              生成授权码
            </Button>
          </Space>
        </Form.Item>
      </Form>

      <Space style={{ width: '100%', justifyContent: 'space-between', marginBottom: 12 }} wrap>
        <Space wrap>
          <Typography.Text type="secondary">已选 {selectedLicenseIDs.length} 项</Typography.Text>
          <Popconfirm
            title={`确认批量吊销 ${selectedLicenseIDs.length} 个授权？`}
            onConfirm={onBatchRevoke}
            okText="确认"
            cancelText="取消"
            disabled={!hasSelectedLicenses}
          >
            <Button danger disabled={!hasSelectedLicenses} loading={batchActionLoading}>
              批量吊销
            </Button>
          </Popconfirm>
          <Popconfirm
            title={`确认批量恢复 ${selectedLicenseIDs.length} 个授权？`}
            onConfirm={onBatchRestore}
            okText="确认"
            cancelText="取消"
            disabled={!hasSelectedLicenses}
          >
            <Button disabled={!hasSelectedLicenses} loading={batchActionLoading}>
              批量恢复
            </Button>
          </Popconfirm>
          <Popconfirm
            title={`确认永久删除 ${selectedLicenseIDs.length} 个授权？`}
            description="删除后不可恢复，并会清理设备绑定。"
            onConfirm={onBatchDelete}
            okText="确认"
            cancelText="取消"
            disabled={!hasSelectedLicenses}
          >
            <Button danger disabled={!hasSelectedLicenses} loading={batchActionLoading}>
              批量删除
            </Button>
          </Popconfirm>
        </Space>

        <Space wrap>
          <Select
            allowClear
            placeholder="选择目标套餐"
            style={{ width: 260 }}
            options={planOptions}
            value={batchPlanID}
            onChange={(value) => setBatchPlanID(value as number | undefined)}
          />
          <Popconfirm
            title={`确认将 ${selectedLicenseIDs.length} 个授权改为所选套餐？`}
            onConfirm={onBatchChangePlan}
            okText="确认"
            cancelText="取消"
            disabled={!hasSelectedLicenses || !batchPlanID}
          >
            <Button type="primary" disabled={!hasSelectedLicenses || !batchPlanID} loading={batchActionLoading}>
              批量改套餐
            </Button>
          </Popconfirm>
          <Button onClick={() => setSelectedRowKeys([])} disabled={!hasSelectedLicenses}>
            清空选择
          </Button>
        </Space>
      </Space>

      <Table
        rowKey="id"
        rowSelection={rowSelection}
        loading={loading}
        dataSource={items}
        scroll={{ x: 1400 }}
        onChange={onTableChange}
        pagination={{
          current: page,
          pageSize,
          total,
          showSizeChanger: true,
          showTotal: (value) => `共 ${value} 条`,
          pageSizeOptions: ['10', '20', '50', '100']
        }}
        columns={[
          { title: '授权码', dataIndex: 'key', width: 260 },
          { title: '客户', dataIndex: 'customer', width: 180 },
          {
            title: '套餐',
            width: 180,
            render: (_, record) => `${record.plan?.name ?? '-'} (${record.plan?.code ?? '-'})`
          },
          {
            title: '状态',
            dataIndex: 'status',
            width: 120,
            sorter: true,
            sortOrder: toAntSortOrder(query, 'status'),
            render: (value: string) => {
              const color = value === 'active' ? 'green' : value === 'revoked' ? 'red' : 'orange'
              return <Tag color={color}>{value}</Tag>
            }
          },
          {
            title: '到期时间',
            dataIndex: 'expires_at',
            width: 180,
            sorter: true,
            sortOrder: toAntSortOrder(query, 'expires_at'),
            render: (value?: string) => (value ? dayjs(value).format('YYYY-MM-DD HH:mm') : '-')
          },
          {
            title: '创建时间',
            dataIndex: 'created_at',
            width: 180,
            sorter: true,
            sortOrder: toAntSortOrder(query, 'created_at'),
            render: (value: string) => dayjs(value).format('YYYY-MM-DD HH:mm')
          },
          {
            title: '设备上限',
            width: 120,
            render: (_, record) => record.max_machines ?? record.plan?.max_machines ?? '-'
          },
          {
            title: '操作',
            fixed: 'right',
            width: 320,
            render: (_, record) => (
              <Space>
                <Button size="small" type="primary" ghost onClick={() => openDetailDrawer(record)}>
                  详情
                </Button>
                {record.status === 'active' ? (
                  <Button danger size="small" onClick={() => changeStatus(record.id, 'revoke')}>
                    吊销
                  </Button>
                ) : (
                  <Button size="small" onClick={() => changeStatus(record.id, 'restore')}>
                    恢复
                  </Button>
                )}
                <Popconfirm
                  title="确认永久删除该授权？"
                  description="删除后不可恢复，并会清理设备绑定。"
                  onConfirm={() => removeLicense(record.id)}
                  okText="确认"
                  cancelText="取消"
                >
                  <Button danger size="small">
                    删除
                  </Button>
                </Popconfirm>
              </Space>
            )
          }
        ]}
      />

      <Drawer
        open={generateOpen}
        title="批量生成授权码"
        onClose={() => setGenerateOpen(false)}
        width={480}
        extra={
          <Button type="primary" onClick={onGenerate}>
            提交
          </Button>
        }
      >
        <Form form={generateForm} layout="vertical" initialValues={{ count: 1, expire_days: 365, max_machines: 3 }}>
          <Form.Item label="客户名称" name="customer" rules={[{ required: true }]}>
            <Input placeholder="客户名" />
          </Form.Item>
          <Form.Item label="套餐" name="plan_id" rules={[{ required: true }]}>
            <Select options={planOptions} placeholder="选择套餐" />
          </Form.Item>
          <Form.Item label="数量" name="count" rules={[{ required: true }]}>
            <InputNumber min={1} max={500} style={{ width: '100%' }} />
          </Form.Item>
          <Form.Item label="有效期(天)" name="expire_days">
            <InputNumber min={1} style={{ width: '100%' }} />
          </Form.Item>
          <Form.Item label="设备上限" name="max_machines">
            <InputNumber min={1} style={{ width: '100%' }} />
          </Form.Item>
          <Form.Item label="备注" name="note">
            <Input.TextArea rows={3} />
          </Form.Item>
        </Form>
      </Drawer>

      <Drawer
        open={detailOpen}
        title={`授权详情 #${detailLicense?.id ?? ''}`}
        onClose={closeDetailDrawer}
        width={920}
        destroyOnClose
      >
        <Spin spinning={detailLoading}>
          {detailLicense ? (
            <Tabs
              activeKey={detailTab}
              onChange={(key) => setDetailTab(key as 'overview' | 'edit' | 'activations')}
              items={[
                {
                  key: 'overview',
                  label: '基础信息',
                  children: (
                    <Space direction="vertical" style={{ width: '100%' }} size={16}>
                      <Descriptions bordered column={2} size="small">
                        <Descriptions.Item label="授权码" span={2}>
                          <Typography.Text code>{detailLicense.key}</Typography.Text>
                        </Descriptions.Item>
                        <Descriptions.Item label="客户">{detailLicense.customer}</Descriptions.Item>
                        <Descriptions.Item label="状态">
                          <Tag color={detailLicense.status === 'active' ? 'green' : detailLicense.status === 'revoked' ? 'red' : 'orange'}>
                            {detailLicense.status}
                          </Tag>
                        </Descriptions.Item>
                        <Descriptions.Item label="套餐">
                          {detailLicense.plan?.name ?? '-'} ({detailLicense.plan?.code ?? '-'})
                        </Descriptions.Item>
                        <Descriptions.Item label="设备上限">
                          {detailLicense.max_machines ?? detailLicense.plan?.max_machines ?? '-'}
                        </Descriptions.Item>
                        <Descriptions.Item label="到期时间">
                          {detailLicense.expires_at ? dayjs(detailLicense.expires_at).format('YYYY-MM-DD HH:mm:ss') : '-'}
                        </Descriptions.Item>
                        <Descriptions.Item label="创建时间">
                          {dayjs(detailLicense.created_at).format('YYYY-MM-DD HH:mm:ss')}
                        </Descriptions.Item>
                        <Descriptions.Item label="更新时间">
                          {dayjs(detailLicense.updated_at).format('YYYY-MM-DD HH:mm:ss')}
                        </Descriptions.Item>
                        <Descriptions.Item label="元数据" span={2}>
                          <Typography.Text>{detailLicense.metadata_json || '-'}</Typography.Text>
                        </Descriptions.Item>
                        <Descriptions.Item label="备注" span={2}>
                          <Typography.Text>{detailLicense.note || '-'}</Typography.Text>
                        </Descriptions.Item>
                      </Descriptions>
                    </Space>
                  )
                },
                {
                  key: 'edit',
                  label: '编辑',
                  children: (
                    <Space direction="vertical" style={{ width: '100%' }} size={12}>
                      <Form form={editForm} layout="vertical">
                        <Form.Item label="授权码" name="key" rules={[{ required: true, message: '请输入授权码' }]}>
                          <Input placeholder="例如 NP-XXXX-XXXX-XXXX" />
                        </Form.Item>
                        <Form.Item label="套餐" name="plan_id" rules={[{ required: true, message: '请选择套餐' }]}>
                          <Select options={planOptions} placeholder="选择套餐" />
                        </Form.Item>
                        <Form.Item label="客户名称" name="customer" rules={[{ required: true, message: '请输入客户名称' }]}>
                          <Input />
                        </Form.Item>
                        <Form.Item label="状态" name="status" rules={[{ required: true, message: '请选择状态' }]}>
                          <Select
                            options={[
                              { label: 'active', value: 'active' },
                              { label: 'revoked', value: 'revoked' },
                              { label: 'expired', value: 'expired' }
                            ]}
                          />
                        </Form.Item>
                        <Form.Item label="到期时间" name="expires_at">
                          <DatePicker showTime style={{ width: '100%' }} />
                        </Form.Item>
                        <Form.Item label="清空到期时间" name="clear_expires_at" valuePropName="checked">
                          <Switch />
                        </Form.Item>
                        <Form.Item label="设备上限" name="max_machines">
                          <InputNumber min={1} style={{ width: '100%' }} />
                        </Form.Item>
                        <Form.Item label="清空设备上限" name="clear_max_machines" valuePropName="checked">
                          <Switch />
                        </Form.Item>
                        <Form.Item label="元数据(JSON)" name="metadata_json">
                          <Input.TextArea rows={3} placeholder='例如 {"region":"cn"}' />
                        </Form.Item>
                        <Form.Item label="备注" name="note">
                          <Input.TextArea rows={3} />
                        </Form.Item>
                      </Form>
                      <Space>
                        <Button type="primary" onClick={onUpdateLicense}>
                          保存修改
                        </Button>
                        <Button onClick={() => fillEditForm(detailLicense)}>重置</Button>
                      </Space>
                    </Space>
                  )
                },
                {
                  key: 'activations',
                  label: `设备绑定 (${activationItems.length})`,
                  children: (
                    <Space direction="vertical" style={{ width: '100%' }} size={12}>
                      <Space>
                        <Popconfirm
                          title="确认清空该授权下的全部绑定？"
                          onConfirm={clearAllActivations}
                          okText="确认"
                          cancelText="取消"
                          disabled={activationItems.length === 0}
                        >
                          <Button danger disabled={activationItems.length === 0}>
                            清空绑定
                          </Button>
                        </Popconfirm>
                        <Button onClick={() => loadActivations(detailLicense.id)}>刷新</Button>
                      </Space>
                      <Table
                        rowKey="id"
                        loading={activationLoading}
                        dataSource={activationItems}
                        pagination={false}
                        scroll={{ x: 960 }}
                        columns={[
                          { title: '机器ID', dataIndex: 'machine_id', width: 220 },
                          { title: '主机名', dataIndex: 'hostname', width: 160, render: (value?: string) => value || '-' },
                          { title: 'IP', dataIndex: 'ip_address', width: 140, render: (value?: string) => value || '-' },
                          {
                            title: '最后活跃',
                            dataIndex: 'last_seen_at',
                            width: 180,
                            render: (value: string) => dayjs(value).format('YYYY-MM-DD HH:mm:ss')
                          },
                          {
                            title: '创建时间',
                            dataIndex: 'created_at',
                            width: 180,
                            render: (value: string) => dayjs(value).format('YYYY-MM-DD HH:mm:ss')
                          },
                          {
                            title: '操作',
                            fixed: 'right',
                            width: 120,
                            render: (_, record) => (
                              <Popconfirm
                                title="确认解绑该设备？"
                                onConfirm={() => unbindActivation(record.id)}
                                okText="确认"
                                cancelText="取消"
                              >
                                <Button danger size="small">
                                  解绑
                                </Button>
                              </Popconfirm>
                            )
                          }
                        ]}
                      />
                    </Space>
                  )
                }
              ]}
            />
          ) : null}
        </Spin>
      </Drawer>
    </>
  )
}
