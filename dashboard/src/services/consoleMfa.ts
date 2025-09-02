import api from './api'

export interface MfaUserItem {
  user_id: string
  account_name: string
  issuer: string
  is_active: boolean
  created_at: string
  updated_at: string
}

export interface ResetMfaResponse {
  qr_code_url: string
  backup_codes: string[]
}

export type CreateMfaResponse = ResetMfaResponse

export async function listMfaUsers(params?: { q?: string; status?: 'active'|'disabled'|'all' }): Promise<MfaUserItem[]> {
  const { data } = await api.get('/console/mfa/', { params })
  return data.data as MfaUserItem[]
}

export async function disableMfaUser(id: string): Promise<void> {
  await api.post(`/console/mfa/${id}/disable`)
}

export async function resetMfaUser(id: string, body?: { account_name?: string; issuer?: string }): Promise<ResetMfaResponse> {
  const { data } = await api.post(`/console/mfa/${id}/reset`, body || {})
  return data as ResetMfaResponse
}

export async function regenerateBackupCodes(id: string): Promise<{ backup_codes: string[] }> {
  const { data } = await api.post(`/console/mfa/${id}/backup_codes/regenerate`)
  return data as { backup_codes: string[] }
}

export function qrImageUrl(id: string): string {
  return `/api/v1/console/mfa/${id}/qr`
}

export async function createMfaUser(body: { id: string; account_name?: string; issuer?: string }): Promise<CreateMfaResponse> {
  const { data } = await api.post('/console/mfa/', body)
  return data as CreateMfaResponse
}

export async function registerMfaWithApiKey(apiKey: string, body: { id: string; account_name?: string; issuer?: string }): Promise<ResetMfaResponse> {
  const { data } = await api.post('/mfa/register', body, {
    headers: { Authorization: `Bearer ${apiKey}` },
  })
  return data as ResetMfaResponse
}

export async function fetchQrBlobWithApiKey(apiKey: string, id: string): Promise<Blob> {
  const { data } = await api.get(`/mfa/${id}/qr`, {
    headers: { Authorization: `Bearer ${apiKey}` },
    responseType: 'blob' as any,
  })
  return data as Blob
}
