import { useMemo } from 'react';
import {
  AreaChart,
  Area,
  XAxis,
  YAxis,
  ResponsiveContainer,
  Tooltip,
  CartesianGrid,
  BarChart,
  Bar,
  Cell,
} from 'recharts';
import { Card, Stat } from '../components/Card';
import { Empty, ErrorState } from '../components/Empty';
import { useDeviceContext } from '../state/deviceContext';
import { useI18n } from '../state/i18nContext';
import { api, HourlyBucket } from '../lib/api';
import { useAsync } from '../hooks/useAsync';
import { formatDuration } from '../lib/format';
import { categoryColor } from '../lib/colors';
import { useChartTokens } from '../lib/chartTokens';

const DAY_LABELS = ['Sun', 'Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat'];
const DAY_LABELS_ZH = ['日', '一', '二', '三', '四', '五', '六'];

export function Insights() {
  const { selectedId } = useDeviceContext();
  const { t, locale } = useI18n();
  const tokens = useChartTokens();
  const hourly = useAsync(() => api.getHourly({ device_id: selectedId, days: 30 }), [selectedId]);
  const balance = useAsync(() => api.getBalance({ device_id: selectedId, days: 14 }), [selectedId]);

  const max = hourly.data?.max_duration ?? 0;
  const buckets = hourly.data?.buckets ?? [];
  const labels = locale === 'zh' ? DAY_LABELS_ZH : DAY_LABELS;

  const weekdayTotals = useMemo(() => {
    const totals = Array(7).fill(0) as number[];
    buckets.forEach((b) => {
      totals[b.weekday] += b.duration;
    });
    return totals.map((v, i) => ({ weekday: i, label: labels[i], duration: Math.round(v / 4) }));
  }, [buckets, labels]);

  const peakHour = useMemo(() => {
    if (buckets.length === 0) return null;
    const totals = Array(24).fill(0) as number[];
    buckets.forEach((b) => (totals[b.hour] += b.duration));
    let best = 0;
    for (let i = 1; i < 24; i++) if (totals[i] > totals[best]) best = i;
    return { hour: best, duration: totals[best] };
  }, [buckets]);

  const mostActiveDay = useMemo(() => {
    if (weekdayTotals.length === 0) return null;
    let best = 0;
    for (let i = 1; i < weekdayTotals.length; i++) {
      if (weekdayTotals[i].duration > weekdayTotals[best].duration) best = i;
    }
    return weekdayTotals[best];
  }, [weekdayTotals]);

  const balanceSeries = useMemo(() => buildBalanceSeries(balance.data?.rows ?? []), [balance.data]);

  return (
    <div className="space-y-6">
      <header>
        <h1 className="text-2xl font-semibold tracking-tight">{t('insights.title')}</h1>
        <p className="text-sm text-muted mt-1">{t('insights.subtitle')}</p>
      </header>

      <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
        <Card>
          <Stat
            label={t('insights.busyHour')}
            value={peakHour ? `${String(peakHour.hour).padStart(2, '0')}:00` : '—'}
            sub={peakHour ? formatDuration(peakHour.duration) : undefined}
          />
        </Card>
        <Card>
          <Stat
            label={t('insights.mostActiveDay')}
            value={mostActiveDay ? mostActiveDay.label : '—'}
            sub={mostActiveDay ? formatDuration(mostActiveDay.duration) : undefined}
          />
        </Card>
        <Card>
          <Stat
            label={t('insights.weekdayAvg')}
            value={
              weekdayTotals.length
                ? formatDuration(
                    Math.round(weekdayTotals.reduce((s, d) => s + d.duration, 0) / weekdayTotals.length),
                  )
                : '—'
            }
            sub={t('insights.weekdayAvgSub')}
          />
        </Card>
      </div>

      <Card title={t('insights.peakHours')} subtitle={t('insights.peakHoursSub')}>
        {hourly.loading ? (
          <p className="text-sm text-muted">{t('common.loading')}</p>
        ) : hourly.error ? (
          <ErrorState error={hourly.error} />
        ) : buckets.length === 0 ? (
          <Empty />
        ) : (
          <HourGrid buckets={buckets} max={max} labels={labels} />
        )}
      </Card>

      <Card title={t('insights.weekdayAvg')} subtitle={t('insights.weekdayAvgSub')}>
        {weekdayTotals.length === 0 ? (
          <Empty />
        ) : (
          <div className="h-48">
            <ResponsiveContainer>
              <BarChart data={weekdayTotals} margin={{ left: 4, right: 4, top: 8 }}>
                <CartesianGrid stroke={tokens.border} strokeDasharray="3 3" vertical={false} />
                <XAxis dataKey="label" stroke={tokens.border} tick={{ fontSize: 11, fill: tokens.muted }} />
                <YAxis stroke={tokens.border} tick={{ fontSize: 10, fill: tokens.muted }} tickFormatter={(v) => formatDuration(Number(v))} />
                <Tooltip
                  contentStyle={{ background: tokens.tooltipBg, border: `1px solid ${tokens.border}`, fontSize: 12, color: tokens.fg }}
                  formatter={(v: number) => formatDuration(Number(v))}
                />
                <Bar dataKey="duration" radius={[3, 3, 0, 0]}>
                  {weekdayTotals.map((_, i) => (
                    <Cell key={i} fill={`hsl(${(i * 50) % 360} 70% 60%)`} />
                  ))}
                </Bar>
              </BarChart>
            </ResponsiveContainer>
          </div>
        )}
      </Card>

      <Card title={t('insights.balance')} subtitle={t('insights.balanceSub')}>
        {balance.loading ? (
          <p className="text-sm text-muted">{t('common.loading')}</p>
        ) : balance.error ? (
          <ErrorState error={balance.error} />
        ) : balanceSeries.data.length === 0 ? (
          <Empty />
        ) : (
          <div className="h-72">
            <ResponsiveContainer>
              <AreaChart data={balanceSeries.data} margin={{ left: 4, right: 4, top: 8 }} stackOffset="expand">
                <CartesianGrid stroke={tokens.border} strokeDasharray="3 3" vertical={false} />
                <XAxis dataKey="label" stroke={tokens.border} tick={{ fontSize: 10, fill: tokens.muted }} />
                <YAxis
                  stroke={tokens.border}
                  tick={{ fontSize: 10, fill: tokens.muted }}
                  tickFormatter={(v) => `${Math.round(Number(v) * 100)}%`}
                />
                <Tooltip
                  contentStyle={{ background: tokens.tooltipBg, border: `1px solid ${tokens.border}`, fontSize: 12, color: tokens.fg }}
                  formatter={(v: number, name: string) => [formatDuration(Number(v)), name]}
                />
                {balanceSeries.keys.map((key) => (
                  <Area
                    key={key}
                    type="monotone"
                    dataKey={key}
                    stackId="1"
                    stroke={categoryColor(key)}
                    fill={categoryColor(key)}
                    fillOpacity={0.7}
                  />
                ))}
              </AreaChart>
            </ResponsiveContainer>
          </div>
        )}
      </Card>
    </div>
  );
}

function HourGrid({ buckets, max, labels }: { buckets: HourlyBucket[]; max: number; labels: string[] }) {
  const grid = useMemo(() => {
    const m = new Map<string, HourlyBucket>();
    buckets.forEach((b) => m.set(`${b.weekday}-${b.hour}`, b));
    return m;
  }, [buckets]);

  return (
    <div className="space-y-1">
      <div className="flex items-center text-[10px] text-muted">
        <div className="w-10" />
        <div className="flex-1 grid grid-cols-24" style={{ gridTemplateColumns: 'repeat(24, minmax(0, 1fr))' }}>
          {Array.from({ length: 24 }).map((_, h) => (
            <span key={h} className="text-center">
              {h % 3 === 0 ? h : ''}
            </span>
          ))}
        </div>
      </div>
      {labels.map((label, w) => (
        <div key={w} className="flex items-center gap-2">
          <span className="w-10 text-xs text-muted">{label}</span>
          <div className="flex-1 grid gap-[2px]" style={{ gridTemplateColumns: 'repeat(24, minmax(0, 1fr))' }}>
            {Array.from({ length: 24 }).map((_, h) => {
              const b = grid.get(`${w}-${h}`);
              const dur = b?.duration ?? 0;
              const t = max > 0 && dur > 0 ? Math.min(1, Math.log(dur + 1) / Math.log(max + 1)) : 0;
              const bg =
                t === 0
                  ? 'rgb(var(--c-cell-empty))'
                  : `rgba(96, 165, 250, ${(0.18 + (1 - 0.18) * t).toFixed(3)})`;
              return (
                <div
                  key={h}
                  className="aspect-square rounded-sm"
                  style={{ background: bg }}
                  title={`${label} ${String(h).padStart(2, '0')}:00 · ${formatDuration(dur)}`}
                />
              );
            })}
          </div>
        </div>
      ))}
    </div>
  );
}

function buildBalanceSeries(rows: { date: string; category: string; duration: number }[]) {
  const byDate = new Map<string, Record<string, number>>();
  const cats = new Set<string>();
  rows.forEach((r) => {
    cats.add(r.category);
    const entry = byDate.get(r.date) ?? {};
    entry[r.category] = (entry[r.category] || 0) + r.duration;
    byDate.set(r.date, entry);
  });
  const dates = Array.from(byDate.keys()).sort();
  const keys = Array.from(cats);
  const data = dates.map((date) => {
    const entry = byDate.get(date) ?? {};
    const filled: Record<string, number | string> = { date, label: date.slice(5) };
    keys.forEach((k) => (filled[k] = entry[k] || 0));
    return filled;
  });
  return { data, keys };
}
