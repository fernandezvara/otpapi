<template>
  <div class="min-h-screen grid place-items-center p-6">
    <div class="w-full max-w-md bg-white border rounded p-6 shadow-sm">
      <h1 class="text-xl font-semibold mb-4 text-center">Reset password</h1>
      <form @submit.prevent="onSubmit" class="space-y-3">
        <div>
          <label class="block text-sm font-medium mb-1">Reset token</label>
          <input v-model="token" type="text" required class="w-full border rounded px-3 py-2 focus:outline-none focus:ring focus:ring-indigo-200" />
        </div>
        <div>
          <label class="block text-sm font-medium mb-1">New password</label>
          <input v-model="password" type="password" required class="w-full border rounded px-3 py-2 focus:outline-none focus:ring focus:ring-indigo-200" />
        </div>
        <div>
          <label class="block text-sm font-medium mb-1">Confirm new password</label>
          <input v-model="confirm" type="password" required class="w-full border rounded px-3 py-2 focus:outline-none focus:ring focus:ring-indigo-200" />
        </div>
        <button :disabled="loading" class="w-full bg-indigo-600 hover:bg-indigo-700 text-white rounded px-3 py-2 disabled:opacity-70">
          {{ loading ? 'Updating...' : 'Update password' }}
        </button>
        <p v-if="error" class="text-sm text-red-600">{{ error }}</p>
      </form>

      <div v-if="success" class="mt-4 text-sm bg-green-50 border border-green-100 rounded p-3">
        <p class="font-medium">Password updated. You can now sign in.</p>
        <RouterLink to="/login" class="text-indigo-600 hover:underline">Go to Sign in</RouterLink>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import { resetPassword } from '../services/auth'
import { RouterLink } from 'vue-router'

const token = ref('')
const password = ref('')
const confirm = ref('')
const loading = ref(false)
const error = ref('')
const success = ref(false)
const route = useRoute()

onMounted(() => {
  const t = route.query.token as string
  if (t) token.value = t
})

async function onSubmit() {
  if (password.value !== confirm.value) {
    error.value = 'Passwords do not match'
    return
  }
  loading.value = true
  error.value = ''
  success.value = false
  try {
    await resetPassword(token.value, password.value)
    success.value = true
  } catch (e: any) {
    error.value = e?.response?.data?.error || 'Reset failed'
  } finally {
    loading.value = false
  }
}
</script>
