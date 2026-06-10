import { useState } from 'react';
import { useAPIKeys, useRevokeAPIKey } from '@/api/apikeys';
import CreateKeyDialog from '@/components/apikey/create-key-dialog';

function fmtDate(iso: string | null): string {
  if (!iso) return '—';
  return new Date(iso).toLocaleString('zh-CN');
}

export default function APIKeysPage() {
  const { data, isLoading, error } = useAPIKeys();
  const revoke = useRevokeAPIKey();
  const [dialogOpen, setDialogOpen] = useState(false);
  const [revokeTarget, setRevokeTarget] = useState<string | null>(null);
  const [revokeErr, setRevokeErr] = useState<string | null>(null);

  return (
    <div className="p-6">
      <div className="mb-4 flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-semibold">API Keys</h1>
          <p className="text-sm text-slate-600">用于后端服务访问 API，权限等同于你的账户角色</p>
        </div>
        <button
          onClick={() => setDialogOpen(true)}
          className="rounded-md bg-slate-900 px-3 py-2 text-sm text-white"
        >
          创建 Key
        </button>
      </div>

      {isLoading && <div>加载中…</div>}
      {error && <div className="text-red-600">加载失败</div>}

      {data && data.length === 0 && (
        <div className="rounded border border-dashed border-slate-300 p-8 text-center text-slate-500">
          还没有 API Key
        </div>
      )}

      {data && data.length > 0 && (
        <table className="w-full text-sm">
          <thead className="bg-slate-100 text-left">
            <tr>
              <th className="p-2">名称</th>
              <th className="p-2">前缀</th>
              <th className="p-2">创建时间</th>
              <th className="p-2">最后使用</th>
              <th className="p-2">状态</th>
              <th className="p-2"></th>
            </tr>
          </thead>
          <tbody>
            {data.map((k) => (
              <tr key={k.id} className="border-b">
                <td className="p-2 font-medium">{k.name}</td>
                <td className="p-2 font-mono text-xs">{k.key_prefix}…</td>
                <td className="p-2 text-slate-600">{fmtDate(k.created_at)}</td>
                <td className="p-2 text-slate-600">{fmtDate(k.last_used_at)}</td>
                <td className="p-2">
                  {k.revoked_at ? (
                    <span className="rounded bg-red-100 px-2 py-0.5 text-xs text-red-800">已吊销</span>
                  ) : (
                    <span className="rounded bg-green-100 px-2 py-0.5 text-xs text-green-800">活跃</span>
                  )}
                </td>
                <td className="p-2 text-right">
                  {!k.revoked_at && (
                    <button
                      onClick={() => setRevokeTarget(k.id)}
                      className="text-xs text-red-600 hover:underline"
                    >
                      吊销
                    </button>
                  )}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      )}

      <CreateKeyDialog open={dialogOpen} onOpenChange={setDialogOpen} />

      {revokeTarget && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40">
          <div className="rounded-lg bg-white p-6 shadow-lg">
            <h3 className="mb-2 font-semibold">确认吊销</h3>
            <p className="mb-4 text-sm text-slate-600">吊销后此 key 立即失效，无法恢复。</p>
            {revokeErr && <p className="mb-2 text-xs text-red-600">{revokeErr}</p>}
            <div className="flex justify-end gap-2">
              <button
                onClick={() => setRevokeTarget(null)}
                className="rounded border border-slate-300 px-3 py-1.5 text-sm"
              >
                取消
              </button>
              <button
                onClick={async () => {
                  if (!revokeTarget) return;
                  setRevokeErr(null);
                  try {
                    await revoke.mutateAsync(revokeTarget);
                    setRevokeTarget(null);
                  } catch (e: any) {
                    setRevokeErr(e?.response?.data?.message ?? e?.message ?? '吊销失败');
                  }
                }}
                disabled={revoke.isPending}
                className="rounded bg-red-600 px-3 py-1.5 text-sm text-white disabled:opacity-50"
              >
                {revoke.isPending ? '吊销中…' : '确认吊销'}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
