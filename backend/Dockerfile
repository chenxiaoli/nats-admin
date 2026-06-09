FROM node:22-alpine AS frontend

WORKDIR /src/frontend
COPY frontend/package.json frontend/pnpm-lock.yaml* ./
RUN corepack enable && corepack prepare pnpm@9 --activate && \
    pnpm install --frozen-lockfile
COPY frontend ./
RUN pnpm build

FROM golang:1.25-alpine AS builder

RUN apk add --no-cache git

WORKDIR /src
COPY backend/go.mod backend/go.sum ./backend/
RUN cd backend && go mod download

COPY backend/ ./backend/

COPY --from=frontend /src/frontend/dist ./backend/cmd/nats-admin/frontend_dist/
COPY backend/internal/db/migrations ./backend/cmd/nats-admin/migrations/

RUN cd backend && CGO_ENABLED=0 go build -ldflags="-s -w" -o /bin/nats-admin ./cmd/nats-admin

FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata
COPY --from=builder /bin/nats-admin /usr/local/bin/nats-admin

EXPOSE 8080
ENTRYPOINT ["nats-admin"]
