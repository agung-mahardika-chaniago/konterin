import { useCallback, useEffect, useRef, useState } from 'react';

import { ApiError } from '@/lib/api-client';
import { useAlertStore } from '@/stores/alert-store';

import { getAlerts, getCharts, getSummary, getTechnicians } from './api';
import { rangeFor } from './format';
import type { DashboardData, DashboardRange } from './types';

type Status = 'idle' | 'loading' | 'ready' | 'error';

export function useDashboardData(outletId: string | null, range: DashboardRange) {
  const [data, setData] = useState<DashboardData | null>(null);
  const [status, setStatus] = useState<Status>('idle');
  const [error, setError] = useState<string | null>(null);
  const [refreshing, setRefreshing] = useState(false);
  const requestRef = useRef(0);
  const dataRef = useRef<DashboardData | null>(null);
  const setAlertCount = useAlertStore((state) => state.setAlertCount);

  useEffect(() => {
    dataRef.current = data;
  }, [data]);

  const load = useCallback(
    async (mode: 'initial' | 'refresh') => {
      if (!outletId) {
        setData(null);
        setStatus('idle');
        setAlertCount(0);
        return;
      }

      const requestId = ++requestRef.current;
      if (mode === 'initial') {
        setStatus('loading');
        setData(null);
      } else {
        setRefreshing(true);
      }
      setError(null);

      const { from, to } = rangeFor(range);
      try {
        const [summary, charts, alerts, technicians] = await Promise.all([
          getSummary(to),
          getCharts(from, to),
          getAlerts(),
          getTechnicians(from, to),
        ]);
        if (requestId !== requestRef.current) return;

        setData({ summary, charts, alerts, technicians });
        setStatus('ready');
        setAlertCount(alerts.overdue.length + alerts.not_picked_up.length + alerts.low_stock.length);
      } catch (caught) {
        if (requestId !== requestRef.current) return;
        setError(
          caught instanceof ApiError
            ? caught.message
            : 'Dashboard tidak dapat dimuat. Periksa koneksi Anda lalu coba lagi.',
        );
        setStatus(dataRef.current ? 'ready' : 'error');
      } finally {
        if (requestId === requestRef.current) setRefreshing(false);
      }
    },
    [outletId, range, setAlertCount],
  );

  useEffect(() => {
    void load('initial');
  }, [load]);

  return {
    data,
    status,
    error,
    refreshing,
    refresh: () => void load('refresh'),
  };
}
