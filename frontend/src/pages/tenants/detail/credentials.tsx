import { useOutletContext, useState } from 'react-router';
import type { Tenant } from '@/api/tenants';
import { useCredentials, useIssueCredential, useRevokeCredential } from '@/api/credentials';
import CredsDialog from '@/components/credential/creds-dialog';
import ConfirmDialog, { useConfirm } from '@/components/ui/confirm-dialog';

type Ctx = { tenant: Tenant; tenantId: string };

export default function TenantCredentials() {
  const { tenantId } = useOutletContext<Ctx>();
  const { data, isLoading } = useCredentials(tenantId);
  const issue = useIssueCredential(tenantId);
  const revoke = useRevokeCredential(tenantId);
  const { confirmProps, confirm } = useConfirm();

  const [issueOpen, setIssueOpen] = useState(false);
  const [credsOpen, setCredsOpen] = useState(false);
  const [creds, setCreds] = useState('');
  const [form, setForm] = useState({ name: '', pub_allow: '', sub_allow: '' });

  const handleIssue = () => {
    issue.mutate(
      {
        name: form.name,
        pub_allow: form.pub_allow.split('\n').filter(Boolean),
        sub_allow: form.sub_allow.split('\n').filter(Boolean),
      },
      {
        onSuccess: (credsText) => {
          setCreds(credsText);
          setCredsOpen(true);
          setIssueOpen(false);
          setForm({ name: '', pub_allow: '', sub_allow: '' });
        },
      },
    );
  };

  const handleRevoke = async (credId: string, credName: string) => {
    const ok = await confirm({ title: '吊销凭证', message: `确定要吊销凭证 "${credName}" 吗？吊销后客户端将立即断开。`, danger: true, confirmLabel: '吊销' });
    if (ok) revoke.mutate(credId);
  };

  if (isLoading) return <div className="text-slate-500">加载中…</div>;

  return (
    <div className="space-y-4">
      <div className="flex justify-end">
        <button onClick={() => setIssueOpen(true)} className="rounded-md bg-slate-900 px-4 py-2 text-sm text-white">签发凭证</button>
      </div>

      <table className="w-full text-sm">
        <thead className="bg-slate-100 text-left">
          <tr>
            <th className="p-2">名称</th>
            <th className="p-2">公钥</th>
            <th className="p-2">Pub 权限</th>
            <th className="p-2">Sub 权限</th>
            <th className="p-2">状态</th>
            <th className="p-2">创建时间</th>
            <th className="p-2" />
          </tr>
        </thead>
        <tbody>
          {data?.map((c) => (
            <tr key={c.id} className="border-b">
              <td className="p-2">{c.name}</td>
              <td className="p-2 font-mono text-xs">{c.user_public_key.slice(0, 20)}…</td>
              <td className="p-2 text-xs">{c.pub_allow?.join(', ') || '—'}</td>
              <td className="p-2 text-xs">{c.sub_allow?.join(', ') || '—'}</td>
              <td className="p-2">{c.revoked_at ? <span className="text-red-600">已吊销</span> : '活跃'}</td>
              <td className="p-2 text-xs">{new Date(c.created_at).toLocaleString('zh-CN')}</td>
              <td className="p-2">
                {!c.revoked_at && (
                  <button onClick={() => handleRevoke(c.id, c.name)} className="text-sm text-red-600 hover:underline">吊销</button>
                )}
              </td>
            </tr>
          ))}
          {!data?.length && (
            <tr><td colSpan={7} className="p-4 text-center text-slate-400">暂无凭证</td></tr>
          )}
        </tbody>
      </table>

      {/* Issue Dialog */}
      {issueOpen && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40">
          <div className="w-full max-w-md rounded-lg bg-white p-6 shadow-xl">
            <h3 className="text-lg font-semibold">签发凭证</h3>
            <div className="mt-4 space-y-3">
              <div>
                <label className="text-sm text-slate-600">名称</label>
                <input className="mt-1 w-full rounded border px-3 py-2 text-sm" value={form.name} onChange={(e) => setForm((f) => ({ ...f, name: e.target.value }))} />
              </div>
              <div>
                <label className="text-sm text-slate-600">Pub 权限（每行一个 subject）</label>
                <textarea className="mt-1 w-full rounded border px-3 py-2 text-sm" rows={3} value={form.pub_allow} onChange={(e) => setForm((f) => ({ ...f, pub_allow: e.target.value }))} placeholder="test.>" />
              </div>
              <div>
                <label className="text-sm text-slate-600">Sub 权限（每行一个 subject）</label>
                <textarea className="mt-1 w-full rounded border px-3 py-2 text-sm" rows={3} value={form.sub_allow} onChange={(e) => setForm((f) => ({ ...f, sub_allow: e.target.value }))} placeholder="_INBOX.>" />
              </div>
            </div>
            <div className="mt-4 flex justify-end gap-2">
              <button onClick={() => setIssueOpen(false)} className="rounded-md border px-4 py-2 text-sm">取消</button>
              <button onClick={handleIssue} disabled={issue.isPending} className="rounded-md bg-slate-900 px-4 py-2 text-sm text-white disabled:opacity-50">
                {issue.isPending ? '签发中…' : '签发'}
              </button>
            </div>
          </div>
        </div>
      )}

      <CredsDialog open={credsOpen} creds={creds} onClose={() => setCredsOpen(false)} />
      <ConfirmDialog {...confirmProps} />
    </div>
  );
}
