import {
  Bell,
  Boxes,
  ClipboardList,
  LayoutDashboard,
  MessageCircle,
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
      { label: 'Laporan', to: '/reports', icon: ClipboardList, permission: 'report.view' },
    ],
  },
  {
    label: 'Purna Jual',
    items: [
      { label: 'Garansi', to: '/warranties', icon: ShieldCheck, permission: 'warranty.view' },
      { label: 'Notifikasi', to: '/notifications', icon: Bell },
      { label: 'Outbox WA', to: '/whatsapp-outbox', icon: MessageCircle, permission: 'settings.manage' },
    ],
  },
  {
    label: 'Pengaturan',
    items: [
      { label: 'Pengguna', to: '/users', icon: Users, permission: 'user.view' },
      { label: 'Outlet', to: '/outlets', icon: Store, permission: 'outlet.view' },
      { label: 'Toko', to: '/settings', icon: Settings, permission: 'settings.view' },
      { label: 'Audit Log', to: '/audit-logs', icon: ShieldCheck, permission: 'audit.view' },
    ],
  },
];
