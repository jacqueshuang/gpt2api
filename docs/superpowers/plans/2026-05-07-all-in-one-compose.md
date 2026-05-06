# All-in-One Compose Deployment Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a docker-compose deployment file and safe env template for running the GHCR all-in-one image with external MySQL and Redis.

**Architecture:** The compose file defines one `gpt2api` service using `${GPT2API_IMAGE:-ghcr.io/432539/gpt2api:latest}`. It maps public ports, mounts named volumes for `/app/storage` and `/app/logs`, and passes required runtime configuration through an ignored env file copied from a checked-in example.

**Tech Stack:** Docker Compose v2, GHCR Docker image, nginx/supervisord all-in-one runtime, MySQL and Redis external services.

---

## File Map

| File | Action | Responsibility |
|---|---|---|
| `deploy/docker-compose.all-in-one.yml` | Create | Single-service compose deployment for the all-in-one image. |
| `deploy/env/.env.all-in-one.example` | Create | Safe placeholder env template for required runtime variables. |
| `README.md` | Modify | Replace raw `docker run` as the primary GHCR deployment path with compose commands. |
| `docs/06-部署与运维规范.md` | Modify | Document operations usage of the all-in-one compose file. |
| `docs/superpowers/specs/2026-05-07-all-in-one-compose-design.md` | Already created | Design source of truth for this change. |

## Implementation Notes

- Do not add MySQL or Redis services to `deploy/docker-compose.all-in-one.yml`; the user selected external dependencies.
- Do not store real secrets in `deploy/env/.env.all-in-one.example`.
- Keep `env/.env.all-in-one` ignored by the existing `.gitignore` rule `.env.*`.
- Keep the all-in-one OpenAI backend internal with `KLEIN_SERVER_OPENAI_HOST=127.0.0.1` and `KLEIN_SERVER_OPENAI_PORT=17201`.
- Use `unless-stopped` so manual stops survive daemon restarts.
- Use a compose healthcheck that validates the public user API health route through nginx.

---

### Task 1: Add all-in-one compose deployment file

**Files:**
- Create: `deploy/docker-compose.all-in-one.yml`

- [ ] **Step 1: Create compose file**

Create `/Users/huangkunhuang/Public/程序工程目录/GO/gpt2api/deploy/docker-compose.all-in-one.yml` with exactly this content:

```yaml
# =====================================================
# KleinAI · GHCR all-in-one 单镜像部署
# 启动：docker compose --env-file ./env/.env.all-in-one -f docker-compose.all-in-one.yml up -d
# 依赖：外部 MySQL / Redis
# =====================================================

services:
  gpt2api:
    image: ${GPT2API_IMAGE:-ghcr.io/432539/gpt2api:latest}
    container_name: ${GPT2API_CONTAINER_NAME:-gpt2api}
    restart: unless-stopped
    environment:
      KLEIN_ENV: ${KLEIN_ENV:-prod}
      KLEIN_DB_DSN: ${KLEIN_DB_DSN:?KLEIN_DB_DSN is required}
      KLEIN_REDIS_ADDR: ${KLEIN_REDIS_ADDR:?KLEIN_REDIS_ADDR is required}
      KLEIN_REDIS_PASSWORD: ${KLEIN_REDIS_PASSWORD:-}
      KLEIN_JWT_SECRET: ${KLEIN_JWT_SECRET:?KLEIN_JWT_SECRET is required}
      KLEIN_JWT_REFRESH_SECRET: ${KLEIN_JWT_REFRESH_SECRET:?KLEIN_JWT_REFRESH_SECRET is required}
      KLEIN_AES_KEY: ${KLEIN_AES_KEY:?KLEIN_AES_KEY is required}
      KLEIN_CORS_ORIGINS: ${KLEIN_CORS_ORIGINS:?KLEIN_CORS_ORIGINS is required}
      KLEIN_OPENAI_BASE: ${KLEIN_OPENAI_BASE:-https://api.openai.com}
      KLEIN_GROK_BASE: ${KLEIN_GROK_BASE:-https://api.x.ai}
      KLEIN_SERVER_OPENAI_HOST: 127.0.0.1
      KLEIN_SERVER_OPENAI_PORT: 17201
      KLEIN_LOG_DIR: /app/logs
      TZ: ${TZ:-Asia/Shanghai}
    ports:
      - "${KLEIN_USER_WEB_PORT:-17080}:17080"
      - "${KLEIN_ADMIN_WEB_PORT:-17088}:17088"
      - "${KLEIN_OPENAI_PORT:-17200}:17200"
    volumes:
      - gpt2api-storage:/app/storage
      - gpt2api-logs:/app/logs
    healthcheck:
      test: ["CMD-SHELL", "wget -qO- http://127.0.0.1:17080/healthz | grep -q '\"ok\":true' || exit 1"]
      interval: 15s
      timeout: 5s
      retries: 5
      start_period: 30s

volumes:
  gpt2api-storage:
  gpt2api-logs:
```

- [ ] **Step 2: Validate compose syntax fails without required env**

Run:

```bash
docker compose -f "/Users/huangkunhuang/Public/程序工程目录/GO/gpt2api/deploy/docker-compose.all-in-one.yml" config
```

Expected: command fails and mentions `KLEIN_DB_DSN is required`.

---

### Task 2: Add all-in-one env example

**Files:**
- Create: `deploy/env/.env.all-in-one.example`

- [ ] **Step 1: Create env example**

Create `/Users/huangkunhuang/Public/程序工程目录/GO/gpt2api/deploy/env/.env.all-in-one.example` with exactly this content:

```dotenv
# =====================================================
# KleinAI · GHCR all-in-one 单镜像部署环境变量样例
# 用法：cp env/.env.all-in-one.example env/.env.all-in-one
# 生产部署必须替换所有 replace-* 值；env/.env.all-in-one 不入仓。
# =====================================================

# 镜像
GPT2API_IMAGE=ghcr.io/432539/gpt2api:latest
GPT2API_CONTAINER_NAME=gpt2api

# 运行环境
KLEIN_ENV=prod
TZ=Asia/Shanghai

# 对外端口
KLEIN_USER_WEB_PORT=17080
KLEIN_ADMIN_WEB_PORT=17088
KLEIN_OPENAI_PORT=17200

# 外部 MySQL / Redis
KLEIN_DB_DSN=klein:replace-mysql-password@tcp(mysql-host:3306)/klein_ai?charset=utf8mb4&parseTime=True&loc=Local
KLEIN_REDIS_ADDR=redis-host:6379
KLEIN_REDIS_PASSWORD=

# 安全密钥
# KLEIN_JWT_SECRET / KLEIN_JWT_REFRESH_SECRET 至少 32 字节。
# KLEIN_AES_KEY 必须是 32 字节原始字符串，或 64 位 hex 字符串。
KLEIN_JWT_SECRET=replace-with-at-least-32-bytes-jwt-secret
KLEIN_JWT_REFRESH_SECRET=replace-with-at-least-32-bytes-refresh-secret
KLEIN_AES_KEY=0123456789abcdef0123456789abcdef

# Provider 基础地址
KLEIN_OPENAI_BASE=https://api.openai.com
KLEIN_GROK_BASE=https://api.x.ai

# CORS 来源
KLEIN_CORS_ORIGINS=http://localhost:17080,http://localhost:17088
```

- [ ] **Step 2: Validate compose renders with env example**

Run:

```bash
docker compose --env-file "/Users/huangkunhuang/Public/程序工程目录/GO/gpt2api/deploy/env/.env.all-in-one.example" -f "/Users/huangkunhuang/Public/程序工程目录/GO/gpt2api/deploy/docker-compose.all-in-one.yml" config >/tmp/gpt2api-all-in-one-compose.yml
```

Expected: command exits with code `0` and writes rendered config to `/tmp/gpt2api-all-in-one-compose.yml`.

- [ ] **Step 3: Verify rendered config includes required routing/runtime values**

Run:

```bash
python3 - <<'PY'
from pathlib import Path
text = Path('/tmp/gpt2api-all-in-one-compose.yml').read_text()
required = [
    'ghcr.io/432539/gpt2api:latest',
    '17080:17080',
    '17088:17088',
    '17200:17200',
    'KLEIN_SERVER_OPENAI_HOST: 127.0.0.1',
    'KLEIN_SERVER_OPENAI_PORT: "17201"',
    'gpt2api-storage:/app/storage',
    'gpt2api-logs:/app/logs',
]
missing = [item for item in required if item not in text]
if missing:
    raise SystemExit('missing from rendered compose: ' + ', '.join(missing))
print('all-in-one compose config ok')
PY
```

Expected output:

```text
all-in-one compose config ok
```

---

### Task 3: Document compose deployment path

**Files:**
- Modify: `README.md`
- Modify: `docs/06-部署与运维规范.md`

- [ ] **Step 1: Update README GHCR section**

In `/Users/huangkunhuang/Public/程序工程目录/GO/gpt2api/README.md`, replace the command block in section `### 3A. 可选：GHCR 单镜像部署` with this compose-first block:

```markdown
```bash
cd deploy
cp env/.env.all-in-one.example env/.env.all-in-one
# 编辑 env/.env.all-in-one，填入外部 MySQL / Redis 与密钥
docker compose --env-file env/.env.all-in-one -f docker-compose.all-in-one.yml up -d
```

默认镜像为 `ghcr.io/432539/gpt2api:latest`，可在 `env/.env.all-in-one` 中通过 `GPT2API_IMAGE` 覆盖。
```

Keep the paragraph after the old command block that explains ports and internal OpenAI routing.

- [ ] **Step 2: Update ops docs all-in-one section**

In `/Users/huangkunhuang/Public/程序工程目录/GO/gpt2api/docs/06-部署与运维规范.md`, replace the raw `docker run` block in `## 3. All-in-One GHCR 单镜像部署` with this compose block:

```markdown
```bash
cd deploy
cp env/.env.all-in-one.example env/.env.all-in-one
# 编辑 env/.env.all-in-one，填入外部 MySQL / Redis 与密钥
docker compose --env-file env/.env.all-in-one -f docker-compose.all-in-one.yml up -d
```

默认镜像为 `ghcr.io/432539/gpt2api:latest`，可在 `env/.env.all-in-one` 中通过 `GPT2API_IMAGE` 覆盖。
```

Keep the sentence that documents public ports and internal `127.0.0.1:17201` routing.

- [ ] **Step 3: Verify docs mention the new files**

Run:

```bash
python3 - <<'PY'
from pathlib import Path
checks = {
    'README.md': [
        'docker-compose.all-in-one.yml',
        'env/.env.all-in-one.example',
        'GPT2API_IMAGE',
    ],
    'docs/06-部署与运维规范.md': [
        'docker-compose.all-in-one.yml',
        'env/.env.all-in-one.example',
        'GPT2API_IMAGE',
    ],
}
root = Path('/Users/huangkunhuang/Public/程序工程目录/GO/gpt2api')
for rel, needles in checks.items():
    text = (root / rel).read_text(encoding='utf-8')
    missing = [needle for needle in needles if needle not in text]
    if missing:
        raise SystemExit(f'{rel} missing: {missing}')
print('all-in-one compose docs ok')
PY
```

Expected output:

```text
all-in-one compose docs ok
```

---

### Task 4: Final validation

**Files:**
- Test: `deploy/docker-compose.all-in-one.yml`
- Test: `deploy/env/.env.all-in-one.example`
- Test: `README.md`
- Test: `docs/06-部署与运维规范.md`

- [ ] **Step 1: Run compose config validation**

Run:

```bash
docker compose --env-file "/Users/huangkunhuang/Public/程序工程目录/GO/gpt2api/deploy/env/.env.all-in-one.example" -f "/Users/huangkunhuang/Public/程序工程目录/GO/gpt2api/deploy/docker-compose.all-in-one.yml" config
```

Expected: rendered compose YAML is printed and command exits with code `0`.

- [ ] **Step 2: Verify ignored local env path**

Run:

```bash
git -C "/Users/huangkunhuang/Public/程序工程目录/GO/gpt2api" check-ignore -q deploy/env/.env.all-in-one && echo ignored
```

Expected output:

```text
ignored
```

- [ ] **Step 3: Check git status**

Run:

```bash
git -C "/Users/huangkunhuang/Public/程序工程目录/GO/gpt2api" status --short
```

Expected: output includes only intended files:

```text
 M README.md
 M docs/06-部署与运维规范.md
?? deploy/docker-compose.all-in-one.yml
?? deploy/env/.env.all-in-one.example
?? docs/superpowers/specs/2026-05-07-all-in-one-compose-design.md
?? docs/superpowers/plans/2026-05-07-all-in-one-compose.md
```

- [ ] **Step 4: Commit if the user explicitly requests it**

If the user explicitly asks for a commit, run:

```bash
git -C "/Users/huangkunhuang/Public/程序工程目录/GO/gpt2api" add README.md docs/06-部署与运维规范.md deploy/docker-compose.all-in-one.yml deploy/env/.env.all-in-one.example docs/superpowers/specs/2026-05-07-all-in-one-compose-design.md docs/superpowers/plans/2026-05-07-all-in-one-compose.md
git -C "/Users/huangkunhuang/Public/程序工程目录/GO/gpt2api" commit -m "$(cat <<'EOF'
feat: add all-in-one compose deployment
EOF
)"
```

Expected: commit succeeds. If the user did not explicitly ask for a commit, do not commit.

---

## Self-Review

| Spec requirement | Covered by |
|---|---|
| One compose service using all-in-one image | Task 1 |
| MySQL and Redis remain external | Task 1, Task 2 |
| Safe env template | Task 2 |
| Public ports configurable with existing defaults | Task 1, Task 2 |
| Persistent storage and logs | Task 1 |
| OpenAI backend remains localhost-only behind nginx | Task 1 |
| README and ops docs updated | Task 3 |
| Compose config validation | Task 2, Task 4 |

Placeholder scan: no `TBD`, `TODO`, `implement later`, or vague validation steps remain.

Type/name consistency: file names and environment variable names match the approved design and existing all-in-one runtime.
