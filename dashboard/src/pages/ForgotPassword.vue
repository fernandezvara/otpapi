<template>
  <div class="min-h-screen grid place-items-center p-6">
    <div class="w-full max-w-md card shadow-sm">
      <h1 class="text-xl font-semibold mb-4 text-center">Forgot password</h1>
      <form @submit.prevent="onSubmit" class="space-y-3">
        <div>
          <label for="email" class="block text-sm font-medium mb-1">Email</label>
          <input id="email" v-model="email" type="email" autocomplete="email" required class="form-control" />
        </div>
        <button :disabled="loading" class="w-full btn-primary">
          {{ loading ? 'Sending...' : 'Send reset email' }}
        </button>
        <p v-if="error" class="text-sm text-red-600" aria-live="polite">{{ error }}</p>
      </form>

      <div v-if="sent" class="mt-4 text-sm bg-indigo-50 border border-indigo-100 rounded p-3">
        <p class="font-medium">If the email exists, a reset token has been generated.</p>
        <p class="mt-1">Dev: token may be returned directly by the API in development.</p>
        <div v-if="devToken" class="mt-2">
          <p class="font-medium">Dev reset token</p>
          <p class="break-all">{{ devToken }}</p>
          <RouterLink :to="{ path: '/reset-password', query: { token: devToken } }" class="text-indigo-600 hover:underline">Go reset password</RouterLink>
        </div>
      </div>

      <p class="mt-4 text-sm text-center">
        <RouterLink to="/login" class="text-indigo-600 hover:underline">Back to Sign in</RouterLink>
      </p>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { RouterLink } from 'vue-router'
import { requestPasswordReset } from '../services/auth'

const email = ref('')
const loading = ref(false)
const error = ref('')
const sent = ref(false)
const devToken = ref('')

async function onSubmit() {
  loading.value = true
  error.value = ''
  sent.value = false
  devToken.value = ''
  try {
    const res = await requestPasswordReset(email.value)
    sent.value = true
    devToken.value = res?.reset_token || ''
  } catch (e: any) {
    // API returns 200 even if not found, but in case of error show generic message
    error.value = e?.response?.data?.error || 'Request failed'
  } finally {
    loading.value = false
  }
}
</script>
