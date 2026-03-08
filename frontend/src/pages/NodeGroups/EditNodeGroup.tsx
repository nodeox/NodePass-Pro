import { ArrowLeftOutlined } from '@ant-design/icons'
import {
  Button,
  Card,
  Form,
  Input,
  InputNumber,
  Select,
  Space,
  Steps,
  Switch,
  message,
} from 'antd'
import { useCallback, useEffect, useMemo, useState } from 'react'
import { useNavigate, useParams } from 'react-router-dom'

import { nodeGroupApi } from '../../services/nodeGroupApi'
import type {
  CreateNodeGroupPayload,
  LoadBalanceStrategy,
  NodeGroupType,
} from '../../types/nodeGroup'
import { getErrorMessage } from '../../utils/error'

type FormValues = {
  name: string
  type: NodeGroupType
  description?: string
  allowed_protocols: string[]
  port_start: number
  port_end: number
  require_exit_group?: boolean
  traffic_multiplier?: number
  dns_load_balance?: boolean
  load_balance_strategy?: LoadBalanceStrategy
  health_check_interval?: number
  health_check_timeout?: number
}

const safeTrim = (value: unknown): string =>
  typeof value === 'string' ? value.trim() : ''

const stepItems = [
  { title: '基本信息' },
  { title: '协议与端口配置' },
  { title: '高级配置' },
]

const EditNodeGroup = () => {
  const navigate = useNavigate()
  const { id } = useParams<{ id: string }>()
  const groupID = Number(id)

  const [form] = Form.useForm<FormValues>()
  const [currentStep, setCurrentStep] = useState<number>(0)
  const [loading, setLoading] = useState<boolean>(false)
  const [submitting, setSubmitting] = useState<boolean>(false)

  const groupType = Form.useWatch('type', form) ?? 'entry'

  const initialValues = useMemo<FormValues>(
    () => ({
      name: '',
      type: 'entry',
      description: '',
      allowed_protocols: ['tcp', 'udp'],
      port_start: 1,
      port_end: 65535,
      require_exit_group: true,
      traffic_multiplier: 1,
      dns_load_balance: false,
      load_balance_strategy: 'round_robin',
      health_check_interval: 30,
      health_check_timeout: 5,
    }),
    [],
  )

  const loadDetail = useCallback(async () => {
    if (!Number.isFinite(groupID) || groupID <= 0) {
      message.error('无效的节点组 ID')
      return
    }

    setLoading(true)
    try {
      const detail = await nodeGroupApi.get(groupID)
      form.setFieldsValue({
        name: detail.name ?? '',
        type: detail.type,
        description: detail.description ?? '',
        allowed_protocols: detail.config?.allowed_protocols ?? ['tcp'],
        port_start: detail.config?.port_range?.start ?? 1,
        port_end: detail.config?.port_range?.end ?? 65535,
        require_exit_group: detail.config?.entry_config?.require_exit_group ?? true,
        traffic_multiplier: detail.config?.entry_config?.traffic_multiplier ?? 1,
        dns_load_balance: detail.config?.entry_config?.dns_load_balance ?? false,
        load_balance_strategy:
          detail.config?.exit_config?.load_balance_strategy ?? 'round_robin',
        health_check_interval: detail.config?.exit_config?.health_check_interval ?? 30,
        health_check_timeout: detail.config?.exit_config?.health_check_timeout ?? 5,
      })
    } catch (error) {
      message.error(getErrorMessage(error, '加载节点组详情失败'))
    } finally {
      setLoading(false)
    }
  }, [form, groupID])

  useEffect(() => {
    void loadDetail()
  }, [loadDetail])

  const handleNext = async () => {
    try {
      if (currentStep === 0) {
        await form.validateFields(['name', 'type', 'description'])
      }

      if (currentStep === 1) {
        await form.validateFields(['allowed_protocols', 'port_start', 'port_end'])
      }

      if (currentStep < stepItems.length - 1) {
        setCurrentStep((prev) => prev + 1)
      }
    } catch {
      // Antd Form 已高亮错误项
    }
  }

  const handlePrev = () => {
    if (currentStep > 0) {
      setCurrentStep((prev) => prev - 1)
    }
  }

  const handleSubmit = async () => {
    if (!Number.isFinite(groupID) || groupID <= 0) {
      message.error('无效的节点组 ID')
      return
    }

    try {
      await form.validateFields()
      setSubmitting(true)
      const values = form.getFieldsValue(true) as FormValues

      const name = safeTrim(values.name)
      if (!name) {
        message.error('节点组名称不能为空')
        return
      }

      const description = safeTrim(values.description)

      const payload: Partial<CreateNodeGroupPayload> = {
        name,
        description: description || undefined,
        config: {
          allowed_protocols: values.allowed_protocols,
          port_range: {
            start: values.port_start,
            end: values.port_end,
          },
          ...(values.type === 'entry'
            ? {
                entry_config: {
                  require_exit_group: values.require_exit_group ?? true,
                  traffic_multiplier: values.traffic_multiplier ?? 1,
                  dns_load_balance: values.dns_load_balance ?? false,
                },
              }
            : {
                exit_config: {
                  load_balance_strategy: values.load_balance_strategy ?? 'round_robin',
                  health_check_interval: values.health_check_interval ?? 30,
                  health_check_timeout: values.health_check_timeout ?? 5,
                },
              }),
        },
      }

      await nodeGroupApi.update(groupID, payload)
      message.success('节点组更新成功')
      navigate(`/node-groups/${groupID}`)
    } catch (error) {
      if (error instanceof Error && error.message.includes('out of date')) {
        return
      }
      message.error(getErrorMessage(error, '更新节点组失败'))
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <Card
      className="shadow-sm"
      loading={loading}
      title="编辑节点组"
      extra={
        <Button icon={<ArrowLeftOutlined />} onClick={() => navigate(`/node-groups/${groupID}`)}>
          返回
        </Button>
      }
    >
      <Space direction="vertical" size={20} className="w-full">
        <Steps current={currentStep} items={stepItems} />

        <Form<FormValues>
          form={form}
          layout="vertical"
          initialValues={initialValues}
          requiredMark="optional"
        >
          {currentStep === 0 ? (
            <>
              <Form.Item
                label="节点组名称"
                name="name"
                rules={[
                  { required: true, message: '请输入节点组名称' },
                  { max: 100, message: '节点组名称不能超过 100 字符' },
                ]}
              >
                <Input placeholder="请输入节点组名称" maxLength={100} />
              </Form.Item>

              <Form.Item
                label="节点组类型"
                name="type"
                rules={[{ required: true, message: '请选择节点组类型' }]}
              >
                <Select
                  disabled
                  options={[
                    { label: '入口组', value: 'entry' },
                    { label: '出口组', value: 'exit' },
                  ]}
                />
              </Form.Item>

              <Form.Item label="描述" name="description">
                <Input.TextArea rows={4} placeholder="可选" />
              </Form.Item>
            </>
          ) : null}

          {currentStep === 1 ? (
            <>
              <Form.Item
                label="允许协议"
                name="allowed_protocols"
                rules={[
                  { required: true, message: '请选择至少一种协议' },
                  {
                    validator: (_, value?: string[]) => {
                      if (Array.isArray(value) && value.length > 0) {
                        return Promise.resolve()
                      }
                      return Promise.reject(new Error('请选择至少一种协议'))
                    },
                  },
                ]}
              >
                <Select
                  mode="multiple"
                  options={[
                    { label: 'TCP', value: 'tcp' },
                    { label: 'UDP', value: 'udp' },
                  ]}
                />
              </Form.Item>

              <Space size={16} align="start" wrap>
                <Form.Item
                  label="端口范围起始"
                  name="port_start"
                  rules={[{ required: true, message: '请输入起始端口' }]}
                >
                  <InputNumber min={1} max={65535} precision={0} style={{ width: 180 }} />
                </Form.Item>

                <Form.Item
                  label="端口范围结束"
                  name="port_end"
                  rules={[
                    { required: true, message: '请输入结束端口' },
                    ({ getFieldValue }) => ({
                      validator(_, value?: number) {
                        const start = getFieldValue('port_start') as number | undefined
                        if (typeof value !== 'number' || typeof start !== 'number') {
                          return Promise.resolve()
                        }
                        if (value <= start) {
                          return Promise.reject(new Error('结束端口必须大于起始端口'))
                        }
                        if (value < 1 || value > 65535) {
                          return Promise.reject(new Error('端口范围必须在 1-65535'))
                        }
                        return Promise.resolve()
                      },
                    }),
                  ]}
                >
                  <InputNumber min={1} max={65535} precision={0} style={{ width: 180 }} />
                </Form.Item>
              </Space>
            </>
          ) : null}

          {currentStep === 2 ? (
            <>
              {groupType === 'entry' ? (
                <>
                  <Form.Item
                    label="是否要求出口组"
                    name="require_exit_group"
                    valuePropName="checked"
                  >
                    <Switch checkedChildren="是" unCheckedChildren="否" />
                  </Form.Item>

                  <Form.Item
                    label="流量倍率"
                    name="traffic_multiplier"
                    rules={[{ required: true, message: '请输入流量倍率' }]}
                  >
                    <InputNumber
                      min={0.1}
                      max={10}
                      step={0.1}
                      precision={1}
                      style={{ width: 180 }}
                    />
                  </Form.Item>

                  <Form.Item
                    label="DNS 负载均衡"
                    name="dns_load_balance"
                    valuePropName="checked"
                  >
                    <Switch checkedChildren="开启" unCheckedChildren="关闭" />
                  </Form.Item>
                </>
              ) : (
                <>
                  <Form.Item
                    label="负载均衡策略"
                    name="load_balance_strategy"
                    rules={[{ required: true, message: '请选择负载均衡策略' }]}
                  >
                    <Select
                      options={[
                        { label: '轮询 (round_robin)', value: 'round_robin' },
                        { label: '最少连接 (least_connections)', value: 'least_connections' },
                        { label: '随机 (random)', value: 'random' },
                      ]}
                    />
                  </Form.Item>

                  <Space size={16} align="start" wrap>
                    <Form.Item
                      label="健康检查间隔（秒）"
                      name="health_check_interval"
                      rules={[{ required: true, message: '请输入健康检查间隔' }]}
                    >
                      <InputNumber min={1} max={3600} precision={0} style={{ width: 200 }} />
                    </Form.Item>

                    <Form.Item
                      label="健康检查超时（秒）"
                      name="health_check_timeout"
                      rules={[{ required: true, message: '请输入健康检查超时' }]}
                    >
                      <InputNumber min={1} max={300} precision={0} style={{ width: 200 }} />
                    </Form.Item>
                  </Space>
                </>
              )}
            </>
          ) : null}
        </Form>

        <Space>
          <Button disabled={currentStep === 0} onClick={handlePrev}>
            上一步
          </Button>

          {currentStep < stepItems.length - 1 ? (
            <Button type="primary" onClick={() => void handleNext()}>
              下一步
            </Button>
          ) : (
            <Button type="primary" loading={submitting} onClick={() => void handleSubmit()}>
              保存修改
            </Button>
          )}
        </Space>
      </Space>
    </Card>
  )
}

export default EditNodeGroup
