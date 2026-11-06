import * as React from 'react';

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
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Select } from '@/components/ui/select';
import { Textarea } from '@/components/ui/textarea';
import { ApiError } from '@/lib/api-client';
import { createSparepart, updateSparepart } from '../api';
import type { Sparepart, SparepartCategory, Supplier } from '../types';

interface SparepartFormDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  sparepart?: Sparepart | null;
  categories: SparepartCategory[];
  suppliers: Supplier[];
  onSuccess: (saved: Sparepart) => void;
}

export function SparepartFormDialog({
  open,
  onOpenChange,
  sparepart,
  categories,
  suppliers,
  onSuccess,
}: SparepartFormDialogProps) {
  const [sku, setSku] = React.useState('');
  const [barcode, setBarcode] = React.useState('');
  const [name, setName] = React.useState('');
  const [categoryId, setCategoryId] = React.useState('');
  const [supplierId, setSupplierId] = React.useState('');
  const [compatibleModels, setCompatibleModels] = React.useState('');
  const [stock, setStock] = React.useState('0');
  const [minStock, setMinStock] = React.useState('0');
  const [costPrice, setCostPrice] = React.useState('0');
  const [sellPrice, setSellPrice] = React.useState('0');
  const [location, setLocation] = React.useState('');
  const [isActive, setIsActive] = React.useState(true);
  const [loading, setLoading] = React.useState(false);
  const [error, setError] = React.useState<string | null>(null);
  const [fieldErrors, setFieldErrors] = React.useState<Record<string, string>>({});

  React.useEffect(() => {
    if (!open) return;
    setSku(sparepart?.sku ?? '');
    setBarcode(sparepart?.barcode ?? '');
    setName(sparepart?.name ?? '');
    setCategoryId(sparepart?.category_id ?? '');
    setSupplierId(sparepart?.supplier_id ?? '');
    setCompatibleModels(sparepart?.compatible_models ?? '');
    setStock(String(sparepart?.stock ?? 0));
    setMinStock(String(sparepart?.min_stock ?? 0));
    setCostPrice(String(sparepart?.cost_price ?? 0));
    setSellPrice(String(sparepart?.sell_price ?? 0));
    setLocation(sparepart?.location ?? '');
    setIsActive(sparepart?.is_active ?? true);
    setError(null);
    setFieldErrors({});
  }, [open, sparepart]);

  const handleSubmit = async (event: React.FormEvent) => {
    event.preventDefault();
    setLoading(true);
    setError(null);
    setFieldErrors({});
    try {
      const common = {
        sku: sku.trim(),
        name: name.trim(),
        barcode: barcode.trim(),
        category_id: categoryId,
        supplier_id: supplierId,
        compatible_models: compatibleModels.trim(),
        min_stock: Number(minStock) || 0,
        cost_price: Number(costPrice) || 0,
        sell_price: Number(sellPrice) || 0,
        location: location.trim(),
        is_active: isActive,
      };
      let saved: Sparepart;
      if (sparepart) {
        saved = await updateSparepart(sparepart.id, common);
      } else {
        saved = await createSparepart({
          ...common,
          stock: Number(stock) || 0,
        });
      }
      onSuccess(saved);
      onOpenChange(false);
    } catch (err: unknown) {
      if (err instanceof ApiError) {
        setError(err.message);
        setFieldErrors(err.fieldErrors());
      } else {
        setError('Gagal menyimpan sparepart.');
      }
    } finally {
      setLoading(false);
    }
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-2xl" onClose={() => onOpenChange(false)}>
        <DialogHeader>
          <DialogTitle>{sparepart ? 'Ubah Sparepart' : 'Sparepart Baru'}</DialogTitle>
          <DialogDescription>
            Stok hanya berubah melalui pembelian atau penyesuaian stok setelah sparepart dibuat.
          </DialogDescription>
        </DialogHeader>

        <form onSubmit={handleSubmit} className="space-y-4">
          {error && <Alert variant="error">{error}</Alert>}

          <div className="grid gap-4 sm:grid-cols-2">
            <div className="space-y-1.5">
              <Label htmlFor="sp-sku">SKU *</Label>
              <Input id="sp-sku" value={sku} onChange={(e) => setSku(e.target.value)} required />
              {fieldErrors.sku && <p className="text-xs text-danger">{fieldErrors.sku}</p>}
            </div>
            <div className="space-y-1.5">
              <Label htmlFor="sp-barcode">Barcode</Label>
              <Input id="sp-barcode" value={barcode} onChange={(e) => setBarcode(e.target.value)} />
              {fieldErrors.barcode && <p className="text-xs text-danger">{fieldErrors.barcode}</p>}
            </div>
          </div>

          <div className="space-y-1.5">
            <Label htmlFor="sp-name">Nama Sparepart *</Label>
            <Input id="sp-name" value={name} onChange={(e) => setName(e.target.value)} required />
            {fieldErrors.name && <p className="text-xs text-danger">{fieldErrors.name}</p>}
          </div>

          <div className="grid gap-4 sm:grid-cols-2">
            <div className="space-y-1.5">
              <Label htmlFor="sp-category">Kategori</Label>
              <Select id="sp-category" value={categoryId} onChange={(e) => setCategoryId(e.target.value)}>
                <option value="">Tanpa kategori</option>
                {categories.map((category) => (
                  <option key={category.id} value={category.id}>
                    {category.name}
                  </option>
                ))}
              </Select>
            </div>
            <div className="space-y-1.5">
              <Label htmlFor="sp-supplier">Supplier</Label>
              <Select id="sp-supplier" value={supplierId} onChange={(e) => setSupplierId(e.target.value)}>
                <option value="">Tanpa supplier</option>
                {suppliers.map((supplier) => (
                  <option key={supplier.id} value={supplier.id}>
                    {supplier.name}
                  </option>
                ))}
              </Select>
            </div>
          </div>

          <div className="space-y-1.5">
            <Label htmlFor="sp-models">Model Kompatibel</Label>
            <Textarea
              id="sp-models"
              rows={2}
              value={compatibleModels}
              onChange={(e) => setCompatibleModels(e.target.value)}
              placeholder="Contoh: iPhone 11, iPhone 12"
            />
          </div>

          <div className="grid gap-4 sm:grid-cols-3">
            {!sparepart && (
              <div className="space-y-1.5">
                <Label htmlFor="sp-stock">Stok Awal</Label>
                <Input id="sp-stock" type="number" min="0" value={stock} onChange={(e) => setStock(e.target.value)} />
                {fieldErrors.stock && <p className="text-xs text-danger">{fieldErrors.stock}</p>}
              </div>
            )}
            <div className="space-y-1.5">
              <Label htmlFor="sp-min">Stok Minimum</Label>
              <Input id="sp-min" type="number" min="0" value={minStock} onChange={(e) => setMinStock(e.target.value)} />
            </div>
            <div className="space-y-1.5">
              <Label htmlFor="sp-location">Lokasi Rak</Label>
              <Input id="sp-location" value={location} onChange={(e) => setLocation(e.target.value)} />
            </div>
          </div>

          <div className="grid gap-4 sm:grid-cols-2">
            <div className="space-y-1.5">
              <Label htmlFor="sp-cost">Harga Modal</Label>
              <Input id="sp-cost" type="number" min="0" value={costPrice} onChange={(e) => setCostPrice(e.target.value)} />
            </div>
            <div className="space-y-1.5">
              <Label htmlFor="sp-sell">Harga Jual</Label>
              <Input id="sp-sell" type="number" min="0" value={sellPrice} onChange={(e) => setSellPrice(e.target.value)} />
            </div>
          </div>

          <label className="flex items-center gap-2 text-sm text-ink">
            <input
              type="checkbox"
              checked={isActive}
              onChange={(e) => setIsActive(e.target.checked)}
              className="h-4 w-4 rounded border-border text-accent-600 focus:ring-accent-500"
            />
            Sparepart aktif dan dapat dipakai di servis
          </label>

          <DialogFooter>
            <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
              Batal
            </Button>
            <Button type="submit" loading={loading}>
              Simpan
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
