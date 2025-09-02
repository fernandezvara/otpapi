import api from './api'

export interface ConsoleKeyItem {
  id: string
  key_name: string
  key_prefix: string
  key_last_four: string
  environment: string
  is_active: boolean
  usage_count: number
  created_at: string
}

export interface UsagePoint {
  day: string
  total: number
  success: number
}

export interface UsageByEndpoint {
  endpoint: string
  total: number
  success: number
}

export interface UsageSummary {
  total: number
  success: number
  failed: number
  first_event?: string
  last_event?: string
  by_day: UsagePoint[]
  by_endpoint: UsageByEndpoint[]
}

export async function listConsoleKeys(): Promise<ConsoleKeyItem[]> {
  const { data } = await api.get('/console/keys/')
  return data.data as ConsoleKeyItem[]
}

export async function createConsoleKey(key_name: string, environment: string): Promise<{ id: string; api_key: string }> {
  const { data } = await api.post('/console/keys/', { key_name, environment })
  return data
}

export async function disableConsoleKey(id: string): Promise<void> {
  await api.post(`/console/keys/${id}/disable`)
}

export async function rotateConsoleKey(id: string): Promise<{ id: string; api_key: string }> {
  const { data } = await api.post(`/console/keys/${id}/rotate`)
  return data
}

export async function getConsoleKeyUsage(id: string, period?: '24h'|'7d'|'30d'|'90d'|'all'): Promise<UsageSummary> {
  const params: Record<string, string> = {}
  if (period) params.period = period
  const { data } = await api.get(`/console/keys/${id}/usage`, { params })
  return data as UsageSummary
}
