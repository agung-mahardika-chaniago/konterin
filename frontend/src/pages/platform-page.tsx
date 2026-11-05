import { useNavigate } from 'react-router-dom';
import { Building2, LogOut, ShieldCheck } from 'lucide-react';

import { BrandMark } from '@/components/shared/brand-mark';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { useAuthStore } from '@/stores/auth-store';

export function PlatformPage() {
  const user = useAuthStore((state) => state.user);
  const logout = useAuthStore((state) => state.logout);
  const navigate = useNavigate();

  if (!user) return null;

  async function handleLogout() {
    await logout();
    navigate('/login', { replace: true });
  }

  const canManage = user.permissions.includes('platform.tenant.manage');

  return (
    <div className="min-h-screen bg-background">
      <header className="border-b border-border bg-surface">
        <div className="mx-auto flex h-16 w-full max-w-5xl items-center justify-between px-4 sm:px-6">
          <BrandMark />
          <Button variant="outline" size="sm" onClick={handleLogout}>
            <LogOut className="h-4 w-4" aria-hidden="true" />
            Keluar
          </Button>
        </div>
      </header>

      <main className="mx-auto w-full max-w-5xl space-y-6 px-4 py-10 sm:px-6">
        <div>
          <p className="text-sm text-muted">Area platform</p>
          <h1 className="mt-1 text-2xl font-bold tracking-tight sm:text-3xl">Halo, {user.name}</h1>
          <p className="mt-2 text-sm text-muted">
            Anda masuk sebagai Super Admin FixPhone. Pengelolaan tenant akan tersedia pada tahap
            platform.
          </p>
        </div>

        <div className="grid gap-4 sm:grid-cols-2">
          <Card>
            <CardHeader>
              <div className="mb-2 flex h-10 w-10 items-center justify-center rounded-lg bg-brand-50 text-brand-900">
                <Building2 className="h-5 w-5" aria-hidden="true" />
              </div>
              <CardTitle>Tenant</CardTitle>
              <CardDescription>
                {canManage ? 'Anda memiliki izin mengelola tenant.' : 'Akses tenant hanya baca.'}
              </CardDescription>
            </CardHeader>
            <CardContent>
              <p className="text-xs text-muted">
                Endpoint platform akan dibangun bersama modul dashboard dan pengaturan.
              </p>
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <div className="mb-2 flex h-10 w-10 items-center justify-center rounded-lg bg-brand-50 text-brand-900">
                <ShieldCheck className="h-5 w-5" aria-hidden="true" />
              </div>
              <CardTitle>Keamanan sesi</CardTitle>
              <CardDescription>Access token di memori, refresh token HttpOnly.</CardDescription>
            </CardHeader>
            <CardContent>
              <p className="text-xs text-muted">
                {user.permissions.length} izin platform aktif untuk akun ini.
              </p>
            </CardContent>
          </Card>
        </div>

        {user.must_change_password && (
          <p className="text-sm text-warning">
            Anda masih menggunakan password awal. Ubah password sebelum melanjutkan.
          </p>
        )}
      </main>
    </div>
  );
}
