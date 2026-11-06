import * as React from 'react';
import { Loader2, Package, Pencil, Plus, Search, Trash2, Wrench } from 'lucide-react';

import { EmptyState } from '@/components/shared/empty-state';
import { Alert } from '@/components/ui/alert';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Select } from '@/components/ui/select';
import { Textarea } from '@/components/ui/textarea';
import { formatCurrency } from '@/features/dashboard/format';
import { ApiError } from '@/lib/api-client';
import { addServiceItem, deleteServiceItem, searchSpareparts, updateServiceItem } from '../api';
import type { InvoiceRef, ServiceOrderItem, SparepartOption } from '../types';

interface ItemsEditorProps {
  orderId: string;
  items: ServiceOrderItem[];
  invoice?: InvoiceRef | null;
  canUpdate: boolean;
  onChanged: () => void;
}

const ITEM_TYPE_LABEL: Record<string, string> = {
  service: 'Jasa',
  sparepart: 'Sparepart',
  custom: 'Custom',
};

const ITEM_TYPE_VARIANT: Record<string, 'info' | 'violet' | 'neutral'> = {
  service: 'info',
  sparepart: 'violet',
  custom: 'neutral',
};

export function ItemsEditor({ orderId, items, invoice, canUpdate, onChanged }: ItemsEditorProps) {
  const [formOpen, setFormOpen] = React.useState(false);
  const [editingItem, setEditingItem] = React.useState<ServiceOrderItem | null>(null);
  const [deletingId, setDeletingId] = React.useState<string | null>(null);
  const [error, setError] = React.useState<string | null>(null);

  const locked = Boolean(invoice && (invoice.status !== 'draft' || invoice.paid_amount > 0));
  const total = items.reduce((sum, item) => sum + item.subtotal, 0);
  const canEdit = canUpdate && !locked;

  const openAdd = () => {
    setEditingItem(null);
    setFormOpen(true);
  };

  const openEdit = (item: ServiceOrderItem) => {
    setEditingItem(item);
    setFormOpen(true);
  };

  const handleDelete = async (item: ServiceOrderItem) => {
    if (!window.confirm(`Hapus item "${item.description}"? Stok sparepart akan dikembalikan otomatis.`)) return;
    setDeletingId(item.id);
    setError(null);
    try {
      await deleteServiceItem(orderId, item.id);
      onChanged();
    } catch (err: unknown) {
      setError(err instanceof ApiError ? err.message : 'Gagal menghapus item.');
    } finally {
      setDeletingId(null);
    }
  };

  return (
    <Card>
      <CardHeader className="flex flex-row items-center justify-between gap-3 pb-3">
        <div className="flex items-center gap-2">
          <Wrench className="h-5 w-5 text-accent-600" aria-hidden="true" />
          <div>
            <CardTitle className="text-base font-semibold">Item Jasa & Sparepart</CardTitle>
            <p className="text-xs text-muted">Harga dihitung server; stok sparepart berkurang otomatis.</p>
          </div>
        </div>
        {canEdit && (
          <Button size="sm" onClick={openAdd} className="gap-1.5 text-xs">
            <Plus className="h-3.5 w-3.5" aria-hidden="true" />
            Tambah Item
          </Button>
        )}
      </CardHeader>
      <CardContent className="space-y-4">
        {error && <Alert variant="error">{error}</Alert>}

        {locked && (
          <Alert variant="info">
            Invoice sudah difinalisasi. Item tidak dapat diubah lagi; gunakan proses pembatalan invoice bila perlu.
          </Alert>
        )}

        {items.length === 0 ? (
          <EmptyState
            icon={Package}
            title="Belum ada item jasa atau sparepart"
            description="Tambahkan jasa perbaikan dan sparepart yang dipakai agar total tagihan otomatis terhitung."
            className="py-6"
          />
        ) : (
          <div className="divide-y divide-border">
            {items.map((item) => (
              <div key={item.id} className="flex flex-col gap-2 py-3 sm:flex-row sm:items-center sm:justify-between">
                <div className="min-w-0 flex-1">
                  <div className="flex flex-wrap items-center gap-2">
                    <Badge variant={ITEM_TYPE_VARIANT[item.item_type] ?? 'neutral'}>
                      {ITEM_TYPE_LABEL[item.item_type] ?? item.item_type}
                    </Badge>
                    <p className="text-sm font-semibold text-ink">{item.description}</p>
                  </div>
                  <p className="mt-1 text-xs text-muted">
                    {item.qty} × {formatCurrency(item.unit_price)}
                    {item.discount_amount > 0 && ` − diskon ${formatCurrency(item.discount_amount)}`}
                    {item.warranty_days > 0 && ` • garansi ${item.warranty_days} hari`}
                    {item.added_by && ` • oleh ${item.added_by}`}
                  </p>
                </div>
                <div className="flex items-center justify-between gap-3 sm:justify-end">
                  <span className="font-mono text-sm font-bold text-ink">{formatCurrency(item.subtotal)}</span>
                  {canEdit && (
                    <div className="flex items-center gap-1">
                      <Button
                        variant="ghost"
                        size="icon"
                        className="h-8 w-8"
                        onClick={() => openEdit(item)}
                        title="Ubah item"
                      >
                        <Pencil className="h-3.5 w-3.5" aria-hidden="true" />
                      </Button>
                      <Button
                        variant="ghost"
                        size="icon"
                        className="h-8 w-8 text-danger hover:text-danger"
                        onClick={() => handleDelete(item)}
                        loading={deletingId === item.id}
                        title="Hapus item"
                      >
                        <Trash2 className="h-3.5 w-3.5" aria-hidden="true" />
                      </Button>
                    </div>
                  )}
                </div>
              </div>
            ))}
          </div>
        )}

        {items.length > 0 && (
          <div className="flex items-center justify-between border-t border-border pt-3">
            <span className="text-sm font-semibold text-ink">Total Item</span>
            <span className="font-mono text-lg font-bold text-ink">{formatCurrency(total)}</span>
          </div>
        )}
      </CardContent>

      <ItemFormDialog
        open={formOpen}
        onOpenChange={setFormOpen}
        orderId={orderId}
        item={editingItem}
        onSaved={() => {
          setFormOpen(false);
          onChanged();
        }}
      />
    </Card>
  );
}

interface ItemFormDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  orderId: string;
  item: ServiceOrderItem | null;
  onSaved: () => void;
}

function ItemFormDialog({ open, onOpenChange, orderId, item, onSaved }: ItemFormDialogProps) {
  const [itemType, setItemType] = React.useState<'service' | 'sparepart' | 'custom'>('service');
  const [description, setDescription] = React.useState('');
  const [qty, setQty] = React.useState('1');
  const [unitPrice, setUnitPrice] = React.useState('');
  const [discount, setDiscount] = React.useState('0');
  const [warranty, setWarranty] = React.useState('0');

  const [spQuery, setSpQuery] = React.useState('');
  const [spOptions, setSpOptions] = React.useState<SparepartOption[]>([]);
  const [spLoading, setSpLoading] = React.useState(false);
  const [selectedSparepart, setSelectedSparepart] = React.useState<SparepartOption | null>(null);

  const [loading, setLoading] = React.useState(false);
  const [error, setError] = React.useState<string | null>(null);

  React.useEffect(() => {
    if (!open) return;
    setError(null);
    if (item) {
      setItemType(item.item_type as 'service' | 'sparepart' | 'custom');
      setDescription(item.description);
      setQty(String(item.qty));
      setUnitPrice(String(item.unit_price));
      setDiscount(String(item.discount_amount));
      setWarranty(String(item.warranty_days));
      setSelectedSparepart(null);
      setSpQuery('');
    } else {
      setItemType('service');
      setDescription('');
      setQty('1');
      setUnitPrice('');
      setDiscount('0');
      setWarranty('0');
      setSelectedSparepart(null);
      setSpQuery('');
    }
  }, [open, item]);

  React.useEffect(() => {
    if (!open || item || itemType !== 'sparepart' || selectedSparepart) return;
    const timer = setTimeout(async () => {
      setSpLoading(true);
      try {
        setSpOptions(await searchSpareparts(spQuery, 8));
      } catch {
        setSpOptions([]);
      } finally {
        setSpLoading(false);
      }
    }, 300);
    return () => clearTimeout(timer);
  }, [open, item, itemType, spQuery, selectedSparepart]);

  const selectSparepart = (sparepart: SparepartOption) => {
    setSelectedSparepart(sparepart);
    setDescription(sparepart.name);
    setUnitPrice(String(sparepart.sell_price));
    setError(null);
  };

  const resetSparepart = () => {
    setSelectedSparepart(null);
    setDescription('');
    setUnitPrice('');
    setSpOptions([]);
  };

  const handleSubmit = async (event: React.FormEvent) => {
    event.preventDefault();
    setError(null);

    const parsedQty = Number(qty);
    const parsedPrice = Number(unitPrice) || 0;
    const parsedDiscount = Number(discount) || 0;
    const parsedWarranty = Number(warranty) || 0;

    if (!Number.isFinite(parsedQty) || parsedQty <= 0) {
      setError('Jumlah item harus lebih dari 0.');
      return;
    }
    if (itemType === 'sparepart' && !item && !selectedSparepart) {
      setError('Pilih sparepart terlebih dahulu.');
      return;
    }
    if (itemType !== 'sparepart' && !description.trim()) {
      setError('Deskripsi item wajib diisi.');
      return;
    }
    if (parsedDiscount > parsedQty * parsedPrice) {
      setError('Diskon tidak boleh melebihi harga item.');
      return;
    }
    if (itemType === 'sparepart' && !Number.isInteger(parsedQty)) {
      setError('Jumlah sparepart harus bilangan bulat.');
      return;
    }

    setLoading(true);
    try {
      if (item) {
        await updateServiceItem(orderId, item.id, {
          description: description.trim(),
          qty: parsedQty,
          unit_price: parsedPrice,
          discount_amount: parsedDiscount,
          warranty_days: parsedWarranty,
        });
      } else {
        await addServiceItem(orderId, {
          item_type: itemType,
          sparepart_id: itemType === 'sparepart' ? selectedSparepart?.id ?? null : null,
          description: description.trim(),
          qty: parsedQty,
          unit_price: parsedPrice,
          discount_amount: parsedDiscount,
          warranty_days: parsedWarranty,
        });
      }
      onSaved();
    } catch (err: unknown) {
      setError(err instanceof ApiError ? err.message : 'Gagal menyimpan item.');
    } finally {
      setLoading(false);
    }
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent onClose={() => onOpenChange(false)} className="max-w-lg">
        <form onSubmit={handleSubmit}>
          <DialogHeader>
            <DialogTitle>{item ? 'Ubah Item' : 'Tambah Item Servis'}</DialogTitle>
            <DialogDescription>
              Total dihitung ulang otomatis oleh server setelah item disimpan.
            </DialogDescription>
          </DialogHeader>

          {error && (
            <div className="mb-4">
              <Alert variant="error">{error}</Alert>
            </div>
          )}

          <div className="space-y-4 py-2 max-h-[60vh] overflow-y-auto px-1">
            <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
              <div>
                <Label htmlFor="item-type">Tipe Item</Label>
                <Select
                  id="item-type"
                  value={itemType}
                  onChange={(event) => {
                    setItemType(event.target.value as 'service' | 'sparepart' | 'custom');
                    setSelectedSparepart(null);
                    setSpOptions([]);
                    setDescription('');
                    setUnitPrice('');
                  }}
                  className="mt-1"
                  disabled={Boolean(item)}
                >
                  <option value="service">Jasa Servis</option>
                  <option value="sparepart">Sparepart</option>
                  <option value="custom">Custom / Lainnya</option>
                </Select>
              </div>
              <div>
                <Label htmlFor="item-qty">Jumlah</Label>
                <Input
                  id="item-qty"
                  type="number"
                  min="0.01"
                  step={itemType === 'sparepart' ? '1' : '0.01'}
                  value={qty}
                  onChange={(event) => setQty(event.target.value)}
                  className="mt-1"
                  required
                />
              </div>
            </div>

            {itemType === 'sparepart' && !item && (
              <div>
                <Label>Cari Sparepart</Label>
                {selectedSparepart ? (
                  <div className="mt-1 flex items-start justify-between gap-3 rounded-lg border border-border bg-background p-3">
                    <div className="min-w-0">
                      <p className="text-sm font-semibold text-ink">{selectedSparepart.name}</p>
                      <p className="text-xs text-muted">
                        {selectedSparepart.sku} • stok {selectedSparepart.stock}
                        {selectedSparepart.location ? ` • ${selectedSparepart.location}` : ''}
                      </p>
                    </div>
                    <Button type="button" variant="outline" size="sm" onClick={resetSparepart}>
                      Ganti
                    </Button>
                  </div>
                ) : (
                  <>
                    <div className="relative mt-1">
                      <Search
                        className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted"
                        aria-hidden="true"
                      />
                      <Input
                        value={spQuery}
                        onChange={(event) => setSpQuery(event.target.value)}
                        placeholder="Ketik/scan nama, SKU, barcode, atau tipe HP..."
                        className="pl-9"
                        autoFocus
                      />
                    </div>
                    <div className="mt-1 max-h-44 overflow-y-auto rounded-lg border border-border">
                      {spLoading ? (
                        <div className="flex items-center gap-2 p-3 text-xs text-muted">
                          <Loader2 className="h-4 w-4 animate-spin" aria-hidden="true" />
                          Mencari sparepart...
                        </div>
                      ) : spOptions.length === 0 ? (
                        <p className="p-3 text-xs text-muted">Sparepart tidak ditemukan.</p>
                      ) : (
                        spOptions.map((sparepart) => (
                          <button
                            key={sparepart.id}
                            type="button"
                            onClick={() => selectSparepart(sparepart)}
                            className="flex w-full items-center justify-between gap-3 border-b border-border px-3 py-2 text-left last:border-b-0 hover:bg-background"
                          >
                            <span className="min-w-0">
                              <span className="block truncate text-xs font-semibold text-ink">
                                {sparepart.name}
                              </span>
                              <span className="block text-[11px] text-muted">
                                {sparepart.sku} • stok {sparepart.stock}
                              </span>
                            </span>
                            <span className="shrink-0 font-mono text-xs font-semibold text-ink">
                              {formatCurrency(sparepart.sell_price)}
                            </span>
                          </button>
                        ))
                      )}
                    </div>
                  </>
                )}
              </div>
            )}

            <div>
              <Label htmlFor="item-description">
                Deskripsi {itemType !== 'sparepart' && <span className="text-danger">*</span>}
              </Label>
              <Textarea
                id="item-description"
                value={description}
                onChange={(event) => setDescription(event.target.value)}
                placeholder={itemType === 'sparepart' ? 'Nama sparepart' : 'Contoh: Jasa ganti LCD'}
                rows={2}
                className="mt-1"
                required={itemType !== 'sparepart'}
              />
            </div>

            <div className="grid grid-cols-1 sm:grid-cols-3 gap-3">
              <div>
                <Label htmlFor="item-price">Harga Satuan (Rp)</Label>
                <Input
                  id="item-price"
                  type="number"
                  min="0"
                  step="500"
                  value={unitPrice}
                  onChange={(event) => setUnitPrice(event.target.value)}
                  className="mt-1"
                  required
                />
              </div>
              <div>
                <Label htmlFor="item-discount">Diskon Baris (Rp)</Label>
                <Input
                  id="item-discount"
                  type="number"
                  min="0"
                  step="500"
                  value={discount}
                  onChange={(event) => setDiscount(event.target.value)}
                  className="mt-1"
                />
              </div>
              <div>
                <Label htmlFor="item-warranty">Garansi (hari)</Label>
                <Input
                  id="item-warranty"
                  type="number"
                  min="0"
                  step="1"
                  value={warranty}
                  onChange={(event) => setWarranty(event.target.value)}
                  className="mt-1"
                />
              </div>
            </div>

            <div className="rounded-lg bg-background p-3 text-xs text-muted">
              Subtotal:{' '}
              <span className="font-mono font-semibold text-ink">
                {formatCurrency(Math.max(0, (Number(qty) || 0) * (Number(unitPrice) || 0) - (Number(discount) || 0)))}
              </span>
            </div>
          </div>

          <DialogFooter>
            <Button type="button" variant="outline" onClick={() => onOpenChange(false)} disabled={loading}>
              Batal
            </Button>
            <Button type="submit" loading={loading}>
              {item ? 'Simpan Perubahan' : 'Tambah Item'}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
