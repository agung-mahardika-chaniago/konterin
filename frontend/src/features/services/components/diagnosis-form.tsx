import * as React from 'react';
import { Clock, Pencil, Save, Stethoscope, User, X } from 'lucide-react';

import { Alert } from '@/components/ui/alert';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Textarea } from '@/components/ui/textarea';
import { ApiError } from '@/lib/api-client';
import { upsertDiagnostic } from '../api';
import type { Diagnostic } from '../types';

interface DiagnosisFormProps {
  orderId: string;
  diagnostic: Diagnostic | null;
  canManage: boolean;
  onSaved: (diagnostic: Diagnostic) => void;
}

interface FormState {
  diagnosis: string;
  findings: string;
  actionTaken: string;
  testResult: string;
  technicianNotes: string;
  estimatedDoneAt: string;
}

const EMPTY_FORM: FormState = {
  diagnosis: '',
  findings: '',
  actionTaken: '',
  testResult: '',
  technicianNotes: '',
  estimatedDoneAt: '',
};

function toLocalInput(value: string | null): string {
  if (!value) return '';
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return '';
  const pad = (n: number) => String(n).padStart(2, '0');
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}T${pad(date.getHours())}:${pad(date.getMinutes())}`;
}

function fromDiagnostic(diagnostic: Diagnostic | null): FormState {
  if (!diagnostic) return EMPTY_FORM;
  return {
    diagnosis: diagnostic.diagnosis ?? '',
    findings: diagnostic.findings ?? '',
    actionTaken: diagnostic.action_taken ?? '',
    testResult: diagnostic.test_result ?? '',
    technicianNotes: diagnostic.technician_notes ?? '',
    estimatedDoneAt: toLocalInput(diagnostic.estimated_done_at),
  };
}

export function DiagnosisForm({ orderId, diagnostic, canManage, onSaved }: DiagnosisFormProps) {
  const [editing, setEditing] = React.useState(false);
  const [form, setForm] = React.useState<FormState>(() => fromDiagnostic(diagnostic));
  const [loading, setLoading] = React.useState(false);
  const [error, setError] = React.useState<string | null>(null);

  React.useEffect(() => {
    setForm(fromDiagnostic(diagnostic));
  }, [diagnostic]);

  const update = (field: keyof FormState, value: string) => {
    setForm((current) => ({ ...current, [field]: value }));
  };

  const startEditing = () => {
    setForm(fromDiagnostic(diagnostic));
    setError(null);
    setEditing(true);
  };

  const cancelEditing = () => {
    setForm(fromDiagnostic(diagnostic));
    setError(null);
    setEditing(false);
  };

  const handleSubmit = async (event: React.FormEvent) => {
    event.preventDefault();
    setLoading(true);
    setError(null);
    try {
      const saved = await upsertDiagnostic(orderId, {
        diagnosis: form.diagnosis.trim() || null,
        findings: form.findings.trim() || null,
        action_taken: form.actionTaken.trim() || null,
        test_result: form.testResult.trim() || null,
        technician_notes: form.technicianNotes.trim() || null,
        estimated_done_at: form.estimatedDoneAt ? new Date(form.estimatedDoneAt).toISOString() : null,
      });
      onSaved(saved);
      setEditing(false);
    } catch (err: unknown) {
      setError(err instanceof ApiError ? err.message : 'Gagal menyimpan diagnosis.');
    } finally {
      setLoading(false);
    }
  };

  return (
    <Card>
      <CardHeader className="flex flex-row items-center justify-between gap-3 pb-3">
        <div className="flex items-center gap-2">
          <Stethoscope className="h-5 w-5 text-accent-600" aria-hidden="true" />
          <div>
            <CardTitle className="text-base font-semibold">Diagnosis & Pengerjaan</CardTitle>
            <p className="text-xs text-muted">Temuan teknisi, tindakan, hasil pengujian, dan target selesai.</p>
          </div>
        </div>
        {canManage && !editing && (
          <Button variant="outline" size="sm" onClick={startEditing} className="gap-1.5 text-xs">
            <Pencil className="h-3.5 w-3.5" aria-hidden="true" />
            {diagnostic ? 'Ubah' : 'Isi Diagnosis'}
          </Button>
        )}
      </CardHeader>
      <CardContent>
        {error && (
          <div className="mb-3">
            <Alert variant="error">{error}</Alert>
          </div>
        )}

        {editing ? (
          <form onSubmit={handleSubmit} className="space-y-3">
            <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
              <div className="sm:col-span-2">
                <Label htmlFor="diag-diagnosis">Diagnosis Utama</Label>
                <Textarea
                  id="diag-diagnosis"
                  value={form.diagnosis}
                  onChange={(event) => update('diagnosis', event.target.value)}
                  placeholder="Contoh: Kerusakan pada IC power / LCD"
                  rows={2}
                  className="mt-1"
                />
              </div>
              <div>
                <Label htmlFor="diag-findings">Temuan</Label>
                <Textarea
                  id="diag-findings"
                  value={form.findings}
                  onChange={(event) => update('findings', event.target.value)}
                  rows={3}
                  className="mt-1"
                />
              </div>
              <div>
                <Label htmlFor="diag-action">Tindakan Perbaikan</Label>
                <Textarea
                  id="diag-action"
                  value={form.actionTaken}
                  onChange={(event) => update('actionTaken', event.target.value)}
                  rows={3}
                  className="mt-1"
                />
              </div>
              <div>
                <Label htmlFor="diag-test">Hasil Testing</Label>
                <Textarea
                  id="diag-test"
                  value={form.testResult}
                  onChange={(event) => update('testResult', event.target.value)}
                  rows={2}
                  className="mt-1"
                />
              </div>
              <div>
                <Label htmlFor="diag-notes">Catatan Internal</Label>
                <Textarea
                  id="diag-notes"
                  value={form.technicianNotes}
                  onChange={(event) => update('technicianNotes', event.target.value)}
                  rows={2}
                  className="mt-1"
                />
              </div>
              <div className="sm:col-span-2">
                <Label htmlFor="diag-estimate">Target Estimasi Selesai</Label>
                <Input
                  id="diag-estimate"
                  type="datetime-local"
                  value={form.estimatedDoneAt}
                  onChange={(event) => update('estimatedDoneAt', event.target.value)}
                  className="mt-1"
                />
              </div>
            </div>

            <div className="flex justify-end gap-2 pt-1">
              <Button type="button" variant="outline" size="sm" onClick={cancelEditing} disabled={loading} className="gap-1.5">
                <X className="h-3.5 w-3.5" aria-hidden="true" />
                Batal
              </Button>
              <Button type="submit" size="sm" loading={loading} className="gap-1.5">
                <Save className="h-3.5 w-3.5" aria-hidden="true" />
                Simpan Diagnosis
              </Button>
            </div>
          </form>
        ) : diagnostic ? (
          <div className="space-y-4">
            <div className="grid grid-cols-1 sm:grid-cols-2 gap-4 text-sm">
              <ReadOnlyField label="Diagnosis" value={diagnostic.diagnosis} />
              <ReadOnlyField label="Temuan" value={diagnostic.findings} />
              <ReadOnlyField label="Tindakan" value={diagnostic.action_taken} />
              <ReadOnlyField label="Hasil Testing" value={diagnostic.test_result} />
              {diagnostic.technician_notes && (
                <div className="sm:col-span-2">
                  <ReadOnlyField label="Catatan Internal" value={diagnostic.technician_notes} />
                </div>
              )}
            </div>
            <div className="flex flex-wrap items-center gap-x-4 gap-y-1 border-t border-border pt-3 text-[11px] text-muted">
              {diagnostic.diagnosed_by && (
                <span className="inline-flex items-center gap-1">
                  <User className="h-3.5 w-3.5" aria-hidden="true" />
                  {diagnostic.diagnosed_by}
                </span>
              )}
              {diagnostic.diagnosed_at && (
                <span className="inline-flex items-center gap-1">
                  <Clock className="h-3.5 w-3.5" aria-hidden="true" />
                  {new Date(diagnostic.diagnosed_at).toLocaleDateString('id-ID', {
                    day: 'numeric',
                    month: 'short',
                    year: 'numeric',
                    hour: '2-digit',
                    minute: '2-digit',
                  })}
                </span>
              )}
              {diagnostic.estimated_done_at && (
                <span className="inline-flex items-center gap-1 font-medium text-ink">
                  <Clock className="h-3.5 w-3.5" aria-hidden="true" />
                  Target selesai:{' '}
                  {new Date(diagnostic.estimated_done_at).toLocaleDateString('id-ID', {
                    day: 'numeric',
                    month: 'short',
                    year: 'numeric',
                    hour: '2-digit',
                    minute: '2-digit',
                  })}
                </span>
              )}
            </div>
          </div>
        ) : (
          <div className="rounded-lg border border-dashed border-border p-6 text-center">
            <Stethoscope className="mx-auto h-8 w-8 text-muted" aria-hidden="true" />
            <p className="mt-2 text-sm font-semibold text-ink">Belum ada diagnosis</p>
            <p className="mt-1 text-xs text-muted">
              {canManage
                ? 'Isi temuan dan tindakan agar progres pengerjaan tercatat.'
                : 'Diagnosis akan muncul setelah teknisi mengisinya.'}
            </p>
          </div>
        )}
      </CardContent>
    </Card>
  );
}

function ReadOnlyField({ label, value }: { label: string; value: string | null }) {
  return (
    <div>
      <span className="text-[11px] font-semibold uppercase tracking-wider text-muted">{label}</span>
      <p className="mt-1 whitespace-pre-line rounded-lg border border-border bg-background/60 p-2.5 text-xs text-ink">
        {value || <span className="italic text-muted">Belum diisi</span>}
      </p>
    </div>
  );
}
