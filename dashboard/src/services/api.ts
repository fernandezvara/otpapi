import axios from 'axios'
import { useAuthStore } from '../stores/auth'

const api = axios.create({
  baseURL: '/api/v1',
})

api.interceptors.request.use((config) => {
  // attach session token if available
  try {
    const auth = useAuthStore()
    const token = auth?.token || (typeof window !== 'undefined' ? localStorage.getItem('session_token') : null)
    if (token) {
      config.headers = config.headers || {}
      ;(config.headers as any)['X-Session-Token'] = token
    }
  } catch {
    /* ignore auth store access errors */
  }
  return config
})

api.interceptors.response.use(
  (res) => res,
  (err) => {
    const status = err?.response?.status
    if (status === 401 && typeof window !== 'undefined') {
      try {
        const auth = useAuthStore()
        auth.clear()
      } catch {
        localStorage.removeItem('session_token')
      }
      const url = new URL(window.location.href)
      const current = url.pathname + url.search + url.hash
      if (!current.startsWith('/login')) {
        window.location.href = `/login?redirect=${encodeURIComponent(current)}`
      }
    }
    return Promise.reject(err)
  }
)

export default api
