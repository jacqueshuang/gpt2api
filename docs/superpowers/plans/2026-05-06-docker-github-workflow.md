# All-in-One Docker GHCR Workflow Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a single Docker image packaging path and GitHub Actions workflow that build the full app stack into GHCR.

**Architecture:** The new all-in-one image builds the four Go backend binaries and both Vite frontend apps, then runs them in one Alpine-based runtime with nginx and supervisord. nginx owns public container ports `17080`, `17088`, and `17200`; the OpenAI backend is moved to localhost `127.0.0.1:17201` via `KLEIN_SERVER_OPENAI_HOST=127.0.0.1` and `KLEIN_SERVER_OPENAI_PORT=17201` to avoid a same-container port conflict and prevent direct container-network access.

**Tech Stack:** Docker Buildx, GitHub Actions, GHCR, Go 1.24, Node 20, pnpm 9.7, Vite, nginx, supervisord, Alpine Linux.

---

## File structure

| Path | Action | Responsibility |
|---|---|---|
| `deploy/all-in-one/nginx.conf` | Create | Serve user/admin static assets and proxy API routes to localhost backend processes. |
| `deploy/all-in-one/supervisord.conf` | Create | Start and restart `api`, `admin`, `openai`, `worker`, and nginx in one container. |
| `Dockerfile.all-in-one` | Create | Build backend and frontend artifacts, assemble runtime image, expose public ports. |
| `Dockerfile.all-in-one.dockerignore` | Create | Limit root Docker build context and prevent secrets/caches from reaching local or remote builders. |
| `.github/workflows/docker-image.yml` | Create | Build PR images and push multi-arch GHCR images on `main`/`v*`. |
| `docs/superpowers/specs/2026-05-06-docker-github-workflow-design.md` | Already updated | Design source of truth. Includes the OpenAI internal port correction. |

## Implementation notes

- Do not modify existing business code, frontend router config, or existing Compose files.
- Keep MySQL and Redis external.
- Do not put secrets in Dockerfile, nginx config, supervisor config, or workflow.
- Use a Dockerfile-specific ignore file for the root-context all-in-one build, because `frontend/.dockerignore` does not apply when the Docker context is the repository root.
- Use exact public ports: `17080`, `17088`, `17200`.
- Use internal OpenAI backend host `127.0.0.1` and port `17201`, not wildcard `:17200`, because nginx binds public port `17200` in the same container and must remain the only container-network entrypoint.
- Use `KLEIN_SNOWFLAKE_NODE_ID`, not `KLEIN_NODE_ID`, because the config key is `snowflake.node_id` and viper maps it to `KLEIN_SNOWFLAKE_NODE_ID`.
- Keep the image root-owned; the existing app writes to `/app/logs` and `/app/storage`, so create those directories during build.
- The project root is `/Users/huangkunhuang/Public/程序工程目录/GO/gpt2api`.

---

### Task 1: Add all-in-one nginx config

**Files:**
- Create: `deploy/all-in-one/nginx.conf`

- [ ] **Step 1: Create the deploy directory**

Run:

```bash
ls -la "/Users/huangkunhuang/Public/程序工程目录/GO/gpt2api/deploy"
mkdir -p "/Users/huangkunhuang/Public/程序工程目录/GO/gpt2api/deploy/all-in-one"
```

Expected: `deploy` exists, and `deploy/all-in-one` is created if missing.

- [ ] **Step 2: Write nginx config**

Create `deploy/all-in-one/nginx.conf` with exactly this content:

```nginx
server {
    listen 17080;
    server_name _;

    root /usr/share/nginx/user;
    index index.html;

    gzip on;
    gzip_types text/plain text/css application/javascript application/json image/svg+xml;
    client_max_body_size 30m;

    location ^~ /api/ {
        proxy_pass http://127.0.0.1:17180;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_read_timeout 90s;
    }

    location ^~ /v1/ {
        proxy_pass http://127.0.0.1:17201;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_buffering off;
        proxy_read_timeout 600s;
    }

    location / {
        try_files $uri /index.html;
    }

    location ~* \.(?:css|js|woff2?|ttf|svg|png|jpg|jpeg|webp|avif)$ {
        expires 30d;
        add_header Cache-Control "public, immutable";
    }
}

server {
    listen 17088;
    server_name _;

    root /usr/share/nginx/admin;
    index index.html;

    gzip on;
    gzip_types text/plain text/css application/javascript application/json image/svg+xml;
    client_max_body_size 30m;

    location ^~ /admin/api/ {
        proxy_pass http://127.0.0.1:17188;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_read_timeout 90s;
    }

    location / {
        try_files $uri /index.html;
        add_header Cache-Control "no-store, no-cache, must-revalidate, proxy-revalidate, max-age=0" always;
        add_header Pragma "no-cache" always;
        add_header Expires "0" always;
    }

    location ~* \.(?:css|js|woff2?|ttf|svg|png|jpg|jpeg|webp|avif)$ {
        expires 30d;
        add_header Cache-Control "public, immutable";
    }
}

server {
    listen 17200;
    server_name _;

    client_max_body_size 30m;

    location ^~ /v1/ {
        proxy_pass http://127.0.0.1:17201;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_buffering off;
        proxy_read_timeout 600s;
    }

    location /healthz {
        proxy_pass http://127.0.0.1:17201/healthz;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }
}
```

- [ ] **Step 3: Validate nginx config syntax with Docker**

Run:

```bash
docker run --rm \
  -v "/Users/huangkunhuang/Public/程序工程目录/GO/gpt2api/deploy/all-in-one/nginx.conf:/etc/nginx/conf.d/default.conf:ro" \
  nginx:1.27-alpine nginx -t
```

Expected output contains:

```text
nginx: the configuration file /etc/nginx/nginx.conf syntax is ok
nginx: configuration file /etc/nginx/nginx.conf test is successful
```

- [ ] **Step 4: Commit nginx config**

Run:

```bash
git add deploy/all-in-one/nginx.conf
git commit -m "$(cat <<'EOF'
feat: add all-in-one nginx routing
EOF
)"
```

Expected: commit succeeds. If the user has not explicitly asked to commit during execution, skip this step and report it as intentionally skipped.

---

### Task 2: Add supervisord process config

**Files:**
- Create: `deploy/all-in-one/supervisord.conf`
- Modify: `backend/pkg/config/config.go`
- Modify: `backend/configs/config.yaml`
- Create: `backend/pkg/config/config_test.go`

Create `backend/pkg/config/config_test.go` with exactly this content:

```go
package config

import "testing"

func TestLoadInternalMapsOpenAIHostEnv(t *testing.T) {
	t.Setenv("KLEIN_ENV", "dev")
	t.Setenv("KLEIN_SERVER_OPENAI_HOST", "127.0.0.1")

	cfg, err := loadInternal()
	if err != nil {
		t.Fatalf("loadInternal() error = %v", err)
	}
	if cfg.Server.OpenAIHost != "127.0.0.1" {
		t.Fatalf("Server.OpenAIHost = %q, want %q", cfg.Server.OpenAIHost, "127.0.0.1")
	}
}
```

Run:

```bash
go -C "/Users/huangkunhuang/Public/程序工程目录/GO/gpt2api/backend" test ./pkg/config
```

Expected before implementation: build fails because `Server.OpenAIHost` does not exist.

- [ ] **Step 2: Add OpenAI bind host config**

Modify `backend/pkg/config/config.go` so `Server` includes:

```go
type Server struct {
	APIPort         int           `mapstructure:"api_port"`
	AdminPort       int           `mapstructure:"admin_port"`
	OpenAIHost      string        `mapstructure:"openai_host"`
	OpenAIPort      int           `mapstructure:"openai_port"`
	WSPort          int           `mapstructure:"ws_port"`
	PprofPort       int           `mapstructure:"pprof_port"`
	ReadTimeout     time.Duration `mapstructure:"read_timeout"`
	WriteTimeout    time.Duration `mapstructure:"write_timeout"`
	ShutdownTimeout time.Duration `mapstructure:"shutdown_timeout"`
}
```

Modify `backend/configs/config.yaml` so the server block includes:

```yaml
server:
  api_port: 17180
  admin_port: 17188
  openai_host: ""
  openai_port: 17200
```

Modify `backend/cmd/openai/main.go` so the HTTP address is built with `net.JoinHostPort`:

```go
import (
	"fmt"
	"net"
	"net/http"
	"strconv"

	"github.com/kleinai/backend/internal/bootstrap"
	"github.com/kleinai/backend/internal/router"
	"github.com/kleinai/backend/pkg/logger"
)

// ...

srv := &http.Server{
	Addr:         openAIAddr(deps.Cfg.Server.OpenAIHost, deps.Cfg.Server.OpenAIPort),
	Handler:      r,
	ReadTimeout:  deps.Cfg.Server.ReadTimeout,
	WriteTimeout: 600,
}

// ...

func openAIAddr(host string, port int) string {
	return net.JoinHostPort(host, strconv.Itoa(port))
}
```

Run:

```bash
gofmt -w backend/cmd/openai/main.go backend/pkg/config/config.go backend/pkg/config/config_test.go
go -C "/Users/huangkunhuang/Public/程序工程目录/GO/gpt2api/backend" test ./pkg/config
```

Expected: test passes.

- [ ] **Step 3: Write supervisor config**

Create `deploy/all-in-one/supervisord.conf` with exactly this content:

```ini
[unix_http_server]
file=/tmp/supervisor.sock
chmod=0700
username=supervisor
password=supervisor

[supervisord]
nodaemon=true
user=root
logfile=/dev/null
logfile_maxbytes=0
pidfile=/tmp/supervisord.pid
childlogdir=/tmp

[supervisorctl]
serverurl=unix:///tmp/supervisor.sock
username=supervisor
password=supervisor

[rpcinterface:supervisor]
supervisor.rpcinterface_factory=supervisor.rpcinterface:make_main_rpcinterface

[program:api]
command=/app/api
directory=/app
autostart=true
autorestart=true
startsecs=5
stopsignal=TERM
stopwaitsecs=20
stdout_logfile=/dev/fd/1
stdout_logfile_maxbytes=0
stderr_logfile=/dev/fd/2
stderr_logfile_maxbytes=0
environment=KLEIN_SNOWFLAKE_NODE_ID="1",KLEIN_LOGGER_CONSOLE="true"

[program:admin]
command=/app/admin
directory=/app
autostart=true
autorestart=true
startsecs=5
stopsignal=TERM
stopwaitsecs=20
stdout_logfile=/dev/fd/1
stdout_logfile_maxbytes=0
stderr_logfile=/dev/fd/2
stderr_logfile_maxbytes=0
environment=KLEIN_SNOWFLAKE_NODE_ID="2",KLEIN_LOGGER_CONSOLE="true"

[program:openai]
command=/app/openai
directory=/app
autostart=true
autorestart=true
startsecs=5
stopsignal=TERM
stopwaitsecs=20
stdout_logfile=/dev/fd/1
stdout_logfile_maxbytes=0
stderr_logfile=/dev/fd/2
stderr_logfile_maxbytes=0
environment=KLEIN_SNOWFLAKE_NODE_ID="3",KLEIN_SERVER_OPENAI_HOST="127.0.0.1",KLEIN_SERVER_OPENAI_PORT="17201",KLEIN_LOGGER_CONSOLE="true"

[program:worker]
command=/app/worker
directory=/app
autostart=true
autorestart=true
startsecs=5
stopsignal=TERM
stopwaitsecs=20
stdout_logfile=/dev/fd/1
stdout_logfile_maxbytes=0
stderr_logfile=/dev/fd/2
stderr_logfile_maxbytes=0
environment=KLEIN_SNOWFLAKE_NODE_ID="4",KLEIN_LOGGER_CONSOLE="true"

[program:nginx]
command=/usr/sbin/nginx -g "daemon off;"
directory=/
autostart=true
autorestart=true
startsecs=1
stopsignal=QUIT
stopwaitsecs=10
stdout_logfile=/dev/fd/1
stdout_logfile_maxbytes=0
stderr_logfile=/dev/fd/2
stderr_logfile_maxbytes=0
```

- [ ] **Step 4: Validate supervisor config parses in Alpine**

Run:

```bash
docker run --rm \
  -v "/Users/huangkunhuang/Public/程序工程目录/GO/gpt2api/deploy/all-in-one/supervisord.conf:/etc/supervisord.conf:ro" \
  alpine:3.20 sh -lc 'apk add --no-cache supervisor >/dev/null && supervisord -n -c /etc/supervisord.conf -h >/dev/null'
```

Expected: command exits with code `0`. It must not print a parse error such as `Error: Format string` or `Error: File contains no section headers`.

- [ ] **Step 5: Commit supervisor and OpenAI bind config**

Run:

```bash
git add deploy/all-in-one/supervisord.conf backend/cmd/openai/main.go backend/pkg/config/config.go backend/pkg/config/config_test.go backend/configs/config.yaml
git commit -m "$(cat <<'EOF'
feat: supervise all-in-one app processes
EOF
)"
```

Expected: commit succeeds. If the user has not explicitly asked to commit during execution, skip this step and report it as intentionally skipped.

---

### Task 3: Add all-in-one Dockerfile

**Files:**
- Create: `Dockerfile.all-in-one`

- [ ] **Step 1: Write Dockerfile**

Create `Dockerfile.all-in-one` with exactly this content:

```dockerfile
# syntax=docker/dockerfile:1.7

FROM --platform=$BUILDPLATFORM golang:1.24-alpine AS backend-build

ARG TARGETOS
ARG TARGETARCH
ARG VERSION=dev
ARG BUILD_TIME=unknown

ENV CGO_ENABLED=0
WORKDIR /src

RUN apk add --no-cache ca-certificates git tzdata

COPY backend/go.mod backend/go.sum ./
RUN go mod download

COPY backend/ ./
RUN GOOS=$TARGETOS GOARCH=$TARGETARCH go build -trimpath -ldflags="-s -w -X github.com/kleinai/backend/pkg/version.Build=${VERSION} -X github.com/kleinai/backend/pkg/version.Time=${BUILD_TIME}" -o /out/api ./cmd/api && \
    GOOS=$TARGETOS GOARCH=$TARGETARCH go build -trimpath -ldflags="-s -w -X github.com/kleinai/backend/pkg/version.Build=${VERSION} -X github.com/kleinai/backend/pkg/version.Time=${BUILD_TIME}" -o /out/admin ./cmd/admin && \
    GOOS=$TARGETOS GOARCH=$TARGETARCH go build -trimpath -ldflags="-s -w -X github.com/kleinai/backend/pkg/version.Build=${VERSION} -X github.com/kleinai/backend/pkg/version.Time=${BUILD_TIME}" -o /out/openai ./cmd/openai && \
    GOOS=$TARGETOS GOARCH=$TARGETARCH go build -trimpath -ldflags="-s -w -X github.com/kleinai/backend/pkg/version.Build=${VERSION} -X github.com/kleinai/backend/pkg/version.Time=${BUILD_TIME}" -o /out/worker ./cmd/worker

FROM --platform=$BUILDPLATFORM node:20-alpine AS frontend-build

WORKDIR /repo
RUN corepack enable

COPY frontend/pnpm-workspace.yaml frontend/pnpm-lock.yaml frontend/package.json ./
COPY frontend/tsconfig.base.json ./
COPY frontend/packages packages
COPY frontend/apps/user apps/user
COPY frontend/apps/admin apps/admin

RUN pnpm install --frozen-lockfile
RUN pnpm --filter @kleinai/user build && \
    pnpm --filter @kleinai/admin build

FROM alpine:3.20 AS runtime

ENV TZ=Asia/Shanghai \
    KLEIN_LOG_DIR=/app/logs \
    KLEIN_STORAGE_ROOT=/app/storage/public \
    KLEIN_GROK_CF_STATE_PATH=/app/storage/grok_cf.json \
    KLEIN_SERVER_OPENAI_HOST=127.0.0.1 \
    KLEIN_SERVER_OPENAI_PORT=17201

WORKDIR /app

RUN apk add --no-cache ca-certificates nginx supervisor tzdata && \
    mkdir -p /app/logs /app/storage/public /run/nginx /usr/share/nginx/user /usr/share/nginx/admin && \
    rm -f /etc/nginx/http.d/default.conf /etc/nginx/conf.d/default.conf

COPY --from=backend-build /out/api /app/api
COPY --from=backend-build /out/admin /app/admin
COPY --from=backend-build /out/openai /app/openai
COPY --from=backend-build /out/worker /app/worker
COPY --from=backend-build /src/configs /app/configs
COPY --from=frontend-build /repo/apps/user/dist /usr/share/nginx/user
COPY --from=frontend-build /repo/apps/admin/dist /usr/share/nginx/admin
COPY deploy/all-in-one/nginx.conf /etc/nginx/http.d/default.conf
COPY deploy/all-in-one/supervisord.conf /etc/supervisord.conf

EXPOSE 17080 17088 17200

CMD ["/usr/bin/supervisord", "-c", "/etc/supervisord.conf"]
```

- [ ] **Step 2: Build the image for local amd64 validation**

Run:

```bash
docker buildx build \
  --platform linux/amd64 \
  -f Dockerfile.all-in-one \
  -t gpt2api:all-in-one \
  --load \
  "/Users/huangkunhuang/Public/程序工程目录/GO/gpt2api"
```

Expected: build succeeds and creates local image `gpt2api:all-in-one`.

- [ ] **Step 3: Validate nginx config inside the built image**

Run:

```bash
docker run --rm gpt2api:all-in-one nginx -t
```

Expected output contains:

```text
nginx: the configuration file /etc/nginx/nginx.conf syntax is ok
nginx: configuration file /etc/nginx/nginx.conf test is successful
```

- [ ] **Step 4: Validate frontend assets exist in the built image**

Run:

```bash
docker run --rm gpt2api:all-in-one sh -lc 'test -f /usr/share/nginx/user/index.html && test -f /usr/share/nginx/admin/index.html && test -x /app/api && test -x /app/admin && test -x /app/openai && test -x /app/worker'
```

Expected: command exits with code `0`.

- [ ] **Step 5: Validate OpenAI backend port override is present**

Run:

```bash
docker run --rm gpt2api:all-in-one sh -lc 'test "$KLEIN_SERVER_OPENAI_HOST" = "127.0.0.1" && test "$KLEIN_SERVER_OPENAI_PORT" = "17201"'
```

Expected: command exits with code `0`.

- [ ] **Step 6: Commit Dockerfile**

Run:

```bash
git add Dockerfile.all-in-one Dockerfile.all-in-one.dockerignore
git commit -m "$(cat <<'EOF'
feat: add all-in-one docker image
EOF
)"
```

Expected: commit succeeds. If the user has not explicitly asked to commit during execution, skip this step and report it as intentionally skipped.

---

### Task 4: Add GHCR GitHub Actions workflow

**Files:**
- Create: `.github/workflows/docker-image.yml`

- [ ] **Step 1: Create workflow directory**

Run:

```bash
ls -la "/Users/huangkunhuang/Public/程序工程目录/GO/gpt2api"
mkdir -p "/Users/huangkunhuang/Public/程序工程目录/GO/gpt2api/.github/workflows"
```

Expected: project root exists, and `.github/workflows` is created if missing.

- [ ] **Step 2: Write workflow file**

Create `.github/workflows/docker-image.yml` with exactly this content:

```yaml
name: Docker Image

on:
  push:
    branches:
      - main
    tags:
      - 'v*'
  pull_request:
    branches:
      - main
  workflow_dispatch:

permissions:
  contents: read
  packages: write

concurrency:
  group: docker-image-${{ github.ref }}
  cancel-in-progress: true

env:
  REGISTRY: ghcr.io
  IMAGE_NAME: ${{ github.repository }}

jobs:
  build:
    runs-on: ubuntu-latest

    steps:
      - name: Checkout
        uses: actions/checkout@v4

      - name: Set up QEMU
        uses: docker/setup-qemu-action@v3

      - name: Set up Docker Buildx
        uses: docker/setup-buildx-action@v3

      - name: Log in to GHCR
        if: github.event_name != 'pull_request'
        uses: docker/login-action@v3
        with:
          registry: ${{ env.REGISTRY }}
          username: ${{ github.actor }}
          password: ${{ secrets.GITHUB_TOKEN }}

      - name: Docker metadata
        id: meta
        uses: docker/metadata-action@v5
        with:
          images: ${{ env.REGISTRY }}/${{ env.IMAGE_NAME }}
          tags: |
            type=raw,value=latest,enable={{is_default_branch}}
            type=ref,event=tag
            type=sha,prefix=sha-,format=short

      - name: Build and push
        uses: docker/build-push-action@v6
        with:
          context: .
          file: ./Dockerfile.all-in-one
          platforms: linux/amd64,linux/arm64
          push: ${{ github.event_name != 'pull_request' }}
          tags: ${{ steps.meta.outputs.tags }}
          labels: ${{ steps.meta.outputs.labels }}
          build-args: |
            VERSION=${{ github.ref_name }}
            BUILD_TIME=${{ github.event.head_commit.timestamp || github.run_id }}
          cache-from: type=gha
          cache-to: type=gha,mode=max
```

- [ ] **Step 3: Validate workflow YAML parses**

Run:

```bash
python3 - <<'PY'
from pathlib import Path
import yaml
path = Path('/Users/huangkunhuang/Public/程序工程目录/GO/gpt2api/.github/workflows/docker-image.yml')
with path.open('r', encoding='utf-8') as f:
    data = yaml.safe_load(f)
assert data['name'] == 'Docker Image'
assert 'build' in data['jobs']
assert data['permissions']['packages'] == 'write'
print('workflow yaml ok')
PY
```

Expected output:

```text
workflow yaml ok
```

If Python cannot import `yaml`, run this fallback instead:

```bash
docker run --rm \
  -v "/Users/huangkunhuang/Public/程序工程目录/GO/gpt2api/.github/workflows/docker-image.yml:/workflow.yml:ro" \
  mikefarah/yq:4 e '.name == "Docker Image" and .jobs.build.runs-on == "ubuntu-latest"' /workflow.yml
```

Expected output:

```text
true
```

- [ ] **Step 4: Validate GitHub Actions expressions with actionlint if available**

Run:

```bash
if command -v actionlint >/dev/null 2>&1; then actionlint "/Users/huangkunhuang/Public/程序工程目录/GO/gpt2api/.github/workflows/docker-image.yml"; else printf 'actionlint not installed; skipped\n'; fi
```

Expected: either no output from `actionlint`, or exactly:

```text
actionlint not installed; skipped
```

- [ ] **Step 5: Commit workflow**

Run:

```bash
git add .github/workflows/docker-image.yml
git commit -m "$(cat <<'EOF'
ci: publish all-in-one docker image
EOF
)"
```

Expected: commit succeeds. If the user has not explicitly asked to commit during execution, skip this step and report it as intentionally skipped.

---

### Task 5: End-to-end packaging verification

**Files:**
- No new files.
- Verify: `Dockerfile.all-in-one`, `deploy/all-in-one/nginx.conf`, `deploy/all-in-one/supervisord.conf`, `.github/workflows/docker-image.yml`

- [ ] **Step 1: Run repository status check**

Run:

```bash
git status --short
```

Expected: only intended files are modified/untracked:

```text
?? .github/
?? Dockerfile.all-in-one
?? deploy/all-in-one/
?? docs/superpowers/
```

If commits were created during execution, `git status --short` may be empty or only show intentionally uncommitted docs.

- [ ] **Step 2: Rebuild the image cleanly for amd64**

Run:

```bash
docker buildx build \
  --platform linux/amd64 \
  -f Dockerfile.all-in-one \
  -t gpt2api:all-in-one \
  --load \
  "/Users/huangkunhuang/Public/程序工程目录/GO/gpt2api"
```

Expected: build succeeds.

- [ ] **Step 3: Start Redis and the image in dev mode without external MySQL**

Run:

```bash
docker rm -f gpt2api-all-in-one-test gpt2api-all-in-one-redis >/dev/null 2>&1 || true
docker network rm gpt2api-all-in-one-net >/dev/null 2>&1 || true
docker network create gpt2api-all-in-one-net
docker run -d \
  --name gpt2api-all-in-one-redis \
  --network gpt2api-all-in-one-net \
  redis:7-alpine
docker run -d \
  --name gpt2api-all-in-one-test \
  --network gpt2api-all-in-one-net \
  -p 17080:17080 \
  -p 17088:17088 \
  -p 17200:17200 \
  -e KLEIN_ENV=dev \
  -e KLEIN_REDIS_ADDR='gpt2api-all-in-one-redis:6379' \
  -e KLEIN_JWT_SECRET='dev-secret-dev-secret-dev-secret-32' \
  -e KLEIN_JWT_REFRESH_SECRET='dev-refresh-dev-refresh-secret-32' \
  -e KLEIN_AES_KEY='12345678901234567890123456789012' \
  gpt2api:all-in-one
```

Expected: the network name, Redis container ID, and application container ID are printed. MySQL remains absent in this smoke test; the backend runs in dev degraded mode for health endpoints.

- [ ] **Step 4: Wait for supervisor-managed processes to start**

Run:

```bash
for i in $(seq 1 30); do
  if docker exec gpt2api-all-in-one-test supervisorctl status | grep -q 'RUNNING'; then
    docker exec gpt2api-all-in-one-test supervisorctl status
    exit 0
  fi
  sleep 1
done
docker logs gpt2api-all-in-one-test
exit 1
```

Expected output includes five `RUNNING` entries:

```text
admin                            RUNNING
api                              RUNNING
nginx                            RUNNING
openai                           RUNNING
worker                           RUNNING
```

- [ ] **Step 5: Verify public health endpoints through nginx**

Run:

```bash
curl -fsS http://localhost:17080/healthz
curl -fsS http://localhost:17088/healthz
curl -fsS http://localhost:17200/healthz
curl -fsS http://localhost:17200/v1/health
```

Expected:

- first response contains `"service":"api"`
- second response contains `"service":"admin"`
- third response contains `"service":"openai"`
- fourth response contains `"ok":true`

- [ ] **Step 6: Verify frontend roots are served**

Run:

```bash
curl -fsS http://localhost:17080/ | grep -qi '<html'
curl -fsS http://localhost:17088/ | grep -qi '<html'
```

Expected: both commands exit with code `0`.

- [ ] **Step 7: Verify internal OpenAI backend port avoids conflict**

Run:

```bash
docker exec gpt2api-all-in-one-test sh -lc 'wget -qO- http://127.0.0.1:17201/v1/health && netstat -ltn | grep "127.0.0.1:17201"'
```

Expected output contains:

```json
{"ok":true}
```

and `netstat` shows a `127.0.0.1:17201` listener, not `0.0.0.0:17201`.

- [ ] **Step 8: Clean up test container**

Run:

```bash
docker rm -f gpt2api-all-in-one-test gpt2api-all-in-one-redis
docker network rm gpt2api-all-in-one-net
```

Expected: output includes:

```text
gpt2api-all-in-one-test
gpt2api-all-in-one-redis
gpt2api-all-in-one-net
```

- [ ] **Step 9: Report validation results**

Report these exact items:

| Check | Result |
|---|---|
| nginx standalone syntax | pass/fail/skipped |
| supervisor config parse | pass/fail/skipped |
| Docker build linux/amd64 | pass/fail/skipped |
| built image nginx syntax | pass/fail/skipped |
| image artifact presence | pass/fail/skipped |
| workflow YAML parse | pass/fail/skipped |
| actionlint | pass/fail/skipped |
| runtime health checks | pass/fail/skipped |

- [ ] **Step 10: Final commit if requested**

If the user explicitly requested commits, run:

```bash
git status --short
git add Dockerfile.all-in-one Dockerfile.all-in-one.dockerignore deploy/all-in-one/nginx.conf deploy/all-in-one/supervisord.conf backend/cmd/openai/main.go backend/pkg/config/config.go backend/pkg/config/config_test.go backend/configs/config.yaml .github/workflows/docker-image.yml docs/superpowers/specs/2026-05-06-docker-github-workflow-design.md docs/superpowers/plans/2026-05-06-docker-github-workflow.md
git commit -m "$(cat <<'EOF'
feat: add GHCR all-in-one docker packaging
EOF
)"
```

Expected: commit succeeds. If earlier task commits already committed all implementation files, do not create an empty commit.

---

## Self-review

| Spec requirement | Covered by |
|---|---|
| Single image with backend, both frontends, and nginx | Task 3 |
| MySQL and Redis external | Implementation notes, Task 5 Redis-backed smoke test |
| GHCR workflow | Task 4 |
| Public ports `17080`, `17088`, `17200` | Task 1, Task 3, Task 5 |
| Multi-arch `linux/amd64`, `linux/arm64` | Task 4 |
| Tags `latest`, `v*`, `sha-xxxxxxx` | Task 4 |
| nginx routing | Task 1 |
| supervisor process management | Task 2 |
| OpenAI port conflict avoidance | Task 1, Task 2, Task 3, Task 5 |
| Verification | Task 1 through Task 5 |

Placeholder scan completed: no `TBD`, `TODO`, `implement later`, or unspecified test steps are intentionally left in this plan.
