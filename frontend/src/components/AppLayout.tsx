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
  Settings as SettingsIcon,
} from 'lucide-react';
import clsx from 'clsx';
import { DeviceBar } from './DeviceBar';

const NAV = [
  { to: '/', label: 'Dashboard', icon: LayoutDashboard, end: true, group: 'Overview' },
  { to: '/timeline', label: 'Timeline', icon: Activity, group: 'Overview' },
  { to: '/categories', label: 'Categories', icon: PieChart, group: 'Overview' },
  { to: '/coding', label: 'Coding', icon: Code2, group: 'Activities' },
  { to: '/gaming', label: 'Gaming', icon: Gamepad2, group: 'Activities' },
  { to: '/shortcuts', label: 'Shortcuts', icon: Keyboard, group: 'Activities' },
  { to: '/devices', label: 'Devices', icon: Cpu, group: 'System' },
  { to: '/settings', label: 'Settings', icon: SettingsIcon, group: 'System' },
];

export function AppLayout({ children }: { children: ReactNode }) {
  return (
    <div className="flex min-h-full">
      <aside className="hidden md:flex w-56 shrink-0 flex-col border-r border-border bg-surface">
        <div className="px-5 py-6">
          <div className="flex items-center gap-2">
            <div className="h-6 w-6 rounded-sm bg-fg" />
            <span className="text-base font-semibold tracking-tight">ArcNode</span>
          </div>
          <p className="mt-1 text-xs text-muted">Personal timeline</p>
        </div>
        <nav className="px-3 pb-6 flex-1 space-y-4">
          {Array.from(new Set(NAV.map((n) => n.group))).map((group) => (
            <div key={group} className="space-y-0.5">
              <p className="px-2 mb-1 text-[10px] uppercase tracking-wider text-muted">{group}</p>
              {NAV.filter((n) => n.group === group).map(({ to, label, icon: Icon, end }) => (
                <NavLink
                  key={to}
                  to={to}
                  end={end}
                  className={({ isActive }) => clsx('nav-link', isActive && 'nav-link-active')}
                >
                  <Icon className="h-4 w-4" />
                  <span>{label}</span>
                </NavLink>
              ))}
            </div>
          ))}
        </nav>
        <div className="px-5 py-4 text-xs text-muted border-t border-border">
          v0.1.0
        </div>
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
