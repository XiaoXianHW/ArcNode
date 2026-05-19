import { useState } from 'react';
import { Card } from '../components/Card';
import { Empty, ErrorState } from '../components/Empty';
import { api, getToken, setToken } from '../lib/api';
import { useAsync } from '../hooks/useAsync';
import { categoryColor } from '../lib/colors';

export function Settings() {
  const [token, setTok] = useState<string>(getToken());
  const [saved, setSaved] = useState(false);
  const rules = useAsync(() => api.getRules(), [saved]);

  const save = () => {
    setToken(token.trim());
    setSaved(true);
    setTimeout(() => setSaved(false), 1500);
  };

  return (
    <div className="space-y-6">
      <header>
        <h1 className="text-2xl font-semibold tracking-tight">Settings</h1>
        <p className="text-sm text-muted mt-1">Configure your gateway connection</p>
      </header>

      <Card title="API token" subtitle="Bearer token attached to all gateway requests">
        <div className="flex gap-2 items-center">
          <input
            type="password"
            placeholder="paste token..."
            value={token}
            onChange={(e) => setTok(e.target.value)}
            className="input flex-1"
          />
          <button onClick={save} className="btn">
            Save
          </button>
          {saved && <span className="text-xs text-muted">Saved</span>}
        </div>
        <p className="text-xs text-muted mt-3">
          Must match the <code className="font-mono">token</code> value in the gateway config file.
        </p>
      </Card>

      <Card title="Category rules" subtitle="Configured in gateway config.toml">
        {rules.loading ? (
          <p className="text-sm text-muted">Loading…</p>
        ) : rules.error ? (
          <ErrorState error={rules.error} />
        ) : !rules.data || Object.keys(rules.data.rules).length === 0 ? (
          <Empty />
        ) : (
          <div className="space-y-4">
            {Object.entries(rules.data.rules).map(([cat, keywords]) => (
              <div key={cat}>
                <div className="flex items-center gap-2 mb-2">
                  <span className="h-2 w-2 rounded-full" style={{ background: categoryColor(cat) }} />
                  <h3 className="text-sm font-medium">{cat}</h3>
                  <span className="text-xs text-muted">{keywords.length} keywords</span>
                </div>
                <div className="flex flex-wrap gap-1">
                  {keywords.map((k) => (
                    <span key={k} className="pill font-mono text-[11px]">
                      {k}
                    </span>
                  ))}
                </div>
              </div>
            ))}
          </div>
        )}
      </Card>
    </div>
  );
}
