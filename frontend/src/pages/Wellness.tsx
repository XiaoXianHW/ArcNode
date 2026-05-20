import { useMemo } from 'react';
import {
  BarChart,
  Bar,
  XAxis,
  YAxis,
  ResponsiveContainer,
  Tooltip,
  CartesianGrid,
  Cell,
} from 'recharts';
import { Card, Stat } from '../components/Card';
import { Empty, ErrorState } from '../components/Empty';
import { useDeviceContext } from '../state/deviceContext';
import { useI18n } from '../state/i18nContext';
import { api } from '../lib/api';
import { useAsync } from '../hooks/useAsync';
import { formatDuration } from '../lib/format';
import { useChartTokens } from '../lib/chartTokens';

const SEDENTARY_THRESHOLD_SECONDS = 3600;

export function Wellness() {
  const { selectedId } = useDeviceContext();
  const { t } = useI18n();
  const tokens = useChartTokens();

  const idle = useAsync(() => api.getIdleRatio({ device_id: selectedId, days: 14 }), [selectedId]);
  const sed = useAsync(
    () => api.getSedentary({ device_id: selectedId, days: 14, threshold: SEDENTARY_THRESHOLD_SECONDS }),
    [selectedId],
  );
  const video = useAsync(() => api.getVideoStats({ device_id: selectedId, days: 30 }), [selectedId]);

  const idleDays = useMemo(() => {
    return (idle.data?.days ?? []).map((d) => ({ ...d, label: d.date.slice(5) }));
  }, [idle.data]);

  const sedDays = useMemo(() => {
    return (sed.data?.days ?? []).map((d) => ({ ...d, label: d.date.slice(5) }));
  }, [sed.data]);

  const totals = useMemo(() => {
    const days = idle.data?.days ?? [];
    return days.reduce(
      (acc, d) => {
        acc.active += d.active;
        acc.idle += d.idle;
        return acc;
      },
      { active: 0, idle: 0 },
    );
  }, [idle.data]);

  const sedSummary = useMemo(() => {
    const days = sed.data?.days ?? [];
    const flagged = days.filter((d) => d.stretches_over_threshold > 0).length;
    const longest = days.reduce((s, d) => Math.max(s, d.longest_stretch), 0);
    return { flagged, longest };
  }, [sed.data]);

  const videoTotal = useMemo(() => {
    return (video.data?.platforms ?? []).reduce((s, p) => s + p.duration, 0);
  }, [video.data]);

  return (
    <div className="space-y-6">
      <header>
        <h1 className="text-2xl font-semibold tracking-tight">{t('wellness.title')}</h1>
        <p className="text-sm text-muted mt-1">{t('wellness.subtitle')}</p>
      </header>

      <div className="grid grid-cols-2 lg:grid-cols-4 gap-4">
        <Card>
          <Stat label={t('wellness.activeWeek')} value={formatDuration(totals.active)} />
        </Card>
        <Card>
          <Stat label={t('wellness.idleWeek')} value={formatDuration(totals.idle)} />
        </Card>
        <Card>
          <Stat
            label={t('wellness.sedentaryDays')}
            value={`${sedSummary.flagged}`}
            sub={t('wellness.sedentaryDaysSub', { n: Math.round(SEDENTARY_THRESHOLD_SECONDS / 60) })}
          />
        </Card>
        <Card>
          <Stat
            label={t('wellness.longestStretch')}
            value={sedSummary.longest > 0 ? formatDuration(sedSummary.longest) : '—'}
          />
        </Card>
      </div>

      <Card title={t('wellness.idleRatio')} subtitle={t('wellness.idleRatioSub')}>
        {idle.loading ? (
          <p className="text-sm text-muted">{t('common.loading')}</p>
        ) : idle.error ? (
          <ErrorState error={idle.error} />
        ) : idleDays.length === 0 ? (
          <Empty />
        ) : (
          <div className="h-56">
            <ResponsiveContainer>
              <BarChart data={idleDays} margin={{ left: 4, right: 4, top: 8 }}>
                <CartesianGrid stroke={tokens.border} strokeDasharray="3 3" vertical={false} />
                <XAxis dataKey="label" tick={{ fontSize: 10, fill: tokens.muted }} stroke={tokens.border} />
                <YAxis tick={{ fontSize: 10, fill: tokens.muted }} stroke={tokens.border} tickFormatter={(v) => formatDuration(Number(v))} />
                <Tooltip
                  contentStyle={{ background: tokens.tooltipBg, border: `1px solid ${tokens.border}`, fontSize: 12, color: tokens.fg }}
                  formatter={(v: number, name: string) => [formatDuration(Number(v)), name]}
                  labelFormatter={(label, payload) => payload?.[0]?.payload?.date ?? label}
                />
                <Bar dataKey="active" stackId="t" name={t('profile.active')} fill="#34d399" radius={[0, 0, 0, 0]} />
                <Bar dataKey="idle" stackId="t" name={t('profile.idle')} fill="#475569" radius={[3, 3, 0, 0]} />
              </BarChart>
            </ResponsiveContainer>
          </div>
        )}
      </Card>

      <Card title={t('wellness.sedentary')} subtitle={t('wellness.sedentarySub')}>
        {sed.loading ? (
          <p className="text-sm text-muted">{t('common.loading')}</p>
        ) : sed.error ? (
          <ErrorState error={sed.error} />
        ) : sedDays.length === 0 ? (
          <Empty />
        ) : (
          <div className="h-56">
            <ResponsiveContainer>
              <BarChart data={sedDays} margin={{ left: 4, right: 4, top: 8 }}>
                <CartesianGrid stroke={tokens.border} strokeDasharray="3 3" vertical={false} />
                <XAxis dataKey="label" tick={{ fontSize: 10, fill: tokens.muted }} stroke={tokens.border} />
                <YAxis tick={{ fontSize: 10, fill: tokens.muted }} stroke={tokens.border} tickFormatter={(v) => formatDuration(Number(v))} />
                <Tooltip
                  contentStyle={{ background: tokens.tooltipBg, border: `1px solid ${tokens.border}`, fontSize: 12, color: tokens.fg }}
                  formatter={(v: number) => formatDuration(Number(v))}
                  labelFormatter={(label, payload) => payload?.[0]?.payload?.date ?? label}
                />
                <Bar dataKey="longest_stretch" radius={[3, 3, 0, 0]} name={t('wellness.longestStretch')}>
                  {sedDays.map((d, i) => (
                    <Cell key={i} fill={d.stretches_over_threshold > 0 ? '#ef4444' : '#fb923c'} />
                  ))}
                </Bar>
              </BarChart>
            </ResponsiveContainer>
          </div>
        )}
      </Card>

      <Card title={t('wellness.video')} subtitle={t('wellness.videoSub')} action={<span className="text-xs text-muted">{t('wellness.videoTotal')}: {formatDuration(videoTotal)}</span>}>
        {video.loading ? (
          <p className="text-sm text-muted">{t('common.loading')}</p>
        ) : video.error ? (
          <ErrorState error={video.error} />
        ) : (video.data?.platforms.length ?? 0) === 0 ? (
          <Empty />
        ) : (
          <PlatformList items={video.data!.platforms} />
        )}
      </Card>
    </div>
  );
}

function PlatformList({ items }: { items: { platform: string; duration: number; count: number }[] }) {
  const max = Math.max(...items.map((i) => i.duration), 1);
  return (
    <ul className="space-y-2">
      {items.map((it, i) => {
        const pct = (it.duration / max) * 100;
        return (
          <li key={`${it.platform}-${i}`} className="text-sm">
            <div className="flex items-baseline justify-between gap-3 mb-1">
              <span className="truncate flex-1 capitalize" title={it.platform}>{it.platform}</span>
              <span className="font-mono text-xs text-fg/80 whitespace-nowrap">{formatDuration(it.duration)}</span>
            </div>
            <div className="relative h-1.5 rounded-sm bg-elevated overflow-hidden">
              <div
                className="absolute inset-y-0 left-0"
                style={{ width: `${pct}%`, background: '#f43f5e', opacity: 0.85 }}
              />
            </div>
          </li>
        );
      })}
    </ul>
  );
}
