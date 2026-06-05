import axios from 'axios';
import { getToken } from '@/lib/auth';

export const client = axios.create({ baseURL: '/api/v1' });

client.interceptors.request.use((cfg) => {
  const tok = getToken();
  if (tok) cfg.headers.Authorization = `Bearer ${tok}`;
  return cfg;
});
