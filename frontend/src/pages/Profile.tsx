import { useEffect, useState } from 'react';
import { Card } from '../components/Card';
import { Empty, ErrorState } from '../components/Empty';
import { useDeviceContext } from '../state/deviceContext';
import { useI18n } from '../state/i18nContext';
import { api, LiveStatus } from '../lib/api';
import { categoryColor } from '../lib/colors';
import { formatDuration, formatRelative, formatTime } from '../lib/format';

const REFRESH_INTERVAL_MS = 15000;

export function Profile() {
  const { selectedId, devices } = useDeviceContext();
  const { t } = useI18n();
  const [data, setData] = useState<LiveStatus | null>(null);
  const [error, setError] = useState<unknown>(null);
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    if (!selectedId) {
      setData(null);
      return;
    }
    let cancelled = false;
    const load = async () => {
      setLoading(true);
      try {
        const res = await api.getLive(selectedId);
        if (!cancelled) {
          setData(res);
          setError(null);
        }
      } catch (e) {
        if (!cancelled) setError(e);
      } finally {
        if (!cancelled) setLoading(false);
      }
    };
    void load();
    const id = window.setInterval(() => void load(), REFRESH_INTERVAL_MS);
    return () => {
      cancelled = true;
      window.clearInterval(id);
    };
  }, [selectedId]);

  const device = devices.find((d) => d.device_id === selectedId);
  const seg = data?.last_segment;
  const segColor = seg ? categoryColor(seg.category) : '#64748b';

  return (
    <div className="space-y-6">
      <header>
        <h1 className="text-2xl font-semibold tracking-tight">{t('profile.title')}</h1>
        <p className="text-sm text-muted mt-1">{t('profile.subtitle')}</p>
      </header>

      <Card>
        {loading && !data ? (
          <p className="text-sm text-muted">{t('common.loading')}</p>
        ) : error && !data ? (
          <ErrorState error={error} />
        ) : !data ? (
          <Empty message={t('profile.noLive')} />
        ) : (
          <div className="flex flex-col sm:flex-row gap-6">
            <div className="flex flex-col items-center sm:items-start gap-3 min-w-[180px]">
              <div className="relative h-24 w-24 rounded-full border border-border bg-elevated flex items-center justify-center text-3xl font-semibold uppercase">
                <span>{(device?.name || data.device_id).slice(0, 2)}</span>
                <span
                  className="absolute bottom-1 right-1 h-4 w-4 rounded-full border-2"
                  style={{
                    background: data.online ? '#22c55e' : '#64748b',
                    borderColor: 'rgb(var(--c-surface))',
                    boxShadow: data.online ? '0 0 12px #22c55e88' : 'none',
                  }}
                />
              </div>
              <div className="text-center sm:text-left">
                <p className="text-base font-medium">{device?.name || data.device_id}</p>
                <p className="text-xs text-muted mt-0.5">{device?.os_name || '—'}</p>
                <div className="flex flex-wrap gap-1 mt-2">
                  <span
                    className="pill text-[11px]"
                    style={{ background: data.online ? '#22c55e22' : '#64748b22', color: data.online ? '#22c55e' : '#94a3b8' }}
                  >
                    {data.online ? t('profile.online') : t('profile.offline')}
                  </span>
                  <span
                    className="pill text-[11px]"
                    style={{ background: data.idle ? '#f59e0b22' : '#22c55e22', color: data.idle ? '#f59e0b' : '#22c55e' }}
                  >
                    {data.idle ? t('profile.idle') : t('profile.active')}
                  </span>
                </div>
              </div>
            </div>

            <div className="flex-1 min-w-0 space-y-3">
              <div>
                <p className="text-xs uppercase tracking-wider text-muted">{t('profile.currentWindow')}</p>
                <p className="text-sm font-medium mt-1 truncate" title={seg?.window_title || '—'}>
                  {seg?.window_title || '—'}
                </p>
              </div>
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <p className="text-xs uppercase tracking-wider text-muted">{t('profile.currentApp')}</p>
                  <p className="text-sm font-mono mt-1 truncate" title={seg?.process_name || '—'}>
                    {seg?.process_name || '—'}
                  </p>
                </div>
                <div>
                  <p className="text-xs uppercase tracking-wider text-muted">{t('profile.currentCategory')}</p>
                  <p className="text-sm mt-1 flex items-center gap-2">
                    <span className="h-2 w-2 rounded-full" style={{ background: segColor }} />
                    <span className="capitalize">{seg?.category || t('category.uncategorized')}</span>
                  </p>
                </div>
              </div>
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <p className="text-xs uppercase tracking-wider text-muted">{t('common.duration')}</p>
                  <p className="text-sm font-mono mt-1">
                    {seg ? formatDuration(seg.duration) : '—'}
                  </p>
                </div>
                <div>
                  <p className="text-xs uppercase tracking-wider text-muted">{t('common.time')}</p>
                  <p className="text-sm font-mono mt-1">
                    {seg ? `${formatTime(seg.start_time)} → ${formatTime(seg.end_time)}` : '—'}
                  </p>
                </div>
              </div>
              <p className="text-xs text-muted">
                {t('profile.lastSeen', { when: data.last_event_at ? formatRelative(data.last_event_at) : '—' })}
                {data.idle && data.idle_since ? ` · ${t('profile.idleSince', { when: formatRelative(data.idle_since) })}` : ''}
              </p>
            </div>
          </div>
        )}
      </Card>

      <Card title={t('profile.recentApps')} subtitle={t('profile.recentAppsSub')}>
        {!data || (data.recent_apps ?? []).length === 0 ? (
          <Empty />
        ) : (
          <ul className="space-y-2">
            {(data.recent_apps ?? []).slice(0, 10).map((a, i) => {
              const max = Math.max(...(data.recent_apps ?? []).map((x) => x.duration), 1);
              const pct = (a.duration / max) * 100;
              const color = categoryColor(a.category);
              return (
                <li key={`${a.process_name}-${i}`} className="text-sm">
                  <div className="flex items-baseline justify-between gap-3 mb-1">
                    <span className="truncate flex-1 flex items-center gap-2">
                      <span className="h-2 w-2 rounded-full shrink-0" style={{ background: color }} />
                      <span className="truncate font-mono text-xs">{a.process_name}</span>
                    </span>
                    <span className="font-mono text-xs text-fg/80 whitespace-nowrap">
                      {formatDuration(a.duration)}
                    </span>
                  </div>
                  <div className="relative h-1.5 rounded-sm bg-elevated overflow-hidden">
                    <div
                      className="absolute inset-y-0 left-0"
                      style={{ width: `${pct}%`, background: color, opacity: 0.85 }}
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
