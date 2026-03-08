import { Card, Form, Input, InputNumber, Switch, Select, Space, Divider, Alert, Collapse } from 'antd'
import { InfoCircleOutlined } from '@ant-design/icons'
import type { Tunnel } from '../../../types/nodeGroup'

interface ProtocolConfigProps {
  protocol: Tunnel['protocol']
}

const ProtocolConfig = ({ protocol }: ProtocolConfigProps) => {
  const getProtocolDescription = () => {
    switch (protocol) {
      case 'tcp':
        return 'TCP 是可靠的、面向连接的传输协议，适用于需要可靠传输的应用。'
      case 'udp':
        return 'UDP 是无连接的传输协议，适用于实时性要求高、可容忍少量丢包的应用。'
      case 'ws':
        return 'WebSocket 提供全双工通信，适用于需要实时双向通信的 Web 应用。'
      case 'wss':
        return 'WebSocket over SSL/TLS，提供加密的 WebSocket 连接，更安全。'
      case 'tls':
        return 'TLS 加密传输，提供端到端加密，保护数据传输安全。'
      case 'quic':
        return 'QUIC 是基于 UDP 的快速可靠传输协议，具有低延迟和多路复用特性。'
      default:
        return ''
    }
  }

  const renderTCPConfig = () => (
    <Space direction="vertical" style={{ width: '100%' }}>
      <Form.Item
        label="TCP Keep-Alive"
        name={['protocol_config', 'tcp_keepalive']}
        valuePropName="checked"
        tooltip="启用 TCP Keep-Alive 以检测死连接"
      >
        <Switch />
      </Form.Item>
      <Form.Item
        label="Keep-Alive 间隔"
        name={['protocol_config', 'keepalive_interval']}
        tooltip="Keep-Alive 探测间隔（秒）"
      >
        <InputNumber min={1} max={300} placeholder="60" style={{ width: '100%' }} />
      </Form.Item>
      <Form.Item
        label="连接超时"
        name={['protocol_config', 'connect_timeout']}
        tooltip="建立连接的超时时间（秒）"
      >
        <InputNumber min={1} max={60} placeholder="10" style={{ width: '100%' }} />
      </Form.Item>
      <Form.Item
        label="读取超时"
        name={['protocol_config', 'read_timeout']}
        tooltip="读取数据的超时时间（秒）"
      >
        <InputNumber min={1} max={300} placeholder="30" style={{ width: '100%' }} />
      </Form.Item>
    </Space>
  )

  const renderUDPConfig = () => (
    <Space direction="vertical" style={{ width: '100%' }}>
      <Form.Item
        label="缓冲区大小"
        name={['protocol_config', 'buffer_size']}
        tooltip="UDP 接收缓冲区大小（字节）"
      >
        <InputNumber min={1024} max={65536} placeholder="8192" style={{ width: '100%' }} />
      </Form.Item>
      <Form.Item
        label="会话超时"
        name={['protocol_config', 'session_timeout']}
        tooltip="UDP 会话超时时间（秒）"
      >
        <InputNumber min={10} max={600} placeholder="60" style={{ width: '100%' }} />
      </Form.Item>
    </Space>
  )

  const renderWebSocketConfig = () => (
    <Space direction="vertical" style={{ width: '100%' }}>
      <Form.Item
        label="WebSocket 路径"
        name={['protocol_config', 'ws_path']}
        tooltip="WebSocket 连接路径"
      >
        <Input placeholder="/ws" />
      </Form.Item>
      <Form.Item
        label="心跳间隔"
        name={['protocol_config', 'ping_interval']}
        tooltip="WebSocket Ping 间隔（秒）"
      >
        <InputNumber min={5} max={300} placeholder="30" style={{ width: '100%' }} />
      </Form.Item>
      <Form.Item
        label="最大消息大小"
        name={['protocol_config', 'max_message_size']}
        tooltip="最大消息大小（KB）"
      >
        <InputNumber min={1} max={10240} placeholder="1024" style={{ width: '100%' }} />
      </Form.Item>
      <Form.Item
        label="压缩"
        name={['protocol_config', 'compression']}
        valuePropName="checked"
        tooltip="启用 WebSocket 压缩"
      >
        <Switch />
      </Form.Item>
    </Space>
  )

  const renderTLSConfig = () => (
    <Space direction="vertical" style={{ width: '100%' }}>
      <Form.Item
        label="TLS 版本"
        name={['protocol_config', 'tls_version']}
        tooltip="最低 TLS 版本"
      >
        <Select
          placeholder="TLS 1.2"
          options={[
            { label: 'TLS 1.2', value: 'tls1.2' },
            { label: 'TLS 1.3', value: 'tls1.3' },
          ]}
        />
      </Form.Item>
      <Form.Item
        label="证书验证"
        name={['protocol_config', 'verify_cert']}
        valuePropName="checked"
        tooltip="验证服务器证书"
      >
        <Switch defaultChecked />
      </Form.Item>
      <Form.Item
        label="SNI"
        name={['protocol_config', 'sni']}
        tooltip="Server Name Indication"
      >
        <Input placeholder="example.com" />
      </Form.Item>
    </Space>
  )

  const renderQUICConfig = () => (
    <Space direction="vertical" style={{ width: '100%' }}>
      <Form.Item
        label="最大流数"
        name={['protocol_config', 'max_streams']}
        tooltip="最大并发流数量"
      >
        <InputNumber min={1} max={1000} placeholder="100" style={{ width: '100%' }} />
      </Form.Item>
      <Form.Item
        label="初始窗口大小"
        name={['protocol_config', 'initial_window']}
        tooltip="初始流控窗口大小（KB）"
      >
        <InputNumber min={16} max={1024} placeholder="256" style={{ width: '100%' }} />
      </Form.Item>
      <Form.Item
        label="空闲超时"
        name={['protocol_config', 'idle_timeout']}
        tooltip="连接空闲超时时间（秒）"
      >
        <InputNumber min={10} max={600} placeholder="30" style={{ width: '100%' }} />
      </Form.Item>
      <Form.Item
        label="0-RTT"
        name={['protocol_config', 'enable_0rtt']}
        valuePropName="checked"
        tooltip="启用 0-RTT 快速握手"
      >
        <Switch />
      </Form.Item>
    </Space>
  )

  const renderProtocolConfig = () => {
    switch (protocol) {
      case 'tcp':
        return renderTCPConfig()
      case 'udp':
        return renderUDPConfig()
      case 'ws':
      case 'wss':
        return renderWebSocketConfig()
      case 'tls':
        return renderTLSConfig()
      case 'quic':
        return renderQUICConfig()
      default:
        return null
    }
  }

  return (
    <Card size="small" style={{ marginTop: 16 }}>
      <Collapse
        items={[
          {
            key: 'protocol-config',
            label: `${protocol.toUpperCase()} 协议配置（可选）`,
            children: (
              <Space direction="vertical" size="middle" style={{ width: '100%' }}>
                <Alert
                  message={getProtocolDescription()}
                  type="info"
                  icon={<InfoCircleOutlined />}
                  showIcon
                />
                <Divider style={{ margin: '12px 0' }} />
                {renderProtocolConfig()}
              </Space>
            ),
          },
        ]}
      />
    </Card>
  )
}

export default ProtocolConfig
