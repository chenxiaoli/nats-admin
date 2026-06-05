# 补全现有功能 — React Router v7

## 目标

补全 nats-admin 前端所有缺失的基础功能，使现有后端 API 全部可用：路由守卫、布局框架、租户管理 UI、凭证管理 UI。

## 依赖变更

```diff
- "react-router-dom": "^6.26.0"
+ "react-router": "^7.6.0"
```

移除 `react-router-dom`，安装 `react-router` v7。

## 路由架构

使用 React Router v7 Data Router（`createBrowserRouter` + `RouterProvider`）。

```ts
// src/router.ts
const router = createBrowserRouter([
  {
    path: '/login',
    element: <LoginPage />,
    loader: loginLoader,       // 已登录则跳转 /
  },
  {
    path: '/',
    element: <AuthLayout />,
    loader: authLoader,        // 未登录跳转 /login
    children: [
      { index: true, element: <Navigate to="/tenants" replace /> },
      {
        path: 'tenants',
        children: [
          { index: true, element: <TenantsList /> },
          { path: 'new', element: <TenantNew /> },
          {
            path: ':id',
            element: <TenantDetail />,
            children: [
              { index: true, element: <TenantOverview /> },
              { path: 'credentials', element: <TenantCredentials /> },
              { path: 'audit', element: <TenantAudit /> },
            ],
          },
        ],
      },
    ],
  },
]);
```

### authLoader

- 从 localStorage 读取 token
- 无 token → `redirect('/login')`
- 有 token → 正常渲染

### loginLoader

- 有 token → `redirect('/tenants')`
- 无 token → 渲染登录页

## 布局（AuthLayout）

经典 Sidebar + TopBar + `<Outlet />`。

### Sidebar

| 菜单项 | 路径 | 状态 |
|--------|------|------|
| 租户 | `/tenants` | 可用 |
| 监控 | `/monitor` | 占位（disabled） |
| 设置 | `/settings` | 占位（disabled） |

当前路由高亮对应菜单项。

### TopBar

- 左侧：当前页面标题
- 右侧：用户邮箱 + 退出按钮（清除 token，跳转 `/login`）

## 新增文件结构

```
src/
├── router.tsx                    createBrowserRouter 定义
├── main.tsx                      改用 RouterProvider
├── lib/
│   └── auth.ts                   getToken / clearToken / isAuthenticated
├── components/
│   ├── layout/
│   │   ├── auth-layout.tsx       Sidebar + TopBar + Outlet
│   │   ├── sidebar.tsx           左侧导航
│   │   └── topbar.tsx            顶栏
│   ├── ui/
│   │   ├── confirm-dialog.tsx    二次确认弹窗
│   │   └── badge.tsx             状态标签（active/suspended）
│   └── credential/
│       └── creds-dialog.tsx      .creds 展示 + 复制 + 下载
├── pages/
│   ├── login/index.tsx           （已有，微调）
│   ├── tenants/
│   │   ├── list.tsx              （已有，微调）
│   │   ├── new.tsx               （已有，保持）
│   │   └── detail/
│   │       ├── index.tsx         标签页容器
│   │       ├── overview.tsx      概览 + 操作按钮
│   │       ├── credentials.tsx   凭证管理标签页
│   │       └── audit.tsx         审计日志标签页
├── api/
│   ├── client.ts                 （已有，保持）
│   ├── tenants.ts                （已有，增加 useUpdateTenant 等）
│   └── credentials.ts            新增
```

## 各页面详细设计

### AuthLayout (`components/layout/auth-layout.tsx`)

```
┌──────────────────────────────────────────┐
│  NATS Admin              admin@…   [退出] │
├──────────┬───────────────────────────────┤
│ 租户     │                               │
│ 监控(dis)│       <Outlet />              │
│ 设置(dis)│                               │
└──────────┴───────────────────────────────┘
```

- Sidebar 宽 200px，深色背景
- TopBar 高 48px，白色背景，底部阴影
- 内容区占满剩余空间

### TenantDetail (`pages/tenants/detail/index.tsx`)

标签页容器，使用 URL query 或嵌套路由区分：

- `?tab=overview`（默认）→ `<TenantOverview />`
- `?tab=credentials` → `<TenantCredentials />`
- `?tab=audit` → `<TenantAudit />`

顶部显示租户名称 + 返回按钮。

### TenantOverview (`pages/tenants/detail/overview.tsx`)

信息卡片展示所有字段（名称、Slug、状态 Badge、Account 公钥、各限制值、Account JWT）。

操作按钮区：
- **编辑限制**：弹窗表单，修改 JetStream/连接/订阅限制，调用 `PUT /tenants/:id`
- **挂起**：确认对话框 → `POST /tenants/:id/suspend`（仅在 status=active 时显示）
- **激活**：确认对话框 → `POST /tenants/:id/activate`（仅在 status=suspended 时显示）
- **删除**：红色按钮 + 确认对话框（输入租户名称确认）→ `DELETE /tenants/:id`，成功后跳转列表

### TenantCredentials (`pages/tenants/detail/credentials.tsx`)

表格列：名称、公钥（截断+复制按钮）、Pub 权限、Sub 权限、状态、创建时间、操作。

- **签发**按钮 → 弹窗表单（名称、pub_allow 文本框每行一个 subject、sub_allow 同上）
  - 提交 → `POST /tenants/:id/credentials`
  - 成功 → 弹出 `<CredsDialog>` 展示 .creds 内容
- **吊销**按钮 → 确认对话框 → `DELETE /tenants/:id/credentials/:cid`
  - 成功后刷新凭证列表

### CredsDialog (`components/credential/creds-dialog.tsx`)

- 顶部警告条："凭证仅显示一次，请立即保存"
- `<pre>` 展示 .creds 完整内容
- "复制到剪贴板" 按钮
- "下载 .creds 文件" 按钮（Blob 下载）
- "我已保存" 关闭按钮

### TenantAudit (`pages/tenants/detail/audit.tsx`)

表格列：时间、操作、资源、详情（JSON 展开）、IP。

分页：简单的前/后翻页（limit + offset query params）。

## 新增 API Hooks

### `src/api/credentials.ts`

```ts
interface Credential {
  id: string;
  name: string;
  user_public_key: string;
  pub_allow: string[];
  sub_allow: string[];
  revoked_at: string | null;
  created_at: string;
}

useCredentials(tenantId: string)        // GET /tenants/:id/credentials
useIssueCredential(tenantId: string)    // POST /tenants/:id/credentials → returns .creds text
useRevokeCredential(tenantId: string)   // DELETE /tenants/:id/credentials/:cid
```

### `src/api/tenants.ts`（追加）

```ts
useUpdateTenant()                       // PUT /tenants/:id
useSuspendTenant()                      // POST /tenants/:id/suspend
useActivateTenant()                     // POST /tenants/:id/activate
useDeleteTenant()                       // DELETE /tenants/:id
```

## 后端变更

无需后端变更。所有 API endpoint 已实现可用：

| Endpoint | 方法 | 状态 |
|----------|------|------|
| `/api/v1/auth/login` | POST | 已实现 |
| `/api/v1/tenants` | GET, POST | 已实现 |
| `/api/v1/tenants/:id` | GET, PUT, DELETE | 已实现 |
| `/api/v1/tenants/:id/suspend` | POST | 已实现 |
| `/api/v1/tenants/:id/activate` | POST | 已实现 |
| `/api/v1/tenants/:id/credentials` | GET, POST | 已实现 |
| `/api/v1/tenants/:id/credentials/:cid` | DELETE | 已实现 |

唯一注意：凭证签发返回 `text/plain`（.creds 内容），前端需用 `responseType: 'text'` 接收。

## 不在范围内

- JetStream 管理（需要后端实现）
- 实时监控（需要后端实现）
- Admin 用户管理（设置页面）
- Dashboard 概览页
- Auth refresh token 机制
- 凭证轮换（rotate）
