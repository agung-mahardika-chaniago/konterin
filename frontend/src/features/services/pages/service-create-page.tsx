import * as React from 'react';
import { useNavigate, useSearchParams } from 'react-router-dom';
import {
  ArrowLeft,
  Camera,
  CheckCircle2,
  KeyRound,
  Plus,
  Printer,
  Search,
  Smartphone,
  User,
  Wrench,
  X,
} from 'lucide-react';

import { Alert } from '@/components/ui/alert';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Select } from '@/components/ui/select';
import { Textarea } from '@/components/ui/textarea';
import { listCustomers } from '@/features/customers/api';
import type { Customer } from '@/features/customers/types';
import { PrintLabelModal } from '@/features/print/print-label-modal';
import { ApiError } from '@/lib/api-client';
import { useOutletStore } from '@/stores/outlet-store';
import { createServiceOrder, getTechnicians, uploadServicePhoto } from '../api';
import type { CreateServiceRequest, TechnicianOption } from '../types';

export function ServiceCreatePage() {
  const navigate = useNavigate();
  const [searchParams] = useSearchParams();
  const outletId = useOutletStore((state) => state.outletId);

  // Mode: existing vs new customer
  const [customerMode, setCustomerMode] = React.useState<'new' | 'search'>('new');
  const [customerSearch, setCustomerSearch] = React.useState('');
  const [customerResults, setCustomerResults] = React.useState<Customer[]>([]);
  const [searchingCustomer, setSearchingCustomer] = React.useState(false);
  const [selectedCustomer, setSelectedCustomer] = React.useState<Customer | null>(null);

  // Customer Form State
  const [custName, setCustName] = React.useState('');
  const [custPhone, setCustPhone] = React.useState('');
  const [custAddress, setCustAddress] = React.useState('');

  // Device Form State
  const [deviceBrand, setDeviceBrand] = React.useState('');
  const [deviceModel, setDeviceModel] = React.useState('');
  const [deviceImei, setDeviceImei] = React.useState('');
  const [deviceColor, setDeviceColor] = React.useState('');
  const [completenessItems, setCompletenessItems] = React.useState<string[]>(['Unit HP']);
  const [customCompleteness, setCustomCompleteness] = React.useState('');
  const [conditionItems, setConditionItems] = React.useState<string[]>([]);
  const [customCondition, setCustomCondition] = React.useState('');

  // Credential State
  const [credType, setCredType] = React.useState<'none' | 'pin' | 'password' | 'pattern'>('none');
  const [credValue, setCredValue] = React.useState('');

  // Complaint & Details State
  const [complaint, setComplaint] = React.useState('');
  const [estimatedCost, setEstimatedCost] = React.useState<string>('0');
  const [estimatedDoneAt, setEstimatedDoneAt] = React.useState('');
  const [assignedTechnicianId, setAssignedTechnicianId] = React.useState('');
  const [priority, setPriority] = React.useState<'normal' | 'express'>('normal');
  const [notes, setNotes] = React.useState('');
  const [internalNotes, setInternalNotes] = React.useState('');

  // Intake Photos
  const [selectedPhotos, setSelectedPhotos] = React.useState<{ file: File; preview: string }[]>([]);

  // Technicians List
  const [technicians, setTechnicians] = React.useState<TechnicianOption[]>([]);

  // Form State
  const [loading, setLoading] = React.useState(false);
  const [error, setError] = React.useState<string | null>(null);
  const [fieldErrors, setFieldErrors] = React.useState<Record<string, string>>({});

  // Success State & Print Modal
  const [createdOrder, setCreatedOrder] = React.useState<{
    id: string;
    ticket_no: string;
    tracking_url: string;
    customer_name: string;
    device_name: string;
    received_at: string;
  } | null>(null);
  const [printModalOpen, setPrintModalOpen] = React.useState(false);

  // Load Technicians
  React.useEffect(() => {
    getTechnicians(outletId ?? undefined)
      .then(setTechnicians)
      .catch((err) => console.error('Error fetching technicians', err));
  }, [outletId]);

  // Handle URL param customer_id
  React.useEffect(() => {
    const custIdParam = searchParams.get('customer_id');
    if (custIdParam) {
      listCustomers({ q: custIdParam }).then((res) => {
        const found = res.data?.find((c) => c.id === custIdParam);
        if (found) {
          setSelectedCustomer(found);
          setCustomerMode('search');
        }
      });
    }
  }, [searchParams]);

  // Debounced search for customers
  React.useEffect(() => {
    if (customerMode !== 'search' || !customerSearch.trim()) {
      setCustomerResults([]);
      return;
    }
    const timer = setTimeout(async () => {
      setSearchingCustomer(true);
      try {
        const res = await listCustomers({ q: customerSearch.trim(), per_page: 5 });
        setCustomerResults(res.data ?? []);
      } catch (err) {
        console.error('Customer search error', err);
      } finally {
        setSearchingCustomer(false);
      }
    }, 300);
    return () => clearTimeout(timer);
  }, [customerSearch, customerMode]);

  const handleSelectCustomer = (c: Customer) => {
    setSelectedCustomer(c);
    setCustName(c.name);
    setCustPhone(c.phone);
    setCustAddress(c.address || '');
    setCustomerResults([]);
    setCustomerSearch('');
  };

  const handleClearSelectedCustomer = () => {
    setSelectedCustomer(null);
    setCustName('');
    setCustPhone('');
    setCustAddress('');
  };

  const handleToggleCompleteness = (item: string) => {
    setCompletenessItems((prev) =>
      prev.includes(item) ? prev.filter((i) => i !== item) : [...prev, item],
    );
  };

  const handleToggleCondition = (item: string) => {
    setConditionItems((prev) =>
      prev.includes(item) ? prev.filter((i) => i !== item) : [...prev, item],
    );
  };

  const handlePhotoSelect = (e: React.ChangeEvent<HTMLInputElement>) => {
    const files = Array.from(e.target.files || []);
    if (files.length === 0) return;

    const newPhotos = files.map((file) => ({
      file,
      preview: URL.createObjectURL(file),
    }));

    setSelectedPhotos((prev) => [...prev, ...newPhotos].slice(0, 5));
  };

  const handleRemovePhoto = (index: number) => {
    setSelectedPhotos((prev) => {
      URL.revokeObjectURL(prev[index].preview);
      return prev.filter((_, i) => i !== index);
    });
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setLoading(true);
    setError(null);
    setFieldErrors({});

    // Build completeness text
    const fullCompleteness = [
      ...completenessItems,
      customCompleteness.trim(),
    ]
      .filter(Boolean)
      .join(', ');

    // Build physical condition text
    const fullCondition = [
      ...conditionItems,
      customCondition.trim(),
    ]
      .filter(Boolean)
      .join(', ');

    const payload: CreateServiceRequest = {
      customer: {
        id: selectedCustomer ? selectedCustomer.id : undefined,
        name: selectedCustomer ? selectedCustomer.name : custName.trim(),
        phone: selectedCustomer ? selectedCustomer.phone : custPhone.trim(),
        address: selectedCustomer ? selectedCustomer.address : custAddress.trim() || undefined,
      },
      device: {
        brand: deviceBrand.trim(),
        model: deviceModel.trim(),
        imei: deviceImei.trim() || undefined,
        color: deviceColor.trim() || undefined,
      },
      complaint: complaint.trim(),
      completeness: fullCompleteness || undefined,
      physical_condition: fullCondition || undefined,
      credential:
        credType !== 'none' && credValue.trim()
          ? {
              type: credType,
              value: credValue.trim(),
            }
          : undefined,
      estimated_cost: Number(estimatedCost) || 0,
      estimated_done_at: estimatedDoneAt ? new Date(estimatedDoneAt).toISOString() : undefined,
      assigned_technician_id: assignedTechnicianId || undefined,
      priority,
      notes: notes.trim() || undefined,
      internal_notes: internalNotes.trim() || undefined,
    };

    try {
      const created = await createServiceOrder(payload);

      // Upload selected photos in parallel if any
      if (selectedPhotos.length > 0) {
        await Promise.allSettled(
          selectedPhotos.map((p) => uploadServicePhoto(created.id, p.file, 'intake', true)),
        );
      }

      setCreatedOrder({
        id: created.id,
        ticket_no: created.ticket_no,
        tracking_url: created.tracking_url,
        customer_name: payload.customer.name,
        device_name: `${payload.device.brand} ${payload.device.model}`,
        received_at: new Date().toISOString(),
      });
    } catch (err: unknown) {
      if (err instanceof ApiError) {
        setError(err.message);
        setFieldErrors(err.fieldErrors());
      } else {
        setError('Gagal membuat tiket penerimaan servis.');
      }
    } finally {
      setLoading(false);
    }
  };

  // SUCCESS SCREEN
  if (createdOrder) {
    const dummyListItem: any = {
      id: createdOrder.id,
      ticket_no: createdOrder.ticket_no,
      status: 'received',
      priority,
      customer: {
        id: '',
        name: createdOrder.customer_name,
        phone: custPhone,
      },
      device: {
        id: '',
        brand: deviceBrand,
        model: deviceModel,
        color: deviceColor,
      },
      complaint,
      estimated_cost: Number(estimatedCost) || 0,
      received_at: createdOrder.received_at,
    };

    return (
      <div className="max-w-2xl mx-auto py-8 text-center space-y-6">
        <div className="flex flex-col items-center">
          <div className="h-16 w-16 rounded-full bg-emerald-100 text-emerald-600 flex items-center justify-center mb-4">
            <CheckCircle2 className="h-10 w-10" aria-hidden="true" />
          </div>
          <h1 className="text-2xl font-bold tracking-tight text-ink">Penerimaan Servis Berhasil!</h1>
          <p className="mt-1 text-sm text-muted">
            Tiket servis telah dibuat secara atomik dan tercatat dalam sistem.
          </p>
        </div>

        <Card className="p-6 text-left space-y-4 bg-surface">
          <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between border-b border-border pb-4 gap-2">
            <div>
              <span className="text-xs uppercase font-semibold text-muted">Nomor Tiket</span>
              <p className="text-2xl font-bold font-mono text-ink">{createdOrder.ticket_no}</p>
            </div>
            <div className="flex items-center gap-2">
              <Button onClick={() => setPrintModalOpen(true)} className="gap-2">
                <Printer className="h-4 w-4" aria-hidden="true" />
                Cetak Label Tiket
              </Button>
            </div>
          </div>

          <div className="grid grid-cols-2 gap-4 text-sm">
            <div>
              <span className="text-xs text-muted block">Pelanggan</span>
              <span className="font-semibold text-ink">{createdOrder.customer_name}</span>
            </div>
            <div>
              <span className="text-xs text-muted block">Perangkat</span>
              <span className="font-semibold text-ink">{createdOrder.device_name}</span>
            </div>
          </div>
        </Card>

        <div className="flex flex-col sm:flex-row items-center justify-center gap-3">
          <Button variant="outline" onClick={() => navigate('/services')} className="w-full sm:w-auto">
            Kembali ke Daftar Servis
          </Button>
          <Button
            variant="secondary"
            onClick={() => {
              setCreatedOrder(null);
              setSelectedCustomer(null);
              setCustName('');
              setCustPhone('');
              setCustAddress('');
              setDeviceBrand('');
              setDeviceModel('');
              setDeviceImei('');
              setDeviceColor('');
              setComplaint('');
              setEstimatedCost('0');
              setEstimatedDoneAt('');
              setSelectedPhotos([]);
            }}
            className="w-full sm:w-auto gap-1.5"
          >
            <Plus className="h-4 w-4" aria-hidden="true" />
            Terima Servis Lagi
          </Button>
          <Button onClick={() => navigate(`/services/${createdOrder.id}`)} className="w-full sm:w-auto">
            Lihat Detail Servis
          </Button>
        </div>

        <PrintLabelModal
          open={printModalOpen}
          onOpenChange={setPrintModalOpen}
          order={dummyListItem}
        />
      </div>
    );
  }

  return (
    <div className="max-w-4xl mx-auto space-y-6 pb-12">
      {/* Top Header */}
      <div className="flex items-center gap-3">
        <Button
          variant="outline"
          size="sm"
          onClick={() => navigate('/services')}
          className="h-9 w-9 p-0 rounded-full"
          title="Kembali"
        >
          <ArrowLeft className="h-4 w-4" aria-hidden="true" />
        </Button>
        <div>
          <h1 className="text-2xl font-bold tracking-tight text-ink sm:text-[28px]">Penerimaan Servis Baru</h1>
          <p className="mt-0.5 text-xs text-muted">
            Isi formulir penerimaan unit pelanggan untuk menerbitkan nomor tiket servis.
          </p>
        </div>
      </div>

      {error && <Alert variant="error">{error}</Alert>}

      <form onSubmit={handleSubmit} className="space-y-6">
        {/* Section 1: Customer */}
        <Card>
          <CardHeader className="flex flex-row items-center justify-between pb-3">
            <div className="flex items-center gap-2">
              <User className="h-5 w-5 text-accent-600" aria-hidden="true" />
              <CardTitle className="text-base font-semibold">1. Data Pelanggan</CardTitle>
            </div>
            <div className="flex rounded-md border border-border p-0.5 bg-background text-xs">
              <button
                type="button"
                onClick={() => {
                  setCustomerMode('new');
                  setSelectedCustomer(null);
                }}
                className={`px-3 py-1 font-medium rounded transition-colors ${
                  customerMode === 'new' ? 'bg-surface text-ink shadow-sm' : 'text-muted hover:text-ink'
                }`}
              >
                Pelanggan Baru
              </button>
              <button
                type="button"
                onClick={() => setCustomerMode('search')}
                className={`px-3 py-1 font-medium rounded transition-colors ${
                  customerMode === 'search' ? 'bg-surface text-ink shadow-sm' : 'text-muted hover:text-ink'
                }`}
              >
                Cari Pelanggan Lama
              </button>
            </div>
          </CardHeader>
          <CardContent className="space-y-4">
            {customerMode === 'search' && !selectedCustomer && (
              <div className="relative">
                <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted" aria-hidden="true" />
                <Input
                  value={customerSearch}
                  onChange={(e) => setCustomerSearch(e.target.value)}
                  placeholder="Ketik nama atau nomor HP pelanggan..."
                  className="pl-9"
                  autoFocus
                />
                {searchingCustomer && (
                  <p className="text-xs text-muted mt-1 italic">Mencari database pelanggan...</p>
                )}
                {customerResults.length > 0 && (
                  <div className="absolute z-10 mt-1 w-full rounded-md border border-border bg-surface shadow-lg divide-y divide-border">
                    {customerResults.map((c) => (
                      <button
                        key={c.id}
                        type="button"
                        onClick={() => handleSelectCustomer(c)}
                        className="w-full text-left p-3 hover:bg-background transition-colors flex items-center justify-between"
                      >
                        <div>
                          <p className="font-semibold text-sm text-ink">{c.name}</p>
                          <p className="text-xs text-muted">{c.phone}</p>
                        </div>
                        <span className="text-xs text-accent-600 font-medium">Pilih &rarr;</span>
                      </button>
                    ))}
                  </div>
                )}
              </div>
            )}

            {selectedCustomer ? (
              <div className="flex items-center justify-between p-3 rounded-lg border border-accent-200 bg-accent-50/40">
                <div className="space-y-0.5">
                  <span className="text-[11px] font-semibold text-accent-700 uppercase">Pelanggan Terpilih</span>
                  <p className="font-bold text-ink">{selectedCustomer.name}</p>
                  <p className="text-xs text-muted">{selectedCustomer.phone}</p>
                  {selectedCustomer.address && <p className="text-xs text-slate-600">{selectedCustomer.address}</p>}
                </div>
                <Button
                  type="button"
                  variant="outline"
                  size="sm"
                  onClick={handleClearSelectedCustomer}
                  className="text-xs h-8"
                >
                  Ganti
                </Button>
              </div>
            ) : (
              <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
                <div>
                  <Label htmlFor="create-cust-name">
                    Nama Pelanggan <span className="text-danger">*</span>
                  </Label>
                  <Input
                    id="create-cust-name"
                    value={custName}
                    onChange={(e) => setCustName(e.target.value)}
                    placeholder="Contoh: Siti Rahma"
                    required
                    className="mt-1"
                  />
                  {fieldErrors['customer.name'] && (
                    <p className="mt-1 text-xs text-danger">{fieldErrors['customer.name']}</p>
                  )}
                </div>
                <div>
                  <Label htmlFor="create-cust-phone">
                    Nomor WhatsApp / HP <span className="text-danger">*</span>
                  </Label>
                  <Input
                    id="create-cust-phone"
                    value={custPhone}
                    onChange={(e) => setCustPhone(e.target.value)}
                    placeholder="0812xxxxxxxx"
                    required
                    className="mt-1 font-mono"
                  />
                  {fieldErrors['customer.phone'] && (
                    <p className="mt-1 text-xs text-danger">{fieldErrors['customer.phone']}</p>
                  )}
                </div>
                <div className="sm:col-span-2">
                  <Label htmlFor="create-cust-addr">Alamat (Opsional)</Label>
                  <Input
                    id="create-cust-addr"
                    value={custAddress}
                    onChange={(e) => setCustAddress(e.target.value)}
                    placeholder="Alamat tempat tinggal / kantor"
                    className="mt-1"
                  />
                </div>
              </div>
            )}
          </CardContent>
        </Card>

        {/* Section 2: Device */}
        <Card>
          <CardHeader className="flex flex-row items-center gap-2 pb-3">
            <Smartphone className="h-5 w-5 text-accent-600" aria-hidden="true" />
            <CardTitle className="text-base font-semibold">2. Data Perangkat</CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
              <div>
                <Label htmlFor="dev-brand">
                  Merek HP <span className="text-danger">*</span>
                </Label>
                <Input
                  id="dev-brand"
                  value={deviceBrand}
                  onChange={(e) => setDeviceBrand(e.target.value)}
                  placeholder="Samsung, Apple, Xiaomi, Oppo, Vivo, dll."
                  required
                  className="mt-1"
                />
              </div>
              <div>
                <Label htmlFor="dev-model">
                  Model / Tipe HP <span className="text-danger">*</span>
                </Label>
                <Input
                  id="dev-model"
                  value={deviceModel}
                  onChange={(e) => setDeviceModel(e.target.value)}
                  placeholder="Galaxy S22 Ultra, iPhone 13 Pro, Redmi Note 12"
                  required
                  className="mt-1"
                />
              </div>
              <div>
                <Label htmlFor="dev-imei">Nomor Seri / IMEI (Opsional)</Label>
                <Input
                  id="dev-imei"
                  value={deviceImei}
                  onChange={(e) => setDeviceImei(e.target.value)}
                  placeholder="15 digit IMEI atau nomor seri"
                  className="mt-1 font-mono"
                />
              </div>
              <div>
                <Label htmlFor="dev-color">Warna Perangkat</Label>
                <Input
                  id="dev-color"
                  value={deviceColor}
                  onChange={(e) => setDeviceColor(e.target.value)}
                  placeholder="Phantom Black, Alpine Green, Silver"
                  className="mt-1"
                />
              </div>
            </div>

            {/* Completeness Checklist */}
            <div>
              <Label className="block mb-2">Kelengkapan yang Disertakan</Label>
              <div className="flex flex-wrap gap-2">
                {['Unit HP', 'Baterai', 'Casing/Softcase', 'SIM Card/Tray', 'Charger/Kabel'].map((item) => (
                  <button
                    key={item}
                    type="button"
                    onClick={() => handleToggleCompleteness(item)}
                    className={`px-3 py-1.5 rounded-full text-xs font-medium border transition-colors ${
                      completenessItems.includes(item)
                        ? 'border-brand-900 bg-brand-900 text-white'
                        : 'border-border bg-background text-muted hover:text-ink'
                    }`}
                  >
                    {completenessItems.includes(item) ? '✓ ' : '+ '}
                    {item}
                  </button>
                ))}
              </div>
              <Input
                value={customCompleteness}
                onChange={(e) => setCustomCompleteness(e.target.value)}
                placeholder="Kelengkapan lain (contoh: dus box, kartu memori SD)..."
                className="mt-2 text-xs"
              />
            </div>

            {/* Physical Condition Checklist */}
            <div>
              <Label className="block mb-2">Kondisi Fisik Saat Diterima</Label>
              <div className="flex flex-wrap gap-2">
                {[
                  'Mulus Normal',
                  'Layar Retak / Pecah',
                  'Body Lecet / Baret',
                  'Frame Bengkok',
                  'Bekas Terkena Air',
                  'Baterai Kembung',
                ].map((item) => (
                  <button
                    key={item}
                    type="button"
                    onClick={() => handleToggleCondition(item)}
                    className={`px-3 py-1.5 rounded-full text-xs font-medium border transition-colors ${
                      conditionItems.includes(item)
                        ? 'border-amber-600 bg-amber-50 text-amber-900 font-semibold'
                        : 'border-border bg-background text-muted hover:text-ink'
                    }`}
                  >
                    {conditionItems.includes(item) ? '✓ ' : '+ '}
                    {item}
                  </button>
                ))}
              </div>
              <Input
                value={customCondition}
                onChange={(e) => setCustomCondition(e.target.value)}
                placeholder="Keterangan fisik lain (contoh: tombol volume macet, bezel mengelupas)..."
                className="mt-2 text-xs"
              />
            </div>
          </CardContent>
        </Card>

        {/* Section 3: Credential (Write-Only PIN/Pattern) */}
        <Card>
          <CardHeader className="flex flex-row items-center gap-2 pb-3">
            <KeyRound className="h-5 w-5 text-accent-600" aria-hidden="true" />
            <div>
              <CardTitle className="text-base font-semibold">3. Kredensial Layar Kunci</CardTitle>
              <p className="text-xs text-muted mt-0.5">
                Dienkripsi otomatis dengan AES-256. Dibutuhkan untuk pengujian fungsi layar & kamera.
              </p>
            </div>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
              <div>
                <Label htmlFor="cred-type">Tipe Kunci Layar</Label>
                <Select
                  id="cred-type"
                  value={credType}
                  onChange={(e) => setCredType(e.target.value as any)}
                  className="mt-1"
                >
                  <option value="none">Tidak Ada Kunci Layar / Tanpa Password</option>
                  <option value="pin">PIN Angka</option>
                  <option value="password">Password / Kata Sandi Teks</option>
                  <option value="pattern">Pola Sentuh (Format: 1-2-5-8)</option>
                </Select>
              </div>
              {credType !== 'none' && (
                <div>
                  <Label htmlFor="cred-value">
                    Nilai PIN / Pola / Sandi <span className="text-danger">*</span>
                  </Label>
                  <Input
                    id="cred-value"
                    type={credType === 'password' ? 'password' : 'text'}
                    value={credValue}
                    onChange={(e) => setCredValue(e.target.value)}
                    placeholder={
                      credType === 'pin' ? 'Contoh: 1234' : credType === 'pattern' ? 'Contoh: 1-2-3-6-9' : 'Kata sandi'
                    }
                    required
                    className="mt-1 font-mono"
                  />
                </div>
              )}
            </div>
          </CardContent>
        </Card>

        {/* Section 4: Complaint & Estimates */}
        <Card>
          <CardHeader className="flex flex-row items-center gap-2 pb-3">
            <Wrench className="h-5 w-5 text-accent-600" aria-hidden="true" />
            <CardTitle className="text-base font-semibold">4. Keluhan & Penugasan</CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            <div>
              <Label htmlFor="complaint-text">
                Keluhan / Kerusakan Utama <span className="text-danger">*</span>
              </Label>
              <Textarea
                id="complaint-text"
                value={complaint}
                onChange={(e) => setComplaint(e.target.value)}
                placeholder="Jelaskan masalah perangkat secara detail (contoh: layar blank hitam setelah jatuh, charger tidak merespons)..."
                required
                className="mt-1"
                rows={3}
              />
            </div>

            <div className="grid grid-cols-1 sm:grid-cols-3 gap-4">
              <div>
                <Label htmlFor="est-cost">Estimasi Biaya Awal (Rp)</Label>
                <Input
                  id="est-cost"
                  type="number"
                  min="0"
                  step="1000"
                  value={estimatedCost}
                  onChange={(e) => setEstimatedCost(e.target.value)}
                  className="mt-1 font-mono"
                />
              </div>
              <div>
                <Label htmlFor="est-done">Estimasi Selesai</Label>
                <Input
                  id="est-done"
                  type="datetime-local"
                  value={estimatedDoneAt}
                  onChange={(e) => setEstimatedDoneAt(e.target.value)}
                  className="mt-1 text-xs"
                />
              </div>
              <div>
                <Label htmlFor="priority-sel">Prioritas Pengerjaan</Label>
                <Select
                  id="priority-sel"
                  value={priority}
                  onChange={(e) => setPriority(e.target.value as 'normal' | 'express')}
                  className="mt-1"
                >
                  <option value="normal">Normal</option>
                  <option value="express">⚡ Express (Prioritas)</option>
                </Select>
              </div>
            </div>

            <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
              <div>
                <Label htmlFor="assigned-tech">Tugaskan ke Teknisi</Label>
                <Select
                  id="assigned-tech"
                  value={assignedTechnicianId}
                  onChange={(e) => setAssignedTechnicianId(e.target.value)}
                  className="mt-1"
                >
                  <option value="">Belum Ditugaskan (Menunggu Antrean)</option>
                  {technicians.map((t) => (
                    <option key={t.id} value={t.id}>
                      {t.name}
                    </option>
                  ))}
                </Select>
              </div>
              <div>
                <Label htmlFor="cust-notes">Catatan Pelanggan</Label>
                <Input
                  id="cust-notes"
                  value={notes}
                  onChange={(e) => setNotes(e.target.value)}
                  placeholder="Contoh: minta dihubungi via WA saat sparepart tiba"
                  className="mt-1"
                />
              </div>
            </div>

            <div>
              <Label htmlFor="int-notes">Catatan Internal Teknisi (Hanya Dilihat Staf)</Label>
              <Textarea
                id="int-notes"
                value={internalNotes}
                onChange={(e) => setInternalNotes(e.target.value)}
                placeholder="Catatan rahasia teknisi, riwayat baut kendor, dll."
                className="mt-1"
                rows={2}
              />
            </div>
          </CardContent>
        </Card>

        {/* Section 5: Intake Photos */}
        <Card>
          <CardHeader className="flex flex-row items-center gap-2 pb-3">
            <Camera className="h-5 w-5 text-accent-600" aria-hidden="true" />
            <div>
              <CardTitle className="text-base font-semibold">5. Foto Kondisi Intake (Opsional)</CardTitle>
              <p className="text-xs text-muted mt-0.5">
                Ambil foto tampak depan, belakang, atau titik retak fisik saat unit diterima (maks. 5 foto).
              </p>
            </div>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="flex flex-wrap items-center gap-3">
              {selectedPhotos.map((photo, idx) => (
                <div key={idx} className="relative group w-24 h-24 rounded-lg border border-border overflow-hidden bg-black/5">
                  <img src={photo.preview} alt="Intake" className="w-full h-full object-cover" />
                  <button
                    type="button"
                    onClick={() => handleRemovePhoto(idx)}
                    className="absolute top-1 right-1 p-1 rounded-full bg-red-600 text-white shadow hover:bg-red-700 transition-colors"
                    aria-label="Hapus foto"
                  >
                    <X className="h-3 w-3" />
                  </button>
                </div>
              ))}

              {selectedPhotos.length < 5 && (
                <label
                  htmlFor="intake-photo-input"
                  className="w-24 h-24 rounded-lg border-2 border-dashed border-border flex flex-col items-center justify-center cursor-pointer hover:border-accent-500 hover:bg-background transition-colors text-muted hover:text-ink"
                >
                  <Camera className="h-6 w-6 mb-1" aria-hidden="true" />
                  <span className="text-[10px] font-medium">+ Foto</span>
                  <input
                    id="intake-photo-input"
                    type="file"
                    accept="image/jpeg,image/png,image/webp"
                    multiple
                    onChange={handlePhotoSelect}
                    className="sr-only"
                  />
                </label>
              )}
            </div>
          </CardContent>
        </Card>

        {/* Submit Actions */}
        <div className="flex items-center justify-end gap-3 pt-4 border-t border-border sticky bottom-0 bg-surface/90 backdrop-blur p-4 rounded-xl shadow-lg border">
          <Button type="button" variant="outline" onClick={() => navigate('/services')} disabled={loading}>
            Batal
          </Button>
          <Button type="submit" loading={loading} className="gap-2 px-6">
            <Plus className="h-4 w-4" aria-hidden="true" />
            Terbitkan Tiket Servis
          </Button>
        </div>
      </form>
    </div>
  );
}
