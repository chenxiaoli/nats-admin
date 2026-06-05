interface Props {
  status: string;
}

const map: Record<string, { label: string; cls: string }> = {
  active: { label: '活跃', cls: 'bg-green-100 text-green-800' },
  suspended: { label: '已挂起', cls: 'bg-yellow-100 text-yellow-800' },
  deleted: { label: '已删除', cls: 'bg-red-100 text-red-800' },
};

export default function StatusBadge({ status }: Props) {
  const s = map[status] ?? { label: status, cls: 'bg-slate-100 text-slate-600' };
  return <span className={`inline-block rounded-full px-2 py-0.5 text-xs font-medium ${s.cls}`}>{s.label}</span>;
}
