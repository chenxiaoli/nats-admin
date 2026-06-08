# Session-Expired Login Modal

## Goal

When an admin's JWT expires mid-session, the frontend should display a modal that re-prompts for email + password. On success, the request that originally 401'd is silently retried with the new token, and the user stays on the same page. After 3 failed attempts, the modal falls back to a full-page redirect to `/login`.

## Background

**Current state**

- `RequireAdmin` middleware returns plain `401 unauthorized` for every failure mode (missing header, bad signature, expired token) — the frontend cannot tell them apart.
- The axios client (`frontend/src/api/client.ts`) has a request interceptor only. A 401 propagates as a rejection with no central handling.
- The token is stored in `localStorage` under `admin_token` and never inspected — `isAuthenticated()` returns true even for an expired token.
- `POST /auth/refresh` is wired to the same `Login` handler (stub). There is no real refresh flow.
- shadcn `Dialog` is not installed. The existing `frontend/src/components/ui/confirm-dialog.tsx` uses a hand-rolled overlay pattern (`fixed inset-0 z-50 … bg-black/40`); new modals should follow the same pattern for visual consistency.

**Why a modal, not a redirect**

A redirect to `/login` would lose the user's current page state, scroll position, and any in-flight form drafts. A modal preserves all of that and is the standard pattern for session expiry in admin UIs (Grafana, GitHub, etc.).

## Architecture

### Backend — distinguish "expired" from "no token"

Modify `RequireAdmin` in `backend/internal/api/middleware/auth.go` to detect the expired case specifically and emit a `WWW-Authenticate: SessionExpired` response header. Other failure modes (missing header, bad signature, malformed claims) keep returning `401 unauthorized` with no header.

Implementation: switch to `jwt.ParseWithClaims` and check `errors.Is(err, jwt.ErrTokenExpired)` before falling through to the generic 401.

### Frontend — response interceptor with shared re-auth

Replace the client in `frontend/src/api/client.ts` so it has both request and response interceptors. The response interceptor detects the session-expired 401 and replays the original request after a successful re-auth.

The re-auth itself is a **single in-flight promise**. If N concurrent requests all 401 with `SessionExpired`, the interceptor awaits one `requestReauth()` call, then retries all N requests. This prevents N modals from opening at once.

```
[Request 1] ─401 SessionExpired─┐
[Request 2] ─401 SessionExpired─┼─→ requestReauth() → token ─→ replay 1
[Request 3] ─401 SessionExpired─┘                          → replay 2
                                                            → replay 3
```

The `/auth/login` endpoint is excluded from this path — a 401 from re-auth itself is propagated as a normal error so the modal can display it inline.

### Frontend — global auth event bus

A small pub-sub (`frontend/src/lib/auth-events.ts`) decouples the axios interceptor (which lives outside React) from the modal. The interceptor calls `authEvents.requestReauth(): Promise<string>`. A `ReauthProvider` mounted in `App.tsx` subscribes to this and renders the modal.

The promise resolves with the new token on successful re-login and rejects on cancel or 3-failure fallback.

### Frontend — the modal

New `frontend/src/components/auth/reauth-modal.tsx`. Follows the `confirm-dialog.tsx` overlay pattern (no shadcn `Dialog` dependency). Contents:

- Title: "会话已过期"
- Body: "请重新登录以继续操作。"
- Email + password fields
- Inline error display on bad credentials
- Buttons: 取消 (rejects the pending promise, original 401 surfaces to caller) / 登录 (calls `client.post('/auth/login', …)`, saves token, resolves the pending promise)
- On 3rd consecutive 401 from `/auth/login`: `clearToken()` + `window.location.href = '/login?reason=session_expired'`. The counter resets on each successful re-auth and on each full page load (a fresh tab starts at 0). Cancelling the modal does not increment the counter.

## Files Changed

| File | Action |
|------|--------|
| `backend/internal/api/middleware/auth.go` | Modify — emit `WWW-Authenticate: SessionExpired` when `jwt.ErrTokenExpired` |
| `backend/internal/api/middleware/auth_test.go` | Create — unit tests for the three 401 cases |
| `frontend/src/api/client.ts` | Modify — add response interceptor with retry queue |
| `frontend/src/lib/auth-events.ts` | Create — pub-sub for `requestReauth()` |
| `frontend/src/components/auth/reauth-modal.tsx` | Create — modal UI |
| `frontend/src/components/auth/reauth-provider.tsx` | Create — mounts modal, exposes `requestReauth` to interceptor |
| `frontend/src/App.tsx` | Modify — wrap `<RouterProvider>` with `<ReauthProvider>` (sibling of `QueryClientProvider`, both inside any other providers) |
| `frontend/src/pages/login/index.tsx` | Modify — read `?reason=session_expired` and show a banner |

## API Contract Change

`POST /api/v1/auth/login` — **no change** (modal re-uses it).

`GET/PUT/POST/DELETE /api/v1/tenants`, etc. — **response change only**:
- Previously: expired token → `401 unauthorized` (text/plain)
- Now: expired token → `401 unauthorized` (text/plain) with header `WWW-Authenticate: SessionExpired`
- Other 401s (missing token, bad sig): unchanged

The body is unchanged so existing curl/scripts that only check status code keep working.

## Testing

**Backend** (`backend/internal/api/middleware/auth_test.go`)

- Missing `Authorization` header → 401, no `WWW-Authenticate` header
- Malformed header (`Bearer foo`) → 401, no `WWW-Authenticate` header
- Valid token with future `exp` → 200 (or handler's response), request reaches handler
- Expired token → 401 with `WWW-Authenticate: SessionExpired` header
- Token signed with wrong secret → 401, no `WWW-Authenticate` header
- Token with non-uuid `sub` → 401, no `WWW-Authenticate` header

**Frontend** (manual)

- Set `admin_token` in localStorage to an HS256 token whose `exp` is in the past; navigate to `/tenants`; modal appears; enter correct creds; list loads without page change
- Same setup, enter wrong password 3×; after 3rd failure, browser navigates to `/login` with `?reason=session_expired` and a banner explains what happened
- 5 concurrent requests when token is expired; only one modal opens; all 5 requests succeed after re-auth
- Cancel the modal; original 401 surfaces to the page (e.g. tenant list shows an error toast)
- 401 from a non-expired-token failure (e.g. tampered signature) does NOT open the modal

## Out of Scope

- Real `/auth/refresh` endpoint — keep the stub. Re-auth is by password.
- Auto-refresh on a timer based on the `exp` claim — we react to 401s, not clocks.
- "Remember me" / extending the session — admin sessions are infrequent and 24h is fine.
- Replacing the existing hand-rolled overlay pattern with shadcn `Dialog` — visual consistency wins.
- Silent retries that originate from the login page (not the modal) — login page uses a full page already, no modal needed there.
