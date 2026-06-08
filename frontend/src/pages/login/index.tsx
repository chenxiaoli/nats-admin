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
