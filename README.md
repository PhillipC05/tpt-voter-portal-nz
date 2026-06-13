# Online Voter Registration & Polling — App 3

RealMe-verified local body polling with zero-knowledge-style ballot anonymity and
public auditability. Electoral Act 1993 scope: **local body polling only** — not for
Parliamentary elections. Coordinate with the Electoral Commission before production use.

## Architecture

```
┌─────────────────────────────────────────────────────┐
│                  Next.js Frontend                    │
│  (TypeScript, React, Tailwind CSS)                  │
└──────────────────┬──────────────────────────────────┘
                   │ HTTP (JSON API)
┌──────────────────▼──────────────────────────────────┐
│                 Go Backend (Chi)                    │
│  ┌────────────┐  ┌────────────┐  ┌───────────────┐  │
│  │  Auth      │  │  Services  │  │  Repository   │  │
│  │  Handlers  │  │  Layer     │  │  (pgx)        │  │
│  └─────┬──────┘  └─────┬──────┘  └──────┬────────┘  │
│        │               │                │           │
│        ▼               ▼                ▼           │
│  RealMe SAML    Registration /    PostgreSQL        │
│  (Verified)     Poll / Tally      (pgxpool)         │
└─────────────────────────────────────────────────────┘
```

## Security Model

| Concern | Solution |
|---------|----------|
| One vote per person | UNIQUE(poll\_id, voter\_token) constraint |
| No PII in ballot store | voter\_token = sha256(flt\_hash + poll\_id + poll\_salt) |
| FLT never stored | Only sha256(FLT) is kept in the voters table |
| Receipt verification | Random receipt\_token returned to voter after casting |
| Tamper-evidence | audit\_root = sha256 of sorted ballot commitments |
| Cross-poll unlinkability | Per-poll random salt isolates voter tokens across polls |

## API Endpoints

### Public (no auth)

| Method | Path | Description |
|--------|------|-------------|
| GET | /polls | List open polls |
| GET | /polls/{id} | Poll details |
| GET | /polls/{id}/results | Vote tally + audit root |
| GET | /polls/{id}/audit | Full ballot list for independent verification |
| GET | /polls/{id}/verify?receipt=TOKEN | Verify a receipt token |
| GET | /health | Health check |

### Authentication

| Method | Path | Description |
|--------|------|-------------|
| GET | /auth/login | Initiate RealMe login |
| GET | /auth/callback | SAML callback |
| GET | /auth/logout | Clear session |
| GET | /auth/metadata | SAML SP metadata XML |
| GET | /auth/status | Auth status (requires login) |

### Protected (requires RealMe Verified identity)

| Method | Path | Description |
|--------|------|-------------|
| POST | /register | Register as a voter (idempotent) |
| GET | /register/status | Check registration eligibility |
| POST | /polls/{id}/vote | Cast a ballot |
| GET | /polls/{id}/my-receipt | Retrieve your receipt token |
| POST | /polls | Create a poll (admin; scope to role in production) |

## Quick Start

### 1. Start Infrastructure

From the project root:

```bash
make dev
```

### 2. Apply Database Migration

```bash
DATABASE_URL="postgres://tptnz:tptnz_dev@localhost:5432/tptnz?sslmode=disable" \
  atlas schema apply --dir "file://packages/app-voter-portal/migrations" \
  --url "$DATABASE_URL" --auto-approve
```

Or manually:

```bash
psql "postgres://tptnz:tptnz_dev@localhost:5432/tptnz?sslmode=disable" \
  -f packages/app-voter-portal/migrations/001_init.sql
```

### 3. Run the Backend

```bash
cd packages/app-voter-portal
DATABASE_URL="postgres://tptnz:tptnz_dev@localhost:5432/tptnz?sslmode=disable" \
  go run ./cmd/server
```

API available at `http://localhost:8080`.

### 4. Run the Frontend

```bash
cd packages/app-voter-portal/web
pnpm install
pnpm dev
```

Frontend at `http://localhost:3006`.

### 5. Docker Compose

```bash
docker compose -f docker-compose.yml \
  -f packages/app-voter-portal/docker-compose.yml up
```

## Testing

```bash
cd packages/app-voter-portal

# Unit tests (no database required)
go test ./...

# With race detection
go test -race ./...

# Specific test
go test -v ./internal/services/ -run TestComputeAuditRoot
```

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| LISTEN_ADDR | :8080 | Server listen address |
| DATABASE_URL | postgres://tptnz:tptnz_dev@... | PostgreSQL connection string |
| REALME_ENVIRONMENT | mts | mts, ite, or production |
| REALME_CERT_FILE | certs/sp.crt | SP certificate |
| REALME_KEY_FILE | certs/sp.key | SP private key |
| REALME_ENTITY_ID | http://localhost:8080/auth/metadata | SAML entity ID |
| REALME_ACS_URL | http://localhost:8080/auth/callback | SAML ACS URL |
| REALME_IDP_METADATA_FILE | (empty) | Local IdP metadata file |
| REALME_IDP_METADATA_URL | http://localhost:8081/metadata | IdP metadata URL |

## Audit Verification (Independent)

To independently verify a poll tally without trusting this server:

1. Fetch `GET /polls/{id}/audit` — get the full ballot list.
2. Extract all `commitment` values.
3. Sort them lexicographically.
4. Concatenate and compute `sha256` of the result.
5. Compare with `auditRoot` from `GET /polls/{id}/results`.

A voter proves their vote was counted by finding their `receiptToken` in the list — without revealing their choice to anyone else (the choice index is visible only with the receipt).

## RealMe Registration

To use this app with real RealMe identities (ITE or Production environments),
you must register a Service Provider with the Department of Internal Affairs.

### MTS (Messaging Test Site) — Development

1. Generate a self-signed certificate and key:
   ```bash
   mkdir -p certs
   openssl req -x509 -nodes -days 365 -newkey rsa:2048 \
     -keyout certs/sp.key -out certs/sp.crt \
     -subj "/CN=localhost" -addext "subjectAltName=DNS:localhost"
   ```

2. In MTS, no formal registration is needed — use the mock IdP in
   `packages/realme-go/testenv/` for local development.

3. Start the mock IdP:
   ```bash
   cd packages/realme-go
   go run ./testenv/ -addr :8081
   ```

4. Configure the app to use the mock IdP:
   ```bash
   REALME_IDP_METADATA_URL=http://localhost:8081/metadata
   ```

### ITE (Integration Test Environment) — Pre-Production

1. Log in to the [RealMe Developer Portal](https://developers.realme.govt.nz/)
   and register a new service.
2. Submit your SP metadata XML (available at `GET /auth/metadata`) to DIA.
3. DIA will provide the ITE IdP metadata URL.
4. Generate a proper certificate (not self-signed) using the naming convention:
   `ite.{service-name}.{org-domain}.nz`
5. Configure environment variables:
   ```bash
   REALME_ENVIRONMENT=ite
   REALME_CERT_FILE=certs/ite.sp.crt
   REALME_KEY_FILE=certs/ite.sp.key
   REALME_IDP_METADATA_URL=<DIA-provided-ITE-url>
   ```

### Production

Follow the ITE steps above, substituting:
```bash
REALME_ENVIRONMENT=production
REALME_CERT_FILE=certs/prod.sp.crt
REALME_KEY_FILE=certs/prod.sp.key
REALME_IDP_METADATA_URL=<DIA-provided-prod-url>
```

**Electoral Commission coordination required** before production deployment. This system may only be used for local body polls and must be approved by the relevant local authority and the Electoral Commission.

## Regulatory Notes

- **Electoral Act 1993**: This system is scoped to local body polls only. Parliamentary elections are governed by the Electoral Commission under separate legislation.
- **Privacy Act 2020**: No name, DOB, or address is stored. Only a hash of the RealMe FLT, which is itself a pseudonymous per-service identifier.
- **RealMe Verified Identity**: Required for registration and voting. The Assertion Service assurance level (LevelVerified) is enforced by the `RequireVerified()` middleware.
