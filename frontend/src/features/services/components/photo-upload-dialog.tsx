import * as React from 'react';
import { Camera, Upload } from 'lucide-react';

import { Alert } from '@/components/ui/alert';
import { Button } from '@/components/ui/button';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Label } from '@/components/ui/label';
import { Select } from '@/components/ui/select';
import { ApiError } from '@/lib/api-client';
import { uploadServicePhoto } from '../api';
import type { ServicePhoto } from '../types';

interface PhotoUploadDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  orderId: string;
  onSuccess: (photo: ServicePhoto) => void;
}

export function PhotoUploadDialog({
  open,
  onOpenChange,
  orderId,
  onSuccess,
}: PhotoUploadDialogProps) {
  const [file, setFile] = React.useState<File | null>(null);
  const [preview, setPreview] = React.useState<string | null>(null);
  const [stage, setStage] = React.useState('intake');
  const [isPublic, setIsPublic] = React.useState(true);
  const [loading, setLoading] = React.useState(false);
  const [error, setError] = React.useState<string | null>(null);

  React.useEffect(() => {
    if (open) {
      setFile(null);
      setPreview(null);
      setStage('intake');
      setIsPublic(true);
      setError(null);
    }
  }, [open]);

  const handleFileChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const selected = e.target.files?.[0];
    if (!selected) return;

    if (!['image/jpeg', 'image/png', 'image/webp'].includes(selected.type)) {
      setError('Format file tidak didukung. Harap unggah foto JPEG, PNG, atau WebP.');
      return;
    }
    if (selected.size > 5 * 1024 * 1024) {
      setError('Ukuran file melebihi batas maksimal 5 MB.');
      return;
    }

    setFile(selected);
    setError(null);
    const reader = new FileReader();
    reader.onload = () => {
      setPreview(reader.result as string);
    };
    reader.readAsDataURL(selected);
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!file) {
      setError('Pilih file foto terlebih dahulu.');
      return;
    }
    setLoading(true);
    setError(null);
    try {
      const uploaded = await uploadServicePhoto(orderId, file, stage, isPublic);
      onSuccess(uploaded);
      onOpenChange(false);
    } catch (err: unknown) {
      if (err instanceof ApiError) {
        setError(err.message);
      } else {
        setError('Gagal mengunggah foto.');
      }
    } finally {
      setLoading(false);
    }
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent onClose={() => onOpenChange(false)} className="max-w-md">
        <form onSubmit={handleSubmit}>
          <DialogHeader>
            <DialogTitle>Unggah Foto Servis</DialogTitle>
            <DialogDescription>
              Dokumentasikan kondisi fisik perangkat atau proses perbaikan (maks. 5 MB).
            </DialogDescription>
          </DialogHeader>

          {error && (
            <div className="mb-4">
              <Alert variant="error">{error}</Alert>
            </div>
          )}

          <div className="space-y-4 py-2">
            <div>
              <Label htmlFor="photo-file">Pilih File Foto</Label>
              <div className="mt-1 flex flex-col items-center justify-center rounded-lg border-2 border-dashed border-border p-4 hover:border-accent-500 transition-colors bg-background/50">
                {preview ? (
                  <div className="space-y-2 text-center">
                    <img src={preview} alt="Pratinjau" className="max-h-48 mx-auto rounded object-contain" />
                    <p className="text-xs text-muted truncate max-w-xs">{file?.name}</p>
                    <Button
                      type="button"
                      variant="outline"
                      size="sm"
                      onClick={() => {
                        setFile(null);
                        setPreview(null);
                      }}
                      className="text-xs"
                    >
                      Ganti Foto
                    </Button>
                  </div>
                ) : (
                  <label
                    htmlFor="photo-file-input"
                    className="flex flex-col items-center justify-center cursor-pointer py-4"
                  >
                    <Camera className="h-8 w-8 text-muted mb-2" aria-hidden="true" />
                    <span className="text-sm font-semibold text-ink">Klik untuk memilih atau memotret</span>
                    <span className="text-xs text-muted mt-1">JPEG, PNG, WebP (maks 5MB)</span>
                    <input
                      id="photo-file-input"
                      type="file"
                      accept="image/jpeg,image/png,image/webp"
                      onChange={handleFileChange}
                      className="sr-only"
                    />
                  </label>
                )}
              </div>
            </div>

            <div>
              <Label htmlFor="photo-stage">Tahap / Kategori Foto</Label>
              <Select id="photo-stage" value={stage} onChange={(e) => setStage(e.target.value)} className="mt-1">
                <option value="intake">Intake (Saat Diterima di Meja)</option>
                <option value="before">Sebelum Pengerjaan</option>
                <option value="after">Setelah Pengerjaan</option>
                <option value="testing">Hasil Testing / Pengujian</option>
                <option value="other">Lainnya</option>
              </Select>
            </div>

            <div className="flex items-center gap-2 pt-1">
              <input
                id="photo-public"
                type="checkbox"
                checked={isPublic}
                onChange={(e) => setIsPublic(e.target.checked)}
                className="h-4 w-4 rounded border-border text-accent-600 focus:ring-accent-500"
              />
              <Label htmlFor="photo-public" className="text-xs font-normal cursor-pointer">
                Tampilkan foto ini di halaman tracking publik untuk pelanggan
              </Label>
            </div>
          </div>

          <DialogFooter>
            <Button type="button" variant="outline" onClick={() => onOpenChange(false)} disabled={loading}>
              Batal
            </Button>
            <Button type="submit" loading={loading} disabled={!file} className="gap-2">
              <Upload className="h-4 w-4" aria-hidden="true" />
              Unggah Foto
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
