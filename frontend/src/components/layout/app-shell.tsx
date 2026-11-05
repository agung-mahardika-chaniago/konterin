import { useEffect, useRef, useState, type ReactNode } from 'react';
import { useLocation } from 'react-router-dom';

import { Sidebar } from '@/components/layout/sidebar';
import { Topbar } from '@/components/layout/topbar';
import { useAlertStore } from '@/stores/alert-store';

const COLLAPSE_KEY = 'fixphone.sidebar_collapsed';

function readCollapsed(): boolean {
  try {
    return window.localStorage.getItem(COLLAPSE_KEY) === '1';
  } catch {
    return false;
  }
}

function todayLabel(): string {
  const label = new Intl.DateTimeFormat('id-ID', {
    weekday: 'long',
    day: 'numeric',
    month: 'long',
    year: 'numeric',
  }).format(new Date());
  return label.charAt(0).toUpperCase() + label.slice(1);
}

export function AppShell({ children }: { children: ReactNode }) {
  const [collapsed, setCollapsed] = useState(readCollapsed);
  const [mobileOpen, setMobileOpen] = useState(false);
  const alertCount = useAlertStore((state) => state.alertCount);
  const location = useLocation();
  const drawerRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    setMobileOpen(false);
  }, [location.pathname]);

  useEffect(() => {
    if (!mobileOpen) return;

    const previousOverflow = document.body.style.overflow;
    document.body.style.overflow = 'hidden';
    const frame = window.requestAnimationFrame(() => {
      drawerRef.current?.querySelector<HTMLElement>('a, button, select')?.focus();
    });
    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key === 'Escape') setMobileOpen(false);
    };
    document.addEventListener('keydown', onKeyDown);

    return () => {
      window.cancelAnimationFrame(frame);
      document.body.style.overflow = previousOverflow;
      document.removeEventListener('keydown', onKeyDown);
    };
  }, [mobileOpen]);

  function toggleCollapse() {
    setCollapsed((value) => {
      const next = !value;
      try {
        window.localStorage.setItem(COLLAPSE_KEY, next ? '1' : '0');
      } catch {
        // Ignore storage failures; the UI still collapses for this session.
      }
      return next;
    });
  }

  return (
    <div className="min-h-screen bg-background">
      <a
        href="#main-content"
        className="sr-only z-50 rounded-control bg-brand-900 px-4 py-2 text-sm font-medium text-white focus:not-sr-only focus:fixed focus:left-4 focus:top-4"
      >
        Lewati ke konten utama
      </a>

      <Sidebar collapsed={collapsed} onToggleCollapse={toggleCollapse} />

      {mobileOpen && (
        <div className="fixed inset-0 z-50 lg:hidden">
          <div
            className="absolute inset-0 bg-brand-950/60"
            aria-hidden="true"
            onClick={() => setMobileOpen(false)}
          />
          <div
            ref={drawerRef}
            role="dialog"
            aria-modal="true"
            aria-label="Menu navigasi"
            className="absolute inset-y-0 left-0 shadow-modal"
          >
            <Sidebar collapsed={false} mobile onClose={() => setMobileOpen(false)} />
          </div>
        </div>
      )}

      <div className={collapsed ? 'lg:pl-[76px]' : 'lg:pl-[264px]'}>
        <Topbar
          onOpenMenu={() => setMobileOpen(true)}
          menuOpen={mobileOpen}
          alertCount={alertCount}
          title="Dashboard"
          subtitle={todayLabel()}
        />
        <main id="main-content" tabIndex={-1} className="mx-auto w-full max-w-[1400px] px-3 py-5 focus:outline-none sm:px-5 sm:py-6">
          {children}
        </main>
      </div>
    </div>
  );
}
