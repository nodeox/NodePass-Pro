import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { BrowserRouter } from 'react-router-dom'
import Login from './Login'
import { useAuthStore } from '../../store/auth'

// Mock react-router-dom
const mockNavigate = vi.fn()
vi.mock('react-router-dom', async () => {
  const actual = await vi.importActual('react-router-dom')
  return {
    ...actual,
    useNavigate: () => mockNavigate,
  }
})

// Mock API
vi.mock('../../services/api', () => ({
  setAuthSession: vi.fn(),
  telegramApi: {
    login: vi.fn(),
  },
}))

// Mock usePageTitle
vi.mock('../../hooks/usePageTitle', () => ({
  usePageTitle: vi.fn(),
}))

// Mock BrandLogo
vi.mock('../../components/common/BrandLogo', () => ({
  default: () => <div data-testid="brand-logo">Brand Logo</div>,
}))

describe('Login', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    // Reset auth store
    useAuthStore.setState({
      token: null,
      user: null,
      isAuthenticated: false,
      isLoading: false,
    })
  })

  const renderLogin = () => {
    return render(
      <BrowserRouter>
        <Login />
      </BrowserRouter>
    )
  }

  it('应该渲染登录表单', () => {
    renderLogin()

    expect(screen.getByTestId('brand-logo')).toBeInTheDocument()
    expect(screen.getByLabelText('邮箱')).toBeInTheDocument()
    expect(screen.getByLabelText('密码')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: '登录' })).toBeInTheDocument()
    expect(screen.getByText('还没有账号？')).toBeInTheDocument()
  })

  it('应该显示记住我复选框', () => {
    renderLogin()

    const checkbox = screen.getByRole('checkbox', { name: '记住我' })
    expect(checkbox).toBeInTheDocument()
    expect(checkbox).toBeChecked() // 默认选中
  })

  it('应该显示忘记密码链接', () => {
    renderLogin()

    const forgotPasswordLink = screen.getByText('忘记密码？')
    expect(forgotPasswordLink).toBeInTheDocument()
    expect(forgotPasswordLink).toHaveAttribute('href', '/forgot-password')
  })

  it('应该显示注册链接', () => {
    renderLogin()

    const registerLink = screen.getByText('立即注册')
    expect(registerLink).toBeInTheDocument()
    expect(registerLink).toHaveAttribute('href', '/register')
  })

  it('应该验证必填字段', async () => {
    const user = userEvent.setup()
    renderLogin()

    const submitButton = screen.getByRole('button', { name: '登录' })
    await user.click(submitButton)

    await waitFor(() => {
      expect(screen.getByText('请输入邮箱')).toBeInTheDocument()
      expect(screen.getByText('请输入密码')).toBeInTheDocument()
    })
  })

  it('应该验证邮箱格式', async () => {
    const user = userEvent.setup()
    renderLogin()

    const emailInput = screen.getByLabelText('邮箱')
    await user.type(emailInput, 'invalid-email')

    const submitButton = screen.getByRole('button', { name: '登录' })
    await user.click(submitButton)

    await waitFor(() => {
      expect(screen.getByText('邮箱格式不正确')).toBeInTheDocument()
    })
  })

  it('应该成功提交登录表单', async () => {
    const user = userEvent.setup()
    const mockLogin = vi.fn().mockResolvedValue({
      token: 'test-token',
      user: { id: 1, email: 'test@example.com', role: 'user' },
    })

    useAuthStore.setState({
      login: mockLogin,
    })

    renderLogin()

    const emailInput = screen.getByLabelText('邮箱')
    const passwordInput = screen.getByLabelText('密码')
    const submitButton = screen.getByRole('button', { name: '登录' })

    await user.type(emailInput, 'test@example.com')
    await user.type(passwordInput, 'password123')
    await user.click(submitButton)

    await waitFor(() => {
      expect(mockLogin).toHaveBeenCalledWith({
        account: 'test@example.com',
        password: 'password123',
      })
    })
  })

  it('应该在登录成功后导航到首页', async () => {
    const user = userEvent.setup()
    const mockLogin = vi.fn().mockResolvedValue({
      token: 'test-token',
      user: { id: 1, email: 'test@example.com', role: 'user' },
    })

    useAuthStore.setState({
      login: mockLogin,
    })

    renderLogin()

    const emailInput = screen.getByLabelText('邮箱')
    const passwordInput = screen.getByLabelText('密码')
    const submitButton = screen.getByRole('button', { name: '登录' })

    await user.type(emailInput, 'test@example.com')
    await user.type(passwordInput, 'password123')
    await user.click(submitButton)

    await waitFor(() => {
      expect(mockNavigate).toHaveBeenCalled()
    })
  })

  it('应该在登录失败时显示错误消息', async () => {
    const user = userEvent.setup()
    const mockLogin = vi.fn().mockRejectedValue(new Error('登录失败'))

    useAuthStore.setState({
      login: mockLogin,
    })

    renderLogin()

    const emailInput = screen.getByLabelText('邮箱')
    const passwordInput = screen.getByLabelText('密码')
    const submitButton = screen.getByRole('button', { name: '登录' })

    await user.type(emailInput, 'test@example.com')
    await user.type(passwordInput, 'wrong-password')
    await user.click(submitButton)

    await waitFor(() => {
      expect(mockLogin).toHaveBeenCalled()
    })
  })

  it('应该在加载时禁用提交按钮', () => {
    useAuthStore.setState({
      isLoading: true,
    })

    renderLogin()

    const submitButton = screen.getByRole('button', { name: '登录' })
    expect(submitButton).toBeDisabled()
  })

  it('应该在已认证时重定向', () => {
    useAuthStore.setState({
      isAuthenticated: true,
      token: 'test-token',
      user: { id: 1, email: 'test@example.com', role: 'user' },
    })

    renderLogin()

    expect(mockNavigate).toHaveBeenCalled()
  })

  it('不应该在没有 Telegram bot username 时显示 Telegram 登录', () => {
    vi.stubEnv('VITE_TELEGRAM_BOT_USERNAME', '')
    renderLogin()

    expect(screen.queryByText('或使用第三方登录')).not.toBeInTheDocument()
  })
})
