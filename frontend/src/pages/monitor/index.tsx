import { useServerStats, useAccountStats } from '@/api/monitor';
import { formatBytes, formatNumber } from '@/lib/utils';

export default function MonitorPage() {
  const { data: servers, isLoading: sLoading } = useServerStats();
  const { data: accounts, isLoading: aLoading } = useAccountStats();

  if (sLoading || aLoading) return <div className="text-slate-500">加载中…</div>;

  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-semibold">实时监控</h1>

      <div>
        <h2 className="mb-2 text-lg font-semibold">服务器状态</h2>
        {!servers?.length ? (
          <div className="text-sm text-slate-400">暂无数据（需要 System Account 连接）</div>
        ) : (
          <table className="w-full text-sm">
            <thead className="bg-slate-100 text-left">
              <tr>
                <th className="p-2">名称</th>
                <th className="p-2">ID</th>
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
                  <td className="p-2 font-mono text-xs">{s.id.slice(0, 16)}…</td>
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
        )}
      </div>

      <div>
        <h2 className="mb-2 text-lg font-semibold">Account 统计</h2>
        {!accounts?.length ? (
          <div className="text-sm text-slate-400">暂无数据</div>
        ) : (
          <table className="w-full text-sm">
            <thead className="bg-slate-100 text-left">
              <tr>
                <th className="p-2">Account ID</th>
                <th className="p-2">连接数</th>
                <th className="p-2">总连接</th>
              </tr>
            </thead>
            <tbody>
              {accounts.map((a) => (
                <tr key={a.account_id} className="border-b">
                  <td className="p-2 font-mono text-xs">{a.account_id}</td>
                  <td className="p-2">{a.connections}</td>
                  <td className="p-2">{a.total_conns}</td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>
    </div>
  );
}
