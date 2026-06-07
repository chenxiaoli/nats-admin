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
