import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { Card, Table, Tag } from 'antd'
import { logApi } from '@/api'
import type { VerifyLog } from '@/types'
import dayjs from 'dayjs'

export default function Logs() {
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(20)

  const { data, isLoading } = useQuery({
    queryKey: ['verify-logs', page, pageSize],
    queryFn: () => logApi.listVerifyLogs({ page, page_size: pageSize }),
  })

  const columns = [
    {
      title: '授权码',
      dataIndex: 'license_key',
      key: 'license_key',
      width: 200,
      render: (text: string) => <code>{text}</code>,
    },
    {
      title: '机器ID',
      dataIndex: 'machine_id',
      key: 'machine_id',
      width: 150,
    },
    {
      title: '操作',
      dataIndex: 'action',
      key: 'action',
      width: 100,
    },
    {
      title: '结果',
      dataIndex: 'result',
      key: 'result',
      width: 100,
      render: (result: string) => (
        <Tag color={result === 'success' ? 'success' : 'error'}>
          {result === 'success' ? '成功' : '失败'}
        </Tag>
      ),
    },
    {
      title: '原因',
      dataIndex: 'reason',
      key: 'reason',
      ellipsis: true,
    },
    {
      title: 'IP地址',
      dataIndex: 'ip_address',
      key: 'ip_address',
      width: 150,
    },
    {
      title: '版本',
      key: 'versions',
      width: 200,
      render: (_: any, record: VerifyLog) => (
        <div style={{ fontSize: 12 }}>
          <div>Panel: {record.panel_version}</div>
          <div>Backend: {record.backend_version}</div>
        </div>
      ),
    },
    {
      title: '时间',
      dataIndex: 'created_at',
      key: 'created_at',
      width: 180,
      render: (text: string) => dayjs(text).format('YYYY-MM-DD HH:mm:ss'),
    },
  ]

  return (
    <Card title="验证日志">
      <Table
        columns={columns}
        dataSource={data?.data?.items || []}
        loading={isLoading}
        rowKey="id"
        scroll={{ x: 1200 }}
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
