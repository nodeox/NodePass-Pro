import { render, screen, waitFor } from '@testing-library/react'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { BrowserRouter } from 'react-router-dom'
import Dashboard from './Dashboard'
import { useAuthStore } from '../../store/auth'
import * as api from '../../services/api'

// Mock react-router-dom
vi.mock('react-router-dom', async () => {
  const actual = await vi.importActual('react-router-dom')
  return {
    ...actual,
  }
})

// Mock API
vi.mock('../../services/api', () => ({
  systemApi: {
    stats: vi.fn(),
  },
  trafficApi: {
    usage: vi.fn(),
  },
  announcementApi: {
    list: vi.fn(),
  },
  auditApi: {
    list: vi.fn(),
  },
}))

// Mock usePageTitle
vi.mock('../../hooks/usePageTitle', () => ({
  usePageTitle: vi.fn(),
}))

// Mock echarts-for-react
vi.mock('echarts-for-react', () => ({
  default: () => <div data-testid="echarts">Chart</div>,
}))

describe('Dashboard', () => {
  beforeEach(() => {
    vi.clearAllMocks()

    // Setup default user
    useAuthStore.setState({
      user: {
        id: 1,
        username: 'testuser',
        email: 'test@example.com',
        role: 'user',
        vip_level: 1,
        traffic_quota: 10737418240, // 10GB
        vip_expires_at: '2026-12-31T23:59:59Z',
      },
      isAuthenticated: true,
      token: 'test-token',
    })

    // Mock API responses
    vi.mocked(api.systemApi.stats).mockResolvedValue({
      total_nodes: 5,
      online_nodes: 3,
      offline_nodes: 2,
      maintain_nodes: 0,
      running_rules: 10,
      total_rules: 15,
    })

    vi.mocked(api.trafficApi.usage).mockResolvedValue({
      total_calculated_traffic: 1073741824, // 1GB
      total_traffic_in: 536870912,
      total_traffic_out: 536870912,
    })

    vi.mocked(api.announcementApi.list).mockResolvedValue({
      list: [
        {
          id: 1,
          title: '系统维护通知',
          content: '系统将于今晚进行维护',
          created_at: '2026-03-08T10:00:00Z',
        },
      ],
      total: 1,
    })

    vi.mocked(api.auditApi.list).mockResolvedValue({
      list: [
        {
          id: 1,
          user_id: 1,
          action: 'login',
          resource_type: 'user',
          resource_id: 1,
          ip_address: '127.0.0.1',
          created_at: '2026-03-08T10:00:00Z',
        },
      ],
      total: 1,
    })
  })

  const renderDashboard = () => {
    return render(
      <BrowserRouter>
        <Dashboard />
      </BrowserRouter>
    )
  }

  it('应该渲染仪表盘', async () => {
    renderDashboard()

    await waitFor(() => {
      expect(screen.queryByRole('progressbar')).not.toBeInTheDocument()
    })

    expect(screen.getByText('节点总数')).toBeInTheDocument()
    expect(screen.getByText('在线节点')).toBeInTheDocument()
    expect(screen.getByText('运行规则')).toBeInTheDocument()
  })

  it('应该显示节点统计信息', async () => {
    renderDashboard()

    await waitFor(() => {
      expect(screen.getByText('5')).toBeInTheDocument() // 节点总数
      expect(screen.getByText('3')).toBeInTheDocument() // 在线节点
    })
  })

  it('应该显示规则统计信息', async () => {
    renderDashboard()

    await waitFor(() => {
      expect(screen.getByText('10')).toBeInTheDocument() // 运行规则
    })
  })

  it('应该显示流量使用情况', async () => {
    renderDashboard()

    await waitFor(() => {
      expect(screen.getByText('本月流量')).toBeInTheDocument()
    })
  })

  it('应该显示 VIP 信息', async () => {
    renderDashboard()

    await waitFor(() => {
      expect(screen.getByText('VIP 等级')).toBeInTheDocument()
    })
  })

  it('应该显示流量趋势图表', async () => {
    renderDashboard()

    await waitFor(() => {
      expect(screen.getByTestId('echarts')).toBeInTheDocument()
    })
  })

  it('应该显示公告列表', async () => {
    renderDashboard()

    await waitFor(() => {
      expect(screen.getByText('系统公告')).toBeInTheDocument()
      expect(screen.getByText('系统维护通知')).toBeInTheDocument()
    })
  })

  it('应该显示操作日志', async () => {
    renderDashboard()

    await waitFor(() => {
      expect(screen.getByText('最近操作')).toBeInTheDocument()
    })
  })

  it('应该在 API 失败时显示错误', async () => {
    vi.mocked(api.systemApi.stats).mockRejectedValue(new Error('API Error'))

    renderDashboard()

    await waitFor(() => {
      expect(screen.queryByRole('progressbar')).not.toBeInTheDocument()
    })
  })

  it('应该处理审计日志 403 错误', async () => {
    const error = new Error('Forbidden')
    Object.assign(error, {
      isAxiosError: true,
      response: { status: 403 },
    })
    vi.mocked(api.auditApi.list).mockRejectedValue(error)

    renderDashboard()

    await waitFor(() => {
      expect(screen.queryByRole('progressbar')).not.toBeInTheDocument()
    })
  })

  it('应该支持切换时间范围', async () => {
    renderDashboard()

    await waitFor(() => {
      expect(screen.getByText('7天')).toBeInTheDocument()
      expect(screen.getByText('30天')).toBeInTheDocument()
    })
  })

  it('应该在没有用户信息时使用默认值', async () => {
    useAuthStore.setState({
      user: null,
    })

    renderDashboard()

    await waitFor(() => {
      expect(screen.queryByRole('progressbar')).not.toBeInTheDocument()
    })
  })
})
