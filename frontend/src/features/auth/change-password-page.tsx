import { useState, type FormEvent } from 'react';
import { KeyRound, LogOut, ShieldCheck } from 'lucide-react';
import { useNavigate } from 'react-router-dom';

import { BrandMark } from '@/components/shared/brand-mark';
import { Alert } from '@/components/ui/alert';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { ApiError } from '@/lib/api-client';
import { useAuthStore } from '@/stores/auth-store';

export function ChangePasswordPage() {
  const user = useAuthStore((state) => state.user);
  const changePassword = useAuthStore((state) => state.changePassword);
  const logout = useAuthStore((state) => state.logout);
  const clearSession = useAuthStore((state) => state.clearSession);
  const navigate = useNavigate();

  const [currentPassword, setCurrentPassword] = useState('');
  const [newPassword, setNewPassword] = useState('');
  const [confirmPassword, setConfirmPassword] = useState('');
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({});

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setError(null);
    setFieldErrors({});

    const nextErrors: Record<string, string> = {};
    if (!currentPassword) nextErrors.current_password = 'Password lama wajib diisi.';
    if (!newPassword) {
      nextErrors.new_password = 'Password baru wajib diisi.';
    } else if (newPassword.length < 8) {
      nextErrors.new_password = 'Password minimal 8 karakter.';
    } else if (!/[A-Za-z]/.test(newPassword) || !/[0-9]/.test(newPassword)) {
      nextErrors.new_password = 'Password harus mengandung huruf dan angka.';
    }
    if (newPassword && confirmPassword !== newPassword) {
      nextErrors.confirm_password = 'Konfirmasi password tidak sama.';
    }
    if (Object.keys(nextErrors).length > 0) {
      setFieldErrors(nextErrors);
      return;
    }

    setSubmitting(true);
    try {
      await changePassword(currentPassword, newPassword);
      clearSession();
      navigate('/login?changed=1', { replace: true });
    } catch (caught) {
      if (caught instanceof ApiError) {
        setError(caught.message);
        setFieldErrors(caught.fieldErrors());
      } else {
        setError('Tidak dapat mengubah password. Coba lagi.');
      }
    } finally {
      setSubmitting(false);
    }
  }

  async function handleLogout() {
    await logout();
    navigate('/login', { replace: true });
  }

  return (
    <div className="flex min-h-screen flex-col bg-background">
      <header className="border-b border-border bg-surface">
        <div className="mx-auto flex h-16 w-full max-w-5xl items-center justify-between px-4 sm:px-6">
          <BrandMark />
          <Button variant="ghost" size="sm" onClick={handleLogout}>
            <LogOut className="h-4 w-4" aria-hidden="true" />
            Keluar
          </Button>
        </div>
      </header>

      <main className="mx-auto flex w-full max-w-lg flex-1 items-start justify-center px-4 py-10 sm:py-16">
        <Card className="w-full">
          <CardHeader className="p-6 pb-4">
            <div className="mb-3 flex h-11 w-11 items-center justify-center rounded-xl bg-brand-50 text-brand-900">
              <ShieldCheck className="h-5 w-5" strokeWidth={1.75} aria-hidden="true" />
            </div>
            <CardTitle className="text-2xl leading-8">Ubah password</CardTitle>
            <CardDescription>
              Hai {user?.name ?? 'Pengguna'}, demi keamanan akun Anda wajib mengganti password
              sebelum melanjutkan.
            </CardDescription>
          </CardHeader>
          <CardContent className="p-6 pt-2">
            {error && (
              <Alert variant="error" className="mb-5">
                {error}
              </Alert>
            )}

            <form onSubmit={handleSubmit} noValidate className="space-y-4">
              <div className="space-y-1.5">
                <Label htmlFor="current_password">Password saat ini</Label>
                <Input
                  id="current_password"
                  type="password"
                  autoComplete="current-password"
                  value={currentPassword}
                  onChange={(event) => setCurrentPassword(event.target.value)}
                  aria-invalid={Boolean(fieldErrors.current_password)}
                  disabled={submitting}
                />
                {fieldErrors.current_password && (
                  <p className="text-xs text-danger">{fieldErrors.current_password}</p>
                )}
              </div>

              <div className="space-y-1.5">
                <Label htmlFor="new_password">Password baru</Label>
                <Input
                  id="new_password"
                  type="password"
                  autoComplete="new-password"
                  value={newPassword}
                  onChange={(event) => setNewPassword(event.target.value)}
                  aria-invalid={Boolean(fieldErrors.new_password)}
                  disabled={submitting}
                />
                <p className="text-xs text-muted">
                  Minimal 8 karakter, mengandung huruf dan angka.
                </p>
                {fieldErrors.new_password && (
                  <p className="text-xs text-danger">{fieldErrors.new_password}</p>
                )}
              </div>

              <div className="space-y-1.5">
                <Label htmlFor="confirm_password">Ulangi password baru</Label>
                <Input
                  id="confirm_password"
                  type="password"
                  autoComplete="new-password"
                  value={confirmPassword}
                  onChange={(event) => setConfirmPassword(event.target.value)}
                  aria-invalid={Boolean(fieldErrors.confirm_password)}
                  disabled={submitting}
                />
                {fieldErrors.confirm_password && (
                  <p className="text-xs text-danger">{fieldErrors.confirm_password}</p>
                )}
              </div>

              <Button type="submit" size="lg" className="w-full" loading={submitting}>
                <KeyRound className="h-4 w-4" aria-hidden="true" />
                {submitting ? 'Menyimpan…' : 'Simpan password baru'}
              </Button>
            </form>

            <p className="mt-4 text-xs leading-5 text-muted">
              Setelah password diubah, semua sesi lain dicabut dan Anda perlu masuk kembali.
            </p>
          </CardContent>
        </Card>
      </main>
    </div>
  );
}
