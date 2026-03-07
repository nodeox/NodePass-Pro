import { NodeIndexOutlined } from '@ant-design/icons'
import { Space, Typography } from 'antd'

type BrandLogoProps = {
  subtitle?: string
}

const BrandLogo = ({ subtitle }: BrandLogoProps) => (
  <Space direction="vertical" size={6} className="w-full text-center">
    <div className="mx-auto flex h-14 w-14 items-center justify-center rounded-full bg-blue-500 text-white shadow-sm">
      <NodeIndexOutlined style={{ fontSize: 24 }} />
    </div>
    <Typography.Title level={3} style={{ margin: 0 }}>
      NodePass Panel
    </Typography.Title>
    {subtitle ? (
      <Typography.Text type="secondary">{subtitle}</Typography.Text>
    ) : null}
  </Space>
)

export default BrandLogo
