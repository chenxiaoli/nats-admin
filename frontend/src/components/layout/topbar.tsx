import { useNavigate } from 'react-router';
import { clearToken } from '@/lib/auth';

export default function TopBar() {
  const navigate = useNavigate();

  const handleLogout = () => {
    clearToken();
    navigate('/login');
  };

  return (
    <header className="flex h-12 items-center justify-between border-b bg-white px-6">
      <div />
      <button
        onClick={handleLogout}
        className="text-sm text-slate-500 hover:text-slate-900"
      >
        退出
      </button>
    </header>
  );
}
