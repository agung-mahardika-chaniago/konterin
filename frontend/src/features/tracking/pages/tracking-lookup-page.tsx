import * as React from 'react';
import { Link, useSearchParams } from 'react-router-dom';
import { Phone, Search, Ticket } from 'lucide-react';

import { Alert } from '@/components/ui/alert';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import { trackByTicket } from '../api';
import { PublicPageShell } from '../components/public-page-shell';
import { TrackingResult } from '../components/tracking-result';
import type { PublicTracking } from '../types';

export function TrackingLookupPage() {
  const [searchParams] = useSearchParams();
  const [ticket, setTicket] = React.useState(searchParams.get('ticket') ?? '');
  const [last4, setLast4] = React.useState('');
  const [result, setResult] = React.useState<PublicTracking | null>(null);
  const [loading, setLoading] = React.useState(false);
  const [error, setError] = React.useState<string | null>(null);

  const submit = async (event: React.FormEvent) => {
    event.preventDefault();
    setError(null);
    setResult(null);
    if (!ticket.trim()) {
      setError('Nomor tiket wajib diisi.');
      return;
    }
    if (!/^\d{4}$/.test(last4.trim())) {
      setError('Masukkan 4 digit terakhir nomor HP pelanggan.');
      return;
    }
    setLoading(true);
    try {
      const data = await trackByTicket(ticket, last4);
      setResult(data);
    } catch (err: unknown) {
      setError(
        err instanceof Error ? err.message : 'Tiket tidak ditemukan. Periksa nomor tiket dan 4 digit terakhir nomor HP.',
      );
    } finally {
      setLoading(false);
    }
  };

  return (
    <PublicPageShell>
      <div className="space-y-5">
        <div className="text-center">
          <h1 className="text-2xl font-bold tracking-tight text-ink sm:text-3xl">Lacak Status Servis</h1>
          <p className="mx-auto mt-2 max-w-md text-sm text-muted">
            Masukkan nomor tiket servis dan 4 digit terakhir nomor HP yang terdaftar untuk melihat progres perbaikan.
          </p>
        </div>

        {error && <Alert variant="error">{error}</Alert>}

        <Card>
          <CardHeader>
            <CardTitle className="text-base">Cari Tiket</CardTitle>
          </CardHeader>
          <CardContent>
            <form onSubmit={submit} className="grid gap-4 sm:grid-cols-[1fr_180px_auto] sm:items-end">
              <div className="space-y-1.5">
                <label htmlFor="ticket" className="text-xs font-semibold text-ink">
                  Nomor Tiket
                </label>
                <div className="relative">
                  <Ticket className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted" aria-hidden="true" />
                  <Input
                    id="ticket"
                    value={ticket}
                    onChange={(event) => setTicket(event.target.value)}
                    placeholder="FP-20261108-0001"
                    className="pl-9 font-mono"
                    autoComplete="off"
                  />
                </div>
              </div>
              <div className="space-y-1.5">
                <label htmlFor="last4" className="text-xs font-semibold text-ink">
                  4 Digit Terakhir HP
                </label>
                <div className="relative">
                  <Phone className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted" aria-hidden="true" />
                  <Input
                    id="last4"
                    value={last4}
                    onChange={(event) => setLast4(event.target.value.replace(/\D/g, '').slice(0, 4))}
                    placeholder="7890"
                    inputMode="numeric"
                    className="pl-9 font-mono tracking-[0.3em]"
                    autoComplete="off"
                  />
                </div>
              </div>
              <Button type="submit" loading={loading} className="gap-2 sm:mb-0">
                <Search className="h-4 w-4" aria-hidden="true" />
                Lacak
              </Button>
            </form>
            {searchParams.get('ticket') && (
              <p className="mt-3 text-xs text-muted">
                Nomor tiket sudah terisi dari label/tautan. Masukkan 4 digit terakhir nomor HP pelanggan untuk
                melanjutkan.
              </p>
            )}
          </CardContent>
        </Card>

        {result && <TrackingResult data={result} />}

        <p className="text-center text-xs text-muted">
          Punya tautan atau QR tracking dari toko? Buka tautan tersebut langsung.{' '}
          <Link to="/login" className="font-semibold text-accent-600 hover:underline">
            Login staf
          </Link>
        </p>
      </div>
    </PublicPageShell>
  );
}
