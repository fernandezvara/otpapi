<template>
  <div class="min-h-screen grid grid-cols-[16rem_1fr]">
    <aside class="bg-white border-r">
      <div class="p-4 font-semibold text-indigo-600">SecureAuth</div>
      <nav class="px-2 space-y-1">
        <RouterLink
          v-for="item in items"
          :key="item.to"
          :to="item.to"
          class="block px-3 py-2 rounded hover:bg-gray-100"
          active-class="bg-indigo-50 text-indigo-600"
        >
          {{ item.label }}
        </RouterLink>
      </nav>
    </aside>
    <div class="flex flex-col">
      <header class="h-14 border-b bg-white flex items-center justify-between px-4">
        <div class="text-sm text-gray-500">Customer Dashboard</div>
        <button @click="onLogout" class="text-sm text-gray-700 hover:text-indigo-600">Logout</button>
      </header>
      <main class="p-6">
        <RouterView />
      </main>
    </div>
  </div>
</template>

<script setup lang="ts">
import { RouterLink, RouterView, useRouter } from 'vue-router'
import { logout } from '../../services/auth'
import { useAuthStore } from '../../stores/auth'

const items = [
  { to: '/dashboard/overview', label: 'Overview' },
  { to: '/dashboard/api-keys', label: 'API Keys' },
  { to: '/dashboard/mfa-users', label: 'MFA Users' },
  { to: '/dashboard/billing', label: 'Billing' },
  { to: '/dashboard/settings', label: 'Settings' },
  { to: '/dashboard/support', label: 'Support' },
  { to: '/dashboard/developer', label: 'Developer' },
]

const router = useRouter()
const auth = useAuthStore()

async function onLogout() {
  try { await logout() } catch {}
  auth.clear()
  router.replace('/login')
}
</script>
