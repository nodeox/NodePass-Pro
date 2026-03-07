import type { ApiSuccessResponse } from '../types'
import type {
  CreateNodeGroupPayload,
  CreateTunnelPayload,
  DeployCommandResponse,
  DeployNodePayload,
  NodeGroup,
  NodeGroupStats,
  NodeInstance,
  PaginationResult,
  Tunnel,
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

  create: (payload: CreateNodeGroupPayload) =>
    apiClient
      .post<ApiSuccessResponse<NodeGroup>>('/node-groups', payload)
      .then(unwrapData),

  update: (id: number, payload: Partial<CreateNodeGroupPayload>) =>
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
