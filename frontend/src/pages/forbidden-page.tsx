import { Link, useNavigate } from 'react-router-dom';
import { ArrowLeft, ShieldAlert } from 'lucide-react';

import { BrandMark } from '@/components/shared/brand-mark';
import { Button } from '@/components/ui/button';
import { useAuthStore } from '@/stores/auth-store';

export function ForbiddenPage() {
  const status = useAuthStore((state) => state.status);
  const navigate = useNavigate();
  return (
    <div className="flex min-h-screen flex-col bg-background">
      <header className="border-b border-border bg-surface">
        <div className="mx-auto flex h-16 w-full max-w-5xl items-center px-4 sm:px-6">
          <BrandMark />
        </div>
      </header>
      <main className="flex flex-1 items-center justify-center px-4 py-12">
        <div className="max-w-md text-center">
          <div className="mx-auto mb-5 flex h-14 w-14 items-center justify-center rounded-2xl bg-red-50 text-danger">
            <ShieldAlert className="h-7 w-7" strokeWidth={1.75} aria-hidden="true" />
          </div>
          <p className="text-sm font-medium text-danger">403</p>
          <h1 className="mt-1 text-2xl font-bold">Akses tidak diizinkan</h1>
          <p className="mt-2 text-sm leading-6 text-muted">
            Role Anda tidak memiliki izin untuk membuka halaman ini. Hubungi owner atau admin toko
            jika Anda membutuhkan akses.
          </p>
          <Button className="mt-6" onClick={() => navigate(-1)}>
            <ArrowLeft className="h-4 w-4" aria-hidden="true" />
            Kembali
          </Button>
          {status === 'unauthenticated' && (
            <p className="mt-4 text-sm">
              <Link to="/login" className="text-brand-600 hover:underline">
                Masuk ke akun Anda
              </Link>
            </p>
          )}
        </div>
      </main>
    </div>
  );
}
