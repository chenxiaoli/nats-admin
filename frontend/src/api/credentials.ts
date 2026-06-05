import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { client } from './client';
import type { Tenant } from './tenants';

export interface Credential {
  id: string;
  name: string;
  user_public_key: string;
  pub_allow: string[];
  sub_allow: string[];
  revoked_at: string | null;
  created_at: string;
}

export const useCredentials = (tenantId: string) =>
  useQuery({
    queryKey: ['credentials', tenantId],
    queryFn: async () => (await client.get<Credential[]>(`/tenants/${tenantId}/credentials`)).data,
    enabled: !!tenantId,
  });

export const useIssueCredential = (tenantId: string) => {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (params: { name: string; pub_allow: string[]; sub_allow: string[] }) => {
      const r = await client.post(`/tenants/${tenantId}/credentials`, params, { responseType: 'text' });
      return r.data as string;
    },
    onSuccess: () => qc.invalidateQueries({ queryKey: ['credentials', tenantId] }),
  });
};

export const useRevokeCredential = (tenantId: string) => {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (credentialId: string) => {
      await client.delete(`/tenants/${tenantId}/credentials/${credentialId}`);
    },
    onSuccess: () => qc.invalidateQueries({ queryKey: ['credentials', tenantId] }),
  });
};

export const useUpdateTenant = () => {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (params: { id: string; data: Record<string, unknown> }) => {
      const r = await client.put<Tenant>(`/tenants/${params.id}`, params.data);
      return r.data;
    },
    onSuccess: () => qc.invalidateQueries({ queryKey: ['tenants'] }),
  });
};

export const useSuspendTenant = () => {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (id: string) => {
      await client.post(`/tenants/${id}/suspend`);
    },
    onSuccess: () => qc.invalidateQueries({ queryKey: ['tenants'] }),
  });
};

export const useActivateTenant = () => {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (id: string) => {
      await client.post(`/tenants/${id}/activate`);
    },
    onSuccess: () => qc.invalidateQueries({ queryKey: ['tenants'] }),
  });
};

export const useDeleteTenant = () => {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (id: string) => {
      await client.delete(`/tenants/${id}`);
    },
    onSuccess: () => qc.invalidateQueries({ queryKey: ['tenants'] }),
  });
};
