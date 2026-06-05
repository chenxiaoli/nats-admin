import { useLocation, useNavigate } from 'react-router';

const items = [
  { label: '租户', path: '/tenants' },
  { label: '监控', path: '/monitor', disabled: true },
  { label: '设置', path: '/settings', disabled: true },
];

export default function Sidebar() {
  const location = useLocation();
  const navigate = useNavigate();

  return (
    <aside className="flex w-48 flex-shrink-0 flex-col bg-slate-900 text-white">
      <div className="flex h-12 items-center px-4 text-lg font-bold tracking-wide">
        NATS Admin
      </div>
      <nav className="mt-2 flex-1 space-y-1 px-2">
        {items.map((it) => {
          const active = !it.disabled && location.pathname.startsWith(it.path);
          return (
            <button
              key={it.path}
              disabled={it.disabled}
              onClick={() => navigate(it.path)}
              className={`w-full rounded px-3 py-2 text-left text-sm transition-colors ${
                active
                  ? 'bg-slate-700 font-medium'
                  : it.disabled
                    ? 'cursor-not-allowed text-slate-500'
                    : 'hover:bg-slate-800'
              }`}
            >
              {it.label}
            </button>
          );
        })}
      </nav>
    </aside>
  );
}
