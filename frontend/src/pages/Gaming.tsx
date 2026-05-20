import { useMemo, useState } from 'react';
import {
  BarChart,
  Bar,
  XAxis,
  YAxis,
  ResponsiveContainer,
  Tooltip,
  CartesianGrid,
} from 'recharts';
import { CategoryPage } from './Coding';
import { Card, Stat } from '../components/Card';
import { Empty, ErrorState } from '../components/Empty';
import { useDeviceContext } from '../state/deviceContext';
import { useI18n } from '../state/i18nContext';
import { api } from '../lib/api';
import { useAsync } from '../hooks/useAsync';
import { formatDuration, formatDate } from '../lib/format';
import { useChartTokens } from '../lib/chartTokens';

export function Gaming() {
  const { t } = useI18n();
  const { selectedId } = useDeviceContext();
  const tokens = useChartTokens();

  const games = useAsync(() => api.getGames({ device_id: selectedId, days: 365 }), [selectedId]);
  const sessions = useAsync(
    () => api.getSessions({ device_id: selectedId, category: 'gaming', days: 90 }),
    [selectedId],
  );

  const list = games.data?.games ?? [];
  const [selectedGame, setSelectedGame] = useState<string | null>(null);
  const current = useMemo(() => {
    if (!list.length) return null;
    return list.find((g) => g.process_name === selectedGame) ?? list[0];
  }, [list, selectedGame]);

  const sessionBars = useMemo(() => {
    const buckets = sessions.data?.buckets ?? [];
    return buckets.map((b) => ({ ...b, label: b.bucket }));
  }, [sessions.data]);

  return (
    <div className="space-y-6">
      <CategoryPage category="gaming" title={t('gaming.title')} subtitle={t('gaming.subtitle')} />

      <Card title={t('games.title')} subtitle={t('games.subtitle')}>
        {games.loading ? (
          <p className="text-sm text-muted">{t('common.loading')}</p>
        ) : games.error ? (
          <ErrorState error={games.error} />
        ) : list.length === 0 ? (
          <Empty />
        ) : (
          <div className="grid grid-cols-1 lg:grid-cols-[14rem_1fr] gap-4">
            <ul className="space-y-1 max-h-80 overflow-auto pr-1">
              {list.map((g) => (
                <li key={g.process_name}>
                  <button
                    onClick={() => setSelectedGame(g.process_name)}
                    className={`w-full text-left rounded-md px-3 py-2 text-sm hover:bg-elevated transition-colors ${current?.process_name === g.process_name ? 'bg-elevated border border-border' : ''}`}
                  >
                    <p className="truncate font-mono text-xs">{g.process_name}</p>
                    <p className="text-xs text-muted truncate">
                      {formatDuration(g.total_duration)} · {g.sessions} sessions
                    </p>
                  </button>
                </li>
              ))}
            </ul>
            <div>
              {current ? (
                <div className="space-y-4">
                  <div className="grid grid-cols-2 sm:grid-cols-4 gap-3">
                    <Stat label={t('games.totalTime')} value={formatDuration(current.total_duration)} />
                    <Stat label={t('games.sessions')} value={`${current.sessions}`} sub={t('games.uniqueDays', { n: current.unique_days })} />
                    <Stat label={t('games.avgSession')} value={formatDuration(current.avg_session)} />
                    <Stat label={t('games.longestSession')} value={formatDuration(current.max_session)} />
                  </div>
                  <div className="grid grid-cols-1 sm:grid-cols-2 gap-3 text-sm">
                    <div className="rounded-md border border-border bg-elevated p-3">
                      <p className="text-xs uppercase tracking-wider text-muted">{t('games.firstPlayed')}</p>
                      <p className="mt-1 font-mono">{current.first_played ? formatDate(current.first_played) : '—'}</p>
                    </div>
                    <div className="rounded-md border border-border bg-elevated p-3">
                      <p className="text-xs uppercase tracking-wider text-muted">{t('games.lastPlayed')}</p>
                      <p className="mt-1 font-mono">{current.last_played ? formatDate(current.last_played) : '—'}</p>
                    </div>
                  </div>
                </div>
              ) : (
                <Empty />
              )}
            </div>
          </div>
        )}
      </Card>

      <Card title={t('games.sessionHistogram')} subtitle={t('games.sessionHistogramSub')}>
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
                <Bar dataKey="count" fill="#22c55e" radius={[3, 3, 0, 0]} />
              </BarChart>
            </ResponsiveContainer>
          </div>
        )}
      </Card>
    </div>
  );
}
