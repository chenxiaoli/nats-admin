# Release Build: Single Binary with Embedded Frontend

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** When a GitHub Release is published, build a single Linux/amd64 binary that embeds the React frontend and also build a Docker image, both pushed with version tags.

**Architecture:** Frontend is compiled with `pnpm build` and the resulting `dist/` is embedded into the Go binary via `//go:embed`. The Go server serves the embedded assets at `/` (with SPA fallback to `index.html`) alongside the existing `/api/v1/*` API routes. A single GitHub Actions workflow runs on `release: published`, builds the frontend, then builds the binary and the Docker image in parallel.

**Tech Stack:** Go 1.25 `embed.FS`, chi v5, pnpm, GitHub Actions, Docker buildx, ghcr.io.

---

## File Structure

| File | Responsibility |
|------|----------------|
| `backend/cmd/server/frontend.go` | `//go:embed` directive exposing `frontend_dist/` as `embed.FS` |
| `backend/internal/api/static.go` | New file: helper that mounts an `embed.FS` onto a chi router with SPA fallback |
| `backend/internal/api/router.go` | Add `FrontendFS embed.FS` field to `Deps`; mount static handler |
| `backend/cmd/server/main.go` | Pass `frontendFS` to `api.Deps` |
| `backend/cmd/server/frontend_dist/.gitkeep` | Placeholder so `//go:embed` compiles when dist is empty |
| `backend/internal/api/router_test.go` | New file: tests for static + SPA fallback behavior |
| `backend/Dockerfile` | Add a Node build stage that produces `frontend/dist/` before `go build` |
| `frontend/.dockerignore` | Exclude `node_modules/`, `dist/`, `.vite/` |
| `.github/workflows/release.yml` | Release workflow: build-frontend → build-binary + build-docker |
| `.gitignore` | Allow `frontend_dist/*` except `.gitkeep` (no change needed; we only need `.gitkeep`) |

---

## Task 1: Add embed directive and placeholder

**Files:**
- Create: `backend/cmd/server/frontend.go`
- Create: `backend/cmd/server/frontend_dist/.gitkeep`

- [ ] **Step 1: Create `backend/cmd/server/frontend_dist/.gitkeep`**

Empty file. Ensures the `frontend_dist` directory exists in source control so `//go:embed frontend_dist/*` does not fail when no frontend build is present.

```bash
mkdir -p backend/cmd/server/frontend_dist
touch backend/cmd/server/frontend_dist/.gitkeep
```

- [ ] **Step 2: Create `backend/cmd/server/frontend.go`**

```go
package main

import "embed"

//go:embed frontend_dist/*
var frontendFS embed.FS
```

- [ ] **Step 3: Verify it compiles**

Run: `cd backend && go build ./cmd/server`
Expected: builds successfully (empty embed is allowed).

- [ ] **Step 4: Commit**

```bash
git add backend/cmd/server/frontend.go backend/cmd/server/frontend_dist/.gitkeep
git commit -m "feat(embed): embed frontend dist via go:embed"
```

---

## Task 2: Add static file serving helper (TDD)

**Files:**
- Create: `backend/internal/api/static.go`
- Create: `backend/internal/api/static_test.go`

- [ ] **Step 1: Write the failing test**

Create `backend/internal/api/static_test.go`:

```go
package api

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"
)

func TestStaticHandler_ServesEmbeddedFile(t *testing.T) {
	fsys := fstest.MapFS{
		"index.html":     &fstest.MapFile{Data: []byte("<html>root</html>")},
		"assets/app.js":  &fstest.MapFile{Data: []byte("console.log(1)")},
		"assets/app.css": &fstest.MapFile{Data: []byte("body{}")},
	}

	handler := staticHandler(fsys)
	srv := httptest.NewServer(handler)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/index.html")
	if err != nil {
		t.Fatalf("get index: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if string(body) != "<html>root</html>" {
		t.Fatalf("unexpected body: %q", body)
	}
	if resp.StatusCode != 200 {
		t.Fatalf("status: %d", resp.StatusCode)
	}
}

func TestStaticHandler_SPAFallback(t *testing.T) {
	fsys := fstest.MapFS{
		"index.html": &fstest.MapFile{Data: []byte("<html>spa</html>")},
	}

	handler := staticHandler(fsys)
	srv := httptest.NewServer(handler)
	defer srv.Close()

	for _, path := range []string{"/", "/tenants", "/tenants/123?tab=creds"} {
		resp, err := http.Get(srv.URL + path)
		if err != nil {
			t.Fatalf("get %s: %v", path, err)
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if string(body) != "<html>spa</html>" {
			t.Fatalf("path %s: expected spa fallback, got %q", path, body)
		}
	}
}

func TestStaticHandler_SkipsAPIPaths(t *testing.T) {
	fsys := fstest.MapFS{
		"index.html": &fstest.MapFile{Data: []byte("spa")},
	}
	handler := staticHandler(fsys)
	srv := httptest.NewServer(handler)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/v1/tenants")
	if err != nil {
		t.Fatalf("get api: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 404 {
		t.Fatalf("api path should pass through (404 from no route), got %d", resp.StatusCode)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd backend && go test ./internal/api/... -run TestStaticHandler -v`
Expected: FAIL — `staticHandler` undefined.

- [ ] **Step 3: Implement `staticHandler`**

Create `backend/internal/api/static.go`:

```go
package api

import (
	"io/fs"
	"net/http"
	"path"
	"strings"
)

// staticHandler serves embedded frontend assets with SPA fallback.
// Requests for paths that don't match an embedded file fall back to
// index.html so client-side routing works. Paths under /api/ and /ws/
// are left to the surrounding router (they will not reach this handler
// when mounted on a sub-router).
func staticHandler(fsys fs.FS) http.Handler {
	fileServer := http.FileServer(http.FS(fsys))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Strip leading slash; embed.FS uses "index.html" not "/index.html".
		clean := strings.TrimPrefix(r.URL.Path, "/")
		if clean == "" {
			clean = "index.html"
		}
		if _, err := fs.Stat(fsys, clean); err != nil {
			// SPA fallback: serve index.html for unknown paths.
			clean = "index.html"
		}
		// Reconstruct a request with the resolved path.
		r2 := r.Clone(r.Context())
		r2.URL.Path = "/" + clean
		http.StripPrefix("", fileServer).ServeHTTP(w, r2)
	})
}

// path.Join kept to avoid unused import on edits above.
var _ = path.Join
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd backend && go test ./internal/api/... -run TestStaticHandler -v`
Expected: PASS for all three tests.

Note: The `TestStaticHandler_SkipsAPIPaths` test passes through `/api/v1/tenants` to the static handler (because the test mounts it directly). When mounted on the real router, `/api/v1/*` is handled by the API group and never reaches this handler. The test verifies the static handler does not 200 on missing files (it falls back to index.html, but since the test passes a 404 from the surrounding httptest server — wait, we need to think about this).

Reconsidering: `http.FileServer` returns 404 for files that don't exist. Our fallback kicks in BEFORE the file server, so it serves index.html. The test's "404" expectation is wrong. Fix: the test should verify the static handler does NOT serve index.html for `/api/...` paths. Better: leave that check to the integration test in Task 3, and remove `TestStaticHandler_SkipsAPIPaths`.

Update the test file to remove `TestStaticHandler_SkipsAPIPaths`. The remaining two tests are sufficient.

- [ ] **Step 5: Re-run tests**

Run: `cd backend && go test ./internal/api/... -run TestStaticHandler -v`
Expected: PASS for the two remaining tests.

- [ ] **Step 6: Commit**

```bash
git add backend/internal/api/static.go backend/internal/api/static_test.go
git commit -m "feat(api): static handler for embedded frontend with SPA fallback"
```

---

## Task 3: Wire embed.FS into router and main

**Files:**
- Modify: `backend/internal/api/router.go`
- Modify: `backend/cmd/server/main.go`

- [ ] **Step 1: Add integration test for router**

Create `backend/internal/api/router_test.go`:

```go
package api

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"
)

func TestRouter_ServesStaticAndAPIPaths(t *testing.T) {
	fsys := fstest.MapFS{
		"index.html": &fstest.MapFile{Data: []byte("<html>app</html>")},
		"assets/x.js": &fstest.MapFile{Data: []byte("x")},
	}

	r := NewRouter(Deps{
		JWTSecret:  []byte("test-secret-test-secret-test-secret-1234"),
		FrontendFS: fsys,
	})
	srv := httptest.NewServer(r)
	defer srv.Close()

	t.Run("static root", func(t *testing.T) {
		resp, err := http.Get(srv.URL + "/")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		if string(body) != "<html>app</html>" {
			t.Fatalf("got %q", body)
		}
	})

	t.Run("spa route falls back to index", func(t *testing.T) {
		resp, err := http.Get(srv.URL + "/tenants/abc")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		if string(body) != "<html>app</html>" {
			t.Fatalf("expected spa fallback, got %q", body)
		}
	})

	t.Run("api path not intercepted by static", func(t *testing.T) {
		resp, err := http.Get(srv.URL + "/api/v1/tenants")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		// Without auth, middleware should 401 — proving the static handler
		// did not intercept the api path.
		if resp.StatusCode != http.StatusUnauthorized {
			t.Fatalf("expected 401 from auth middleware, got %d", resp.StatusCode)
		}
	})
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd backend && go test ./internal/api/... -run TestRouter_ServesStaticAndAPIPaths -v`
Expected: FAIL — `FrontendFS` field does not exist on `Deps`.

- [ ] **Step 3: Modify `backend/internal/api/router.go`**

Add `"embed"` and `"io/fs"` to imports. Add `FrontendFS fs.FS` to `Deps`. Mount the static handler after the `/api/v1` route group:

```go
package api

import (
	"embed"
	"io/fs"

	"github.com/chenxiaoli/nats-admin/internal/api/handler"
	"github.com/chenxiaoli/nats-admin/internal/api/middleware"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Deps struct {
	Pool       *pgxpool.Pool
	JWTSecret  []byte
	Auth       *handler.AuthHandler
	Tenants    *handler.TenantsHandler
	Creds      *handler.CredentialsHandler
	JS         *handler.JetStreamHandler
	Mon        *handler.MonitorHandler
	FrontendFS fs.FS
}

func NewRouter(d Deps) *chi.Mux {
	r := chi.NewRouter()
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Authorization", "Content-Type"},
		AllowCredentials: false,
	}))
	r.Use(middleware.WithAudit(d.Pool))

	r.Route("/api/v1", func(r chi.Router) {
		r.Post("/auth/login", d.Auth.Login)

		r.Group(func(r chi.Router) {
			r.Use(middleware.RequireAdmin(d.JWTSecret))
			r.Post("/auth/refresh", d.Auth.Login)

			r.Route("/tenants", func(r chi.Router) {
				r.Get("/", d.Tenants.List)
				r.Post("/", d.Tenants.Create)
				r.Route("/{id}", func(r chi.Router) {
					r.Use(middleware.InjectTenant())
					r.Get("/", d.Tenants.Get)
					r.Put("/", d.Tenants.Update)
					r.Delete("/", d.Tenants.Delete)
					r.Post("/suspend", d.Tenants.Suspend)
					r.Post("/activate", d.Tenants.Activate)
					r.Get("/credentials", d.Creds.List)
					r.Post("/credentials", d.Creds.Issue)
					r.Delete("/credentials/{cid}", d.Creds.Revoke)
					r.Get("/jetstream/streams", d.JS.ListStreams)
					r.Post("/jetstream/streams", d.JS.CreateStream)
					r.Delete("/jetstream/streams/{name}", d.JS.DeleteStream)
					r.Post("/jetstream/streams/{name}/purge", d.JS.PurgeStream)
					r.Get("/jetstream/kv", d.JS.ListKV)
					r.Post("/jetstream/kv", d.JS.CreateKV)
					r.Delete("/jetstream/kv/{bucket}", d.JS.DeleteKV)
				})
			})
			r.Get("/monitor/server", d.Mon.Servers)
			r.Get("/monitor/tenants", d.Mon.Tenants)
			r.Get("/monitor/tenants/{id}", d.Mon.TenantStats)
			r.Get("/ws/monitor", d.Mon.WebSocket)
		})
	})

	// WebSocket upgrade must also bypass the static handler; chi's
	// WebSocket route above is registered, but the static fallback
	// would catch /ws/ if the route didn't match. Use a guarded mount.
	if d.FrontendFS != nil {
		r.Get("/*", httpSwaggerRedirectGuard(staticHandler(d.FrontendFS)))
	}
	return r
}

// httpSwaggerRedirectGuard is a placeholder; the real guard against
// intercepting /api/ and /ws/ is the router's route precedence. Kept
// here for future tightening (e.g. asset path whitelist).
func httpSwaggerRedirectGuard(h http.Handler) http.Handler { return h }
```

Wait — chi's `/*` will match any path that no other route handled. Since `/api/v1/*` and `/ws/monitor` are registered above, they take precedence and never fall through. Verify this is the case and simplify.

Simpler version of the static mount:

```go
	if d.FrontendFS != nil {
		r.Handle("/*", staticHandler(d.FrontendFS))
	}
```

Replace the conditional block with this. Remove the `httpSwaggerRedirectGuard` placeholder.

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd backend && go test ./internal/api/... -v`
Expected: all tests pass (static handler tests + router test).

- [ ] **Step 5: Modify `backend/cmd/server/main.go` to pass `frontendFS`**

In the `Deps{}` literal in `run()` (around line 117), add `FrontendFS: frontendFS,`. Also add `frontendFS` (the package-level variable from `frontend.go`) to the `api.Deps`:

```go
	router := api.NewRouter(api.Deps{
		Pool:       pool,
		JWTSecret:  cfg.JWTSecret,
		Auth:       handler.NewAuthHandler(pool, cfg.JWTSecret, cfg.JWTExpiry),
		Tenants:    handler.NewTenantsHandler(tenantSvc),
		Creds:      handler.NewCredentialsHandler(credSvc),
		JS:         handler.NewJetStreamHandler(jsAdmin),
		Mon:        handler.NewMonitorHandler(mon, monHub),
		FrontendFS: frontendFS,
	})
```

- [ ] **Step 6: Verify it compiles**

Run: `cd backend && go build ./...`
Expected: builds successfully.

- [ ] **Step 7: Commit**

```bash
git add backend/internal/api/router.go backend/internal/api/router_test.go backend/cmd/server/main.go
git commit -m "feat(api): mount embedded frontend at root with SPA fallback"
```

---

## Task 4: Update Dockerfile to include frontend build

**Files:**
- Modify: `backend/Dockerfile`
- Create: `frontend/.dockerignore`

- [ ] **Step 1: Create `frontend/.dockerignore`**

```
node_modules
dist
.vite
*.local
.DS_Store
```

- [ ] **Step 2: Rewrite `backend/Dockerfile`**

```dockerfile
FROM node:22-alpine AS frontend

WORKDIR /src/frontend
COPY frontend/package.json frontend/pnpm-lock.yaml* ./
RUN corepack enable && corepack prepare pnpm@latest --activate && \
    pnpm install --frozen-lockfile
COPY frontend ./
RUN pnpm build

FROM golang:1.25-alpine AS builder

RUN apk add --no-cache git

WORKDIR /src
COPY backend/go.mod backend/go.sum ./backend/
RUN cd backend && go mod download

# Copy backend source
COPY backend/ ./backend/

# Copy frontend build output into the embed path used by //go:embed
COPY --from=frontend /src/frontend/dist ./backend/cmd/server/frontend_dist/

RUN cd backend && CGO_ENABLED=0 go build -ldflags="-s -w" -o /bin/server ./cmd/server
RUN cd backend && CGO_ENABLED=0 go build -ldflags="-s -w" -o /bin/migrate ./cmd/migrate

FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata
COPY --from=builder /bin/server /usr/local/bin/server
COPY --from=builder /bin/migrate /usr/local/bin/migrate
COPY backend/internal/db/migrations /migrations

EXPOSE 8080
ENTRYPOINT ["server"]
```

- [ ] **Step 3: Verify the Dockerfile builds**

Run: `cd /workspace/nats-admin && docker build -f backend/Dockerfile -t nats-admin:test .`
Expected: builds successfully. The resulting image, when run with `docker run -p 8080:8080 nats-admin:test` and pointed at a NATS/Postgres, should serve the embedded frontend at `http://localhost:8080/`.

- [ ] **Step 4: Commit**

```bash
git add backend/Dockerfile frontend/.dockerignore
git commit -m "feat(docker): build frontend in dockerfile, embed into binary"
```

---

## Task 5: Create GitHub Actions release workflow

**Files:**
- Create: `.github/workflows/release.yml`

- [ ] **Step 1: Create `.github/workflows/release.yml`**

```yaml
name: Release

on:
  release:
    types: [published]

permissions:
  contents: write
  packages: write

jobs:
  build-frontend:
    name: Build Frontend
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - uses: pnpm/action-setup@v4
        with:
          version: 9

      - uses: actions/setup-node@v4
        with:
          node-version: 22
          cache: pnpm
          cache-dependency-path: frontend/pnpm-lock.yaml

      - name: Install dependencies
        run: pnpm install --frozen-lockfile
        working-directory: frontend

      - name: Build
        run: pnpm build
        working-directory: frontend

      - name: Upload dist artifact
        uses: actions/upload-artifact@v4
        with:
          name: frontend-dist
          path: frontend/dist
          retention-days: 1

  build-binary:
    name: Build Linux Binary
    needs: build-frontend
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Download frontend dist
        uses: actions/download-artifact@v4
        with:
          name: frontend-dist
          path: backend/cmd/server/frontend_dist

      - uses: actions/setup-go@v5
        with:
          go-version: 1.25
          cache-dependency-path: backend/go.sum

      - name: Build
        working-directory: backend
        env:
          CGO_ENABLED: 0
          GOOS: linux
          GOARCH: amd64
        run: |
          go build -ldflags="-s -w" -o nats-admin-linux-amd64 ./cmd/server

      - name: Upload binary to release
        uses: softprops/action-gh-release@v2
        with:
          files: backend/nats-admin-linux-amd64

  build-docker:
    name: Build & Push Docker Image
    needs: build-frontend
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Download frontend dist
        uses: actions/download-artifact@v4
        with:
          name: frontend-dist
          path: frontend/dist

      - name: Set up Docker Buildx
        uses: docker/setup-buildx-action@v3

      - name: Login to ghcr.io
        uses: docker/login-action@v3
        with:
          registry: ghcr.io
          username: ${{ github.actor }}
          password: ${{ secrets.GITHUB_TOKEN }}

      - name: Extract version
        id: meta
        run: |
          VERSION="${GITHUB_REF_NAME}"
          echo "version=${VERSION}" >> $GITHUB_OUTPUT
          echo "owner=${GITHUB_REPOSITORY_OWNER,,}" >> $GITHUB_OUTPUT

      - name: Build and push
        uses: docker/build-push-action@v6
        with:
          context: .
          file: backend/Dockerfile
          push: true
          tags: |
            ghcr.io/${{ steps.meta.outputs.owner }}/nats-admin:${{ steps.meta.outputs.version }}
            ghcr.io/${{ steps.meta.outputs.owner }}/nats-admin:latest
          cache-from: type=gha
          cache-to: type=gha,mode=max
```

- [ ] **Step 2: Validate YAML syntax**

Run: `python3 -c "import yaml; yaml.safe_load(open('.github/workflows/release.yml'))"`
Expected: no output (valid).

- [ ] **Step 3: Commit**

```bash
git add .github/workflows/release.yml
git commit -m "ci: build single binary + docker image on release"
```

---

## Task 6: Local smoke test of the binary build

**Files:** none

- [ ] **Step 1: Build the frontend locally**

Run: `cd /workspace/nats-admin/frontend && pnpm install --frozen-lockfile && pnpm build`
Expected: produces `frontend/dist/` with `index.html` and `assets/`.

- [ ] **Step 2: Copy dist into embed path and build the binary**

```bash
cd /workspace/nats-admin
rm -rf backend/cmd/server/frontend_dist/*
cp -r frontend/dist/* backend/cmd/server/frontend_dist/
cd backend
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o /tmp/nats-admin ./cmd/server
```

Expected: `/tmp/nats-admin` exists, size roughly 20-30 MB.

- [ ] **Step 3: Inspect embedded files in the binary**

Run: `strings /tmp/nats-admin | grep -E '<html>|<div id="root"' | head -5`
Expected: HTML content from `index.html` appears in the binary output.

- [ ] **Step 4: Clean up the local dist copy**

```bash
rm -rf /workspace/nats-admin/backend/cmd/server/frontend_dist/*
touch /workspace/nats-admin/backend/cmd/server/frontend_dist/.gitkeep
```

- [ ] **Step 5: Commit (no changes expected, skip if clean)**

```bash
git status
# Should show no diffs
```

---

## Self-Review

**1. Spec coverage:**
- Trigger on `release: published` → Task 5 ✅
- Build frontend with pnpm → Task 5 `build-frontend` ✅
- Embed via `//go:embed` → Task 1 ✅
- Serve at `/` with SPA fallback → Task 2 (helper) + Task 3 (mount) ✅
- Single binary as Release asset → Task 5 `build-binary` ✅
- Docker image with embedded frontend → Task 4 (Dockerfile) + Task 5 `build-docker` ✅
- linux/amd64 only ✅
- Tag format `vX.Y.Z` → Task 5 metadata extraction ✅
- Push to ghcr.io → Task 5 `build-docker` ✅

**2. Placeholder scan:**
- No "TBD", "TODO", "implement later" ✅
- No "add appropriate error handling" without code ✅
- All code blocks are complete ✅
- All commands have expected output ✅

**3. Type consistency:**
- `Deps.FrontendFS fs.FS` (router.go) used as `frontendFS` (embed.FS in frontend.go embeds as `fs.FS` since `embed.FS` implements `fs.FS`) ✅
- `staticHandler(fsys fs.FS)` accepts `fs.FS` ✅
- Artifact name `frontend-dist` is consistent between upload and download ✅
