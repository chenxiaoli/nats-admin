import axios from 'axios';

export const client = axios.create({ baseURL: '/api/v1' });

client.interceptors.request.use((cfg) => {
  const tok = localStorage.getItem('admin_token');
  if (tok) cfg.headers.Authorization = `Bearer ${tok}`;
  return cfg;
});
