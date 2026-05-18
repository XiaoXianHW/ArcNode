import { ReactNode } from 'react';
import { NavLink } from 'react-router-dom';
import { LayoutDashboard, Activity, PieChart, Keyboard, Cpu, Settings as SettingsIcon } from 'lucide-react';
import clsx from 'clsx';
import { DeviceBar } from './DeviceBar';

const NAV = [
  { to: '/', label: 'Dashboard', icon: LayoutDashboard, end: true },
  { to: '/timeline', label: 'Timeline', icon: Activity },
  { to: '/categories', label: 'Categories', icon: PieChart },
  { to: '/shortcuts', label: 'Shortcuts', icon: Keyboard },
  { to: '/devices', label: 'Devices', icon: Cpu },
  { to: '/settings', label: 'Settings', icon: SettingsIcon },
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
        <nav className="px-3 pb-6 flex-1 space-y-0.5">
          {NAV.map(({ to, label, icon: Icon, end }) => (
            <NavLink
              key={to}
              to={to}
              end={end}
              className={({ isActive }) =>
                clsx('nav-link', isActive && 'nav-link-active')
              }
            >
              <Icon className="h-4 w-4" />
              <span>{label}</span>
            </NavLink>
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
