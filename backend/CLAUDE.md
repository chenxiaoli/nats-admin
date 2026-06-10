# Backend (Go)

## 文件结构

```
backend/
├── cmd/
│   ├── server/main.go         HTTP server 入口
│   ├── migrate/main.go        DB migration runner
│   └── bootstrap/main.go      一次性 Operator 初始化（见 @../docs/nats-auth.md）
├── internal/
│   ├── config/config.go       env config（viper）
│   ├── operator/
│   │   ├── operator.go        Operator NKey 加载、Account JWT 签发
│   │   └── operator_test.go
│   ├── tenant/
│   │   ├── service.go         Account 生命周期业务逻辑
│   │   ├── repository.go      sqlc 查询封装
│   │   └── resolver.go        push/delete Account JWT 到 NATS resolver
│   ├── credential/
│   │   ├── service.go         User JWT 签发、吊销、轮换
│   │   ├── crypto.go          AES-256-GCM（见 @../docs/nats-auth.md）
│   │   └── credential_test.go
│   ├── jetstream/
│   │   ├── manager.go         per-tenant NATS 连接池
│   │   └── admin.go           Stream/Consumer/KV/OBJ CRUD
│   ├── monitor/
│   │   ├── sysaccount.go      System Account 订阅，缓存统计
│   │   └── metrics.go         WebSocket 广播
│   ├── api/
│   │   ├── router.go
│   │   ├── middleware/
│   │   │   ├── auth.go        Admin JWT 校验
│   │   │   ├── tenant.go      tenant_id 注入 context
│   │   │   └── audit.go       自动写审计日志
│   │   └── handler/
│   │       ├── auth.go
│   │       ├── tenants.go
│   │       ├── credentials.go
│   │       ├── jetstream.go
│   │       └── monitor.go
│   └── db/
│       ├── migrations/        golang-migrate .sql 文件
│       └── sqlc/              sqlc 生成代码（不手写）
├── go.mod
└── sqlc.yaml
```

## API 路由

```
POST   /api/v1/auth/login
POST   /api/v1/auth/refresh

GET    /api/v1/tenants
POST   /api/v1/tenants                            # 生成 NKey → 签 JWT → push → 入 DB
GET    /api/v1/tenants/:id
PUT    /api/v1/tenants/:id                        # 更新 limits → 重签 → push
DELETE /api/v1/tenants/:id
POST   /api/v1/tenants/:id/suspend                # delete from resolver
POST   /api/v1/tenants/:id/activate               # re-push JWT

GET    /api/v1/tenants/:id/credentials
POST   /api/v1/tenants/:id/credentials            # 返回 .creds（仅此一次）
DELETE /api/v1/tenants/:id/credentials/:cid       # 重签 Account JWT + push
POST   /api/v1/tenants/:id/credentials/:cid/rotate

GET    /api/v1/tenants/:id/jetstream/streams
POST   /api/v1/tenants/:id/jetstream/streams
DELETE /api/v1/tenants/:id/jetstream/streams/:name
POST   /api/v1/tenants/:id/jetstream/streams/:name/purge
GET    /api/v1/tenants/:id/jetstream/kv
POST   /api/v1/tenants/:id/jetstream/kv
DELETE /api/v1/tenants/:id/jetstream/kv/:bucket

GET    /api/v1/monitor/server
GET    /api/v1/monitor/tenants
GET    /api/v1/monitor/tenants/:id
WS     /api/v1/ws/monitor

GET    /api/v1/settings/api-keys                  # 列出当前 admin 的 keys
POST   /api/v1/settings/api-keys                  # 创建 key（原始值仅响应一次）
DELETE /api/v1/settings/api-keys/:id              # 吊销（不可恢复）
```

## 认证

- 浏览器：`/api/v1/auth/login` 拿到 HS256 JWT，作为 `Authorization: Bearer <jwt>`
- 后端/CI：API key `nak_live_<32 base62>`，同样 `Authorization: Bearer nak_live_...`；中间件按 `nak_live_` 前缀分流，SHA-256 散列后查表，命中即认证为对应 admin（权限继承）；revoke 立即失效

## 连接池（per-tenant）

```go
// manager.go — 避免每次请求建立新连接
// 用 nats.UserJWTAndSeed() 替代落盘 .creds，防止 seed 临时文件泄露
type Manager struct {
    mu    sync.RWMutex
    conns map[uuid.UUID]*nats.Conn
}

func (m *Manager) GetJS(tenantID uuid.UUID, jwtStr, seed string) (nats.JetStreamContext, error) {
    m.mu.RLock()
    conn, ok := m.conns[tenantID]
    m.mu.RUnlock()
    if ok && conn.IsConnected() {
        return conn.JetStream()
    }
    // write-lock，double-check，建立新连接
    nc, _ := nats.Connect(url,
        nats.UserJWTAndSeed(jwtStr, seed),   // 不落盘
        nats.ClosedHandler(func(c *nats.Conn) {
            m.mu.Lock()
            delete(m.conns, tenantID)        // 必须清理，防止死循环重连
            m.mu.Unlock()
        }),
    )
    m.conns[tenantID] = nc
    return nc.JetStream()
}
```

## 环境变量

```env
PORT=8080
ENV=development
DATABASE_URL=postgres://admin:secret@localhost:5432/nats_admin?sslmode=disable
NATS_URL=nats://localhost:4222
OPERATOR_SEED=SO...                  # 绝不入 DB
SYSTEM_ACCOUNT_SEED=SA...
MASTER_KEY=<64-char hex, 32 bytes>   # AES-256 key
JWT_SECRET=<64 chars>
JWT_EXPIRY=24h
BOOTSTRAP_ADMIN_EMAIL=admin@example.com
BOOTSTRAP_ADMIN_PASSWORD=changeme
```

## 依赖

```
github.com/nats-io/nats.go         v1.36.0
github.com/nats-io/nkeys           v0.4.7
github.com/nats-io/jwt/v2          v2.7.0
github.com/go-chi/chi/v5           v5.1.0
github.com/go-chi/cors             v1.2.1
github.com/jackc/pgx/v5            v5.6.0
github.com/golang-jwt/jwt/v5       v5.2.1   # admin web JWT，≠ NATS JWT
github.com/google/uuid             v1.6.0
golang.org/x/crypto                v0.24.0  # bcrypt
github.com/spf13/viper             v1.19.0
# tools
github.com/sqlc-dev/sqlc           v1.27.0
github.com/golang-migrate/migrate/v4 v4.17.0
```

## 开发命令

```bash
sqlc generate                              # 修改 SQL 后执行
go run ./cmd/migrate/... up
go run ./cmd/server/...
go test ./internal/... -race -count=1
```
