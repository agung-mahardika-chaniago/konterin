import {
  Boxes,
  ClipboardList,
  LayoutDashboard,
  Receipt,
  ReceiptText,
  Settings,
  ShieldCheck,
  ShoppingCart,
  Store,
  Users,
  Wallet,
  Warehouse,
} from 'lucide-react';
import type { LucideIcon } from 'lucide-react';

export interface NavItem {
  label: string;
  to?: string;
  icon: LucideIcon;
  permission?: string;
  stage?: number;
}

export interface NavGroup {
  label: string;
  items: NavItem[];
}

/**
 * Navigation mirrors docs/08-uiux-guideline.md. Items that belong to later
 * stages stay visible but disabled so the shell never links to a dead page.
 */
export const navGroups: NavGroup[] = [
  {
    label: 'Utama',
    items: [{ label: 'Dashboard', to: '/dashboard', icon: LayoutDashboard, permission: 'dashboard.view' }],
  },
  {
    label: 'Operasional',
    items: [
      { label: 'Servis', to: '/services', icon: ClipboardList, permission: 'service.view' },
      { label: 'Pelanggan', to: '/customers', icon: Users, permission: 'customer.view' },
    ],
  },
  {
    label: 'Inventori',
    items: [
      { label: 'Sparepart', to: '/spareparts', icon: Boxes, permission: 'sparepart.view' },
      { label: 'Pergerakan Stok', to: '/inventory/movements', icon: Warehouse, permission: 'inventory.view' },
      { label: 'Pembelian', to: '/inventory/purchases/new', icon: ShoppingCart, permission: 'inventory.purchase' },
    ],
  },
  {
    label: 'Keuangan',
    items: [
      { label: 'Kasir', to: '/cashier', icon: ReceiptText, permission: 'payment.view' },
      { label: 'Invoice', to: '/invoices', icon: Wallet, permission: 'invoice.view' },
      { label: 'Pengeluaran', to: '/expenses', icon: Receipt, permission: 'expense.view' },
      { label: 'Laporan', icon: ClipboardList, permission: 'report.view', stage: 9 },
    ],
  },
  {
    label: 'Purna Jual',
    items: [{ label: 'Garansi', icon: ShieldCheck, permission: 'warranty.view', stage: 10 }],
  },
  {
    label: 'Pengaturan',
    items: [
      { label: 'Pengguna', icon: Users, permission: 'user.view', stage: 11 },
      { label: 'Outlet', icon: Store, permission: 'outlet.view', stage: 11 },
      { label: 'Toko', icon: Settings, permission: 'settings.view', stage: 11 },
      { label: 'Audit Log', icon: ShieldCheck, permission: 'audit.view', stage: 11 },
    ],
  },
];
