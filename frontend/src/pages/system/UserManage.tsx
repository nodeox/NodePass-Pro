import {
  EditOutlined,
  MoreOutlined,
  ReloadOutlined,
  RetweetOutlined,
  StopOutlined,
  SearchOutlined,
  FilterOutlined,
  ClearOutlined,
  CheckCircleOutlined,
  CloseCircleOutlined,
  EyeOutlined,
  DownloadOutlined,
} from '@ant-design/icons'
import {
  Alert,
  Button,
  Dropdown,
  Form,
  Input,
  InputNumber,
  Modal,
  Select,
  Space,
  Table,
  Tag,
  Tooltip,
  Typography,
  Card,
  Row,
  Col,
  message,
} from 'antd'
import { useCallback, useEffect, useMemo, useState } from 'react'
import { useNavigate } from 'react-router-dom'

import PageContainer from '../../components/common/PageContainer'
import { usePageTitle } from '../../hooks/usePageTitle'
import { roleApi, trafficApi, userAdminApi, vipApi } from '../../services/api'
import type { AdminUserRecord, RoleRecord, UserRole, VipLevelRecord } from '../../types'
import { getErrorMessage } from '../../utils/error'
import { formatTraffic } from '../../utils/format'

type RoleFormValues = {
  role: UserRole
}

type VipFormValues = {
  level: number
  duration_days: number
}

type TrafficFormValues = {
  traffic_quota: number
}

type SearchFilters = {
  keyword?: string
  role?: UserRole
  status?: string
  vip_level?: number
}

const statusTagMap: Record<string, { color: string; label: string }> = {
  normal: { color: 'green', label: '正常' },
  paused: { color: 'orange', label: '暂停' },
  banned: { color: 'red', label: '封禁' },
  overlimit: { color: 'magenta', label: '超限' },
}

const UserManage = () => {
  usePageTitle('用户管理')
  const navigate = useNavigate()

  const [roleForm] = Form.useForm<RoleFormValues>()
  const [vipForm] = Form.useForm<VipFormValues>()
  const [searchForm] = Form.useForm<SearchFilters>()
  const [trafficForm] = Form.useForm<TrafficFormValues>()

  const [loading, setLoading] = useState<boolean>(false)
  const [actionLoading, setActionLoading] = useState<string | null>(null)
  const [backendUnsupported, setBackendUnsupported] = useState<boolean>(false)

  const [users, setUsers] = useState<AdminUserRecord[]>([])
  const [levels, setLevels] = useState<VipLevelRecord[]>([])
  const [roles, setRoles] = useState<RoleRecord[]>([])
  const [roleModalOpen, setRoleModalOpen] = useState<boolean>(false)
  const [vipModalOpen, setVipModalOpen] = useState<boolean>(false)
  const [trafficModalOpen, setTrafficModalOpen] = useState<boolean>(false)
  const [editingUser, setEditingUser] = useState<AdminUserRecord | null>(null)
  const [showFilters, setShowFilters] = useState<boolean>(false)

  const [page, setPage] = useState<number>(1)
  const [pageSize, setPageSize] = useState<number>(10)
  const [total, setTotal] = useState<number>(0)
  const [filters, setFilters] = useState<SearchFilters>({})
  const [selectedRowKeys, setSelectedRowKeys] = useState<React.Key[]>([])
  const [batchLoading, setBatchLoading] = useState<boolean>(false)

  const loadUsers = useCallback(
    async (targetPage = page, targetPageSize = pageSize): Promise<void> => {
      setLoading(true)
      try {
        const result = await userAdminApi.list({
          page: targetPage,
          pageSize: targetPageSize,
          ...filters,
        })
        setUsers(result.list ?? [])
        setTotal(result.total ?? 0)
        setPage(result.page || targetPage)
        setPageSize(result.page_size || targetPageSize)
        setBackendUnsupported(false)
      } catch (error) {
        const messageText = getErrorMessage(error, '用户列表加载失败')
        if (messageText.includes('404')) {
          setBackendUnsupported(true)
        } else {
          message.error(messageText)
        }
      } finally {
        setLoading(false)
      }
    },
    [page, pageSize, filters],
  )

  const loadLevels = useCallback(async (): Promise<void> => {
    try {
      const result = await vipApi.levels()
      setLevels(result.list ?? [])
    } catch (error) {
      message.error(getErrorMessage(error, 'VIP 等级加载失败'))
    }
  }, [])

  const loadRoles = useCallback(async (): Promise<void> => {
    try {
      const result = await roleApi.list({ include_disabled: true })
      setRoles(result.list ?? [])
    } catch (error) {
      message.error(getErrorMessage(error, '角色列表加载失败'))
    }
  }, [])

  useEffect(() => {
    void loadUsers()
    void loadLevels()
    void loadRoles()
  }, [loadLevels, loadRoles, loadUsers])

  const levelMap = useMemo(() => {
    const map = new Map<number, VipLevelRecord>()
    levels.forEach((level) => map.set(level.level, level))
    return map
  }, [levels])

  const roleMap = useMemo(() => {
    const map = new Map<string, RoleRecord>()
    roles.forEach((role) => map.set(role.code, role))
    return map
  }, [roles])

  const runAction = async (
    key: string,
    action: () => Promise<void>,
    successMessage: string,
  ): Promise<void> => {
    setActionLoading(key)
    try {
      await action()
      message.success(successMessage)
      await loadUsers()
    } catch (error) {
      message.error(getErrorMessage(error, '操作失败'))
    } finally {
      setActionLoading(null)
    }
  }

  const openRoleModal = (user: AdminUserRecord): void => {
    setEditingUser(user)
    roleForm.setFieldsValue({ role: user.role })
    setRoleModalOpen(true)
  }

  const closeRoleModal = (): void => {
    setRoleModalOpen(false)
    setEditingUser(null)
    roleForm.resetFields()
  }

  const openVipModal = (user: AdminUserRecord): void => {
    setEditingUser(user)
    vipForm.setFieldsValue({
      level: user.vip_level,
      duration_days: 30,
    })
    setVipModalOpen(true)
  }

  const closeVipModal = (): void => {
    setVipModalOpen(false)
    setEditingUser(null)
    vipForm.resetFields()
  }

  const openTrafficModal = (user: AdminUserRecord): void => {
    setEditingUser(user)
    trafficForm.setFieldsValue({
      traffic_quota: user.traffic_quota,
    })
    setTrafficModalOpen(true)
  }

  const closeTrafficModal = (): void => {
    setTrafficModalOpen(false)
    setEditingUser(null)
    trafficForm.resetFields()
  }

  const submitRole = async (values: RoleFormValues): Promise<void> => {
    if (!editingUser) {
      return
    }
    await runAction(
      `role-${editingUser.id}`,
      async () => {
        await userAdminApi.updateRole(editingUser.id, values.role)
        closeRoleModal()
      },
      '用户角色更新成功',
    )
  }

  const submitVip = async (values: VipFormValues): Promise<void> => {
    if (!editingUser) {
      return
    }
    await runAction(
      `vip-${editingUser.id}`,
      async () => {
        await vipApi.upgradeUser(editingUser.id, {
          level: values.level,
          duration_days: values.duration_days,
        })
        closeVipModal()
      },
      '用户 VIP 更新成功',
    )
  }

  const submitTraffic = async (values: TrafficFormValues): Promise<void> => {
    if (!editingUser) {
      return
    }
    await runAction(
      `traffic-${editingUser.id}`,
      async () => {
        await trafficApi.updateQuota(editingUser.id, values.traffic_quota)
        closeTrafficModal()
      },
      '流量配额更新成功',
    )
  }

  const handleSearch = (values: SearchFilters): void => {
    setFilters(values)
    setPage(1)
  }

  const handleClearFilters = (): void => {
    searchForm.resetFields()
    setFilters({})
    setPage(1)
  }

  const hasActiveFilters = Object.keys(filters).some(
    (key) => filters[key as keyof SearchFilters] !== undefined && filters[key as keyof SearchFilters] !== '',
  )

  const handleBatchBan = async (): Promise<void> => {
    if (selectedRowKeys.length === 0) {
      message.warning('请先选择要封禁的用户')
      return
    }
    Modal.confirm({
      title: '批量封禁用户',
      content: `确认封禁选中的 ${selectedRowKeys.length} 个用户吗？`,
      okText: '封禁',
      okType: 'danger',
      cancelText: '取消',
      onOk: async () => {
        setBatchLoading(true)
        try {
          await Promise.all(
            selectedRowKeys.map((id) => userAdminApi.updateStatus(Number(id), 'banned'))
          )
          message.success(`已封禁 ${selectedRowKeys.length} 个用户`)
          setSelectedRowKeys([])
          await loadUsers()
        } catch (error) {
          message.error(getErrorMessage(error, '批量封禁失败'))
        } finally {
          setBatchLoading(false)
        }
      },
    })
  }

  const handleBatchUnban = async (): Promise<void> => {
    if (selectedRowKeys.length === 0) {
      message.warning('请先选择要解封的用户')
      return
    }
    setBatchLoading(true)
    try {
      await Promise.all(
        selectedRowKeys.map((id) => userAdminApi.updateStatus(Number(id), 'normal'))
      )
      message.success(`已解封 ${selectedRowKeys.length} 个用户`)
      setSelectedRowKeys([])
      await loadUsers()
    } catch (error) {
      message.error(getErrorMessage(error, '批量解封失败'))
    } finally {
      setBatchLoading(false)
    }
  }

  const handleBatchResetTraffic = async (): Promise<void> => {
    if (selectedRowKeys.length === 0) {
      message.warning('请先选择要重置流量的用户')
      return
    }
    Modal.confirm({
      title: '批量重置流量',
      content: `确认重置选中的 ${selectedRowKeys.length} 个用户的流量吗？`,
      okText: '重置',
      cancelText: '取消',
      onOk: async () => {
        setBatchLoading(true)
        try {
          await Promise.all(
            selectedRowKeys.map((id) => trafficApi.resetQuota(Number(id)))
          )
          message.success(`已重置 ${selectedRowKeys.length} 个用户的流量`)
          setSelectedRowKeys([])
          await loadUsers()
        } catch (error) {
          message.error(getErrorMessage(error, '批量重置流量失败'))
        } finally {
          setBatchLoading(false)
        }
      },
    })
  }

  const handleExportCSV = (): void => {
    try {
      // CSV 表头
      const headers = [
        'ID',
        '用户名',
        '邮箱',
        '角色',
        '状态',
        'VIP等级',
        '流量配额(GB)',
        '已用流量(GB)',
        '使用率(%)',
        'Telegram ID',
        'Telegram用户名',
        '注册时间',
      ]

      // 转换数据
      const rows = users.map((user) => {
        const quotaGB = (user.traffic_quota / 1073741824).toFixed(2)
        const usedGB = (user.traffic_used / 1073741824).toFixed(2)
        const usagePercent = user.traffic_quota > 0
          ? ((user.traffic_used / user.traffic_quota) * 100).toFixed(2)
          : '0.00'

        return [
          user.id,
          user.username,
          user.email,
          roleMap.get(user.role)?.name || user.role,
          statusTagMap[user.status]?.label || user.status,
          user.vip_level,
          quotaGB,
          usedGB,
          usagePercent,
          user.telegram_id || '-',
          user.telegram_username || '-',
          user.created_at,
        ]
      })

      // 生成 CSV 内容
      const csvContent = [
        headers.join(','),
        ...rows.map((row) => row.map((cell) => `"${cell}"`).join(',')),
      ].join('\n')

      // 添加 BOM 以支持中文
      const BOM = '\uFEFF'
      const blob = new Blob([BOM + csvContent], { type: 'text/csv;charset=utf-8;' })
      const url = URL.createObjectURL(blob)
      const link = document.createElement('a')
      link.href = url
      link.download = `用户列表_${new Date().toISOString().split('T')[0]}.csv`
      document.body.appendChild(link)
      link.click()
      document.body.removeChild(link)
      URL.revokeObjectURL(url)

      message.success('导出成功')
    } catch {
      message.error('导出失败')
    }
  }

  return (
    <PageContainer
      title="用户管理"
      description="管理员维护用户角色、VIP、状态与流量配额。"
      extra={
        <Space>
          <Button
            icon={<DownloadOutlined />}
            onClick={handleExportCSV}
            disabled={users.length === 0}
          >
            导出 CSV
          </Button>
          <Button
            icon={<FilterOutlined />}
            onClick={() => setShowFilters(!showFilters)}
            type={hasActiveFilters ? 'primary' : 'default'}
          >
            {showFilters ? '隐藏筛选' : '显示筛选'}
          </Button>
          <Button
            icon={<ReloadOutlined />}
            onClick={() => void loadUsers()}
            loading={loading}
          >
            刷新
          </Button>
        </Space>
      }
    >
      {backendUnsupported ? (
        <Alert
          type="warning"
          showIcon
          message="当前后端未开放用户列表接口（GET /api/v1/users），请先补齐后端用户管理 API。"
          style={{ marginBottom: 16 }}
        />
      ) : null}

      {showFilters && (
        <Card size="small" style={{ marginBottom: 16 }}>
          <Form<SearchFilters>
            form={searchForm}
            layout="vertical"
            onFinish={handleSearch}
          >
            <Row gutter={16}>
              <Col xs={24} sm={12} md={6}>
                <Form.Item label="关键词" name="keyword">
                  <Input
                    placeholder="用户名或邮箱"
                    prefix={<SearchOutlined />}
                    allowClear
                  />
                </Form.Item>
              </Col>
              <Col xs={24} sm={12} md={6}>
                <Form.Item label="角色" name="role">
                  <Select
                    placeholder="全部角色"
                    allowClear
                    options={roles.map((role) => ({
                      label: role.name,
                      value: role.code,
                    }))}
                  />
                </Form.Item>
              </Col>
              <Col xs={24} sm={12} md={6}>
                <Form.Item label="状态" name="status">
                  <Select
                    placeholder="全部状态"
                    allowClear
                    options={[
                      { label: '正常', value: 'normal' },
                      { label: '暂停', value: 'paused' },
                      { label: '封禁', value: 'banned' },
                      { label: '超限', value: 'overlimit' },
                    ]}
                  />
                </Form.Item>
              </Col>
              <Col xs={24} sm={12} md={6}>
                <Form.Item label="VIP 等级" name="vip_level">
                  <Select
                    placeholder="全部等级"
                    allowClear
                    options={levels.map((level) => ({
                      label: `Lv.${level.level} ${level.name}`,
                      value: level.level,
                    }))}
                  />
                </Form.Item>
              </Col>
            </Row>
            <Row>
              <Col span={24}>
                <Space>
                  <Button type="primary" htmlType="submit" icon={<SearchOutlined />}>
                    搜索
                  </Button>
                  <Button icon={<ClearOutlined />} onClick={handleClearFilters}>
                    清空
                  </Button>
                  {hasActiveFilters && (
                    <Typography.Text type="secondary">
                      已应用 {Object.keys(filters).filter((key) => filters[key as keyof SearchFilters]).length} 个筛选条件
                    </Typography.Text>
                  )}
                </Space>
              </Col>
            </Row>
          </Form>
        </Card>
      )}

      {selectedRowKeys.length > 0 && (
        <Card size="small" style={{ marginBottom: 16 }}>
          <Space wrap>
            <Typography.Text strong>
              已选择 {selectedRowKeys.length} 个用户
            </Typography.Text>
            <Button
              size="small"
              icon={<CloseCircleOutlined />}
              onClick={() => setSelectedRowKeys([])}
            >
              取消选择
            </Button>
            <Button
              size="small"
              type="primary"
              danger
              icon={<StopOutlined />}
              onClick={() => void handleBatchBan()}
              loading={batchLoading}
            >
              批量封禁
            </Button>
            <Button
              size="small"
              icon={<CheckCircleOutlined />}
              onClick={() => void handleBatchUnban()}
              loading={batchLoading}
            >
              批量解封
            </Button>
            <Button
              size="small"
              icon={<RetweetOutlined />}
              onClick={() => void handleBatchResetTraffic()}
              loading={batchLoading}
            >
              批量重置流量
            </Button>
          </Space>
        </Card>
      )}

      <Table<AdminUserRecord>
        rowKey="id"
        size="small"
        loading={loading}
        dataSource={users}
        scroll={{ x: 1220 }}
        rowSelection={{
          selectedRowKeys,
          onChange: setSelectedRowKeys,
          preserveSelectedRowKeys: true,
        }}
        pagination={{
          current: page,
          pageSize,
          total,
          showSizeChanger: true,
          showTotal: (recordTotal) => `共 ${recordTotal} 条`,
          onChange: (nextPage, nextPageSize) => {
            setPage(nextPage)
            setPageSize(nextPageSize)
          },
        }}
        columns={[
          { title: 'ID', dataIndex: 'id', width: 64 },
          {
            title: '用户名',
            dataIndex: 'username',
            width: 140,
            render: (value: string) => (
              <Tooltip title={value}>
                <Typography.Text ellipsis style={{ maxWidth: 120 }}>
                  {value}
                </Typography.Text>
              </Tooltip>
            ),
          },
          {
            title: '邮箱',
            dataIndex: 'email',
            width: 210,
            render: (value: string) => (
              <Tooltip title={value}>
                <Typography.Text ellipsis style={{ maxWidth: 190 }}>
                  {value}
                </Typography.Text>
              </Tooltip>
            ),
          },
          {
            title: '角色',
            dataIndex: 'role',
            width: 100,
            render: (role: UserRole) => (
              <Tag color={role === 'admin' ? 'purple' : 'blue'}>
                {roleMap.get(role)?.name || role}
              </Tag>
            ),
          },
          {
            title: 'VIP 等级',
            dataIndex: 'vip_level',
            width: 120,
            render: (value: number) => {
              const level = levelMap.get(value)
              return level ? `Lv.${value} ${level.name}` : `Lv.${value}`
            },
          },
          {
            title: '状态',
            dataIndex: 'status',
            width: 110,
            render: (status: string) => {
              const item = statusTagMap[status]
              if (!item) {
                return <Tag>{status}</Tag>
              }
              return <Tag color={item.color}>{item.label}</Tag>
            },
          },
          {
            title: '流量(已用/配额)',
            width: 190,
            render: (_, record) =>
              `${formatTraffic(record.traffic_used)} / ${formatTraffic(record.traffic_quota)}`,
          },
          {
            title: 'Telegram',
            width: 150,
            render: (_, record) => {
              const telegram = record.telegram_username
                ? `@${record.telegram_username}`
                : record.telegram_id
                  ? record.telegram_id
                  : '-'
              return (
                <Tooltip title={telegram}>
                  <Typography.Text ellipsis style={{ maxWidth: 130 }}>
                    {telegram}
                  </Typography.Text>
                </Tooltip>
              )
            },
          },
          {
            title: '操作',
            width: 220,
            fixed: 'right',
            align: 'center',
            render: (_, record) => (
              <Space size={6} style={{ whiteSpace: 'nowrap' }}>
                <Button
                  size="small"
                  icon={<EyeOutlined />}
                  onClick={() => navigate(`/admin/system/users/${record.id}`)}
                >
                  详情
                </Button>
                <Button
                  size="small"
                  icon={<EditOutlined />}
                  onClick={() => openRoleModal(record)}
                >
                  角色
                </Button>
                <Button
                  size="small"
                  icon={<EditOutlined />}
                  onClick={() => openVipModal(record)}
                >
                  VIP
                </Button>
                <Button
                  size="small"
                  icon={<EditOutlined />}
                  onClick={() => openTrafficModal(record)}
                >
                  流量
                </Button>
                <Dropdown
                  trigger={['click']}
                  menu={{
                    items: [
                      {
                        key: 'toggle',
                        icon: <StopOutlined />,
                        label: record.status === 'banned' ? '解封' : '封禁',
                      },
                      {
                        key: 'reset',
                        icon: <RetweetOutlined />,
                        label: '重置流量',
                      },
                    ],
                    onClick: ({ key }) => {
                      if (key === 'toggle') {
                        void runAction(
                          `status-${record.id}`,
                          async () => {
                            await userAdminApi.updateStatus(
                              record.id,
                              record.status === 'banned' ? 'normal' : 'banned',
                            )
                          },
                          record.status === 'banned' ? '用户已解封' : '用户已封禁',
                        )
                        return
                      }
                      if (key === 'reset') {
                        Modal.confirm({
                          title: '重置用户流量',
                          content: `确认重置用户「${record.username}」的流量吗？`,
                          okText: '重置',
                          cancelText: '取消',
                          onOk: async () => {
                            await runAction(
                              `reset-${record.id}`,
                              async () => {
                                await trafficApi.resetQuota(record.id)
                              },
                              '用户流量已重置',
                            )
                          },
                        })
                      }
                    },
                  }}
                >
                  <Button size="small" icon={<MoreOutlined />}>
                    更多
                  </Button>
                </Dropdown>
              </Space>
            ),
          },
        ]}
      />

      <Modal
        title="编辑角色"
        open={roleModalOpen}
        onCancel={closeRoleModal}
        onOk={() => void roleForm.submit()}
        confirmLoading={actionLoading === `role-${editingUser?.id ?? 0}`}
      >
        <Form<RoleFormValues>
          form={roleForm}
          layout="vertical"
          onFinish={(values) => void submitRole(values)}
        >
          <Form.Item label="角色" name="role" rules={[{ required: true }]}>
            <Select
              options={roles
                .filter((role) => role.is_enabled || role.code === editingUser?.role)
                .map((role) => ({
                  label: role.name,
                  value: role.code,
                }))}
            />
          </Form.Item>
        </Form>
      </Modal>

      <Modal
        title="修改 VIP"
        open={vipModalOpen}
        onCancel={closeVipModal}
        onOk={() => void vipForm.submit()}
        confirmLoading={actionLoading === `vip-${editingUser?.id ?? 0}`}
      >
        <Form<VipFormValues>
          form={vipForm}
          layout="vertical"
          onFinish={(values) => void submitVip(values)}
        >
          <Form.Item label="VIP 等级" name="level" rules={[{ required: true }]}>
            <Select
              options={levels.map((level) => ({
                label: `Lv.${level.level} ${level.name}`,
                value: level.level,
              }))}
            />
          </Form.Item>

          <Form.Item
            label="时长 (天)"
            name="duration_days"
            rules={[{ required: true }]}
          >
            <InputNumber min={1} precision={0} className="w-full" />
          </Form.Item>
        </Form>
      </Modal>

      <Modal
        title="调整流量配额"
        open={trafficModalOpen}
        onCancel={closeTrafficModal}
        onOk={() => void trafficForm.submit()}
        confirmLoading={actionLoading === `traffic-${editingUser?.id ?? 0}`}
      >
        <Form<TrafficFormValues>
          form={trafficForm}
          layout="vertical"
          onFinish={(values) => void submitTraffic(values)}
        >
          <Form.Item
            label="流量配额 (字节)"
            name="traffic_quota"
            rules={[{ required: true, message: '请输入流量配额' }]}
            tooltip="1 GB = 1073741824 字节"
          >
            <InputNumber
              min={0}
              precision={0}
              className="w-full"
              placeholder="例如: 10737418240 (10GB)"
            />
          </Form.Item>
          {editingUser && (
            <Space direction="vertical" style={{ width: '100%' }}>
              <Typography.Text type="secondary">
                当前已用流量: {formatTraffic(editingUser.traffic_used)}
              </Typography.Text>
              <Typography.Text type="secondary">
                当前配额: {formatTraffic(editingUser.traffic_quota)}
              </Typography.Text>
              <Typography.Text type="secondary">
                快捷设置:
              </Typography.Text>
              <Space wrap>
                <Button size="small" onClick={() => trafficForm.setFieldsValue({ traffic_quota: 1073741824 })}>
                  1 GB
                </Button>
                <Button size="small" onClick={() => trafficForm.setFieldsValue({ traffic_quota: 10737418240 })}>
                  10 GB
                </Button>
                <Button size="small" onClick={() => trafficForm.setFieldsValue({ traffic_quota: 53687091200 })}>
                  50 GB
                </Button>
                <Button size="small" onClick={() => trafficForm.setFieldsValue({ traffic_quota: 107374182400 })}>
                  100 GB
                </Button>
                <Button size="small" onClick={() => trafficForm.setFieldsValue({ traffic_quota: 1099511627776 })}>
                  1 TB
                </Button>
              </Space>
            </Space>
          )}
        </Form>
      </Modal>
    </PageContainer>
  )
}

export default UserManage
