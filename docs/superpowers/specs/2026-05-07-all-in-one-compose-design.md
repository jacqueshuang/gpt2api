# All-in-One Compose Deployment Design

Date: 2026-05-07

## Goal

Add a compose-based deployment path for the already-built all-in-one GHCR image. The compose file runs only the application container; MySQL and Redis remain external dependencies supplied through environment variables.

## Decisions

| Topic | Decision |
|---|---|
| Compose shape | One `gpt2api` service using the all-in-one image |
| Image | `${GPT2API_IMAGE:-ghcr.io/432539/gpt2api:latest}` |
| External services | MySQL and Redis stay outside this compose file |
| Env file | Add `deploy/env/.env.all-in-one.example`; users copy it to an ignored local env file |
| Public ports | `${KLEIN_USER_WEB_PORT:-17080}:17080`, `${KLEIN_ADMIN_WEB_PORT:-17088}:17088`, `${KLEIN_OPENAI_PORT:-17200}:17200` |
| Volumes | `gpt2api-storage:/app/storage`, `gpt2api-logs:/app/logs` |
| Healthcheck | `wget -qO- http://127.0.0.1:17080/healthz` |

## Files

| File | Purpose |
|---|---|
| `deploy/docker-compose.all-in-one.yml` | Direct compose deployment for the GHCR all-in-one image |
| `deploy/env/.env.all-in-one.example` | Safe environment template with placeholder secrets and external DB/Redis addresses |
| `README.md` | Document the compose startup path |
| `docs/06-部署与运维规范.md` | Document the operational compose startup path |

## Runtime configuration

The compose service passes these environment variables to the container:

| Variable | Purpose |
|---|---|
| `KLEIN_ENV` | Defaults to `prod` |
| `KLEIN_DB_DSN` | External MySQL DSN |
| `KLEIN_REDIS_ADDR` | External Redis address |
| `KLEIN_REDIS_PASSWORD` | Optional Redis password |
| `KLEIN_JWT_SECRET` | Access token secret, at least 32 bytes |
| `KLEIN_JWT_REFRESH_SECRET` | Refresh token secret, at least 32 bytes |
| `KLEIN_AES_KEY` | AES key, exactly 32 raw bytes or 64 hex chars |
| `KLEIN_CORS_ORIGINS` | Allowed frontend origins |
| `KLEIN_SERVER_OPENAI_HOST=127.0.0.1` | Keep the OpenAI backend internal to the container |
| `KLEIN_SERVER_OPENAI_PORT=17201` | Internal OpenAI backend port behind nginx |
| `KLEIN_LOG_DIR=/app/logs` | Persist application logs |

`deploy/env/.env.all-in-one.example` contains only safe placeholder values and must not contain real credentials.

## Usage

```bash
cd deploy
cp env/.env.all-in-one.example env/.env.all-in-one
# edit env/.env.all-in-one
docker compose --env-file env/.env.all-in-one -f docker-compose.all-in-one.yml up -d
```

## Verification

Local checks after adding the files:

```bash
docker compose --env-file deploy/env/.env.all-in-one.example -f deploy/docker-compose.all-in-one.yml config
```

Runtime checks after external MySQL and Redis are reachable:

```bash
curl http://localhost:17080/healthz
curl http://localhost:17088/healthz
curl http://localhost:17200/healthz
curl http://localhost:17200/v1/health
```

## Scope boundaries

This change does not alter the all-in-one image, application routes, existing source-build compose files, database initialization, or Redis setup. It only adds a direct compose wrapper for deployments that already have MySQL and Redis available.
