<template>
  <div class="min-h-screen grid place-items-center p-6">
    <div class="w-full max-w-md bg-white border rounded p-6 shadow-sm">
      <h1 class="text-xl font-semibold mb-4 text-center">Create your account</h1>
      <form @submit.prevent="onSubmit" class="space-y-3">
        <div>
          <label class="block text-sm font-medium mb-1">Company name</label>
          <input v-model="company" type="text" required class="w-full border rounded px-3 py-2 focus:outline-none focus:ring focus:ring-indigo-200" />
        </div>
        <div>
          <label class="block text-sm font-medium mb-1">Email</label>
          <input v-model="email" type="email" required class="w-full border rounded px-3 py-2 focus:outline-none focus:ring focus:ring-indigo-200" />
        </div>
        <div>
          <label class="block text-sm font-medium mb-1">Password</label>
          <input v-model="password" type="password" required class="w-full border rounded px-3 py-2 focus:outline-none focus:ring focus:ring-indigo-200" />
        </div>
        <button :disabled="loading" class="w-full bg-indigo-600 hover:bg-indigo-700 text-white rounded px-3 py-2 disabled:opacity-70">
          {{ loading ? 'Creating...' : 'Create account' }}
        </button>
        <p v-if="error" class="text-sm text-red-600">{{ error }}</p>
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
