<template>
  <div class="min-h-screen md:grid md:grid-cols-[16rem_1fr]">
    <a
      href="#main-content"
      class="absolute left-2 -top-10 focus:top-2 z-50 bg-white border rounded px-3 py-2 shadow"
    >Skip to content</a>
    <!-- Desktop sidebar -->
    <aside class="hidden md:block bg-white border-r">
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

    <!-- Mobile off-canvas sidebar -->
    <div v-if="drawerOpen" class="md:hidden">
      <div
        class="fixed inset-0 z-40"
        role="dialog"
        aria-modal="true"
        @keydown.esc.prevent.stop="closeDrawer"
      >
        <div class="fixed inset-0 bg-black/40" aria-hidden="true" @click="closeDrawer"></div>
        <aside
          ref="drawerRef"
          tabindex="-1"
          class="fixed inset-y-0 left-0 z-50 w-72 max-w-[85vw] bg-white shadow-xl outline-none flex flex-col"
          aria-label="Sidebar navigation"
          @keydown.tab.prevent="trapFocus($event)"
        >
          <div class="flex items-center justify-between p-4 border-b">
            <div class="font-semibold text-indigo-600">SecureAuth</div>
            <button
              class="p-2 rounded hover:bg-gray-100"
              aria-label="Close navigation"
              @click="closeDrawer"
            >
              ✕
            </button>
          </div>
          <nav class="p-2 space-y-1 overflow-y-auto">
            <RouterLink
              v-for="item in items"
              :key="item.to"
              :to="item.to"
              class="block px-3 py-2 rounded hover:bg-gray-100"
              active-class="bg-indigo-50 text-indigo-600"
              @click="closeDrawer"
            >
              {{ item.label }}
            </RouterLink>
          </nav>
        </aside>
      </div>
    </div>

    <!-- Main column -->
    <div class="flex flex-col">
      <header class="h-14 border-b bg-white flex items-center justify-between px-4 sticky top-0 z-10 pl-[env(safe-area-inset-left)] pr-[env(safe-area-inset-right)] pt-[env(safe-area-inset-top)]">
        <div class="flex items-center gap-2">
          <button
            class="md:hidden p-2 -ml-2 rounded hover:bg-gray-100"
            aria-label="Open navigation"
            :aria-expanded="drawerOpen ? 'true' : 'false'"
            @click="openDrawer"
          >
            ☰
          </button>
          <div class="text-sm text-gray-500">Customer Dashboard</div>
        </div>
        <button @click="onLogout" class="text-sm text-gray-700 hover:text-indigo-600">Logout</button>
      </header>
      <main id="main-content" class="p-6 pb-[env(safe-area-inset-bottom)]">
        <RouterView />
      </main>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onBeforeUnmount, nextTick } from 'vue'
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

// Mobile drawer state
const drawerOpen = ref(false)
const drawerRef = ref<HTMLElement | null>(null)

function openDrawer() {
  drawerOpen.value = true
  nextTick(() => {
    drawerRef.value?.focus()
  })
}
function closeDrawer() {
  drawerOpen.value = false
}

function trapFocus(e: KeyboardEvent) {
  const root = drawerRef.value
  if (!root) return
  const focusable = root.querySelectorAll<HTMLElement>(
    'a[href], button, textarea, input, select, [tabindex]:not([tabindex="-1"])'
  )
  if (focusable.length === 0) return
  const first = focusable[0]
  const last = focusable[focusable.length - 1]
  const current = document.activeElement as HTMLElement | null
  if (e.shiftKey) {
    if (!current || current === first) {
      e.preventDefault()
      last.focus()
    }
  } else {
    if (!current || current === last) {
      e.preventDefault()
      first.focus()
    }
  }
}

// Close drawer on route change (mobile)
onMounted(() => {
  const stop = router.afterEach(() => { drawerOpen.value = false })
  window.addEventListener('keydown', onGlobalKeydown)
  onBeforeUnmount(() => {
    stop()
    window.removeEventListener('keydown', onGlobalKeydown)
  })
})

function onGlobalKeydown(e: KeyboardEvent) {
  if (e.key === 'Escape' && drawerOpen.value) {
    e.preventDefault()
    closeDrawer()
  }
}

async function onLogout() {
  try { await logout() } catch {
    /* ignore logout errors */
  }
  auth.clear()
  router.replace('/login')
}
</script>
