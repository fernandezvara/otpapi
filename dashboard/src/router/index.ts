import { createRouter, createWebHistory, type RouteRecordRaw, type RouteLocationNormalized } from 'vue-router'
import Layout from '../components/layout/Layout.vue'
import Overview from '../pages/Overview.vue'
import ApiKeys from '../pages/ApiKeys.vue'
import MfaUsers from '../pages/MfaUsers.vue'
import Billing from '../pages/Billing.vue'
import Settings from '../pages/Settings.vue'
import Support from '../pages/Support.vue'
import Developer from '../pages/Developer.vue'
import Login from '../pages/Login.vue'
import Register from '../pages/Register.vue'
import VerifyEmail from '../pages/VerifyEmail.vue'
import ForgotPassword from '../pages/ForgotPassword.vue'
import ResetPassword from '../pages/ResetPassword.vue'
import { useAuthStore } from '../stores/auth'

const routes: RouteRecordRaw[] = [
  { path: '/', redirect: '/dashboard/overview' },
  { path: '/login', component: Login },
  { path: '/register', component: Register },
  { path: '/verify-email', component: VerifyEmail },
  { path: '/forgot-password', component: ForgotPassword },
  { path: '/reset-password', component: ResetPassword },
  {
    path: '/dashboard',
    component: Layout,
    meta: { requiresAuth: true },
    children: [
      { path: 'overview', component: Overview },
      { path: 'api-keys', component: ApiKeys },
      { path: 'mfa-users', component: MfaUsers },
      { path: 'billing', component: Billing },
      { path: 'settings', component: Settings },
      { path: 'support', component: Support },
      { path: 'developer', component: Developer },
    ],
  },
]

const router = createRouter({
  history: createWebHistory(),
  routes,
})

router.beforeEach((to: RouteLocationNormalized) => {
  const auth = useAuthStore()
  if (to.meta.requiresAuth && !auth.isAuthenticated) {
    return { path: '/login', query: { redirect: to.fullPath } }
  }
  if (to.path === '/login' && auth.isAuthenticated) {
    return { path: '/dashboard/overview' }
  }
})

export default router
