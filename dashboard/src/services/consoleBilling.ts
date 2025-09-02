import api from './api'

export interface BillingEventItem {
  id: string
  event_type: string
  created_at: string
  payload: Record<string, any>
}

export interface InvoiceSummary {
  event_type: string
  created_at: string
  invoice_id?: string
  amount_due?: number
  amount_paid?: number
  currency?: string
  status?: string
}

export interface SubscriptionSummary {
  status?: string
  current_period_start?: string
  current_period_end?: string
}

export interface BillingSummary {
  last_invoice?: InvoiceSummary
  subscription?: SubscriptionSummary
}

export async function listBillingEvents(params?: { limit?: number; type?: string; since?: string }): Promise<BillingEventItem[]> {
  const { data } = await api.get('/console/billing/events', { params })
  return (data?.data || []) as BillingEventItem[]
}

export async function getBillingSummary(): Promise<BillingSummary> {
  const { data } = await api.get('/console/billing/summary')
  return data as BillingSummary
}
