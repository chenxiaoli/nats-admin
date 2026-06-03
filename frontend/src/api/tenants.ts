import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { client } from './client';

export interface Tenant {
  id: string;
  name: string;
  slug: string;
  account_public_key: string;
  account_jwt: string;
  js_max_memory_storage: number;
  js_max_disk_storage: number;
  js_max_streams: number;
  js_max_consumers: number;
  max_connections: number;
  max_subscriptions: number;
  status: 'active' | 'suspended' | 'deleted';
  created_at: string;
  updated_at: string;
}

export interface CreateTenantReq {
  name: string;
  slug: string;
  js_max_memory_storage: number;
  js_max_disk_storage: number;
  js_max_streams: number;
  js_max_consumers: number;
  max_connections: number;
  max_subscriptions: number;
}

export const useTenants = () =>
  useQuery({ queryKey: ['tenants'], queryFn: async () => (await client.get<Tenant[]>('/tenants')).data });

export const useTenant = (id: string) =>
  useQuery({
    queryKey: ['tenant', id],
    queryFn: async () => (await client.get<Tenant>(`/tenants/${id}`)).data,
    enabled: !!id,
  });

export const useCreateTenant = () => {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (req: CreateTenantReq) => client.post('/tenants', req),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['tenants'] }),
  });
};
