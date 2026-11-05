import { Link } from 'react-router-dom';
import { Compass } from 'lucide-react';

import { BrandMark } from '@/components/shared/brand-mark';
import { Button } from '@/components/ui/button';

export function NotFoundPage() {
  return (
    <div className="flex min-h-screen flex-col bg-background">
      <header className="border-b border-border bg-surface">
        <div className="mx-auto flex h-16 w-full max-w-5xl items-center px-4 sm:px-6">
          <BrandMark />
        </div>
      </header>
      <main className="flex flex-1 items-center justify-center px-4 py-12">
        <div className="max-w-md text-center">
          <div className="mx-auto mb-5 flex h-14 w-14 items-center justify-center rounded-2xl bg-brand-50 text-brand-900">
            <Compass className="h-7 w-7" strokeWidth={1.75} aria-hidden="true" />
          </div>
          <p className="text-sm font-medium text-muted">404</p>
          <h1 className="mt-1 text-2xl font-bold">Halaman tidak ditemukan</h1>
          <p className="mt-2 text-sm leading-6 text-muted">
            Alamat yang Anda buka tidak tersedia atau sudah dipindahkan.
          </p>
          <Link to="/" className="mt-6 inline-block">
            <Button>Kembali ke beranda</Button>
          </Link>
        </div>
      </main>
    </div>
  );
}
