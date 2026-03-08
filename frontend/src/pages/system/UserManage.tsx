import {
  EditOutlined,
  MoreOutlined,
  ReloadOutlined,
  RetweetOutlined,
  StopOutlined,
} from '@ant-design/icons'
import {
  Alert,
  Button,
  Dropdown,
  Form,
  InputNumber,
  Modal,
  Select,
  Space,
  Table,
  Tag,
  Tooltip,
  Typography,
  message,
} from 'antd'
import { useCallback, useEffect, useMemo, useState } from 'react'

import PageContainer from '../../components/common/PageContainer'
import { usePageTitle } from '../../hooks/usePageTitle'
import { trafficApi, userAdminApi, vipApi } from '../../services/api'
import type { AdminUserRecord, UserRole, VipLevelRecord } from '../../types'
import { getErrorMessage } from '../../utils/error'
import { formatTraffic } from '../../utils/format'

type RoleFormValues = {
  role: UserRole
}

type VipFormValues = {
  level: number
  duration_days: number
}

const statusTagMap: Record<string, { color: string; label: string }> = {
  normal: { color: 'green', label: '正常' },
  paused: { color: 'orange', label: '暂停' },
  banned: { color: 'red', label: '封禁' },
  overlimit: { color: 'magenta', label: '超限' },
}

const UserManage = () => {
  usePageTitle('用户管理')

  const [roleForm] = Form.useForm<RoleFormValues>()
  const [vipForm] = Form.useForm<VipFormValues>()

  const [loading, setLoading] = useState<boolean>(false)
  const [actionLoading, setActionLoading] = useState<string | null>(null)
  const [backendUnsupported, setBackendUnsupported] = useState<boolean>(false)

  const [users, setUsers] = useState<AdminUserRecord[]>([])
  const [levels, setLevels] = useState<VipLevelRecord[]>([])
  const [roleModalOpen, setRoleModalOpen] = useState<boolean>(false)
  const [vipModalOpen, setVipModalOpen] = useState<boolean>(false)
  const [editingUser, setEditingUser] = useState<AdminUserRecord | null>(null)

  const [page, setPage] = useState<number>(1)
  const [pageSize, setPageSize] = useState<number>(10)
  const [total, setTotal] = useState<number>(0)

  const loadUsers = useCallback(
    async (targetPage = page, targetPageSize = pageSize): Promise<void> => {
      setLoading(true)
      try {
        const result = await userAdminApi.list({
          page: targetPage,
          pageSize: targetPageSize,
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
    [page, pageSize],
  )

  const loadLevels = useCallback(async (): Promise<void> => {
    try {
      const result = await vipApi.levels()
      setLevels(result.list ?? [])
    } catch (error) {
      message.error(getErrorMessage(error, 'VIP 等级加载失败'))
    }
  }, [])

  useEffect(() => {
    void loadUsers()
    void loadLevels()
  }, [loadLevels, loadUsers])

  const levelMap = useMemo(() => {
    const map = new Map<number, VipLevelRecord>()
    levels.forEach((level) => map.set(level.level, level))
    return map
  }, [levels])

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

  return (
    <PageContainer
      title="用户管理"
      description="管理员维护用户角色、VIP、状态与流量配额。"
      extra={
        <Button
          icon={<ReloadOutlined />}
          onClick={() => void loadUsers()}
          loading={loading}
        >
          刷新
        </Button>
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

      <Table<AdminUserRecord>
        rowKey="id"
        size="small"
        loading={loading}
        dataSource={users}
        scroll={{ x: 1220 }}
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
              <Tag color={role === 'admin' ? 'purple' : 'blue'}>{role}</Tag>
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
              options={[
                { label: 'user', value: 'user' },
                { label: 'admin', value: 'admin' },
              ]}
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
    </PageContainer>
  )
}

export default UserManage
