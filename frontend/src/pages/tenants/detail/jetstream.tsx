import { useOutletContext } from 'react-router';
import { useState } from 'react';
import { useStreams, useCreateStream, useDeleteStream, usePurgeStream, useKVBuckets, useCreateKV, useDeleteKV, useConsumers } from '@/api/jetstream';
import type { StreamInfo } from '@/api/jetstream';
import ConfirmDialog, { useConfirm } from '@/components/ui/confirm-dialog';
import { formatBytes, formatNumber } from '@/lib/utils';
import type { Tenant } from '@/api/tenants';

type Ctx = { tenant: Tenant; tenantId: string };

export default function TenantJetStream() {
  const { tenantId } = useOutletContext<Ctx>();
  const [tab, setTab] = useState<'streams' | 'kv'>('streams');

  return (
    <div className="space-y-4">
      <div className="flex gap-2 border-b">
        <button onClick={() => setTab('streams')} className={`px-4 py-2 text-sm ${tab === 'streams' ? 'border-b-2 border-slate-900 font-medium' : 'text-slate-500'}`}>Streams</button>
        <button onClick={() => setTab('kv')} className={`px-4 py-2 text-sm ${tab === 'kv' ? 'border-b-2 border-slate-900 font-medium' : 'text-slate-500'}`}>KV Buckets</button>
      </div>
      {tab === 'streams' ? <StreamsTab tenantId={tenantId} /> : <KVTab tenantId={tenantId} />}
    </div>
  );
}

function StreamsTab({ tenantId }: { tenantId: string }) {
  const { data, isLoading } = useStreams(tenantId);
  const create = useCreateStream(tenantId);
  const deleteStream = useDeleteStream(tenantId);
  const purgeStream = usePurgeStream(tenantId);
  const { confirmProps, confirm } = useConfirm();
  const [showCreate, setShowCreate] = useState(false);
  const [expanded, setExpanded] = useState<string | null>(null);
  const [form, setForm] = useState({ name: '', subjects: '', max_bytes: -1, max_msgs: -1 });

  const handleCreate = () => {
    create.mutate({
      name: form.name,
      subjects: form.subjects.split(',').map((s) => s.trim()).filter(Boolean),
      max_bytes: form.max_bytes,
      max_msgs: form.max_msgs,
    }, { onSuccess: () => { setShowCreate(false); setForm({ name: '', subjects: '', max_bytes: -1, max_msgs: -1 }); } });
  };

  const handleDelete = async (name: string) => {
    const ok = await confirm({ title: '删除 Stream', message: `确定要删除 Stream "${name}" 吗？`, danger: true });
    if (ok) deleteStream.mutate(name);
  };

  const handlePurge = async (name: string) => {
    const ok = await confirm({ title: '清空 Stream', message: `确定要清空 Stream "${name}" 的所有消息吗？`, danger: true });
    if (ok) purgeStream.mutate(name);
  };

  if (isLoading) return <div className="text-slate-500">加载中…</div>;

  return (
    <div className="space-y-3">
      <div className="flex justify-end">
        <button onClick={() => setShowCreate(true)} className="rounded-md bg-slate-900 px-4 py-2 text-sm text-white">新建 Stream</button>
      </div>
      <table className="w-full text-sm">
        <thead className="bg-slate-100 text-left">
          <tr>
            <th className="p-2 w-6" />
            <th className="p-2">名称</th>
            <th className="p-2">Subjects</th>
            <th className="p-2">消息数</th>
            <th className="p-2">大小</th>
            <th className="p-2" />
          </tr>
        </thead>
        <tbody>
          {data?.map((s) => (
            <StreamRow
              key={s.name}
              tenantId={tenantId}
              stream={s}
              expanded={expanded === s.name}
              onToggle={() => setExpanded(expanded === s.name ? null : s.name)}
              onPurge={handlePurge}
              onDelete={handleDelete}
            />
          ))}
          {!data?.length && <tr><td colSpan={6} className="p-4 text-center text-slate-400">暂无 Stream</td></tr>}
        </tbody>
      </table>

      {showCreate && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40">
          <div className="w-full max-w-md rounded-lg bg-white p-6 shadow-xl">
            <h3 className="text-lg font-semibold">新建 Stream</h3>
            <div className="mt-4 space-y-3">
              <div>
                <label className="text-sm text-slate-600">名称</label>
                <input className="mt-1 w-full rounded border px-3 py-2 text-sm" value={form.name} onChange={(e) => setForm((f) => ({ ...f, name: e.target.value }))} placeholder="ORDERS" />
              </div>
              <div>
                <label className="text-sm text-slate-600">Subjects（逗号分隔）</label>
                <input className="mt-1 w-full rounded border px-3 py-2 text-sm" value={form.subjects} onChange={(e) => setForm((f) => ({ ...f, subjects: e.target.value }))} placeholder="orders.>, events.>" />
              </div>
              <div className="grid grid-cols-2 gap-3">
                <div>
                  <label className="text-sm text-slate-600">Max Bytes (-1=不限)</label>
                  <input type="number" className="mt-1 w-full rounded border px-3 py-2 text-sm" value={form.max_bytes} onChange={(e) => setForm((f) => ({ ...f, max_bytes: Number(e.target.value) }))} />
                </div>
                <div>
                  <label className="text-sm text-slate-600">Max Msgs (-1=不限)</label>
                  <input type="number" className="mt-1 w-full rounded border px-3 py-2 text-sm" value={form.max_msgs} onChange={(e) => setForm((f) => ({ ...f, max_msgs: Number(e.target.value) }))} />
                </div>
              </div>
            </div>
            <div className="mt-4 flex justify-end gap-2">
              <button onClick={() => setShowCreate(false)} className="rounded-md border px-4 py-2 text-sm">取消</button>
              <button onClick={handleCreate} disabled={create.isPending} className="rounded-md bg-slate-900 px-4 py-2 text-sm text-white disabled:opacity-50">{create.isPending ? '创建中…' : '创建'}</button>
            </div>
          </div>
        </div>
      )}
      <ConfirmDialog {...confirmProps} />
    </div>
  );
}

function StreamRow({ tenantId, stream, expanded, onToggle, onPurge, onDelete }: {
  tenantId: string;
  stream: StreamInfo;
  expanded: boolean;
  onToggle: () => void;
  onPurge: (name: string) => void;
  onDelete: (name: string) => void;
}) {
  return (
    <>
      <tr className="border-b">
        <td className="p-2 text-center">
          <button onClick={onToggle} className="text-slate-400 hover:text-slate-600 text-xs">
            {expanded ? '▼' : '▶'}
          </button>
        </td>
        <td className="p-2 font-mono text-xs">{stream.name}</td>
        <td className="p-2 text-xs">{stream.subjects?.join(', ') || '—'}</td>
        <td className="p-2">{formatNumber(stream.messages)}</td>
        <td className="p-2">{formatBytes(stream.bytes)}</td>
        <td className="p-2 space-x-2">
          <button onClick={() => onPurge(stream.name)} className="text-sm text-yellow-600 hover:underline">清空</button>
          <button onClick={() => onDelete(stream.name)} className="text-sm text-red-600 hover:underline">删除</button>
        </td>
      </tr>
      {expanded && (
        <tr>
          <td colSpan={6} className="bg-slate-50 px-6 py-3">
            <ConsumersPanel tenantId={tenantId} stream={stream.name} />
          </td>
        </tr>
      )}
    </>
  );
}

function ConsumersPanel({ tenantId, stream }: { tenantId: string; stream: string }) {
  const { data, isLoading } = useConsumers(tenantId, stream);

  if (isLoading) return <div className="text-xs text-slate-400">加载 Consumer…</div>;
  if (!data?.length) return <div className="text-xs text-slate-400">该 Stream 暂无 Consumer</div>;

  return (
    <table className="w-full text-xs">
      <thead className="text-left text-slate-500">
        <tr>
          <th className="p-1.5">Consumer</th>
          <th className="p-1.5">未消费</th>
          <th className="p-1.5">待 Ack</th>
          <th className="p-1.5">已投递到</th>
          <th className="p-1.5">Ack 位</th>
          <th className="p-1.5">重投递</th>
        </tr>
      </thead>
      <tbody>
        {data.map((c) => (
          <tr key={c.name} className="border-t border-slate-200">
            <td className="p-1.5 font-mono">{c.name}</td>
            <td className="p-1.5 font-semibold text-orange-600">{formatNumber(c.num_pending)}</td>
            <td className="p-1.5">{c.num_ack_pending}</td>
            <td className="p-1.5">{formatNumber(c.delivered_stream_seq)}</td>
            <td className="p-1.5">{formatNumber(c.ack_floor_stream_seq)}</td>
            <td className="p-1.5">{c.num_redelivered}</td>
          </tr>
        ))}
      </tbody>
    </table>
  );
}

function KVTab({ tenantId }: { tenantId: string }) {
  const { data, isLoading } = useKVBuckets(tenantId);
  const create = useCreateKV(tenantId);
  const deleteKV = useDeleteKV(tenantId);
  const { confirmProps, confirm } = useConfirm();
  const [showCreate, setShowCreate] = useState(false);
  const [form, setForm] = useState({ bucket: '', history: 1, max_bytes: -1 });

  const handleCreate = () => {
    create.mutate(form, { onSuccess: () => { setShowCreate(false); setForm({ bucket: '', history: 1, max_bytes: -1 }); } });
  };

  const handleDelete = async (bucket: string) => {
    const ok = await confirm({ title: '删除 KV Bucket', message: `确定要删除 Bucket "${bucket}" 吗？`, danger: true });
    if (ok) deleteKV.mutate(bucket);
  };

  if (isLoading) return <div className="text-slate-500">加载中…</div>;

  return (
    <div className="space-y-3">
      <div className="flex justify-end">
        <button onClick={() => setShowCreate(true)} className="rounded-md bg-slate-900 px-4 py-2 text-sm text-white">新建 Bucket</button>
      </div>
      <table className="w-full text-sm">
        <thead className="bg-slate-100 text-left">
          <tr>
            <th className="p-2">Bucket</th>
            <th className="p-2">Values</th>
            <th className="p-2">History</th>
            <th className="p-2" />
          </tr>
        </thead>
        <tbody>
          {data?.map((b) => (
            <tr key={b.bucket} className="border-b">
              <td className="p-2 font-mono text-xs">{b.bucket}</td>
              <td className="p-2">{b.values}</td>
              <td className="p-2">{b.history}</td>
              <td className="p-2"><button onClick={() => handleDelete(b.bucket)} className="text-sm text-red-600 hover:underline">删除</button></td>
            </tr>
          ))}
          {!data?.length && <tr><td colSpan={4} className="p-4 text-center text-slate-400">暂无 KV Bucket</td></tr>}
        </tbody>
      </table>

      {showCreate && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40">
          <div className="w-full max-w-sm rounded-lg bg-white p-6 shadow-xl">
            <h3 className="text-lg font-semibold">新建 KV Bucket</h3>
            <div className="mt-4 space-y-3">
              <div>
                <label className="text-sm text-slate-600">Bucket 名称</label>
                <input className="mt-1 w-full rounded border px-3 py-2 text-sm" value={form.bucket} onChange={(e) => setForm((f) => ({ ...f, bucket: e.target.value }))} />
              </div>
              <div className="grid grid-cols-2 gap-3">
                <div>
                  <label className="text-sm text-slate-600">History</label>
                  <input type="number" className="mt-1 w-full rounded border px-3 py-2 text-sm" value={form.history} onChange={(e) => setForm((f) => ({ ...f, history: Number(e.target.value) }))} />
                </div>
                <div>
                  <label className="text-sm text-slate-600">Max Bytes (-1=不限)</label>
                  <input type="number" className="mt-1 w-full rounded border px-3 py-2 text-sm" value={form.max_bytes} onChange={(e) => setForm((f) => ({ ...f, max_bytes: Number(e.target.value) }))} />
                </div>
              </div>
            </div>
            <div className="mt-4 flex justify-end gap-2">
              <button onClick={() => setShowCreate(false)} className="rounded-md border px-4 py-2 text-sm">取消</button>
              <button onClick={handleCreate} disabled={create.isPending} className="rounded-md bg-slate-900 px-4 py-2 text-sm text-white disabled:opacity-50">{create.isPending ? '创建中…' : '创建'}</button>
            </div>
          </div>
        </div>
      )}
      <ConfirmDialog {...confirmProps} />
    </div>
  );
}
