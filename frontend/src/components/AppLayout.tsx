import { ReactNode } from 'react';
import { NavLink } from 'react-router-dom';
import {
  LayoutDashboard,
  Activity,
  PieChart,
  Keyboard,
  Cpu,
  Code2,
  Gamepad2,
  Sparkles,
  Settings as SettingsIcon,
  Target,
  Heart,
  Server,
  User,
  HelpCircle,
  Sun,
  Moon,
  Languages,
} from 'lucide-react';
import clsx from 'clsx';
import { DeviceBar } from './DeviceBar';
import { useI18n } from '../state/i18nContext';
import { useTheme } from '../state/themeContext';
import { useDeviceContext } from '../state/deviceContext';
import { formatRelative } from '../lib/format';

type NavGroupKey =
  | 'nav.group.overview'
  | 'nav.group.activities'
  | 'nav.group.wellness'
  | 'nav.group.system';

interface NavItem {
  to: string;
  key: string;
  icon: typeof LayoutDashboard;
  end?: boolean;
  group: NavGroupKey;
}

const NAV: NavItem[] = [
  { to: '/', key: 'nav.dashboard', icon: LayoutDashboard, end: true, group: 'nav.group.overview' },
  { to: '/profile', key: 'nav.profile', icon: User, group: 'nav.group.overview' },
  { to: '/timeline', key: 'nav.timeline', icon: Activity, group: 'nav.group.overview' },
  { to: '/categories', key: 'nav.categories', icon: PieChart, group: 'nav.group.overview' },
  { to: '/focus', key: 'nav.focus', icon: Target, group: 'nav.group.activities' },
  { to: '/coding', key: 'nav.coding', icon: Code2, group: 'nav.group.activities' },
  { to: '/gaming', key: 'nav.gaming', icon: Gamepad2, group: 'nav.group.activities' },
  { to: '/insights', key: 'nav.insights', icon: Sparkles, group: 'nav.group.activities' },
  { to: '/shortcuts', key: 'nav.shortcuts', icon: Keyboard, group: 'nav.group.activities' },
  { to: '/wellness', key: 'nav.wellness', icon: Heart, group: 'nav.group.wellness' },
  { to: '/system', key: 'nav.system', icon: Server, group: 'nav.group.system' },
  { to: '/devices', key: 'nav.devices', icon: Cpu, group: 'nav.group.system' },
  { to: '/settings', key: 'nav.settings', icon: SettingsIcon, group: 'nav.group.system' },
];

const GROUPS: NavGroupKey[] = [
  'nav.group.overview',
  'nav.group.activities',
  'nav.group.wellness',
  'nav.group.system',
];

export function AppLayout({ children }: { children: ReactNode }) {
  const { t } = useI18n();
  const { theme, toggle } = useTheme();
  const { locale, setLocale } = useI18n();
  const { devices, selectedId } = useDeviceContext();
  const selected = devices.find((d) => d.device_id === selectedId);

  return (
    <div className="flex h-screen overflow-hidden">
      <aside className="hidden md:flex w-60 shrink-0 flex-col border-r border-border bg-surface">
        <div className="px-5 pt-5 pb-4 border-b border-border">
          <p className="text-sm font-semibold tracking-tight leading-tight">{t('app.name')}</p>
          <p className="text-[11px] text-muted truncate">{t('app.tagline')}</p>
        </div>

        {selected ? (
          <div className="mx-3 mt-3 rounded-lg border border-border bg-elevated px-3 py-2.5">
            <div className="flex items-center gap-2.5">
              <div className="relative">
                <div className="h-8 w-8 rounded-md bg-bg flex items-center justify-center text-[10px] font-mono text-fg/80 uppercase">
                  {(selected.name || selected.device_id).slice(0, 2)}
                </div>
                <span className="absolute -bottom-0.5 -right-0.5 h-2 w-2 rounded-full bg-emerald-500 ring-2 ring-elevated" />
              </div>
              <div className="min-w-0 flex-1">
                <p className="text-xs font-medium text-fg truncate">
                  {selected.name || selected.device_id.slice(0, 8)}
                </p>
                <p className="text-[10px] text-muted truncate">
                  {formatRelative(selected.last_seen)}
                </p>
              </div>
            </div>
          </div>
        ) : null}

        <nav className="flex-1 overflow-y-auto px-2 pt-4 pb-4 space-y-4">
          {GROUPS.map((group) => {
            const items = NAV.filter((n) => n.group === group);
            if (!items.length) return null;
            return (
              <div key={group} className="space-y-0.5">
                <p className="px-3 mb-1 text-[10px] uppercase tracking-[0.08em] text-muted/80">
                  {t(group)}
                </p>
                {items.map(({ to, key, icon: Icon, end }) => (
                  <NavLink
                    key={to}
                    to={to}
                    end={end}
                    className={({ isActive }) =>
                      clsx('nav-link relative', isActive && 'nav-link-active')
                    }
                  >
                    {({ isActive }) => (
                      <>
                        {isActive ? (
                          <span className="absolute left-0 top-1.5 bottom-1.5 w-0.5 rounded-r bg-fg" />
                        ) : null}
                        <Icon className="h-4 w-4" />
                        <span>{t(key)}</span>
                      </>
                    )}
                  </NavLink>
                ))}
              </div>
            );
          })}
        </nav>

        <div className="border-t border-border px-3 py-2.5 flex items-center justify-between">
          <div className="flex items-center gap-1">
            <button
              onClick={toggle}
              className="btn-ghost px-2"
              title={t('settings.theme')}
              aria-label={t('settings.theme')}
            >
              {theme === 'dark' ? <Sun className="h-3.5 w-3.5" /> : <Moon className="h-3.5 w-3.5" />}
            </button>
            <button
              onClick={() => setLocale(locale === 'zh' ? 'en' : 'zh')}
              className="btn-ghost px-2"
              title={t('settings.language')}
              aria-label={t('settings.language')}
            >
              <Languages className="h-3.5 w-3.5" />
            </button>
            <a
              className="btn-ghost px-2"
              href="https://github.com/XiaoXianHW/ArcNode"
              target="_blank"
              rel="noreferrer"
              title="GitHub"
              aria-label="GitHub"
            >
              <HelpCircle className="h-3.5 w-3.5" />
            </a>
          </div>
          <span className="text-[10px] text-muted font-mono">v0.3.0</span>
        </div>
      </aside>

      <main className="flex-1 min-w-0 flex flex-col h-screen">
        <DeviceBar />
        <div className="flex-1 overflow-y-auto">
          <div className="mx-auto max-w-6xl p-6">{children}</div>
        </div>
      </main>
    </div>
  );
}


