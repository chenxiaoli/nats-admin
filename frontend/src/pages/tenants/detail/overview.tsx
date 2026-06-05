import { useOutletContext, useNavigate } from 'react-router';
import type { Tenant } from '@/api/tenants';
import { useUpdateTenant, useSuspendTenant, useActivateTenant, useDeleteTenant } from '@/api/credentials';
import { formatBytes, formatNumber } from '@/lib/utils';
import StatusBadge from '@/components/ui/badge';
import ConfirmDialog, { useConfirm } from '@/components/ui/confirm-dialog';
import { useState } from 'react';

type Ctx = { tenant: Tenant; tenantId: string };

export default function TenantOverview() {
  const { tenant: t, tenantId } = useOutletContext<Ctx>();
  const navigate = useNavigate();
  const updateTenant = useUpdateTenant();
  const suspendTenant = useSuspendTenant();
  const activateTenant = useActivateTenant();
  const deleteTenant = useDeleteTenant();
  const { confirmProps, confirm } = useConfirm();

  const [editOpen, setEditOpen] = useState(false);
  const [form, setForm] = useState({
    js_max_memory_storage: t.js_max_memory_storage,
    js_max_disk_storage: t.js_max_disk_storage,
    js_max_streams: t.js_max_streams,
    js_max_consumers: t.js_max_consumers,
    max_connections: t.max_connections,
    max_subscriptions: t.max_subscriptions,
  });

  const handleEdit = () => {
    updateTenant.mutate(
      { id: tenantId, data: form },
      { onSuccess: () => setEditOpen(false) },
    );
  };

  const handleSuspend = async () => {
    const ok = await confirm({ title: '挂起租户', message: `确定要挂起 ${t.name} 吗？所有连接将被断开。`, danger: true });
    if (ok) suspendTenant.mutate(tenantId, { onSuccess: () => navigate('/tenants') });
  };

  const handleActivate = async () => {
    const ok = await confirm({ title: '激活租户', message: `确定要激活 ${t.name} 吗？` });
    if (ok) activateTenant.mutate(tenantId);
  };

  const handleDelete = async () => {
    const ok = await confirm({ title: '删除租户', message: `确定要删除 ${t.name} 吗？此操作不可撤销。`, danger: true, confirmLabel: '删除' });
    if (ok) deleteTenant.mutate(tenantId, { onSuccess: () => navigate('/tenants') });
  };

  const fields = [
    ['Slug', t.slug],
    ['状态', <StatusBadge key="s" status={t.status} />],
    ['Account 公钥', <span key="pk" className="font-mono text-xs">{t.account_public_key}</span>],
    ['JS Memory', formatBytes(t.js_max_memory_storage)],
    ['JS Disk', formatBytes(t.js_max_disk_storage)],
    ['JS Streams', formatNumber(t.js_max_streams)],
    ['JS Consumers', formatNumber(t.js_max_consumers)],
    ['Max Connections', formatNumber(t.max_connections)],
    ['Max Subscriptions', formatNumber(t.max_subscriptions)],
    ['创建时间', new Date(t.created_at).toLocaleString('zh-CN')],
    ['更新时间', new Date(t.updated_at).toLocaleString('zh-CN')],
  ];

  return (
    <div className="space-y-6">
      <div className="rounded-lg border bg-white p-5">
        <div className="grid grid-cols-2 gap-x-8 gap-y-3 text-sm">
          {fields.map(([label, value]) => (
            <div key={String(label)}>
              <span className="text-slate-500">{label}：</span>
              {value}
            </div>
          ))}
        </div>
        <div className="mt-4">
          <span className="text-slate-500 text-sm">Account JWT：</span>
          <pre className="mt-1 max-h-32 overflow-auto rounded bg-slate-100 p-3 text-xs">{t.account_jwt}</pre>
        </div>
      </div>

      <div className="flex gap-2">
        <button onClick={() => setEditOpen(true)} className="rounded-md bg-slate-900 px-4 py-2 text-sm text-white">编辑限制</button>
        {t.status === 'active' && (
          <button onClick={handleSuspend} className="rounded-md border border-yellow-500 px-4 py-2 text-sm text-yellow-700">挂起</button>
        )}
        {t.status === 'suspended' && (
          <button onClick={handleActivate} className="rounded-md bg-green-600 px-4 py-2 text-sm text-white">激活</button>
        )}
        <button onClick={handleDelete} className="rounded-md border border-red-300 px-4 py-2 text-sm text-red-600">删除</button>
      </div>

      {/* Edit Dialog */}
      {editOpen && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40">
          <div className="w-full max-w-md rounded-lg bg-white p-6 shadow-xl">
            <h3 className="text-lg font-semibold">编辑 JetStream 限制</h3>
            <div className="mt-4 grid grid-cols-2 gap-3">
              {([
                ['JS Memory (bytes)', 'js_max_memory_storage'],
                ['JS Disk (bytes)', 'js_max_disk_storage'],
                ['Max Streams', 'js_max_streams'],
                ['Max Consumers', 'js_max_consumers'],
                ['Max Connections', 'max_connections'],
                ['Max Subscriptions', 'max_subscriptions'],
              ] as const).map(([label, key]) => (
                <div key={key}>
                  <label className="text-xs text-slate-500">{label}</label>
                  <input
                    type="number"
                    className="mt-1 w-full rounded border px-3 py-2 text-sm"
                    value={form[key]}
                    onChange={(e) => setForm((f) => ({ ...f, [key]: Number(e.target.value) }))}
                  />
                </div>
              ))}
            </div>
            <div className="mt-4 flex justify-end gap-2">
              <button onClick={() => setEditOpen(false)} className="rounded-md border px-4 py-2 text-sm">取消</button>
              <button onClick={handleEdit} disabled={updateTenant.isPending} className="rounded-md bg-slate-900 px-4 py-2 text-sm text-white disabled:opacity-50">
                {updateTenant.isPending ? '保存中…' : '保存'}
              </button>
            </div>
          </div>
        </div>
      )}

      <ConfirmDialog {...confirmProps} />
    </div>
  );
}
