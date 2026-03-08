import { Outlet } from 'react-router-dom'
import { Layout, Menu, Avatar, Dropdown, Space, Badge } from 'antd'
import {
  DashboardOutlined,
  SafetyCertificateOutlined,
  AppstoreOutlined,
  BellOutlined,
  ApiOutlined,
  TagsOutlined,
  FileTextOutlined,
  UserOutlined,
  LogoutOutlined,
  SettingOutlined,
} from '@ant-design/icons'
import { useNavigate, useLocation } from 'react-router-dom'
import { useAuthStore } from '@/store/auth'
import { useQuery } from '@tanstack/react-query'
import { alertApi } from '@/api'

const { Header, Sider, Content } = Layout

export default function MainLayout() {
  const navigate = useNavigate()
  const location = useLocation()
  const { user, logout } = useAuthStore()

  // 获取未读告警数
  const { data: alertStats } = useQuery({
    queryKey: ['alert-stats'],
    queryFn: () => alertApi.getStats(),
    refetchInterval: 30000, // 30秒刷新一次
  })

  const menuItems = [
    {
      key: '/dashboard',
      icon: <DashboardOutlined />,
      label: '仪表盘',
    },
    {
      key: '/licenses',
      icon: <SafetyCertificateOutlined />,
      label: '授权码管理',
    },
    {
      key: '/plans',
      icon: <AppstoreOutlined />,
      label: '套餐管理',
    },
    {
      key: '/alerts',
      icon: (
        <Badge count={alertStats?.data?.unread_count || 0} size="small" offset={[10, 0]}>
          <BellOutlined />
        </Badge>
      ),
      label: '告警管理',
    },
    {
      key: '/webhooks',
      icon: <ApiOutlined />,
      label: 'Webhook',
    },
    {
      key: '/tags',
      icon: <TagsOutlined />,
      label: '标签管理',
    },
    {
      key: '/logs',
      icon: <FileTextOutlined />,
      label: '验证日志',
    },
  ]

  const handleLogout = () => {
    logout()
    navigate('/login')
  }

  const userMenuItems = [
    {
      key: 'profile',
      icon: <UserOutlined />,
      label: '个人信息',
    },
    {
      key: 'settings',
      icon: <SettingOutlined />,
      label: '系统设置',
    },
    {
      type: 'divider' as const,
    },
    {
      key: 'logout',
      icon: <LogoutOutlined />,
      label: '退出登录',
      onClick: handleLogout,
    },
  ]

  return (
    <Layout style={{ minHeight: '100vh' }}>
      <Sider
        theme="dark"
        width={220}
        style={{
          overflow: 'auto',
          height: '100vh',
          position: 'fixed',
          left: 0,
          top: 0,
          bottom: 0,
        }}
      >
        <div
          style={{
            height: 64,
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            gap: 12,
            color: '#fff',
            fontSize: 18,
            fontWeight: 600,
            borderBottom: '1px solid rgba(255,255,255,0.1)',
          }}
        >
          <SafetyCertificateOutlined style={{ fontSize: 24 }} />
          <span>License Center</span>
        </div>
        <Menu
          theme="dark"
          mode="inline"
          selectedKeys={[location.pathname]}
          items={menuItems}
          onClick={({ key }) => navigate(key)}
          style={{ borderRight: 0 }}
        />

        {/* 版本信息 */}
        <div style={{
          position: 'absolute',
          bottom: 16,
          left: 0,
          right: 0,
          textAlign: 'center',
          color: 'rgba(255,255,255,0.45)',
          fontSize: 12,
        }}>
          v0.2.0
        </div>
      </Sider>

      <Layout style={{ marginLeft: 220 }}>
        <Header
          style={{
            padding: '0 24px',
            background: '#fff',
            display: 'flex',
            justifyContent: 'space-between',
            alignItems: 'center',
            boxShadow: '0 1px 4px rgba(0,21,41,.08)',
            position: 'sticky',
            top: 0,
            zIndex: 999,
          }}
        >
          {/* 面包屑或标题 */}
          <div style={{ fontSize: 16, fontWeight: 500, color: '#262626' }}>
            {menuItems.find(item => item.key === location.pathname)?.label || ''}
          </div>

          {/* 用户信息 */}
          <Dropdown menu={{ items: userMenuItems }} placement="bottomRight">
            <Space style={{ cursor: 'pointer' }}>
              <Avatar
                style={{
                  background: 'linear-gradient(135deg, #667eea 0%, #764ba2 100%)',
                }}
                icon={<UserOutlined />}
              />
              <span style={{ fontWeight: 500 }}>{user?.username}</span>
            </Space>
          </Dropdown>
        </Header>

        <Content
          style={{
            margin: '24px',
            minHeight: 'calc(100vh - 112px)',
          }}
        >
          <Outlet />
        </Content>
      </Layout>
    </Layout>
  )
}
