# Session-Expired Login Modal Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** When an admin JWT expires, show a modal that re-prompts for credentials. On success, silently retry the original 401 request. After 3 failed password attempts, fall back to a full-page redirect to `/login`.

**Architecture:** Backend middleware emits `WWW-Authenticate: SessionExpired` header when the JWT is expired (other 401 modes stay unchanged). Frontend axios response interceptor detects the header and awaits a single in-flight `requestReauth()` promise (shared across concurrent 401s) before replaying the original request. A `ReauthProvider` mounts the modal and exposes the promise to the interceptor via a tiny pub-sub.

**Tech Stack:** Go 1.25 + chi v5 + golang-jwt/v5 (backend), React 18 + axios + react-router v7 (frontend). No new deps.

---

## File Map

| File | Role |
|------|------|
| `backend/internal/api/middleware/auth.go` | Modify — emit header on expired tokens |
| `backend/internal/api/middleware/auth_test.go` | New — table-driven tests for the three 401 modes |
| `frontend/src/lib/auth-events.ts` | New — pub-sub for `requestReauth()` promise |
| `frontend/src/components/auth/reauth-modal.tsx` | New — modal UI (email + password, cancel, login, 3-fail fallback) |
| `frontend/src/components/auth/reauth-provider.tsx` | New — mounts modal, exposes `requestReauth` |
| `frontend/src/api/client.ts` | Modify — response interceptor with retry queue |
| `frontend/src/App.tsx` | Modify — wrap with `ReauthProvider` |
| `frontend/src/pages/login/index.tsx` | Modify — read `?reason=session_expired`, show banner |

---

### Task 1: Backend middleware test for expired-token header

**Files:**
- Create: `backend/internal/api/middleware/auth_test.go`

- [ ] **Step 1: Write the failing test**

```go
package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

const testSecret = "test-secret-test-secret-test-secret-1234"

func mintToken(t *testing.T, exp time.Time, sub string) string {
	t.Helper()
	if sub == "" {
		sub = uuid.New().String()
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":  sub,
		"role": "admin",
		"exp":  exp.Unix(),
		"iat":  time.Now().Unix(),
	})
	signed, err := tok.SignedString([]byte(testSecret))
	if err != nil {
		t.Fatal(err)
	}
	return signed
}

func protectedHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}

func runMiddleware(req *http.Request) *httptest.ResponseRecorder {
	rr := httptest.NewRecorder()
	mw := RequireAdmin([]byte(testSecret))(http.HandlerFunc(protectedHandler))
	mw.ServeHTTP(rr, req)
	return rr
}

func TestRequireAdmin_MissingHeader(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	rr := runMiddleware(req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status: got %d", rr.Code)
	}
	if got := rr.Header().Get("WWW-Authenticate"); got != "" {
		t.Fatalf("expected no WWW-Authenticate, got %q", got)
	}
}

func TestRequireAdmin_MalformedHeader(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer not-a-jwt")
	rr := runMiddleware(req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status: got %d", rr.Code)
	}
	if got := rr.Header().Get("WWW-Authenticate"); got != "" {
		t.Fatalf("expected no WWW-Authenticate, got %q", got)
	}
}

func TestRequireAdmin_ExpiredToken(t *testing.T) {
	tok := mintToken(t, time.Now().Add(-1*time.Hour), "")
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	rr := runMiddleware(req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status: got %d", rr.Code)
	}
	if got := rr.Header().Get("WWW-Authenticate"); got != "SessionExpired" {
		t.Fatalf("WWW-Authenticate: got %q, want SessionExpired", got)
	}
}

func TestRequireAdmin_BadSignature(t *testing.T) {
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":  uuid.New().String(),
		"exp":  time.Now().Add(time.Hour).Unix(),
	})
	signed, _ := tok.SignedString([]byte("wrong-secret-wrong-secret-wrong-secret-123"))
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer "+signed)
	rr := runMiddleware(req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status: got %d", rr.Code)
	}
	if got := rr.Header().Get("WWW-Authenticate"); got != "" {
		t.Fatalf("expected no WWW-Authenticate, got %q", got)
	}
}

func TestRequireAdmin_NonUUIDSub(t *testing.T) {
	tok := mintToken(t, time.Now().Add(time.Hour), "not-a-uuid")
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	rr := runMiddleware(req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status: got %d", rr.Code)
	}
	if got := rr.Header().Get("WWW-Authenticate"); got != "" {
		t.Fatalf("expected no WWW-Authenticate, got %q", got)
	}
}

func TestRequireAdmin_ValidToken(t *testing.T) {
	tok := mintToken(t, time.Now().Add(time.Hour), "")
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	rr := runMiddleware(req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status: got %d, want 200", rr.Code)
	}
	if body := rr.Body.String(); body != "ok" {
		t.Fatalf("body: got %q", body)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd /workspace/nats-admin/backend && go test ./internal/api/middleware/ -run TestRequireAdmin_ExpiredToken -v`
Expected: FAIL — `RequireAdmin` doesn't emit the header yet.

- [ ] **Step 3: Commit**

```bash
cd /workspace/nats-admin && git add backend/internal/api/middleware/auth_test.go && git commit -m "test: add session-expired middleware tests"
```

---

### Task 2: Backend middleware implementation

**Files:**
- Modify: `backend/internal/api/middleware/auth.go`

- [ ] **Step 1: Replace middleware with version that emits header on expired tokens**

```go
package middleware

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type ctxKey int

const adminIDKey ctxKey = 1

func AdminID(ctx context.Context) uuid.UUID {
	v, _ := ctx.Value(adminIDKey).(uuid.UUID)
	return v
}

const wwwAuthSessionExpired = "SessionExpired"

func RequireAdmin(secret []byte) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			h := r.Header.Get("Authorization")
			if !strings.HasPrefix(h, "Bearer ") {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			raw := strings.TrimPrefix(h, "Bearer ")
			parsed, err := jwt.Parse(raw, func(t *jwt.Token) (any, error) {
				if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, jwt.ErrSignatureInvalid
				}
				return secret, nil
			})
			if err != nil {
				if errors.Is(err, jwt.ErrTokenExpired) {
					w.Header().Set("WWW-Authenticate", wwwAuthSessionExpired)
				}
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			if !parsed.Valid {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			claims, _ := parsed.Claims.(jwt.MapClaims)
			sub, _ := claims["sub"].(string)
			id, err := uuid.Parse(sub)
			if err != nil {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			ctx := context.WithValue(r.Context(), adminIDKey, id)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
```

- [ ] **Step 2: Run all middleware tests**

Run: `cd /workspace/nats-admin/backend && go test ./internal/api/middleware/ -v`
Expected: all 6 tests PASS.

- [ ] **Step 3: Run the rest of the backend tests to confirm no regression**

Run: `cd /workspace/nats-admin/backend && go test ./... -count=1`
Expected: PASS for every package.

- [ ] **Step 4: Commit**

```bash
cd /workspace/nats-admin && git add backend/internal/api/middleware/auth.go && git commit -m "feat(middleware): emit WWW-Authenticate: SessionExpired on expired JWT"
```

---

### Task 3: Frontend auth-events pub-sub

**Files:**
- Create: `frontend/src/lib/auth-events.ts`

- [ ] **Step 1: Create the file**

```ts
let inFlight: Promise<string> | null = null;

export function requestReauth(): Promise<string> {
  if (inFlight) return inFlight;
  inFlight = new Promise<string>((resolve, reject) => {
    const handler = (e: Event) => {
      const detail = (e as CustomEvent<{ token?: string; cancelled?: boolean }>).detail;
      bus.removeEventListener('reauth', handler);
      inFlight = null;
      if (detail.cancelled) reject(new Error('reauth cancelled'));
      else if (detail.token) resolve(detail.token);
      else reject(new Error('reauth failed'));
    };
    bus.addEventListener('reauth', handler);
    bus.dispatchEvent(new CustomEvent('reauth-request'));
  });
  return inFlight;
}

function completeReauth(detail: { token?: string; cancelled?: boolean }) {
  bus.dispatchEvent(new CustomEvent('reauth', { detail }));
}

export const reauthController = {
  /** Called by the provider to tell the bus "a modal opened, await completeReauth" */
  onRequest(handler: () => void) {
    bus.addEventListener('reauth-request', handler);
    return () => bus.removeEventListener('reauth-request', handler);
  },
  /** Called by the modal on successful login */
  succeed(token: string) {
    completeReauth({ token });
  },
  /** Called by the modal on cancel */
  cancel() {
    completeReauth({ cancelled: true });
  },
};

const bus = new EventTarget();
```

- [ ] **Step 2: Verify it compiles**

Run: `cd /workspace/nats-admin/frontend && pnpm build`
Expected: exit 0.

- [ ] **Step 3: Commit**

```bash
cd /workspace/nats-admin && git add frontend/src/lib/auth-events.ts && git commit -m "feat(frontend): add auth-events pub-sub for reauth coordination"
```

---

### Task 4: Frontend reauth modal component

**Files:**
- Create: `frontend/src/components/auth/reauth-modal.tsx`

- [ ] **Step 1: Create the file**

```tsx
import { useEffect, useState } from 'react';
import { client } from '@/api/client';
import { setToken, clearToken } from '@/lib/auth';

interface Props {
  open: boolean;
  onSolved: (token: string) => void;
  onCancelled: () => void;
}

const MAX_ATTEMPTS = 3;

export default function ReauthModal({ open, onSolved, onCancelled }: Props) {
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [err, setErr] = useState('');
  const [attempts, setAttempts] = useState(0);
  const [submitting, setSubmitting] = useState(false);

  useEffect(() => {
    if (open) {
      setEmail('');
      setPassword('');
      setErr('');
      setAttempts(0);
      setSubmitting(false);
    }
  }, [open]);

  if (!open) return null;

  const submit = async () => {
    setSubmitting(true);
    setErr('');
    try {
      const r = await client.post('/auth/login', { email, password });
      setToken(r.data.token);
      onSolved(r.data.token);
    } catch {
      const next = attempts + 1;
      setAttempts(next);
      if (next >= MAX_ATTEMPTS) {
        clearToken();
        window.location.href = '/login?reason=session_expired';
        return;
      }
      setErr(`登录失败（${next}/${MAX_ATTEMPTS}）`);
      setSubmitting(false);
    }
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40">
      <div className="w-full max-w-sm rounded-lg bg-white p-6 shadow-xl">
        <h3 className="text-lg font-semibold">会话已过期</h3>
        <p className="mt-2 text-sm text-slate-600">请重新登录以继续操作。</p>
        <label className="mt-4 mb-1 block text-sm">Email</label>
        <input
          autoFocus
          className="mb-3 w-full rounded-md border border-slate-300 px-3 py-2 text-sm"
          value={email}
          onChange={(e) => setEmail(e.target.value)}
          disabled={submitting}
        />
        <label className="mb-1 block text-sm">Password</label>
        <input
          type="password"
          className="mb-3 w-full rounded-md border border-slate-300 px-3 py-2 text-sm"
          value={password}
          onChange={(e) => setPassword(e.target.value)}
          onKeyDown={(e) => e.key === 'Enter' && !submitting && submit()}
          disabled={submitting}
        />
        {err && <div className="mb-2 text-sm text-red-600">{err}</div>}
        <div className="mt-4 flex justify-end gap-2">
          <button
            onClick={onCancelled}
            disabled={submitting}
            className="rounded-md border px-4 py-2 text-sm disabled:opacity-50"
          >
            取消
          </button>
          <button
            onClick={submit}
            disabled={submitting || !email || !password}
            className="rounded-md bg-slate-900 px-4 py-2 text-sm text-white hover:bg-slate-700 disabled:opacity-50"
          >
            登录
          </button>
        </div>
      </div>
    </div>
  );
}
```

- [ ] **Step 2: Verify it compiles**

Run: `cd /workspace/nats-admin/frontend && pnpm build`
Expected: exit 0.

- [ ] **Step 3: Commit**

```bash
cd /workspace/nats-admin && git add frontend/src/components/auth/reauth-modal.tsx && git commit -m "feat(frontend): add reauth modal component"
```

---

### Task 5: Frontend reauth provider

**Files:**
- Create: `frontend/src/components/auth/reauth-provider.tsx`

- [ ] **Step 1: Create the file**

```tsx
import { useEffect, useState } from 'react';
import ReauthModal from './reauth-modal';
import { reauthController } from '@/lib/auth-events';

export default function ReauthProvider({ children }: { children: React.ReactNode }) {
  const [open, setOpen] = useState(false);
  const [resolver, setResolver] = useState<((token: string) => void) | null>(null);
  const [rejecter, setRejecter] = useState<(() => void) | null>(null);

  useEffect(() => {
    return reauthController.onRequest(() => {
      setOpen(true);
      setResolver(() => (token: string) => reauthController.succeed(token));
      setRejecter(() => () => reauthController.cancel());
    });
  }, []);

  return (
    <>
      {children}
      <ReauthModal
        open={open}
        onSolved={(token) => {
          setOpen(false);
          resolver?.(token);
        }}
        onCancelled={() => {
          setOpen(false);
          rejecter?.();
        }}
      />
    </>
  );
}
```

- [ ] **Step 2: Verify it compiles**

Run: `cd /workspace/nats-admin/frontend && pnpm build`
Expected: exit 0.

- [ ] **Step 3: Commit**

```bash
cd /workspace/nats-admin && git add frontend/src/components/auth/reauth-provider.tsx && git commit -m "feat(frontend): add reauth provider to mount modal from event bus"
```

---

### Task 6: Frontend client response interceptor

**Files:**
- Modify: `frontend/src/api/client.ts`

- [ ] **Step 1: Replace the file**

```ts
import axios, { AxiosError, InternalAxiosRequestConfig } from 'axios';
import { getToken } from '@/lib/auth';
import { requestReauth } from '@/lib/auth-events';

export const client = axios.create({ baseURL: '/api/v1' });

client.interceptors.request.use((cfg) => {
  const tok = getToken();
  if (tok) cfg.headers.Authorization = `Bearer ${tok}`;
  return cfg;
});

client.interceptors.response.use(undefined, async (err: AxiosError) => {
  if (err.response?.status !== 401) throw err;
  if (err.config?.url === '/auth/login') throw err;
  if (err.response.headers['www-authenticate'] !== 'SessionExpired') throw err;

  let token: string;
  try {
    token = await requestReauth();
  } catch {
    throw err;
  }

  const retryCfg: InternalAxiosRequestConfig = {
    ...err.config,
    headers: { ...err.config.headers, Authorization: `Bearer ${token}` },
  };
  return client.request(retryCfg);
});
```

- [ ] **Step 2: Verify it compiles**

Run: `cd /workspace/nats-admin/frontend && pnpm build`
Expected: exit 0.

- [ ] **Step 3: Commit**

```bash
cd /workspace/nats-admin && git add frontend/src/api/client.ts && git commit -m "feat(frontend): add axios response interceptor with session-expired retry"
```

---

### Task 7: Wire ReauthProvider into App.tsx

**Files:**
- Modify: `frontend/src/App.tsx`

- [ ] **Step 1: Wrap RouterProvider with ReauthProvider**

Edit the file. The export block at the bottom becomes:

```tsx
import ReauthProvider from '@/components/auth/reauth-provider';

export default function App() {
  return (
    <QueryClientProvider client={qc}>
      <ReauthProvider>
        <RouterProvider router={router} />
      </ReauthProvider>
    </QueryClientProvider>
  );
}
```

Add the import at the top with the other component imports.

- [ ] **Step 2: Verify it compiles**

Run: `cd /workspace/nats-admin/frontend && pnpm build`
Expected: exit 0.

- [ ] **Step 3: Commit**

```bash
cd /workspace/nats-admin && git add frontend/src/App.tsx && git commit -m "feat(frontend): wrap router with reauth provider"
```

---

### Task 8: Login page banner for `?reason=session_expired`

**Files:**
- Modify: `frontend/src/pages/login/index.tsx`

- [ ] **Step 1: Update the login page to read the query string and show a banner**

Replace the file with:

```tsx
import { useState } from 'react';
import { useNavigate, useSearchParams } from 'react-router';
import { client } from '@/api/client';
import { setToken } from '@/lib/auth';

const REASONS: Record<string, string> = {
  session_expired: '会话已过期，请重新登录。',
};

export default function LoginPage() {
  const [email, setEmail] = useState('admin@example.com');
  const [password, setPassword] = useState('changeme');
  const [err, setErr] = useState('');
  const [params] = useSearchParams();
  const reason = params.get('reason');
  const banner = reason && REASONS[reason];
  const nav = useNavigate();

  const submit = async () => {
    try {
      const r = await client.post('/auth/login', { email, password });
      setToken(r.data.token);
      nav('/tenants');
    } catch {
      setErr('登录失败');
    }
  };

  return (
    <div className="mx-auto mt-24 max-w-sm rounded-lg border bg-white p-6 shadow-sm">
      <h1 className="mb-4 text-xl font-semibold">登录</h1>
      {banner && (
        <div className="mb-3 rounded-md border border-amber-300 bg-amber-50 px-3 py-2 text-sm text-amber-800">
          {banner}
        </div>
      )}
      <label className="mb-2 block text-sm">Email</label>
      <input className="mb-3 w-full rounded-md border border-slate-300 px-3 py-2 text-sm" value={email} onChange={(e) => setEmail(e.target.value)} />
      <label className="mb-2 block text-sm">Password</label>
      <input className="mb-3 w-full rounded-md border border-slate-300 px-3 py-2 text-sm" type="password" value={password} onChange={(e) => setPassword(e.target.value)} />
      {err && <div className="mb-2 text-sm text-red-600">{err}</div>}
      <button className="w-full rounded-md bg-slate-900 px-3 py-2 text-sm font-medium text-white hover:bg-slate-700" onClick={submit}>登录</button>
    </div>
  );
}
```

- [ ] **Step 2: Verify it compiles**

Run: `cd /workspace/nats-admin/frontend && pnpm build`
Expected: exit 0.

- [ ] **Step 3: Commit**

```bash
cd /workspace/nats-admin && git add frontend/src/pages/login/index.tsx && git commit -m "feat(frontend): show banner on login when redirected from session-expired fallback"
```

---

### Task 9: End-to-end manual smoke test

- [ ] **Step 1: Run the backend tests one last time**

Run: `cd /workspace/nats-admin/backend && go test ./... -count=1`
Expected: PASS for every package.

- [ ] **Step 2: Build the frontend**

Run: `cd /workspace/nats-admin/frontend && pnpm build`
Expected: exit 0.

- [ ] **Step 3: Manual scenarios (record results in commit body)**

Run the dev environment, then verify each of these manually:

- **A. Expired token triggers modal, success path**: log in, then in DevTools set `localStorage.setItem('admin_token', '<HS256 token with exp in the past>')`, refresh `/tenants`. The modal appears. Enter real credentials, the list loads without page navigation.
- **B. 3-fail fallback**: same as A but enter wrong password 3 times. After the 3rd failure, browser navigates to `/login?reason=session_expired` and a banner explains what happened.
- **C. Concurrent 401s coalesce**: open DevTools, on `/tenants` set the expired token, then trigger 5 simultaneous requests. Only one modal opens. After successful re-auth, all 5 requests succeed.
- **D. Cancel surfaces the 401**: with expired token, click "取消" in the modal. The original page (e.g. tenants list) shows an error.
- **E. Non-expired 401 doesn't open modal**: temporarily corrupt the token in localStorage (mutate one character), refresh. The page should not show the modal — it should show a generic error (or just fail).

If any scenario fails, file a follow-up; do not mark this plan complete.

- [ ] **Step 4: Commit any manual notes**

If you have notes worth keeping (e.g. a regression you noticed), add them as a commit. Otherwise this step is a no-op.

---

## Self-Review

1. **Spec coverage**:
   - Backend distinguishes expired from no token → Tasks 1, 2 ✓
   - Frontend response interceptor with retry queue → Task 6 ✓
   - Auth event bus → Task 3 ✓
   - Reauth modal UI → Task 4 ✓
   - Reauth provider → Task 5 ✓
   - App.tsx integration → Task 7 ✓
   - Login page banner → Task 8 ✓
   - Backend tests (3 cases) → Task 1 ✓
   - 3-fail fallback rule → Task 4 (logic) ✓
   - /auth/login exclusion → Task 6 ✓

2. **Placeholder scan**: No TBD/TODO. Every step has complete code.

3. **Type consistency**:
   - `requestReauth()` returns `Promise<string>` → matches modal's `onSolved: (token: string) => void` → matches `setToken(r.data.token)` return shape
   - `reauthController.succeed(token: string)` and `cancel()` shapes used consistently
   - `WWW-Authenticate: SessionExpired` written in backend, read as `'www-authenticate'` (axios lowercases all header keys) on the frontend
   - File names: `auth-events.ts` (not `events.ts`), `reauth-modal.tsx`, `reauth-provider.tsx` — referenced consistently

4. **No type drift detected.**
