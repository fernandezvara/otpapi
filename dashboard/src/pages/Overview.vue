<template>
  <div>
    <h1 class="text-2xl font-semibold mb-4">Overview</h1>
    <div class="rounded border bg-white p-4">
      <p class="mb-3">Real-time analytics and system status will appear here.</p>

      <div class="grid grid-cols-3 gap-4 mb-4">
        <div class="p-3 rounded border">
          <div class="text-sm text-gray-500">Requests</div>
          <div class="text-xl font-semibold">{{ totals.requests }}</div>
        </div>
        <div class="p-3 rounded border">
          <div class="text-sm text-gray-500">Success</div>
          <div class="text-xl font-semibold text-green-600">{{ totals.success }}</div>
        </div>
        <div class="p-3 rounded border">
          <div class="text-sm text-gray-500">Failed</div>
          <div class="text-xl font-semibold text-red-600">{{ totals.failed }}</div>
        </div>
      </div>

      <div class="mb-4">
        <h2 class="font-medium mb-2">By Endpoint</h2>
        <div v-if="Object.keys(byEndpoint).length === 0" class="text-sm text-gray-500">No data yet</div>
        <ul v-else class="space-y-1">
          <li v-for="(stats, ep) in byEndpoint" :key="ep" class="flex justify-between text-sm">
            <span class="font-mono">{{ ep }}</span>
            <span>
              total {{ stats.total }} ·
              <span class="text-green-600">ok {{ stats.success }}</span> ·
              <span class="text-red-600">fail {{ stats.failed }}</span>
            </span>
          </li>
        </ul>
      </div>

      <div>
        <h2 class="font-medium mb-2">Recent Events</h2>
        <div v-if="events.length === 0" class="text-sm text-gray-500">Waiting for events…</div>
        <ul v-else class="space-y-1 max-h-64 overflow-auto text-sm">
          <li v-for="(e, idx) in events" :key="idx" class="font-mono">
            <span class="text-gray-500">{{ new Date(e.ts || Date.now()).toLocaleTimeString() }}</span>
            <template v-if="e.type === 'usage'">
              · usage {{ e.endpoint }} · <span :class="e.success ? 'text-green-600' : 'text-red-600'">{{ e.success ? 'ok' : 'fail' }}</span>
            </template>
            <template v-else-if="e.type === 'audit'">
              · audit {{ e.event }} ({{ e.actor_type }}) {{ e.ip ? '· ' + e.ip : '' }}
            </template>
            <template v-else>
              · {{ e.type }}
            </template>
          </li>
        </ul>
      </div>

      <div class="mt-3 text-sm">
        <span v-if="!connected" class="text-gray-500">Connecting to stream…</span>
        <span v-else class="text-green-600">Live</span>
        <span v-if="error" class="text-red-600 ml-2">{{ error }}</span>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { onMounted, onBeforeUnmount, ref, reactive } from 'vue'
import { useAuthStore } from '../stores/auth'

const auth = useAuthStore()
const token = auth.token || (typeof window !== 'undefined' ? localStorage.getItem('session_token') : null)

const connected = ref(false)
const error = ref<string | null>(null)
const totals = reactive({ requests: 0, success: 0, failed: 0 })
const byEndpoint = reactive<Record<string, { total: number; success: number; failed: number }>>({})
const events = ref<any[]>([])

let es: EventSource | null = null

function connect() {
  if (!token) {
    error.value = 'Not authenticated'
    return
  }
  const url = `/api/v1/console/analytics/stream?token=${encodeURIComponent(token)}`
  es = new EventSource(url)
  es.addEventListener('hello', () => {
    connected.value = true
  })
  es.addEventListener('heartbeat', () => {})
  es.addEventListener('usage', (evt: MessageEvent) => {
    try {
      const ev = JSON.parse(evt.data)
      const d = ev.data || {}
      totals.requests++
      if (d.success) totals.success++
      else totals.failed++
      const ep = d.endpoint || 'unknown'
      if (!byEndpoint[ep]) byEndpoint[ep] = { total: 0, success: 0, failed: 0 }
      byEndpoint[ep].total++
      if (d.success) byEndpoint[ep].success++
      else byEndpoint[ep].failed++
      pushEvent({ type: 'usage', ts: ev.ts, endpoint: ep, success: d.success })
    } catch {}
  })
  es.addEventListener('audit', (evt: MessageEvent) => {
    try {
      const ev = JSON.parse(evt.data)
      const d = ev.data || {}
      pushEvent({ type: 'audit', ts: ev.ts, event: d.event, actor_type: d.actor_type, ip: d.ip })
    } catch {}
  })
  es.onerror = () => {
    error.value = 'Stream error'
  }
}

function pushEvent(item: any) {
  events.value.unshift(item)
  if (events.value.length > 50) events.value.pop()
}

onMounted(connect)
onBeforeUnmount(() => {
  if (es) {
    es.close()
    es = null
  }
})
</script>
