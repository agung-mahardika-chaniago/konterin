import { Bell, CheckCircle2, Clock3, PackageCheck, PackageOpen, ShieldAlert, ShieldCheck } from 'lucide-react';

const ICONS: Record<string, typeof Bell> = {
  service_ready: PackageCheck,
  service_picked: CheckCircle2,
  service_overdue: Clock3,
  service_uncollected: PackageOpen,
  warranty_claim: ShieldAlert,
  warranty_expiring: ShieldCheck,
};

export function NotificationIcon({ type, className }: { type: string; className?: string }) {
  const Icon = ICONS[type] ?? Bell;
  return <Icon className={className} aria-hidden="true" />;
}
