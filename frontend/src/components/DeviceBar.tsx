import { ChevronDown, Calendar, RefreshCw } from 'lucide-react';
import { useDeviceContext } from '../state/deviceContext';
import { formatRelative } from '../lib/format';

export function DeviceBar() {
  const { devices, selectedId, selectDevice, date, setDate, refresh, loading } = useDeviceContext();
  const selected = devices.find((d) => d.device_id === selectedId);

  return (
    <div className="border-b border-border bg-surface/60 backdrop-blur">
      <div className="mx-auto max-w-6xl flex items-center gap-3 px-6 py-3">
        <div className="relative">
          <select
            value={selectedId}
            onChange={(e) => selectDevice(e.target.value)}
            className="input pr-8 appearance-none"
          >
            {devices.length === 0 && <option value="">No devices</option>}
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
          <span>Refresh</span>
        </button>
        <div className="ml-auto text-xs text-muted">
          {selected ? (
            <>
              <span className="font-mono text-fg/80">{selected.device_id.slice(0, 8)}</span>
              <span className="mx-2">·</span>
              <span>last seen {formatRelative(selected.last_seen)}</span>
            </>
          ) : (
            <span>—</span>
          )}
        </div>
      </div>
    </div>
  );
}
