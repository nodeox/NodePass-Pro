import { CopyOutlined, ExclamationCircleOutlined } from '@ant-design/icons'
import {
  Alert,
  Button,
  Form,
  Input,
  InputNumber,
  Modal,
  Select,
  Space,
  Switch,
  Typography,
  message,
} from 'antd'
import { useEffect, useMemo, useState } from 'react'

import { nodeApi } from '../../services/api'
import type {
  CreateNodeResult,
  CreateNodePayload,
  NodeQuotaInfo,
  NodeRecord,
  UpdateNodePayload,
} from '../../types'
import { getErrorMessage } from '../../utils/error'

type NodeFormMode = 'create' | 'edit'

type NodeFormValues = {
  name: string
  host: string
  port?: number
  region?: string
  is_self_hosted: boolean
  description?: string
}

export type NodeFormProps = {
  open: boolean
  mode: NodeFormMode
  initialData?: NodeRecord | null
  quota?: NodeQuotaInfo | null
  onClose: () => void
  onSuccess: () => Promise<void> | void
}

const regionOptions = [
  { label: '美国西部', value: 'us-west' },
  { label: '美国东部', value: 'us-east' },
  { label: '欧洲', value: 'europe' },
  { label: '亚洲东部', value: 'asia-east' },
  { label: '亚洲东南', value: 'asia-se' },
  { label: '中国香港', value: 'hk' },
]

const copyToClipboard = async (
  content: string,
  successMessage: string,
): Promise<void> => {
  if (!navigator.clipboard) {
    message.warning('当前环境不支持剪贴板')
    return
  }

  try {
    await navigator.clipboard.writeText(content)
    message.success(successMessage)
  } catch (_error) {
    message.error('复制失败，请手动复制')
  }
}

const normalizeText = (value?: string): string | undefined => {
  if (!value) {
    return undefined
  }
  const trimmed = value.trim()
  return trimmed === '' ? undefined : trimmed
}

const NodeForm = ({
  open,
  mode,
  initialData,
  quota,
  onClose,
  onSuccess,
}: NodeFormProps) => {
  const [form] = Form.useForm<NodeFormValues>()
  const isSelfHosted = Form.useWatch('is_self_hosted', form)
  const [submitting, setSubmitting] = useState<boolean>(false)
  const [secretInfo, setSecretInfo] = useState<CreateNodeResult | null>(null)

  useEffect(() => {
    if (!open) {
      return
    }

    if (mode === 'edit' && initialData) {
      form.setFieldsValue({
        name: initialData.name,
        host: initialData.host,
        port: initialData.port,
        region: initialData.region ?? undefined,
        is_self_hosted: initialData.is_self_hosted,
        description: initialData.description ?? undefined,
      })
      return
    }

    form.setFieldsValue({
      name: '',
      host: '',
      port: undefined,
      region: undefined,
      is_self_hosted: false,
      description: '',
    })
  }, [form, initialData, mode, open])

  const quotaText = useMemo(() => {
    if (!quota) {
      return '配额加载中...'
    }

    const used = quota.used_self_hosted_nodes ?? 0
    return `入口: ${used}/${quota.max_self_hosted_entry_nodes}，出口: ${used}/${quota.max_self_hosted_exit_nodes}，剩余: ${quota.remaining_self_hosted_quota}`
  }, [quota])

  const handleSubmit = async (values: NodeFormValues): Promise<void> => {
    const port = values.port ?? 0
    setSubmitting(true)
    try {
      if (mode === 'create') {
        const payload: CreateNodePayload = {
          name: values.name.trim(),
          host: values.host.trim(),
          port,
          region: normalizeText(values.region),
          is_self_hosted: values.is_self_hosted,
          is_public: false,
          description: normalizeText(values.description),
        }
        const result = await nodeApi.create(payload)
        setSecretInfo(result)
        message.success('节点创建成功')
      } else {
        if (!initialData) {
          return
        }

        const payload: UpdateNodePayload = {
          name: values.name.trim(),
          host: values.host.trim(),
          port,
          region: normalizeText(values.region),
          is_self_hosted: values.is_self_hosted,
          description: normalizeText(values.description),
        }
        await nodeApi.update(initialData.id, payload)
        message.success('节点更新成功')
      }

      onClose()
      form.resetFields()
      await Promise.resolve(onSuccess())
    } catch (error) {
      message.error(getErrorMessage(error, '节点保存失败'))
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <>
      <Modal
        title={mode === 'create' ? '创建节点' : '编辑节点'}
        open={open}
        onCancel={onClose}
        onOk={() => void form.submit()}
        okText={mode === 'create' ? '创建' : '保存'}
        confirmLoading={submitting}
        destroyOnClose
      >
        <Form<NodeFormValues>
          form={form}
          layout="vertical"
          onFinish={(values) => void handleSubmit(values)}
          preserve={false}
        >
          <Form.Item
            label="节点名称"
            name="name"
            rules={[
              { required: true, message: '请输入节点名称' },
              { max: 100, message: '节点名称不能超过 100 个字符' },
            ]}
          >
            <Input placeholder="例如：美国西部入口节点" />
          </Form.Item>

          <Form.Item
            label="主机地址"
            name="host"
            rules={[{ required: true, message: '请输入主机地址' }]}
          >
            <Input placeholder="例如：127.0.0.1 / node.example.com" />
          </Form.Item>

          <Form.Item
            label="端口"
            name="port"
            rules={[
              { required: true, message: '请输入端口' },
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

          <Form.Item label="区域" name="region">
            <Select
              allowClear
              placeholder="请选择区域"
              options={regionOptions}
            />
          </Form.Item>

          <Form.Item
            label="自托管节点"
            name="is_self_hosted"
            valuePropName="checked"
          >
            <Switch />
          </Form.Item>

          {isSelfHosted ? (
            <Alert
              type="info"
              showIcon
              icon={<ExclamationCircleOutlined />}
              message="自托管节点配额提示"
              description={quotaText}
              style={{ marginBottom: 16 }}
            />
          ) : null}

          <Form.Item label="描述" name="description">
            <Input.TextArea rows={3} placeholder="可选：记录节点用途、网络环境等" />
          </Form.Item>
        </Form>
      </Modal>

      <Modal
        title="节点初始化信息"
        open={Boolean(secretInfo)}
        onCancel={() => setSecretInfo(null)}
        footer={[
          <Button key="ok" type="primary" onClick={() => setSecretInfo(null)}>
            我已保存
          </Button>,
        ]}
      >
        <Space direction="vertical" size={16} className="w-full">
          <Alert
            type="warning"
            showIcon
            message="Token 仅显示一次，请妥善保存"
          />

          <div>
            <Typography.Text strong>节点 Token</Typography.Text>
            <Space.Compact className="w-full" style={{ marginTop: 8 }}>
              <Input value={secretInfo?.token ?? ''} readOnly />
              <Button
                icon={<CopyOutlined />}
                onClick={() =>
                  void copyToClipboard(secretInfo?.token ?? '', 'Token 已复制')
                }
              >
                复制
              </Button>
            </Space.Compact>
          </div>

          <div>
            <Typography.Text strong>一键安装命令</Typography.Text>
            <div
              style={{
                marginTop: 8,
                padding: 12,
                borderRadius: 8,
                background: '#0f172a',
                color: '#e2e8f0',
                fontFamily: 'monospace',
                fontSize: 12,
                whiteSpace: 'pre-wrap',
                wordBreak: 'break-all',
              }}
            >
              {secretInfo?.install_command ?? ''}
            </div>
            <Button
              style={{ marginTop: 8 }}
              icon={<CopyOutlined />}
              onClick={() =>
                void copyToClipboard(
                  secretInfo?.install_command ?? '',
                  '安装命令已复制',
                )
              }
            >
              复制安装命令
            </Button>
          </div>
        </Space>
      </Modal>
    </>
  )
}

export default NodeForm
