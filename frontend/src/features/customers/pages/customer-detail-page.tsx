import * as React from 'react';
import { Link, useNavigate, useParams } from 'react-router-dom';
import {
  ArrowLeft,
  Calendar,
  MessageCircle,
  Pencil,
  Phone,
  Plus,
  Smartphone,
  Wallet,
  Wrench,
} from 'lucide-react';

import { EmptyState } from '@/components/shared/empty-state';
import { Alert } from '@/components/ui/alert';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Skeleton } from '@/components/ui/skeleton';
import { formatCurrency } from '@/features/dashboard/format';
import { StatusBadge } from '@/features/services/components/status-badge';
import type { ServiceOrderListItem } from '@/features/services/types';
import { getCustomer, listCustomerDevices, listCustomerServices } from '../api';
import { CustomerDeviceDialog } from '../components/customer-device-dialog';
import { CustomerFormDialog } from '../components/customer-form-dialog';
import type { Customer, Device } from '../types';

export function CustomerDetailPage() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();

  const [customer, setCustomer] = React.useState<Customer | null>(null);
  const [devices, setDevices] = React.useState<Device[]>([]);
  const [services, setServices] = React.useState<ServiceOrderListItem[]>([]);
  const [loading, setLoading] = React.useState(true);
  const [error, setError] = React.useState<string | null>(null);

  // Modals
  const [editCustomerOpen, setEditCustomerOpen] = React.useState(false);
  const [addDeviceOpen, setAddDeviceOpen] = React.useState(false);

  const loadAll = React.useCallback(async () => {
    if (!id) return;
    setLoading(true);
    setError(null);
    try {
      const [custData, devData, srvData] = await Promise.all([
        getCustomer(id),
        listCustomerDevices(id),
        listCustomerServices(id),
      ]);
      setCustomer(custData);
      setDevices(devData ?? []);
      setServices(srvData.data ?? []);
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Gagal memuat rincian pelanggan.');
    } finally {
      setLoading(false);
    }
  }, [id]);

  React.useEffect(() => {
    loadAll();
  }, [loadAll]);

  if (loading) {
    return (
      <div className="space-y-5">
        <Skeleton className="h-8 w-40" />
        <Skeleton className="h-32 w-full" />
        <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
          <Skeleton className="h-24" />
          <Skeleton className="h-24" />
          <Skeleton className="h-24" />
        </div>
        <Skeleton className="h-64 w-full" />
      </div>
    );
  }

  if (error || !customer) {
    return (
      <div className="space-y-4">
        <Button variant="ghost" onClick={() => navigate('/customers')} className="gap-2">
          <ArrowLeft className="h-4 w-4" aria-hidden="true" />
          Kembali ke Pelanggan
        </Button>
        <Alert variant="error">{error || 'Data pelanggan tidak ditemukan.'}</Alert>
      </div>
    );
  }

  const cleanPhone = customer.phone.replace(/\D/g, '');
  const waUrl = `https://wa.me/${cleanPhone.startsWith('0') ? '62' + cleanPhone.slice(1) : cleanPhone}`;

  return (
    <div className="space-y-6">
      {/* Back button and title */}
      <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        <div className="flex items-center gap-3">
          <Button
            variant="outline"
            size="sm"
            onClick={() => navigate('/customers')}
            className="h-9 w-9 p-0 rounded-full"
            title="Kembali"
          >
            <ArrowLeft className="h-4 w-4" aria-hidden="true" />
          </Button>
          <div>
            <h1 className="text-2xl font-bold tracking-tight text-ink">{customer.name}</h1>
            <p className="text-xs text-muted">ID Pelanggan: {customer.id}</p>
          </div>
        </div>

        <div className="flex flex-wrap items-center gap-2">
          <a href={waUrl} target="_blank" rel="noopener noreferrer">
            <Button variant="outline" size="sm" className="gap-1.5 text-emerald-700 hover:text-emerald-800">
              <MessageCircle className="h-4 w-4" aria-hidden="true" />
              Chat WhatsApp
            </Button>
          </a>
          <Button variant="outline" size="sm" onClick={() => setEditCustomerOpen(true)} className="gap-1.5">
            <Pencil className="h-4 w-4" aria-hidden="true" />
            Ubah Data
          </Button>
          <Link to={`/services/new?customer_id=${customer.id}`}>
            <Button size="sm" className="gap-1.5">
              <Plus className="h-4 w-4" aria-hidden="true" />
              Servis Baru
            </Button>
          </Link>
        </div>
      </div>

      {/* Overview Card */}
      <Card>
        <CardContent className="p-6">
          <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
            <div className="space-y-1">
              <span className="text-xs font-semibold uppercase tracking-wider text-muted">Kontak & Telepon</span>
              <p className="font-semibold text-ink flex items-center gap-2">
                <Phone className="h-4 w-4 text-muted" aria-hidden="true" />
                {customer.phone}
              </p>
            </div>
            <div className="space-y-1">
              <span className="text-xs font-semibold uppercase tracking-wider text-muted">Alamat</span>
              <p className="text-sm text-ink">{customer.address || <span className="italic text-muted">Belum ada alamat</span>}</p>
            </div>
            <div className="space-y-1">
              <span className="text-xs font-semibold uppercase tracking-wider text-muted">Catatan Khusus</span>
              <p className="text-sm text-ink">{customer.notes || <span className="italic text-muted">Tidak ada catatan</span>}</p>
            </div>
          </div>
        </CardContent>
      </Card>

      {/* Stats Cards */}
      <div className="grid grid-cols-1 sm:grid-cols-3 gap-4">
        <Card>
          <CardContent className="p-5 flex items-center gap-4">
            <div className="h-11 w-11 rounded-lg bg-blue-50 text-blue-600 flex items-center justify-center shrink-0">
              <Wrench className="h-5 w-5" aria-hidden="true" />
            </div>
            <div>
              <p className="text-xs text-muted">Total Servis</p>
              <p className="text-xl font-bold text-ink">{customer.stats?.total_services ?? services.length}</p>
            </div>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="p-5 flex items-center gap-4">
            <div className="h-11 w-11 rounded-lg bg-emerald-50 text-emerald-600 flex items-center justify-center shrink-0">
              <Wallet className="h-5 w-5" aria-hidden="true" />
            </div>
            <div>
              <p className="text-xs text-muted">Total Transaksi</p>
              <p className="text-xl font-bold text-ink">{formatCurrency(customer.stats?.total_paid ?? 0)}</p>
            </div>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="p-5 flex items-center gap-4">
            <div className="h-11 w-11 rounded-lg bg-purple-50 text-purple-600 flex items-center justify-center shrink-0">
              <Calendar className="h-5 w-5" aria-hidden="true" />
            </div>
            <div>
              <p className="text-xs text-muted">Servis Terakhir</p>
              <p className="text-sm font-bold text-ink">
                {customer.stats?.last_service_at
                  ? new Date(customer.stats.last_service_at).toLocaleDateString('id-ID', {
                      day: 'numeric',
                      month: 'short',
                      year: 'numeric',
                    })
                  : '-'}
              </p>
            </div>
          </CardContent>
        </Card>
      </div>

      {/* Perangkat Terdaftar */}
      <Card>
        <CardHeader className="flex flex-row items-center justify-between pb-3">
          <div>
            <CardTitle className="text-base font-semibold">Perangkat Terdaftar</CardTitle>
            <p className="text-xs text-muted mt-0.5">Daftar smartphone atau gadget milik pelanggan ini.</p>
          </div>
          <Button size="sm" variant="outline" onClick={() => setAddDeviceOpen(true)} className="gap-1.5">
            <Plus className="h-3.5 w-3.5" aria-hidden="true" />
            Tambah Perangkat
          </Button>
        </CardHeader>
        <CardContent>
          {devices.length === 0 ? (
            <EmptyState
              icon={Smartphone}
              title="Belum ada perangkat terdaftar"
              description="Perangkat otomatis ditambahkan saat membuat order servis atau tambah manual di sini."
              className="py-6"
            />
          ) : (
            <div className="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-3 gap-3">
              {devices.map((dev) => (
                <div key={dev.id} className="rounded-lg border border-border p-4 bg-background/40 hover:bg-background transition-colors">
                  <div className="flex items-center gap-2">
                    <Smartphone className="h-4 w-4 text-accent-600 shrink-0" aria-hidden="true" />
                    <p className="font-semibold text-sm text-ink">{dev.brand} {dev.model}</p>
                  </div>
                  {dev.color && <p className="text-xs text-muted mt-1">Warna: {dev.color}</p>}
                  {dev.imei && <p className="text-xs text-muted font-mono mt-0.5">IMEI: {dev.imei}</p>}
                  {dev.notes && <p className="text-xs text-slate-500 italic mt-2 border-t border-border pt-1.5">{dev.notes}</p>}
                </div>
              ))}
            </div>
          )}
        </CardContent>
      </Card>

      {/* Riwayat Servis */}
      <Card>
        <CardHeader className="pb-3">
          <CardTitle className="text-base font-semibold">Riwayat Servis</CardTitle>
          <p className="text-xs text-muted mt-0.5">Seluruh pesanan reparasi pelanggan di outlet ini.</p>
        </CardHeader>
        <CardContent className="p-0">
          {services.length === 0 ? (
            <EmptyState
              icon={Wrench}
              title="Belum ada riwayat servis"
              description="Pelanggan ini belum pernah melakukan servis HP."
              className="py-8"
            />
          ) : (
            <div className="overflow-x-auto">
              <table className="w-full text-left text-sm">
                <thead className="bg-background text-xs uppercase tracking-wider text-muted border-y border-border">
                  <tr>
                    <th className="px-5 py-3 font-semibold">Tiket & Tanggal</th>
                    <th className="px-5 py-3 font-semibold">Perangkat</th>
                    <th className="px-5 py-3 font-semibold">Keluhan</th>
                    <th className="px-5 py-3 font-semibold">Status</th>
                    <th className="px-5 py-3 font-semibold text-right">Biaya</th>
                    <th className="px-5 py-3 font-semibold text-right">Aksi</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-border">
                  {services.map((s) => (
                    <tr key={s.id} className="hover:bg-background/50 transition-colors">
                      <td className="px-5 py-3">
                        <Link to={`/services/${s.id}`} className="font-bold text-ink hover:text-accent-600">
                          {s.ticket_no}
                        </Link>
                        <p className="text-xs text-muted">
                          {new Date(s.received_at).toLocaleDateString('id-ID', {
                            day: 'numeric',
                            month: 'short',
                            year: 'numeric',
                          })}
                        </p>
                      </td>
                      <td className="px-5 py-3 text-xs font-medium">
                        {s.device.brand} {s.device.model}
                      </td>
                      <td className="px-5 py-3 text-xs text-muted max-w-xs truncate">
                        {s.complaint}
                      </td>
                      <td className="px-5 py-3">
                        <StatusBadge status={s.status} />
                      </td>
                      <td className="px-5 py-3 text-right text-xs font-semibold">
                        {formatCurrency(s.estimated_cost)}
                      </td>
                      <td className="px-5 py-3 text-right">
                        <Link to={`/services/${s.id}`}>
                          <Button variant="ghost" size="sm" className="h-8 text-xs">
                            Detail
                          </Button>
                        </Link>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </CardContent>
      </Card>

      {/* Edit Customer Dialog */}
      <CustomerFormDialog
        open={editCustomerOpen}
        onOpenChange={setEditCustomerOpen}
        customer={customer}
        onSuccess={() => loadAll()}
      />

      {/* Add Device Dialog */}
      <CustomerDeviceDialog
        open={addDeviceOpen}
        onOpenChange={setAddDeviceOpen}
        customerId={customer.id}
        customerName={customer.name}
        onSuccess={() => loadAll()}
      />
    </div>
  );
}
