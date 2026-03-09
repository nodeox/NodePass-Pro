import {
  App,
  Button,
  ConfigProvider,
  Layout,
  Menu,
  Space,
  Typography,
  message,
  theme
} from 'antd'
import {
  DashboardOutlined,
  KeyOutlined,
  LogoutOutlined,
  ProfileOutlined,
  TagsOutlined,
  ThunderboltOutlined
} from '@ant-design/icons'
import { useEffect, useMemo, useState } from 'react'
import LoginPage from './pages/LoginPage'
import DashboardPage from './pages/DashboardPage'
import PlansPage from './pages/PlansPage'
import LicensesPage from './pages/LicensesPage'
import VersionCenterPage from './pages/VersionCenterPage'
import VerifyLogsPage from './pages/VerifyLogsPage'
import { useAuthStore } from './store/auth'
import { authApi } from './utils/api'
import { extractErrorMessage } from './utils/request'

const { Header, Sider, Content } = Layout

type PageKey = 'dashboard' | 'plans' | 'licenses' | 'version' | 'logs'
const activePageStorageKey = 'license-unified-active-page'

function isPageKey(value: string | null): value is PageKey {
  return value === 'dashboard' || value === 'plans' || value === 'licenses' || value === 'version' || value === 'logs'
}

function MainView() {
  const [activePage, setActivePage] = useState<PageKey>(() => {
    const stored = window.localStorage.getItem(activePageStorageKey)
    if (isPageKey(stored)) {
      return stored
    }
    return 'dashboard'
  })
  const { user, clearAuth } = useAuthStore()

  const menuItems = useMemo(
    () => [
      { key: 'dashboard', icon: <DashboardOutlined />, label: '仪表盘' },
      { key: 'plans', icon: <TagsOutlined />, label: '套餐管理' },
      { key: 'licenses', icon: <KeyOutlined />, label: '授权管理' },
      { key: 'version', icon: <ThunderboltOutlined />, label: '版本中心' },
      { key: 'logs', icon: <ProfileOutlined />, label: '校验日志' }
    ],
    []
  )

  const content = useMemo(() => {
    if (activePage === 'dashboard') return <DashboardPage />
    if (activePage === 'plans') return <PlansPage />
    if (activePage === 'licenses') return <LicensesPage />
    if (activePage === 'version') return <VersionCenterPage />
    return <VerifyLogsPage />
  }, [activePage])

  return (
    <Layout className="main-layout">
      <Sider theme="light" width={240} breakpoint="lg" collapsedWidth="0">
        <div className="brand-block">
          <div className="brand-title">NodePass</div>
          <div className="brand-subtitle">License + Version</div>
        </div>
        <Menu
          mode="inline"
          selectedKeys={[activePage]}
          items={menuItems}
          onClick={(e) => {
            const next = e.key as PageKey
            setActivePage(next)
            window.localStorage.setItem(activePageStorageKey, next)
          }}
        />
      </Sider>

      <Layout>
        <Header className="main-header">
          <Space style={{ width: '100%', justifyContent: 'space-between' }}>
            <Typography.Title level={4} style={{ margin: 0 }}>
              授权与版本统一控制台
            </Typography.Title>
            <Space>
              <Typography.Text type="secondary">{user?.username}</Typography.Text>
              <Button icon={<LogoutOutlined />} onClick={clearAuth}>
                退出
              </Button>
            </Space>
          </Space>
        </Header>
        <Content className="main-content">{content}</Content>
      </Layout>
    </Layout>
  )
}

export default function RootApp() {
  const { token, setAuth, clearAuth } = useAuthStore()

  useEffect(() => {
    if (!token) {
      return
    }
    authApi.me().catch((err) => {
      message.error(`登录状态失效：${extractErrorMessage(err)}`)
      window.localStorage.removeItem(activePageStorageKey)
      clearAuth()
    })
  }, [token, clearAuth])

  return (
    <ConfigProvider
      theme={{
        algorithm: theme.defaultAlgorithm,
        token: {
          colorPrimary: '#0f766e',
          borderRadius: 10,
          fontFamily:
            'IBM Plex Sans, Noto Sans SC, PingFang SC, Microsoft YaHei, -apple-system, BlinkMacSystemFont, sans-serif'
        }
      }}
    >
      <App>
        {token ? (
          <MainView />
        ) : (
          <LoginPage
            onSuccess={(newToken, user) => {
              setAuth(newToken, user)
              message.success('登录成功')
            }}
          />
        )}
      </App>
    </ConfigProvider>
  )
}
