import { useMemo, useState } from 'react';
import { BarChart, Bar, Cell, XAxis, YAxis, ResponsiveContainer, Tooltip, CartesianGrid } from 'recharts';
import { Card } from '../components/Card';
import { Empty, ErrorState } from '../components/Empty';
import { useDeviceContext } from '../state/deviceContext';
import { api } from '../lib/api';
import { useAsync } from '../hooks/useAsync';
import { formatDuration } from '../lib/format';
import { categoryColor } from '../lib/colors';

export function Categories() {
  const { selectedId, date } = useDeviceContext();
  const cats = useAsync(() => api.getCategories({ device_id: selectedId, date }), [selectedId, date]);
  const apps = useAsync(() => api.getApps({ device_id: selectedId, date, limit: 50 }), [selectedId, date]);
  const [filter, setFilter] = useState<string>('');

  const total = useMemo(
    () => (cats.data?.categories ?? []).reduce((s, c) => s + c.duration, 0),
    [cats.data],
  );

  if (cats.loading || apps.loading) return <p className="text-sm text-muted">Loading…</p>;
  if (cats.error) return <ErrorState error={cats.error} />;
  if (apps.error) return <ErrorState error={apps.error} />;

  const filtered = (apps.data?.apps ?? []).filter((a) => !filter || a.category === filter);

  return (
    <div className="space-y-6">
      <header>
        <h1 className="text-2xl font-semibold tracking-tight">Categories</h1>
        <p className="text-sm text-muted mt-1">Time spent by category for {date}</p>
      </header>

      <Card title="Breakdown" subtitle={`Total tracked: ${formatDuration(total)}`}>
        {(cats.data?.categories ?? []).length === 0 ? (
          <Empty />
        ) : (
          <div className="h-64">
            <ResponsiveContainer>
              <BarChart data={cats.data!.categories} layout="vertical" margin={{ left: 12, right: 12 }}>
                <CartesianGrid stroke="#1f1f1f" strokeDasharray="3 3" />
                <XAxis
                  type="number"
                  stroke="#666"
                  tick={{ fontSize: 11 }}
                  tickFormatter={(v) => formatDuration(Number(v))}
                />
                <YAxis dataKey="category" type="category" stroke="#666" tick={{ fontSize: 11 }} width={100} />
                <Tooltip
                  contentStyle={{ background: '#0a0a0a', border: '1px solid #1f1f1f', fontSize: 12 }}
                  formatter={(v: number) => formatDuration(Number(v))}
                  labelStyle={{ color: '#fafafa' }}
                />
                <Bar dataKey="duration" radius={[0, 4, 4, 0]}>
                  {cats.data!.categories.map((c, i) => (
                    <Cell key={i} fill={categoryColor(c.category)} />
                  ))}
                </Bar>
              </BarChart>
            </ResponsiveContainer>
          </div>
        )}
      </Card>

      <Card
        title="Apps"
        subtitle={`${filtered.length} apps`}
        action={
          <select className="input" value={filter} onChange={(e) => setFilter(e.target.value)}>
            <option value="">All categories</option>
            {(cats.data?.categories ?? []).map((c) => (
              <option key={c.category} value={c.category}>
                {c.category}
              </option>
            ))}
          </select>
        }
      >
        {filtered.length === 0 ? (
          <Empty />
        ) : (
          <table className="w-full table">
            <thead>
              <tr>
                <th className="py-2 px-3">App</th>
                <th className="py-2 px-3">Category</th>
                <th className="py-2 px-3 text-right">Events</th>
                <th className="py-2 px-3 text-right">Duration</th>
              </tr>
            </thead>
            <tbody>
              {filtered.map((a) => (
                <tr key={a.process_name} className="border-t border-border">
                  <td className="py-2 px-3 truncate max-w-[260px]" title={a.process_name}>
                    {a.process_name}
                  </td>
                  <td className="py-2 px-3">
                    <span className="pill" style={{ borderColor: categoryColor(a.category || 'uncategorized') }}>
                      {a.category || 'uncategorized'}
                    </span>
                  </td>
                  <td className="py-2 px-3 text-right font-mono">{a.count}</td>
                  <td className="py-2 px-3 text-right font-mono">{formatDuration(a.duration)}</td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </Card>
    </div>
  );
}
