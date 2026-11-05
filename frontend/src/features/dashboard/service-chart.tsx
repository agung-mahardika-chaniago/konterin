import { Bar, BarChart, CartesianGrid, ResponsiveContainer, Tooltip, XAxis, YAxis } from 'recharts';

import { formatNumber } from './format';
import type { DashboardChartDay } from './types';

export function ServiceChart({ days }: { days: DashboardChartDay[] }) {
  const hasData = days.some((day) => day.services_received > 0 || day.services_completed > 0);

  return (
    <div className="relative h-[260px] w-full" role="img" aria-label="Grafik servis masuk dan selesai">
      {!hasData && (
        <p className="absolute inset-x-0 top-1/2 z-10 -translate-y-1/2 text-center text-xs text-muted">
          Belum ada servis pada rentang ini.
        </p>
      )}
      <ResponsiveContainer width="100%" height="100%">
        <BarChart data={days} margin={{ top: 8, right: 8, bottom: 0, left: 0 }} barGap={2}>
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
            width={32}
            allowDecimals={false}
            tickLine={false}
            axisLine={false}
            tick={{ fontSize: 11, fill: '#64748B' }}
          />
          <Tooltip
            cursor={{ fill: '#F6F8FB' }}
            contentStyle={{
              borderRadius: 10,
              border: '1px solid #E2E8F0',
              boxShadow: '0 8px 24px rgba(15,23,42,.08)',
              fontSize: 12,
            }}
            labelFormatter={(label) => `Tanggal ${String(label)}`}
            formatter={(value, name) => [formatNumber(Number(value)), String(name)]}
          />
          <Bar dataKey="services_received" name="Servis masuk" fill="#164278" radius={[4, 4, 0, 0]} maxBarSize={18} />
          <Bar dataKey="services_completed" name="Servis selesai" fill="#06B6D4" radius={[4, 4, 0, 0]} maxBarSize={18} />
        </BarChart>
      </ResponsiveContainer>
      <table className="sr-only">
        <caption>Servis masuk dan selesai per hari</caption>
        <tbody>
          {days.map((day) => (
            <tr key={day.date}>
              <th scope="row">{day.date}</th>
              <td>{day.services_received} masuk</td>
              <td>{day.services_completed} selesai</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
