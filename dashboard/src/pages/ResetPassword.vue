<template>
  <div class="min-h-screen grid place-items-center p-6">
    <div class="w-full max-w-md card shadow-sm">
      <h1 class="text-xl font-semibold mb-4 text-center">Reset password</h1>
      <form @submit.prevent="onSubmit" class="space-y-3">
        <div>
          <label for="token" class="block text-sm font-medium mb-1">Reset token</label>
          <input id="token" v-model="token" type="text" autocomplete="one-time-code" required class="form-control" />
        </div>
        <div>
          <label for="new_password" class="block text-sm font-medium mb-1">New password</label>
          <input id="new_password" v-model="password" type="password" autocomplete="new-password" required class="form-control" />
        </div>
        <div>
          <label for="confirm_password" class="block text-sm font-medium mb-1">Confirm new password</label>
          <input id="confirm_password" v-model="confirm" type="password" autocomplete="new-password" required class="form-control" />
        </div>
        <button :disabled="loading" class="w-full btn-primary">
          {{ loading ? 'Updating...' : 'Update password' }}
        </button>
        <p v-if="error" class="text-sm text-red-600" aria-live="polite">{{ error }}</p>
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
