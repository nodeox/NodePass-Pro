import { Spin } from 'antd'
import { useEffect } from 'react'
import { Navigate, Outlet, createBrowserRouter } from 'react-router-dom'

import MainLayout from './components/Layout/MainLayout'
import Login from './pages/auth/Login'
import Register from './pages/auth/Register'
import BenefitCodeManage from './pages/benefit-codes/BenefitCodeManage'
import RedeemCode from './pages/benefit-codes/RedeemCode'
import Dashboard from './pages/dashboard/Dashboard'
import UserDashboard from './pages/dashboard/UserDashboard'
import CreateNodeGroupPage from './pages/NodeGroups/CreateNodeGroup'
import DeployNodePage from './pages/NodeGroups/DeployNode'
import NodeGroupDetailPage from './pages/NodeGroups/NodeGroupDetail'
import NodeGroupsPage from './pages/NodeGroups'
import Profile from './pages/profile/Profile'
import Announcements from './pages/system/Announcements'
import AuditLogs from './pages/system/AuditLogs'
import SystemConfig from './pages/system/SystemConfig'
import UserManage from './pages/system/UserManage'
import TunnelDetail from './pages/tunnels/TunnelDetail'
import TunnelList from './pages/tunnels/TunnelList'
import TrafficStats from './pages/traffic/TrafficStats'
import VipCenter from './pages/vip/VipCenter'
import VipLevelManage from './pages/vip/VipLevelManage'
import { useAuthStore } from './store/auth'
import { getHomePathByRole } from './utils/route'

const FullScreenLoading = () => (
  <div className="flex min-h-screen items-center justify-center">
    <Spin size="large" />
  </div>
)

const ProtectedRoute = () => {
  const token = useAuthStore((state) => state.token)
  const user = useAuthStore((state) => state.user)
  const isAuthenticated = useAuthStore((state) => state.isAuthenticated)
  const isLoading = useAuthStore((state) => state.isLoading)
  const fetchMe = useAuthStore((state) => state.fetchMe)

  useEffect(() => {
    if (token && !user && !isLoading) {
      void fetchMe()
    }
  }, [fetchMe, isLoading, token, user])

  if (isLoading && token) {
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

  return <MainLayout portal={user.role === 'admin' ? 'admin' : 'user'} />
}

const router = createBrowserRouter([
  {
    path: '/login',
    element: <Login />,
  },
  {
    path: '/register',
    element: <Register />,
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
            element: <NodeGroupsPage />,
          },
          {
            path: 'node-groups/create',
            element: <CreateNodeGroupPage />,
          },
          {
            path: 'node-groups/:id',
            element: <NodeGroupDetailPage />,
          },
          {
            path: 'node-groups/:id/edit',
            element: <NodeGroupDetailPage />,
          },
          {
            path: 'node-groups/:id/deploy',
            element: <DeployNodePage />,
          },
          {
            path: 'tunnels/:id',
            element: <TunnelDetail />,
          },
        ],
      },
      {
        path: 'user',
        element: <Outlet />,
        children: [
          {
            element: <MainLayout portal="user" />,
            children: [
              {
                index: true,
                element: <Navigate to="/user/dashboard" replace />,
              },
              {
                path: 'dashboard',
                element: <UserDashboard />,
              },
              {
                path: 'node-groups',
                element: <NodeGroupsPage />,
              },
              {
                path: 'tunnels',
                element: <TunnelList />,
              },
              {
                path: 'tunnels/:id',
                element: <TunnelDetail />,
              },
              {
                path: 'traffic',
                element: <TrafficStats />,
              },
              {
                path: 'vip',
                element: <VipCenter />,
              },
              {
                path: 'benefit-codes/redeem',
                element: <RedeemCode />,
              },
              {
                path: 'profile',
                element: <Profile />,
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
            element: <MainLayout portal="admin" />,
            children: [
              {
                index: true,
                element: <Navigate to="/admin/dashboard" replace />,
              },
              {
                path: 'dashboard',
                element: <Dashboard />,
              },
              {
                path: 'node-groups',
                element: <NodeGroupsPage />,
              },
              {
                path: 'tunnels',
                element: <TunnelList />,
              },
              {
                path: 'tunnels/:id',
                element: <TunnelDetail />,
              },
              {
                path: 'traffic',
                element: <TrafficStats />,
              },
              {
                path: 'vip/levels',
                element: <VipLevelManage />,
              },
              {
                path: 'benefit-codes',
                element: <BenefitCodeManage />,
              },
              {
                path: 'system/config',
                element: <SystemConfig />,
              },
              {
                path: 'system/announcements',
                element: <Announcements />,
              },
              {
                path: 'system/audit-logs',
                element: <AuditLogs />,
              },
              {
                path: 'system/users',
                element: <UserManage />,
              },
              {
                path: 'profile',
                element: <Profile />,
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
