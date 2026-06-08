import { createBrowserRouter, Navigate } from 'react-router';
import { RouterProvider } from 'react-router/dom';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { loginLoader } from './components/layout/auth-layout';
import AuthLayout, { authLoader } from './components/layout/auth-layout';
import ReauthProvider from '@/components/auth/reauth-provider';
import LoginPage from './pages/login';
import DashboardPage from './pages/dashboard';
import TenantsList from './pages/tenants/list';
import TenantNew from './pages/tenants/new';
import TenantDetail from './pages/tenants/detail';
import TenantOverview from './pages/tenants/detail/overview';
import TenantCredentials from './pages/tenants/detail/credentials';
import TenantJetStream from './pages/tenants/detail/jetstream';
import TenantAudit from './pages/tenants/detail/audit';
import MonitorPage from './pages/monitor';

const router = createBrowserRouter([
  {
    path: '/login',
    element: <LoginPage />,
    loader: loginLoader,
  },
  {
    path: '/',
    element: <AuthLayout />,
    loader: authLoader,
    children: [
      { index: true, element: <DashboardPage /> },
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
              { path: 'jetstream', element: <TenantJetStream /> },
              { path: 'audit', element: <TenantAudit /> },
            ],
          },
        ],
      },
      { path: 'monitor', element: <MonitorPage /> },
    ],
  },
]);

const qc = new QueryClient();

export default function App() {
  return (
    <QueryClientProvider client={qc}>
      <ReauthProvider>
        <RouterProvider router={router} />
      </ReauthProvider>
    </QueryClientProvider>
  );
}
