import { useEffect, type ReactNode } from 'react';
import { BrowserRouter } from 'react-router-dom';

import { useAuthStore } from '@/stores/auth-store';

const IDLE_TIMEOUT_MS = 30 * 60 * 1000;
const ACTIVITY_EVENTS = ['mousemove', 'keydown', 'click', 'touchstart', 'scroll'] as const;

export function Providers({ children }: { children: ReactNode }) {
  const bootstrap = useAuthStore((state) => state.bootstrap);
  const status = useAuthStore((state) => state.status);
  const logout = useAuthStore((state) => state.logout);

  useEffect(() => {
    void bootstrap();
  }, [bootstrap]);

  useEffect(() => {
    if (status !== 'authenticated') return;

    let timer = 0;
    const reset = () => {
      window.clearTimeout(timer);
      timer = window.setTimeout(() => {
        void logout();
      }, IDLE_TIMEOUT_MS);
    };

    ACTIVITY_EVENTS.forEach((event) => window.addEventListener(event, reset, { passive: true }));
    reset();

    return () => {
      window.clearTimeout(timer);
      ACTIVITY_EVENTS.forEach((event) => window.removeEventListener(event, reset));
    };
  }, [status, logout]);

  return <BrowserRouter future={{ v7_startTransition: true, v7_relativeSplatPath: true }}>{children}</BrowserRouter>;
}
