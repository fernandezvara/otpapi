import api from './api'

export async function login(email: string, password: string) {
  const { data } = await api.post('/auth/login', { email, password })
  return data
}

export async function logout() {
  const { data } = await api.post('/auth/logout')
  return data
}

export async function register(company_name: string, email: string, password: string) {
  const { data } = await api.post('/auth/register', { company_name, email, password })
  return data
}

export async function verifyEmail(token: string) {
  const { data } = await api.post('/auth/verify_email', { token })
  return data
}

export async function requestPasswordReset(email: string) {
  const { data } = await api.post('/auth/password/request_reset', { email })
  return data
}

export async function resetPassword(token: string, new_password: string) {
  const { data } = await api.post('/auth/password/reset', { token, new_password })
  return data
}
