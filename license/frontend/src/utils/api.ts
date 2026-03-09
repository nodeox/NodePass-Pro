import request, { unwrap } from './request'
import type {
  ApiSuccess,
  BatchActionResult,
  BatchLicenseUpdateFields,
  DashboardStats,
  License,
  LicenseListParams,
  LicenseActivation,
  LicenseListResult,
  LicensePlan,
  LoginResponse,
  PlanStatus,
  ProductRelease,
  VersionSyncConfig,
  VersionSyncResult,
  UpdateLicensePayload,
  VerifyLogListResult,
  VersionPolicy
} from '../types/api'

export const authApi = {
  async login(payload: { username: string; password: string }) {
    const res = await request.post<ApiSuccess<LoginResponse>>('/auth/login', payload)
    return unwrap(res.data)
  },
  async me() {
    const res = await request.get<ApiSuccess<{ id: number; username: string; email: string }>>('/auth/me')
    return unwrap(res.data)
  }
}

export const dashboardApi = {
  async stats() {
    const res = await request.get<ApiSuccess<DashboardStats>>('/dashboard')
    return unwrap(res.data)
  }
}

export const planApi = {
  async list() {
    const res = await request.get<ApiSuccess<LicensePlan[]>>('/plans')
    return unwrap(res.data)
  },
  async create(payload: {
    code: string
    name: string
    description: string
    max_machines: number
    duration_days: number
    status: PlanStatus
  }) {
    const res = await request.post<ApiSuccess<LicensePlan>>('/plans', payload)
    return unwrap(res.data)
  },
  async update(
    id: number,
    payload: {
      code: string
      name: string
      description: string
      max_machines: number
      duration_days: number
      status: PlanStatus
    }
  ) {
    const res = await request.put<ApiSuccess<LicensePlan>>(`/plans/${id}`, payload)
    return unwrap(res.data)
  },
  async clone(
    id: number,
    payload?: {
      code?: string
      name?: string
      description?: string
      status?: PlanStatus
    }
  ) {
    const res = await request.post<ApiSuccess<LicensePlan>>(`/plans/${id}/clone`, payload ?? {})
    return unwrap(res.data)
  },
  async remove(id: number, force = false) {
    await request.delete(`/plans/${id}`, {
      params: force ? { force: true } : undefined
    })
  }
}

export const licenseApi = {
  async generate(payload: {
    plan_id: number
    customer: string
    count: number
    expire_days: number
    max_machines?: number
    metadata_json?: string
    note?: string
  }) {
    const res = await request.post<ApiSuccess<License[]>>('/licenses/generate', payload)
    return unwrap(res.data)
  },
  async list(params: LicenseListParams) {
    const res = await request.get<ApiSuccess<LicenseListResult>>('/licenses', { params })
    return unwrap(res.data)
  },
  async get(id: number) {
    const res = await request.get<ApiSuccess<License>>(`/licenses/${id}`)
    return unwrap(res.data)
  },
  async update(id: number, payload: UpdateLicensePayload) {
    const res = await request.put<ApiSuccess<License>>(`/licenses/${id}`, payload)
    return unwrap(res.data)
  },
  async listActivations(id: number) {
    const res = await request.get<ApiSuccess<LicenseActivation[]>>(`/licenses/${id}/activations`)
    return unwrap(res.data)
  },
  async unbindActivation(id: number, activationId: number) {
    await request.delete(`/licenses/${id}/activations/${activationId}`)
  },
  async clearActivations(id: number) {
    const res = await request.delete<ApiSuccess<{ license_id: number; cleared_count: number; remaining_count: number }>>(
      `/licenses/${id}/activations`
    )
    return unwrap(res.data)
  },
  async revoke(id: number) {
    await request.post(`/licenses/${id}/revoke`)
  },
  async restore(id: number) {
    await request.post(`/licenses/${id}/restore`)
  },
  async remove(id: number) {
    await request.delete(`/licenses/${id}`)
  },
  async batchDelete(licenseIDs: number[]) {
    const res = await request.post<ApiSuccess<BatchActionResult>>('/licenses/batch/delete', {
      license_ids: licenseIDs
    })
    return unwrap(res.data)
  },
  async batchUpdate(licenseIDs: number[], updates: BatchLicenseUpdateFields) {
    const res = await request.post<ApiSuccess<BatchActionResult>>('/licenses/batch/update', {
      license_ids: licenseIDs,
      updates
    })
    return unwrap(res.data)
  },
  async batchRevoke(licenseIDs: number[]) {
    const res = await request.post<ApiSuccess<BatchActionResult>>('/licenses/batch/revoke', {
      license_ids: licenseIDs
    })
    return unwrap(res.data)
  },
  async batchRestore(licenseIDs: number[]) {
    const res = await request.post<ApiSuccess<BatchActionResult>>('/licenses/batch/restore', {
      license_ids: licenseIDs
    })
    return unwrap(res.data)
  }
}

export const versionApi = {
  async getSyncConfigs() {
    const res = await request.get<ApiSuccess<VersionSyncConfig[]>>('/version-sync/configs')
    return unwrap(res.data)
  },
  async listReleases() {
    const res = await request.get<ApiSuccess<ProductRelease[]>>('/releases')
    return unwrap(res.data)
  },
  async getSyncConfig() {
    const res = await request.get<ApiSuccess<VersionSyncConfig>>('/version-sync/config')
    return unwrap(res.data)
  },
  async listRecycleReleases() {
    const res = await request.get<ApiSuccess<ProductRelease[]>>('/releases/recycle')
    return unwrap(res.data)
  },
  async updateSyncConfig(payload: {
    product?: string
    enabled?: boolean
    auto_sync?: boolean
    interval_minutes?: number
    github_owner?: string
    github_repo?: string
    github_token?: string
    channel?: string
    include_prerelease?: boolean
    api_base_url?: string
  }) {
    const res = await request.put<ApiSuccess<VersionSyncConfig>>('/version-sync/config', payload)
    return unwrap(res.data)
  },
  async manualSync(payload?: { product?: string }) {
    const res = await request.post<ApiSuccess<VersionSyncResult>>('/version-sync/manual', payload ?? {})
    return unwrap(res.data)
  },
  async createRelease(payload: {
    product: string
    version: string
    channel: string
    is_mandatory: boolean
    release_notes: string
  }) {
    const res = await request.post<ApiSuccess<ProductRelease>>('/releases', payload)
    return unwrap(res.data)
  },
  async updateRelease(
    id: number,
    payload: {
      product?: string
      version?: string
      channel?: string
      is_mandatory?: boolean
      release_notes?: string
      is_active?: boolean
      published_at?: string
    }
  ) {
    const res = await request.put<ApiSuccess<ProductRelease>>(`/releases/${id}`, payload)
    return unwrap(res.data)
  },
  async uploadRelease(payload: FormData) {
    const res = await request.post<ApiSuccess<ProductRelease>>('/releases/upload', payload, {
      headers: { 'Content-Type': 'multipart/form-data' },
      timeout: 120000
    })
    return unwrap(res.data)
  },
  async downloadReleaseFile(id: number) {
    const res = await request.get<Blob>(`/releases/${id}/file`, { responseType: 'blob', timeout: 120000 })
    const header = res.headers['content-disposition'] as string | undefined
    let filename = `release-${id}.bin`
    if (header) {
      const matched = header.match(/filename\*?=(?:UTF-8''|")?([^\";]+)/i)
      if (matched?.[1]) {
        filename = decodeURIComponent(matched[1].replace(/"/g, '').trim())
      }
    }
    return { blob: res.data, filename }
  },
  async replaceReleaseFile(id: number, payload: FormData) {
    const res = await request.put<ApiSuccess<ProductRelease>>(`/releases/${id}/file`, payload, {
      headers: { 'Content-Type': 'multipart/form-data' },
      timeout: 120000
    })
    return unwrap(res.data)
  },
  async deleteRelease(id: number) {
    await request.delete(`/releases/${id}`)
  },
  async restoreRelease(id: number) {
    const res = await request.post<ApiSuccess<ProductRelease>>(`/releases/${id}/restore`)
    return unwrap(res.data)
  },
  async purgeRelease(id: number) {
    await request.delete(`/releases/${id}/purge`)
  },
  async listPolicies() {
    const res = await request.get<ApiSuccess<VersionPolicy[]>>('/version-policies')
    return unwrap(res.data)
  },
  async createPolicy(payload: {
    product: string
    channel: string
    min_supported_version: string
    recommended_version?: string
    message?: string
    is_active: boolean
  }) {
    const res = await request.post<ApiSuccess<VersionPolicy>>('/version-policies', payload)
    return unwrap(res.data)
  },
  async updatePolicy(
    id: number,
    payload: {
      product: string
      channel: string
      min_supported_version: string
      recommended_version?: string
      message?: string
      is_active: boolean
    }
  ) {
    const res = await request.put<ApiSuccess<VersionPolicy>>(`/version-policies/${id}`, payload)
    return unwrap(res.data)
  },
  async deletePolicy(id: number) {
    await request.delete(`/version-policies/${id}`)
  }
}

export const verifyLogApi = {
  async list(params: { page: number; page_size: number }) {
    const res = await request.get<ApiSuccess<VerifyLogListResult>>('/verify-logs', { params })
    return unwrap(res.data)
  }
}
