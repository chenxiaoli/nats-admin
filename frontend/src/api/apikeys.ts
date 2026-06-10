import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { client } from './client';

export interface APIKey {
  id: string;
  name: string;
  key_prefix: string;
  last_used_at: string | null;
  created_at: string;
  revoked_at: string | null;
}

export interface CreateAPIKeyResp extends APIKey {
  key: string;
}

export const useAPIKeys = () =>
  useQuery({
    queryKey: ['api-keys'],
    queryFn: async () => (await client.get<APIKey[]>('/settings/api-keys')).data,
  });

export const useCreateAPIKey = () => {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (name: string) =>
      (await client.post<CreateAPIKeyResp>('/settings/api-keys', { name })).data,
    onSuccess: () => qc.invalidateQueries({ queryKey: ['api-keys'] }),
  });
};

export const useRevokeAPIKey = () => {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => client.delete(`/settings/api-keys/${id}`),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['api-keys'] }),
  });
};
