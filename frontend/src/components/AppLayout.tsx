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
} from 'lucide-react';
import clsx from 'clsx';
import { DeviceBar } from './DeviceBar';
import { useI18n } from '../state/i18nContext';

const NAV = [
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

export function AppLayout({ children }: { children: ReactNode }) {
  const { t } = useI18n();
  return (
    <div className="flex min-h-full">
      <aside className="hidden md:flex w-56 shrink-0 flex-col border-r border-border bg-surface">
        <div className="px-5 pt-6 pb-3">
          <span className="text-base font-semibold tracking-tight">{t('app.name')}</span>
          <p className="mt-1 text-xs text-muted">{t('app.tagline')}</p>
        </div>
        <nav className="px-3 pb-6 flex-1 space-y-4">
          {Array.from(new Set(NAV.map((n) => n.group))).map((group) => (
            <div key={group} className="space-y-0.5">
              <p className="px-2 mb-1 text-[10px] uppercase tracking-wider text-muted">{t(group)}</p>
              {NAV.filter((n) => n.group === group).map(({ to, key, icon: Icon, end }) => (
                <NavLink
                  key={to}
                  to={to}
                  end={end}
                  className={({ isActive }) => clsx('nav-link', isActive && 'nav-link-active')}
                >
                  <Icon className="h-4 w-4" />
                  <span>{t(key)}</span>
                </NavLink>
              ))}
            </div>
          ))}
        </nav>
        <div className="px-5 py-4 text-xs text-muted border-t border-border">v0.2.0</div>
      </aside>
      <main className="flex-1 min-w-0 flex flex-col">
        <DeviceBar />
        <div className="flex-1 overflow-auto">
          <div className="mx-auto max-w-6xl p-6">{children}</div>
        </div>
      </main>
    </div>
  );
}
