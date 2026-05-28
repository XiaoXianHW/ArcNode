import { useMemo } from 'react';
import {
  LineChart,
  Line,
  XAxis,
  YAxis,
  ResponsiveContainer,
  Tooltip,
  CartesianGrid,
} from 'recharts';
import { Card, Stat } from '../components/Card';
import { Empty, ErrorState } from '../components/Empty';
import { useDeviceContext } from '../state/deviceContext';
import { useI18n } from '../state/i18nContext';
import { api } from '../lib/api';
import { useAsync } from '../hooks/useAsync';
import { formatTime } from '../lib/format';
import { useChartTokens } from '../lib/chartTokens';

export function System() {
  const { selectedId, startUnix, endUnix, days } = useDeviceContext();
  const { t } = useI18n();
  const tokens = useChartTokens();

  const samples = useAsync(
    () => api.getSystem({ device_id: selectedId, start: startUnix, end: endUnix }),
    [selectedId, startUnix, endUnix],
  );
  const pairs = useAsync(
    () => api.getAppPairs({ device_id: selectedId, days, limit: 20 }),
    [selectedId, days],
  );

  const series = useMemo(() => {
    const list = samples.data?.samples ?? [];
    return list.map((s) => ({
      ts: s.timestamp,
      label: formatTime(s.timestamp),
      cpu: Math.round(s.cpu * 10) / 10,
      memory: Math.round(s.memory * 10) / 10,
    }));
  }, [samples.data]);

  const stats = useMemo(() => {
    const list = samples.data?.samples ?? [];
    if (!list.length) return { avgCpu: 0, avgMem: 0, peakCpu: 0, peakMem: 0 };
    let cpuSum = 0;
    let memSum = 0;
    let peakCpu = 0;
    let peakMem = 0;
    list.forEach((s) => {
      cpuSum += s.cpu;
      memSum += s.memory;
      if (s.cpu > peakCpu) peakCpu = s.cpu;
      if (s.memory > peakMem) peakMem = s.memory;
    });
    return {
      avgCpu: Math.round(cpuSum / list.length),
      avgMem: Math.round(memSum / list.length),
      peakCpu: Math.round(peakCpu),
      peakMem: Math.round(peakMem),
    };
  }, [samples.data]);

  const pairRows = pairs.data?.pairs ?? [];
  const maxPair = Math.max(...pairRows.map((p) => p.count), 1);

  return (
    <div className="space-y-6">
      <header>
        <h1 className="text-2xl font-semibold tracking-tight">{t('system.title')}</h1>
        <p className="text-sm text-muted mt-1">{t('system.subtitle')}</p>
      </header>

      <div className="grid grid-cols-2 lg:grid-cols-4 gap-4">
        <Card>
          <Stat label={t('system.avgCpu')} value={`${stats.avgCpu}%`} />
        </Card>
        <Card>
          <Stat label={t('system.peakCpu')} value={`${stats.peakCpu}%`} />
        </Card>
        <Card>
          <Stat label={t('system.avgMem')} value={`${stats.avgMem}%`} />
        </Card>
        <Card>
          <Stat label={t('system.peakMem')} value={`${stats.peakMem}%`} />
        </Card>
      </div>

      <Card title={t('system.cpu')} subtitle={t('system.cpuSub')}>
        {samples.loading ? (
          <p className="text-sm text-muted">{t('common.loading')}</p>
        ) : samples.error ? (
          <ErrorState error={samples.error} />
        ) : series.length === 0 ? (
          <Empty />
        ) : (
          <div className="h-56">
            <ResponsiveContainer>
              <LineChart data={series} margin={{ left: 4, right: 4, top: 8 }}>
                <CartesianGrid stroke={tokens.border} strokeDasharray="3 3" vertical={false} />
                <XAxis dataKey="label" tick={{ fontSize: 10, fill: tokens.muted }} stroke={tokens.border} interval="preserveStartEnd" />
                <YAxis tick={{ fontSize: 10, fill: tokens.muted }} stroke={tokens.border} domain={[0, 100]} tickFormatter={(v) => `${v}%`} />
                <Tooltip
                  contentStyle={{ background: tokens.tooltipBg, border: `1px solid ${tokens.border}`, fontSize: 12, color: tokens.fg }}
                  formatter={(v: number) => `${v}%`}
                />
                <Line type="monotone" dataKey="cpu" stroke="#f97316" strokeWidth={2} dot={false} />
              </LineChart>
            </ResponsiveContainer>
          </div>
        )}
      </Card>

      <Card title={t('system.memory')} subtitle={t('system.memorySub')}>
        {samples.loading ? (
          <p className="text-sm text-muted">{t('common.loading')}</p>
        ) : samples.error ? (
          <ErrorState error={samples.error} />
        ) : series.length === 0 ? (
          <Empty />
        ) : (
          <div className="h-56">
            <ResponsiveContainer>
              <LineChart data={series} margin={{ left: 4, right: 4, top: 8 }}>
                <CartesianGrid stroke={tokens.border} strokeDasharray="3 3" vertical={false} />
                <XAxis dataKey="label" tick={{ fontSize: 10, fill: tokens.muted }} stroke={tokens.border} interval="preserveStartEnd" />
                <YAxis tick={{ fontSize: 10, fill: tokens.muted }} stroke={tokens.border} domain={[0, 100]} tickFormatter={(v) => `${v}%`} />
                <Tooltip
                  contentStyle={{ background: tokens.tooltipBg, border: `1px solid ${tokens.border}`, fontSize: 12, color: tokens.fg }}
                  formatter={(v: number) => `${v}%`}
                />
                <Line type="monotone" dataKey="memory" stroke="#22d3ee" strokeWidth={2} dot={false} />
              </LineChart>
            </ResponsiveContainer>
          </div>
        )}
      </Card>

      <Card title={t('system.appPairs')} subtitle={t('system.appPairsSub')}>
        {pairs.loading ? (
          <p className="text-sm text-muted">{t('common.loading')}</p>
        ) : pairs.error ? (
          <ErrorState error={pairs.error} />
        ) : pairRows.length === 0 ? (
          <Empty />
        ) : (
          <ul className="space-y-2">
            {pairRows.map((p, i) => {
              const pct = (p.count / maxPair) * 100;
              return (
                <li key={`${p.a}-${p.b}-${i}`} className="text-sm">
                  <div className="flex items-baseline justify-between gap-3 mb-1">
                    <span className="truncate flex-1" title={`${p.a} ↔ ${p.b}`}>
                      {t('system.appPairsCol', { a: p.a, b: p.b })}
                    </span>
                    <span className="font-mono text-xs text-fg/80 whitespace-nowrap">
                      {t('system.appPairsCount', { n: p.count })}
                    </span>
                  </div>
                  <div className="relative h-1.5 rounded-sm bg-elevated overflow-hidden">
                    <div
                      className="absolute inset-y-0 left-0"
                      style={{ width: `${pct}%`, background: '#a78bfa', opacity: 0.85 }}
                    />
                  </div>
                </li>
              );
            })}
          </ul>
        )}
      </Card>
    </div>
  );
}
