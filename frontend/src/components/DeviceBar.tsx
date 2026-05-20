import { ChevronDown, Calendar, RefreshCw, Sun, Moon, Languages } from 'lucide-react';
import { useDeviceContext } from '../state/deviceContext';
import { useTheme } from '../state/themeContext';
import { useI18n } from '../state/i18nContext';
import { formatRelative } from '../lib/format';

export function DeviceBar() {
  const { devices, selectedId, selectDevice, date, setDate, refresh, loading } = useDeviceContext();
  const { theme, toggle } = useTheme();
  const { locale, setLocale, t } = useI18n();
  const selected = devices.find((d) => d.device_id === selectedId);

  return (
    <div className="border-b border-border bg-surface/60 backdrop-blur">
      <div className="mx-auto max-w-6xl flex items-center gap-3 px-6 py-3 flex-wrap">
        <div className="relative">
          <select
            value={selectedId}
            onChange={(e) => selectDevice(e.target.value)}
            className="input pr-8 appearance-none"
          >
            {devices.length === 0 && <option value="">{t('devicebar.noDevices')}</option>}
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
            value={date}
            onChange={(e) => setDate(e.target.value)}
            className="input"
          />
        </div>
        <button onClick={refresh} className="btn" disabled={loading}>
          <RefreshCw className={`h-3.5 w-3.5 ${loading ? 'animate-spin' : ''}`} />
          <span>{t('common.refresh')}</span>
        </button>
        <div className="ml-auto flex items-center gap-3 text-xs text-muted">
          {selected ? (
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
          <button
            onClick={() => setLocale(locale === 'zh' ? 'en' : 'zh')}
            className="btn-ghost"
            title={t('settings.language')}
          >
            <Languages className="h-3.5 w-3.5" />
            <span>{locale === 'zh' ? t('common.lang.zh') : t('common.lang.en')}</span>
          </button>
          <button
            onClick={toggle}
            className="btn-ghost"
            title={t('settings.theme')}
          >
            {theme === 'dark' ? (
              <Sun className="h-3.5 w-3.5" />
            ) : (
              <Moon className="h-3.5 w-3.5" />
            )}
          </button>
        </div>
      </div>
    </div>
  );
}
