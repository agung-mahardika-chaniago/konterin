import { useState, type FormEvent } from 'react';
import { Eye, EyeOff, LockKeyhole, ShieldCheck, Smartphone, Sparkles } from 'lucide-react';
import { Navigate, useNavigate, useSearchParams } from 'react-router-dom';

import { BrandMark } from '@/components/shared/brand-mark';
import { Alert } from '@/components/ui/alert';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { ApiError } from '@/lib/api-client';
import { primaryRouteFor } from '@/lib/permissions';
import { useAuthStore } from '@/stores/auth-store';

const inputClass =
  'h-11 pl-10';

export function LoginPage() {
  const status = useAuthStore((state) => state.status);
  const user = useAuthStore((state) => state.user);
  const login = useAuthStore((state) => state.login);
  const navigate = useNavigate();
  const [searchParams] = useSearchParams();

  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [showPassword, setShowPassword] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({});

  if (status === 'authenticated' && user) {
    return <Navigate to={primaryRouteFor(user)} replace />;
  }

  const changed = searchParams.get('changed') === '1';

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setError(null);
    setFieldErrors({});

    const nextErrors: Record<string, string> = {};
    if (!email.trim()) nextErrors.email = 'Email wajib diisi.';
    if (!password) nextErrors.password = 'Password wajib diisi.';
    if (Object.keys(nextErrors).length > 0) {
      setFieldErrors(nextErrors);
      return;
    }

    setSubmitting(true);
    try {
      const signedIn = await login(email.trim(), password);
      navigate(primaryRouteFor(signedIn), { replace: true });
    } catch (caught) {
      if (caught instanceof ApiError) {
        setError(caught.message);
        setFieldErrors(caught.fieldErrors());
      } else {
        setError('Tidak dapat masuk. Pastikan server FixPhone berjalan lalu coba lagi.');
      }
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <div className="min-h-screen bg-background lg:grid lg:grid-cols-[1.05fr_1fr]">
      <aside className="relative hidden overflow-hidden bg-brand-950 px-12 py-10 text-white lg:flex lg:flex-col lg:justify-between xl:px-16">
        <div
          className="pointer-events-none absolute -right-32 -top-32 h-96 w-96 rounded-full bg-accent-500/15 blur-3xl"
          aria-hidden="true"
        />
        <div
          className="pointer-events-none absolute -bottom-40 -left-24 h-96 w-96 rounded-full bg-brand-600/25 blur-3xl"
          aria-hidden="true"
        />

        <BrandMark inverted className="relative" />

        <div className="relative max-w-lg">
          <p className="mb-4 inline-flex items-center gap-2 rounded-full border border-white/15 bg-white/5 px-3 py-1 text-xs font-medium text-brand-100">
            <Sparkles className="h-3.5 w-3.5 text-accent-500" aria-hidden="true" />
            Satu tempat untuk seluruh operasional servis
          </p>
          <h1 className="text-4xl font-bold leading-[1.15] tracking-tight xl:text-5xl">
            Kelola servis HP
            <span className="block text-accent-500">lebih cepat dan rapi.</span>
          </h1>
          <p className="mt-5 max-w-md text-sm leading-6 text-brand-100/80">
            Terima servis, pantau pengerjaan teknisi, catat pembayaran, dan jaga stok sparepart
            tanpa berpindah aplikasi.
          </p>

          <ul className="mt-8 space-y-4">
            {[
              { icon: Smartphone, text: 'Alur servis dari tiket masuk sampai garansi.' },
              { icon: ShieldCheck, text: 'Akses sesuai peran dan data terpisah per toko.' },
              { icon: LockKeyhole, text: 'Sesi aman dengan refresh token HttpOnly.' },
            ].map(({ icon: Icon, text }) => (
              <li key={text} className="flex items-start gap-3 text-sm text-brand-100/90">
                <span className="mt-0.5 flex h-8 w-8 shrink-0 items-center justify-center rounded-lg bg-white/10 text-accent-500">
                  <Icon className="h-4 w-4" strokeWidth={1.75} aria-hidden="true" />
                </span>
                <span className="pt-1.5">{text}</span>
              </li>
            ))}
          </ul>
        </div>

        <p className="relative text-xs text-brand-100/50">© 2026 FixPhone. Semua hak dilindungi.</p>
      </aside>

      <main className="flex min-h-screen flex-col justify-center px-4 py-10 sm:px-8 lg:px-12">
        <div className="mx-auto w-full max-w-md">
          <div className="mb-8 flex items-center justify-between lg:hidden">
            <BrandMark />
          </div>

          <Card className="border-border/80 shadow-card">
            <CardHeader className="space-y-1.5 p-6 pb-4">
              <CardTitle className="text-2xl leading-8">Masuk ke akun Anda</CardTitle>
              <CardDescription>
                Gunakan email dan password yang diberikan administrator toko.
              </CardDescription>
            </CardHeader>
            <CardContent className="p-6 pt-2">
              {changed && (
                <Alert variant="success" className="mb-5">
                  Password berhasil diubah. Silakan masuk dengan password baru Anda.
                </Alert>
              )}
              {error && (
                <Alert variant="error" className="mb-5">
                  {error}
                </Alert>
              )}

              <form onSubmit={handleSubmit} noValidate className="space-y-4">
                <div className="space-y-1.5">
                  <Label htmlFor="email">Email</Label>
                  <div className="relative">
                    <Smartphone
                      className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted"
                      aria-hidden="true"
                    />
                    <Input
                      id="email"
                      name="email"
                      type="email"
                      inputMode="email"
                      autoComplete="username"
                      autoFocus
                      placeholder="nama@toko.id"
                      className={inputClass}
                      value={email}
                      onChange={(event) => setEmail(event.target.value)}
                      aria-invalid={Boolean(fieldErrors.email)}
                      aria-describedby={fieldErrors.email ? 'email-error' : undefined}
                      disabled={submitting}
                    />
                  </div>
                  {fieldErrors.email && (
                    <p id="email-error" className="text-xs text-danger">
                      {fieldErrors.email}
                    </p>
                  )}
                </div>

                <div className="space-y-1.5">
                  <Label htmlFor="password">Password</Label>
                  <div className="relative">
                    <LockKeyhole
                      className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted"
                      aria-hidden="true"
                    />
                    <Input
                      id="password"
                      name="password"
                      type={showPassword ? 'text' : 'password'}
                      autoComplete="current-password"
                      placeholder="Masukkan password"
                      className="h-11 px-10"
                      value={password}
                      onChange={(event) => setPassword(event.target.value)}
                      aria-invalid={Boolean(fieldErrors.password)}
                      aria-describedby={fieldErrors.password ? 'password-error' : undefined}
                      disabled={submitting}
                    />
                    <button
                      type="button"
                      onClick={() => setShowPassword((value) => !value)}
                      className="absolute right-2 top-1/2 -translate-y-1/2 rounded p-1.5 text-muted hover:bg-background hover:text-ink"
                      aria-label={showPassword ? 'Sembunyikan password' : 'Tampilkan password'}
                      tabIndex={-1}
                    >
                      {showPassword ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
                    </button>
                  </div>
                  {fieldErrors.password && (
                    <p id="password-error" className="text-xs text-danger">
                      {fieldErrors.password}
                    </p>
                  )}
                </div>

                <Button type="submit" size="lg" className="mt-2 w-full" loading={submitting}>
                  {submitting ? 'Memproses…' : 'Masuk'}
                </Button>
              </form>

              <div className="mt-6 rounded-control bg-background p-3 text-xs leading-5 text-muted">
                Lupa password? Hubungi owner atau admin toko Anda untuk reset password.
                Anda akan diminta mengganti password sementara saat pertama masuk.
              </div>
            </CardContent>
          </Card>

          <p className="mt-6 text-center text-xs text-muted">
            Dengan masuk, Anda menyetujui penggunaan data sesuai kebijakan toko.
          </p>
        </div>
      </main>
    </div>
  );
}
