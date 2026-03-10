import {
  DeleteOutlined,
  EditOutlined,
  PlusOutlined,
  ReloadOutlined,
  SafetyCertificateOutlined,
} from '@ant-design/icons'
import {
  Button,
  Card,
  Form,
  Input,
  Modal,
  Popconfirm,
  Select,
  Space,
  Switch,
  Table,
  Tag,
  Tooltip,
  Typography,
  message,
} from 'antd'
import { useCallback, useEffect, useMemo, useState } from 'react'

import PageContainer from '../../components/common/PageContainer'
import { usePageTitle } from '../../hooks/usePageTitle'
import { roleApi } from '../../services/api'
import type { RoleRecord } from '../../types'
import { getErrorMessage } from '../../utils/error'

type RoleFormValues = {
  code?: string
  name: string
  description?: string
  is_enabled: boolean
  permissions?: string[]
}

type PermissionFormValues = {
  permissions: string[]
}

const permissionLabelMap: Record<string, string> = {
  'users.read': '查看用户',
  'users.write': '管理用户',
  'roles.read': '查看角色',
  'roles.write': '管理角色',
  'node_groups.read': '查看节点组',
  'node_groups.write': '管理节点组',
  'node_instances.read': '查看节点实例',
  'node_instances.write': '管理节点实例',
  'tunnels.read': '查看隧道',
  'tunnels.write': '管理隧道',
  'traffic.read': '查看流量',
  'traffic.write': '管理流量',
  'vip.read': '查看 VIP',
  'vip.write': '管理 VIP',
  'benefit_codes.read': '查看权益码',
  'benefit_codes.write': '管理权益码',
  'announcements.read': '查看公告',
  'announcements.write': '管理公告',
  'system.config.read': '查看系统配置',
  'system.config.write': '管理系统配置',
  'audit_logs.read': '查看审计日志',
  'alerts.read': '查看告警',
  'alerts.write': '管理告警',
  'notification_channels.read': '查看通知渠道',
  'notification_channels.write': '管理通知渠道',
}

const getPermissionLabel = (permission: string): string =>
  permissionLabelMap[permission] ?? permission

const getPermissionOptionLabel = (permission: string): string => {
  const label = getPermissionLabel(permission)
  if (label === permission) {
    return permission
  }
  return `${label} (${permission})`
}

const RoleManage = () => {
  usePageTitle('角色管理')

  const [form] = Form.useForm<RoleFormValues>()
  const [permissionForm] = Form.useForm<PermissionFormValues>()

  const [loading, setLoading] = useState<boolean>(false)
  const [submitting, setSubmitting] = useState<boolean>(false)
  const [permissionSubmitting, setPermissionSubmitting] = useState<boolean>(false)

  const [roles, setRoles] = useState<RoleRecord[]>([])
  const [permissionOptions, setPermissionOptions] = useState<string[]>([])
  const [modalOpen, setModalOpen] = useState<boolean>(false)
  const [permissionModalOpen, setPermissionModalOpen] = useState<boolean>(false)
  const [editingRole, setEditingRole] = useState<RoleRecord | null>(null)

  const loadPermissions = useCallback(async () => {
    try {
      const result = await roleApi.permissions()
      setPermissionOptions(result.list ?? [])
    } catch (error) {
      message.error(getErrorMessage(error, '加载权限列表失败'))
    }
  }, [])

  const loadRoles = useCallback(async () => {
    setLoading(true)
    try {
      const result = await roleApi.list({ include_disabled: true })
      setRoles(result.list ?? [])
    } catch (error) {
      message.error(getErrorMessage(error, '加载角色列表失败'))
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    void loadRoles()
    void loadPermissions()
  }, [loadPermissions, loadRoles])

  const openCreateModal = (): void => {
    setEditingRole(null)
    form.setFieldsValue({
      code: '',
      name: '',
      description: '',
      is_enabled: true,
      permissions: [],
    })
    setModalOpen(true)
  }

  const openEditModal = (role: RoleRecord): void => {
    setEditingRole(role)
    form.setFieldsValue({
      name: role.name,
      description: role.description ?? '',
      is_enabled: role.is_enabled,
      permissions: role.permissions ?? [],
    })
    setModalOpen(true)
  }

  const closeModal = (): void => {
    setModalOpen(false)
    setEditingRole(null)
    form.resetFields()
  }

  const openPermissionModal = (role: RoleRecord): void => {
    setEditingRole(role)
    permissionForm.setFieldsValue({
      permissions: role.permissions ?? [],
    })
    setPermissionModalOpen(true)
  }

  const closePermissionModal = (): void => {
    setPermissionModalOpen(false)
    setEditingRole(null)
    permissionForm.resetFields()
  }

  const submitRole = async (values: RoleFormValues): Promise<void> => {
    setSubmitting(true)
    try {
      if (editingRole) {
        const updated = await roleApi.update(editingRole.id, {
          name: values.name,
          description: values.description?.trim() || null,
          is_enabled: values.is_enabled,
        })

        await roleApi.updatePermissions(editingRole.id, {
          permissions: values.permissions ?? [],
        })

        message.success(`角色「${updated.name}」更新成功`)
      } else {
        await roleApi.create({
          code: (values.code ?? '').trim(),
          name: values.name.trim(),
          description: values.description?.trim(),
          is_enabled: values.is_enabled,
          permissions: values.permissions ?? [],
        })
        message.success('角色创建成功')
      }

      closeModal()
      await loadRoles()
    } catch (error) {
      message.error(getErrorMessage(error, '保存角色失败'))
    } finally {
      setSubmitting(false)
    }
  }

  const submitPermissions = async (values: PermissionFormValues): Promise<void> => {
    if (!editingRole) return

    setPermissionSubmitting(true)
    try {
      await roleApi.updatePermissions(editingRole.id, {
        permissions: values.permissions ?? [],
      })
      message.success('角色权限更新成功')
      closePermissionModal()
      await loadRoles()
    } catch (error) {
      message.error(getErrorMessage(error, '更新角色权限失败'))
    } finally {
      setPermissionSubmitting(false)
    }
  }

  const handleDelete = async (role: RoleRecord): Promise<void> => {
    try {
      await roleApi.remove(role.id)
      message.success('角色删除成功')
      await loadRoles()
    } catch (error) {
      message.error(getErrorMessage(error, '删除角色失败'))
    }
  }

  const roleStats = useMemo(() => {
    const total = roles.length
    const enabled = roles.filter((role) => role.is_enabled).length
    const system = roles.filter((role) => role.is_system).length
    return { total, enabled, system }
  }, [roles])

  return (
    <PageContainer
      title="角色管理"
      description="创建业务角色并配置权限，支持为用户分配自定义角色。"
      extra={
        <Space>
          <Button icon={<ReloadOutlined />} onClick={() => void loadRoles()} loading={loading}>
            刷新
          </Button>
          <Button type="primary" icon={<PlusOutlined />} onClick={openCreateModal}>
            新建角色
          </Button>
        </Space>
      }
    >
      <Card size="small" style={{ marginBottom: 16 }}>
        <Space size={24}>
          <Typography.Text>
            角色总数: <Typography.Text strong>{roleStats.total}</Typography.Text>
          </Typography.Text>
          <Typography.Text>
            启用角色: <Typography.Text strong>{roleStats.enabled}</Typography.Text>
          </Typography.Text>
          <Typography.Text>
            系统角色: <Typography.Text strong>{roleStats.system}</Typography.Text>
          </Typography.Text>
        </Space>
      </Card>

      <Table<RoleRecord>
        rowKey="id"
        size="small"
        loading={loading}
        dataSource={roles}
        pagination={false}
        scroll={{ x: 980 }}
        columns={[
          {
            title: '角色编码',
            dataIndex: 'code',
            width: 150,
            render: (value: string) => <Typography.Text code>{value}</Typography.Text>,
          },
          {
            title: '角色名称',
            dataIndex: 'name',
            width: 180,
            render: (value: string, record) => (
              <Space size={6}>
                <Typography.Text strong>{value}</Typography.Text>
                {record.is_system ? <Tag color="gold">系统</Tag> : null}
              </Space>
            ),
          },
          {
            title: '描述',
            dataIndex: 'description',
            ellipsis: true,
            render: (value?: string | null) => value || '-',
          },
          {
            title: '状态',
            dataIndex: 'is_enabled',
            width: 100,
            render: (value: boolean) =>
              value ? <Tag color="green">启用</Tag> : <Tag color="red">禁用</Tag>,
          },
          {
            title: '权限数',
            width: 100,
            render: (_, record) => record.permissions?.length ?? 0,
          },
          {
            title: '权限预览',
            width: 260,
            render: (_, record) => {
              const permissions = record.permissions ?? []
              if (permissions.length === 0) {
                return <Typography.Text type="secondary">-</Typography.Text>
              }
              return (
                <Space size={4} wrap>
                  {permissions.slice(0, 3).map((permission) => (
                    <Tooltip key={permission} title={permission}>
                      <Tag>{getPermissionLabel(permission)}</Tag>
                    </Tooltip>
                  ))}
                  {permissions.length > 3 ? (
                    <Tag color="blue">+{permissions.length - 3}</Tag>
                  ) : null}
                </Space>
              )
            },
          },
          {
            title: '操作',
            fixed: 'right',
            width: 230,
            render: (_, record) => (
              <Space size={8}>
                <Button size="small" icon={<EditOutlined />} onClick={() => openEditModal(record)}>
                  编辑
                </Button>
                <Button
                  size="small"
                  icon={<SafetyCertificateOutlined />}
                  onClick={() => openPermissionModal(record)}
                >
                  权限
                </Button>
                <Popconfirm
                  title="确认删除该角色吗？"
                  description="删除后不可恢复。"
                  okText="删除"
                  okType="danger"
                  cancelText="取消"
                  disabled={record.is_system}
                  onConfirm={() => void handleDelete(record)}
                >
                  <Button
                    size="small"
                    danger
                    icon={<DeleteOutlined />}
                    disabled={record.is_system}
                  >
                    删除
                  </Button>
                </Popconfirm>
              </Space>
            ),
          },
        ]}
      />

      <Modal
        open={modalOpen}
        title={editingRole ? '编辑角色' : '新建角色'}
        onCancel={closeModal}
        onOk={() => void form.submit()}
        confirmLoading={submitting}
        destroyOnClose
      >
        <Form<RoleFormValues> form={form} layout="vertical" onFinish={(values) => void submitRole(values)}>
          {!editingRole ? (
            <Form.Item
              label="角色编码"
              name="code"
              rules={[
                { required: true, message: '请输入角色编码' },
                {
                  pattern: /^[a-z][a-z0-9_-]{1,49}$/,
                  message: '仅支持小写字母、数字、下划线和短横线，长度 2-50',
                },
              ]}
            >
              <Input placeholder="例如: operator" maxLength={50} />
            </Form.Item>
          ) : null}

          <Form.Item label="角色名称" name="name" rules={[{ required: true, message: '请输入角色名称' }]}>
            <Input maxLength={100} />
          </Form.Item>

          <Form.Item label="描述" name="description">
            <Input.TextArea rows={3} maxLength={300} />
          </Form.Item>

          <Form.Item label="权限" name="permissions">
            <Select
              mode="multiple"
              allowClear
              placeholder="请选择角色权限"
              options={permissionOptions.map((permission) => ({
                label: getPermissionOptionLabel(permission),
                value: permission,
              }))}
            />
          </Form.Item>

          <Form.Item label="启用状态" name="is_enabled" valuePropName="checked">
            <Switch disabled={Boolean(editingRole?.is_system)} />
          </Form.Item>
        </Form>
      </Modal>

      <Modal
        open={permissionModalOpen}
        title={editingRole ? `编辑权限 - ${editingRole.name}` : '编辑权限'}
        onCancel={closePermissionModal}
        onOk={() => void permissionForm.submit()}
        confirmLoading={permissionSubmitting}
        destroyOnClose
      >
        <Form<PermissionFormValues>
          form={permissionForm}
          layout="vertical"
          onFinish={(values) => void submitPermissions(values)}
        >
          <Form.Item label="权限列表" name="permissions">
            <Select
              mode="multiple"
              allowClear
              options={permissionOptions.map((permission) => ({
                label: getPermissionOptionLabel(permission),
                value: permission,
              }))}
              placeholder="请选择权限"
            />
          </Form.Item>
        </Form>
      </Modal>
    </PageContainer>
  )
}

export default RoleManage
