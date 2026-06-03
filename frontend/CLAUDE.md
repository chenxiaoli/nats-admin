# Frontend (React)

## 技术栈
React 18 + Vite + TypeScript + shadcn/ui + TailwindCSS + TanStack Query v5 + React Router v6

## 页面结构

```
/login
/dashboard                  全局概览（server stats、tenant count、active connections）
/tenants                    租户列表（搜索、status 过滤）
/tenants/new                创建租户（JetStream limits 表单）
/tenants/:id                租户详情
  ├── ?tab=overview         连接数、消息流量、JetStream 用量
  ├── ?tab=credentials      凭证列表 + 签发入口
  ├── ?tab=jetstream        Streams / KV 管理
  └── ?tab=audit            操作日志
/monitor                    实时监控（WebSocket，5s 刷新）
/settings                   Admin 用户管理
```

## 文件结构

```
src/
├── api/
│   ├── client.ts           axios 实例，自动注入 Bearer token
│   ├── tenants.ts          useTenants / useTenant / useCreateTenant ...
│   ├── credentials.ts      useCredentials / useIssueCredential ...
│   ├── jetstream.ts
│   └── monitor.ts          useMonitor（WebSocket hook）
├── pages/
│   ├── login/
│   ├── dashboard/
│   ├── tenants/
│   │   ├── list.tsx
│   │   ├── new.tsx
│   │   └── detail/
│   │       ├── index.tsx   tab 路由
│   │       ├── overview.tsx
│   │       ├── credentials.tsx
│   │       └── jetstream.tsx
│   └── monitor/
├── components/
│   ├── layout/             Sidebar + TopBar
│   ├── tenant/             TenantForm、StatusBadge、LimitsCard
│   ├── credential/         CredentialTable、IssueDialog、CredsDownload
│   └── monitor/            MetricsCard、ConnectionList
└── lib/
    ├── auth.ts             token 存取、useAuth hook
    └── utils.ts            formatBytes、formatNumber
```

## API Hook 约定

```ts
// 统一用 TanStack Query，mutation 后 invalidate 相关 query
export function useCreateTenant() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (req: CreateTenantReq) => client.post('/tenants', req),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['tenants'] }),
  })
}

// .creds 文件下载：后端返回 text/plain，前端 Blob 下载
// 展示一次性提示：「凭证只显示一次，请立即下载」
```

## WebSocket 监控

```ts
// 连接到 /api/v1/ws/monitor，每 5s 收到 ServerStats 推送
export function useMonitorWS() {
  const [stats, setStats] = useState<ServerStats | null>(null)
  useEffect(() => {
    const ws = new WebSocket(`${WS_BASE}/api/v1/ws/monitor`)
    ws.onmessage = e => setStats(JSON.parse(e.data))
    return () => ws.close()
  }, [])
  return stats
}
```

## 约定

- shadcn/ui 组件直接用，不二次封装
- 表单用 react-hook-form + zod 校验
- 数字全部用 `formatBytes()` / `Intl.NumberFormat` 显示，不裸输出
- `-1` 显示为「不限制」
- 危险操作（删除租户、吊销凭证）必须二次确认 Dialog

## 开发命令

```bash
pnpm dev
pnpm build
pnpm lint
```
