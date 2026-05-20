import { useMemo } from 'react';
import { Card } from '../components/Card';
import { Empty, ErrorState } from '../components/Empty';
import { useDeviceContext } from '../state/deviceContext';
import { useI18n } from '../state/i18nContext';
import { api, Segment } from '../lib/api';
import { useAsync } from '../hooks/useAsync';
import { formatDuration, formatTime } from '../lib/format';
import { categoryColor } from '../lib/colors';

export function Timeline() {
  const { selectedId, date } = useDeviceContext();
  const { t } = useI18n();
  const { data, loading, error } = useAsync(
    () => api.getSegments({ device_id: selectedId, date }),
    [selectedId, date],
  );

  if (loading) return <Skeleton />;
  if (error) return <ErrorState error={error} />;

  const segments = data?.segments ?? [];

  return (
    <div className="space-y-6">
      <header>
        <h1 className="text-2xl font-semibold tracking-tight">{t('timeline.title')}</h1>
        <p className="text-sm text-muted mt-1">{t('timeline.subtitle', { date })}</p>
      </header>

      <Card title={t('timeline.hourGrid')} subtitle={t('timeline.hourGridSub')}>
        {segments.length === 0 ? <Empty /> : <HourGrid segments={segments} date={date} />}
      </Card>

      <Card title={t('timeline.segments')} subtitle={t('timeline.segmentsSub', { n: segments.length })}>
        {segments.length === 0 ? (
          <Empty />
        ) : (
          <div className="overflow-auto">
            <table className="w-full table">
              <thead>
                <tr>
                  <th className="py-2 px-3">Time</th>
                  <th className="py-2 px-3">App</th>
                  <th className="py-2 px-3">Title</th>
                  <th className="py-2 px-3">Category</th>
                  <th className="py-2 px-3 text-right">Duration</th>
                </tr>
              </thead>
              <tbody>
                {[...segments].reverse().map((s) => (
                  <tr key={s.id} className="border-t border-border">
                    <td className="py-2 px-3 font-mono text-xs text-muted">
                      {formatTime(s.start_time)} – {formatTime(s.end_time)}
                    </td>
                    <td className="py-2 px-3">{s.process_name}</td>
                    <td className="py-2 px-3 text-fg/70 truncate max-w-[280px]" title={s.window_title}>
                      {s.window_title || '—'}
                    </td>
                    <td className="py-2 px-3">
                      <span className="pill" style={{ borderColor: categoryColor(s.category || 'uncategorized') }}>
                        {s.category || 'uncategorized'}
                      </span>
                    </td>
                    <td className="py-2 px-3 text-right font-mono">{formatDuration(s.duration)}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </Card>
    </div>
  );
}

function HourGrid({ segments, date }: { segments: Segment[]; date: string }) {
  const dayStart = useMemo(() => Math.floor(new Date(date + 'T00:00:00').getTime() / 1000), [date]);
  const hours = Array.from({ length: 24 }, (_, i) => i);

  const bucketsByHour = useMemo(() => {
    const buckets: Record<number, Segment[]> = {};
    segments.forEach((s) => {
      const start = Math.max(s.start_time, dayStart);
      const end = Math.min(s.end_time, dayStart + 24 * 3600);
      if (end <= start) return;
      const startHour = Math.floor((start - dayStart) / 3600);
      const endHour = Math.floor((end - dayStart) / 3600);
      for (let h = startHour; h <= endHour && h < 24; h++) {
        buckets[h] = buckets[h] || [];
        buckets[h].push(s);
      }
    });
    return buckets;
  }, [segments, dayStart]);

  return (
    <div className="space-y-1">
      {hours.map((h) => {
        const segs = bucketsByHour[h] ?? [];
        return (
          <div key={h} className="flex items-center gap-3">
            <span className="font-mono text-xs text-muted w-10">{String(h).padStart(2, '0')}:00</span>
            <div className="relative flex-1 h-6 rounded-sm bg-elevated overflow-hidden">
              {segs.map((s) => {
                const hourStart = dayStart + h * 3600;
                const hourEnd = hourStart + 3600;
                const a = Math.max(s.start_time, hourStart);
                const b = Math.min(s.end_time, hourEnd);
                if (b <= a) return null;
                const left = ((a - hourStart) / 3600) * 100;
                const width = ((b - a) / 3600) * 100;
                return (
                  <div
                    key={`${s.id}-${h}`}
                    title={`${s.process_name} · ${formatDuration(s.duration)}`}
                    className="absolute top-0 bottom-0 opacity-90"
                    style={{
                      left: `${left}%`,
                      width: `${Math.max(width, 0.5)}%`,
                      background: categoryColor(s.category || 'uncategorized'),
                    }}
                  />
                );
              })}
            </div>
          </div>
        );
      })}
    </div>
  );
}

function Skeleton() {
  return (
    <div className="space-y-1">
      {Array.from({ length: 6 }).map((_, i) => (
        <div key={i} className="h-6 bg-elevated rounded animate-pulse" />
      ))}
    </div>
  );
}
