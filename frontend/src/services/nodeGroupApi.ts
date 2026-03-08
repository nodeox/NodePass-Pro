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
}
