import axios, { AxiosError, InternalAxiosRequestConfig } from 'axios';
import { getToken } from '@/lib/auth';
import { requestReauth } from '@/lib/auth-events';

export const client = axios.create({ baseURL: '/api/v1' });

client.interceptors.request.use((cfg) => {
  const tok = getToken();
  if (tok) cfg.headers.Authorization = `Bearer ${tok}`;
  return cfg;
});

client.interceptors.response.use(undefined, async (err: AxiosError) => {
  if (err.response?.status !== 401) throw err;
  if (err.config?.url === '/auth/login') throw err;
  if (err.response.headers['www-authenticate'] !== 'SessionExpired') throw err;

  let token: string;
  try {
    token = await requestReauth();
  } catch {
    throw err;
  }

  const cfg = err.config!;
  const retryCfg: InternalAxiosRequestConfig = {
    ...cfg,
    headers: { ...cfg.headers, Authorization: `Bearer ${token}` } as InternalAxiosRequestConfig['headers'],
  };
  return client.request(retryCfg);
});
