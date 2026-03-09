import { Table, Tag } from 'antd'
import dayjs from 'dayjs'
import { useEffect, useState } from 'react'
import type { VerifyLog } from '../types/api'
import { verifyLogApi } from '../utils/api'

export default function VerifyLogsPage() {
  const [items, setItems] = useState<VerifyLog[]>([])
  const [loading, setLoading] = useState(false)

  useEffect(() => {
    setLoading(true)
    verifyLogApi
      .list({ page: 1, page_size: 100 })
      .then((res) => setItems(res.items))
      .finally(() => setLoading(false))
  }, [])

  return (
    <Table
      rowKey="id"
      loading={loading}
      dataSource={items}
      pagination={false}
      scroll={{ x: 1300 }}
      columns={[
        { title: '时间', dataIndex: 'created_at', width: 180, render: (v: string) => dayjs(v).format('YYYY-MM-DD HH:mm:ss') },
        { title: '授权码', dataIndex: 'license_key', width: 260 },
        { title: '设备', dataIndex: 'machine_id', width: 200 },
        { title: '产品', dataIndex: 'product', width: 120 },
        { title: '客户端版本', dataIndex: 'client_version', width: 120 },
        {
          title: '校验结果',
          dataIndex: 'verified',
          width: 120,
          render: (v: boolean) => (v ? <Tag color="green">通过</Tag> : <Tag color="red">失败</Tag>)
        },
        { title: '状态', dataIndex: 'status', width: 140 },
        { title: '原因', dataIndex: 'reason' }
      ]}
    />
  )
}
