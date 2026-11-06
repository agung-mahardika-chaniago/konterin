import * as React from 'react';
import { Pencil, Plus } from 'lucide-react';

import { Alert } from '@/components/ui/alert';
import { Button } from '@/components/ui/button';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { ApiError } from '@/lib/api-client';
import { createCategory, createSupplier, updateCategory, updateSupplier } from '../api';
import type { SparepartCategory, Supplier } from '../types';

interface MasterDataDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  categories: SparepartCategory[];
  suppliers: Supplier[];
  canManage: boolean;
  onChanged: () => void;
}

export function MasterDataDialog({
  open,
  onOpenChange,
  categories,
  suppliers,
  canManage,
  onChanged,
}: MasterDataDialogProps) {
  const [tab, setTab] = React.useState<'categories' | 'suppliers'>('categories');
  const [categoryName, setCategoryName] = React.useState('');
  const [editingCategory, setEditingCategory] = React.useState<SparepartCategory | null>(null);
  const [supplierForm, setSupplierForm] = React.useState({
    name: '',
    contact_name: '',
    phone: '',
    address: '',
  });
  const [editingSupplier, setEditingSupplier] = React.useState<Supplier | null>(null);
  const [loading, setLoading] = React.useState(false);
  const [error, setError] = React.useState<string | null>(null);

  React.useEffect(() => {
    if (!open) return;
    setTab('categories');
    setCategoryName('');
    setEditingCategory(null);
    setSupplierForm({ name: '', contact_name: '', phone: '', address: '' });
    setEditingSupplier(null);
    setError(null);
  }, [open]);

  const submitCategory = async (event: React.FormEvent) => {
    event.preventDefault();
    setLoading(true);
    setError(null);
    try {
      if (editingCategory) {
        await updateCategory(editingCategory.id, categoryName.trim());
      } else {
        await createCategory(categoryName.trim());
      }
      setCategoryName('');
      setEditingCategory(null);
      onChanged();
    } catch (err: unknown) {
      setError(err instanceof ApiError ? err.message : 'Gagal menyimpan kategori.');
    } finally {
      setLoading(false);
    }
  };

  const submitSupplier = async (event: React.FormEvent) => {
    event.preventDefault();
    setLoading(true);
    setError(null);
    try {
      const payload = {
        name: supplierForm.name.trim(),
        contact_name: supplierForm.contact_name.trim(),
        phone: supplierForm.phone.trim(),
        address: supplierForm.address.trim(),
      };
      if (editingSupplier) {
        await updateSupplier(editingSupplier.id, payload);
      } else {
        await createSupplier(payload);
      }
      setSupplierForm({ name: '', contact_name: '', phone: '', address: '' });
      setEditingSupplier(null);
      onChanged();
    } catch (err: unknown) {
      setError(err instanceof ApiError ? err.message : 'Gagal menyimpan supplier.');
    } finally {
      setLoading(false);
    }
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-2xl" onClose={() => onOpenChange(false)}>
        <DialogHeader>
          <DialogTitle>Kategori & Supplier</DialogTitle>
          <DialogDescription>Kelola kelompok sparepart dan pemasok untuk pembelian stok.</DialogDescription>
        </DialogHeader>

        <div className="mb-4 flex gap-1 rounded-control bg-background p-1">
          {(['categories', 'suppliers'] as const).map((value) => (
            <button
              key={value}
              type="button"
              onClick={() => setTab(value)}
              className={`flex-1 rounded-control px-3 py-2 text-sm font-medium transition-colors ${
                tab === value ? 'bg-surface text-ink shadow-sm' : 'text-muted hover:text-ink'
              }`}
            >
              {value === 'categories' ? 'Kategori' : 'Supplier'}
            </button>
          ))}
        </div>

        {error && <Alert variant="error" className="mb-4">{error}</Alert>}

        {tab === 'categories' ? (
          <div className="space-y-4">
            {canManage && (
              <form onSubmit={submitCategory} className="flex items-end gap-2">
                <div className="flex-1 space-y-1.5">
                  <Label htmlFor="category-name">Nama Kategori</Label>
                  <Input
                    id="category-name"
                    value={categoryName}
                    onChange={(e) => setCategoryName(e.target.value)}
                    required
                  />
                </div>
                <Button type="submit" loading={loading} className="gap-1.5">
                  <Plus className="h-4 w-4" aria-hidden="true" />
                  {editingCategory ? 'Ubah' : 'Tambah'}
                </Button>
                {editingCategory && (
                  <Button
                    type="button"
                    variant="outline"
                    onClick={() => {
                      setEditingCategory(null);
                      setCategoryName('');
                    }}
                  >
                    Batal
                  </Button>
                )}
              </form>
            )}
            <div className="max-h-64 divide-y divide-border overflow-y-auto rounded-control border border-border">
              {categories.length === 0 ? (
                <p className="px-4 py-6 text-center text-sm text-muted">Belum ada kategori.</p>
              ) : (
                categories.map((category) => (
                  <div key={category.id} className="flex items-center justify-between gap-3 px-4 py-2.5">
                    <div>
                      <p className="text-sm font-medium text-ink">{category.name}</p>
                      <p className="text-xs text-muted">{category.sparepart_count} sparepart</p>
                    </div>
                    {canManage && (
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={() => {
                          setEditingCategory(category);
                          setCategoryName(category.name);
                        }}
                        className="h-8 px-2"
                        title="Ubah kategori"
                      >
                        <Pencil className="h-3.5 w-3.5" aria-hidden="true" />
                      </Button>
                    )}
                  </div>
                ))
              )}
            </div>
          </div>
        ) : (
          <div className="space-y-4">
            {canManage && (
              <form onSubmit={submitSupplier} className="space-y-3 rounded-control border border-border p-3">
                <div className="grid gap-3 sm:grid-cols-2">
                  <div className="space-y-1.5">
                    <Label htmlFor="supplier-name">Nama Supplier *</Label>
                    <Input
                      id="supplier-name"
                      value={supplierForm.name}
                      onChange={(e) => setSupplierForm((f) => ({ ...f, name: e.target.value }))}
                      required
                    />
                  </div>
                  <div className="space-y-1.5">
                    <Label htmlFor="supplier-contact">Kontak</Label>
                    <Input
                      id="supplier-contact"
                      value={supplierForm.contact_name}
                      onChange={(e) => setSupplierForm((f) => ({ ...f, contact_name: e.target.value }))}
                    />
                  </div>
                  <div className="space-y-1.5">
                    <Label htmlFor="supplier-phone">Telepon</Label>
                    <Input
                      id="supplier-phone"
                      value={supplierForm.phone}
                      onChange={(e) => setSupplierForm((f) => ({ ...f, phone: e.target.value }))}
                    />
                  </div>
                  <div className="space-y-1.5">
                    <Label htmlFor="supplier-address">Alamat</Label>
                    <Input
                      id="supplier-address"
                      value={supplierForm.address}
                      onChange={(e) => setSupplierForm((f) => ({ ...f, address: e.target.value }))}
                    />
                  </div>
                </div>
                <div className="flex justify-end gap-2">
                  {editingSupplier && (
                    <Button
                      type="button"
                      variant="outline"
                      onClick={() => {
                        setEditingSupplier(null);
                        setSupplierForm({ name: '', contact_name: '', phone: '', address: '' });
                      }}
                    >
                      Batal
                    </Button>
                  )}
                  <Button type="submit" loading={loading} className="gap-1.5">
                    <Plus className="h-4 w-4" aria-hidden="true" />
                    {editingSupplier ? 'Simpan Perubahan' : 'Tambah Supplier'}
                  </Button>
                </div>
              </form>
            )}
            <div className="max-h-64 divide-y divide-border overflow-y-auto rounded-control border border-border">
              {suppliers.length === 0 ? (
                <p className="px-4 py-6 text-center text-sm text-muted">Belum ada supplier.</p>
              ) : (
                suppliers.map((supplier) => (
                  <div key={supplier.id} className="flex items-center justify-between gap-3 px-4 py-2.5">
                    <div>
                      <p className="text-sm font-medium text-ink">{supplier.name}</p>
                      <p className="text-xs text-muted">
                        {[supplier.contact_name, supplier.phone].filter(Boolean).join(' · ') || 'Tanpa kontak'}
                      </p>
                    </div>
                    {canManage && (
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={() => {
                          setEditingSupplier(supplier);
                          setSupplierForm({
                            name: supplier.name,
                            contact_name: supplier.contact_name ?? '',
                            phone: supplier.phone ?? '',
                            address: supplier.address ?? '',
                          });
                        }}
                        className="h-8 px-2"
                        title="Ubah supplier"
                      >
                        <Pencil className="h-3.5 w-3.5" aria-hidden="true" />
                      </Button>
                    )}
                  </div>
                ))
              )}
            </div>
          </div>
        )}
      </DialogContent>
    </Dialog>
  );
}
