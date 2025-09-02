<template>
  <div>
    <h1 class="text-2xl font-semibold mb-4">API Keys</h1>
    <div class="card space-y-3 mb-6">
      <h2 class="text-lg font-medium">Create a new key</h2>
      <form @submit.prevent="onCreate" class="grid gap-3 sm:grid-cols-3">
        <div class="sm:col-span-2">
          <label for="keyName" class="block text-sm font-medium mb-1">Key name</label>
          <input id="keyName" v-model="newName" placeholder="e.g. Server key" class="form-control" autocomplete="off" required />
        </div>
        <div>
          <label for="keyEnv" class="block text-sm font-medium mb-1">Environment</label>
          <select id="keyEnv" v-model="newEnv" class="form-control">
            <option value="test">test</option>
            <option value="live">live</option>
          </select>
        </div>
        <button type="submit" class="btn-primary sm:col-span-3" :disabled="creating">
          {{ creating ? 'Creating...' : 'Create API key' }}
        </button>
        <p v-if="createError" class="text-sm text-red-600 sm:col-span-3" aria-live="polite">{{ createError }}</p>
      </form>

      <div v-if="createdKey" class="mt-3 rounded border border-amber-200 bg-amber-50 p-3">
        <p class="font-medium">Copy your new API key now. You won't be able to see it again.</p>
        <div class="mt-2 flex items-center gap-2">
          <code class="text-sm break-all bg-white border rounded px-2 py-1 flex-1">{{ createdKey }}</code>
          <button @click="copy(createdKey)" class="text-sm border rounded px-2 py-1">Copy</button>
        </div>
      </div>
    </div>

    <div class="card">
      <div class="flex items-center justify-between mb-3">
        <h2 class="text-lg font-medium">Your keys</h2>
        <button @click="load()" class="text-sm text-indigo-600 hover:underline">Refresh</button>
      </div>

      <div v-if="loading" class="text-sm text-gray-500">Loading...</div>
      <div v-else>
        <div v-if="items.length === 0" class="text-sm text-gray-500">No keys yet.</div>
        <div v-else>
          <ResponsiveTable :columns="columns" :items="items" :rowKey="rowKey">
            <template #cell-is_active="{ item }">
              <span :class="item.is_active ? 'text-green-700' : 'text-gray-500'">{{ item.is_active ? 'active' : 'disabled' }}</span>
            </template>
            <template #cell-created_at="{ item }">
              {{ formatDate(item.created_at) }}
            </template>
            <template #cell-actions="{ item }">
              <!-- Desktop actions -->
              <div class="hidden md:flex gap-2">
                <button @click="onViewUsage(item.id)" class="border rounded px-2 py-1">Usage</button>
                <button @click="onRotate(item.id)" class="border rounded px-2 py-1">Rotate</button>
                <button @click="onDisable(item.id)" class="border rounded px-2 py-1" :disabled="!item.is_active">Disable</button>
              </div>
              <!-- Mobile kebab -->
              <div class="md:hidden relative">
                <!-- @vue-ignore Volar false-positive in slot scope: parent ref is accessible in template -->
                <button :ref="el => setMenuButtonRef(item.id, el as HTMLElement)" class="border rounded px-2 py-1" @click="actionsOpenId = actionsOpenId === item.id ? '' : item.id" aria-label="More actions" aria-haspopup="menu" :aria-expanded="actionsOpenId === item.id">⋯</button>
                <!-- @vue-ignore Volar false-positive for :ref in slot scope -->
                <div :ref="el => setMenuContainerRef(item.id, el as HTMLElement)" v-if="actionsOpenId === item.id" class="absolute mt-1 right-0 z-10 w-40 rounded border bg-white shadow" role="menu" tabindex="0" @keydown.escape.stop.prevent="actionsOpenId = ''">
                  <!-- @vue-ignore Volar false-positive for :ref in slot scope -->
                  <button :ref="el => setFirstMenuItemRef(item.id, el as HTMLElement)" @click="onViewUsage(item.id); actionsOpenId = ''" class="block w-full text-left px-3 py-2 hover:bg-gray-50" role="menuitem">Usage</button>
                  <button @click="onRotate(item.id); actionsOpenId = ''" class="block w-full text-left px-3 py-2 hover:bg-gray-50" role="menuitem">Rotate</button>
                  <button @click="onDisable(item.id); actionsOpenId = ''" :disabled="!item.is_active" class="block w-full text-left px-3 py-2 hover:bg-gray-50 disabled:text-gray-400" role="menuitem">Disable</button>
                </div>
              </div>
            </template>
          </ResponsiveTable>
          <div v-if="viewingUsageId" class="mt-4 rounded border bg-white">
            <div class="flex items-center justify-between p-3 border-b">
              <div class="flex items-center gap-3">
                <h3 class="font-medium">Usage ({{ usagePeriod }})</h3>
                <select v-model="usagePeriod" @change="reloadUsage()" class="form-control text-sm">
                  <option value="24h">24h</option>
                  <option value="7d">7d</option>
                  <option value="30d">30d</option>
                  <option value="90d">90d</option>
                  <option value="all">all</option>
                </select>
              </div>
              <button @click="closeUsage" class="text-sm text-gray-600">Close</button>
            </div>
            <div class="p-3">
              <div v-if="usageLoading" class="text-sm text-gray-500">Loading usage...</div>
              <div v-else>
                <p v-if="usageError" class="text-sm text-red-600">{{ usageError }}</p>
                <div v-if="usage" class="space-y-3">
                  <div class="grid grid-cols-1 sm:grid-cols-3 gap-3">
                    <div class="rounded border p-3"><div class="text-xs text-gray-500">Total</div><div class="text-lg font-semibold">{{ usage.total }}</div></div>
                    <div class="rounded border p-3"><div class="text-xs text-gray-500">Success</div><div class="text-lg font-semibold">{{ usage.success }}</div></div>
                    <div class="rounded border p-3"><div class="text-xs text-gray-500">Failed</div><div class="text-lg font-semibold">{{ usage.failed }}</div></div>
                  </div>
                  <div>
                    <h4 class="font-medium mb-2">By Endpoint</h4>
                    <div v-if="usage.by_endpoint.length === 0" class="text-sm text-gray-500">No data.</div>
                    <div v-else class="table-wrapper">
                      <table class="min-w-full text-sm">
                        <thead>
                          <tr class="text-left border-b">
                            <th class="py-2 pr-4">Endpoint</th>
                            <th class="py-2 pr-4">Total</th>
                            <th class="py-2 pr-4">Success</th>
                          </tr>
                        </thead>
                        <tbody>
                          <tr v-for="e in usage.by_endpoint" :key="e.endpoint" class="border-b last:border-0">
                            <td class="py-2 pr-4">{{ e.endpoint }}</td>
                            <td class="py-2 pr-4">{{ e.total }}</td>
                            <td class="py-2 pr-4">{{ e.success }}</td>
                          </tr>
                        </tbody>
                      </table>
                    </div>
                  </div>
                </div>
              </div>
            </div>
          </div>
          <div v-if="rotatedKey" class="mt-3 rounded border border-amber-200 bg-amber-50 p-3">
            <p class="font-medium">Copy your rotated API key now. You won't be able to see it again.</p>
            <div class="mt-2 flex items-center gap-2">
              <code class="text-sm break-all bg-white border rounded px-2 py-1 flex-1">{{ rotatedKey }}</code>
              <button @click="copy(rotatedKey)" class="text-sm border rounded px-2 py-1">Copy</button>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, nextTick, onBeforeUnmount, watch } from 'vue'
import ResponsiveTable from '../components/common/ResponsiveTable.vue'
import { listConsoleKeys, createConsoleKey, disableConsoleKey, rotateConsoleKey, getConsoleKeyUsage, type ConsoleKeyItem, type UsageSummary } from '../services/consoleKeys'

const items = ref<ConsoleKeyItem[]>([])
const loading = ref(false)

const newName = ref('')
const newEnv = ref<'test'|'live'>('test')
const creating = ref(false)
const createError = ref('')
const createdKey = ref('')

const rotatedKey = ref('')
const actionsOpenId = ref('')

// Kebab menu focus management and click-outside handling
const menuButtonRefs = ref<Record<string, HTMLElement | null>>({})
const menuContainerRefs = ref<Record<string, HTMLElement | null>>({})
const firstMenuItemRefs = ref<Record<string, HTMLElement | null>>({})

function setMenuButtonRef(id: string, el: HTMLElement | null) {
  menuButtonRefs.value[id] = el
}
function setMenuContainerRef(id: string, el: HTMLElement | null) {
  menuContainerRefs.value[id] = el
}
function setFirstMenuItemRef(id: string, el: HTMLElement | null) {
  firstMenuItemRefs.value[id] = el
}

function closeActionsMenu() {
  actionsOpenId.value = ''
}

const onDocClick = (e: MouseEvent) => {
  const openId = actionsOpenId.value
  if (!openId) return
  const menuEl = menuContainerRefs.value[openId]
  const btnEl = menuButtonRefs.value[openId]
  const target = e.target as Node
  if (menuEl?.contains(target) || btnEl?.contains(target)) return
  closeActionsMenu()
}

watch(actionsOpenId, async (newId: string, oldId: string) => {
  if (newId) {
    document.addEventListener('click', onDocClick, true)
    await nextTick()
    firstMenuItemRefs.value[newId]?.focus()
  } else {
    document.removeEventListener('click', onDocClick, true)
    // Return focus to the kebab button that opened the menu
    if (oldId) {
      menuButtonRefs.value[oldId]?.focus()
    }
  }
})

onBeforeUnmount(() => {
  document.removeEventListener('click', onDocClick, true)
})

const columns = [
  { key: 'key_name', label: 'Name' },
  { key: 'environment', label: 'Env' },
  { key: 'key_prefix', label: 'Prefix' },
  { key: 'key_last_four', label: 'Last 4' },
  { key: 'is_active', label: 'Active' },
  { key: 'usage_count', label: 'Usage' },
  { key: 'created_at', label: 'Created' },
  { key: 'actions', label: 'Actions' },
]

const rowKey = (r: ConsoleKeyItem) => r.id

function formatDate(s: string) {
  try { return new Date(s).toLocaleString() } catch { return s }
}

async function load() {
  loading.value = true
  try {
    const data = await listConsoleKeys()
    items.value = data
  } finally {
    loading.value = false
  }
}

async function onCreate() {
  creating.value = true
  createError.value = ''
  createdKey.value = ''
  try {
    const res = await createConsoleKey(newName.value, newEnv.value)
    createdKey.value = res.api_key
    newName.value = ''
    newEnv.value = 'test'
    await load()
  } catch (e: any) {
    createError.value = e?.response?.data?.error || 'Failed to create key'
  } finally {
    creating.value = false
  }
}

async function onDisable(id: string) {
  await disableConsoleKey(id)
  await load()
}

async function onRotate(id: string) {
  const res = await rotateConsoleKey(id)
  rotatedKey.value = res.api_key
  await load()
}

async function copy(text: string) {
  try {
    await navigator.clipboard.writeText(text)
  } catch {
    /* ignore clipboard errors */
  }
}

// Usage view state
const viewingUsageId = ref('')
const usage = ref<UsageSummary | null>(null)
const usageLoading = ref(false)
const usageError = ref('')
const usagePeriod = ref<'24h'|'7d'|'30d'|'90d'|'all'>('30d')

async function onViewUsage(id: string) {
  viewingUsageId.value = id
  await fetchUsage()
}

async function fetchUsage() {
  if (!viewingUsageId.value) return
  usageLoading.value = true
  usageError.value = ''
  usage.value = null
  try {
    usage.value = await getConsoleKeyUsage(viewingUsageId.value, usagePeriod.value)
  } catch (e: any) {
    usageError.value = e?.response?.data?.error || 'Failed to load usage'
  } finally {
    usageLoading.value = false
  }
}

async function reloadUsage() {
  await fetchUsage()
}

function closeUsage() {
  viewingUsageId.value = ''
  usage.value = null
}

onMounted(load)
</script>
