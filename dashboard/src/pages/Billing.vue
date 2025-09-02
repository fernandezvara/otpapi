<template>
  <div>
    <h1 class="text-2xl font-semibold mb-4">Billing</h1>

    <div class="rounded border bg-white p-4 relative">
      <div v-if="toast" class="absolute top-2 right-2 bg-indigo-600 text-white text-xs px-3 py-1 rounded">{{ toast }}</div>

      <div class="flex flex-col sm:flex-row sm:items-center gap-2 mb-4">
        <label class="text-sm text-gray-600">Period</label>
        <select v-model="period" class="border rounded px-2 py-1 text-sm">
          <option value="24h">24h</option>
          <option value="7d">7d</option>
          <option value="30d">30d</option>
          <option value="90d">90d</option>
          <option value="all">all</option>
        </select>
        <button @click="reload" class="sm:ml-auto text-sm px-3 py-1 border rounded hover:bg-gray-50">Refresh</button>
      </div>

      <div class="grid grid-cols-2 md:grid-cols-4 gap-3 mb-4">
        <div class="p-3 rounded border">
          <div class="text-xs text-gray-500">Requests</div>
          <div class="text-xl font-semibold">{{ summary?.total ?? 0 }}</div>
        </div>
        <div class="p-3 rounded border">
          <div class="text-xs text-gray-500">Estimated Cost (USD)</div>
          <div class="text-xl font-semibold">{{ formatUSD(summary?.estimated_cost_usd || 0) }}</div>
        </div>
        <div class="p-3 rounded border">
          <div class="text-xs text-gray-500">Last Invoice</div>
          <div class="text-sm">{{ lastInvoiceText }}</div>
        </div>
        <div class="p-3 rounded border">
          <div class="text-xs text-gray-500">Subscription</div>
          <div class="text-sm">{{ subscriptionText }}</div>
        </div>
      </div>

      <div class="grid grid-cols-1 lg:grid-cols-2 gap-4">
        <div class="border rounded p-3">
          <div class="text-sm font-medium mb-2">Requests by day</div>
          <LineChart v-if="chartData.labels.length" :chart-data="chartData" :options="chartOptions" class="h-64 md:h-72"/>
          <div v-else class="text-sm text-gray-500">No data</div>
        </div>
        <div class="border rounded p-3">
          <div class="text-sm font-medium mb-2">Recent Billing Events</div>
          <div v-if="events.length === 0" class="text-sm text-gray-500">No events</div>
          <div v-else>
            <ResponsiveTable :columns="eventColumns" :items="events" :rowKey="(r) => r.id">
              <template #cell-created_at="{ item }">
                {{ new Date(item.created_at).toLocaleString() }}
              </template>
              <template #cell-event_type="{ item }">
                <span class="font-mono">{{ item.event_type }}</span>
              </template>
              <template #cell-id="{ item }">
                <span class="text-gray-600 break-all">{{ item.id }}</span>
              </template>
            </ResponsiveTable>
          </div>
        </div>
      </div>

      <div v-if="error" class="mt-3 text-sm text-red-600">{{ error }}</div>
    </div>
  </div>
  
</template>

<script setup lang="ts">
import { onMounted, onBeforeUnmount, ref, computed, watch } from 'vue'
import { LineChart } from 'vue-chart-3'
import { Chart, registerables } from 'chart.js'
import { getCustomerUsageSummary, type UsageSummary } from '../services/consoleUsage'
import { getBillingSummary, listBillingEvents, type BillingSummary, type BillingEventItem } from '../services/consoleBilling'
import { useAuthStore } from '../stores/auth'
import ResponsiveTable from '../components/common/ResponsiveTable.vue'

Chart.register(...registerables)

const auth = useAuthStore()
const token = auth.token || (typeof window !== 'undefined' ? localStorage.getItem('session_token') : null)

const period = ref<'24h'|'7d'|'30d'|'90d'|'all'>('30d')
const summary = ref<UsageSummary | null>(null)
const bill = ref<BillingSummary | null>(null)
const events = ref<BillingEventItem[]>([])
const error = ref<string | null>(null)
const toast = ref<string | null>(null)

const eventColumns = [
  { key: 'created_at', label: 'Time' },
  { key: 'event_type', label: 'Type' },
  { key: 'id', label: 'ID' },
]

let es: EventSource | null = null

function formatUSD(v: number) {
  return `$${v.toFixed(6)}`
}

const chartData = computed(() => {
  const labels = summary.value?.by_day?.map((p: { day: string }) => new Date(p.day).toLocaleDateString()) ?? []
  const totals = summary.value?.by_day?.map((p: { total: number }) => p.total) ?? []
  return {
    labels,
    datasets: [
      {
        label: 'Requests',
        data: totals,
        borderColor: '#6366f1',
        backgroundColor: 'rgba(99,102,241,0.2)',
        tension: 0.2,
        fill: true,
      },
    ],
  }
})

const chartOptions = {
  responsive: true,
  maintainAspectRatio: false,
  plugins: { legend: { display: false } },
  scales: { x: { ticks: { autoSkip: true } }, y: { beginAtZero: true } },
}

const lastInvoiceText = computed(() => {
  const inv = bill.value?.last_invoice
  if (!inv) return '—'
  const amt = inv.amount_due ?? inv.amount_paid
  const cur = inv.currency?.toUpperCase() || 'USD'
  const status = inv.status || 'n/a'
  return `${amt ?? 0} ${cur} · ${status}`
})

const subscriptionText = computed(() => {
  const s = bill.value?.subscription
  if (!s) return '—'
  const st = s.status || 'n/a'
  const until = s.current_period_end ? new Date(s.current_period_end).toLocaleDateString() : ''
  return `${st}${until ? ' · until ' + until : ''}`
})

async function reload() {
  try {
    error.value = null
    const [u, b, ev] = await Promise.all([
      getCustomerUsageSummary(period.value),
      getBillingSummary(),
      listBillingEvents({ limit: 50 }),
    ])
    summary.value = u
    bill.value = b
    events.value = ev
  } catch (e: any) {
    error.value = e?.message || 'Failed to load'
  }
}

function connectSSE() {
  if (!token) return
  const url = `/api/v1/console/analytics/stream?token=${encodeURIComponent(token)}`
  es = new EventSource(url)
  es.addEventListener('audit', (evt: MessageEvent) => {
    try {
      const ev = JSON.parse(evt.data)
      const d = ev.data || {}
      if (d.event === 'billing.webhook.received') {
        toast.value = 'New billing event received'
        reload()
        setTimeout(() => (toast.value = null), 2500)
      }
    } catch {
      /* ignore malformed billing audit event */
    }
  })
}

onMounted(() => {
  reload()
  connectSSE()
})

onBeforeUnmount(() => {
  if (es) { es.close(); es = null }
})

watch(period, reload)
</script>

<style scoped>
.max-w-\[12rem\] { max-width: 12rem; }
</style>
