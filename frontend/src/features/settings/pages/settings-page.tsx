import * as React from 'react';
import { Save, Settings as SettingsIcon } from 'lucide-react';

import { Alert } from '@/components/ui/alert';
import { Button } from '@/components/ui/button';
import { Card } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Select } from '@/components/ui/select';
import { Skeleton } from '@/components/ui/skeleton';
import { Textarea } from '@/components/ui/textarea';
import { ApiError } from '@/lib/api-client';
import { hasPermission } from '@/lib/permissions';
import { cn } from '@/lib/utils';
import { formatDateTime } from '@/features/invoices/format';
import { useAuthStore } from '@/stores/auth-store';
import { getSettings, updateSettings } from '../api';
import { BackupPanel } from '../components/backup-panel';
import { waProviderLabel } from '../format';
import type { Settings, SettingsForm } from '../types';

type SettingsTab = 'identity' | 'documents' | 'warranty' | 'whatsapp' | 'backup';

const TABS: { key: SettingsTab; label: string; manageOnly?: boolean }[] = [
  { key: 'identity', label: 'Identitas Toko' },
  { key: 'documents', label: 'Tiket & Invoice' },
  { key: 'warranty', label: 'Garansi & Struk' },
  { key: 'whatsapp', label: 'WhatsApp' },
  { key: 'backup', label: 'Backup', manageOnly: true },
];

function settingsToForm(settings: Settings): SettingsForm {
  return {
    store_name: settings.store_name,
    legal_name: settings.legal_name ?? '',
    address: settings.address ?? '',
    phone: settings.phone ?? '',
    email: settings.email ?? '',
    tax_id: settings.tax_id ?? '',
    timezone: settings.timezone,
    ticket_prefix: settings.ticket_prefix,
    ticket_format: settings.ticket_format,
    invoice_prefix: settings.invoice_prefix,
    invoice_format: settings.invoice_format,
    invoice_terms: settings.invoice_terms ?? '',
    invoice_footer: settings.invoice_footer ?? '',
    default_service_warranty_days: settings.default_service_warranty_days,
    default_part_warranty_days: settings.default_part_warranty_days,
    receipt_paper_width: settings.receipt_paper_width,
    technician_sees_only_assigned: settings.technician_sees_only_assigned,
    public_tracking_enabled: settings.public_tracking_enabled,
    wa_provider: settings.wa_provider,
    wa_templates: { ...settings.wa_templates },
  };
}

function TextField({
  id,
  label,
  value,
  onChange,
  error,
  disabled,
  placeholder,
  maxLength,
  type = 'text',
  hint,
}: {
  id: string;
  label: string;
  value: string;
  onChange: (value: string) => void;
  error?: string;
  disabled: boolean;
  placeholder?: string;
  maxLength?: number;
  type?: string;
  hint?: string;
}) {
  return (
    <div className="space-y-1.5">
      <Label htmlFor={id}>{label}</Label>
      <Input
        id={id}
        type={type}
        value={value}
        onChange={(event) => onChange(event.target.value)}
        disabled={disabled}
        placeholder={placeholder}
        maxLength={maxLength}
      />
      {hint && !error && <p className="text-xs text-muted">{hint}</p>}
      {error && <p className="text-xs text-danger">{error}</p>}
    </div>
  );
}

function NumberField({
  id,
  label,
  value,
  onChange,
  error,
  disabled,
  min,
  max,
  hint,
}: {
  id: string;
  label: string;
  value: number;
  onChange: (value: number) => void;
  error?: string;
  disabled: boolean;
  min?: number;
  max?: number;
  hint?: string;
}) {
  return (
    <div className="space-y-1.5">
      <Label htmlFor={id}>{label}</Label>
      <Input
        id={id}
        type="number"
        value={String(value)}
        min={min}
        max={max}
        onChange={(event) => onChange(Number(event.target.value) || 0)}
        disabled={disabled}
        className="font-mono"
      />
      {hint && !error && <p className="text-xs text-muted">{hint}</p>}
      {error && <p className="text-xs text-danger">{error}</p>}
    </div>
  );
}

function ToggleField({
  id,
  label,
  description,
  checked,
  onChange,
  disabled,
}: {
  id: string;
  label: string;
  description: string;
  checked: boolean;
  onChange: (checked: boolean) => void;
  disabled: boolean;
}) {
  return (
    <label htmlFor={id} className="flex items-start gap-3 rounded-control border border-border bg-surface p-3">
      <input
        id={id}
        type="checkbox"
        checked={checked}
        onChange={(event) => onChange(event.target.checked)}
        disabled={disabled}
        className="mt-0.5 h-4 w-4 rounded border-border text-accent-600 focus:ring-accent-500"
      />
      <span>
        <span className="block text-sm font-medium text-ink">{label}</span>
        <span className="block text-xs text-muted">{description}</span>
      </span>
    </label>
  );
}

export function SettingsPage() {
  const user = useAuthStore((state) => state.user);
  const canManage = hasPermission(user, 'settings.manage');

  const [activeTab, setActiveTab] = React.useState<SettingsTab>('identity');
  const [form, setForm] = React.useState<SettingsForm | null>(null);
  const [updatedAt, setUpdatedAt] = React.useState<string | null>(null);
  const [loading, setLoading] = React.useState(true);
  const [saving, setSaving] = React.useState(false);
  const [error, setError] = React.useState<string | null>(null);
  const [success, setSuccess] = React.useState<string | null>(null);
  const [fieldErrors, setFieldErrors] = React.useState<Record<string, string>>({});

  React.useEffect(() => {
    let cancelled = false;
    setLoading(true);
    getSettings()
      .then((settings) => {
        if (cancelled) return;
        setForm(settingsToForm(settings));
        setUpdatedAt(settings.updated_at);
      })
      .catch((err: unknown) => {
        if (!cancelled) setError(err instanceof Error ? err.message : 'Gagal memuat pengaturan.');
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, []);

  const updateField = <K extends keyof SettingsForm>(key: K, value: SettingsForm[K]) => {
    setForm((current) => (current ? { ...current, [key]: value } : current));
    setSuccess(null);
  };

  const updateTemplate = (key: keyof SettingsForm['wa_templates'], value: string) => {
    setForm((current) =>
      current ? { ...current, wa_templates: { ...current.wa_templates, [key]: value } } : current,
    );
    setSuccess(null);
  };

  const handleSave = async (event: React.FormEvent) => {
    event.preventDefault();
    if (!form) return;
    setSaving(true);
    setError(null);
    setSuccess(null);
    setFieldErrors({});
    try {
      const saved = await updateSettings(form);
      setForm(settingsToForm(saved));
      setUpdatedAt(saved.updated_at);
      setSuccess('Pengaturan berhasil disimpan.');
    } catch (err: unknown) {
      if (err instanceof ApiError) {
        setError(err.message);
        setFieldErrors(err.fieldErrors());
      } else {
        setError('Gagal menyimpan pengaturan.');
      }
    } finally {
      setSaving(false);
    }
  };

  const visibleTabs = TABS.filter((tab) => !tab.manageOnly || canManage);
  const effectiveTab = visibleTabs.some((tab) => tab.key === activeTab) ? activeTab : 'identity';
  const isFormTab = effectiveTab !== 'backup';

  return (
    <div className="space-y-5">
      <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <h1 className="text-2xl font-bold tracking-tight text-ink sm:text-[28px]">Pengaturan Toko</h1>
          <p className="mt-1 text-sm text-muted">
            Identitas, penomoran dokumen, garansi bawaan, dan template pesan WhatsApp.
          </p>
        </div>
        {updatedAt && <p className="text-xs text-muted">Terakhir diperbarui {formatDateTime(updatedAt)}</p>}
      </div>

      {success && <Alert variant="success">{success}</Alert>}
      {error && <Alert variant="error">{error}</Alert>}

      <div
        className="flex flex-wrap gap-1 rounded-control border border-border bg-surface p-1"
        role="tablist"
        aria-label="Bagian pengaturan"
      >
        {visibleTabs.map((tab) => (
          <button
            key={tab.key}
            type="button"
            role="tab"
            aria-selected={effectiveTab === tab.key}
            onClick={() => setActiveTab(tab.key)}
            className={cn(
              'rounded-control px-3 py-1.5 text-sm font-medium transition-colors',
              'focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-accent-500',
              effectiveTab === tab.key ? 'bg-brand-900 text-white shadow-sm' : 'text-muted hover:bg-background hover:text-ink',
            )}
          >
            {tab.label}
          </button>
        ))}
      </div>

      {loading ? (
        <Card className="space-y-4 p-6">
          {[1, 2, 3, 4].map((i) => (
            <div key={i} className="space-y-2">
              <Skeleton className="h-4 w-32" />
              <Skeleton className="h-10 w-full" />
            </div>
          ))}
        </Card>
      ) : !form ? (
        <EmptySettings error={error} />
      ) : effectiveTab === 'backup' ? (
        <BackupPanel />
      ) : (
        <form onSubmit={handleSave}>
          <Card className="space-y-5 p-5 sm:p-6">
            {effectiveTab === 'identity' && (
              <div className="space-y-4">
                <TextField
                  id="store-name"
                  label="Nama Toko *"
                  value={form.store_name}
                  onChange={(value) => updateField('store_name', value)}
                  error={fieldErrors.store_name}
                  disabled={!canManage}
                  maxLength={150}
                />
                <div className="grid gap-4 sm:grid-cols-2">
                  <TextField
                    id="legal-name"
                    label="Nama Badan Usaha"
                    value={form.legal_name}
                    onChange={(value) => updateField('legal_name', value)}
                    error={fieldErrors.legal_name}
                    disabled={!canManage}
                    maxLength={150}
                  />
                  <TextField
                    id="tax-id"
                    label="NPWP"
                    value={form.tax_id}
                    onChange={(value) => updateField('tax_id', value)}
                    error={fieldErrors.tax_id}
                    disabled={!canManage}
                    maxLength={30}
                  />
                  <TextField
                    id="phone"
                    label="Telepon"
                    value={form.phone}
                    onChange={(value) => updateField('phone', value)}
                    error={fieldErrors.phone}
                    disabled={!canManage}
                    maxLength={30}
                    placeholder="08xxxxxxxxxx"
                  />
                  <TextField
                    id="email"
                    label="Email"
                    type="email"
                    value={form.email}
                    onChange={(value) => updateField('email', value)}
                    error={fieldErrors.email}
                    disabled={!canManage}
                    maxLength={150}
                  />
                </div>
                <TextField
                  id="address"
                  label="Alamat"
                  value={form.address}
                  onChange={(value) => updateField('address', value)}
                  error={fieldErrors.address}
                  disabled={!canManage}
                  placeholder="Jl. ... No. ..., Kota, Kode Pos"
                />
                <div className="grid gap-4 sm:grid-cols-2">
                  <div className="space-y-1.5">
                    <Label htmlFor="timezone">Zona Waktu</Label>
                    <Input
                      id="timezone"
                      list="timezone-suggestions"
                      value={form.timezone}
                      onChange={(event) => updateField('timezone', event.target.value)}
                      disabled={!canManage}
                      placeholder="Asia/Jakarta"
                      maxLength={50}
                    />
                    <datalist id="timezone-suggestions">
                      <option value="Asia/Jakarta">WIB — Jakarta</option>
                      <option value="Asia/Makassar">WITA — Makassar</option>
                      <option value="Asia/Jayapura">WIT — Jayapura</option>
                    </datalist>
                    {fieldErrors.timezone ? (
                      <p className="text-xs text-danger">{fieldErrors.timezone}</p>
                    ) : (
                      <p className="text-xs text-muted">Dipakai untuk batas hari pada laporan dan nomor dokumen.</p>
                    )}
                  </div>
                </div>
              </div>
            )}

            {effectiveTab === 'documents' && (
              <div className="space-y-4">
                <div className="grid gap-4 sm:grid-cols-2">
                  <TextField
                    id="ticket-prefix"
                    label="Prefiks Tiket"
                    value={form.ticket_prefix}
                    onChange={(value) => updateField('ticket_prefix', value)}
                    error={fieldErrors.ticket_prefix}
                    disabled={!canManage}
                    maxLength={10}
                    placeholder="TCK"
                  />
                  <TextField
                    id="invoice-prefix"
                    label="Prefiks Invoice"
                    value={form.invoice_prefix}
                    onChange={(value) => updateField('invoice_prefix', value)}
                    error={fieldErrors.invoice_prefix}
                    disabled={!canManage}
                    maxLength={10}
                    placeholder="INV"
                  />
                </div>
                <div className="grid gap-4 sm:grid-cols-2">
                  <div className="space-y-1.5">
                    <Label htmlFor="ticket-format">Format Tiket</Label>
                    <Input
                      id="ticket-format"
                      value={form.ticket_format}
                      onChange={(event) => updateField('ticket_format', event.target.value)}
                      disabled={!canManage}
                      maxLength={60}
                      className="font-mono text-xs"
                      placeholder="{{prefix}}-{{date}}-{{seq}}"
                    />
                    {fieldErrors.ticket_format ? (
                      <p className="text-xs text-danger">{fieldErrors.ticket_format}</p>
                    ) : (
                      <p className="text-xs text-muted">Token: {'{{prefix}}'}, {'{{date}}'}, {'{{seq}}'}, {'{{outlet}}'}</p>
                    )}
                  </div>
                  <div className="space-y-1.5">
                    <Label htmlFor="invoice-format">Format Invoice</Label>
                    <Input
                      id="invoice-format"
                      value={form.invoice_format}
                      onChange={(event) => updateField('invoice_format', event.target.value)}
                      disabled={!canManage}
                      maxLength={60}
                      className="font-mono text-xs"
                      placeholder="{{prefix}}-{{date}}-{{seq}}"
                    />
                    {fieldErrors.invoice_format ? (
                      <p className="text-xs text-danger">{fieldErrors.invoice_format}</p>
                    ) : (
                      <p className="text-xs text-muted">Token: {'{{prefix}}'}, {'{{date}}'}, {'{{seq}}'}, {'{{outlet}}'}</p>
                    )}
                  </div>
                </div>
                <div className="space-y-1.5">
                  <Label htmlFor="invoice-terms">Syarat & Ketentuan Invoice</Label>
                  <Textarea
                    id="invoice-terms"
                    value={form.invoice_terms}
                    onChange={(event) => updateField('invoice_terms', event.target.value)}
                    disabled={!canManage}
                    rows={3}
                    placeholder="Contoh: Garansi servis 7 hari. Kerusakan lain di luar tanggung jawab toko."
                  />
                  {fieldErrors.invoice_terms && <p className="text-xs text-danger">{fieldErrors.invoice_terms}</p>}
                </div>
                <div className="space-y-1.5">
                  <Label htmlFor="invoice-footer">Catatan Kaki Invoice</Label>
                  <Textarea
                    id="invoice-footer"
                    value={form.invoice_footer}
                    onChange={(event) => updateField('invoice_footer', event.target.value)}
                    disabled={!canManage}
                    rows={2}
                    placeholder="Terima kasih telah mempercayakan perbaikan kepada kami."
                  />
                  {fieldErrors.invoice_footer && <p className="text-xs text-danger">{fieldErrors.invoice_footer}</p>}
                </div>
              </div>
            )}

            {effectiveTab === 'warranty' && (
              <div className="space-y-4">
                <div className="grid gap-4 sm:grid-cols-2">
                  <NumberField
                    id="service-warranty"
                    label="Garansi Jasa (hari)"
                    value={form.default_service_warranty_days}
                    onChange={(value) => updateField('default_service_warranty_days', value)}
                    error={fieldErrors.default_service_warranty_days}
                    disabled={!canManage}
                    min={0}
                    max={3650}
                    hint="Nilai bawaan saat membuat servis baru"
                  />
                  <NumberField
                    id="part-warranty"
                    label="Garansi Sparepart (hari)"
                    value={form.default_part_warranty_days}
                    onChange={(value) => updateField('default_part_warranty_days', value)}
                    error={fieldErrors.default_part_warranty_days}
                    disabled={!canManage}
                    min={0}
                    max={3650}
                    hint="Nilai bawaan untuk item sparepart"
                  />
                </div>
                <div className="space-y-1.5">
                  <Label htmlFor="receipt-width">Lebar Kertas Struk</Label>
                  <Select
                    id="receipt-width"
                    value={String(form.receipt_paper_width)}
                    onChange={(event) => updateField('receipt_paper_width', Number(event.target.value))}
                    disabled={!canManage}
                  >
                    <option value="58">58 mm — struk mini</option>
                    <option value="80">80 mm — struk lebar</option>
                  </Select>
                  {fieldErrors.receipt_paper_width && (
                    <p className="text-xs text-danger">{fieldErrors.receipt_paper_width}</p>
                  )}
                </div>
                <div className="grid gap-3 sm:grid-cols-2">
                  <ToggleField
                    id="technician-assigned-only"
                    label="Teknisi hanya melihat servis yang ditugaskan"
                    description="Membatasi daftar servis teknisi ke tiket yang ditugaskan padanya."
                    checked={form.technician_sees_only_assigned}
                    onChange={(checked) => updateField('technician_sees_only_assigned', checked)}
                    disabled={!canManage}
                  />
                  <ToggleField
                    id="public-tracking"
                    label="Aktifkan pelacakan publik"
                    description="Pelanggan dapat melacak status servis lewat halaman /track tanpa login."
                    checked={form.public_tracking_enabled}
                    onChange={(checked) => updateField('public_tracking_enabled', checked)}
                    disabled={!canManage}
                  />
                </div>
              </div>
            )}

            {effectiveTab === 'whatsapp' && (
              <div className="space-y-4">
                <div className="space-y-1.5">
                  <Label htmlFor="wa-provider">Provider WhatsApp</Label>
                  <Select
                    id="wa-provider"
                    value={form.wa_provider}
                    onChange={(event) => updateField('wa_provider', event.target.value)}
                    disabled={!canManage}
                  >
                    <option value="noop">{waProviderLabel('noop')}</option>
                    <option value="fonnte">{waProviderLabel('fonnte')}</option>
                    <option value="wablas">{waProviderLabel('wablas')}</option>
                  </Select>
                  {fieldErrors.wa_provider && <p className="text-xs text-danger">{fieldErrors.wa_provider}</p>}
                </div>
                <div className="space-y-4">
                  {(
                    [
                      ['status_update', 'Status Diperbarui', 'Terkirim saat status servis berubah.'],
                      ['ready_for_pickup', 'Siap Diambil', 'Terkirim saat servis siap diambil pelanggan.'],
                      ['invoice', 'Invoice Diterbitkan', 'Terkait saat invoice diterbitkan.'],
                      ['reminder', 'Pengingat', 'Pengingat untuk pelanggan yang belum mengambil perangkat.'],
                      ['warranty', 'Garansi Dibuat', 'Terkirim saat garansi baru dibuat.'],
                      ['claim_received', 'Klaim Diterima', 'Terkirim saat klaim garansi diterima.'],
                    ] as const
                  ).map(([key, label, description]) => (
                    <div key={key} className="space-y-1.5">
                      <Label htmlFor={`wa-${key}`}>{label}</Label>
                      <Textarea
                        id={`wa-${key}`}
                        value={form.wa_templates[key]}
                        onChange={(event) => updateTemplate(key, event.target.value)}
                        disabled={!canManage}
                        rows={3}
                        placeholder={`Halo {{customer_name}}, servis {{ticket_no}} sudah ${label.toLowerCase()}.`}
                      />
                      <p className="text-xs text-muted">{description}</p>
                    </div>
                  ))}
                </div>
              </div>
            )}

            {isFormTab && canManage && (
              <div className="flex justify-end border-t border-border pt-4">
                <Button type="submit" loading={saving} className="gap-2">
                  <Save className="h-4 w-4" aria-hidden="true" />
                  Simpan Perubahan
                </Button>
              </div>
            )}
            {isFormTab && !canManage && (
              <p className="border-t border-border pt-4 text-xs text-muted">
                Anda hanya dapat melihat pengaturan. Hubungi pemilik toko untuk melakukan perubahan.
              </p>
            )}
          </Card>
        </form>
      )}
    </div>
  );
}

function EmptySettings({ error }: { error: string | null }) {
  return (
    <Card className="flex flex-col items-center justify-center px-4 py-12 text-center">
      <span className="mb-3 flex h-11 w-11 items-center justify-center rounded-full bg-background text-muted">
        <SettingsIcon className="h-5 w-5" aria-hidden="true" />
      </span>
      <p className="text-sm font-medium text-ink">Pengaturan tidak dapat dimuat</p>
      <p className="mt-1 max-w-sm text-xs leading-5 text-muted">
        {error ?? 'Terjadi kesalahan saat memuat pengaturan toko. Coba muat ulang halaman.'}
      </p>
    </Card>
  );
}
