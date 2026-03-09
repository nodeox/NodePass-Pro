/* eslint-disable react-refresh/only-export-components */
import { Spin } from 'antd'
import { Suspense, lazy, useEffect, type ReactNode } from 'react'
import { Navigate, Outlet, createBrowserRouter } from 'react-router-dom'

import { useAuthStore } from './store/auth'
import { getHomePathByRole } from './utils/route'

const MainLayout = lazy(() => import('./components/Layout/MainLayout'))
const Login = lazy(() => import('./pages/auth/Login'))
const Register = lazy(() => import('./pages/auth/Register'))
const TelegramCallback = lazy(() => import('./pages/auth/TelegramCallback'))
const ForgotPassword = lazy(() => import('./pages/auth/ForgotPassword'))
const BenefitCodeManage = lazy(() => import('./pages/benefit-codes/BenefitCodeManage'))
const RedeemCode = lazy(() => import('./pages/benefit-codes/RedeemCode'))
const Dashboard = lazy(() => import('./pages/dashboard/Dashboard'))
const UserDashboard = lazy(() => import('./pages/dashboard/UserDashboard'))
const CreateNodeGroupPage = lazy(() => import('./pages/NodeGroups/CreateNodeGroup'))
const DeployNodePage = lazy(() => import('./pages/NodeGroups/DeployNode'))
const EditNodeGroupPage = lazy(() => import('./pages/NodeGroups/EditNodeGroup'))
const NodeGroupDetailPage = lazy(() => import('./pages/NodeGroups/NodeGroupDetail'))
const NodeGroupsPage = lazy(() => import('./pages/NodeGroups'))
const MyNodes = lazy(() => import('./pages/nodes/MyNodes'))
const NodeStatus = lazy(() => import('./pages/nodes/NodeStatus'))
const Profile = lazy(() => import('./pages/profile/Profile'))
const Announcements = lazy(() => import('./pages/system/Announcements'))
const AuditLogs = lazy(() => import('./pages/system/AuditLogs'))
const SystemConfig = lazy(() => import('./pages/system/SystemConfig'))
const UserManage = lazy(() => import('./pages/system/UserManage'))
const UserDetail = lazy(() => import('./pages/system/UserDetail'))
const TunnelDetail = lazy(() => import('./pages/tunnels/TunnelDetail'))
const TunnelList = lazy(() => import('./pages/tunnels/TunnelList'))
const TrafficStats = lazy(() => import('./pages/traffic/TrafficStats'))
const VipCenter = lazy(() => import('./pages/vip/VipCenter'))
const VipLevelManage = lazy(() => import('./pages/vip/VipLevelManage'))

const FullScreenLoading = () => (
  <div className="flex min-h-screen items-center justify-center">
    <Spin size="large" />
  </div>
)

const LazyPage = ({ children }: { children: ReactNode }) => (
  <Suspense fallback={<FullScreenLoading />}>{children}</Suspense>
)

const withLazy = (element: ReactNode) => <LazyPage>{element}</LazyPage>

const ProtectedRoute = () => {
  const token = useAuthStore((state) => state.token)
  const user = useAuthStore((state) => state.user)
  const isAuthenticated = useAuthStore((state) => state.isAuthenticated)
  const authChecked = useAuthStore((state) => state.authChecked)
  const isLoading = useAuthStore((state) => state.isLoading)
  const fetchMe = useAuthStore((state) => state.fetchMe)

  useEffect(() => {
    if (!authChecked && !isLoading) {
      void fetchMe()
      return
    }
    if (authChecked && token && !user && !isLoading) {
      void fetchMe()
    }
  }, [authChecked, fetchMe, isLoading, token, user])

  if (!authChecked || isLoading) {
    return <FullScreenLoading />
  }

  if (!isAuthenticated && !token) {
    return <Navigate to="/login" replace />
  }

  return <Outlet />
}

const AdminRoute = () => {
  const user = useAuthStore((state) => state.user)

  if (!user) {
    return <FullScreenLoading />
  }

  if (user.role !== 'admin') {
    return <Navigate to="/user/dashboard" replace />
  }

  return <Outlet />
}

const RoleHomeRedirect = () => {
  const user = useAuthStore((state) => state.user)

  if (!user) {
    return <FullScreenLoading />
  }

  return <Navigate to={getHomePathByRole(user.role)} replace />
}

const AuthLayout = () => {
  const user = useAuthStore((state) => state.user)

  if (!user) {
    return <FullScreenLoading />
  }

  return withLazy(<MainLayout portal={user.role === 'admin' ? 'admin' : 'user'} />)
}

const router = createBrowserRouter([
  {
    path: '/login',
    element: withLazy(<Login />),
  },
  {
    path: '/register',
    element: withLazy(<Register />),
  },
  {
    path: '/telegram/callback',
    element: withLazy(<TelegramCallback />),
  },
  {
    path: '/forgot-password',
    element: withLazy(<ForgotPassword />),
  },
  {
    path: '/',
    element: <ProtectedRoute />,
    children: [
      {
        index: true,
        element: <RoleHomeRedirect />,
      },
      {
        element: <AuthLayout />,
        children: [
          {
            path: 'node-groups',
            element: withLazy(<NodeGroupsPage />),
          },
          {
            path: 'node-groups/create',
            element: withLazy(<CreateNodeGroupPage />),
          },
          {
            path: 'node-groups/:id',
            element: withLazy(<NodeGroupDetailPage />),
          },
          {
            path: 'node-groups/:id/edit',
            element: withLazy(<EditNodeGroupPage />),
          },
          {
            path: 'node-groups/:id/deploy',
            element: withLazy(<DeployNodePage />),
          },
          {
            path: 'tunnels/:id',
            element: withLazy(<TunnelDetail />),
          },
        ],
      },
      {
        path: 'user',
        element: <Outlet />,
        children: [
          {
            element: withLazy(<MainLayout portal="user" />),
            children: [
              {
                index: true,
                element: <Navigate to="/user/dashboard" replace />,
              },
              {
                path: 'dashboard',
                element: withLazy(<UserDashboard />),
              },
              {
                path: 'node-groups',
                element: <Navigate to="/user/my-nodes" replace />,
              },
              {
                path: 'my-nodes',
                element: withLazy(<MyNodes />),
              },
              {
                path: 'node-status',
                element: withLazy(<NodeStatus />),
              },
              {
                path: 'tunnels',
                element: withLazy(<TunnelList />),
              },
              {
                path: 'tunnels/:id',
                element: withLazy(<TunnelDetail />),
              },
              {
                path: 'traffic',
                element: withLazy(<TrafficStats />),
              },
              {
                path: 'vip',
                element: withLazy(<VipCenter />),
              },
              {
                path: 'benefit-codes/redeem',
                element: withLazy(<RedeemCode />),
              },
              {
                path: 'profile',
                element: withLazy(<Profile />),
              },
            ],
          },
        ],
      },
      {
        path: 'admin',
        element: <AdminRoute />,
        children: [
          {
            element: withLazy(<MainLayout portal="admin" />),
            children: [
              {
                index: true,
                element: <Navigate to="/admin/dashboard" replace />,
              },
              {
                path: 'dashboard',
                element: withLazy(<Dashboard />),
              },
              {
                path: 'node-groups',
                element: withLazy(<NodeGroupsPage />),
              },
              {
                path: 'tunnels',
                element: withLazy(<TunnelList />),
              },
              {
                path: 'tunnels/:id',
                element: withLazy(<TunnelDetail />),
              },
              {
                path: 'traffic',
                element: withLazy(<TrafficStats />),
              },
              {
                path: 'vip/levels',
                element: withLazy(<VipLevelManage />),
              },
              {
                path: 'benefit-codes',
                element: withLazy(<BenefitCodeManage />),
              },
              {
                path: 'system/config',
                element: withLazy(<SystemConfig />),
              },
              {
                path: 'system/announcements',
                element: withLazy(<Announcements />),
              },
              {
                path: 'system/audit-logs',
                element: withLazy(<AuditLogs />),
              },
              {
                path: 'system/users',
                element: withLazy(<UserManage />),
              },
              {
                path: 'system/users/:id',
                element: withLazy(<UserDetail />),
              },
              {
                path: 'profile',
                element: withLazy(<Profile />),
              },
            ],
          },
        ],
      },
    ],
  },
  {
    path: '*',
    element: <Navigate to="/" replace />,
  },
])

export default router
