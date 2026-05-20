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

export interface DailyBucket {
  date: string;
  duration: number;
  count: number;
}

export interface ProjectStat {
  window_title: string;
  process_name: string;
  duration: number;
  count: number;
}

export interface Heatmap {
  days: DailyBucket[];
  start: number;
  end: number;
  max_duration: number;
  total_duration: number;
  active_days: number;
  current_streak: number;
  longest_streak: number;
}

export interface LanguageStat {
  language: string;
  duration: number;
  count: number;
}

export interface HourlyBucket {
  weekday: number;
  hour: number;
  duration: number;
  count: number;
}

export interface BalanceRow {
  date: string;
  category: string;
  duration: number;
}

export interface CustomKeyword {
  id: number;
  category: string;
  keyword: string;
  created_at: number;
}

export interface FocusBlock {
  category: string;
  start_time: number;
  end_time: number;
  duration: number;
  apps: number;
}

export interface FocusResponse {
  blocks: FocusBlock[];
  start: number;
  end: number;
  total_focus: number;
  longest: number;
  daily: { date: string; duration: number }[];
  min_duration: number;
  max_gap_seconds: number;
}

export interface SwitchBucket {
  date: string;
  switches: number;
  unique_apps: number;
}

export interface HourlySwitch {
  weekday: number;
  hour: number;
  switches: number;
}

export interface FlowDay {
  date: string;
  active_seconds: number;
  idle_seconds: number;
  focus_seconds: number;
  switches: number;
  unique_apps: number;
  score: number;
}

export interface SessionBucket {
  bucket: string;
  min: number;
  max: number;
  count: number;
  duration: number;
}

export interface FileStat {
  file: string;
  language: string;
  duration: number;
  count: number;
}

export interface ProjectDailyRow {
  date: string;
  project: string;
  duration: number;
}

export interface AppPair {
  a: string;
  b: string;
  count: number;
}

export interface VideoRow {
  platform: string;
  duration: number;
  count: number;
}

export interface SedentaryDay {
  date: string;
  longest_stretch: number;
  stretches_over_threshold: number;
  total_active: number;
  total_idle: number;
}

export interface UncategorizedRow {
  process_name: string;
  sample_title: string;
  duration: number;
  count: number;
}

export interface SystemSample {
  timestamp: number;
  cpu: number;
  memory: number;
  battery_pct?: number;
}

export interface LiveStatus {
  device_id: string;
  online: boolean;
  idle: boolean;
  last_event_at: number;
  last_segment?: Segment;
  recent_apps?: AppStat[];
  idle_since?: number;
}

export interface IdleRatioDay {
  date: string;
  active: number;
  idle: number;
}

export interface GameReport {
  process_name: string;
  title: string;
  total_duration: number;
  sessions: number;
  avg_session: number;
  max_session: number;
  first_played: number;
  last_played: number;
  unique_days: number;
}

export interface WeeklyReport {
  start: number;
  end: number;
  total_active: number;
  total_focus: number;
  top_categories: CategoryStat[];
  top_apps: AppStat[];
  top_languages: LanguageStat[];
  top_games: AppStat[];
  avg_flow_score: number;
  best_day: string;
  best_day_duration: number;
  longest_focus: number;
  busiest_hour: number;
  busiest_weekday: number;
  context_switches: number;
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
  getDaily: (params: { device_id?: string; category?: string; days?: number; end?: number }) =>
    request<{ days: DailyBucket[]; start: number; end: number }>(`/stats/daily${qs(params)}`),
  getHeatmap: (params: { device_id?: string; category?: string; days?: number; end?: number }) =>
    request<Heatmap>(`/stats/heatmap${qs(params)}`),
  getProjects: (params: { device_id?: string; category?: string; date?: string; limit?: number }) =>
    request<{ projects: ProjectStat[] }>(`/stats/projects${qs(params)}`),
  getLanguages: (params: { device_id?: string; days?: number; end?: number }) =>
    request<{ languages: LanguageStat[]; start: number; end: number }>(`/stats/languages${qs(params)}`),
  getHourly: (params: { device_id?: string; category?: string; days?: number; end?: number }) =>
    request<{ buckets: HourlyBucket[]; start: number; end: number; max_duration: number }>(`/stats/hourly${qs(params)}`),
  getBalance: (params: { device_id?: string; days?: number; end?: number }) =>
    request<{ rows: BalanceRow[]; start: number; end: number }>(`/stats/balance${qs(params)}`),
  listCustomKeywords: () => request<{ keywords: CustomKeyword[] }>('/custom-keywords'),
  addCustomKeyword: (body: { category: string; keyword: string }) =>
    request<CustomKeyword>('/custom-keywords', { method: 'POST', body: JSON.stringify(body) }),
  deleteCustomKeyword: (id: number) =>
    request<{ ok: true }>(`/custom-keywords/${id}`, { method: 'DELETE' }),

  getFocus: (params: { device_id?: string; category?: string; days?: number; min_duration?: number; max_gap?: number }) =>
    request<FocusResponse>(`/stats/focus${qs(params)}`),
  getSwitches: (params: { device_id?: string; days?: number }) =>
    request<{ daily: SwitchBucket[]; hourly: HourlySwitch[]; start: number; end: number }>(`/stats/switches${qs(params)}`),
  getFlow: (params: { device_id?: string; days?: number }) =>
    request<{ days: FlowDay[]; start: number; end: number }>(`/stats/flow${qs(params)}`),
  getSessions: (params: { device_id?: string; category?: string; days?: number }) =>
    request<{ buckets: SessionBucket[]; start: number; end: number }>(`/stats/sessions${qs(params)}`),
  getFiles: (params: { device_id?: string; days?: number; limit?: number }) =>
    request<{ files: FileStat[]; start: number; end: number }>(`/stats/files${qs(params)}`),
  getProjectsDaily: (params: { device_id?: string; category?: string; days?: number }) =>
    request<{ rows: ProjectDailyRow[]; start: number; end: number }>(`/stats/projects-daily${qs(params)}`),
  getAppPairs: (params: { device_id?: string; days?: number; limit?: number }) =>
    request<{ pairs: AppPair[]; start: number; end: number }>(`/stats/app-pairs${qs(params)}`),
  getVideoStats: (params: { device_id?: string; days?: number }) =>
    request<{ platforms: VideoRow[]; start: number; end: number }>(`/stats/video${qs(params)}`),
  getIdleRatio: (params: { device_id?: string; days?: number }) =>
    request<{ days: IdleRatioDay[]; start: number; end: number }>(`/stats/idle-ratio${qs(params)}`),
  getSedentary: (params: { device_id?: string; days?: number; threshold?: number }) =>
    request<{ days: SedentaryDay[]; threshold: number; start: number; end: number }>(`/stats/sedentary${qs(params)}`),
  getSuggestions: (params: { device_id?: string; days?: number; limit?: number }) =>
    request<{ items: UncategorizedRow[]; start: number; end: number }>(`/stats/suggestions${qs(params)}`),
  getSystem: (params: { device_id?: string; days?: number }) =>
    request<{ samples: SystemSample[]; start: number; end: number }>(`/stats/system${qs(params)}`),
  getGames: (params: { device_id?: string; days?: number }) =>
    request<{ games: GameReport[]; start: number; end: number }>(`/stats/games${qs(params)}`),
  getLive: (device_id: string) => request<LiveStatus>(`/devices/${encodeURIComponent(device_id)}/live`),
  getWeeklyReport: (params: { device_id?: string; days?: number }) =>
    request<WeeklyReport>(`/stats/weekly-report${qs(params)}`),
};

export function exportSegmentsCSVURL(params: { device_id?: string; date?: string; category?: string; start?: number; end?: number }): string {
  const search = new URLSearchParams();
  Object.entries(params).forEach(([k, v]) => {
    if (v !== undefined && v !== '' && v !== null) search.set(k, String(v));
  });
  const s = search.toString();
  return `/api/v1/export/segments.csv${s ? `?${s}` : ''}`;
}

export function exportEventsJSONURL(params: { device_id?: string; start?: number; end?: number; type?: string; limit?: number }): string {
  const search = new URLSearchParams();
  Object.entries(params).forEach(([k, v]) => {
    if (v !== undefined && v !== '' && v !== null) search.set(k, String(v));
  });
  const s = search.toString();
  return `/api/v1/export/events.json${s ? `?${s}` : ''}`;
}
