import { Card, Radio, Space, Tag, Typography, Alert } from 'antd'
import {
  ApiOutlined,
  ThunderboltOutlined,
  RocketOutlined,
  SafetyOutlined,
  CheckCircleOutlined
} from '@ant-design/icons'
import type { Tunnel } from '../../../types/nodeGroup'

interface ProtocolSelectorProps {
  value?: Tunnel['protocol']
  onChange?: (value: Tunnel['protocol']) => void
  showRecommendation?: boolean
  scenario?: 'web' | 'game' | 'file' | 'stream' | 'general'
}

const protocolOptions = [
  {
    value: 'tcp' as const,
    label: 'TCP',
    icon: <ApiOutlined />,
    color: '#1890ff',
    description: '可靠的面向连接传输协议',
    features: ['可靠传输', '有序交付', '错误检测'],
    scenarios: ['HTTP', '数据库', '文件传输'],
    performance: { latency: '中', reliability: '高', speed: '中' },
  },
  {
    value: 'udp' as const,
    label: 'UDP',
    icon: <ThunderboltOutlined />,
    color: '#52c41a',
    description: '无连接的快速传输协议',
    features: ['低延迟', '无连接开销', '适合实时'],
    scenarios: ['视频', '游戏', 'VoIP'],
    performance: { latency: '低', reliability: '中', speed: '高' },
  },
  {
    value: 'ws' as const,
    label: 'WebSocket',
    icon: <RocketOutlined />,
    color: '#722ed1',
    description: '全双工通信协议',
    features: ['双向通信', '低延迟', '浏览器支持'],
    scenarios: ['聊天', '推送', '协作'],
    performance: { latency: '低', reliability: '高', speed: '高' },
  },
  {
    value: 'wss' as const,
    label: 'WebSocket SSL',
    icon: <SafetyOutlined />,
    color: '#fa8c16',
    description: '加密的 WebSocket 连接',
    features: ['加密传输', '双向通信', '安全'],
    scenarios: ['安全聊天', '金融', '敏感数据'],
    performance: { latency: '中', reliability: '高', speed: '中' },
  },
  {
    value: 'tls' as const,
    label: 'TLS',
    icon: <SafetyOutlined />,
    color: '#fa541c',
    description: 'TLS 加密传输',
    features: ['端到端加密', '身份验证', '数据完整性'],
    scenarios: ['HTTPS', '安全传输', '加密通信'],
    performance: { latency: '中', reliability: '高', speed: '中' },
  },
  {
    value: 'quic' as const,
    label: 'QUIC',
    icon: <ThunderboltOutlined />,
    color: '#eb2f96',
    description: '快速可靠传输协议',
    features: ['0-RTT', '多路复用', '连接迁移'],
    scenarios: ['HTTP/3', '流媒体', '低延迟'],
    performance: { latency: '极低', reliability: '高', speed: '极高' },
  },
]

const getRecommendedProtocol = (scenario?: string): Tunnel['protocol'] | null => {
  switch (scenario) {
    case 'web':
      return 'wss'
    case 'game':
      return 'quic'
    case 'file':
      return 'tcp'
    case 'stream':
      return 'quic'
    default:
      return null
  }
}

const ProtocolSelector = ({ value, onChange, showRecommendation, scenario }: ProtocolSelectorProps) => {
  const recommended = showRecommendation ? getRecommendedProtocol(scenario) : null

  return (
    <Space direction="vertical" size="middle" style={{ width: '100%' }}>
      {recommended && (
        <Alert
          message={`推荐使用 ${protocolOptions.find(p => p.value === recommended)?.label} 协议`}
          description={`根据您的使用场景，${protocolOptions.find(p => p.value === recommended)?.label} 协议最适合`}
          type="info"
          showIcon
          icon={<CheckCircleOutlined />}
        />
      )}

      <Radio.Group value={value} onChange={(e) => onChange?.(e.target.value)} style={{ width: '100%' }}>
        <Space direction="vertical" size="small" style={{ width: '100%' }}>
          {protocolOptions.map((protocol) => (
            <Card
              key={protocol.value}
              size="small"
              hoverable
              style={{
                borderColor: value === protocol.value ? protocol.color : undefined,
                borderWidth: value === protocol.value ? 2 : 1,
              }}
            >
              <Radio value={protocol.value} style={{ width: '100%' }}>
                <Space direction="vertical" size={4} style={{ width: '100%' }}>
                  <Space>
                    <span style={{ color: protocol.color, fontSize: 18 }}>
                      {protocol.icon}
                    </span>
                    <Typography.Text strong style={{ fontSize: 16 }}>
                      {protocol.label}
                    </Typography.Text>
                    {recommended === protocol.value && (
                      <Tag color="green" icon={<CheckCircleOutlined />}>
                        推荐
                      </Tag>
                    )}
                  </Space>
                  <Typography.Text type="secondary" style={{ fontSize: 13 }}>
                    {protocol.description}
                  </Typography.Text>
                  <Space size={4} wrap>
                    {protocol.features.map((feature) => (
                      <Tag key={feature} color="blue" style={{ fontSize: 12 }}>
                        {feature}
                      </Tag>
                    ))}
                  </Space>
                  <Space size={8}>
                    <Typography.Text type="secondary" style={{ fontSize: 12 }}>
                      延迟: <Tag color="default">{protocol.performance.latency}</Tag>
                    </Typography.Text>
                    <Typography.Text type="secondary" style={{ fontSize: 12 }}>
                      可靠性: <Tag color="default">{protocol.performance.reliability}</Tag>
                    </Typography.Text>
                    <Typography.Text type="secondary" style={{ fontSize: 12 }}>
                      速度: <Tag color="default">{protocol.performance.speed}</Tag>
                    </Typography.Text>
                  </Space>
                  <Typography.Text type="secondary" style={{ fontSize: 12 }}>
                    适用: {protocol.scenarios.join('、')}
                  </Typography.Text>
                </Space>
              </Radio>
            </Card>
          ))}
        </Space>
      </Radio.Group>
    </Space>
  )
}

export default ProtocolSelector
