import { useQuery } from '@tanstack/react-query';
import { client } from './client';

export interface ServerStats {
  name: string;
  id: string;
  connections: number;
  subscriptions: number;
  in_msgs: number;
  out_msgs: number;
  in_bytes: number;
  out_bytes: number;
  uptime: string;
}

export interface AccountStats {
  account_id: string;
  connections: number;
  total_conns: number;
}

export const useServerStats = () =>
  useQuery({
    queryKey: ['monitor', 'servers'],
    queryFn: async () => {
      const r = await client.get<{ servers: ServerStats[] }>('/monitor/server');
      return r.data.servers ?? [];
    },
    refetchInterval: 10000,
  });

export const useAccountStats = () =>
  useQuery({
    queryKey: ['monitor', 'accounts'],
    queryFn: async () => {
      const r = await client.get<{ accounts: AccountStats[]> }>('/monitor/tenants');
      return r.data.accounts ?? [];
    },
    refetchInterval: 10000,
  });

export function useMonitorWS() {
  const [data, setData] = useState<{ servers: ServerStats[]; accounts: AccountStats[] } | null>(null);
  useEffect(() => {
    const proto = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const ws = new WebSocket(`${proto}//${window.location.host}/api/v1/ws/monitor`);
    ws.onmessage = (e) => setData(JSON.parse(e.data));
    ws.onerror = () => ws.close();
    return () => ws.close();
  }, []);
  return data;
}

import { useState, useEffect } from 'react';
