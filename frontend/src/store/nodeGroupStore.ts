import { create } from 'zustand'

import { nodeGroupApi } from '../services/nodeGroupApi'
import type { CreateNodeGroupPayload, NodeGroup } from '../types/nodeGroup'
import { getErrorMessage } from '../utils/error'

interface NodeGroupStore {
  nodeGroups: NodeGroup[]
  currentGroup: NodeGroup | null
  loading: boolean
  error: string | null
  fetchNodeGroups: (params?: { type?: string; enabled?: boolean }) => Promise<void>
  fetchNodeGroup: (id: number) => Promise<void>
  createNodeGroup: (payload: CreateNodeGroupPayload) => Promise<NodeGroup>
  updateNodeGroup: (id: number, payload: Partial<CreateNodeGroupPayload>) => Promise<void>
  deleteNodeGroup: (id: number) => Promise<void>
  toggleNodeGroup: (id: number) => Promise<void>
  clearCurrentGroup: () => void
  clearError: () => void
}

export const useNodeGroupStore = create<NodeGroupStore>((set, get) => ({
  nodeGroups: [],
  currentGroup: null,
  loading: false,
  error: null,

  fetchNodeGroups: async (params) => {
    set({ loading: true, error: null })
    try {
      const result = await nodeGroupApi.list(params)
      set({
        nodeGroups: result.items ?? [],
        loading: false,
      })
    } catch (error) {
      const message = getErrorMessage(error, '加载节点组失败')
      set({ loading: false, error: message })
      throw error
    }
  },

  fetchNodeGroup: async (id) => {
    set({ loading: true, error: null })
    try {
      const result = await nodeGroupApi.get(id)
      set({ currentGroup: result, loading: false })
    } catch (error) {
      const message = getErrorMessage(error, '加载节点组详情失败')
      set({ loading: false, error: message })
      throw error
    }
  },

  createNodeGroup: async (payload) => {
    set({ loading: true, error: null })
    try {
      const created = await nodeGroupApi.create(payload)
      set((state) => ({
        nodeGroups: [...state.nodeGroups, created],
        currentGroup: created,
        loading: false,
      }))
      return created
    } catch (error) {
      const message = getErrorMessage(error, '创建节点组失败')
      set({ loading: false, error: message })
      throw error
    }
  },

  updateNodeGroup: async (id, payload) => {
    set({ loading: true, error: null })
    try {
      const updated = await nodeGroupApi.update(id, payload)
      set((state) => ({
        nodeGroups: state.nodeGroups.map((item) =>
          item.id === id ? { ...item, ...updated } : item,
        ),
        currentGroup:
          state.currentGroup?.id === id
            ? { ...state.currentGroup, ...updated }
            : state.currentGroup,
        loading: false,
      }))
    } catch (error) {
      const message = getErrorMessage(error, '更新节点组失败')
      set({ loading: false, error: message })
      throw error
    }
  },

  deleteNodeGroup: async (id) => {
    set({ loading: true, error: null })
    try {
      await nodeGroupApi.delete(id)
      set((state) => ({
        nodeGroups: state.nodeGroups.filter((item) => item.id !== id),
        currentGroup: state.currentGroup?.id === id ? null : state.currentGroup,
        loading: false,
      }))
    } catch (error) {
      const message = getErrorMessage(error, '删除节点组失败')
      set({ loading: false, error: message })
      throw error
    }
  },

  toggleNodeGroup: async (id) => {
    set({ loading: true, error: null })
    try {
      const toggled = await nodeGroupApi.toggle(id)
      set((state) => {
        const previous = state.nodeGroups.find((item) => item.id === id)
        const nextEnabled =
          typeof toggled?.is_enabled === 'boolean'
            ? toggled.is_enabled
            : previous
              ? !previous.is_enabled
              : false

        return {
          nodeGroups: state.nodeGroups.map((item) =>
            item.id === id ? { ...item, is_enabled: nextEnabled } : item,
          ),
          currentGroup:
            state.currentGroup?.id === id
              ? { ...state.currentGroup, is_enabled: nextEnabled }
              : state.currentGroup,
          loading: false,
        }
      })
    } catch (error) {
      const message = getErrorMessage(error, '切换节点组状态失败')
      set({ loading: false, error: message })
      throw error
    }
  },

  clearCurrentGroup: () => {
    set({ currentGroup: null })
  },

  clearError: () => {
    if (get().error) {
      set({ error: null })
    }
  },
}))

export type { NodeGroupStore }
