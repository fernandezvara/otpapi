<template>
  <div class="min-h-screen grid place-items-center p-6">
    <div class="w-full max-w-md card shadow-sm">
      <h1 class="text-xl font-semibold mb-4 text-center">Verify your email</h1>
      <form @submit.prevent="onSubmit" class="space-y-3">
        <div>
          <label for="token" class="block text-sm font-medium mb-1">Verification token</label>
          <input id="token" v-model="token" type="text" autocomplete="one-time-code" required class="form-control" />
        </div>
        <button :disabled="loading" class="w-full btn-primary">
          {{ loading ? 'Verifying...' : 'Verify email' }}
        </button>
        <p v-if="error" class="text-sm text-red-600" aria-live="polite">{{ error }}</p>
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
