import {
  AuditOutlined,
  BellOutlined,
  ClearOutlined,
  CrownOutlined,
  DashboardOutlined,
  DisconnectOutlined,
  GiftOutlined,
  LogoutOutlined,
  NodeIndexOutlined,
  NotificationOutlined,
  SettingOutlined,
  UserOutlined,
  WifiOutlined,
} from '@ant-design/icons'
import {
  Avatar,
  Badge,
  Button,
  Dropdown,
  Layout,
  Menu,
  Space,
  Tooltip,
  Typography,
  type MenuProps,
} from 'antd'
import { useMemo } from 'react'
import { Outlet, useLocation, useNavigate } from 'react-router-dom'

import { useWebSocket } from '../../hooks/useWebSocket'
import { useAuthStore } from '../../store/auth'
import { useAppStore } from '../../store/app'
import type { AppNotification } from '../../types'

type MenuEntry = {
  key: string
  label: string
  icon: React.ReactNode
  adminOnly?: boolean
}

const menuEntries: MenuEntry[] = [
  {
    key: '/dashboard',
    label: '仪表盘',
    icon: <DashboardOutlined />,
  },
  {
    key: '/nodes',
    label: '节点管理',
    icon: <NodeIndexOutlined />,
  },
  {
    key: '/rules',
    label: '规则管理',
    icon: <SettingOutlined />,
  },
  {
    key: '/traffic',
    label: '流量统计',
    icon: <DashboardOutlined />,
  },
  {
    key: '/vip',
    label: 'VIP 中心',
    icon: <CrownOutlined />,
  },
  {
    key: '/benefit-codes/redeem',
    label: '权益码',
    icon: <GiftOutlined />,
  },
  {
    key: '/profile',
    label: '个人中心',
    icon: <UserOutlined />,
  },
  {
    key: '/system/users',
    label: '用户管理',
    icon: <UserOutlined />,
    adminOnly: true,
  },
  {
    key: '/system/config',
    label: '系统管理',
    icon: <SettingOutlined />,
    adminOnly: true,
  },
  {
    key: '/system/announcements',
    label: '系统公告',
    icon: <NotificationOutlined />,
    adminOnly: true,
  },
  {
    key: '/system/audit-logs',
    label: '审计日志',
    icon: <AuditOutlined />,
    adminOnly: true,
  },
]

const getSelectedMenuKey = (pathname: string, items: MenuEntry[]): string => {
  const matched = items
    .map((item) => item.key)
    .filter((key) => pathname === key || pathname.startsWith(`${key}/`))
    .sort((left, right) => right.length - left.length)

  return matched[0] ?? '/dashboard'
}

const formatNotificationTime = (value: string): string => {
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) {
    return value
  }
  const hours = String(date.getHours()).padStart(2, '0')
  const minutes = String(date.getMinutes()).padStart(2, '0')
  return `${hours}:${minutes}`
}

const renderNotificationItem = (item: AppNotification) => (
  <div style={{ maxWidth: 320 }}>
    <Typography.Text strong>{item.title}</Typography.Text>
    <Typography.Paragraph
      ellipsis={{ rows: 2, tooltip: item.content }}
      style={{ marginBottom: 0 }}
      type="secondary"
    >
      {item.content}
    </Typography.Paragraph>
    <Typography.Text type="secondary" style={{ fontSize: 12 }}>
      {formatNotificationTime(item.created_at)}
    </Typography.Text>
  </div>
)

const MainLayout = () => {
  const navigate = useNavigate()
  const location = useLocation()

  const token = useAuthStore((state) => state.token)
  const user = useAuthStore((state) => state.user)
  const logout = useAuthStore((state) => state.logout)

  const siderCollapsed = useAppStore((state) => state.siderCollapsed)
  const setSiderCollapsed = useAppStore((state) => state.setSiderCollapsed)
  const wsConnected = useAppStore((state) => state.wsConnected)
  const setWsConnected = useAppStore((state) => state.setWsConnected)
  const notifications = useAppStore((state) => state.notifications)
  const notificationCount = useAppStore((state) => state.notificationCount)
  const clearNotifications = useAppStore((state) => state.clearNotifications)
  const markAllNotificationsRead = useAppStore(
    (state) => state.markAllNotificationsRead,
  )
  const handleWebSocketMessage = useAppStore(
    (state) => state.handleWebSocketMessage,
  )

  useWebSocket({
    token,
    enabled: Boolean(token),
    onConnectedChange: setWsConnected,
    onMessage: handleWebSocketMessage,
  })

  const visibleMenus = useMemo(() => {
    if (user?.role === 'admin') {
      return menuEntries
    }
    return menuEntries.filter((menu) => !menu.adminOnly)
  }, [user?.role])

  const selectedKey = useMemo(
    () => getSelectedMenuKey(location.pathname, visibleMenus),
    [location.pathname, visibleMenus],
  )

  const menuItems = useMemo<MenuProps['items']>(
    () =>
      visibleMenus.map((menu) => ({
        key: menu.key,
        icon: menu.icon,
        label: menu.label,
      })),
    [visibleMenus],
  )

  const notificationMenuItems = useMemo<MenuProps['items']>(() => {
    if (notifications.length === 0) {
      return [
        {
          key: 'empty',
          disabled: true,
          label: <Typography.Text type="secondary">暂无消息</Typography.Text>,
        },
      ]
    }

    const items = notifications.slice(0, 8).map((item) => ({
      key: item.id,
      label: renderNotificationItem(item),
    }))

    return [
      ...items,
      { type: 'divider' as const },
      {
        key: 'clear',
        icon: <ClearOutlined />,
        label: '清空通知',
      },
    ]
  }, [notifications])

  const userMenuItems = useMemo<MenuProps['items']>(
    () => [
      {
        key: 'profile',
        label: '个人中心',
        icon: <UserOutlined />,
      },
      {
        key: 'logout',
        label: '退出登录',
        icon: <LogoutOutlined />,
        danger: true,
      },
    ],
    [],
  )

  const handleUserMenuClick: MenuProps['onClick'] = ({ key }) => {
    if (key === 'logout') {
      logout()
      navigate('/login', { replace: true })
      return
    }

    if (key === 'profile') {
      navigate('/profile')
    }
  }

  const handleMenuClick: MenuProps['onClick'] = ({ key }) => {
    navigate(key)
  }

  const handleNotificationMenuClick: MenuProps['onClick'] = ({ key }) => {
    if (key === 'clear') {
      clearNotifications()
    }
  }

  return (
    <Layout className="min-h-screen">
      <Layout.Sider
        collapsible
        collapsed={siderCollapsed}
        onCollapse={setSiderCollapsed}
        theme="light"
        width={220}
      >
        <div className="px-4 py-5">
          <Typography.Title level={5} style={{ marginBottom: 0 }}>
            {siderCollapsed ? 'NP' : 'NodePass Panel'}
          </Typography.Title>
        </div>
        <Menu
          mode="inline"
          selectedKeys={[selectedKey]}
          items={menuItems}
          onClick={handleMenuClick}
        />
      </Layout.Sider>

      <Layout>
        <Layout.Header
          className="flex items-center justify-between px-6"
          style={{ backgroundColor: '#fff' }}
        >
          <Typography.Text type="secondary">
            欢迎回来，{user?.username ?? '用户'}
          </Typography.Text>

          <Space size={16}>
            <Space size={8}>
              {wsConnected ? (
                <Tooltip title="WebSocket 已连接">
                  <WifiOutlined style={{ color: '#52c41a' }} />
                </Tooltip>
              ) : (
                <Tooltip title="WebSocket 未连接">
                  <DisconnectOutlined style={{ color: '#ff4d4f' }} />
                </Tooltip>
              )}
              <Dropdown
                trigger={['click']}
                menu={{
                  items: notificationMenuItems,
                  onClick: handleNotificationMenuClick,
                }}
                onOpenChange={(open) => {
                  if (open && notificationCount > 0) {
                    markAllNotificationsRead()
                  }
                }}
              >
                <Badge count={notificationCount} size="small">
                  <Button shape="circle" icon={<BellOutlined />} />
                </Badge>
              </Dropdown>
            </Space>

            <Dropdown menu={{ items: userMenuItems, onClick: handleUserMenuClick }}>
              <Button type="text" className="flex items-center gap-2">
                <Avatar size="small" icon={<UserOutlined />} />
                <Typography.Text>{user?.username ?? '未登录'}</Typography.Text>
              </Button>
            </Dropdown>
          </Space>
        </Layout.Header>

        <Layout.Content className="p-6">
          <Outlet />
        </Layout.Content>
      </Layout>
    </Layout>
  )
}

export default MainLayout
