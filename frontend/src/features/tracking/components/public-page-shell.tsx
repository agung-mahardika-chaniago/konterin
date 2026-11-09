import type { ReactNode } from 'react';
import { Link } from 'react-router-dom';

import { BrandMark } from '@/components/shared/brand-mark';

/** Minimal, unauthenticated shell used by the public tracking pages. */
export function PublicPageShell({ children }: { children: ReactNode }) {
  return (
    <div className="min-h-screen bg-background">
      <header className="border-b border-border bg-surface">
        <div className="mx-auto flex w-full max-w-3xl items-center justify-between px-4 py-4 sm:px-6">
          <Link to="/track" aria-label="FixPhone tracking">
            <BrandMark />
          </Link>
          <span className="text-xs font-medium text-muted">Lacak Servis</span>
        </div>
      </header>
      <main className="mx-auto w-full max-w-3xl px-4 py-8 sm:px-6">{children}</main>
      <footer className="mx-auto w-full max-w-3xl px-4 pb-8 text-center text-[11px] text-muted sm:px-6">
        FixPhone Service Manager
      </footer>
    </div>
  );
}
