import { useServerStats, useAccountStats } from '@/api/monitor';
import { formatBytes, formatNumber } from '@/lib/utils';
import { useTenants } from '@/api/tenants';

export default function DashboardPage() {
  const { data: tenants } = useTenants();
  const { data: servers } = useServerStats();
  const { data: accounts } = useAccountStats();

  const activeTenants = tenants?.filter((t) => t.status === 'active').length ?? 0;

  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-semibold">概览</h1>

      <div className="grid grid-cols-4 gap-4">
        <Card label="租户数" value={String(tenants?.length ?? 0)} sub={`活跃 ${activeTenants}`} />
        <Card label="NATS 服务器" value={String(servers?.length ?? 0)} />
        <Card label="总连接数" value={formatNumber(servers?.reduce((s, v) => s + v.connections, 0) ?? 0)} />
        <Card label="Account 数" value={String(accounts?.length ?? 0)} />
      </div>

      {servers && servers.length > 0 && (
        <div>
          <h2 className="mb-2 text-lg font-semibold">服务器</h2>
          <table className="w-full text-sm">
            <thead className="bg-slate-100 text-left">
              <tr>
                <th className="p-2">名称</th>
                <th className="p-2">连接数</th>
                <th className="p-2">订阅数</th>
                <th className="p-2">入站消息</th>
                <th className="p-2">出站消息</th>
                <th className="p-2">入站字节</th>
                <th className="p-2">出站字节</th>
                <th className="p-2">运行时间</th>
              </tr>
            </thead>
            <tbody>
              {servers.map((s) => (
                <tr key={s.id} className="border-b">
                  <td className="p-2">{s.name}</td>
                  <td className="p-2">{s.connections}</td>
                  <td className="p-2">{formatNumber(s.subscriptions)}</td>
                  <td className="p-2">{formatNumber(s.in_msgs)}</td>
                  <td className="p-2">{formatNumber(s.out_msgs)}</td>
                  <td className="p-2">{formatBytes(s.in_bytes)}</td>
                  <td className="p-2">{formatBytes(s.out_bytes)}</td>
                  <td className="p-2 text-xs">{s.uptime}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}

function Card({ label, value, sub }: { label: string; value: string; sub?: string }) {
  return (
    <div className="rounded-lg border bg-white p-4">
      <div className="text-sm text-slate-500">{label}</div>
      <div className="mt-1 text-2xl font-bold">{value}</div>
      {sub && <div className="mt-1 text-xs text-slate-400">{sub}</div>}
    </div>
  );
}
