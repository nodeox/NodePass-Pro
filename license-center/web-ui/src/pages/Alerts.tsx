import { useState } from 'react'
import { useQuery, useMutation } from '@tanstack/react-query'
import { Card, Table, Button, Space, Tag, Badge, message, Popconfirm, Select } from 'antd'
import { BellOutlined, CheckOutlined, DeleteOutlined } from '@ant-design/icons'
import { alertApi } from '@/api'
import type { Alert } from '@/types'
import dayjs from 'dayjs'

export default function Alerts() {
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(20)
  const [filters, setFilters] = useState<any>({})

  const { data, isLoading, refetch } = useQuery({
    queryKey: ['alerts', page, pageSize, filters],
    queryFn: () => alertApi.list({ ...filters, page, page_size: pageSize }),
  })

  const { data: statsData } = useQuery({
    queryKey: ['alert-stats'],
    queryFn: () => alertApi.getStats(),
  })

  const markReadMutation = useMutation({
    mutationFn: alertApi.markRead,
    onSuccess: () => {
      message.success('已标记为已读')
      refetch()
    },
  })

  const markAllReadMutation = useMutation({
    mutationFn: alertApi.markAllRead,
    onSuccess: () => {
      message.success('已全部标记为已读')
      refetch()
    },
  })

  const deleteMutation = useMutation({
    mutationFn: alertApi.delete,
    onSuccess: () => {
      message.success('删除成功')
      refetch()
    },
  })

  const levelColorMap: any = {
    info: 'blue',
    warning: 'orange',
    error: 'red',
    critical: 'red',
  }

  const levelTextMap: any = {
    info: '信息',
    warning: '警告',
    error: '错误',
    critical: '严重',
  }

  const columns = [
    {
      title: '状态',
      dataIndex: 'is_read',
      key: 'is_read',
      width: 80,
      render: (isRead: boolean) => (
        <Badge status={isRead ? 'default' : 'processing'} />
      ),
    },
    {
      title: '级别',
      dataIndex: 'level',
      key: 'level',
      width: 100,
      render: (level: string) => (
        <Tag color={levelColorMap[level]}>{levelTextMap[level]}</Tag>
      ),
    },
    {
      title: '标题',
      dataIndex: 'title',
      key: 'title',
    },
    {
      title: '消息',
      dataIndex: 'message',
      key: 'message',
      ellipsis: true,
    },
    {
      title: '类型',
      dataIndex: 'type',
      key: 'type',
      width: 150,
    },
    {
      title: '创建时间',
      dataIndex: 'created_at',
      key: 'created_at',
      width: 180,
      render: (text: string) => dayjs(text).format('YYYY-MM-DD HH:mm:ss'),
    },
    {
      title: '操作',
      key: 'action',
      width: 150,
      render: (_: any, record: Alert) => (
        <Space>
          {!record.is_read && (
            <Button
              type="link"
              size="small"
              icon={<CheckOutlined />}
              onClick={() => markReadMutation.mutate(record.id)}
            >
              标记已读
            </Button>
          )}
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

  const stats = statsData?.data

  return (
    <Card
      title={
        <Space>
          <BellOutlined />
          <span>告警管理</span>
          {stats && (
            <Space size="large" style={{ marginLeft: 16 }}>
              <span>
                未读: <Badge count={stats.unread_count} showZero />
              </span>
              <span>
                严重: <Badge count={stats.critical_count} showZero status="error" />
              </span>
              <span>
                警告: <Badge count={stats.warning_count} showZero status="warning" />
              </span>
            </Space>
          )}
        </Space>
      }
      extra={
        <Space>
          <Button onClick={() => refetch()}>刷新</Button>
          <Popconfirm
            title="确定要全部标记为已读吗？"
            onConfirm={() => markAllReadMutation.mutate()}
          >
            <Button type="primary">全部标记已读</Button>
          </Popconfirm>
        </Space>
      }
    >
      <Space style={{ marginBottom: 16 }}>
        <Select
          placeholder="状态"
          style={{ width: 120 }}
          allowClear
          onChange={(value) => setFilters({ ...filters, is_read: value })}
        >
          <Select.Option value={false}>未读</Select.Option>
          <Select.Option value={true}>已读</Select.Option>
        </Select>
        <Select
          placeholder="级别"
          style={{ width: 120 }}
          allowClear
          onChange={(value) => setFilters({ ...filters, level: value })}
        >
          <Select.Option value="info">信息</Select.Option>
          <Select.Option value="warning">警告</Select.Option>
          <Select.Option value="error">错误</Select.Option>
          <Select.Option value="critical">严重</Select.Option>
        </Select>
      </Space>

      <Table
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
    </Card>
  )
}
