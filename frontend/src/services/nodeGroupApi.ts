import type { ApiSuccessResponse } from '../types'
import type {
  AccessibleNodeGroup,
  CreateNodeGroupPayload,
  CreateTunnelPayload,
  DeployCommandResponse,
  DeployNodePayload,
  NodeGroup,
  NodeGroupRelation,
  NodeGroupStats,
  NodeInstance,
  PaginationResult,
  Tunnel,
  UpdateNodeGroupPayload,
} from '../types/nodeGroup'

import apiClient, { unwrapData } from './api'

export const nodeGroupApi = {
  list: (params?: {
    type?: string
    enabled?: boolean
    page?: number
    page_size?: number
  }) =>
    apiClient
      .get<ApiSuccessResponse<PaginationResult<NodeGroup>>>('/node-groups', {
        params,
      })
      .then(unwrapData),

  get: (id: number) =>
    apiClient.get<ApiSuccessResponse<NodeGroup>>(`/node-groups/${id}`).then(unwrapData),

  accessibleNodes: () =>
    apiClient
      .get<ApiSuccessResponse<{ items: AccessibleNodeGroup[] }>>('/node-groups/accessible-nodes')
      .then(unwrapData),

  create: (payload: CreateNodeGroupPayload) =>
    apiClient
      .post<ApiSuccessResponse<NodeGroup>>('/node-groups', payload)
      .then(unwrapData),

  update: (id: number, payload: UpdateNodeGroupPayload) =>
    apiClient
      .put<ApiSuccessResponse<NodeGroup>>(`/node-groups/${id}`, payload)
      .then(unwrapData),

  delete: (id: number) =>
    apiClient
      .delete<ApiSuccessResponse<null>>(`/node-groups/${id}`)
      .then(unwrapData),

  toggle: (id: number) =>
    apiClient
      .post<ApiSuccessResponse<NodeGroup>>(`/node-groups/${id}/toggle`)
      .then(unwrapData),

  getStats: (id: number) =>
    apiClient
      .get<ApiSuccessResponse<NodeGroupStats>>(`/node-groups/${id}/stats`)
      .then(unwrapData),

  generateDeployCommand: (id: number, payload: DeployNodePayload) =>
    apiClient
      .post<ApiSuccessResponse<DeployCommandResponse>>(
        `/node-groups/${id}/generate-deploy-command`,
        payload,
      )
      .then(unwrapData),

  listNodes: (id: number) =>
    apiClient
      .get<ApiSuccessResponse<NodeInstance[]>>(`/node-groups/${id}/nodes`)
      .then(unwrapData),

  addNode: (id: number, payload: { name: string; host: string; port: number }) =>
    apiClient
      .post<ApiSuccessResponse<NodeInstance>>(`/node-groups/${id}/nodes`, payload)
      .then(unwrapData),

  listRelations: (id: number) =>
    apiClient
      .get<ApiSuccessResponse<NodeGroupRelation[]>>(`/node-groups/${id}/relations`)
      .then(unwrapData),

  createRelation: (id: number, payload: { exit_group_id: number }) =>
    apiClient
      .post<ApiSuccessResponse<NodeGroupRelation>>(`/node-groups/${id}/relations`, payload)
      .then(unwrapData),

  deleteRelation: (id: number) =>
    apiClient
      .delete<ApiSuccessResponse<null>>(`/node-group-relations/${id}`)
      .then(unwrapData),

  toggleRelation: (id: number) =>
    apiClient
      .post<ApiSuccessResponse<{ id: number }>>(`/node-group-relations/${id}/toggle`)
      .then(unwrapData),
}

export const nodeInstanceApi = {
  get: (id: number) =>
    apiClient
      .get<ApiSuccessResponse<NodeInstance>>(`/node-instances/${id}`)
      .then(unwrapData),

  update: (id: number, payload: Partial<NodeInstance>) =>
    apiClient
      .put<ApiSuccessResponse<NodeInstance>>(`/node-instances/${id}`, payload)
      .then(unwrapData),

  delete: (id: number) =>
    apiClient
      .delete<ApiSuccessResponse<null>>(`/node-instances/${id}`)
      .then(unwrapData),

  restart: (id: number) =>
    apiClient
      .post<ApiSuccessResponse<NodeInstance>>(`/node-instances/${id}/restart`)
      .then(unwrapData),
}

export const tunnelApi = {
  list: (params?: { status?: string; page?: number; page_size?: number }) =>
    apiClient
      .get<ApiSuccessResponse<PaginationResult<Tunnel>>>('/tunnels', {
        params,
      })
      .then(unwrapData),

  get: (id: number) =>
    apiClient.get<ApiSuccessResponse<Tunnel>>(`/tunnels/${id}`).then(unwrapData),

  create: (payload: CreateTunnelPayload) =>
    apiClient.post<ApiSuccessResponse<Tunnel>>('/tunnels', payload).then(unwrapData),

  update: (id: number, payload: Partial<CreateTunnelPayload>) =>
    apiClient
      .put<ApiSuccessResponse<Tunnel>>(`/tunnels/${id}`, payload)
      .then(unwrapData),

  delete: (id: number) =>
    apiClient.delete<ApiSuccessResponse<null>>(`/tunnels/${id}`).then(unwrapData),

  start: (id: number) =>
    apiClient
      .post<ApiSuccessResponse<Tunnel>>(`/tunnels/${id}/start`)
      .then(unwrapData),

  stop: (id: number) =>
    apiClient
      .post<ApiSuccessResponse<Tunnel>>(`/tunnels/${id}/stop`)
      .then(unwrapData),

  export: (payload: { tunnel_ids: number[]; format: 'json' | 'yaml' }) =>
    apiClient
      .post<ApiSuccessResponse<{ data: string; format: string }>>('/tunnels/export', payload)
      .then(unwrapData),

  exportAll: (payload: { format: 'json' | 'yaml' }) =>
    apiClient
      .post<ApiSuccessResponse<{ data: string; format: string }>>('/tunnels/export-all', payload)
      .then(unwrapData),

  import: (payload: {
    format: 'json' | 'yaml'
    data: string
    entry_group_id: number
    exit_group_id?: number
    skip_errors: boolean
  }) =>
    apiClient
      .post<ApiSuccessResponse<{
        total: number
        success: number
        failed: number
        errors?: Array<{ index: number; name: string; message: string }>
        tunnels: Tunnel[]
      }>>('/tunnels/import', payload)
      .then(unwrapData),

  applyTemplate: (payload: {
    template_id: number
    name: string
    description?: string
    entry_group_id: number
    exit_group_id?: number
  }) =>
    apiClient
      .post<ApiSuccessResponse<Tunnel>>('/tunnels/apply-template', payload)
      .then(unwrapData),
}

export const tunnelTemplateApi = {
  list: (params?: { protocol?: string; is_public?: boolean; page?: number; page_size?: number }) =>
    apiClient
      .get<ApiSuccessResponse<PaginationResult<TunnelTemplate>>>('/tunnel-templates', {
        params,
      })
      .then(unwrapData),

  get: (id: number) =>
    apiClient.get<ApiSuccessResponse<TunnelTemplate>>(`/tunnel-templates/${id}`).then(unwrapData),

  create: (payload: CreateTunnelTemplatePayload) =>
    apiClient
      .post<ApiSuccessResponse<TunnelTemplate>>('/tunnel-templates', payload)
      .then(unwrapData),

  update: (id: number, payload: Partial<CreateTunnelTemplatePayload>) =>
    apiClient
      .put<ApiSuccessResponse<TunnelTemplate>>(`/tunnel-templates/${id}`, payload)
      .then(unwrapData),

  delete: (id: number) =>
    apiClient.delete<ApiSuccessResponse<null>>(`/tunnel-templates/${id}`).then(unwrapData),
}

export interface TunnelTemplate {
  id: number
  user_id: number
  name: string
  description?: string
  protocol: string
  config: TunnelTemplateConfig
  is_public: boolean
  usage_count: number
  created_at: string
  updated_at: string
}

export interface TunnelTemplateConfig {
  listen_host?: string
  listen_port?: number
  remote_host: string
  remote_port: number
  load_balance_strategy: string
  ip_type: string
  enable_proxy_protocol: boolean
  forward_targets: Array<{ host: string; port: number; weight: number }>
  health_check_interval: number
  health_check_timeout: number
  protocol_config?: Record<string, unknown>
}

export interface CreateTunnelTemplatePayload {
  name: string
  description?: string
  protocol: string
  config: TunnelTemplateConfig
  is_public: boolean
}
