import { Navigate, Outlet, redirect } from 'react-router';
import { isAuthenticated } from '@/lib/auth';
import Sidebar from '@/components/layout/sidebar';
import TopBar from '@/components/layout/topbar';

export function authLoader() {
  if (!isAuthenticated()) return redirect('/login');
  return null;
}

export function loginLoader() {
  if (isAuthenticated()) return redirect('/tenants');
  return null;
}

export default function AuthLayout() {
  return (
    <div className="flex h-screen">
      <Sidebar />
      <div className="flex flex-1 flex-col overflow-hidden">
        <TopBar />
        <main className="flex-1 overflow-auto bg-slate-50 p-6">
          <Outlet />
        </main>
      </div>
    </div>
  );
}
