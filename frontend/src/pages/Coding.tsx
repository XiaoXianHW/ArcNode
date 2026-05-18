import { useMemo } from 'react';
import { BarChart, Bar, XAxis, YAxis, ResponsiveContainer, Tooltip, CartesianGrid } from 'recharts';
import { Card, Stat } from '../components/Card';
import { Empty, ErrorState } from '../components/Empty';
import { Heatmap } from '../components/Heatmap';
import { useDeviceContext } from '../state/deviceContext';
import { api } from '../lib/api';
import { useAsync } from '../hooks/useAsync';
import { formatDuration, todayISO } from '../lib/format';
import { categoryColor } from '../lib/colors';

const CATEGORY = 'coding';

export function Coding() {
  return <CategoryPage category={CATEGORY} title="Coding" subtitle="Time at the keyboard across IDEs, editors and terminals" />;
}

interface PageProps {
  category: string;
  title: string;
  subtitle: string;
}

export function CategoryPage({ category, title, subtitle }: PageProps) {
  const { selectedId } = useDeviceContext();
  const color = categoryColor(category);

  const heat = useAsync(
    () => api.getHeatmap({ device_id: selectedId, category, days: 365 }),
    [selectedId, category],
  );
  const daily30 = useAsync(
    () => api.getDaily({ device_id: selectedId, category, days: 30 }),
    [selectedId, category],
  );
  const today = todayISO();
  const apps7 = useAsync(
    () =>
      api
        .getDaily({ device_id: selectedId, category, days: 7 })
        .then(() => api.getApps({ device_id: selectedId, limit: 50 })),
    [selectedId, category],
  );
  const projects = useAsync(
    () => api.getProjects({ device_id: selectedId, category, limit: 15, date: today }),
    [selectedId, category, today],
  );

  const filteredApps = useMemo(
    () => (apps7.data?.apps ?? []).filter((a) => a.category === category).slice(0, 12),
    [apps7.data, category],
  );

  const daily30Display = useMemo(() => {
    const list = daily30.data?.days ?? [];
    return list.map((d) => ({ ...d, label: d.date.slice(5) }));
  }, [daily30]);

  return (
    <div className="space-y-6">
      <header className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-semibold tracking-tight">{title}</h1>
          <p className="text-sm text-muted mt-1">{subtitle}</p>
        </div>
        <span
          className="h-3 w-3 rounded-full"
          style={{ background: color, boxShadow: `0 0 16px ${color}55` }}
        />
      </header>

      <div className="grid grid-cols-2 lg:grid-cols-4 gap-4">
        <Card>
          <Stat
            label="Total (year)"
            value={heat.data ? formatDuration(heat.data.total_duration) : '—'}
            sub={heat.data ? `${heat.data.active_days} active days` : undefined}
          />
        </Card>
        <Card>
          <Stat
            label="Current streak"
            value={heat.data ? `${heat.data.current_streak}d` : '—'}
            sub={heat.data ? `longest ${heat.data.longest_streak}d` : undefined}
          />
        </Card>
        <Card>
          <Stat
            label="Best day"
            value={heat.data ? formatDuration(heat.data.max_duration) : '—'}
          />
        </Card>
        <Card>
          <Stat
            label="Daily avg (30d)"
            value={
              daily30.data
                ? formatDuration(
                    Math.round(
                      (daily30.data.days.reduce((s, d) => s + d.duration, 0) || 0) /
                        Math.max(daily30.data.days.length, 1),
                    ),
                  )
                : '—'
            }
          />
        </Card>
      </div>

      <Card title="Contributions" subtitle="Daily activity over the last year">
        {heat.loading ? (
          <p className="text-sm text-muted">Loading…</p>
        ) : heat.error ? (
          <ErrorState error={heat.error} />
        ) : (heat.data?.days.length ?? 0) === 0 ? (
          <Empty message="No activity recorded yet" />
        ) : (
          <Heatmap days={heat.data!.days} start={heat.data!.start} end={heat.data!.end} color={color} weeks={53} />
        )}
      </Card>

      <Card title="Last 30 days" subtitle="Daily time spent">
        {daily30.loading ? (
          <p className="text-sm text-muted">Loading…</p>
        ) : daily30.error ? (
          <ErrorState error={daily30.error} />
        ) : daily30Display.length === 0 ? (
          <Empty />
        ) : (
          <div className="h-56">
            <ResponsiveContainer>
              <BarChart data={daily30Display} margin={{ left: 4, right: 4, top: 8 }}>
                <CartesianGrid stroke="#1f1f1f" strokeDasharray="3 3" vertical={false} />
                <XAxis dataKey="label" tick={{ fontSize: 10, fill: '#666' }} interval={2} stroke="#333" />
                <YAxis
                  tick={{ fontSize: 10, fill: '#666' }}
                  stroke="#333"
                  tickFormatter={(v) => formatDuration(Number(v))}
                />
                <Tooltip
                  contentStyle={{ background: '#0a0a0a', border: '1px solid #1f1f1f', fontSize: 12 }}
                  formatter={(v: number) => formatDuration(Number(v))}
                  labelFormatter={(label, payload) => payload?.[0]?.payload?.date ?? label}
                />
                <Bar dataKey="duration" fill={color} radius={[3, 3, 0, 0]} />
              </BarChart>
            </ResponsiveContainer>
          </div>
        )}
      </Card>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
        <Card title="Top apps" subtitle={`Most-used ${category} apps`}>
          {apps7.loading ? (
            <p className="text-sm text-muted">Loading…</p>
          ) : apps7.error ? (
            <ErrorState error={apps7.error} />
          ) : filteredApps.length === 0 ? (
            <Empty />
          ) : (
            <RankList
              items={filteredApps.map((a) => ({
                primary: a.process_name,
                secondary: `${a.count} sessions`,
                value: a.duration,
              }))}
              color={color}
            />
          )}
        </Card>

        <Card title="Top windows today" subtitle="What you focused on today">
          {projects.loading ? (
            <p className="text-sm text-muted">Loading…</p>
          ) : projects.error ? (
            <ErrorState error={projects.error} />
          ) : (projects.data?.projects.length ?? 0) === 0 ? (
            <Empty />
          ) : (
            <RankList
              items={projects.data!.projects.map((p) => ({
                primary: p.window_title || p.process_name,
                secondary: p.process_name,
                value: p.duration,
              }))}
              color={color}
            />
          )}
        </Card>
      </div>
    </div>
  );
}

function RankList({
  items,
  color,
}: {
  items: { primary: string; secondary?: string; value: number }[];
  color: string;
}) {
  const max = Math.max(...items.map((i) => i.value), 1);
  return (
    <ul className="space-y-2">
      {items.map((it, i) => {
        const pct = (it.value / max) * 100;
        return (
          <li key={`${it.primary}-${i}`} className="text-sm">
            <div className="flex items-baseline justify-between gap-3 mb-1">
              <span className="truncate flex-1" title={it.primary}>
                {it.primary}
              </span>
              <span className="font-mono text-xs text-fg/80 whitespace-nowrap">{formatDuration(it.value)}</span>
            </div>
            <div className="relative h-1.5 rounded-sm bg-elevated overflow-hidden">
              <div
                className="absolute inset-y-0 left-0"
                style={{ width: `${pct}%`, background: color, opacity: 0.85 }}
              />
            </div>
            {it.secondary && <p className="text-xs text-muted mt-1 truncate">{it.secondary}</p>}
          </li>
        );
      })}
    </ul>
  );
}
