import { defineStore } from 'pinia'

interface AuthState {
  token: string | null
}

const STORAGE_KEY = 'session_token'

export const useAuthStore = defineStore('auth', {
  state: (): AuthState => ({
    token: typeof window !== 'undefined' ? localStorage.getItem(STORAGE_KEY) : null,
  }),
  getters: {
    isAuthenticated: (s) => !!s.token,
  },
  actions: {
    setToken(token: string) {
      this.token = token
      localStorage.setItem(STORAGE_KEY, token)
    },
    clear() {
      this.token = null
      localStorage.removeItem(STORAGE_KEY)
    },
  },
})
