import { Area, AreaChart, CartesianGrid, ResponsiveContainer, Tooltip, XAxis, YAxis } from 'recharts';

import { formatCompactCurrency, formatCurrency } from './format';
import type { DashboardChartDay } from './types';

export function RevenueChart({ days }: { days: DashboardChartDay[] }) {
  const hasData = days.some((day) => day.revenue > 0);

  return (
    <div className="relative h-[260px] w-full" role="img" aria-label="Grafik pendapatan harian">
      {!hasData && (
        <p className="absolute inset-x-0 top-1/2 z-10 -translate-y-1/2 text-center text-xs text-muted">
          Belum ada pendapatan pada rentang ini.
        </p>
      )}
      <ResponsiveContainer width="100%" height="100%">
        <AreaChart data={days} margin={{ top: 8, right: 8, bottom: 0, left: 0 }}>
          <defs>
            <linearGradient id="revenue-fill" x1="0" y1="0" x2="0" y2="1">
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
            labelFormatter={(label) => `Tanggal ${String(label)}`}
            formatter={(value) => [formatCurrency(Number(value)), 'Pendapatan']}
          />
          <Area
            type="monotone"
            dataKey="revenue"
            name="Pendapatan"
            stroke="#06B6D4"
            strokeWidth={2}
            fill="url(#revenue-fill)"
            activeDot={{ r: 4, strokeWidth: 2, stroke: '#FFFFFF' }}
          />
        </AreaChart>
      </ResponsiveContainer>
      <table className="sr-only">
        <caption>Pendapatan harian</caption>
        <tbody>
          {days.map((day) => (
            <tr key={day.date}>
              <th scope="row">{day.date}</th>
              <td>{formatCurrency(day.revenue)}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
