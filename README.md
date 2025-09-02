# OTP MFA API (Go)

A minimal Multi-Factor Authentication (MFA) API written in Go, featuring TOTP registration, QR code generation, and OTP validation. The project follows a standard Go module layout with a clear separation of concerns under `internal/` and an entrypoint at `cmd/api/main.go`.

## Features

- TOTP-based MFA registration and validation
- QR code PNG generation for authenticator apps
- AES-256-GCM encryption of TOTP secrets and backup codes
- API key authentication (hashed, stored server-side)
- Bootstrap endpoint to create a test customer and API key
- Dockerfile and Docker Compose for local development

## Project Structure

```
.
├─ cmd/
│  └─ api/
│     └─ main.go          # Entrypoint (Gin setup, routes, middleware)
├─ internal/
│  ├─ api/
│  │  ├─ bootstrap.go     # /api/v1/bootstrap/seed
│  │  └─ mfa.go           # /api/v1/mfa/... endpoints
│  ├─ config/
│  │  └─ config.go        # Environment configuration
│  ├─ crypto/
│  │  └─ crypto.go        # AES-256-GCM encrypt/decrypt
│  ├─ db/
│  │  └─ db.go            # DB init + runtime schema creation
│  ├─ keys/
│  │  └─ keys.go          # API key hashing and random utils
│  └─ middleware/
│     └─ api_key_auth.go  # Bearer API key middleware
├─ tools/
│  └─ generate_otp.go     # Helper for generating TOTP codes from secrets
├─ index.html             # Simple test UI
├─ Dockerfile
├─ docker-compose.yaml
├─ go.mod / go.sum
└─ README.md
```

## Requirements

- Go 1.24+
- PostgreSQL 15+ (via Docker Compose or installed locally)

## Configuration (Environment Variables)

- `DATABASE_URL` – PostgreSQL connection string.
  - Example: `host=localhost port=5432 user=postgres password=postgres dbname=mfa_mvp sslmode=disable`
- `ENCRYPTION_KEY` – 32-character key for AES-256-GCM. Required.
- `PORT` – API port, default `8080`.
- `BOOTSTRAP_TOKEN` – token to authorize the bootstrap endpoint.
- `ISSUER` – default issuer fallback used for legacy records when issuer is empty.

Note: Encryption key must be exactly 32 characters.

## Running Locally (without Docker)

1. Ensure PostgreSQL is running and reachable via `DATABASE_URL`.
2. Set env vars (at minimum `DATABASE_URL` and a 32-char `ENCRYPTION_KEY`).
3. Run the API:

```bash
go run ./cmd/api
```

The server starts on `:8080` by default.

## Running with Docker Compose

Build and start:

```bash
docker compose up --build
```

Compose services:

- `postgres`: PostgreSQL 15-alpine
- `mfa-api`: the API built from this repo

The API is available at `http://localhost:8080`.

## API Overview

All MFA endpoints require a Bearer API key in the `Authorization` header: `Authorization: Bearer <api_key>`.

### Health

- `GET /healthz`
- Response: `{"status":"ok"}`

### Bootstrap (create customer + API key)

- `POST /api/v1/bootstrap/seed`
- Headers: `X-Bootstrap-Token: <BOOTSTRAP_TOKEN>`
- Body:

```json
{
  "company_name": "Acme Inc",
  "email": "admin@acme.com",
  "key_name": "Test Key",
  "environment": "test"
}
```

- 201 Response:

```json
{
  "customer_id": "<uuid>",
  "api_key_id": "<uuid>",
  "api_key": "sk_test_..." // shown once
}
```

### Register MFA

- `POST /api/v1/mfa/register`
- Headers: `Authorization: Bearer <api_key>`
- Body:

```json
{
  "id": "user-123",
  "issuer": "Acme Inc",
  "account_name": "user@example.com"
}
```

- 201 Response:

```json
{
  "qr_code_url": "http://localhost:8080/api/v1/mfa/user-123/qr",
  "backup_codes": ["XXXXXXXX", "..."]
}
```

Notes:

- Secrets and backup codes are encrypted at rest.
- `account_name` may be set to "-" to omit it from the QR label (`issuer` only).

### Get QR Code PNG

- `GET /api/v1/mfa/:id/qr`
- Headers: `Authorization: Bearer <api_key>`
- Response: `image/png`

The QR contains an `otpauth://` TOTP URL using stored `issuer` and `account_name`.
For legacy records with empty issuer, the `ISSUER` env fallback is used.

### Validate OTP

- `POST /api/v1/mfa/:id`
- Headers: `Authorization: Bearer <api_key>`
- Body:

```json
{ "otp": "123456" }
```

- 200 Response:

```json
{ "valid": true, "message": "OTP is valid" }
```

- 401 Response:

```json
{ "valid": false, "message": "Invalid OTP" }
```

## Security Notes

- API keys are hashed (SHA-256) and stored server-side; only shown once on creation.
- TOTP secret and backup codes are encrypted with AES-256-GCM.
- The bootstrap endpoint is protected by `X-Bootstrap-Token`.
- CORS is permissive for MVP; consider tightening in production.
- Configure Gin trusted proxies for deployments behind proxies/load balancers.
- For production, do not return secrets in any response.

## Database

- Runtime schema creation is performed on startup (see `internal/db/db.go`).
- Replace with a proper migration system for production (e.g., `golang-migrate`).

## Development Tips

- Build binary:

```bash
go build -o bin/api ./cmd/api
```

- Lint/format (example):

```bash
go fmt ./...
go vet ./...
```

- Test helper: `tools/generate_otp.go` can generate TOTP codes from a known secret.

## Roadmap

- Migrations instead of runtime schema
- Unit/integration tests
- CORS hardening and trusted proxies
- Rate limiting & permissions
- API key lifecycle management
- OpenAPI/Swagger documentation
