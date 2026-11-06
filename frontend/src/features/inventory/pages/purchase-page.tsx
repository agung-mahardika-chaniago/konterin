import * as React from 'react';
import { Link, useSearchParams } from 'react-router-dom';
import { ArrowLeft, Plus, ShoppingCart, Trash2 } from 'lucide-react';

import { Alert } from '@/components/ui/alert';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Select } from '@/components/ui/select';
import { Textarea } from '@/components/ui/textarea';
import { formatCurrency } from '@/features/dashboard/format';
import { ApiError } from '@/lib/api-client';
import { listSpareparts, listSuppliers, purchaseStock } from '../api';
import { todayInput } from '../format';
import type { PurchaseResult, Sparepart, Supplier } from '../types';

interface Line {
  key: string;
  sparepart_id: string;
  qty: string;
  unit_cost: string;
}

function newLine(sparepartId = ''): Line {
  return {
    key: Math.random().toString(36).slice(2),
    sparepart_id: sparepartId,
    qty: '1',
    unit_cost: '0',
  };
}

export function PurchasePage() {
  const [searchParams] = useSearchParams();
  const preselected = searchParams.get('sparepart') ?? '';

  const [parts, setParts] = React.useState<Sparepart[]>([]);
  const [suppliers, setSuppliers] = React.useState<Supplier[]>([]);
  const [supplierId, setSupplierId] = React.useState('');
  const [referenceNo, setReferenceNo] = React.useState('');
  const [purchasedAt, setPurchasedAt] = React.useState(todayInput());
  const [notes, setNotes] = React.useState('');
  const [lines, setLines] = React.useState<Line[]>([newLine(preselected)]);
  const [loading, setLoading] = React.useState(false);
  const [error, setError] = React.useState<string | null>(null);
  const [result, setResult] = React.useState<PurchaseResult | null>(null);

  React.useEffect(() => {
    Promise.all([listSpareparts({ per_page: 100, is_active: true }), listSuppliers()])
      .then(([partResult, supplierList]) => {
        setParts(partResult.data ?? []);
        setSuppliers(supplierList);
      })
      .catch((err: unknown) => {
        setError(err instanceof Error ? err.message : 'Gagal memuat data pembelian.');
      });
  }, []);

  const updateLine = (key: string, patch: Partial<Line>) => {
    setLines((current) => current.map((line) => (line.key === key ? { ...line, ...patch } : line)));
  };

  React.useEffect(() => {
    if (!preselected || parts.length === 0) return;
    const part = parts.find((item) => item.id === preselected);
    if (part) {
      setLines([{ ...newLine(preselected), unit_cost: String(part.cost_price) }]);
    }
  }, [parts, preselected]);

  const selectPart = (key: string, sparepartId: string) => {
    const part = parts.find((item) => item.id === sparepartId);
    updateLine(key, {
      sparepart_id: sparepartId,
      unit_cost: part ? String(part.cost_price) : '0',
    });
  };

  const total = lines.reduce(
    (sum, line) => sum + (Number(line.qty) || 0) * (Number(line.unit_cost) || 0),
    0,
  );

  const handleSubmit = async (event: React.FormEvent) => {
    event.preventDefault();
    setLoading(true);
    setError(null);
    setResult(null);
    try {
      const payload = {
        supplier_id: supplierId || undefined,
        reference_no: referenceNo.trim() || undefined,
        purchased_at: new Date(`${purchasedAt}T09:00:00`).toISOString(),
        items: lines
          .filter((line) => line.sparepart_id)
          .map((line) => ({
            sparepart_id: line.sparepart_id,
            qty: Number(line.qty) || 0,
            unit_cost: Number(line.unit_cost) || 0,
          })),
        notes: notes.trim() || undefined,
      };
      if (payload.items.length === 0) {
        setError('Pilih minimal satu sparepart.');
        return;
      }
      const saved = await purchaseStock(payload);
      setResult(saved);
      setReferenceNo('');
      setNotes('');
      setLines([newLine()]);
      const refreshed = await listSpareparts({ per_page: 100, is_active: true });
      setParts(refreshed.data ?? []);
    } catch (err: unknown) {
      setError(err instanceof ApiError ? err.message : 'Gagal menyimpan pembelian.');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="space-y-5">
      <div className="flex items-start gap-3">
        <Link to="/spareparts">
          <Button variant="ghost" size="icon" title="Kembali">
            <ArrowLeft className="h-4 w-4" aria-hidden="true" />
          </Button>
        </Link>
        <div>
          <h1 className="text-2xl font-bold tracking-tight text-ink sm:text-[28px]">Pembelian Stok</h1>
          <p className="mt-1 text-sm text-muted">
            Catat barang masuk dari supplier. Stok dan harga modal sparepart diperbarui otomatis.
          </p>
        </div>
      </div>

      {error && <Alert variant="error">{error}</Alert>}

      {result && (
        <Alert variant="success">
          <p className="font-semibold">Pembelian berhasil dicatat.</p>
          <p className="mt-0.5 text-xs">
            Total {formatCurrency(result.total_cost)} · {result.items.length} item ·{' '}
            <Link to="/inventory/movements" className="underline">
              lihat pergerakan stok
            </Link>
          </p>
        </Alert>
      )}

      <form onSubmit={handleSubmit} className="space-y-4">
        <Card>
          <CardHeader>
            <CardTitle className="text-base">Informasi Pembelian</CardTitle>
          </CardHeader>
          <CardContent className="grid gap-4 sm:grid-cols-2">
            <div className="space-y-1.5">
              <Label htmlFor="purchase-supplier">Supplier</Label>
              <Select id="purchase-supplier" value={supplierId} onChange={(e) => setSupplierId(e.target.value)}>
                <option value="">Tanpa supplier</option>
                {suppliers.map((supplier) => (
                  <option key={supplier.id} value={supplier.id}>
                    {supplier.name}
                  </option>
                ))}
              </Select>
            </div>
            <div className="space-y-1.5">
              <Label htmlFor="purchase-reference">No. Referensi</Label>
              <Input
                id="purchase-reference"
                value={referenceNo}
                onChange={(e) => setReferenceNo(e.target.value)}
                placeholder="Contoh: INV-SUP-001"
              />
            </div>
            <div className="space-y-1.5">
              <Label htmlFor="purchase-date">Tanggal Pembelian</Label>
              <Input
                id="purchase-date"
                type="date"
                value={purchasedAt}
                onChange={(e) => setPurchasedAt(e.target.value)}
                required
              />
            </div>
            <div className="space-y-1.5 sm:col-span-2">
              <Label htmlFor="purchase-notes">Catatan</Label>
              <Textarea id="purchase-notes" rows={2} value={notes} onChange={(e) => setNotes(e.target.value)} />
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between gap-3">
            <CardTitle className="text-base">Item Pembelian</CardTitle>
            <Button
              type="button"
              variant="outline"
              size="sm"
              onClick={() => setLines((current) => [...current, newLine()])}
              className="gap-1.5"
            >
              <Plus className="h-3.5 w-3.5" aria-hidden="true" />
              Tambah Baris
            </Button>
          </CardHeader>
          <CardContent className="space-y-3">
            {lines.map((line) => {
              const part = parts.find((item) => item.id === line.sparepart_id);
              return (
                <div key={line.key} className="grid gap-3 rounded-control border border-border p-3 sm:grid-cols-12">
                  <div className="space-y-1.5 sm:col-span-5">
                    <Label>Sparepart</Label>
                    <Select value={line.sparepart_id} onChange={(e) => selectPart(line.key, e.target.value)}>
                      <option value="">Pilih sparepart</option>
                      {parts.map((item) => (
                        <option key={item.id} value={item.id}>
                          {item.name} ({item.sku}) · stok {item.stock}
                        </option>
                      ))}
                    </Select>
                  </div>
                  <div className="space-y-1.5 sm:col-span-2">
                    <Label>Qty</Label>
                    <Input
                      type="number"
                      min="1"
                      value={line.qty}
                      onChange={(e) => updateLine(line.key, { qty: e.target.value })}
                    />
                  </div>
                  <div className="space-y-1.5 sm:col-span-3">
                    <Label>Harga Satuan</Label>
                    <Input
                      type="number"
                      min="0"
                      value={line.unit_cost}
                      onChange={(e) => updateLine(line.key, { unit_cost: e.target.value })}
                    />
                  </div>
                  <div className="flex items-end justify-between gap-2 sm:col-span-2">
                    <div className="text-right">
                      <p className="text-[11px] uppercase tracking-wide text-muted">Subtotal</p>
                      <p className="text-sm font-semibold text-ink">
                        {formatCurrency((Number(line.qty) || 0) * (Number(line.unit_cost) || 0))}
                      </p>
                      {part && part.low_stock && <p className="text-[11px] text-amber-600">Stok menipis</p>}
                    </div>
                    <Button
                      type="button"
                      variant="ghost"
                      size="icon"
                      onClick={() => setLines((current) => (current.length > 1 ? current.filter((item) => item.key !== line.key) : current))}
                      disabled={lines.length === 1}
                      title="Hapus baris"
                      className="text-muted hover:text-danger"
                    >
                      <Trash2 className="h-4 w-4" aria-hidden="true" />
                    </Button>
                  </div>
                </div>
              );
            })}

            <div className="flex items-center justify-between border-t border-border pt-3">
              <span className="text-sm text-muted">Total pembelian</span>
              <span className="text-lg font-bold text-ink">{formatCurrency(total)}</span>
            </div>
          </CardContent>
        </Card>

        <div className="flex justify-end gap-2">
          <Link to="/spareparts">
            <Button type="button" variant="outline">
              Batal
            </Button>
          </Link>
          <Button type="submit" loading={loading} className="gap-2">
            <ShoppingCart className="h-4 w-4" aria-hidden="true" />
            Simpan Pembelian
          </Button>
        </div>
      </form>
    </div>
  );
}
