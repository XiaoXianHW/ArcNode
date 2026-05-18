const TOKEN_KEY = 'arcnode_token';

export function getToken(): string {
  return localStorage.getItem(TOKEN_KEY) ?? '';
}

export function setToken(t: string) {
  if (t) localStorage.setItem(TOKEN_KEY, t);
  else localStorage.removeItem(TOKEN_KEY);
}

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const token = getToken();
  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
    ...((init?.headers as Record<string, string>) ?? {}),
  };
  if (token) headers['Authorization'] = `Bearer ${token}`;
  const res = await fetch(`/api/v1${path}`, { ...init, headers });
  if (!res.ok) {
    const text = await res.text();
    throw new Error(text || `${res.status} ${res.statusText}`);
  }
  return (await res.json()) as T;
}

export interface Device {
  device_id: string;
  name: string;
  platform: string;
  cpu_brand: string;
  cpu_cores: number;
  total_memory: number;
  total_disk: number;
  os_name: string;
  os_version: string;
  architecture: string;
  last_seen: number;
  created_at: number;
}

export interface CategoryStat {
  category: string;
  duration: number;
  count: number;
}

export interface AppStat {
  process_name: string;
  category: string;
  duration: number;
  count: number;
}

export interface ShortcutStat {
  shortcut: string;
  application: string;
  count: number;
}

export interface Segment {
  id: number;
  device_id: string;
  process_name: string;
  window_title: string;
  category: string;
  start_time: number;
  end_time: number;
  duration: number;
}

export interface EventItem {
  id: number;
  device_id: string;
  timestamp: number;
  event_type: string;
  category: string;
  process_name: string;
  window_title: string;
  pid: number;
  metadata?: Record<string, unknown>;
}

export interface Summary {
  start: number;
  end: number;
  categories: CategoryStat[];
  top_apps: AppStat[];
  shortcuts: ShortcutStat[];
  idle: { idle_seconds: number; active_seconds: number };
}

function qs(params: Record<string, string | number | undefined>): string {
  const search = new URLSearchParams();
  Object.entries(params).forEach(([k, v]) => {
    if (v !== undefined && v !== '' && v !== null) search.set(k, String(v));
  });
  const s = search.toString();
  return s ? `?${s}` : '';
}

export const api = {
  listDevices: () => request<{ devices: Device[] }>('/devices'),
  getSummary: (params: { device_id?: string; date?: string }) =>
    request<Summary>(`/stats/summary${qs(params)}`),
  getCategories: (params: { device_id?: string; date?: string }) =>
    request<{ categories: CategoryStat[]; start: number; end: number }>(`/stats/categories${qs(params)}`),
  getApps: (params: { device_id?: string; date?: string; limit?: number }) =>
    request<{ apps: AppStat[]; start: number; end: number }>(`/stats/apps${qs(params)}`),
  getShortcuts: (params: { device_id?: string; date?: string; limit?: number }) =>
    request<{ shortcuts: ShortcutStat[]; start: number; end: number }>(`/stats/shortcuts${qs(params)}`),
  getSegments: (params: { device_id?: string; date?: string; category?: string }) =>
    request<{ segments: Segment[] }>(`/segments${qs(params)}`),
  getEvents: (params: {
    device_id?: string;
    start?: number;
    end?: number;
    type?: string;
    category?: string;
    limit?: number;
  }) => request<{ events: EventItem[] }>(`/events${qs(params)}`),
  getRules: () => request<{ rules: Record<string, string[]> }>('/categories'),
};
