import { useParams } from 'react-router-dom';
import { useTenant } from '@/api/tenants';
import { formatBytes, formatNumber } from '@/lib/utils';

export default function TenantDetail() {
  const { id } = useParams<{ id: string }>();
  const { data, isLoading, error } = useTenant(id!);

  if (isLoading) return <div className="p-6">加载中…</div>;
  if (error) return <div className="p-6 text-red-600">加载失败</div>;
  if (!data) return null;

  return (
    <div className="p-6 space-y-6">
      <h1 className="text-2xl font-semibold">{data.name}</h1>
      <div className="grid grid-cols-2 gap-4 text-sm">
        <div><span className="text-slate-500">Slug：</span><span className="font-mono">{data.slug}</span></div>
        <div><span className="text-slate-500">状态：</span>{data.status}</div>
        <div><span className="text-slate-500">Account 公钥：</span><span className="font-mono">{data.account_public_key}</span></div>
        <div><span className="text-slate-500">JS Memory：</span>{formatBytes(data.js_max_memory_storage)}</div>
        <div><span className="text-slate-500">JS Disk：</span>{formatBytes(data.js_max_disk_storage)}</div>
        <div><span className="text-slate-500">JS Streams：</span>{formatNumber(data.js_max_streams)}</div>
        <div><span className="text-slate-500">JS Consumers：</span>{formatNumber(data.js_max_consumers)}</div>
        <div><span className="text-slate-500">Max Connections：</span>{formatNumber(data.max_connections)}</div>
        <div><span className="text-slate-500">Max Subscriptions：</span>{formatNumber(data.max_subscriptions)}</div>
      </div>
      <div>
        <h2 className="text-lg font-semibold">Account JWT</h2>
        <pre className="overflow-auto rounded bg-slate-100 p-3 text-xs">{data.account_jwt}</pre>
      </div>
    </div>
  );
}
