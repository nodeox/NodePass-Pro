import {
  AuditOutlined,
  BellOutlined,
  ClusterOutlined,
  ClearOutlined,
  CrownOutlined,
  DashboardOutlined,
  DisconnectOutlined,
  GiftOutlined,
  LogoutOutlined,
  NotificationOutlined,
  SettingOutlined,
  SwapOutlined,
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
  Tag,
  Tooltip,
  Typography,
  type MenuProps,
} from 'antd'
import { type ReactNode, useMemo } from 'react'
import { Outlet, useLocation, useNavigate } from 'react-router-dom'

import { useWebSocket } from '../../hooks/useWebSocket'
import { useAuthStore } from '../../store/auth'
import { useAppStore } from '../../store/app'
import type { AppNotification } from '../../types'
import { buildPortalPath, type PortalType } from '../../utils/route'

type MenuEntry = {
  key: string
  label: string
  icon: ReactNode
}

type MainLayoutProps = {
  portal: PortalType
}

const getUserMenuEntries = (): MenuEntry[] => [
  {
    key: '/user/dashboard',
    label: '仪表盘',
    icon: <DashboardOutlined />,
  },
  {
    key: '/user/my-nodes',
    label: '我的节点',
    icon: <ClusterOutlined />,
  },
  {
    key: '/user/node-status',
    label: '节点状态',
    icon: <WifiOutlined />,
  },
  {
    key: '/user/tunnels',
    label: '我的隧道',
    icon: <SwapOutlined />,
  },
  {
    key: '/user/traffic',
    label: '流量统计',
    icon: <DashboardOutlined />,
  },
  {
    key: '/user/vip',
    label: 'VIP 中心',
    icon: <CrownOutlined />,
  },
  {
    key: '/user/benefit-codes/redeem',
    label: '权益码',
    icon: <GiftOutlined />,
  },
  {
    key: '/user/profile',
    label: '个人中心',
    icon: <UserOutlined />,
  },
]

const getAdminMenuEntries = (): MenuEntry[] => [
  {
    key: '/admin/dashboard',
    label: '仪表盘',
    icon: <DashboardOutlined />,
  },
  {
    key: '/admin/system/users',
    label: '用户管理',
    icon: <UserOutlined />,
  },
  {
    key: '/node-groups',
    label: '节点组管理',
    icon: <ClusterOutlined />,
  },
  {
    key: '/admin/tunnels',
    label: '隧道管理',
    icon: <SettingOutlined />,
  },
  {
    key: '/admin/traffic',
    label: '流量统计',
    icon: <DashboardOutlined />,
  },
  {
    key: '/admin/vip/levels',
    label: 'VIP 等级',
    icon: <CrownOutlined />,
  },
  {
    key: '/admin/benefit-codes',
    label: '权益码管理',
    icon: <GiftOutlined />,
  },
  {
    key: '/admin/system/config',
    label: '系统配置',
    icon: <SettingOutlined />,
  },
  {
    key: '/admin/system/announcements',
    label: '公告管理',
    icon: <NotificationOutlined />,
  },
  {
    key: '/admin/system/audit-logs',
    label: '审计日志',
    icon: <AuditOutlined />,
  },
  {
    key: '/admin/profile',
    label: '个人中心',
    icon: <UserOutlined />,
  },
]

const getSelectedMenuKey = (pathname: string, items: MenuEntry[]): string => {
  const matched = items
    .map((item) => item.key)
    .filter((key) => pathname === key || pathname.startsWith(`${key}/`))
    .sort((left, right) => right.length - left.length)

  return matched[0] ?? items[0]?.key ?? '/user/dashboard'
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

const MainLayout = ({ portal }: MainLayoutProps) => {
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

  const menuEntries = useMemo<MenuEntry[]>(() => {
    if (portal === 'admin') {
      return getAdminMenuEntries()
    }
    return getUserMenuEntries()
  }, [portal])

  const selectedKey = useMemo(
    () => getSelectedMenuKey(location.pathname, menuEntries),
    [location.pathname, menuEntries],
  )

  const menuItems = useMemo<MenuProps['items']>(
    () =>
      menuEntries.map((menu) => ({
        key: menu.key,
        icon: menu.icon,
        label: menu.label,
      })),
    [menuEntries],
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
        key: 'switch-portal',
        label: portal === 'admin' ? '切换到用户端' : '切换到管理端',
        icon: <SwapOutlined />,
        disabled: user?.role !== 'admin',
      },
      {
        key: 'logout',
        label: '退出登录',
        icon: <LogoutOutlined />,
        danger: true,
      },
    ],
    [portal, user?.role],
  )

  const handleUserMenuClick: MenuProps['onClick'] = ({ key }) => {
    if (key === 'logout') {
      logout()
      navigate('/login', { replace: true })
      return
    }

    if (key === 'profile') {
      navigate(buildPortalPath(portal, '/profile'))
      return
    }

    if (key === 'switch-portal' && user?.role === 'admin') {
      const nextPortal: PortalType = portal === 'admin' ? 'user' : 'admin'
      navigate(buildPortalPath(nextPortal, '/dashboard'))
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

  const portalLabel = portal === 'admin' ? '管理端' : '用户端'
  const portalTagColor = portal === 'admin' ? 'gold' : 'blue'

  return (
    <Layout className="min-h-screen">
      <Layout.Sider
        collapsible
        collapsed={siderCollapsed}
        onCollapse={setSiderCollapsed}
        theme="light"
        width={220}
        style={{ borderRight: '1px solid #f0f0f0' }}
      >
        <div className="px-4 py-5">
          <Space direction="vertical" size={4}>
            <Typography.Title level={5} style={{ marginBottom: 0 }}>
              {siderCollapsed ? 'NP' : 'NodePass Panel'}
            </Typography.Title>
            {!siderCollapsed ? (
              <Tag color={portalTagColor} style={{ width: 'fit-content', marginInlineEnd: 0 }}>
                {portalLabel}
              </Tag>
            ) : null}
          </Space>
        </div>
        <Menu
          mode="inline"
          selectedKeys={[selectedKey]}
          items={menuItems}
          onClick={handleMenuClick}
          style={{ borderInlineEnd: 'none' }}
        />
      </Layout.Sider>

      <Layout>
        <Layout.Header
          className="flex items-center justify-between px-6"
          style={{ backgroundColor: '#fff', borderBottom: '1px solid #f0f0f0' }}
        >
          <Space>
            <Tag color={portalTagColor} style={{ marginInlineEnd: 0 }}>
              {portalLabel}
            </Tag>
            <Typography.Text type="secondary">
              欢迎回来，{user?.username ?? '用户'}
            </Typography.Text>
          </Space>

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
