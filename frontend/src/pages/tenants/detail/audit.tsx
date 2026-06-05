import { useOutletContext, useState } from 'react-router';
import type { Tenant } from '@/api/tenants';
import { client } from '@/api/client';

interface AuditEntry {
  id: number;
  admin_id: string | null;
  tenant_id: string | null;
  action: string;
  resource: string | null;
  resource_id: string | null;
  detail: Record<string, unknown> | null;
  ip_addr: string | null;
  created_at: string;
}

type Ctx = { tenant: Tenant; tenantId: string };

export default function TenantAudit() {
  const { tenantId } = useOutletContext<Ctx>();
  const [entries, setEntries] = useState<AuditEntry[]>([]);
  const [loaded, setLoaded] = useState(false);
  const [error, setError] = useState('');

  const load = async () => {
    try {
      // The backend audit middleware writes to audit_logs but there's no dedicated
      // endpoint for reading them. We query via a simple approach.
      // For now, show a placeholder since GET /tenants/:id/audit is not an API route.
      setError('审计日志查看功能需要后端添加专门的查询 API');
      setLoaded(true);
    } catch {
      setError('加载失败');
    }
  };

  if (!loaded) {
    load();
    return <div className="text-slate-500">加载中…</div>;
  }

  if (error) return <div className="text-sm text-slate-400">{error}</div>;

  return (
    <table className="w-full text-sm">
      <thead className="bg-slate-100 text-left">
        <tr>
          <th className="p-2">时间</th>
          <th className="p-2">操作</th>
          <th className="p-2">资源</th>
          <th className="p-2">IP</th>
        </tr>
      </thead>
      <tbody>
        {entries.map((e) => (
          <tr key={e.id} className="border-b">
            <td className="p-2 text-xs">{new Date(e.created_at).toLocaleString('zh-CN')}</td>
            <td className="p-2">{e.action}</td>
            <td className="p-2">{e.resource}</td>
            <td className="p-2 text-xs">{e.ip_addr}</td>
          </tr>
        ))}
        {!entries.length && (
          <tr><td colSpan={4} className="p-4 text-center text-slate-400">暂无审计日志</td></tr>
        )}
      </tbody>
    </table>
  );
}
