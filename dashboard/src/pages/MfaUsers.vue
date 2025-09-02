<template>
  <div>
    <h1 class="text-2xl font-semibold mb-4">MFA Users</h1>

    <div class="rounded border bg-white p-4 space-y-3 mb-6">
      <div class="grid gap-3 sm:grid-cols-4">
        <input v-model="query" @keyup.enter="load" placeholder="Search by user id, account name, or issuer" class="border rounded px-3 py-2 sm:col-span-2" />
        <select v-model="status" class="border rounded px-3 py-2">
          <option value="all">all</option>
          <option value="active">active</option>
          <option value="disabled">disabled</option>
        </select>
        <div class="flex gap-2">
          <button @click="load" class="border rounded px-3 py-2">Search</button>
          <button @click="resetFilters" class="border rounded px-3 py-2">Reset</button>
        </div>
      </div>
      <p class="text-sm text-gray-500">Manage your end-user MFA enrollments. Reset to rotate secrets and regenerate backup codes.</p>
    </div>

    <div class="rounded border bg-white p-4">
      <div class="flex items-center justify-between mb-3">
        <h2 class="text-lg font-medium">Users</h2>
        <button @click="load" class="text-sm text-indigo-600 hover:underline">Refresh</button>
      </div>

      <div v-if="loading" class="text-sm text-gray-500">Loading...</div>
      <div v-else>
        <div v-if="items.length === 0" class="text-sm text-gray-500">No users found.</div>
        <div v-else class="overflow-x-auto">
          <table class="min-w-full text-sm">
            <thead>
              <tr class="text-left border-b">
                <th class="py-2 pr-4">User ID</th>
                <th class="py-2 pr-4">Account</th>
                <th class="py-2 pr-4">Issuer</th>
                <th class="py-2 pr-4">Status</th>
                <th class="py-2 pr-4">Created</th>
                <th class="py-2 pr-4">Actions</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="u in items" :key="u.user_id" class="border-b last:border-0">
                <td class="py-2 pr-4">{{ u.user_id }}</td>
                <td class="py-2 pr-4">{{ u.account_name || '-' }}</td>
                <td class="py-2 pr-4">{{ u.issuer }}</td>
                <td class="py-2 pr-4">
                  <span :class="u.is_active ? 'text-green-700' : 'text-gray-500'">{{ u.is_active ? 'active' : 'disabled' }}</span>
                </td>
                <td class="py-2 pr-4">{{ formatDate(u.created_at) }}</td>
                <td class="py-2 pr-4">
                  <div class="flex gap-2">
                    <button @click="viewQr(u.user_id)" class="border rounded px-2 py-1" :disabled="!u.is_active">QR</button>
                    <button @click="onReset(u.user_id)" class="border rounded px-2 py-1">Reset</button>
                    <button @click="onRegenerate(u.user_id)" class="border rounded px-2 py-1" :disabled="!u.is_active">Backup Codes</button>
                    <button @click="onDisable(u.user_id)" class="border rounded px-2 py-1" :disabled="!u.is_active">Disable</button>
                  </div>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>
    </div>

    <div class="rounded border bg-white p-4 mb-6">
      <h2 class="text-lg font-medium mb-1">Create MFA User</h2>
      <p class="text-sm text-gray-600 mb-3">Paste an API key token. It will be used to call the public API endpoint.</p>
      <div class="grid gap-3 sm:grid-cols-3">
        <input v-model="newId" placeholder="User ID (required)" class="border rounded px-3 py-2 sm:col-span-1" />
        <input v-model="newAccountName" placeholder="Account Name (optional)" class="border rounded px-3 py-2 sm:col-span-1" />
        <input v-model="newIssuer" placeholder="Issuer (optional)" class="border rounded px-3 py-2 sm:col-span-1" />
      </div>
      <div class="grid gap-3 sm:grid-cols-3 mt-3">
        <input v-model="apiKeySecret" type="password" placeholder="Paste API key token (required)" class="border rounded px-3 py-2 sm:col-span-3" />
      </div>
      <div class="mt-3">
        <button @click="onCreate" :disabled="creating || !newId || !apiKeySecret" class="border rounded px-3 py-2">
          {{ creating ? 'Creating...' : 'Create' }}
        </button>
      </div>
    </div>

    <div v-if="qrUrl" class="mt-4 rounded border bg-white p-3">
      <div class="flex items-center justify-between mb-2">
        <h3 class="font-medium">QR Code</h3>
        <button @click="qrUrl = ''" class="text-sm text-gray-600">Close</button>
      </div>
      <img :src="qrUrl" alt="QR Code" class="w-48 h-48" />
    </div>

    <div v-if="resetResult" class="mt-4 rounded border border-amber-200 bg-amber-50 p-3">
      <div class="flex items-center justify-between mb-2">
        <h3 class="font-medium">Enrollment Details</h3>
        <button @click="resetResult = null" class="text-sm text-gray-600">Close</button>
      </div>
      <p class="mb-2">Scan the new QR code and save the new backup codes below.</p>
      <div class="flex items-start gap-4">
        <img :src="resetResult.qr_code_url" alt="QR Code" class="w-48 h-48" />
        <div>
          <h4 class="font-medium mb-1">Backup Codes</h4>
          <ul class="list-disc ml-5">
            <li v-for="c in resetResult.backup_codes" :key="c"><code>{{ c }}</code></li>
          </ul>
        </div>
      </div>
    </div>

    <div v-if="backupCodes.length" class="mt-4 rounded border bg-white p-3">
      <div class="flex items-center justify-between mb-2">
        <h3 class="font-medium">New Backup Codes</h3>
        <button @click="backupCodes = []" class="text-sm text-gray-600">Close</button>
      </div>
      <ul class="list-disc ml-5">
        <li v-for="c in backupCodes" :key="c"><code>{{ c }}</code></li>
      </ul>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { listMfaUsers, disableMfaUser, resetMfaUser, regenerateBackupCodes, registerMfaWithApiKey, fetchQrBlobWithApiKey, type MfaUserItem, type ResetMfaResponse } from '../services/consoleMfa'

const items = ref<MfaUserItem[]>([])
const loading = ref(false)
const query = ref('')
const status = ref<'all'|'active'|'disabled'>('all')

const qrUrl = ref('')
const resetResult = ref<ResetMfaResponse | null>(null)
const backupCodes = ref<string[]>([])

// Create form state (API key required)
const newId = ref('')
const newAccountName = ref('')
const newIssuer = ref('')
const creating = ref(false)
const apiKeySecret = ref('')

function formatDate(s: string) {
  try { return new Date(s).toLocaleString() } catch { return s }
}

async function load() {
  loading.value = true
  try {
    items.value = await listMfaUsers({ q: query.value || undefined, status: status.value })
  } finally {
    loading.value = false
  }
}

function resetFilters() {
  query.value = ''
  status.value = 'all'
  load()
}

async function viewQr(id: string) {
  if (!apiKeySecret.value) {
    alert('Paste API key token to view QR')
    return
  }
  const blob = await fetchQrBlobWithApiKey(apiKeySecret.value, id)
  if (qrUrl.value && qrUrl.value.startsWith('blob:')) {
    try { URL.revokeObjectURL(qrUrl.value) } catch {}
  }
  qrUrl.value = URL.createObjectURL(blob)
}

async function onDisable(id: string) {
  await disableMfaUser(id)
  await load()
}

async function onReset(id: string) {
  if (!apiKeySecret.value) {
    alert('API key token required to fetch QR')
    return
  }
  const res = await resetMfaUser(id)
  const blob = await fetchQrBlobWithApiKey(apiKeySecret.value, id)
  resetResult.value = { ...res, qr_code_url: URL.createObjectURL(blob) }
  await load()
}

async function onRegenerate(id: string) {
  const res = await regenerateBackupCodes(id)
  backupCodes.value = res.backup_codes
}

async function onCreate() {
  if (!newId.value || !apiKeySecret.value) return
  creating.value = true
  try {
    const res = await registerMfaWithApiKey(apiKeySecret.value, { id: newId.value, account_name: newAccountName.value || undefined, issuer: newIssuer.value || undefined })
    const blob = await fetchQrBlobWithApiKey(apiKeySecret.value, newId.value)
    resetResult.value = { ...res, qr_code_url: URL.createObjectURL(blob) }
    // clear inputs and refresh list
    newId.value = ''
    newAccountName.value = ''
    newIssuer.value = ''
    apiKeySecret.value = ''
    await load()
  } catch (e: any) {
    alert(e?.response?.data?.error || 'Failed to create MFA user')
  } finally {
    creating.value = false
  }
}

onMounted(async () => {
  await load()
})
</script>
