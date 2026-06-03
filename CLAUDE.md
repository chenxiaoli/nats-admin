# NATS Multi-Tenant Web Admin

## 项目概述
NATS 多租户管理后台：Operator/Account/User 生命周期、JetStream 资源限制、凭证签发吊销、实时监控。

## Tech Stack（已决策）

| Layer       | Choice                       | Reason                              |
|-------------|------------------------------|-------------------------------------|
| Backend     | Go 1.22+                     | nkeys/jwt/v2/nats.go 无替代方案     |
| HTTP        | chi v5                       | 轻量，middleware 组合最干净         |
| DB          | PostgreSQL 16                |                                     |
| DB Access   | sqlc + pgx/v5                | 类型安全，不用 ORM                  |
| Frontend    | React 18 + Vite + TypeScript |                                     |
| UI          | shadcn/ui + TailwindCSS      |                                     |
| Admin Auth  | JWT HS256（独立于 NATS JWT） |                                     |
| Seed 加密   | AES-256-GCM + env master key |                                     |

## 子模块文档

@docs/nats-auth.md       NATS JWT 信任链、NKey、Resolver、核心代码模式
@docs/db-schema.md       PostgreSQL 完整表定义
@backend/CLAUDE.md       Go 文件结构、API 路由、实现模式、依赖
@frontend/CLAUDE.md      React 页面结构、组件约定

## 关键约束（全局，所有模块必须遵守）

1. Operator Seed 永远不进 DB，只存 env/secret manager
2. Account JWT 更新后必须 push 到 NATS resolver，仅更新 DB 无效
3. User 吊销必须重签 Account JWT（加 revocations map）再 push，不能只删 DB
4. User .creds 文件只在签发时返回一次，之后不重新拼接
5. Resolver push 失败必须回滚 DB（或补偿删除），不允许状态漂移
6. NKey 前缀：Operator seed `SO`，Account seed `SA`，User seed `SU`，搞错 JWT 验证失败
7. Account JWT 由 Operator 签；User JWT 由 Account 签——顺序不可反

## 开发启动

```bash
docker compose up -d nats postgres
go run ./cmd/bootstrap/...      # 首次：生成 Operator，输出写入 .env
go run ./cmd/migrate/... up
go run ./cmd/server/...
cd frontend && pnpm dev
```
