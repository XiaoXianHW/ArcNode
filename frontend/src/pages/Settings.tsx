import { useState } from 'react';
import { Card, Stat } from '../components/Card';
import { Empty, ErrorState } from '../components/Empty';
import { useI18n } from '../state/i18nContext';
import { useTheme } from '../state/themeContext';
import {
  api,
  exportEventsJSONURL,
  exportSegmentsCSVURL,
  getToken,
  setToken,
  CategoryStat,
  AppStat,
  UncategorizedRow,
  WeeklyReport,
} from '../lib/api';
import { useAsync } from '../hooks/useAsync';
import { categoryColor } from '../lib/colors';
import { useDeviceContext } from '../state/deviceContext';
import { formatDuration } from '../lib/format';

export function Settings() {
  const { t, locale, setLocale } = useI18n();
  const { theme, setTheme } = useTheme();
  const { selectedId } = useDeviceContext();
  const [token, setTok] = useState<string>(getToken());
  const [saved, setSaved] = useState(false);
  const rules = useAsync(() => api.getRules(), [saved]);
  const suggestions = useAsync(
    () => api.getSuggestions({ device_id: selectedId, days: 30, limit: 12 }),
    [selectedId],
  );
  const [weekly, setWeekly] = useState<WeeklyReport | null>(null);
  const [weeklyLoading, setWeeklyLoading] = useState(false);
  const [weeklyError, setWeeklyError] = useState<unknown>(null);
  const [refreshTick, setRefreshTick] = useState(0);

  const ruleKeys = Object.keys(rules.data?.rules ?? {});
  const [addingFor, setAddingFor] = useState<string | null>(null);

  const save = () => {
    setToken(token.trim());
    setSaved(true);
    setTimeout(() => setSaved(false), 1500);
  };

  const runWeekly = async () => {
    setWeeklyLoading(true);
    setWeeklyError(null);
    try {
      const r = await api.getWeeklyReport({ device_id: selectedId, days: 7 });
      setWeekly(r);
    } catch (e) {
      setWeeklyError(e);
    } finally {
      setWeeklyLoading(false);
    }
  };

  const addSuggestion = async (item: UncategorizedRow, category: string) => {
    try {
      await api.addCustomKeyword({ category, keyword: item.process_name });
      setAddingFor(null);
      setRefreshTick((x) => x + 1);
    } catch (e) {
      setWeeklyError(e);
    }
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

      <Card title={t('settings.weekly')} subtitle={t('settings.weeklySub')} action={
        <button className="btn" onClick={runWeekly} disabled={weeklyLoading}>
          {weeklyLoading ? t('common.loading') : t('settings.weeklyGenerate')}
        </button>
      }>
        {weeklyError ? (
          <ErrorState error={weeklyError} />
        ) : weekly ? (
          <WeeklySummary report={weekly} t={t} />
        ) : (
          <p className="text-sm text-muted">{t('settings.weeklySub')}</p>
        )}
      </Card>

      <Card title={t('settings.suggestions')} subtitle={t('settings.suggestionsSub')}>
        {suggestions.loading ? (
          <p className="text-sm text-muted">{t('common.loading')}</p>
        ) : suggestions.error ? (
          <ErrorState error={suggestions.error} />
        ) : (suggestions.data?.items.length ?? 0) === 0 ? (
          <Empty />
        ) : (
          <ul className="space-y-3" key={refreshTick}>
            {suggestions.data!.items.map((item) => (
              <li key={item.process_name} className="flex items-center justify-between gap-3 text-sm rounded-md border border-border bg-elevated p-3">
                <div className="min-w-0 flex-1">
                  <p className="font-mono text-xs truncate">{item.process_name}</p>
                  {item.sample_title && (
                    <p className="text-xs text-muted truncate">{item.sample_title}</p>
                  )}
                  <p className="text-xs text-muted mt-0.5">
                    {formatDuration(item.duration)} · {item.count} {t('common.events')}
                  </p>
                </div>
                <div className="shrink-0">
                  {addingFor === item.process_name ? (
                    <div className="flex flex-wrap gap-1 max-w-xs justify-end">
                      {ruleKeys.map((cat) => (
                        <button
                          key={cat}
                          onClick={() => void addSuggestion(item, cat)}
                          className="pill text-[11px] hover:bg-fg/10 capitalize"
                          style={{ borderColor: categoryColor(cat) }}
                        >
                          {cat}
                        </button>
                      ))}
                      <button
                        onClick={() => setAddingFor(null)}
                        className="pill text-[11px]"
                      >
                        {t('common.cancel')}
                      </button>
                    </div>
                  ) : (
                    <button className="btn" onClick={() => setAddingFor(item.process_name)}>
                      {t('common.add')}
                    </button>
                  )}
                </div>
              </li>
            ))}
          </ul>
        )}
      </Card>

      <Card title={t('settings.export')} subtitle={t('settings.exportSub')}>
        <div className="flex flex-wrap gap-2">
          <a
            className="btn"
            href={exportSegmentsCSVURL({ device_id: selectedId })}
            download
          >
            {t('settings.exportCSV')}
          </a>
          <a
            className="btn"
            href={exportEventsJSONURL({ device_id: selectedId, limit: 5000 })}
            download
          >
            {t('settings.exportJSON')}
          </a>
        </div>
      </Card>

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
            {Object.entries(rules.data.rules).map(([cat, rule]) => {
              const total = rule.process.length + rule.title.length;
              return (
                <div key={cat}>
                  <div className="flex items-center gap-2 mb-2">
                    <span className="h-2 w-2 rounded-full" style={{ background: categoryColor(cat) }} />
                    <h3 className="text-sm font-medium">{cat}</h3>
                    <span className="text-xs text-muted">{total}</span>
                  </div>
                  {rule.process.length > 0 && (
                    <RuleRow label={t('categories.scope.process')} items={rule.process} />
                  )}
                  {rule.title.length > 0 && (
                    <RuleRow label={t('categories.scope.title')} items={rule.title} />
                  )}
                </div>
              );
            })}
          </div>
        )}
      </Card>
    </div>
  );
}

function WeeklySummary({ report, t }: { report: WeeklyReport; t: (k: string, p?: Record<string, string | number>) => string }) {
  return (
    <div className="space-y-4">
      <div className="grid grid-cols-2 sm:grid-cols-4 gap-3">
        <Stat label={t('settings.weeklyTotal')} value={formatDuration(report.total_active)} />
        <Stat label={t('settings.weeklyFocus')} value={formatDuration(report.total_focus)} sub={t('settings.weeklyLongestFocus') + ': ' + formatDuration(report.longest_focus)} />
        <Stat label={t('settings.weeklyAvgFlow')} value={`${report.avg_flow_score}`} sub="0–100" />
        <Stat label={t('settings.weeklySwitches')} value={`${report.context_switches}`} />
      </div>
      <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
        <RankColumn title={t('settings.weeklyTopCats')} items={report.top_categories.map((c: CategoryStat) => ({ label: c.category, value: c.duration, color: categoryColor(c.category) }))} />
        <RankColumn title={t('settings.weeklyTopApps')} items={report.top_apps.slice(0, 6).map((a: AppStat) => ({ label: a.process_name, value: a.duration, color: categoryColor(a.category) }))} />
        {report.top_languages.length > 0 && (
          <RankColumn title={t('settings.weeklyTopLangs')} items={report.top_languages.slice(0, 6).map((l) => ({ label: l.language, value: l.duration, color: '#60a5fa' }))} />
        )}
        {report.top_games.length > 0 && (
          <RankColumn title={t('settings.weeklyTopGames')} items={report.top_games.slice(0, 6).map((g: AppStat) => ({ label: g.process_name, value: g.duration, color: categoryColor('gaming') }))} />
        )}
      </div>
    </div>
  );
}

function RankColumn({ title, items }: { title: string; items: { label: string; value: number; color: string }[] }) {
  if (items.length === 0) return null;
  const max = Math.max(...items.map((i) => i.value), 1);
  return (
    <div>
      <p className="text-xs uppercase tracking-wider text-muted mb-2">{title}</p>
      <ul className="space-y-2">
        {items.map((it, i) => {
          const pct = (it.value / max) * 100;
          return (
            <li key={`${it.label}-${i}`} className="text-sm">
              <div className="flex items-baseline justify-between gap-3 mb-1">
                <span className="truncate flex-1 capitalize" title={it.label}>{it.label}</span>
                <span className="font-mono text-xs text-fg/80 whitespace-nowrap">{formatDuration(it.value)}</span>
              </div>
              <div className="relative h-1.5 rounded-sm bg-elevated overflow-hidden">
                <div
                  className="absolute inset-y-0 left-0"
                  style={{ width: `${pct}%`, background: it.color, opacity: 0.85 }}
                />
              </div>
            </li>
          );
        })}
      </ul>
    </div>
  );
}

function RuleRow({ label, items }: { label: string; items: string[] }) {
  return (
    <div className="mb-2 last:mb-0">
      <p className="text-[10px] uppercase tracking-wider text-muted mb-1">{label}</p>
      <div className="flex flex-wrap gap-1">
        {items.map((k) => (
          <span key={k} className="pill font-mono text-[11px]">
            {k}
          </span>
        ))}
      </div>
    </div>
  );
}
