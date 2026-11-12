import * as React from 'react';
import { Archive, Download, HardDriveDownload } from 'lucide-react';

import { EmptyState } from '@/components/shared/empty-state';
import { Alert } from '@/components/ui/alert';
import { Button } from '@/components/ui/button';
import { Card } from '@/components/ui/card';
import { Skeleton } from '@/components/ui/skeleton';
import { formatDateTime } from '@/features/invoices/format';
import { createBackup, downloadBackup, listBackups } from '../api';
import { formatBytes } from '../format';
import type { Backup } from '../types';

export function BackupPanel() {
  const [items, setItems] = React.useState<Backup[]>([]);
  const [loading, setLoading] = React.useState(true);
  const [creating, setCreating] = React.useState(false);
  const [downloadingId, setDownloadingId] = React.useState<string | null>(null);
  const [error, setError] = React.useState<string | null>(null);
  const [success, setSuccess] = React.useState<string | null>(null);

  const load = React.useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const result = await listBackups();
      setItems(result.data ?? []);
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Gagal memuat daftar backup.');
    } finally {
      setLoading(false);
    }
  }, []);

  React.useEffect(() => {
    load();
  }, [load]);

  const handleCreate = async () => {
    if (!window.confirm('Buat backup data toko sekarang? Proses ini bisa memakan beberapa detik.')) {
      return;
    }
    setCreating(true);
    setError(null);
    setSuccess(null);
    try {
      const backup = await createBackup();
      setSuccess(`Backup ${backup.file_name} berhasil dibuat (${formatBytes(backup.size_bytes)}).`);
      await load();
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Gagal membuat backup.');
    } finally {
      setCreating(false);
    }
  };

  const handleDownload = async (backup: Backup) => {
    setDownloadingId(backup.id);
    setError(null);
    try {
      await downloadBackup(backup.id);
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Gagal mengunduh backup.');
    } finally {
      setDownloadingId(null);
    }
  };

  return (
    <div className="space-y-4">
      <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        <p className="text-sm text-muted">
          Backup manual menyimpan seluruh data toko dalam satu berkas .sql.gz. Unduh dan simpan di tempat aman.
        </p>
        <Button onClick={handleCreate} loading={creating} className="gap-2 self-start sm:self-auto">
          <HardDriveDownload className="h-4 w-4" aria-hidden="true" />
          Buat Backup Baru
        </Button>
      </div>

      {success && <Alert variant="success">{success}</Alert>}
      {error && <Alert variant="error">{error}</Alert>}

      <Card className="overflow-hidden">
        {loading ? (
          <div className="space-y-3 p-4">
            {[1, 2, 3].map((i) => (
              <div key={i} className="flex items-center justify-between py-2">
                <div className="space-y-1">
                  <Skeleton className="h-4 w-56" />
                  <Skeleton className="h-3 w-32" />
                </div>
                <Skeleton className="h-8 w-24" />
              </div>
            ))}
          </div>
        ) : items.length === 0 ? (
          <EmptyState
            icon={Archive}
            title="Belum ada backup"
            description="Buat backup pertama untuk menyimpan salinan seluruh data toko."
            action={
              <Button onClick={handleCreate} loading={creating} size="sm" className="gap-1.5">
                <HardDriveDownload className="h-4 w-4" aria-hidden="true" />
                Buat Backup Baru
              </Button>
            }
            className="py-12"
          />
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full text-left text-sm">
              <thead className="border-b border-border bg-background text-xs uppercase tracking-wider text-muted">
                <tr>
                  <th scope="col" className="px-5 py-3 font-semibold">Berkas</th>
                  <th scope="col" className="px-5 py-3 font-semibold">Ukuran</th>
                  <th scope="col" className="px-5 py-3 font-semibold">Tabel</th>
                  <th scope="col" className="px-5 py-3 font-semibold">Baris</th>
                  <th scope="col" className="px-5 py-3 font-semibold">Dibuat</th>
                  <th scope="col" className="px-5 py-3 text-right font-semibold">Aksi</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-border">
                {items.map((backup) => (
                  <tr key={backup.id} className="transition-colors hover:bg-background/50">
                    <td className="px-5 py-3.5">
                      <p className="font-mono text-xs text-ink">{backup.file_name}</p>
                      {backup.created_by && <p className="text-[11px] text-muted">oleh {backup.created_by}</p>}
                    </td>
                    <td className="px-5 py-3.5 text-xs text-ink">{formatBytes(backup.size_bytes)}</td>
                    <td className="px-5 py-3.5 text-xs text-muted">{backup.table_count}</td>
                    <td className="px-5 py-3.5 text-xs text-muted">{backup.row_count.toLocaleString('id-ID')}</td>
                    <td className="px-5 py-3.5 text-xs text-muted">{formatDateTime(backup.created_at)}</td>
                    <td className="whitespace-nowrap px-5 py-3.5 text-right">
                      <Button
                        variant="outline"
                        size="sm"
                        onClick={() => handleDownload(backup)}
                        loading={downloadingId === backup.id}
                        className="h-8 gap-1.5"
                      >
                        <Download className="h-3.5 w-3.5" aria-hidden="true" />
                        Unduh
                      </Button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </Card>
    </div>
  );
}
