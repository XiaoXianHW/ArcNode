import { useMemo } from 'react';
import { Card } from '../components/Card';
import { Empty, ErrorState } from '../components/Empty';
import { useDeviceContext } from '../state/deviceContext';
import { useI18n } from '../state/i18nContext';
import { api, Segment } from '../lib/api';
import { useAsync } from '../hooks/useAsync';
import { formatDuration, formatTime, formatDate, toISODate } from '../lib/format';
import { categoryColor, deviceColor } from '../lib/colors';

export function Timeline() {
  const { selectedId, isMerged, devices, range, startUnix, endUnix } = useDeviceContext();
  const { t } = useI18n();
  const { data, loading, error } = useAsync(
    () => api.getSegments({ device_id: selectedId, start: startUnix, end: endUnix }),
    [selectedId, startUnix, endUnix],
  );

  const deviceName = useMemo(() => {
    const m: Record<string, string> = {};
    devices.forEach((d) => (m[d.device_id] = d.name || d.device_id.slice(0, 8)));
    return (id: string) => m[id] ?? id.slice(0, 8);
  }, [devices]);

  const segments = data?.segments ?? [];
  const rangeLabel = range.start === range.end ? range.start : `${range.start} → ${range.end}`;
  const multiDay = range.start !== range.end;

  // Group by local day for the multi-day view (hook must run unconditionally).
  const days = useMemoDays(segments, range.start, range.end);

  if (loading) return <Skeleton />;
  if (error) return <ErrorState error={error} />;

  return (
    <div className="space-y-6">
      <header>
        <h1 className="text-2xl font-semibold tracking-tight">{t('timeline.title')}</h1>
        <p className="text-sm text-muted mt-1">{t('timeline.subtitle', { date: rangeLabel })}</p>
      </header>

      {isMerged && segments.length > 0 && (
        <Card title={t('timeline.devices')} subtitle={t('timeline.devicesSub')}>
          <div className="flex flex-wrap gap-3">
            {Array.from(new Set(segments.map((s) => s.device_id))).map((id) => (
              <span key={id} className="inline-flex items-center gap-2 text-sm">
                <span className="h-2.5 w-2.5 rounded-full" style={{ background: deviceColor(id) }} />
                {deviceName(id)}
              </span>
            ))}
          </div>
        </Card>
      )}

      <Card title={t('timeline.hourGrid')} subtitle={t('timeline.hourGridSub')}>
        {segments.length === 0 ? (
          <Empty />
        ) : multiDay ? (
          <div className="space-y-4">
            {days.map((d) => (
              <div key={d.iso}>
                <p className="text-xs text-muted mb-1 font-mono">{d.iso}</p>
                <HourGrid segments={d.segments} date={d.iso} colorBy={isMerged ? 'device' : 'category'} />
              </div>
            ))}
          </div>
        ) : (
          <HourGrid segments={segments} date={range.start} colorBy={isMerged ? 'device' : 'category'} />
        )}
      </Card>

      <Card title={t('timeline.segments')} subtitle={t('timeline.segmentsSub', { n: segments.length })}>
        {segments.length === 0 ? (
          <Empty />
        ) : (
          <div className="overflow-auto">
            <table className="w-full table">
              <thead>
                <tr>
                  <th className="py-2 px-3">{t('common.time')}</th>
                  {multiDay && <th className="py-2 px-3">{t('common.date')}</th>}
                  {isMerged && <th className="py-2 px-3">{t('nav.devices')}</th>}
                  <th className="py-2 px-3">{t('common.app')}</th>
                  <th className="py-2 px-3">{t('common.title')}</th>
                  <th className="py-2 px-3">{t('common.category')}</th>
                  <th className="py-2 px-3 text-right">{t('common.duration')}</th>
                </tr>
              </thead>
              <tbody>
                {[...segments].reverse().map((s) => (
                  <tr key={s.id} className="border-t border-border">
                    <td className="py-2 px-3 font-mono text-xs text-muted">
                      {formatTime(s.start_time)} – {formatTime(s.end_time)}
                    </td>
                    {multiDay && (
                      <td className="py-2 px-3 font-mono text-xs text-muted">{formatDate(s.start_time)}</td>
                    )}
                    {isMerged && (
                      <td className="py-2 px-3">
                        <span className="inline-flex items-center gap-1.5">
                          <span
                            className="h-2 w-2 rounded-full"
                            style={{ background: deviceColor(s.device_id) }}
                          />
                          <span className="text-xs text-fg/80">{deviceName(s.device_id)}</span>
                        </span>
                      </td>
                    )}
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

function useMemoDays(segments: Segment[], start: string, end: string) {
  return useMemo(() => {
    const out: { iso: string; segments: Segment[] }[] = [];
    const cur = new Date(`${start}T00:00:00`);
    const last = new Date(`${end}T00:00:00`);
    while (cur <= last) {
      const iso = toISODate(cur);
      const dayStart = Math.floor(new Date(`${iso}T00:00:00`).getTime() / 1000);
      const dayEnd = dayStart + 24 * 3600;
      out.push({
        iso,
        segments: segments.filter((s) => s.end_time > dayStart && s.start_time < dayEnd),
      });
      cur.setDate(cur.getDate() + 1);
    }
    return out.filter((d) => d.segments.length > 0);
  }, [segments, start, end]);
}

function HourGrid({
  segments,
  date,
  colorBy,
}: {
  segments: Segment[];
  date: string;
  colorBy: 'category' | 'device';
}) {
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

  const segColor = (s: Segment) =>
    colorBy === 'device' ? deviceColor(s.device_id) : categoryColor(s.category || 'uncategorized');

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
                      background: segColor(s),
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
