import { ReactNode } from 'react';
import clsx from 'clsx';

export function Card({
  title,
  subtitle,
  action,
  children,
  className,
}: {
  title?: string;
  subtitle?: string;
  action?: ReactNode;
  children: ReactNode;
  className?: string;
}) {
  return (
    <section className={clsx('card', className)}>
      {(title || action) && (
        <header className="flex items-start justify-between mb-4">
          <div>
            {title && <h2 className="card-title">{title}</h2>}
            {subtitle && <p className="card-subtitle mt-0.5">{subtitle}</p>}
          </div>
          {action}
        </header>
      )}
      {children}
    </section>
  );
}

export function Stat({ label, value, sub }: { label: string; value: ReactNode; sub?: string }) {
  return (
    <div>
      <p className="card-subtitle">{label}</p>
      <p className="stat-value mt-2">{value}</p>
      {sub && <p className="card-subtitle mt-1">{sub}</p>}
    </div>
  );
}
