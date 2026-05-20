import { useMemo } from 'react';
import { PieChart, Pie, Cell, ResponsiveContainer, Tooltip } from 'recharts';
import { Card, Stat } from '../components/Card';
import { Empty, ErrorState } from '../components/Empty';
import { useDeviceContext } from '../state/deviceContext';
import { useI18n } from '../state/i18nContext';
import { api } from '../lib/api';
import { useAsync } from '../hooks/useAsync';
import { formatDuration } from '../lib/format';
import { categoryColor } from '../lib/colors';
import { useChartTokens } from '../lib/chartTokens';

export function Dashboard() {
  const { selectedId, date } = useDeviceContext();
  const { t } = useI18n();
  const { data, loading, error } = useAsync(
    () => api.getSummary({ device_id: selectedId, date }),
    [selectedId, date],
  );

  const totalActive = data?.idle?.active_seconds ?? 0;
  const totalIdle = data?.idle?.idle_seconds ?? 0;
  const total = totalActive + totalIdle;
  const activeRatio = total > 0 ? Math.round((totalActive / total) * 100) : 0;

  const totalTracked = useMemo(
    () => (data?.categories ?? []).reduce((sum, c) => sum + c.duration, 0),
    [data],
  );

  if (loading) return <SkeletonGrid />;
  if (error) return <ErrorState error={error} />;
  if (!data) return <Empty />;

  return (
    <div className="space-y-6">
      <header>
        <h1 className="text-2xl font-semibold tracking-tight">{t('dashboard.title')}</h1>
        <p className="text-sm text-muted mt-1">{t('dashboard.subtitle', { date })}</p>
      </header>

      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
        <Card><Stat label={t('dashboard.tracked')} value={formatDuration(totalTracked)} sub={t('dashboard.trackedSub')} /></Card>
        <Card><Stat label={t('dashboard.active')} value={formatDuration(totalActive)} sub={t('dashboard.ofDay', { pct: activeRatio })} /></Card>
        <Card><Stat label={t('dashboard.idle')} value={formatDuration(totalIdle)} sub={total > 0 ? t('dashboard.ofDay', { pct: 100 - activeRatio }) : '—'} /></Card>
        <Card>
          <Stat
            label={t('dashboard.topCategory')}
            value={data.categories[0]?.category ?? '—'}
            sub={data.categories[0] ? formatDuration(data.categories[0].duration) : undefined}
          />
        </Card>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-4">
        <Card title={t('dashboard.categories')} subtitle={t('dashboard.categoriesSub')} className="lg:col-span-1">
          <CategoryDonut data={data.categories} />
        </Card>

        <Card title={t('dashboard.topApps')} subtitle={t('dashboard.topAppsSub')} className="lg:col-span-2">
          {data.top_apps.length === 0 ? (
            <Empty message={t('dashboard.noActivity')} />
          ) : (
            <ul className="space-y-2">
              {data.top_apps.map((app) => (
                <li key={app.process_name} className="flex items-center gap-3">
                  <span
                    className="h-2 w-2 rounded-full"
                    style={{ background: categoryColor(app.category || 'uncategorized') }}
                  />
                  <span className="text-sm text-fg flex-1 truncate" title={app.process_name}>
                    {app.process_name}
                  </span>
                  <span className="pill">{app.category || 'uncategorized'}</span>
                  <span className="font-mono text-sm text-fg/80 w-20 text-right">
                    {formatDuration(app.duration)}
                  </span>
                </li>
              ))}
            </ul>
          )}
        </Card>
      </div>

      <Card title={t('dashboard.topShortcuts')} subtitle={t('dashboard.topShortcutsSub')}>
        {data.shortcuts.length === 0 ? (
          <Empty message={t('dashboard.noActivity')} />
        ) : (
          <div className="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-5 gap-3">
            {data.shortcuts.map((sc, i) => (
              <div key={`${sc.shortcut}-${i}`} className="rounded-md border border-border bg-elevated p-3">
                <p className="font-mono text-sm text-fg">{sc.shortcut}</p>
                <p className="text-xs text-muted mt-1 truncate" title={sc.application}>
                  {sc.application || '—'}
                </p>
                <p className="font-mono text-xs mt-2 text-fg/80">×{sc.count}</p>
              </div>
            ))}
          </div>
        )}
      </Card>
    </div>
  );
}

function CategoryDonut({ data }: { data: { category: string; duration: number }[] }) {
  const tokens = useChartTokens();
  if (data.length === 0) return <Empty />;
  return (
    <div className="h-56">
      <ResponsiveContainer>
        <PieChart>
          <Pie
            data={data}
            dataKey="duration"
            nameKey="category"
            innerRadius={60}
            outerRadius={88}
            stroke={tokens.surface}
            strokeWidth={2}
          >
            {data.map((entry, i) => (
              <Cell key={i} fill={categoryColor(entry.category)} />
            ))}
          </Pie>
          <Tooltip
            contentStyle={{ background: tokens.tooltipBg, border: `1px solid ${tokens.border}`, fontSize: 12, color: tokens.fg }}
            formatter={(value: number, name: string) => [formatDuration(Number(value)), name]}
          />
        </PieChart>
      </ResponsiveContainer>
    </div>
  );
}

function SkeletonGrid() {
  return (
    <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
      {Array.from({ length: 4 }).map((_, i) => (
        <div key={i} className="card animate-pulse">
          <div className="h-3 w-16 bg-border rounded" />
          <div className="h-8 w-24 bg-border rounded mt-3" />
        </div>
      ))}
    </div>
  );
}
