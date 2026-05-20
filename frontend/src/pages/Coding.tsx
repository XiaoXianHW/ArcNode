import { useMemo } from 'react';
import { BarChart, Bar, XAxis, YAxis, ResponsiveContainer, Tooltip, CartesianGrid } from 'recharts';
import { Card, Stat } from '../components/Card';
import { Empty, ErrorState } from '../components/Empty';
import { Heatmap } from '../components/Heatmap';
import { LanguageBar } from '../components/LanguageBar';
import { useDeviceContext } from '../state/deviceContext';
import { useI18n } from '../state/i18nContext';
import { api } from '../lib/api';
import { useAsync } from '../hooks/useAsync';
import { formatDuration, todayISO } from '../lib/format';
import { categoryColor } from '../lib/colors';
import { useChartTokens } from '../lib/chartTokens';

const CATEGORY = 'coding';

export function Coding() {
  const { t } = useI18n();
  return (
    <CategoryPage
      category={CATEGORY}
      title={t('coding.title')}
      subtitle={t('coding.subtitle')}
      showLanguages
    />
  );
}

interface PageProps {
  category: string;
  title: string;
  subtitle: string;
  showLanguages?: boolean;
}

export function CategoryPage({ category, title, subtitle, showLanguages }: PageProps) {
  const { selectedId } = useDeviceContext();
  const { t } = useI18n();
  const tokens = useChartTokens();
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
  const languages = useAsync(
    () =>
      showLanguages
        ? api.getLanguages({ device_id: selectedId, days: 30 })
        : Promise.resolve({ languages: [], start: 0, end: 0 }),
    [selectedId, showLanguages],
  );
  const files = useAsync(
    () =>
      showLanguages
        ? api.getFiles({ device_id: selectedId, days: 14, limit: 15 })
        : Promise.resolve({ files: [], start: 0, end: 0 }),
    [selectedId, showLanguages],
  );
  const projectsDaily = useAsync(
    () =>
      showLanguages
        ? api.getProjectsDaily({ device_id: selectedId, category, days: 14 })
        : Promise.resolve({ rows: [], start: 0, end: 0 }),
    [selectedId, category, showLanguages],
  );

  const topProjectNames = useMemo(() => {
    if (!showLanguages) return [] as string[];
    const totals = new Map<string, number>();
    (projectsDaily.data?.rows ?? []).forEach((r) => {
      totals.set(r.project, (totals.get(r.project) || 0) + r.duration);
    });
    return Array.from(totals.entries())
      .sort((a, b) => b[1] - a[1])
      .slice(0, 6)
      .map(([name]) => name);
  }, [projectsDaily.data, showLanguages]);

  const projectDailySeries = useMemo(() => {
    if (!showLanguages || !projectsDaily.data) return [] as Array<Record<string, number | string>>;
    const allowed = new Set(topProjectNames);
    const byDate = new Map<string, Record<string, number>>();
    projectsDaily.data.rows.forEach((r) => {
      if (!allowed.has(r.project)) return;
      const entry = byDate.get(r.date) ?? {};
      entry[r.project] = (entry[r.project] || 0) + r.duration;
      byDate.set(r.date, entry);
    });
    const dates = Array.from(byDate.keys()).sort();
    return dates.map((date) => {
      const entry = byDate.get(date) ?? {};
      const row: Record<string, number | string> = { date, label: date.slice(5) };
      topProjectNames.forEach((p) => (row[p] = entry[p] || 0));
      return row;
    });
  }, [projectsDaily.data, topProjectNames, showLanguages]);

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
            label={t('coding.totalYear')}
            value={heat.data ? formatDuration(heat.data.total_duration) : '—'}
            sub={heat.data ? t('coding.activeDays', { n: heat.data.active_days }) : undefined}
          />
        </Card>
        <Card>
          <Stat
            label={t('coding.currentStreak')}
            value={heat.data ? `${heat.data.current_streak}${t('common.day')}` : '—'}
            sub={heat.data ? t('coding.longestStreak', { n: heat.data.longest_streak }) : undefined}
          />
        </Card>
        <Card>
          <Stat
            label={t('coding.bestDay')}
            value={heat.data ? formatDuration(heat.data.max_duration) : '—'}
          />
        </Card>
        <Card>
          <Stat
            label={t('coding.dailyAvg')}
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

      <Card title={t('coding.contributions')} subtitle={t('coding.contributionsSub')}>
        {heat.loading ? (
          <p className="text-sm text-muted">{t('common.loading')}</p>
        ) : heat.error ? (
          <ErrorState error={heat.error} />
        ) : (heat.data?.days.length ?? 0) === 0 ? (
          <Empty />
        ) : (
          <Heatmap days={heat.data!.days} start={heat.data!.start} end={heat.data!.end} color={color} weeks={53} />
        )}
      </Card>

      {showLanguages && (
        <Card title={t('coding.languages')} subtitle={t('coding.languagesSub')}>
          {languages.loading ? (
            <p className="text-sm text-muted">{t('common.loading')}</p>
          ) : languages.error ? (
            <ErrorState error={languages.error} />
          ) : (languages.data?.languages.length ?? 0) === 0 ? (
            <Empty />
          ) : (
            <LanguageBar items={languages.data!.languages} />
          )}
        </Card>
      )}

      <Card title={t('coding.last30')} subtitle={t('coding.last30Sub')}>
        {daily30.loading ? (
          <p className="text-sm text-muted">{t('common.loading')}</p>
        ) : daily30.error ? (
          <ErrorState error={daily30.error} />
        ) : daily30Display.length === 0 ? (
          <Empty />
        ) : (
          <div className="h-56">
            <ResponsiveContainer>
              <BarChart data={daily30Display} margin={{ left: 4, right: 4, top: 8 }}>
                <CartesianGrid stroke={tokens.border} strokeDasharray="3 3" vertical={false} />
                <XAxis dataKey="label" tick={{ fontSize: 10, fill: tokens.muted }} interval={2} stroke={tokens.border} />
                <YAxis
                  tick={{ fontSize: 10, fill: tokens.muted }}
                  stroke={tokens.border}
                  tickFormatter={(v) => formatDuration(Number(v))}
                />
                <Tooltip
                  contentStyle={{ background: tokens.tooltipBg, border: `1px solid ${tokens.border}`, fontSize: 12, color: tokens.fg }}
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
        <Card title={t('coding.topApps')} subtitle={t('coding.topAppsSub', { category })}>
          {apps7.loading ? (
            <p className="text-sm text-muted">{t('common.loading')}</p>
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

        <Card title={t('coding.topWindows')} subtitle={t('coding.topWindowsSub')}>
          {projects.loading ? (
            <p className="text-sm text-muted">{t('common.loading')}</p>
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

      {showLanguages && (
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
          <Card title={t('coding.files')} subtitle={t('coding.filesSub')}>
            {files.loading ? (
              <p className="text-sm text-muted">{t('common.loading')}</p>
            ) : files.error ? (
              <ErrorState error={files.error} />
            ) : (files.data?.files.length ?? 0) === 0 ? (
              <Empty />
            ) : (
              <RankList
                items={files.data!.files.map((f) => ({
                  primary: f.file,
                  secondary: f.language ? `.${f.language}` : undefined,
                  value: f.duration,
                }))}
                color={color}
              />
            )}
          </Card>

          <Card title={t('coding.projectsDaily')} subtitle={t('coding.projectsDailySub')}>
            {projectsDaily.loading ? (
              <p className="text-sm text-muted">{t('common.loading')}</p>
            ) : projectsDaily.error ? (
              <ErrorState error={projectsDaily.error} />
            ) : projectDailySeries.length === 0 || topProjectNames.length === 0 ? (
              <Empty />
            ) : (
              <div className="h-56">
                <ResponsiveContainer>
                  <BarChart data={projectDailySeries} margin={{ left: 4, right: 4, top: 8 }}>
                    <CartesianGrid stroke={tokens.border} strokeDasharray="3 3" vertical={false} />
                    <XAxis dataKey="label" tick={{ fontSize: 10, fill: tokens.muted }} stroke={tokens.border} />
                    <YAxis
                      tick={{ fontSize: 10, fill: tokens.muted }}
                      stroke={tokens.border}
                      tickFormatter={(v) => formatDuration(Number(v))}
                    />
                    <Tooltip
                      contentStyle={{ background: tokens.tooltipBg, border: `1px solid ${tokens.border}`, fontSize: 12, color: tokens.fg }}
                      formatter={(v: number, name: string) => [formatDuration(Number(v)), name]}
                    />
                    {topProjectNames.map((name, i) => (
                      <Bar
                        key={name}
                        dataKey={name}
                        stackId="p"
                        fill={`hsl(${(i * 47) % 360} 65% 60%)`}
                        radius={i === topProjectNames.length - 1 ? [3, 3, 0, 0] : undefined}
                      />
                    ))}
                  </BarChart>
                </ResponsiveContainer>
              </div>
            )}
          </Card>
        </div>
      )}
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
