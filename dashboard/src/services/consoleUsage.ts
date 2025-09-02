import api from './api'

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
  estimated_cost_usd: number
  first_event?: string
  last_event?: string
  by_day: UsagePoint[]
  by_endpoint: UsageByEndpoint[]
}

export async function getCustomerUsageSummary(period?: '24h'|'7d'|'30d'|'90d'|'all'): Promise<UsageSummary> {
  const params: Record<string, string> = {}
  if (period) params.period = period
  const { data } = await api.get('/console/usage/summary', { params })
  return data as UsageSummary
}
