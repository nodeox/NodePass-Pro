import { ArrowLeftOutlined, CopyOutlined } from '@ant-design/icons'
import {
  Alert,
  Button,
  Card,
  Form,
  Input,
  Space,
  Steps,
  Switch,
  Tag,
  Typography,
  message,
} from 'antd'
import { useCallback, useEffect, useMemo, useState } from 'react'
import { useNavigate, useParams } from 'react-router-dom'

import { nodeGroupApi } from '../../services/nodeGroupApi'
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

const random4 = (): string => `${Math.floor(1000 + Math.random() * 9000)}`

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
      const resp = await nodeGroupApi.generateDeployCommand(groupID, {
        service_name: values.service_name.trim(),
        debug_mode: values.debug_mode,
      })

      const nodeID = resp.node_id
      const command = resolveCommand(resp)
      const serviceName = resp.service_name || values.service_name

      if (!nodeID || !command) {
        throw new Error('服务返回的部署信息不完整')
      }

      setDeployResult({
        nodeID,
        command,
        serviceName,
      })
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
