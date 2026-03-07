import {
  DeleteOutlined,
  EditOutlined,
  PlayCircleOutlined,
  RedoOutlined,
  ReloadOutlined,
  StopOutlined,
} from '@ant-design/icons'
import {
  Button,
  Popconfirm,
  Select,
  Space,
  Table,
  Tag,
  Tooltip,
  Typography,
  message,
} from 'antd'
import { useCallback, useEffect, useState } from 'react'
import { useLocation, useNavigate } from 'react-router-dom'

import PageContainer from '../../components/common/PageContainer'
import { usePageTitle } from '../../hooks/usePageTitle'
import { ruleApi } from '../../services/api'
import { useAppStore } from '../../store/app'
import type { RuleRecord } from '../../types'
import { getErrorMessage } from '../../utils/error'
import { formatBytes } from '../../utils/format'
import { buildPortalPath, resolvePortalByPathname } from '../../utils/route'

const modeMap: Record<
  string,
  { label: string; color: 'blue' | 'purple' | 'default' }
> = {
  single: { label: '单节点', color: 'blue' },
  tunnel: { label: '隧道', color: 'purple' },
}

const statusMap: Record<
  string,
  { label: string; color: 'green' | 'orange' | 'default' }
> = {
  running: { label: '运行中', color: 'green' },
  stopped: { label: '已停止', color: 'default' },
  paused: { label: '已暂停', color: 'orange' },
}

const protocolLabelMap: Record<string, string> = {
  tcp: 'TCP',
  udp: 'UDP',
  ws: 'WebSocket',
  tls: 'TLS',
  quic: 'QUIC',
}

const renderModeTag = (mode: string) => {
  const mapped = modeMap[mode]
  if (!mapped) {
    return <Tag>{mode}</Tag>
  }
  return <Tag color={mapped.color}>{mapped.label}</Tag>
}

const renderStatusTag = (status: string) => {
  const mapped = statusMap[status]
  if (!mapped) {
    return <Tag>{status}</Tag>
  }
  return <Tag color={mapped.color}>{mapped.label}</Tag>
}

const RuleList = () => {
  usePageTitle('规则管理')
  const navigate = useNavigate()
  const location = useLocation()
  const portal = resolvePortalByPathname(location.pathname)

  const [rules, setRules] = useState<RuleRecord[]>([])
  const [loading, setLoading] = useState<boolean>(false)
  const [actionLoadingKey, setActionLoadingKey] = useState<string | null>(null)

  const [statusFilter, setStatusFilter] = useState<string | undefined>()
  const [modeFilter, setModeFilter] = useState<'single' | 'tunnel' | undefined>()

  const [page, setPage] = useState<number>(1)
  const [pageSize, setPageSize] = useState<number>(10)
  const [total, setTotal] = useState<number>(0)
  const ruleStatusMap = useAppStore((state) => state.ruleStatusMap)
  const nodeStatusMap = useAppStore((state) => state.nodeStatusMap)

  const loadRules = useCallback(
    async (targetPage = page, targetPageSize = pageSize): Promise<void> => {
      setLoading(true)
      try {
        const result = await ruleApi.list({
          page: targetPage,
          pageSize: targetPageSize,
          status: statusFilter,
          mode: modeFilter,
        })
        setRules(result.list ?? [])
        setTotal(result.total ?? 0)
        setPage(result.page || targetPage)
        setPageSize(result.page_size || targetPageSize)
      } catch (error) {
        message.error(getErrorMessage(error, '规则列表加载失败'))
      } finally {
        setLoading(false)
      }
    },
    [modeFilter, page, pageSize, statusFilter],
  )

  useEffect(() => {
    void loadRules()
  }, [loadRules])

  useEffect(() => {
    const ids = Object.keys(ruleStatusMap)
    if (ids.length === 0) {
      return
    }

    setRules((current) => {
      let changed = false
      const next = current.map((rule) => {
        const nextStatus = ruleStatusMap[rule.id]
        if (!nextStatus || nextStatus === rule.status) {
          return rule
        }
        changed = true
        return {
          ...rule,
          status: nextStatus,
        }
      })
      return changed ? next : current
    })
  }, [ruleStatusMap])

  useEffect(() => {
    const ids = Object.keys(nodeStatusMap)
    if (ids.length === 0) {
      return
    }

    setRules((current) => {
      let changed = false
      const next = current.map((rule) => {
        const entryStatus = rule.entry_node ? nodeStatusMap[rule.entry_node.id] : undefined
        const exitStatus = rule.exit_node ? nodeStatusMap[rule.exit_node.id] : undefined

        if (!entryStatus && !exitStatus) {
          return rule
        }

        const nextRule: RuleRecord = {
          ...rule,
          entry_node:
            entryStatus && rule.entry_node
              ? {
                  ...rule.entry_node,
                  status: entryStatus,
                }
              : rule.entry_node,
          exit_node:
            exitStatus && rule.exit_node
              ? {
                  ...rule.exit_node,
                  status: exitStatus,
                }
              : rule.exit_node,
        }

        if (
          nextRule.entry_node?.status !== rule.entry_node?.status ||
          nextRule.exit_node?.status !== rule.exit_node?.status
        ) {
          changed = true
          return nextRule
        }
        return rule
      })

      return changed ? next : current
    })
  }, [nodeStatusMap])

  const executeAction = async (
    key: string,
    action: () => Promise<unknown>,
    successMessage: string,
  ): Promise<void> => {
    setActionLoadingKey(key)
    try {
      await action()
      message.success(successMessage)
      await loadRules()
    } catch (error) {
      message.error(getErrorMessage(error, '规则操作失败'))
    } finally {
      setActionLoadingKey(null)
    }
  }

  return (
    <PageContainer
      title="规则管理"
      description="管理单节点与隧道转发规则，支持启动/停止/重启。"
      extra={
        <Space wrap>
          <Button
            icon={<ReloadOutlined />}
            loading={loading}
            onClick={() => void loadRules()}
          >
            刷新
          </Button>
          <Button
            type="primary"
            onClick={() => navigate(buildPortalPath(portal, '/rules/new'))}
          >
            创建规则
          </Button>
        </Space>
      }
    >
      <Space wrap style={{ marginBottom: 16 }}>
        <Select
          allowClear
          style={{ minWidth: 160 }}
          placeholder="按状态过滤"
          value={statusFilter}
          options={[
            { label: '运行中', value: 'running' },
            { label: '已停止', value: 'stopped' },
            { label: '已暂停', value: 'paused' },
          ]}
          onChange={(value) => {
            setStatusFilter(value)
            setPage(1)
          }}
        />
        <Select
          allowClear
          style={{ minWidth: 160 }}
          placeholder="按模式过滤"
          value={modeFilter}
          options={[
            { label: '单节点', value: 'single' },
            { label: '隧道', value: 'tunnel' },
          ]}
          onChange={(value) => {
            setModeFilter(value)
            setPage(1)
          }}
        />
      </Space>

      <Table<RuleRecord>
        rowKey="id"
        loading={loading}
        dataSource={rules}
        scroll={{ x: 1500 }}
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
          {
            title: 'ID',
            dataIndex: 'id',
            width: 80,
          },
          {
            title: '名称',
            dataIndex: 'name',
            width: 160,
          },
          {
            title: '转发模式',
            dataIndex: 'mode',
            width: 120,
            render: (mode: string) => renderModeTag(mode),
          },
          {
            title: '协议',
            dataIndex: 'protocol',
            width: 120,
            render: (protocol: string) => protocolLabelMap[protocol] ?? protocol,
          },
          {
            title: '入口节点',
            width: 180,
            render: (_, record) => (
              <Space size={4}>
                <span>{record.entry_node?.name ?? `ID:${record.entry_node_id}`}</span>
                {record.entry_node ? renderStatusTag(record.entry_node.status) : null}
              </Space>
            ),
          },
          {
            title: '出口节点',
            width: 180,
            render: (_, record) => {
              if (record.mode !== 'tunnel') {
                return <Typography.Text type="secondary">-</Typography.Text>
              }
              return (
                <Space size={4}>
                  <span>{record.exit_node?.name ?? `ID:${record.exit_node_id ?? '-'}`}</span>
                  {record.exit_node ? renderStatusTag(record.exit_node.status) : null}
                </Space>
              )
            },
          },
          {
            title: '目标地址:端口',
            width: 180,
            render: (_, record) => `${record.target_host}:${record.target_port}`,
          },
          {
            title: '监听端口',
            width: 160,
            render: (_, record) => `${record.listen_host}:${record.listen_port}`,
          },
          {
            title: '状态',
            dataIndex: 'status',
            width: 120,
            render: (status: string) => renderStatusTag(status),
          },
          {
            title: '流量(上/下)',
            width: 200,
            render: (_, record) =>
              `${formatBytes(record.traffic_in ?? 0)} / ${formatBytes(record.traffic_out ?? 0)}`,
          },
          {
            title: '操作',
            fixed: 'right',
            width: 320,
            render: (_, record) => (
              <Space size={4}>
                {record.status === 'running' ? (
                  <>
                    <Button
                      type="link"
                      icon={<StopOutlined />}
                      loading={actionLoadingKey === `stop-${record.id}`}
                      onClick={() =>
                        void executeAction(
                          `stop-${record.id}`,
                          () => ruleApi.stop(record.id),
                          '规则已停止',
                        )
                      }
                    >
                      停止
                    </Button>
                    <Button
                      type="link"
                      icon={<RedoOutlined />}
                      loading={actionLoadingKey === `restart-${record.id}`}
                      onClick={() =>
                        void executeAction(
                          `restart-${record.id}`,
                          () => ruleApi.restart(record.id),
                          '规则已重启',
                        )
                      }
                    >
                      重启
                    </Button>
                  </>
                ) : (
                  <Button
                    type="link"
                    icon={<PlayCircleOutlined />}
                    loading={actionLoadingKey === `start-${record.id}`}
                    onClick={() =>
                      void executeAction(
                        `start-${record.id}`,
                        () => ruleApi.start(record.id),
                        '规则已启动',
                      )
                    }
                  >
                    启动
                  </Button>
                )}

                <Tooltip
                  title={record.status === 'stopped' ? '' : '仅 stopped 状态可编辑'}
                >
                  <Button
                    type="link"
                    icon={<EditOutlined />}
                    disabled={record.status !== 'stopped'}
                    onClick={() =>
                      navigate(buildPortalPath(portal, `/rules/${record.id}/edit`))
                    }
                  >
                    编辑
                  </Button>
                </Tooltip>

                <Popconfirm
                  title="确定删除该规则吗？"
                  okText="删除"
                  cancelText="取消"
                  onConfirm={() =>
                    void executeAction(
                      `delete-${record.id}`,
                      () => ruleApi.remove(record.id),
                      '规则已删除',
                    )
                  }
                >
                  <Button
                    type="link"
                    danger
                    icon={<DeleteOutlined />}
                    loading={actionLoadingKey === `delete-${record.id}`}
                  >
                    删除
                  </Button>
                </Popconfirm>
              </Space>
            ),
          },
        ]}
      />
    </PageContainer>
  )
}

export default RuleList
