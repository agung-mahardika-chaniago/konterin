import { create } from 'zustand';

interface AlertState {
  alertCount: number;
  setAlertCount: (count: number) => void;
}

/** Shared alert badge count between the dashboard data and the topbar. */
export const useAlertStore = create<AlertState>((set) => ({
  alertCount: 0,
  setAlertCount: (alertCount) => set({ alertCount }),
}));
