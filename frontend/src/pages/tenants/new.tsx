import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { useCreateTenant, type CreateTenantReq } from '@/api/tenants';

const defaults: CreateTenantReq = {
  name: '',
  slug: '',
  js_max_memory_storage: -1,
  js_max_disk_storage: -1,
  js_max_streams: -1,
  js_max_consumers: -1,
  max_connections: -1,
  max_subscriptions: -1,
};

export default function TenantNew() {
  const navigate = useNavigate();
  const create = useCreateTenant();
  const [form, setForm] = useState<CreateTenantReq>(defaults);

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    create.mutate(form, {
      onSuccess: () => navigate('/tenants'),
    });
  };

  const set = (k: keyof CreateTenantReq, v: string | number) =>
    setForm((f) => ({ ...f, [k]: v }));

  return (
    <div className="p-6">
      <h1 className="mb-4 text-2xl font-semibold">新建租户</h1>
      <form onSubmit={handleSubmit} className="max-w-lg space-y-4 text-sm">
        <div>
          <label className="block text-slate-600">名称</label>
          <input
            className="mt-1 w-full rounded border px-3 py-2"
            value={form.name}
            onChange={(e) => set('name', e.target.value)}
            required
          />
        </div>
        <div>
          <label className="block text-slate-600">Slug</label>
          <input
            className="mt-1 w-full rounded border px-3 py-2 font-mono"
            value={form.slug}
            onChange={(e) => set('slug', e.target.value)}
            required
          />
        </div>
        <div className="grid grid-cols-2 gap-4">
          <div>
            <label className="block text-slate-600">JS Memory (bytes, -1=不限)</label>
            <input
              type="number"
              className="mt-1 w-full rounded border px-3 py-2"
              value={form.js_max_memory_storage}
              onChange={(e) => set('js_max_memory_storage', Number(e.target.value))}
            />
          </div>
          <div>
            <label className="block text-slate-600">JS Disk (bytes, -1=不限)</label>
            <input
              type="number"
              className="mt-1 w-full rounded border px-3 py-2"
              value={form.js_max_disk_storage}
              onChange={(e) => set('js_max_disk_storage', Number(e.target.value))}
            />
          </div>
          <div>
            <label className="block text-slate-600">Max Streams (-1=不限)</label>
            <input
              type="number"
              className="mt-1 w-full rounded border px-3 py-2"
              value={form.js_max_streams}
              onChange={(e) => set('js_max_streams', Number(e.target.value))}
            />
          </div>
          <div>
            <label className="block text-slate-600">Max Consumers (-1=不限)</label>
            <input
              type="number"
              className="mt-1 w-full rounded border px-3 py-2"
              value={form.js_max_consumers}
              onChange={(e) => set('js_max_consumers', Number(e.target.value))}
            />
          </div>
          <div>
            <label className="block text-slate-600">Max Connections (-1=不限)</label>
            <input
              type="number"
              className="mt-1 w-full rounded border px-3 py-2"
              value={form.max_connections}
              onChange={(e) => set('max_connections', Number(e.target.value))}
            />
          </div>
          <div>
            <label className="block text-slate-600">Max Subscriptions (-1=不限)</label>
            <input
              type="number"
              className="mt-1 w-full rounded border px-3 py-2"
              value={form.max_subscriptions}
              onChange={(e) => set('max_subscriptions', Number(e.target.value))}
            />
          </div>
        </div>
        {create.error && (
          <div className="text-red-600">创建失败：{String(create.error)}</div>
        )}
        <div className="flex gap-3 pt-2">
          <button
            type="submit"
            disabled={create.isPending}
            className="rounded-md bg-slate-900 px-4 py-2 text-white disabled:opacity-50"
          >
            {create.isPending ? '创建中…' : '创建'}
          </button>
          <button
            type="button"
            onClick={() => navigate('/tenants')}
            className="rounded-md border px-4 py-2"
          >
            取消
          </button>
        </div>
      </form>
    </div>
  );
}
