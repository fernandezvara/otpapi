<template>
  <div class="min-h-screen grid place-items-center p-6">
    <div class="w-full max-w-md bg-white border rounded p-6 shadow-sm">
      <h1 class="text-xl font-semibold mb-4 text-center">Verify your email</h1>
      <form @submit.prevent="onSubmit" class="space-y-3">
        <div>
          <label class="block text-sm font-medium mb-1">Verification token</label>
          <input v-model="token" type="text" required class="w-full border rounded px-3 py-2 focus:outline-none focus:ring focus:ring-indigo-200" />
        </div>
        <button :disabled="loading" class="w-full bg-indigo-600 hover:bg-indigo-700 text-white rounded px-3 py-2 disabled:opacity-70">
          {{ loading ? 'Verifying...' : 'Verify email' }}
        </button>
        <p v-if="error" class="text-sm text-red-600">{{ error }}</p>
      </form>

      <div v-if="success" class="mt-4 text-sm bg-green-50 border border-green-100 rounded p-3">
        <p class="font-medium">Email verified!</p>
        <RouterLink to="/login" class="text-indigo-600 hover:underline">Go to Sign in</RouterLink>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import { verifyEmail } from '../services/auth'
import { RouterLink } from 'vue-router'

const token = ref('')
const loading = ref(false)
const error = ref('')
const success = ref(false)
const route = useRoute()

onMounted(() => {
  const t = route.query.token as string
  if (t) token.value = t
})

async function onSubmit() {
  loading.value = true
  error.value = ''
  success.value = false
  try {
    await verifyEmail(token.value)
    success.value = true
  } catch (e: any) {
    error.value = e?.response?.data?.error || 'Verification failed'
  } finally {
    loading.value = false
  }
}
</script>
