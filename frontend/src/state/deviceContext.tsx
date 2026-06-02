import { createContext, useContext, useEffect, useMemo, useState, ReactNode, useCallback } from 'react';
import { api, Device } from '../lib/api';
import { todayISO, startOfDayUnix, endOfDayUnix, rangeDays } from '../lib/format';

export interface DateRange {
  start: string; // ISO YYYY-MM-DD (inclusive)
  end: string; // ISO YYYY-MM-DD (inclusive)
}

interface DeviceContextValue {
  devices: Device[];
  selectedId: string; // '' means all devices (merged)
  selectDevice: (id: string) => void;
  isMerged: boolean; // true when no specific device is selected
  range: DateRange;
  setRange: (r: DateRange) => void;
  /** unix seconds at local 00:00 of range.start */
  startUnix: number;
  /** unix seconds at local 23:59:59 of range.end */
  endUnix: number;
  /** inclusive number of days in the current range */
  days: number;
  loading: boolean;
  error: string | null;
  refresh: () => void;
}

const DeviceContext = createContext<DeviceContextValue | null>(null);

const SELECTED_KEY = 'arcnode_device';
const RANGE_KEY = 'arcnode_range';

function readInitialRange(): DateRange {
  try {
    const raw = localStorage.getItem(RANGE_KEY);
    if (raw) {
      const parsed = JSON.parse(raw) as DateRange;
      if (parsed.start && parsed.end) return parsed;
    }
  } catch {
    /* ignore */
  }
  const t = todayISO();
  return { start: t, end: t };
}

export function DeviceProvider({ children }: { children: ReactNode }) {
  const [devices, setDevices] = useState<Device[]>([]);
  const [selectedId, setSelectedId] = useState<string>(() => localStorage.getItem(SELECTED_KEY) ?? '');
  const [range, setRangeState] = useState<DateRange>(readInitialRange);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const load = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const res = await api.listDevices();
      setDevices(res.devices);
      // Keep current selection if still valid; otherwise default to merged ('').
      setSelectedId((prev) => {
        if (prev === '') return '';
        if (prev && res.devices.some((d) => d.device_id === prev)) return prev;
        return '';
      });
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

  const selectDevice = useCallback((id: string) => {
    setSelectedId(id);
    if (id) localStorage.setItem(SELECTED_KEY, id);
    else localStorage.setItem(SELECTED_KEY, '');
  }, []);

  const setRange = useCallback((r: DateRange) => {
    // Normalise so start <= end.
    const normalised: DateRange = r.start <= r.end ? r : { start: r.end, end: r.start };
    setRangeState(normalised);
    localStorage.setItem(RANGE_KEY, JSON.stringify(normalised));
  }, []);

  const value = useMemo<DeviceContextValue>(
    () => ({
      devices,
      selectedId,
      selectDevice,
      isMerged: selectedId === '',
      range,
      setRange,
      startUnix: startOfDayUnix(range.start),
      endUnix: endOfDayUnix(range.end),
      days: rangeDays(range.start, range.end),
      loading,
      error,
      refresh: load,
    }),
    [devices, selectedId, selectDevice, range, setRange, loading, error, load],
  );

  return <DeviceContext.Provider value={value}>{children}</DeviceContext.Provider>;
}

export function useDeviceContext(): DeviceContextValue {
  const ctx = useContext(DeviceContext);
  if (!ctx) throw new Error('useDeviceContext must be used within DeviceProvider');
  return ctx;
}
