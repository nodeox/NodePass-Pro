import type { ReactNode } from 'react'
import { Card, Empty, Space, Typography } from 'antd'

type PageContainerProps = {
  title: string
  description?: string
  extra?: ReactNode
  children?: ReactNode
}

const PageContainer = ({
  title,
  description,
  extra,
  children,
}: PageContainerProps) => (
  <Card className="shadow-sm" extra={extra}>
    <Space direction="vertical" size={16} className="w-full">
      <div>
        <Typography.Title level={4} style={{ marginBottom: 4 }}>
          {title}
        </Typography.Title>
        {description ? (
          <Typography.Text type="secondary">{description}</Typography.Text>
        ) : null}
      </div>
      {children ?? <Empty description="页面开发中" />}
    </Space>
  </Card>
)

export default PageContainer
