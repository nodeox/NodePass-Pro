import { message, notification } from 'antd'
import { create } from 'zustand'

import type { AppNotification, WebSocketEventMessage } from '../types'

const MAX_NOTIFICATIONS = 100

const toNumber = (value: unknown): number | null => {
  if (typeof value === 'number' && Number.isFinite(value)) {
    return value
  }
  if (typeof value === 'string' && value.trim() !== '') {
    const parsed = Number(value)
    return Number.isFinite(parsed) ? parsed : null
  }
  return null
}

const toText = (value: unknown): string | null => {
  if (typeof value === 'string') {
    const trimmed = value.trim()
    return trimmed === '' ? null : trimmed
  }
  if (typeof value === 'number' || typeof value === 'boolean') {
    return String(value)
  }
  return null
}

const getPayload = (
  event: WebSocketEventMessage,
): Record<string, unknown> => {
  if (event.data && typeof event.data === 'object') {
    return event.data
  }
  return {}
}

const createNotification = (params: {
  type: string
  title: string
  content: string
  payload?: Record<string, unknown>
  createdAt?: string
}): AppNotification => ({
  id: `${Date.now()}-${Math.random().toString(36).slice(2, 10)}`,
  type: params.type,
  title: params.title,
  content: params.content,
  created_at: params.createdAt ?? new Date().toISOString(),
  read: false,
  payload: params.payload,
})

interface AppState {
  siderCollapsed: boolean
  wsConnected: boolean
  notifications: AppNotification[]
  notificationCount: number
  nodeStatusMap: Record<number, string>
  ruleStatusMap: Record<number, string>
  setSiderCollapsed: (collapsed: boolean) => void
  setWsConnected: (connected: boolean) => void
  setNotificationCount: (count: number) => void
  setNodeStatus: (nodeID: number, status: string) => void
  setRuleStatus: (ruleID: number, status: string) => void
  markAllNotificationsRead: () => void
  clearNotifications: () => void
  handleWebSocketMessage: (event: WebSocketEventMessage) => void
}

export const useAppStore = create<AppState>((set) => ({
  siderCollapsed: false,
  wsConnected: false,
  notifications: [],
  notificationCount: 0,
  nodeStatusMap: {},
  ruleStatusMap: {},
  setSiderCollapsed: (collapsed) => set({ siderCollapsed: collapsed }),
  setWsConnected: (connected) => set({ wsConnected: connected }),
  setNotificationCount: (count) => set({ notificationCount: count }),
  setNodeStatus: (nodeID, status) =>
    set((state) => ({
      nodeStatusMap: {
        ...state.nodeStatusMap,
        [nodeID]: status,
      },
    })),
  setRuleStatus: (ruleID, status) =>
    set((state) => ({
      ruleStatusMap: {
        ...state.ruleStatusMap,
        [ruleID]: status,
      },
    })),
  markAllNotificationsRead: () =>
    set((state) => ({
      notifications: state.notifications.map((item) =>
        item.read ? item : { ...item, read: true },
      ),
      notificationCount: 0,
    })),
  clearNotifications: () =>
    set({
      notifications: [],
      notificationCount: 0,
    }),
  handleWebSocketMessage: (event) => {
    const payload = getPayload(event)

    const appendNotification = (
      title: string,
      content: string,
      type: string,
    ): void => {
      set((state) => {
        const next = [
          createNotification({
            type,
            title,
            content,
            payload,
            createdAt: event.timestamp,
          }),
          ...state.notifications,
        ].slice(0, MAX_NOTIFICATIONS)

        const unreadCount = next.filter((item) => !item.read).length
        return {
          notifications: next,
          notificationCount: unreadCount,
        }
      })
    }

    if (event.type === 'node_status_changed') {
      const nodeID =
        toNumber(payload.node_id) ??
        toNumber(payload.nodeID) ??
        toNumber(payload.id)
      const status = toText(payload.status)
      if (nodeID && status) {
        set((state) => ({
          nodeStatusMap: {
            ...state.nodeStatusMap,
            [nodeID]: status,
          },
        }))
      }
      return
    }

    if (event.type === 'rule_status_changed') {
      const ruleID =
        toNumber(payload.rule_id) ??
        toNumber(payload.ruleID) ??
        toNumber(payload.id)
      const status = toText(payload.status)
      if (ruleID && status) {
        set((state) => ({
          ruleStatusMap: {
            ...state.ruleStatusMap,
            [ruleID]: status,
          },
        }))
      }
      return
    }

    if (event.type === 'traffic_alert') {
      const content =
        toText(payload.message) ??
        toText(event.message) ??
        '检测到流量告警，请及时处理。'
      notification.warning({
        message: '流量告警',
        description: content,
        placement: 'topRight',
      })
      appendNotification('流量告警', content, event.type)
      return
    }

    if (event.type === 'announcement') {
      const title = toText(payload.title) ?? '系统公告'
      const content =
        toText(payload.content) ??
        toText(event.message) ??
        '收到新的系统公告。'
      notification.info({
        message: title,
        description: content,
        placement: 'topRight',
      })
      appendNotification(title, content, event.type)
      return
    }

    if (event.type === 'config_updated') {
      const content =
        toText(payload.message) ?? toText(event.message) ?? '配置已更新。'
      message.info(content)
      appendNotification('配置更新', content, event.type)
    }
  },
}))
