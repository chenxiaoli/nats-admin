import { Outlet, useLocation, useNavigate, useParams } from 'react-router';
import { Link } from 'react-router';
import { useTenant } from '@/api/tenants';

const tabs = [
  { key: '', label: '概览' },
  { key: 'credentials', label: '凭证' },
  { key: 'jetstream', label: 'JetStream' },
  { key: 'audit', label: '审计日志' },
] as const;

export default function TenantDetail() {
  const { id } = useParams<{ id: string }>();
  const location = useLocation();
  const navigate = useNavigate();
  const { data, isLoading, error } = useTenant(id!);

  const currentTab = location.pathname.split('/').pop() === 'credentials'
    ? 'credentials'
    : location.pathname.split('/').pop() === 'jetstream'
      ? 'jetstream'
      : location.pathname.split('/').pop() === 'audit'
        ? 'audit'
        : '';

  if (isLoading) return <div className="text-slate-500">加载中…</div>;
  if (error) return <div className="text-red-600">加载失败</div>;
  if (!data) return null;

  return (
    <div className="space-y-4">
      <div className="flex items-center gap-3">
        <Link to="/tenants" className="text-sm text-slate-500 hover:text-slate-900">← 租户列表</Link>
        <h1 className="text-2xl font-semibold">{data.name}</h1>
      </div>

      <div className="flex gap-1 border-b">
        {tabs.map((t) => (
          <button
            key={t.key}
            onClick={() => navigate(`/tenants/${id}/${t.key ? t.key : ''}`)}
            className={`px-4 py-2 text-sm ${currentTab === t.key ? 'border-b-2 border-slate-900 font-medium' : 'text-slate-500 hover:text-slate-900'}`}
          >
            {t.label}
          </button>
        ))}
      </div>

      <Outlet context={{ tenant: data, tenantId: id! }} />
    </div>
  );
}
