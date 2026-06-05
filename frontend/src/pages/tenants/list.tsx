import { Link } from 'react-router';
import { useTenants } from '@/api/tenants';

export default function TenantsList() {
  const { data, isLoading, error } = useTenants();

  if (isLoading) return <div className="p-6">加载中…</div>;
  if (error) return <div className="p-6 text-red-600">加载失败</div>;

  return (
    <div className="p-6">
      <div className="mb-4 flex items-center justify-between">
        <h1 className="text-2xl font-semibold">租户</h1>
        <Link to="/tenants/new" className="rounded-md bg-slate-900 px-3 py-2 text-sm text-white">新建</Link>
      </div>
      <table className="w-full text-sm">
        <thead className="bg-slate-100 text-left">
          <tr>
            <th className="p-2">名称</th>
            <th className="p-2">Slug</th>
            <th className="p-2">状态</th>
            <th className="p-2">Account 公钥</th>
            <th className="p-2">JS Streams</th>
            <th className="p-2"></th>
          </tr>
        </thead>
        <tbody>
          {data?.map((t) => (
            <tr key={t.id} className="border-b">
              <td className="p-2">{t.name}</td>
              <td className="p-2 font-mono text-xs">{t.slug}</td>
              <td className="p-2">{t.status}</td>
              <td className="p-2 font-mono text-xs">{t.account_public_key}</td>
              <td className="p-2">{t.js_max_streams}</td>
              <td className="p-2">
                <Link to={`/tenants/${t.id}`} className="text-blue-600 hover:underline">详情</Link>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
