import { Routes, Route, Navigate } from 'react-router-dom';
import LoginPage from './pages/login';
import TenantsList from './pages/tenants/list';
import TenantDetail from './pages/tenants/detail';

export default function App() {
  return (
    <Routes>
      <Route path="/login" element={<LoginPage />} />
      <Route path="/tenants" element={<TenantsList />} />
      <Route path="/tenants/:id" element={<TenantDetail />} />
      <Route path="*" element={<Navigate to="/tenants" replace />} />
    </Routes>
  );
}
