import { useState } from 'react';
import { Card } from '../components/Card';
import { Empty, ErrorState } from '../components/Empty';
import { useI18n } from '../state/i18nContext';
import { useTheme } from '../state/themeContext';
import { api, getToken, setToken } from '../lib/api';
import { useAsync } from '../hooks/useAsync';
import { categoryColor } from '../lib/colors';

export function Settings() {
  const { t, locale, setLocale } = useI18n();
  const { theme, setTheme } = useTheme();
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
        <h1 className="text-2xl font-semibold tracking-tight">{t('settings.title')}</h1>
        <p className="text-sm text-muted mt-1">{t('settings.subtitle')}</p>
      </header>

      <Card title={t('settings.tokenTitle')} subtitle={t('settings.tokenSub')}>
        <div className="flex gap-2 items-center">
          <input
            type="password"
            placeholder={t('settings.tokenPlaceholder')}
            value={token}
            onChange={(e) => setTok(e.target.value)}
            className="input flex-1"
          />
          <button onClick={save} className="btn">{t('common.save')}</button>
          {saved && <span className="text-xs text-muted">{t('common.saved')}</span>}
        </div>
        <p className="text-xs text-muted mt-3">{t('settings.tokenHelp')}</p>
      </Card>

      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
        <Card title={t('settings.theme')} subtitle={t('settings.themeSub')}>
          <div className="flex gap-2">
            <button
              onClick={() => setTheme('dark')}
              className={`btn ${theme === 'dark' ? 'border-fg/50' : ''}`}
            >
              {t('common.theme.dark')}
            </button>
            <button
              onClick={() => setTheme('light')}
              className={`btn ${theme === 'light' ? 'border-fg/50' : ''}`}
            >
              {t('common.theme.light')}
            </button>
          </div>
        </Card>
        <Card title={t('settings.language')} subtitle={t('settings.languageSub')}>
          <div className="flex gap-2">
            <button
              onClick={() => setLocale('en')}
              className={`btn ${locale === 'en' ? 'border-fg/50' : ''}`}
            >
              English
            </button>
            <button
              onClick={() => setLocale('zh')}
              className={`btn ${locale === 'zh' ? 'border-fg/50' : ''}`}
            >
              中文
            </button>
          </div>
        </Card>
      </div>

      <Card title={t('settings.mcp')} subtitle={t('settings.mcpSub')}>
        <pre className="text-xs text-muted font-mono bg-elevated rounded-md border border-border p-3 overflow-auto">
{`POST /mcp
Authorization: Bearer <token>
Content-Type: application/json

{"jsonrpc":"2.0","id":1,"method":"tools/list"}`}
        </pre>
      </Card>

      <Card title={t('settings.rulesTitle')} subtitle={t('settings.rulesSub')}>
        {rules.loading ? (
          <p className="text-sm text-muted">{t('common.loading')}</p>
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
                  <span className="text-xs text-muted">{keywords.length}</span>
                </div>
                <div className="flex flex-wrap gap-1">
                  {keywords.map((k) => (
                    <span key={k} className="pill font-mono text-[11px]">{k}</span>
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
