# All-in-One Docker Image and GHCR Workflow Design

Date: 2026-05-06

## Goal

Add a GitHub Actions workflow that builds and publishes a single Docker image to GitHub Container Registry. The image contains the Go backend processes, both frontend apps, and nginx. MySQL and Redis remain external runtime dependencies.

## Decisions

| Topic | Decision |
|---|---|
| Registry | GitHub Container Registry |
| Runtime shape | One Docker image / one container for app processes and nginx |
| External services | MySQL and Redis stay outside the image |
| Public ports | `17080` user, `17088` admin, `17200` OpenAI-compatible API |
| Image platforms | `linux/amd64`, `linux/arm64` |
| Tags | `latest` on `main`, semver tag on `v*`, and `sha-xxxxxxx` on every build |

## Architecture

Add an independent all-in-one packaging path without changing business logic or the existing Compose deployment.

| File | Purpose |
|---|---|
| `backend/pkg/config/config.go` | Adds `server.openai_host` / `KLEIN_SERVER_OPENAI_HOST` so the all-in-one OpenAI backend can bind to localhost |
| `backend/cmd/openai/main.go` | Uses the optional OpenAI bind host with the existing OpenAI port |
| `backend/configs/config.yaml` | Documents default empty OpenAI bind host for existing deployments |
| `Dockerfile.all-in-one` | Multi-stage build for backend binaries, frontend assets, and runtime image |
| `Dockerfile.all-in-one.dockerignore` | Limit the root Docker build context and prevent local secrets/caches from reaching builders |
| `deploy/all-in-one/nginx.conf` | nginx config for static frontend serving and localhost backend proxying |
| `deploy/all-in-one/supervisord.conf` | Process supervision for `api`, `admin`, `openai`, `worker`, and nginx |
| `.github/workflows/docker-image.yml` | Build and publish multi-arch image to GHCR |

## Runtime routing

| External endpoint | nginx behavior | Internal target |
|---|---|---|
| `:17080/` | Serve user SPA | `/usr/share/nginx/user` |
| `:17080/api/` | Proxy user API | `127.0.0.1:17180` |
| `:17080/healthz` | Proxy user API health | `127.0.0.1:17180/healthz` |
| `:17080/v1/` | Proxy OpenAI API | `127.0.0.1:17201` |
| `:17088/` | Serve admin SPA | `/usr/share/nginx/admin` |
| `:17088/admin/api/` | Proxy admin API | `127.0.0.1:17188` |
| `:17088/healthz` | Proxy admin API health | `127.0.0.1:17188/healthz` |
| `:17200/v1/` | Proxy OpenAI API | `127.0.0.1:17201` |
| `:17200/healthz` | Proxy OpenAI API health | `127.0.0.1:17201/healthz` |

The all-in-one runtime sets `KLEIN_SERVER_OPENAI_HOST=127.0.0.1` and `KLEIN_SERVER_OPENAI_PORT=17201` for backend processes. This avoids a same-container port conflict: nginx owns the public container port `17200`, while the OpenAI backend listens only behind nginx on localhost port `17201`.

## Docker build design

`Dockerfile.all-in-one` uses three stages:

1. Go builder: build `api`, `admin`, `openai`, and `worker` from `backend/cmd/*`.
2. Frontend builder: run `corepack enable`, install pnpm dependencies with the lockfile, then build `@kleinai/user` and `@kleinai/admin` explicitly so both `apps/*/dist` directories exist.
3. Runtime: Alpine with nginx, supervisor, CA certificates, and timezone data.

Runtime layout:

| Path | Content |
|---|---|
| `/app/api` | User API binary |
| `/app/admin` | Admin API binary |
| `/app/openai` | OpenAI-compatible API binary |
| `/app/worker` | Worker binary |
| `/app/configs` | Backend config YAML files |
| `/app/logs` | Runtime logs, volume-friendly |
| `/app/storage` | Runtime storage, volume-friendly |
| `/usr/share/nginx/user` | User frontend dist |
| `/usr/share/nginx/admin` | Admin frontend dist |

The container entrypoint is `supervisord` in foreground mode. Supervisor restarts backend processes if they exit unexpectedly. The Go programs set `KLEIN_LOGGER_CONSOLE=true` under supervisor so production containers still emit application logs to Docker stdout/stderr.

## GitHub Actions design

Workflow triggers:

| Trigger | Behavior |
|---|---|
| Push to `main` | Build and push `latest` plus `sha-xxxxxxx` |
| Push tag `v*` | Build and push the tag plus `sha-xxxxxxx` |
| Pull request | Build only, no push |

Required workflow permissions:

```yaml
permissions:
  contents: read
  packages: write
```

The workflow uses Docker Buildx and `docker/metadata-action` to generate GHCR image tags.

## Runtime configuration

The image does not contain secrets. Production runs must provide required backend environment variables, including:

| Variable | Purpose |
|---|---|
| `KLEIN_ENV=prod` | Enable production config validation |
| `KLEIN_SERVER_OPENAI_HOST=127.0.0.1` | Bind the internal OpenAI backend to localhost so nginx is the only container-network entrypoint |
| `KLEIN_SERVER_OPENAI_PORT=17201` | Internal OpenAI backend port used behind nginx |
| `KLEIN_SNOWFLAKE_NODE_ID` | Optional per-process node id set by supervisor for all-in-one runtime |
| `KLEIN_DB_DSN` | MySQL connection string |
| `KLEIN_REDIS_ADDR` | Redis address |
| `KLEIN_JWT_SECRET` | Access token secret, at least 32 bytes |
| `KLEIN_JWT_REFRESH_SECRET` | Refresh token secret, at least 32 bytes |
| `KLEIN_AES_KEY` | AES key, at least 32 bytes |
| `KLEIN_CORS_ORIGINS` | Allowed frontend origins |
| `KLEIN_LOGGER_CONSOLE=true` | Supervisor sets this for Go programs so prod logs also reach Docker stdout/stderr |
| `KLEIN_LOG_DIR=/app/logs` | Log directory |

Recommended persistent mounts:

| Mount | Reason |
|---|---|
| `/app/storage` | Generated files and provider state |
| `/app/logs` | Application logs |

## Example run command

```bash
docker run -d \
  --name gpt2api \
  -p 17080:17080 \
  -p 17088:17088 \
  -p 17200:17200 \
  -e KLEIN_ENV=prod \
  -e KLEIN_SERVER_OPENAI_HOST=127.0.0.1 \
  -e KLEIN_SERVER_OPENAI_PORT=17201 \
  -e KLEIN_DB_DSN='klein:password@tcp(mysql-host:3306)/klein_ai?charset=utf8mb4&parseTime=True&loc=Local' \
  -e KLEIN_REDIS_ADDR='redis-host:6379' \
  -e KLEIN_JWT_SECRET='replace-with-at-least-32-bytes-secret' \
  -e KLEIN_JWT_REFRESH_SECRET='replace-with-at-least-32-bytes-secret' \
  -e KLEIN_AES_KEY='replace-with-at-least-32-bytes-secret' \
  -e KLEIN_CORS_ORIGINS='http://localhost:17080,http://localhost:17088' \
  -e KLEIN_LOG_DIR=/app/logs \
  -v gpt2api-storage:/app/storage \
  -v gpt2api-logs:/app/logs \
  ghcr.io/<owner>/<repo>:latest
```

## Verification

Closest local checks:

```bash
docker buildx build --platform linux/amd64 -f Dockerfile.all-in-one -t gpt2api:all-in-one --load .
docker run --rm gpt2api:all-in-one nginx -t
```

Runtime checks after MySQL and Redis are available:

```bash
curl http://localhost:17080/healthz
curl http://localhost:17088/healthz
curl http://localhost:17200/healthz
curl http://localhost:17200/v1/health
```

Internal diagnostic check for the OpenAI backend port:

```bash
docker exec gpt2api sh -lc 'wget -qO- http://127.0.0.1:17201/v1/health'
```

CI checks:

| Context | Expected result |
|---|---|
| Pull request | Multi-arch build succeeds without pushing |
| Push to `main` | Image is pushed with `latest` and `sha-xxxxxxx` |
| Push tag `v*` | Image is pushed with tag and `sha-xxxxxxx` |

## Scope boundaries

This design does not change application routes, frontend router base paths, database setup, Redis setup, or the existing Compose deployment. It only adds a new single-image packaging and publishing path.
