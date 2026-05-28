import { ChevronDown, Calendar, RefreshCw, Layers } from 'lucide-react';
import { useDeviceContext } from '../state/deviceContext';
import { useI18n } from '../state/i18nContext';
import { formatRelative, todayISO, daysAgoISO } from '../lib/format';

export function DeviceBar() {
  const { devices, selectedId, selectDevice, range, setRange, isMerged, refresh, loading } =
    useDeviceContext();
  const { t } = useI18n();
  const selected = devices.find((d) => d.device_id === selectedId);

  const presets: { key: string; start: string; end: string }[] = [
    { key: 'range.today', start: todayISO(), end: todayISO() },
    { key: 'range.7d', start: daysAgoISO(6), end: todayISO() },
    { key: 'range.30d', start: daysAgoISO(29), end: todayISO() },
  ];
  const activePreset = presets.find((p) => p.start === range.start && p.end === range.end)?.key;

  return (
    <div className="sticky top-0 z-20 border-b border-border bg-surface/80 backdrop-blur">
      <div className="mx-auto max-w-6xl flex items-center gap-3 px-6 py-3 flex-wrap">
        <div className="relative">
          <select
            value={selectedId}
            onChange={(e) => selectDevice(e.target.value)}
            className="input pr-8 appearance-none"
          >
            <option value="">{t('devicebar.allDevices')}</option>
            {devices.map((d) => (
              <option key={d.device_id} value={d.device_id}>
                {d.name || d.device_id.slice(0, 8)} · {d.platform}
              </option>
            ))}
          </select>
          <ChevronDown className="absolute right-2 top-1/2 -translate-y-1/2 h-3.5 w-3.5 text-muted pointer-events-none" />
        </div>
        <div className="flex items-center gap-2">
          <Calendar className="h-3.5 w-3.5 text-muted" />
          <input
            type="date"
            value={range.start}
            max={range.end}
            onChange={(e) => setRange({ start: e.target.value, end: range.end })}
            className="input"
            aria-label={t('range.start')}
          />
          <span className="text-muted text-xs">→</span>
          <input
            type="date"
            value={range.end}
            min={range.start}
            max={todayISO()}
            onChange={(e) => setRange({ start: range.start, end: e.target.value })}
            className="input"
            aria-label={t('range.end')}
          />
        </div>
        <div className="flex items-center gap-1">
          {presets.map((p) => (
            <button
              key={p.key}
              onClick={() => setRange({ start: p.start, end: p.end })}
              className={`btn px-2.5 py-1 text-xs ${activePreset === p.key ? 'bg-elevated text-fg' : ''}`}
            >
              {t(p.key)}
            </button>
          ))}
        </div>
        <button onClick={refresh} className="btn" disabled={loading}>
          <RefreshCw className={`h-3.5 w-3.5 ${loading ? 'animate-spin' : ''}`} />
          <span>{t('common.refresh')}</span>
        </button>
        <div className="ml-auto flex items-center gap-3 text-xs text-muted">
          {isMerged ? (
            <span className="inline-flex items-center gap-1.5 text-fg/80">
              <Layers className="h-3.5 w-3.5" />
              {t('devicebar.mergedCount', { n: devices.length })}
            </span>
          ) : selected ? (
            <>
              <span className="font-mono text-fg/80">{selected.device_id.slice(0, 8)}</span>
              <span className="hidden sm:inline">·</span>
              <span className="hidden sm:inline">
                {t('devicebar.lastSeen')} {formatRelative(selected.last_seen)}
              </span>
            </>
          ) : (
            <span>—</span>
          )}
        </div>
      </div>
    </div>
  );
}
