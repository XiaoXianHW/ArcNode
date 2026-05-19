import { Card } from '../components/Card';
import { Empty, ErrorState } from '../components/Empty';
import { useDeviceContext } from '../state/deviceContext';
import { api } from '../lib/api';
import { useAsync } from '../hooks/useAsync';

export function Shortcuts() {
  const { selectedId, date } = useDeviceContext();
  const { data, loading, error } = useAsync(
    () => api.getShortcuts({ device_id: selectedId, date, limit: 100 }),
    [selectedId, date],
  );

  if (loading) return <p className="text-sm text-muted">Loading…</p>;
  if (error) return <ErrorState error={error} />;
  const shortcuts = data?.shortcuts ?? [];

  return (
    <div className="space-y-6">
      <header>
        <h1 className="text-2xl font-semibold tracking-tight">Shortcuts</h1>
        <p className="text-sm text-muted mt-1">Most-used keyboard shortcuts for {date}</p>
      </header>

      <Card title="Top shortcuts" subtitle={`${shortcuts.length} unique combos`}>
        {shortcuts.length === 0 ? (
          <Empty />
        ) : (
          <table className="w-full table">
            <thead>
              <tr>
                <th className="py-2 px-3">Shortcut</th>
                <th className="py-2 px-3">Application</th>
                <th className="py-2 px-3 text-right">Count</th>
              </tr>
            </thead>
            <tbody>
              {shortcuts.map((sc, i) => (
                <tr key={`${sc.shortcut}-${i}`} className="border-t border-border">
                  <td className="py-2 px-3 font-mono">{sc.shortcut}</td>
                  <td className="py-2 px-3 text-fg/70">{sc.application || '—'}</td>
                  <td className="py-2 px-3 text-right font-mono">{sc.count}</td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </Card>
    </div>
  );
}
