import { ArrowLeftOutlined, CopyOutlined } from '@ant-design/icons'
import {
  Alert,
  Button,
  Card,
  Form,
  Input,
  Modal,
  Space,
  Steps,
  Switch,
  Tag,
  Typography,
  message,
} from 'antd'
import { useCallback, useEffect, useMemo, useState } from 'react'
import { useNavigate, useParams } from 'react-router-dom'

import { nodeGroupApi, nodeInstanceApi } from '../../services/nodeGroupApi'
import type { DeployCommandResponse, NodeGroup } from '../../types/nodeGroup'
import { getErrorMessage } from '../../utils/error'

type FormValues = {
  service_name: string
  debug_mode: boolean
}

type DeployResult = {
  nodeID: string
  command: string
  serviceName: string
}

type DeployNodeStatus =
  | 'pending'
  | 'online'
  | 'offline'
  | 'maintain'
  | 'not_found'
  | 'deleted_manual'
  | 'deleted_timeout'

const DEPLOY_TIMEOUT_MS = 10 * 60 * 1000

const random4 = (): string => `${Math.floor(1000 + Math.random() * 9000)}`
const safeTrim = (value: unknown): string =>
  typeof value === 'string' ? value.trim() : ''

const normalizeServiceName = (name: string): string => {
  const normalized = name
    .toLowerCase()
    .replace(/[^a-z0-9-]+/g, '-')
    .replace(/-+/g, '-')
    .replace(/^-|-$/g, '')
  return normalized || 'group'
}

const buildDefaultServiceName = (groupName: string): string =>
  `nodepass-${normalizeServiceName(groupName)}-${random4()}`

const resolveCommand = (resp: DeployCommandResponse): string => {
  if (resp.command) {
    return resp.command
  }
  return ''
}

const DeployNode = () => {
  const navigate = useNavigate()
  const { id } = useParams<{ id: string }>()
  const groupID = Number(id)

  const [form] = Form.useForm<FormValues>()
  const [group, setGroup] = useState<NodeGroup | null>(null)
  const [loading, setLoading] = useState<boolean>(false)
  const [generating, setGenerating] = useState<boolean>(false)
  const [deployResult, setDeployResult] = useState<DeployResult | null>(null)
  const [deployNodeStatus, setDeployNodeStatus] = useState<DeployNodeStatus>('pending')
  const [deployStatusLoading, setDeployStatusLoading] = useState<boolean>(false)
  const [deployLastHeartbeat, setDeployLastHeartbeat] = useState<string | null>(null)
  const [deployInstanceID, setDeployInstanceID] = useState<number | null>(null)
  const [deployWatchStartedAt, setDeployWatchStartedAt] = useState<number | null>(null)
  const [onlineNotified, setOnlineNotified] = useState<boolean>(false)

  const loadGroup = useCallback(async () => {
    if (!Number.isFinite(groupID) || groupID <= 0) {
      message.error('无效的节点组 ID')
      return
    }

    setLoading(true)
    try {
      const detail = await nodeGroupApi.get(groupID)
      setGroup(detail)
      form.setFieldsValue({
        service_name: buildDefaultServiceName(detail.name),
        debug_mode: false,
      })
    } catch (error) {
      message.error(getErrorMessage(error, '加载节点组信息失败'))
    } finally {
      setLoading(false)
    }
  }, [form, groupID])

  useEffect(() => {
    void loadGroup()
  }, [loadGroup])

  const onGenerate = async (values: FormValues) => {
    if (!Number.isFinite(groupID) || groupID <= 0) {
      message.error('无效的节点组 ID')
      return
    }

    setGenerating(true)
    try {
      const serviceName = safeTrim(values.service_name)
      if (!serviceName) {
        message.error('服务名称不能为空')
        return
      }

      const resp = await nodeGroupApi.generateDeployCommand(groupID, {
        service_name: serviceName,
        debug_mode: values.debug_mode,
      })

      const nodeID = resp.node_id
      const command = resolveCommand(resp)
      const finalServiceName = resp.service_name || serviceName

      if (!nodeID || !command) {
        throw new Error('服务返回的部署信息不完整')
      }

      setDeployResult({
        nodeID,
        command,
        serviceName: finalServiceName,
      })
      setDeployNodeStatus('pending')
      setDeployLastHeartbeat(null)
      setDeployInstanceID(null)
      setDeployWatchStartedAt(Date.now())
      setOnlineNotified(false)
      message.success('部署命令生成成功')
    } catch (error) {
      message.error(getErrorMessage(error, '生成部署命令失败'))
    } finally {
      setGenerating(false)
    }
  }

  const copyText = async (text: string, successMessage: string) => {
    try {
      await navigator.clipboard.writeText(text)
      message.success(successMessage)
    } catch {
      message.error('复制失败，请手动复制')
    }
  }

  const refreshDeployStatus = useCallback(
    async (silent = false): Promise<DeployNodeStatus | null> => {
      if (!deployResult || !Number.isFinite(groupID) || groupID <= 0) {
        return null
      }

      if (!silent) {
        setDeployStatusLoading(true)
      }

      try {
        const list = await nodeGroupApi.listNodes(groupID)
        const target = list.find((item) => item.node_id === deployResult.nodeID)
        if (!target) {
          setDeployNodeStatus('not_found')
          setDeployLastHeartbeat(null)
          setDeployInstanceID(null)
          return 'not_found'
        }
        setDeployInstanceID(target.id)
        setDeployNodeStatus(target.status ?? 'offline')
        setDeployLastHeartbeat(target.last_heartbeat_at ?? null)
        return target.status ?? 'offline'
      } catch (error) {
        if (!silent) {
          message.error(getErrorMessage(error, '检测节点状态失败'))
        }
        return null
      } finally {
        if (!silent) {
          setDeployStatusLoading(false)
        }
      }
    },
    [deployResult, groupID],
  )

  const deleteDeployNode = useCallback(
    async (reason: 'manual' | 'timeout') => {
      if (!deployResult || !Number.isFinite(groupID) || groupID <= 0) {
        return
      }

      let instanceID = deployInstanceID
      if (!instanceID) {
        const list = await nodeGroupApi.listNodes(groupID)
        const target = list.find((item) => item.node_id === deployResult.nodeID)
        if (!target) {
          setDeployNodeStatus(reason === 'timeout' ? 'deleted_timeout' : 'deleted_manual')
          setDeployInstanceID(null)
          setDeployLastHeartbeat(null)
          return
        }
        instanceID = target.id
      }

      await nodeInstanceApi.delete(instanceID)
      setDeployNodeStatus(reason === 'timeout' ? 'deleted_timeout' : 'deleted_manual')
      setDeployInstanceID(null)
      setDeployLastHeartbeat(null)
      if (reason === 'timeout') {
        message.warning('超过 10 分钟未上线，节点已自动删除')
      } else {
        message.success('节点已手动删除')
      }
    },
    [deployInstanceID, deployResult, groupID],
  )

  useEffect(() => {
    if (!deployResult) {
      return
    }

    void refreshDeployStatus()

    if (
      deployNodeStatus === 'online' ||
      deployNodeStatus === 'deleted_manual' ||
      deployNodeStatus === 'deleted_timeout'
    ) {
      return
    }

    const timer = window.setInterval(() => {
      void (async () => {
        const latestStatus = await refreshDeployStatus(true)
        if (
          latestStatus &&
          latestStatus !== 'online' &&
          deployWatchStartedAt &&
          Date.now() - deployWatchStartedAt >= DEPLOY_TIMEOUT_MS
        ) {
          try {
            await deleteDeployNode('timeout')
          } catch (error) {
            message.error(getErrorMessage(error, '自动删除超时节点失败'))
          }
        }
      })()
    }, 5000)

    return () => {
      window.clearInterval(timer)
    }
  }, [deployResult, deployNodeStatus, deployWatchStartedAt, refreshDeployStatus, deleteDeployNode])

  useEffect(() => {
    if (deployNodeStatus === 'online' && !onlineNotified) {
      message.success('节点已上线')
      setOnlineNotified(true)
    }
  }, [deployNodeStatus, onlineNotified])

  const deployStatusTag = useMemo(() => {
    if (deployNodeStatus === 'online') {
      return <Tag color="green">已上线</Tag>
    }
    if (deployNodeStatus === 'maintain') {
      return <Tag color="orange">维护中</Tag>
    }
    if (deployNodeStatus === 'offline') {
      return <Tag color="red">离线</Tag>
    }
    if (deployNodeStatus === 'deleted_manual') {
      return <Tag color="default">已手动删除</Tag>
    }
    if (deployNodeStatus === 'deleted_timeout') {
      return <Tag color="default">超时已删除</Tag>
    }
    if (deployNodeStatus === 'not_found') {
      return <Tag color="default">等待注册</Tag>
    }
    return <Tag color="processing">检测中</Tag>
  }, [deployNodeStatus])

  const stepItems = useMemo(
    () => [
      { title: '复制上方命令' },
      { title: '在目标服务器上执行' },
      { title: '等待节点上线（状态变为 online）' },
      { title: '返回节点管理页面查看' },
    ],
    [],
  )

  return (
    <Space direction="vertical" size={16} className="w-full">
      <Card
        title="部署节点"
        loading={loading}
        extra={
          <Button
            icon={<ArrowLeftOutlined />}
            onClick={() => navigate(`/node-groups/${groupID}`)}
          >
            返回
          </Button>
        }
      >
        <Space size={12} wrap>
          <Typography.Title level={4} style={{ margin: 0 }}>
            {group?.name ?? '-'}
          </Typography.Title>
          {group?.type === 'entry' ? <Tag color="blue">入口组</Tag> : <Tag color="green">出口组</Tag>}
        </Space>
      </Card>

      <Card title="部署配置" loading={loading}>
        <Form<FormValues>
          form={form}
          layout="vertical"
          initialValues={{ service_name: '', debug_mode: false }}
          onFinish={(values) => void onGenerate(values)}
        >
          <Form.Item
            label="服务名称"
            name="service_name"
            rules={[
              { required: true, message: '请输入服务名称' },
              { max: 100, message: '服务名称不能超过 100 字符' },
            ]}
          >
            <Input placeholder="例如：nodepass-entry-us-1234" />
          </Form.Item>

          <Form.Item label="调试模式" name="debug_mode" valuePropName="checked">
            <Switch />
          </Form.Item>

          <Button type="primary" htmlType="submit" loading={generating}>
            生成命令
          </Button>
        </Form>
      </Card>

      {deployResult ? (
        <Card title="部署命令">
          <Space direction="vertical" size={12} className="w-full">
            <Alert
              type="warning"
              showIcon
              message="Node ID 仅显示一次，请妥善保存"
            />

            <Space>
              <Typography.Text strong>Node ID：</Typography.Text>
              <Typography.Text code>{deployResult.nodeID}</Typography.Text>
              <Button
                size="small"
                icon={<CopyOutlined />}
                onClick={() => void copyText(deployResult.nodeID, 'Node ID 已复制')}
              >
                复制
              </Button>
            </Space>

            <Space>
              <Typography.Text strong>部署状态：</Typography.Text>
              {deployStatusTag}
              <Button size="small" loading={deployStatusLoading} onClick={() => void refreshDeployStatus()}>
                刷新状态
              </Button>
              <Button
                danger
                size="small"
                onClick={() => {
                  Modal.confirm({
                    title: '手动删除节点',
                    content: '确认删除当前部署节点吗？删除后需重新生成部署命令。',
                    okText: '删除',
                    okType: 'danger',
                    cancelText: '取消',
                    onOk: async () => {
                      try {
                        await deleteDeployNode('manual')
                      } catch (error) {
                        message.error(getErrorMessage(error, '删除节点失败'))
                      }
                    },
                  })
                }}
              >
                手动删除节点
              </Button>
              {deployLastHeartbeat ? (
                <Typography.Text type="secondary">
                  最后心跳：{deployLastHeartbeat}
                </Typography.Text>
              ) : null}
            </Space>

            <Typography.Text strong>部署命令</Typography.Text>
            <Typography.Paragraph
              copyable={{ text: deployResult.command }}
              style={{
                marginBottom: 0,
                background: '#f5f5f5',
                padding: 12,
                borderRadius: 8,
                whiteSpace: 'pre-wrap',
                fontFamily: 'ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace',
              }}
            >
              {deployResult.command}
            </Typography.Paragraph>
          </Space>
        </Card>
      ) : null}

      <Card title="部署说明">
        <Steps direction="vertical" current={-1} items={stepItems} />
      </Card>
    </Space>
  )
}

export default DeployNode
