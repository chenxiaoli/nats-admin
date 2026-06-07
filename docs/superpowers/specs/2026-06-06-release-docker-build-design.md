# GitHub CI/CD: Release Build

## Goal

When a GitHub Release is published:
1. Build frontend, embed into Go binary via `embed.FS` → single executable
2. Upload executable as Release asset
3. Build Docker image (same embedded binary) and push to ghcr.io

## Trigger

- Event: `release` with type `published`
- Tag format: `vX.Y.Z`

## Architecture: Single Binary

Frontend is compiled into the Go binary at build time:

```
frontend/  →  pnpm build  →  frontend/dist/
                                    ↓ copy
backend/cmd/server/frontend_dist/  →  //go:embed  →  single binary
                                    ↓
                              serves /api/v1/*  (API)
                              serves /*          (SPA static files)
```

## Workflow: `.github/workflows/release.yml`

Three jobs:

### Job 1: `build-frontend`

1. Checkout repo
2. Setup pnpm + Node 22
3. `cd frontend && pnpm install --frozen-lockfile && pnpm build`
4. Upload `frontend/dist/` as artifact

### Job 2: `build-binary` (depends on `build-frontend`)

1. Checkout repo
2. Download frontend dist artifact → copy to `backend/cmd/server/frontend_dist/`
3. Setup Go 1.25
4. `CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o nats-admin ./cmd/server`
5. Upload `nats-admin` to GitHub Release assets

### Job 3: `build-docker` (depends on `build-frontend`)

1. Checkout repo
2. Download frontend dist artifact → copy to `frontend/dist/`
3. Docker login to ghcr.io using `GITHUB_TOKEN`
4. Build `backend/Dockerfile` (multi-stage: node build + go build)
5. Push with tags: `ghcr.io/<owner>/nats-admin:vX.Y.Z`, `ghcr.io/<owner>/nats-admin:latest`

## Code Changes

### New file: `backend/cmd/server/frontend.go`

```go
package main

import "embed"

//go:embed frontend_dist/*
var frontendFS embed.FS
```

### Modified: `backend/internal/api/router.go`

After all `/api/v1` routes, add a catch-all that:
- Serves static files from `embed.FS` at `/` (non-API paths)
- For unknown paths, returns `index.html` (SPA client-side routing)
- Skips paths starting with `/api/` and `/ws/`

New field on `Deps`: `FrontendFS embed.FS`

### Modified: `backend/Dockerfile`

Updated to multi-stage build that includes frontend:

```
Stage 1: node:22-alpine  →  pnpm build frontend  →  dist/
Stage 2: golang:1.25-alpine  →  copy dist/ into source tree  →  go build
Stage 3: alpine:3.20  →  copy binary + migrations
```

### New file: `frontend/.dockerignore`

Exclude `node_modules/`, `dist/`, `.vite/`

## Image Naming

Single image (no separate frontend image):

```
ghcr.io/<owner>/nats-admin:vX.Y.Z
ghcr.io/<owner>/nats-admin:latest
```

## Release Assets

```
nats-admin-linux-amd64    (standalone binary, ~20-30MB)
```

## Secrets

Uses automatically-provided `GITHUB_TOKEN` with `packages: write` and `contents: write` permissions.

## Files Changed

| File | Action |
|------|--------|
| `.github/workflows/release.yml` | Create |
| `backend/cmd/server/frontend.go` | Create (embed directive) |
| `backend/internal/api/router.go` | Modify (add SPA static serving) |
| `backend/cmd/server/main.go` | Modify (pass embed.FS to router) |
| `backend/Dockerfile` | Modify (add node build stage) |
| `frontend/.dockerignore` | Create |

## Out of Scope

- PR validation builds
- Push-to-main builds
- Automated deployment
- Multi-architecture builds (only linux/amd64)
- macOS/Windows binaries
