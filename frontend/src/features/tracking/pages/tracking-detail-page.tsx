import * as React from 'react';
import { Link, useParams } from 'react-router-dom';
import { ArrowLeft, Loader2, SearchX } from 'lucide-react';

import { Alert } from '@/components/ui/alert';
import { Button } from '@/components/ui/button';
import { Card, CardContent } from '@/components/ui/card';
import { trackByToken } from '../api';
import { PublicPageShell } from '../components/public-page-shell';
import { TrackingResult } from '../components/tracking-result';
import type { PublicTracking } from '../types';

export function TrackingDetailPage() {
  const { token } = useParams<{ token: string }>();
  const [data, setData] = React.useState<PublicTracking | null>(null);
  const [loading, setLoading] = React.useState(true);
  const [error, setError] = React.useState<string | null>(null);

  React.useEffect(() => {
    let active = true;
    async function load() {
      if (!token) {
        setError('Token tracking tidak valid.');
        setLoading(false);
        return;
      }
      setLoading(true);
      setError(null);
      try {
        const result = await trackByToken(token);
        if (active) setData(result);
      } catch (err: unknown) {
        if (active) {
          setError(err instanceof Error ? err.message : 'Tautan tracking tidak valid atau sudah kedaluwarsa.');
        }
      } finally {
        if (active) setLoading(false);
      }
    }
    load();
    return () => {
      active = false;
    };
  }, [token]);

  return (
    <PublicPageShell>
      <div className="space-y-5">
        <Link to="/track" className="inline-flex items-center gap-1.5 text-xs font-semibold text-accent-600 hover:underline">
          <ArrowLeft className="h-3.5 w-3.5" aria-hidden="true" />
          Lacak tiket lain
        </Link>

        {loading ? (
          <Card>
            <CardContent className="flex flex-col items-center justify-center gap-3 py-16 text-sm text-muted">
              <Loader2 className="h-6 w-6 animate-spin text-accent-600" aria-hidden="true" />
              Memuat status servis...
            </CardContent>
          </Card>
        ) : error ? (
          <>
            <Alert variant="error">{error}</Alert>
            <Card>
              <CardContent className="flex flex-col items-center gap-3 py-12 text-center">
                <SearchX className="h-8 w-8 text-muted" aria-hidden="true" />
                <p className="text-sm font-medium text-ink">Status servis tidak dapat ditampilkan</p>
                <p className="max-w-sm text-xs text-muted">
                  Tautan mungkin sudah kedaluwarsa. Silakan cari menggunakan nomor tiket dan 4 digit terakhir nomor HP.
                </p>
                <Link to="/track">
                  <Button variant="outline" size="sm">
                    Cari Tiket
                  </Button>
                </Link>
              </CardContent>
            </Card>
          </>
        ) : data ? (
          <TrackingResult data={data} />
        ) : null}
      </div>
    </PublicPageShell>
  );
}
