<template>
  <div class="min-h-screen grid place-items-center p-6">
    <div class="w-full max-w-md card shadow-sm">
      <h1 class="text-xl font-semibold mb-4 text-center">Create your account</h1>
      <form @submit.prevent="onSubmit" class="space-y-3">
        <div>
          <label for="company" class="block text-sm font-medium mb-1">Company name</label>
          <input id="company" v-model="company" type="text" autocomplete="organization" required class="form-control" />
        </div>
        <div>
          <label for="email" class="block text-sm font-medium mb-1">Email</label>
          <input id="email" v-model="email" type="email" autocomplete="email" required class="form-control" />
        </div>
        <div>
          <label for="password" class="block text-sm font-medium mb-1">Password</label>
          <input id="password" v-model="password" type="password" autocomplete="new-password" required class="form-control" />
        </div>
        <button :disabled="loading" class="w-full btn-primary">
          {{ loading ? 'Creating...' : 'Create account' }}
        </button>
        <p v-if="error" class="text-sm text-red-600" aria-live="polite">{{ error }}</p>
      </form>

      <div v-if="verificationToken" class="mt-4 text-sm bg-indigo-50 border border-indigo-100 rounded p-3">
        <p class="font-medium">Dev: Email verification token</p>
        <p class="break-all">{{ verificationToken }}</p>
        <RouterLink :to="{ path: '/verify-email', query: { token: verificationToken } }" class="text-indigo-600 hover:underline">Go verify email</RouterLink>
      </div>

      <p class="mt-4 text-sm text-center">
        Already have an account?
        <RouterLink to="/login" class="text-indigo-600 hover:underline">Sign in</RouterLink>
      </p>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { RouterLink } from 'vue-router'
import { register } from '../services/auth'

const company = ref('')
const email = ref('')
const password = ref('')
const loading = ref(false)
const error = ref('')
const verificationToken = ref('')

async function onSubmit() {
  loading.value = true
  error.value = ''
  verificationToken.value = ''
  try {
    const res = await register(company.value, email.value, password.value)
    verificationToken.value = res?.verification_token || ''
  } catch (e: any) {
    error.value = e?.response?.data?.error || 'Registration failed'
  } finally {
    loading.value = false
  }
}
</script>
