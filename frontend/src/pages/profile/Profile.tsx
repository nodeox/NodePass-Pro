import { Card, Descriptions } from 'antd'

import PageContainer from '../../components/common/PageContainer'
import { usePageTitle } from '../../hooks/usePageTitle'
import { useAuthStore } from '../../store/auth'
import { formatBytes, formatDateTime } from '../../utils/format'

const Profile = () => {
  usePageTitle('个人中心')

  const user = useAuthStore((state) => state.user)

  return (
    <PageContainer title="个人中心" description="查看账户信息、配额和绑定状态。">
      <Card>
        <Descriptions column={1} bordered>
          <Descriptions.Item label="用户名">{user?.username ?? '-'}</Descriptions.Item>
          <Descriptions.Item label="邮箱">{user?.email ?? '-'}</Descriptions.Item>
          <Descriptions.Item label="角色">{user?.role ?? '-'}</Descriptions.Item>
          <Descriptions.Item label="状态">{user?.status ?? '-'}</Descriptions.Item>
          <Descriptions.Item label="VIP 等级">{user?.vip_level ?? 0}</Descriptions.Item>
          <Descriptions.Item label="流量配额">
            {formatBytes(user?.traffic_quota ?? 0)}
          </Descriptions.Item>
          <Descriptions.Item label="已用流量">
            {formatBytes(user?.traffic_used ?? 0)}
          </Descriptions.Item>
          <Descriptions.Item label="VIP 到期">
            {formatDateTime(user?.vip_expires_at)}
          </Descriptions.Item>
        </Descriptions>
      </Card>
    </PageContainer>
  )
}

export default Profile
