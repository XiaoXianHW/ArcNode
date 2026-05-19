import { createContext, useContext, useEffect, useMemo, useState, ReactNode, useCallback } from 'react';
import { api, Device } from '../lib/api';
import { todayISO } from '../lib/format';

interface DeviceContextValue {
  devices: Device[];
  selectedId: string;
  selectDevice: (id: string) => void;
  date: string;
  setDate: (d: string) => void;
  loading: boolean;
  error: string | null;
  refresh: () => void;
}

const DeviceContext = createContext<DeviceContextValue | null>(null);

const SELECTED_KEY = 'arcnode_device';

export function DeviceProvider({ children }: { children: ReactNode }) {
  const [devices, setDevices] = useState<Device[]>([]);
  const [selectedId, setSelectedId] = useState<string>(() => localStorage.getItem(SELECTED_KEY) ?? '');
  const [date, setDate] = useState<string>(() => todayISO());
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const load = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const res = await api.listDevices();
      setDevices(res.devices);
      setSelectedId((prev) => {
        if (prev && res.devices.some((d) => d.device_id === prev)) return prev;
        return res.devices[0]?.device_id ?? '';
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
    else localStorage.removeItem(SELECTED_KEY);
  }, []);

  const value = useMemo<DeviceContextValue>(
    () => ({ devices, selectedId, selectDevice, date, setDate, loading, error, refresh: load }),
    [devices, selectedId, selectDevice, date, loading, error, load],
  );

  return <DeviceContext.Provider value={value}>{children}</DeviceContext.Provider>;
}

export function useDeviceContext(): DeviceContextValue {
  const ctx = useContext(DeviceContext);
  if (!ctx) throw new Error('useDeviceContext must be used within DeviceProvider');
  return ctx;
}
