import { ReloadOutlined } from '@ant-design/icons'
import {
  Button,
  DatePicker,
  Select,
  Space,
  Table,
  Typography,
  message,
} from 'antd'
import type { Dayjs } from 'dayjs'
import { useCallback, useEffect, useMemo, useState } from 'react'

import PageContainer from '../../components/common/PageContainer'
import { usePageTitle } from '../../hooks/usePageTitle'
import { auditApi } from '../../services/api'
import type { AuditLogRecord } from '../../types'
import { getErrorMessage } from '../../utils/error'
import { formatDateTime } from '../../utils/format'

const parseDetails = (raw?: string | null): string => {
  if (!raw) {
    return '-'
  }

  try {
    const parsed = JSON.parse(raw)
    return JSON.stringify(parsed, null, 2)
  } catch (_error) {
    return raw
  }
}

const AuditLogs = () => {
  usePageTitle('审计日志')

  const [loading, setLoading] = useState<boolean>(false)
  const [records, setRecords] = useState<AuditLogRecord[]>([])

  const [userFilter, setUserFilter] = useState<number | undefined>()
  const [actionFilter, setActionFilter] = useState<string | undefined>()
  const [timeRange, setTimeRange] = useState<[Dayjs, Dayjs] | null>(null)

  const [page, setPage] = useState<number>(1)
  const [pageSize, setPageSize] = useState<number>(10)
  const [total, setTotal] = useState<number>(0)

  const loadLogs = useCallback(
    async (targetPage = page, targetPageSize = pageSize): Promise<void> => {
      setLoading(true)
      try {
        const result = await auditApi.list({
          page: targetPage,
          pageSize: targetPageSize,
          user_id: userFilter,
          action: actionFilter,
          start_time: timeRange?.[0]?.toISOString(),
          end_time: timeRange?.[1]?.toISOString(),
        })
        setRecords(result.list ?? [])
        setTotal(result.total ?? 0)
        setPage(result.page || targetPage)
        setPageSize(result.page_size || targetPageSize)
      } catch (error) {
        message.error(getErrorMessage(error, '审计日志加载失败'))
      } finally {
        setLoading(false)
      }
    },
    [actionFilter, page, pageSize, timeRange, userFilter],
  )

  useEffect(() => {
    void loadLogs()
  }, [loadLogs])

  const userOptions = useMemo(() => {
    const users = new Map<number, string>()
    records.forEach((item) => {
      if (!item.user_id) {
        return
      }
      const label = item.user?.username
        ? `${item.user.username} (#${item.user_id})`
        : `用户 #${item.user_id}`
      users.set(item.user_id, label)
    })
    return Array.from(users.entries()).map(([value, label]) => ({
      value,
      label,
    }))
  }, [records])

  const actionOptions = useMemo(() => {
    const actions = Array.from(
      new Set(records.map((item) => item.action).filter(Boolean)),
    )
    return actions.map((action) => ({
      value: action,
      label: action,
    }))
  }, [records])

  return (
    <PageContainer title="审计日志" description="查询系统操作记录，仅支持只读。">
      <Space wrap style={{ marginBottom: 16 }}>
        <Select
          allowClear
          placeholder="按用户筛选"
          style={{ minWidth: 220 }}
          options={userOptions}
          value={userFilter}
          onChange={(value) => {
            setUserFilter(value)
            setPage(1)
          }}
        />

        <Select
          allowClear
          placeholder="按操作类型筛选"
          style={{ minWidth: 180 }}
          options={actionOptions}
          value={actionFilter}
          onChange={(value) => {
            setActionFilter(value)
            setPage(1)
          }}
        />

        <DatePicker.RangePicker
          value={timeRange}
          showTime
          onChange={(values) => {
            if (values?.[0] && values[1]) {
              setTimeRange([values[0], values[1]])
            } else {
              setTimeRange(null)
            }
            setPage(1)
          }}
        />

        <Button
          icon={<ReloadOutlined />}
          loading={loading}
          onClick={() => void loadLogs()}
        >
          刷新
        </Button>
      </Space>

      <Table<AuditLogRecord>
        rowKey="id"
        loading={loading}
        dataSource={records}
        expandable={{
          expandedRowRender: (record) => (
            <Space direction="vertical" size={8} className="w-full">
              <Typography.Text strong>详情</Typography.Text>
              <Typography.Paragraph
                style={{
                  whiteSpace: 'pre-wrap',
                  marginBottom: 0,
                  background: '#fafafa',
                  padding: 12,
                  borderRadius: 8,
                  fontFamily: 'monospace',
                }}
              >
                {parseDetails(record.details)}
              </Typography.Paragraph>
              <Typography.Text type="secondary">
                UserAgent: {record.user_agent ?? '-'}
              </Typography.Text>
            </Space>
          ),
          rowExpandable: (record) => Boolean(record.details || record.user_agent),
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
            title: '时间',
            dataIndex: 'created_at',
            width: 180,
            render: (value: string) => formatDateTime(value),
          },
          {
            title: '用户',
            width: 180,
            render: (_, record) =>
              record.user?.username
                ? `${record.user.username} (#${record.user_id ?? '-'})`
                : `用户 #${record.user_id ?? '-'}`,
          },
          {
            title: '操作',
            dataIndex: 'action',
            width: 160,
          },
          {
            title: '资源类型',
            dataIndex: 'resource_type',
            width: 130,
            render: (value?: string | null) => value ?? '-',
          },
          {
            title: '资源ID',
            dataIndex: 'resource_id',
            width: 100,
            render: (value?: number | null) => value ?? '-',
          },
          {
            title: 'IP',
            dataIndex: 'ip_address',
            width: 140,
            render: (value?: string | null) => value ?? '-',
          },
          {
            title: '详情',
            dataIndex: 'details',
            render: (value?: string | null) => (
              <Typography.Text type="secondary">
                {value ? '点击行展开查看详情' : '-'}
              </Typography.Text>
            ),
          },
        ]}
      />
    </PageContainer>
  )
}

export default AuditLogs
