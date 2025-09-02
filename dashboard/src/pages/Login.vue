<template>
  <div class="min-h-screen grid place-items-center p-6">
    <div class="w-full max-w-sm card shadow-sm">
      <h1 class="text-xl font-semibold mb-4 text-center">SecureAuth Login</h1>
      <form @submit.prevent="onSubmit" class="space-y-3">
        <div>
          <label for="email" class="block text-sm font-medium mb-1">Email</label>
          <input id="email" v-model="email" type="email" autocomplete="email" required class="form-control" />
        </div>
        <div>
          <label for="password" class="block text-sm font-medium mb-1">Password</label>
          <input id="password" v-model="password" type="password" autocomplete="current-password" required class="form-control" />
        </div>
        <button :disabled="loading" class="w-full btn-primary">
          {{ loading ? 'Signing in...' : 'Sign in' }}
        </button>
        <p v-if="error" class="text-sm text-red-600" aria-live="polite">{{ error }}</p>
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
