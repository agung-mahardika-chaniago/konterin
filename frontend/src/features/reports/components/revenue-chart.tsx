import { Area, AreaChart, CartesianGrid, ResponsiveContainer, Tooltip, XAxis, YAxis } from 'recharts';

import { formatCompactCurrency, formatCurrency } from '@/features/dashboard/format';
import type { RevenuePoint } from '../types';

interface RevenueChartProps {
  points: RevenuePoint[];
}

/** Revenue trend per day/month/year with paid vs unpaid breakdown. */
export function RevenueChart({ points }: RevenueChartProps) {
  const hasData = points.some((point) => point.total_revenue > 0);

  return (
    <div className="relative h-[280px] w-full" role="img" aria-label="Grafik pendapatan per periode">
      {!hasData && (
        <p className="absolute inset-x-0 top-1/2 z-10 -translate-y-1/2 text-center text-xs text-muted">
          Belum ada pendapatan pada rentang ini.
        </p>
      )}
      <ResponsiveContainer width="100%" height="100%">
        <AreaChart data={points} margin={{ top: 8, right: 8, bottom: 0, left: 0 }}>
          <defs>
            <linearGradient id="report-revenue-fill" x1="0" y1="0" x2="0" y2="1">
              <stop offset="0%" stopColor="#06B6D4" stopOpacity={0.28} />
              <stop offset="100%" stopColor="#06B6D4" stopOpacity={0.02} />
            </linearGradient>
          </defs>
          <CartesianGrid strokeDasharray="3 3" vertical={false} stroke="#E2E8F0" />
          <XAxis
            dataKey="label"
            tickLine={false}
            axisLine={false}
            minTickGap={24}
            tickMargin={8}
            tick={{ fontSize: 11, fill: '#64748B' }}
          />
          <YAxis
            width={62}
            tickLine={false}
            axisLine={false}
            tick={{ fontSize: 11, fill: '#64748B' }}
            tickFormatter={(value) => formatCompactCurrency(Number(value))}
          />
          <Tooltip
            cursor={{ stroke: '#06B6D4', strokeWidth: 1, strokeDasharray: '4 4' }}
            contentStyle={{
              borderRadius: 10,
              border: '1px solid #E2E8F0',
              boxShadow: '0 8px 24px rgba(15,23,42,.08)',
              fontSize: 12,
            }}
            formatter={(value: unknown, name: unknown) => [formatCurrency(Number(value)), String(name)]}
          />
          <Area
            type="monotone"
            dataKey="total_revenue"
            name="Total Pendapatan"
            stroke="#06B6D4"
            strokeWidth={2}
            fill="url(#report-revenue-fill)"
            activeDot={{ r: 4, strokeWidth: 2, stroke: '#FFFFFF' }}
          />
          <Area
            type="monotone"
            dataKey="paid_amount"
            name="Sudah Dibayar"
            stroke="#22C55E"
            strokeWidth={1.5}
            fill="none"
            activeDot={{ r: 3, strokeWidth: 2, stroke: '#FFFFFF' }}
          />
          <Area
            type="monotone"
            dataKey="unpaid_amount"
            name="Belum Dibayar"
            stroke="#F59E0B"
            strokeWidth={1.5}
            fill="none"
            activeDot={{ r: 3, strokeWidth: 2, stroke: '#FFFFFF' }}
          />
        </AreaChart>
      </ResponsiveContainer>
      <table className="sr-only">
        <caption>Pendapatan per periode</caption>
        <tbody>
          {points.map((point) => (
            <tr key={point.label}>
              <th scope="row">{point.label}</th>
              <td>{formatCurrency(point.total_revenue)}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
