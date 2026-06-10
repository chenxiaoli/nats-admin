import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { client } from './client';

export interface StreamInfo {
  name: string;
  subjects: string[];
  messages: number;
  bytes: number;
}

export interface KVBucketInfo {
  bucket: string;
  values: number;
  history: number;
}

export interface ConsumerInfo {
  name: string;
  stream: string;
  num_pending: number;
  num_ack_pending: number;
  num_redelivered: number;
  delivered_stream_seq: number;
  ack_floor_stream_seq: number;
  created: string;
}

export const useStreams = (tenantId: string) =>
  useQuery({
    queryKey: ['streams', tenantId],
    queryFn: async () => {
      const r = await client.get<{ streams: StreamInfo[] }>(`/tenants/${tenantId}/jetstream/streams`);
      return r.data.streams ?? [];
    },
    enabled: !!tenantId,
  });

export const useCreateStream = (tenantId: string) => {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (params: { name: string; subjects: string[]; max_bytes?: number; max_msgs?: number }) => {
      const r = await client.post(`/tenants/${tenantId}/jetstream/streams`, params);
      return r.data;
    },
    onSuccess: () => qc.invalidateQueries({ queryKey: ['streams', tenantId] }),
  });
};

export const useDeleteStream = (tenantId: string) => {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (name: string) => {
      await client.delete(`/tenants/${tenantId}/jetstream/streams/${name}`);
    },
    onSuccess: () => qc.invalidateQueries({ queryKey: ['streams', tenantId] }),
  });
};

export const usePurgeStream = (tenantId: string) => {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (name: string) => {
      await client.post(`/tenants/${tenantId}/jetstream/streams/${name}/purge`);
    },
    onSuccess: () => qc.invalidateQueries({ queryKey: ['streams', tenantId] }),
  });
};

export const useKVBuckets = (tenantId: string) =>
  useQuery({
    queryKey: ['kv', tenantId],
    queryFn: async () => {
      const r = await client.get<{ buckets: KVBucketInfo[] }>(`/tenants/${tenantId}/jetstream/kv`);
      return r.data.buckets ?? [];
    },
    enabled: !!tenantId,
  });

export const useCreateKV = (tenantId: string) => {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (params: { bucket: string; history?: number; max_bytes?: number }) => {
      const r = await client.post(`/tenants/${tenantId}/jetstream/kv`, params);
      return r.data;
    },
    onSuccess: () => qc.invalidateQueries({ queryKey: ['kv', tenantId] }),
  });
};

export const useDeleteKV = (tenantId: string) => {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (bucket: string) => {
      await client.delete(`/tenants/${tenantId}/jetstream/kv/${bucket}`);
    },
    onSuccess: () => qc.invalidateQueries({ queryKey: ['kv', tenantId] }),
  });
};

export const useConsumers = (tenantId: string, stream: string) =>
  useQuery({
    queryKey: ['consumers', tenantId, stream],
    queryFn: async () => {
      const r = await client.get<{ consumers: ConsumerInfo[] }>(`/tenants/${tenantId}/jetstream/streams/${stream}/consumers`);
      return r.data.consumers ?? [];
    },
    enabled: !!tenantId && !!stream,
  });
