import { useMemo } from 'react';
import {
  BarChart,
  Bar,
  XAxis,
  YAxis,
  ResponsiveContainer,
  Tooltip,
  CartesianGrid,
  AreaChart,
  Area,
} from 'recharts';
import { Card, Stat } from '../components/Card';
import { Empty, ErrorState } from '../components/Empty';
import { useDeviceContext } from '../state/deviceContext';
import { useI18n } from '../state/i18nContext';
import { api, HourlySwitch, FocusBlock } from '../lib/api';
import { useAsync } from '../hooks/useAsync';
import { formatDuration, formatTime, formatDate } from '../lib/format';
import { categoryColor } from '../lib/colors';
import { useChartTokens } from '../lib/chartTokens';

const MIN_FOCUS = 1500; // 25 minutes
const MAX_GAP = 120;

export function Focus() {
  const { selectedId } = useDeviceContext();
  const { t, locale } = useI18n();
  const tokens = useChartTokens();

  const focus = useAsync(
    () => api.getFocus({ device_id: selectedId, days: 14, min_duration: MIN_FOCUS, max_gap: MAX_GAP }),
    [selectedId],
  );
  const flow = useAsync(() => api.getFlow({ device_id: selectedId, days: 14 }), [selectedId]);
  const switches = useAsync(() => api.getSwitches({ device_id: selectedId, days: 28 }), [selectedId]);
  const sessions = useAsync(() => api.getSessions({ device_id: selectedId, days: 14 }), [selectedId]);

  const weekdayLabels =
    locale === 'zh'
      ? [t('focus.weekdaySun'), t('focus.weekdayMon'), t('focus.weekdayTue'), t('focus.weekdayWed'), t('focus.weekdayThu'), t('focus.weekdayFri'), t('focus.weekdaySat')]
      : ['Sun', 'Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat'];

  const dailyFocus = useMemo(() => {
    const days = focus.data?.daily ?? [];
    return days.map((d) => ({ ...d, label: d.date.slice(5) }));
  }, [focus.data]);

  const flowDays = useMemo(() => {
    const days = flow.data?.days ?? [];
    return days.map((d) => ({ ...d, label: d.date.slice(5) }));
  }, [flow.data]);

  const avgFlow = useMemo(() => {
    const days = flow.data?.days ?? [];
    if (!days.length) return 0;
    return Math.round(days.reduce((s, d) => s + d.score, 0) / days.length);
  }, [flow.data]);

  const avgSwitches = useMemo(() => {
    const daily = switches.data?.daily ?? [];
    if (!daily.length) return 0;
    return Math.round(daily.reduce((s, d) => s + d.switches, 0) / daily.length);
  }, [switches.data]);

  const sessionBars = useMemo(() => {
    const buckets = sessions.data?.buckets ?? [];
    return buckets.map((b) => ({ ...b, label: b.bucket }));
  }, [sessions.data]);

  return (
    <div className="space-y-6">
      <header>
        <h1 className="text-2xl font-semibold tracking-tight">{t('focus.title')}</h1>
        <p className="text-sm text-muted mt-1">{t('focus.subtitle')}</p>
      </header>

      <div className="grid grid-cols-2 lg:grid-cols-4 gap-4">
        <Card>
          <Stat
            label={t('focus.totalFocus')}
            value={focus.data ? formatDuration(focus.data.total_focus) : '—'}
            sub={t('focus.totalFocusSub', { min: Math.round(MIN_FOCUS / 60) })}
          />
        </Card>
        <Card>
          <Stat
            label={t('focus.longestBlock')}
            value={focus.data ? formatDuration(focus.data.longest) : '—'}
          />
        </Card>
        <Card>
          <Stat label={t('focus.avgFlow')} value={avgFlow > 0 ? `${avgFlow}` : '—'} sub="0–100" />
        </Card>
        <Card>
          <Stat label={t('focus.avgSwitches')} value={avgSwitches > 0 ? `${avgSwitches}` : '—'} />
        </Card>
      </div>

      <Card title={t('focus.flowScore')} subtitle={t('focus.flowScoreSub')}>
        {flow.loading ? (
          <p className="text-sm text-muted">{t('common.loading')}</p>
        ) : flow.error ? (
          <ErrorState error={flow.error} />
        ) : flowDays.length === 0 ? (
          <Empty />
        ) : (
          <div className="h-56">
            <ResponsiveContainer>
              <AreaChart data={flowDays} margin={{ left: 4, right: 4, top: 8 }}>
                <defs>
                  <linearGradient id="flowGrad" x1="0" y1="0" x2="0" y2="1">
                    <stop offset="0%" stopColor="#60a5fa" stopOpacity={0.5} />
                    <stop offset="100%" stopColor="#60a5fa" stopOpacity={0.05} />
                  </linearGradient>
                </defs>
                <CartesianGrid stroke={tokens.border} strokeDasharray="3 3" vertical={false} />
                <XAxis dataKey="label" tick={{ fontSize: 10, fill: tokens.muted }} stroke={tokens.border} />
                <YAxis tick={{ fontSize: 10, fill: tokens.muted }} stroke={tokens.border} domain={[0, 100]} />
                <Tooltip
                  contentStyle={{ background: tokens.tooltipBg, border: `1px solid ${tokens.border}`, fontSize: 12, color: tokens.fg }}
                  formatter={(v: number) => `${v}`}
                  labelFormatter={(label, payload) => payload?.[0]?.payload?.date ?? label}
                />
                <Area type="monotone" dataKey="score" stroke="#60a5fa" strokeWidth={2} fill="url(#flowGrad)" />
              </AreaChart>
            </ResponsiveContainer>
          </div>
        )}
      </Card>

      <Card title={t('focus.dailyBlocks')} subtitle={t('focus.dailyBlocksSub')}>
        {focus.loading ? (
          <p className="text-sm text-muted">{t('common.loading')}</p>
        ) : focus.error ? (
          <ErrorState error={focus.error} />
        ) : dailyFocus.length === 0 ? (
          <Empty />
        ) : (
          <div className="h-48">
            <ResponsiveContainer>
              <BarChart data={dailyFocus} margin={{ left: 4, right: 4, top: 8 }}>
                <CartesianGrid stroke={tokens.border} strokeDasharray="3 3" vertical={false} />
                <XAxis dataKey="label" tick={{ fontSize: 10, fill: tokens.muted }} stroke={tokens.border} />
                <YAxis tick={{ fontSize: 10, fill: tokens.muted }} stroke={tokens.border} tickFormatter={(v) => formatDuration(Number(v))} />
                <Tooltip
                  contentStyle={{ background: tokens.tooltipBg, border: `1px solid ${tokens.border}`, fontSize: 12, color: tokens.fg }}
                  formatter={(v: number) => formatDuration(Number(v))}
                  labelFormatter={(label, payload) => payload?.[0]?.payload?.date ?? label}
                />
                <Bar dataKey="duration" fill="#60a5fa" radius={[3, 3, 0, 0]} />
              </BarChart>
            </ResponsiveContainer>
          </div>
        )}
      </Card>

      <Card title={t('focus.switchGrid')} subtitle={t('focus.switchGridSub')}>
        {switches.loading ? (
          <p className="text-sm text-muted">{t('common.loading')}</p>
        ) : switches.error ? (
          <ErrorState error={switches.error} />
        ) : (switches.data?.hourly.length ?? 0) === 0 ? (
          <Empty />
        ) : (
          <SwitchGrid buckets={switches.data!.hourly} labels={weekdayLabels} />
        )}
      </Card>

      <Card title={t('focus.sessions')} subtitle={t('focus.sessionsSub')}>
        {sessions.loading ? (
          <p className="text-sm text-muted">{t('common.loading')}</p>
        ) : sessions.error ? (
          <ErrorState error={sessions.error} />
        ) : sessionBars.length === 0 ? (
          <Empty />
        ) : (
          <div className="h-48">
            <ResponsiveContainer>
              <BarChart data={sessionBars} margin={{ left: 4, right: 4, top: 8 }}>
                <CartesianGrid stroke={tokens.border} strokeDasharray="3 3" vertical={false} />
                <XAxis dataKey="label" tick={{ fontSize: 10, fill: tokens.muted }} stroke={tokens.border} />
                <YAxis tick={{ fontSize: 10, fill: tokens.muted }} stroke={tokens.border} />
                <Tooltip
                  contentStyle={{ background: tokens.tooltipBg, border: `1px solid ${tokens.border}`, fontSize: 12, color: tokens.fg }}
                  formatter={(v: number, _name: string, item: { payload?: { duration?: number } }) => [
                    `${v} (${formatDuration(item?.payload?.duration ?? 0)})`,
                    t('common.events'),
                  ]}
                />
                <Bar dataKey="count" fill="#a78bfa" radius={[3, 3, 0, 0]} />
              </BarChart>
            </ResponsiveContainer>
          </div>
        )}
      </Card>

      <Card title={t('focus.blocks')} subtitle={t('focus.blocksSub')}>
        {focus.loading ? (
          <p className="text-sm text-muted">{t('common.loading')}</p>
        ) : focus.error ? (
          <ErrorState error={focus.error} />
        ) : (focus.data?.blocks.length ?? 0) === 0 ? (
          <Empty />
        ) : (
          <BlockList blocks={focus.data!.blocks.slice(0, 20)} t={t} />
        )}
      </Card>
    </div>
  );
}

function SwitchGrid({ buckets, labels }: { buckets: HourlySwitch[]; labels: string[] }) {
  const grid = useMemo(() => {
    const m = new Map<string, number>();
    let max = 0;
    buckets.forEach((b) => {
      m.set(`${b.weekday}-${b.hour}`, b.switches);
      if (b.switches > max) max = b.switches;
    });
    return { m, max };
  }, [buckets]);

  return (
    <div className="space-y-1">
      <div className="flex items-center text-[10px] text-muted">
        <div className="w-10" />
        <div className="flex-1 grid" style={{ gridTemplateColumns: 'repeat(24, minmax(0, 1fr))' }}>
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
              const v = grid.m.get(`${w}-${h}`) ?? 0;
              const t = grid.max > 0 && v > 0 ? Math.min(1, Math.log(v + 1) / Math.log(grid.max + 1)) : 0;
              const bg =
                t === 0
                  ? 'rgb(var(--c-cell-empty))'
                  : `rgba(167, 139, 250, ${(0.18 + (1 - 0.18) * t).toFixed(3)})`;
              return (
                <div
                  key={h}
                  className="aspect-square rounded-sm"
                  style={{ background: bg }}
                  title={`${label} ${String(h).padStart(2, '0')}:00 · ${v} switches`}
                />
              );
            })}
          </div>
        </div>
      ))}
    </div>
  );
}

function BlockList({ blocks, t }: { blocks: FocusBlock[]; t: (k: string, p?: Record<string, string | number>) => string }) {
  const max = Math.max(...blocks.map((b) => b.duration), 1);
  return (
    <ul className="space-y-2">
      {blocks.map((b, i) => {
        const pct = (b.duration / max) * 100;
        const color = categoryColor(b.category);
        return (
          <li key={`${b.start_time}-${i}`} className="text-sm">
            <div className="flex items-baseline justify-between gap-3 mb-1">
              <span className="flex items-center gap-2 truncate flex-1">
                <span className="h-2 w-2 rounded-full shrink-0" style={{ background: color }} />
                <span className="capitalize truncate">{b.category}</span>
                <span className="text-xs text-muted">· {t('focus.appsCount', { n: b.apps })}</span>
              </span>
              <span className="font-mono text-xs text-fg/80 whitespace-nowrap">{formatDuration(b.duration)}</span>
            </div>
            <div className="relative h-1.5 rounded-sm bg-elevated overflow-hidden">
              <div
                className="absolute inset-y-0 left-0"
                style={{ width: `${pct}%`, background: color, opacity: 0.85 }}
              />
            </div>
            <p className="text-xs text-muted mt-1">
              {formatDate(b.start_time)} · {formatTime(b.start_time)} → {formatTime(b.end_time)}
            </p>
          </li>
        );
      })}
    </ul>
  );
}
