import { Card } from '../components/Card';
import { Empty, ErrorState } from '../components/Empty';
import { useDeviceContext } from '../state/deviceContext';
import { useI18n } from '../state/i18nContext';
import { formatBytes, formatRelative, formatDate } from '../lib/format';

export function Devices() {
  const { devices, loading, error, selectedId, selectDevice } = useDeviceContext();
  const { t } = useI18n();

  if (loading) return <p className="text-sm text-muted">{t('common.loading')}</p>;
  if (error) return <ErrorState error={error} />;

  return (
    <div className="space-y-6">
      <header>
        <h1 className="text-2xl font-semibold tracking-tight">{t('devices.title')}</h1>
        <p className="text-sm text-muted mt-1">{t('devices.subtitle')}</p>
      </header>

      {devices.length === 0 ? (
        <Card>
          <Empty message="No devices have reported yet. Start the agent with a remote storage config pointing here." />
        </Card>
      ) : (
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          {devices.map((d) => (
            <Card key={d.device_id}>
              <header className="flex items-start justify-between">
                <div>
                  <h2 className="text-base font-semibold tracking-tight">{d.name || d.device_id.slice(0, 8)}</h2>
                  <p className="font-mono text-xs text-muted mt-0.5">{d.device_id}</p>
                </div>
                <button
                  className={`btn ${selectedId === d.device_id ? 'bg-fg text-bg' : ''}`}
                  onClick={() => selectDevice(d.device_id)}
                >
                  {selectedId === d.device_id ? 'Selected' : 'Select'}
                </button>
              </header>
              <dl className="mt-4 grid grid-cols-2 gap-x-4 gap-y-2 text-sm">
                <Row label="Platform" value={`${d.platform || '—'} ${d.architecture || ''}`.trim()} />
                <Row label="OS" value={`${d.os_name || '—'} ${d.os_version || ''}`.trim()} />
                <Row label="CPU" value={d.cpu_brand ? `${d.cpu_brand} · ${d.cpu_cores}c` : '—'} />
                <Row label="Memory" value={formatBytes(d.total_memory)} />
                <Row label="Disk" value={formatBytes(d.total_disk)} />
                <Row label="Last seen" value={formatRelative(d.last_seen)} />
                <Row label="Registered" value={formatDate(d.created_at)} />
              </dl>
            </Card>
          ))}
        </div>
      )}
    </div>
  );
}

function Row({ label, value }: { label: string; value: string }) {
  return (
    <>
      <dt className="text-muted text-xs">{label}</dt>
      <dd className="text-fg/90 truncate" title={value}>{value}</dd>
    </>
  );
}
