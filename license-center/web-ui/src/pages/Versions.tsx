import React, { useEffect, useState } from 'react'
import {
  Card,
  Row,
  Col,
  Tag,
  Table,
  Alert,
  Spin,
  Button,
  Modal,
  Form,
  Input,
  Select,
  DatePicker,
  message,
  Tabs,
  Descriptions,
  Space,
  Statistic,
} from 'antd'
import {
  CheckCircleOutlined,
  CloseCircleOutlined,
  WarningOutlined,
  SyncOutlined,
  HistoryOutlined,
  SettingOutlined,
} from '@ant-design/icons'
import { versionApi } from '@/api'
import dayjs from 'dayjs'

const { TabPane } = Tabs

interface ComponentVersion {
  id: number
  component: string
  version: string
  build_time?: string
  git_commit?: string
  git_branch?: string
  description?: string
  is_active: boolean
  created_at: string
  updated_at: string
}

interface CompatibilityInfo {
  is_compatible: boolean
  warnings: string[]
  errors: string[]
}

interface SystemVersionInfo {
  backend?: ComponentVersion
  frontend?: ComponentVersion
  node_client?: ComponentVersion
  license_center?: ComponentVersion
  compatibility?: CompatibilityInfo
}

interface VersionCompatibility {
  id: number
  backend_version: string
  min_frontend_version: string
  min_node_client_version: string
  min_license_center_version: string
  description?: string
  is_active: boolean
  created_at: string
  updated_at: string
}

const Versions: React.FC = () => {
  const [loading, setLoading] = useState(false)
  const [systemInfo, setSystemInfo] = useState<SystemVersionInfo | null>(null)
  const [compatibilityConfigs, setCompatibilityConfigs] = useState<VersionCompatibility[]>([])
  const [versionHistory, setVersionHistory] = useState<{ [key: string]: ComponentVersion[] }>({})
  const [updateModalVisible, setUpdateModalVisible] = useState(false)
  const [compatModalVisible, setCompatModalVisible] = useState(false)
  const [selectedComponent, setSelectedComponent] = useState<string>('')
  const [form] = Form.useForm()
  const [compatForm] = Form.useForm()

  useEffect(() => {
    loadSystemInfo()
    loadCompatibilityConfigs()
  }, [])

  const loadSystemInfo = async () => {
    setLoading(true)
    try {
      const response = await versionApi.getSystemInfo()
      setSystemInfo(response.data)
    } catch (error: any) {
      message.error('加载系统版本信息失败: ' + (error.response?.data?.message || error.message))
    } finally {
      setLoading(false)
    }
  }

  const loadCompatibilityConfigs = async () => {
    try {
      const response = await versionApi.listCompatibilityConfigs()
      setCompatibilityConfigs(response.data || [])
    } catch (error: any) {
      message.error('加载兼容性配置失败: ' + (error.response?.data?.message || error.message))
    }
  }

  const loadVersionHistory = async (component: string) => {
    try {
      const response = await versionApi.getComponentHistory(component, 20)
      setVersionHistory(prev => ({ ...prev, [component]: response.data || [] }))
    } catch (error: any) {
      message.error('加载版本历史失败: ' + (error.response?.data?.message || error.message))
    }
  }

  const handleUpdateVersion = async (values: any) => {
    try {
      await versionApi.updateComponentVersion({
        component: selectedComponent,
        version: values.version,
        git_commit: values.git_commit,
        git_branch: values.git_branch,
        description: values.description,
        build_time: values.build_time ? values.build_time.toISOString() : undefined,
      })
      message.success('版本更新成功')
      setUpdateModalVisible(false)
      form.resetFields()
      loadSystemInfo()
    } catch (error: any) {
      message.error('更新版本失败: ' + (error.response?.data?.message || error.message))
    }
  }

  const handleCreateCompatibility = async (values: any) => {
    try {
      await versionApi.createCompatibilityConfig(values)
      message.success('兼容性配置创建成功')
      setCompatModalVisible(false)
      compatForm.resetFields()
      loadCompatibilityConfigs()
    } catch (error: any) {
      message.error('创建兼容性配置失败: ' + (error.response?.data?.message || error.message))
    }
  }

  const openUpdateModal = (component: string) => {
    setSelectedComponent(component)
    setUpdateModalVisible(true)
  }

  const getComponentName = (component: string) => {
    const names: { [key: string]: string } = {
      backend: '后端服务',
      frontend: '前端应用',
      node_client: '节点客户端',
      license_center: '授权中心',
    }
    return names[component] || component
  }

  const renderVersionCard = (title: string, component: string, version?: ComponentVersion) => {
    return (
      <Card
        title={title}
        extra={
          <Space>
            <Button
              size="small"
              icon={<HistoryOutlined />}
              onClick={() => {
                loadVersionHistory(component)
              }}
            >
              历史
            </Button>
            <Button
              size="small"
              type="primary"
              icon={<SyncOutlined />}
              onClick={() => openUpdateModal(component)}
            >
              更新
            </Button>
          </Space>
        }
      >
        {version ? (
          <Descriptions column={1} size="small">
            <Descriptions.Item label="版本号">
              <Tag color="blue" style={{ fontSize: '16px' }}>
                v{version.version}
              </Tag>
            </Descriptions.Item>
            {version.git_commit && (
              <Descriptions.Item label="Git Commit">
                <code>{version.git_commit.substring(0, 8)}</code>
              </Descriptions.Item>
            )}
            {version.git_branch && (
              <Descriptions.Item label="Git Branch">{version.git_branch}</Descriptions.Item>
            )}
            {version.build_time && (
              <Descriptions.Item label="构建时间">
                {dayjs(version.build_time).format('YYYY-MM-DD HH:mm:ss')}
              </Descriptions.Item>
            )}
            {version.description && (
              <Descriptions.Item label="描述">{version.description}</Descriptions.Item>
            )}
            <Descriptions.Item label="更新时间">
              {dayjs(version.updated_at).format('YYYY-MM-DD HH:mm:ss')}
            </Descriptions.Item>
          </Descriptions>
        ) : (
          <Alert message="未检测到版本信息" type="warning" showIcon />
        )}
      </Card>
    )
  }

  const compatibilityColumns = [
    {
      title: '后端版本',
      dataIndex: 'backend_version',
      key: 'backend_version',
      render: (text: string) => <Tag color="blue">v{text}</Tag>,
    },
    {
      title: '最低前端版本',
      dataIndex: 'min_frontend_version',
      key: 'min_frontend_version',
      render: (text: string) => <Tag>v{text}</Tag>,
    },
    {
      title: '最低节点客户端版本',
      dataIndex: 'min_node_client_version',
      key: 'min_node_client_version',
      render: (text: string) => <Tag>v{text}</Tag>,
    },
    {
      title: '最低授权中心版本',
      dataIndex: 'min_license_center_version',
      key: 'min_license_center_version',
      render: (text: string) => <Tag>v{text}</Tag>,
    },
    {
      title: '描述',
      dataIndex: 'description',
      key: 'description',
    },
    {
      title: '状态',
      dataIndex: 'is_active',
      key: 'is_active',
      render: (active: boolean) =>
        active ? <Tag color="green">活跃</Tag> : <Tag>非活跃</Tag>,
    },
  ]

  const historyColumns = [
    {
      title: '版本号',
      dataIndex: 'version',
      key: 'version',
      render: (text: string) => <Tag color="blue">v{text}</Tag>,
    },
    {
      title: 'Git Commit',
      dataIndex: 'git_commit',
      key: 'git_commit',
      render: (text: string) => (text ? <code>{text.substring(0, 8)}</code> : '-'),
    },
    {
      title: 'Git Branch',
      dataIndex: 'git_branch',
      key: 'git_branch',
      render: (text: string) => text || '-',
    },
    {
      title: '状态',
      dataIndex: 'is_active',
      key: 'is_active',
      render: (active: boolean) =>
        active ? <Tag color="green">当前</Tag> : <Tag>历史</Tag>,
    },
    {
      title: '创建时间',
      dataIndex: 'created_at',
      key: 'created_at',
      render: (text: string) => dayjs(text).format('YYYY-MM-DD HH:mm:ss'),
    },
  ]

  return (
    <div>
      <h2>版本管理</h2>

      <Spin spinning={loading}>
        {/* 兼容性状态 */}
        {systemInfo?.compatibility && (
          <Card style={{ marginBottom: 16 }}>
            <Row gutter={16} align="middle">
              <Col span={4}>
                <Statistic
                  title="兼容性状态"
                  value={systemInfo.compatibility.is_compatible ? '兼容' : '不兼容'}
                  valueStyle={{
                    color: systemInfo.compatibility.is_compatible ? '#3f8600' : '#cf1322',
                  }}
                  prefix={
                    systemInfo.compatibility.is_compatible ? (
                      <CheckCircleOutlined />
                    ) : (
                      <CloseCircleOutlined />
                    )
                  }
                />
              </Col>
              <Col span={20}>
                {systemInfo.compatibility.errors.length > 0 && (
                  <Alert
                    message="错误"
                    description={
                      <ul style={{ margin: 0, paddingLeft: 20 }}>
                        {systemInfo.compatibility.errors.map((err, idx) => (
                          <li key={idx}>{err}</li>
                        ))}
                      </ul>
                    }
                    type="error"
                    showIcon
                    style={{ marginBottom: 8 }}
                  />
                )}
                {systemInfo.compatibility.warnings.length > 0 && (
                  <Alert
                    message="警告"
                    description={
                      <ul style={{ margin: 0, paddingLeft: 20 }}>
                        {systemInfo.compatibility.warnings.map((warn, idx) => (
                          <li key={idx}>{warn}</li>
                        ))}
                      </ul>
                    }
                    type="warning"
                    showIcon
                  />
                )}
              </Col>
            </Row>
          </Card>
        )}

        <Tabs defaultActiveKey="versions">
          <TabPane tab="组件版本" key="versions">
            <Row gutter={[16, 16]}>
              <Col span={12}>
                {renderVersionCard('后端服务', 'backend', systemInfo?.backend)}
              </Col>
              <Col span={12}>
                {renderVersionCard('前端应用', 'frontend', systemInfo?.frontend)}
              </Col>
              <Col span={12}>
                {renderVersionCard('节点客户端', 'node_client', systemInfo?.node_client)}
              </Col>
              <Col span={12}>
                {renderVersionCard('授权中心', 'license_center', systemInfo?.license_center)}
              </Col>
            </Row>
          </TabPane>

          <TabPane tab="版本历史" key="history">
            <Tabs tabPosition="left">
              {['backend', 'frontend', 'node_client', 'license_center'].map(component => (
                <TabPane tab={getComponentName(component)} key={component}>
                  <Table
                    dataSource={versionHistory[component] || []}
                    columns={historyColumns}
                    rowKey="id"
                    pagination={{ pageSize: 10 }}
                  />
                </TabPane>
              ))}
            </Tabs>
          </TabPane>

          <TabPane tab="兼容性配置" key="compatibility">
            <Button
              type="primary"
              icon={<SettingOutlined />}
              onClick={() => setCompatModalVisible(true)}
              style={{ marginBottom: 16 }}
            >
              新增配置
            </Button>
            <Table
              dataSource={compatibilityConfigs}
              columns={compatibilityColumns}
              rowKey="id"
              pagination={{ pageSize: 10 }}
            />
          </TabPane>
        </Tabs>
      </Spin>

      {/* 更新版本对话框 */}
      <Modal
        title={`更新${getComponentName(selectedComponent)}版本`}
        open={updateModalVisible}
        onCancel={() => {
          setUpdateModalVisible(false)
          form.resetFields()
        }}
        onOk={() => form.submit()}
      >
        <Form form={form} layout="vertical" onFinish={handleUpdateVersion}>
          <Form.Item
            label="版本号"
            name="version"
            rules={[{ required: true, message: '请输入版本号' }]}
          >
            <Input placeholder="例如: 1.0.0" />
          </Form.Item>
          <Form.Item label="Git Commit" name="git_commit">
            <Input placeholder="例如: abc123def456" />
          </Form.Item>
          <Form.Item label="Git Branch" name="git_branch">
            <Input placeholder="例如: main" />
          </Form.Item>
          <Form.Item label="构建时间" name="build_time">
            <DatePicker showTime style={{ width: '100%' }} />
          </Form.Item>
          <Form.Item label="描述" name="description">
            <Input.TextArea rows={3} placeholder="版本描述" />
          </Form.Item>
        </Form>
      </Modal>

      {/* 新增兼容性配置对话框 */}
      <Modal
        title="新增兼容性配置"
        open={compatModalVisible}
        onCancel={() => {
          setCompatModalVisible(false)
          compatForm.resetFields()
        }}
        onOk={() => compatForm.submit()}
        width={600}
      >
        <Form form={compatForm} layout="vertical" onFinish={handleCreateCompatibility}>
          <Form.Item
            label="后端版本"
            name="backend_version"
            rules={[{ required: true, message: '请输入后端版本' }]}
          >
            <Input placeholder="例如: 1.0.0" />
          </Form.Item>
          <Form.Item
            label="最低前端版本"
            name="min_frontend_version"
            rules={[{ required: true, message: '请输入最低前端版本' }]}
          >
            <Input placeholder="例如: 1.0.0" />
          </Form.Item>
          <Form.Item
            label="最低节点客户端版本"
            name="min_node_client_version"
            rules={[{ required: true, message: '请输入最低节点客户端版本' }]}
          >
            <Input placeholder="例如: 0.9.0" />
          </Form.Item>
          <Form.Item
            label="最低授权中心版本"
            name="min_license_center_version"
            rules={[{ required: true, message: '请输入最低授权中心版本' }]}
          >
            <Input placeholder="例如: 1.0.0" />
          </Form.Item>
          <Form.Item label="描述" name="description">
            <Input.TextArea rows={3} placeholder="配置描述" />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  )
}

export default Versions
