import { ArrowLeftOutlined } from '@ant-design/icons'
import {
  Alert,
  Button,
  Card,
  Form,
  Input,
  InputNumber,
  Radio,
  Select,
  Skeleton,
  Space,
  Tag,
  Typography,
  message,
} from 'antd'
import { type ReactNode, useEffect, useMemo, useState } from 'react'
import { useLocation, useNavigate, useParams } from 'react-router-dom'

import PageContainer from '../../components/common/PageContainer'
import { usePageTitle } from '../../hooks/usePageTitle'
import { nodeApi, nodePairApi, ruleApi } from '../../services/api'
import type { CreateRulePayload, NodeRecord, UpdateRulePayload } from '../../types'
import { getErrorMessage } from '../../utils/error'
import { buildPortalPath, resolvePortalByPathname } from '../../utils/route'

type RuleMode = 'single' | 'tunnel'
type RuleProtocol = 'tcp' | 'udp' | 'ws' | 'tls' | 'quic'

type RuleFormValues = {
  mode: RuleMode
  entry_node_id?: number
  exit_node_id?: number
  name: string
  protocol: RuleProtocol
  target_host: string
  target_port?: number
  listen_port?: number
}

type PairCheckStatus = 'idle' | 'loading' | 'missing' | 'disabled' | 'enabled'

type PairCheckState = {
  status: PairCheckStatus
  message: string
}

const statusLabelMap: Record<string, string> = {
  online: '在线',
  offline: '离线',
  maintain: '维护',
}

const statusColorMap: Record<string, string> = {
  online: 'green',
  offline: 'default',
  maintain: 'orange',
}

const protocolOptions: Array<{ label: string; value: RuleProtocol }> = [
  { label: 'TCP', value: 'tcp' },
  { label: 'UDP', value: 'udp' },
  { label: 'WebSocket', value: 'ws' },
  { label: 'TLS', value: 'tls' },
  { label: 'QUIC', value: 'quic' },
]

type NodeSelectOption = {
  value: number
  label: ReactNode
  searchText: string
}

const toRuleID = (rawID?: string): number | null => {
  if (!rawID) {
    return null
  }
  const parsed = Number(rawID)
  if (!Number.isInteger(parsed) || parsed <= 0) {
    return null
  }
  return parsed
}

const RuleForm = () => {
  const navigate = useNavigate()
  const location = useLocation()
  const { id } = useParams<{ id: string }>()
  const ruleID = toRuleID(id)
  const isEdit = ruleID !== null
  const portal = resolvePortalByPathname(location.pathname)

  usePageTitle(isEdit ? '编辑规则' : '创建规则')

  const [form] = Form.useForm<RuleFormValues>()
  const mode = Form.useWatch('mode', form)
  const entryNodeID = Form.useWatch('entry_node_id', form)
  const exitNodeID = Form.useWatch('exit_node_id', form)

  const [nodes, setNodes] = useState<NodeRecord[]>([])
  const [loading, setLoading] = useState<boolean>(true)
  const [submitting, setSubmitting] = useState<boolean>(false)
  const [pairCheck, setPairCheck] = useState<PairCheckState>({
    status: 'idle',
    message: '',
  })

  useEffect(() => {
    let cancelled = false

    const loadData = async (): Promise<void> => {
      setLoading(true)
      try {
        const nodeResultPromise = nodeApi.list({
          page: 1,
          pageSize: 500,
        })

        if (isEdit && ruleID) {
          const [nodeResult, ruleDetail] = await Promise.all([
            nodeResultPromise,
            ruleApi.detail(ruleID),
          ])
          if (cancelled) {
            return
          }

          setNodes(nodeResult.list ?? [])
          form.setFieldsValue({
            mode: ruleDetail.mode,
            entry_node_id: ruleDetail.entry_node_id,
            exit_node_id: ruleDetail.exit_node_id ?? undefined,
            name: ruleDetail.name,
            protocol: ruleDetail.protocol,
            target_host: ruleDetail.target_host,
            target_port: ruleDetail.target_port,
            listen_port: ruleDetail.listen_port,
          })
          return
        }

        const nodeResult = await nodeResultPromise
        if (cancelled) {
          return
        }

        setNodes(nodeResult.list ?? [])
        form.setFieldsValue({
          mode: 'single',
          protocol: 'tcp',
        })
      } catch (error) {
        message.error(getErrorMessage(error, '规则页面初始化失败'))
        if (!cancelled) {
          navigate(buildPortalPath(portal, '/rules'), { replace: true })
        }
      } finally {
        if (!cancelled) {
          setLoading(false)
        }
      }
    }

    void loadData()
    return () => {
      cancelled = true
    }
  }, [form, isEdit, navigate, portal, ruleID])

  useEffect(() => {
    if (mode === 'single') {
      form.setFieldValue('exit_node_id', undefined)
      setPairCheck({
        status: 'idle',
        message: '',
      })
    }
  }, [form, mode])

  useEffect(() => {
    if (mode !== 'tunnel' || !entryNodeID || !exitNodeID) {
      setPairCheck({
        status: 'idle',
        message: '',
      })
      return
    }

    let cancelled = false
    const checkPair = async (): Promise<void> => {
      setPairCheck({
        status: 'loading',
        message: '正在检查节点配对状态...',
      })

      try {
        const pairResult = await nodePairApi.list()
        if (cancelled) {
          return
        }

        const matchedPair = (pairResult.list ?? []).find(
          (pair) =>
            pair.entry_node_id === entryNodeID && pair.exit_node_id === exitNodeID,
        )

        if (!matchedPair) {
          setPairCheck({
            status: 'missing',
            message: '入口节点与出口节点未配对，请先在节点管理中创建配对',
          })
          return
        }

        if (!matchedPair.is_enabled) {
          setPairCheck({
            status: 'disabled',
            message: '该配对已禁用',
          })
          return
        }

        setPairCheck({
          status: 'enabled',
          message: '✓ 已配对',
        })
      } catch (error) {
        if (!cancelled) {
          setPairCheck({
            status: 'missing',
            message: '配对状态检查失败，请稍后重试',
          })
        }
      }
    }

    void checkPair()
    return () => {
      cancelled = true
    }
  }, [entryNodeID, exitNodeID, mode])

  const entryNodeOptions = useMemo(
    () =>
      nodes.map((node) => ({
        value: node.id,
        label: (
          <Space size={6}>
            <Typography.Text
              style={node.status === 'offline' ? { color: '#999' } : undefined}
            >
              {node.name}
            </Typography.Text>
            <Tag color={statusColorMap[node.status] ?? 'default'}>
              {statusLabelMap[node.status] ?? node.status}
            </Tag>
            <Typography.Text type="secondary">
              {node.region ?? '未知区域'}
            </Typography.Text>
          </Space>
        ),
        searchText: `${node.name} ${node.region ?? ''} ${statusLabelMap[node.status] ?? node.status}`,
      })),
    [nodes],
  )

  const exitNodeOptions = useMemo(
    () =>
      nodes
        .filter((node) => node.id !== entryNodeID)
        .map((node) => ({
          value: node.id,
          label: (
            <Space size={6}>
              <Typography.Text
                style={node.status === 'offline' ? { color: '#999' } : undefined}
              >
                {node.name}
              </Typography.Text>
              <Tag color={statusColorMap[node.status] ?? 'default'}>
                {statusLabelMap[node.status] ?? node.status}
              </Tag>
              <Typography.Text type="secondary">
                {node.region ?? '未知区域'}
              </Typography.Text>
            </Space>
          ),
          searchText: `${node.name} ${node.region ?? ''} ${statusLabelMap[node.status] ?? node.status}`,
        })),
    [entryNodeID, nodes],
  )

  const filterNodeOption = (input: string, option?: NodeSelectOption): boolean => {
    if (!option?.searchText) {
      return false
    }
    return option.searchText.toLowerCase().includes(input.toLowerCase())
  }

  const handleSubmit = async (values: RuleFormValues): Promise<void> => {
    if (!values.entry_node_id) {
      message.error('请选择入口节点')
      return
    }

    if (values.mode === 'tunnel') {
      if (!values.exit_node_id) {
        message.error('隧道模式必须选择出口节点')
        return
      }
      if (pairCheck.status !== 'enabled') {
        message.error('当前节点配对不可用，请先修复配对状态')
        return
      }
    }

    const targetPort = values.target_port ?? 0
    const listenPort = values.listen_port ?? 0

    const payload: CreateRulePayload = {
      name: values.name.trim(),
      mode: values.mode,
      protocol: values.protocol,
      entry_node_id: values.entry_node_id,
      exit_node_id: values.mode === 'tunnel' ? values.exit_node_id : undefined,
      target_host: values.target_host.trim(),
      target_port: targetPort,
      listen_host: '0.0.0.0',
      listen_port: listenPort,
    }

    setSubmitting(true)
    try {
      if (isEdit && ruleID) {
        const updatePayload: UpdateRulePayload = { ...payload }
        await ruleApi.update(ruleID, updatePayload)
        message.success('规则更新成功')
      } else {
        await ruleApi.create(payload)
        message.success('规则创建成功')
      }
      navigate(buildPortalPath(portal, '/rules'))
    } catch (error) {
      message.error(getErrorMessage(error, '规则保存失败'))
    } finally {
      setSubmitting(false)
    }
  }

  const renderPairCheckAlert = () => {
    if (mode !== 'tunnel' || !entryNodeID || !exitNodeID) {
      return null
    }

    if (pairCheck.status === 'loading') {
      return <Alert type="info" showIcon message={pairCheck.message} />
    }
    if (pairCheck.status === 'missing') {
      return <Alert type="warning" showIcon message={pairCheck.message} />
    }
    if (pairCheck.status === 'disabled') {
      return <Alert type="warning" showIcon message={pairCheck.message} />
    }
    if (pairCheck.status === 'enabled') {
      return <Alert type="success" showIcon message={pairCheck.message} />
    }
    return null
  }

  return (
    <PageContainer
      title={isEdit ? '编辑规则' : '创建规则'}
      description="支持单节点转发和隧道转发。"
      extra={
        <Button
          icon={<ArrowLeftOutlined />}
          onClick={() => navigate(buildPortalPath(portal, '/rules'))}
        >
          返回列表
        </Button>
      }
    >
      <Card>
        {loading ? (
          <Skeleton active paragraph={{ rows: 8 }} />
        ) : (
          <Form<RuleFormValues>
            form={form}
            layout="vertical"
            onFinish={(values) => void handleSubmit(values)}
          >
            <Form.Item
              label="转发模式"
              name="mode"
              rules={[{ required: true, message: '请选择转发模式' }]}
            >
              <Radio.Group>
                <Radio.Button value="single">单节点转发 (入口直出)</Radio.Button>
                <Radio.Button value="tunnel">隧道转发</Radio.Button>
              </Radio.Group>
            </Form.Item>

            <Form.Item
              label="入口节点"
              name="entry_node_id"
              rules={[{ required: true, message: '请选择入口节点' }]}
            >
              <Select
                placeholder="请选择入口节点"
                options={entryNodeOptions}
                filterOption={filterNodeOption}
                showSearch
              />
            </Form.Item>

            {mode === 'tunnel' ? (
              <Form.Item
                label="出口节点"
                name="exit_node_id"
                rules={[{ required: true, message: '隧道模式必须选择出口节点' }]}
              >
                <Select
                  placeholder="请选择出口节点"
                  options={exitNodeOptions}
                  filterOption={filterNodeOption}
                  showSearch
                />
              </Form.Item>
            ) : null}

            {renderPairCheckAlert()}

            <Form.Item
              label="规则名称"
              name="name"
              rules={[
                { required: true, message: '请输入规则名称' },
                { max: 100, message: '规则名称不能超过 100 个字符' },
              ]}
            >
              <Input placeholder="例如：Tokyo TCP Forward" />
            </Form.Item>

            <Form.Item
              label="协议"
              name="protocol"
              rules={[{ required: true, message: '请选择协议' }]}
            >
              <Select options={protocolOptions} />
            </Form.Item>

            <Form.Item
              label="目标地址"
              name="target_host"
              rules={[{ required: true, message: '请输入目标地址' }]}
            >
              <Input placeholder="例如：10.0.0.8 或 db.internal.local" />
            </Form.Item>

            <Form.Item
              label="目标端口"
              name="target_port"
              rules={[
                { required: true, message: '请输入目标端口' },
                { type: 'number', min: 1, max: 65535, message: '端口范围为 1-65535' },
              ]}
            >
              <InputNumber
                min={1}
                max={65535}
                precision={0}
                className="w-full"
                placeholder="1-65535"
              />
            </Form.Item>

            <Form.Item
              label="监听端口"
              name="listen_port"
              rules={[
                { required: true, message: '请输入监听端口' },
                { type: 'number', min: 1, max: 65535, message: '端口范围为 1-65535' },
              ]}
            >
              <InputNumber
                min={1}
                max={65535}
                precision={0}
                className="w-full"
                placeholder="1-65535"
              />
            </Form.Item>

            <Form.Item>
              <Space>
                <Button
                  type="primary"
                  htmlType="submit"
                  loading={submitting}
                >
                  {isEdit ? '保存规则' : '创建规则'}
                </Button>
                <Button onClick={() => navigate(buildPortalPath(portal, '/rules'))}>
                  取消
                </Button>
              </Space>
            </Form.Item>
          </Form>
        )}
      </Card>
    </PageContainer>
  )
}

export default RuleForm
