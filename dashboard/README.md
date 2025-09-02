# SecureAuth Dashboard (Vue 3 + Vite + Tailwind + Pinia + Vue Router)

This is the customer portal for the MFA-as-a-Service platform.

## Prerequisites
- Node 18+
- Go backend running locally on :8080

## Install & Run (Frontend)
```bash
# from repo root or dashboard/
cd dashboard
npm install
npm run dev
```
Vite dev server runs at http://localhost:5173 and proxies API calls to http://localhost:8080.

## Run (Backend) with CORS for Vite
Make sure the backend allows the Vite origin. Example:
```bash
export CORS_ALLOWED_ORIGINS="http://localhost:5173,http://localhost:8080"
# in repo root
go run ./cmd/api
```

## Auth
- POST /api/v1/auth/login returns { session_token, expires_at }
- Frontend stores the token and sends it via X-Session-Token header using Axios interceptors.
- Protected routes are under /dashboard/*; unauthenticated users are redirected to /login.
- Logout calls POST /api/v1/auth/logout and clears local session.

## Next steps
- Wire API Keys UI to /api/v1/console/keys endpoints
- MFA Users list/search
- Billing usage charts
- WebSocket-based real-time analytics
