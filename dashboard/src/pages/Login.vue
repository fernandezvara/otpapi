<template>
  <div class="min-h-screen grid place-items-center p-6">
    <div class="w-full max-w-sm bg-white border rounded p-6 shadow-sm">
      <h1 class="text-xl font-semibold mb-4 text-center">SecureAuth Login</h1>
      <form @submit.prevent="onSubmit" class="space-y-3">
        <div>
          <label class="block text-sm font-medium mb-1">Email</label>
          <input v-model="email" type="email" required class="w-full border rounded px-3 py-2 focus:outline-none focus:ring focus:ring-indigo-200" />
        </div>
        <div>
          <label class="block text-sm font-medium mb-1">Password</label>
          <input v-model="password" type="password" required class="w-full border rounded px-3 py-2 focus:outline-none focus:ring focus:ring-indigo-200" />
        </div>
        <button :disabled="loading" class="w-full bg-indigo-600 hover:bg-indigo-700 text-white rounded px-3 py-2 disabled:opacity-70">
          {{ loading ? 'Signing in...' : 'Sign in' }}
        </button>
        <p v-if="error" class="text-sm text-red-600">{{ error }}</p>
      </form>
      <div class="mt-4 text-sm flex items-center justify-between">
        <RouterLink to="/register" class="text-indigo-600 hover:underline">Create account</RouterLink>
        <RouterLink to="/forgot-password" class="text-indigo-600 hover:underline">Forgot password?</RouterLink>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useRouter, useRoute, RouterLink } from 'vue-router'
import { useAuthStore } from '../stores/auth'
import { login } from '../services/auth'

const email = ref('')
const password = ref('')
const loading = ref(false)
const error = ref('')
const router = useRouter()
const route = useRoute()
const auth = useAuthStore()

async function onSubmit() {
  loading.value = true
  error.value = ''
  try {
    const res = await login(email.value, password.value)
    if (res && res.session_token) {
      auth.setToken(res.session_token)
      const redirect = (route.query.redirect as string) || '/dashboard/overview'
      router.replace(redirect)
    } else {
      error.value = 'Unexpected response from server'
    }
  } catch (e: any) {
    error.value = e?.response?.data?.error || 'Login failed'
  } finally {
    loading.value = false
  }
}
</script>
