import { useEffect, useState } from 'react';
import { client } from '@/api/client';
import { setToken, clearToken } from '@/lib/auth';

interface Props {
  open: boolean;
  onSolved: (token: string) => void;
  onCancelled: () => void;
}

const MAX_ATTEMPTS = 3;

export default function ReauthModal({ open, onSolved, onCancelled }: Props) {
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [err, setErr] = useState('');
  const [attempts, setAttempts] = useState(0);
  const [submitting, setSubmitting] = useState(false);

  useEffect(() => {
    if (open) {
      setEmail('');
      setPassword('');
      setErr('');
      setAttempts(0);
      setSubmitting(false);
    }
  }, [open]);

  if (!open) return null;

  const submit = async () => {
    setSubmitting(true);
    setErr('');
    try {
      const r = await client.post('/auth/login', { email, password });
      setToken(r.data.token);
      onSolved(r.data.token);
    } catch {
      const next = attempts + 1;
      setAttempts(next);
      if (next >= MAX_ATTEMPTS) {
        clearToken();
        window.location.href = '/login?reason=session_expired';
        return;
      }
      setErr(`登录失败（${next}/${MAX_ATTEMPTS}）`);
      setSubmitting(false);
    }
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40">
      <div className="w-full max-w-sm rounded-lg bg-white p-6 shadow-xl">
        <h3 className="text-lg font-semibold">会话已过期</h3>
        <p className="mt-2 text-sm text-slate-600">请重新登录以继续操作。</p>
        <label className="mt-4 mb-1 block text-sm">Email</label>
        <input
          autoFocus
          className="mb-3 w-full rounded-md border border-slate-300 px-3 py-2 text-sm"
          value={email}
          onChange={(e) => setEmail(e.target.value)}
          disabled={submitting}
        />
        <label className="mb-1 block text-sm">Password</label>
        <input
          type="password"
          className="mb-3 w-full rounded-md border border-slate-300 px-3 py-2 text-sm"
          value={password}
          onChange={(e) => setPassword(e.target.value)}
          onKeyDown={(e) => e.key === 'Enter' && !submitting && submit()}
          disabled={submitting}
        />
        {err && <div className="mb-2 text-sm text-red-600">{err}</div>}
        <div className="mt-4 flex justify-end gap-2">
          <button
            onClick={onCancelled}
            disabled={submitting}
            className="rounded-md border px-4 py-2 text-sm disabled:opacity-50"
          >
            取消
          </button>
          <button
            onClick={submit}
            disabled={submitting || !email || !password}
            className="rounded-md bg-slate-900 px-4 py-2 text-sm text-white hover:bg-slate-700 disabled:opacity-50"
          >
            登录
          </button>
        </div>
      </div>
    </div>
  );
}
