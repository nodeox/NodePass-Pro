import { ArrowLeftOutlined, PauseCircleOutlined, PlayCircleOutlined, ReloadOutlined } from '@ant-design/icons'
import { Button, Card, Descriptions, Result, Space, Spin, Tag, Typography, message } from 'antd'
import { useCallback, useEffect, useMemo, useState } from 'react'
import { useNavigate, useParams } from 'react-router-dom'

import PageContainer from '../../components/common/PageContainer'
import { tunnelApi } from '../../services/nodeGroupApi'
import type { Tunnel } from '../../types/nodeGroup'
import { getErrorMessage } from '../../utils/error'
import { formatDateTime } from '../../utils/format'

const TunnelDetail = () => {
  const navigate = useNavigate()
  const { id } = useParams()
  const tunnelID = Number(id)

  const [loading, setLoading] = useState<boolean>(false)
  const [updating, setUpdating] = useState<boolean>(false)
  const [tunnel, setTunnel] = useState<Tunnel | null>(null)

  const loadTunnel = useCallback(async () => {
    if (!Number.isFinite(tunnelID) || tunnelID <= 0) {
      return
    }
    setLoading(true)
    try {
      const detail = await tunnelApi.get(tunnelID)
      setTunnel(detail)
    } catch (error) {
      message.error(getErrorMessage(error, '隧道详情加载失败'))
    } finally {
      setLoading(false)
    }
  }, [tunnelID])

  useEffect(() => {
    void loadTunnel()
  }, [loadTunnel])

  const statusTag = useMemo(() => {
    if (!tunnel) {
      return null
    }
    const colorMap: Record<string, string> = {
      running: 'green',
      stopped: 'default',
      error: 'red',
      paused: 'orange',
    }
    return <Tag color={colorMap[tunnel.status] || 'default'}>{tunnel.status}</Tag>
  }, [tunnel])

  const handleToggle = async () => {
    if (!tunnel) {
      return
    }
    setUpdating(true)
    try {
      if (tunnel.status === 'running') {
        await tunnelApi.stop(tunnel.id)
        message.success('隧道已停止')
      } else {
        await tunnelApi.start(tunnel.id)
        message.success('隧道已启动')
      }
      await loadTunnel()
    } catch (error) {
      message.error(getErrorMessage(error, '隧道状态更新失败'))
    } finally {
      setUpdating(false)
    }
  }

  if (!Number.isFinite(tunnelID) || tunnelID <= 0) {
    return (
      <Result
        status="404"
        title="无效的隧道 ID"
        extra={
          <Button type="primary" onClick={() => navigate(-1)}>
            返回
          </Button>
        }
      />
    )
  }

  return (
    <PageContainer
      title={tunnel ? `隧道详情 · ${tunnel.name}` : '隧道详情'}
      description="查看隧道状态、组信息与目标配置。"
      extra={
        <Space>
          <Button icon={<ArrowLeftOutlined />} onClick={() => navigate(-1)}>
            返回
          </Button>
          <Button icon={<ReloadOutlined />} onClick={() => void loadTunnel()} loading={loading}>
            刷新
          </Button>
          {tunnel ? (
            <Button
              type={tunnel.status === 'running' ? 'default' : 'primary'}
              icon={tunnel.status === 'running' ? <PauseCircleOutlined /> : <PlayCircleOutlined />}
              onClick={() => void handleToggle()}
              loading={updating}
            >
              {tunnel.status === 'running' ? '停止' : '启动'}
            </Button>
          ) : null}
        </Space>
      }
    >
      <Spin spinning={loading}>
        {tunnel ? (
          <Card>
            <Descriptions bordered column={2} size="middle">
              <Descriptions.Item label="ID">{tunnel.id}</Descriptions.Item>
              <Descriptions.Item label="状态">{statusTag}</Descriptions.Item>
              <Descriptions.Item label="名称">{tunnel.name}</Descriptions.Item>
              <Descriptions.Item label="协议">
                <Tag color="blue">{tunnel.protocol.toUpperCase()}</Tag>
              </Descriptions.Item>
              <Descriptions.Item label="入口组">
                {tunnel.entry_group?.name || `#${tunnel.entry_group_id}`}
              </Descriptions.Item>
              <Descriptions.Item label="出口组">
                {tunnel.exit_group_id ? tunnel.exit_group?.name || `#${tunnel.exit_group_id}` : '直连'}
              </Descriptions.Item>
              <Descriptions.Item label="监听地址">
                {`${tunnel.listen_host}:${tunnel.listen_port || '自动'}`}
              </Descriptions.Item>
              <Descriptions.Item label="远程目标">
                {`${tunnel.remote_host}:${tunnel.remote_port}`}
              </Descriptions.Item>
              <Descriptions.Item label="上行流量">{tunnel.traffic_in}</Descriptions.Item>
              <Descriptions.Item label="下行流量">{tunnel.traffic_out}</Descriptions.Item>
              <Descriptions.Item label="创建时间">
                {formatDateTime(tunnel.created_at)}
              </Descriptions.Item>
              <Descriptions.Item label="更新时间">
                {formatDateTime(tunnel.updated_at)}
              </Descriptions.Item>
              <Descriptions.Item label="说明" span={2}>
                <Typography.Text>{tunnel.description || '-'}</Typography.Text>
              </Descriptions.Item>
            </Descriptions>
          </Card>
        ) : (
          <Result status="404" title="未找到隧道" />
        )}
      </Spin>
    </PageContainer>
  )
}

export default TunnelDetail
