import { useMemo, useState } from 'react';
import { BarChart, Bar, Cell, XAxis, YAxis, ResponsiveContainer, Tooltip, CartesianGrid } from 'recharts';
import { Plus, Trash2 } from 'lucide-react';
import { Card } from '../components/Card';
import { Empty, ErrorState } from '../components/Empty';
import { useDeviceContext } from '../state/deviceContext';
import { useI18n } from '../state/i18nContext';
import { api, CustomKeyword } from '../lib/api';
import { useAsync } from '../hooks/useAsync';
import { formatDuration } from '../lib/format';
import { categoryColor } from '../lib/colors';
import { useChartTokens } from '../lib/chartTokens';

const DEFAULT_CATS = [
  'coding',
  'gaming',
  'ai_tools',
  'design',
  'video',
  'music',
  'communication',
  'browsing',
  'productivity',
  'reading',
  'social',
];

export function Categories() {
  const { selectedId, date } = useDeviceContext();
  const { t } = useI18n();
  const tokens = useChartTokens();
  const cats = useAsync(() => api.getCategories({ device_id: selectedId, date }), [selectedId, date]);
  const apps = useAsync(() => api.getApps({ device_id: selectedId, date, limit: 50 }), [selectedId, date]);
  const [filter, setFilter] = useState<string>('');
  const [customRefresh, setCustomRefresh] = useState(0);
  const customKw = useAsync(() => api.listCustomKeywords(), [customRefresh]);

  const total = useMemo(
    () => (cats.data?.categories ?? []).reduce((s, c) => s + c.duration, 0),
    [cats.data],
  );

  if (cats.loading || apps.loading) return <p className="text-sm text-muted">{t('common.loading')}</p>;
  if (cats.error) return <ErrorState error={cats.error} />;
  if (apps.error) return <ErrorState error={apps.error} />;

  const filtered = (apps.data?.apps ?? []).filter((a) => !filter || a.category === filter);
  const categoryOptions = Array.from(
    new Set([...DEFAULT_CATS, ...(cats.data?.categories ?? []).map((c) => c.category)]),
  ).filter((c) => c !== 'uncategorized');

  return (
    <div className="space-y-6">
      <header>
        <h1 className="text-2xl font-semibold tracking-tight">{t('categories.title')}</h1>
        <p className="text-sm text-muted mt-1">{t('categories.subtitle', { date })}</p>
      </header>

      <Card title={t('categories.breakdown')} subtitle={t('categories.breakdownSub', { value: formatDuration(total) })}>
        {(cats.data?.categories ?? []).length === 0 ? (
          <Empty />
        ) : (
          <div className="h-64">
            <ResponsiveContainer>
              <BarChart data={cats.data!.categories} layout="vertical" margin={{ left: 12, right: 12 }}>
                <CartesianGrid stroke={tokens.border} strokeDasharray="3 3" />
                <XAxis
                  type="number"
                  stroke={tokens.muted}
                  tick={{ fontSize: 11, fill: tokens.muted }}
                  tickFormatter={(v) => formatDuration(Number(v))}
                />
                <YAxis dataKey="category" type="category" stroke={tokens.muted} tick={{ fontSize: 11, fill: tokens.muted }} width={100} />
                <Tooltip
                  contentStyle={{ background: tokens.tooltipBg, border: `1px solid ${tokens.border}`, fontSize: 12, color: tokens.fg }}
                  formatter={(v: number) => formatDuration(Number(v))}
                  labelStyle={{ color: tokens.fg }}
                />
                <Bar dataKey="duration" radius={[0, 4, 4, 0]}>
                  {cats.data!.categories.map((c, i) => (
                    <Cell key={i} fill={categoryColor(c.category)} />
                  ))}
                </Bar>
              </BarChart>
            </ResponsiveContainer>
          </div>
        )}
      </Card>

      <CustomKeywordsCard
        keywords={customKw.data?.keywords ?? []}
        loading={customKw.loading}
        error={customKw.error}
        categories={categoryOptions}
        onChanged={() => setCustomRefresh((n) => n + 1)}
      />

      <Card
        title={t('categories.apps')}
        subtitle={t('categories.appsSub', { n: filtered.length })}
        action={
          <select className="input" value={filter} onChange={(e) => setFilter(e.target.value)}>
            <option value="">{t('categories.allCategories')}</option>
            {(cats.data?.categories ?? []).map((c) => (
              <option key={c.category} value={c.category}>
                {c.category}
              </option>
            ))}
          </select>
        }
      >
        {filtered.length === 0 ? (
          <Empty />
        ) : (
          <table className="w-full table">
            <thead>
              <tr>
                <th className="py-2 px-3">{t('common.app')}</th>
                <th className="py-2 px-3">{t('common.category')}</th>
                <th className="py-2 px-3 text-right">{t('common.events')}</th>
                <th className="py-2 px-3 text-right">{t('common.duration')}</th>
              </tr>
            </thead>
            <tbody>
              {filtered.map((a) => (
                <tr key={a.process_name} className="border-t border-border">
                  <td className="py-2 px-3 truncate max-w-[260px]" title={a.process_name}>
                    {a.process_name}
                  </td>
                  <td className="py-2 px-3">
                    <span className="pill" style={{ borderColor: categoryColor(a.category || 'uncategorized') }}>
                      {a.category || 'uncategorized'}
                    </span>
                  </td>
                  <td className="py-2 px-3 text-right font-mono">{a.count}</td>
                  <td className="py-2 px-3 text-right font-mono">{formatDuration(a.duration)}</td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </Card>
    </div>
  );
}

function CustomKeywordsCard({
  keywords,
  loading,
  error,
  categories,
  onChanged,
}: {
  keywords: CustomKeyword[];
  loading: boolean;
  error: unknown;
  categories: string[];
  onChanged: () => void;
}) {
  const { t } = useI18n();
  const [cat, setCat] = useState(categories[0] ?? 'coding');
  const [keyword, setKeyword] = useState('');
  const [submitting, setSubmitting] = useState(false);
  const [err, setErr] = useState<string>('');

  const submit = async () => {
    const value = keyword.trim();
    if (!value) return;
    setSubmitting(true);
    setErr('');
    try {
      await api.addCustomKeyword({ category: cat, keyword: value });
      setKeyword('');
      onChanged();
    } catch (e) {
      setErr(e instanceof Error ? e.message : String(e));
    } finally {
      setSubmitting(false);
    }
  };

  const remove = async (id: number) => {
    try {
      await api.deleteCustomKeyword(id);
      onChanged();
    } catch (e) {
      setErr(e instanceof Error ? e.message : String(e));
    }
  };

  const grouped = keywords.reduce<Record<string, CustomKeyword[]>>((acc, k) => {
    (acc[k.category] = acc[k.category] || []).push(k);
    return acc;
  }, {});

  return (
    <Card title={t('categories.customKeywords')} subtitle={t('categories.customKeywordsSub')}>
      <div className="flex flex-wrap gap-2 items-center">
        <select className="input" value={cat} onChange={(e) => setCat(e.target.value)}>
          {categories.map((c) => (
            <option key={c} value={c}>
              {c}
            </option>
          ))}
        </select>
        <input
          className="input flex-1 min-w-[180px]"
          placeholder={t('categories.addPlaceholder')}
          value={keyword}
          onChange={(e) => setKeyword(e.target.value)}
          onKeyDown={(e) => {
            if (e.key === 'Enter') submit();
          }}
        />
        <button onClick={submit} className="btn" disabled={submitting || !keyword.trim()}>
          <Plus className="h-3.5 w-3.5" />
          {t('common.add')}
        </button>
      </div>
      {err && <p className="mt-2 text-xs text-red-400">{err}</p>}
      {error ? (
        <ErrorState error={error} />
      ) : loading ? (
        <p className="mt-3 text-sm text-muted">{t('common.loading')}</p>
      ) : keywords.length === 0 ? (
        <p className="mt-3 text-sm text-muted">{t('common.empty')}</p>
      ) : (
        <div className="mt-4 space-y-3">
          {Object.entries(grouped).map(([c, items]) => (
            <div key={c}>
              <div className="flex items-center gap-2 mb-2">
                <span className="h-2 w-2 rounded-full" style={{ background: categoryColor(c) }} />
                <h3 className="text-sm font-medium">{c}</h3>
                <span className="text-xs text-muted">{items.length}</span>
              </div>
              <div className="flex flex-wrap gap-1">
                {items.map((k) => (
                  <span key={k.id} className="pill font-mono text-[11px]">
                    {k.keyword}
                    <button
                      onClick={() => remove(k.id)}
                      className="ml-1 text-muted hover:text-fg"
                      title={t('common.delete')}
                    >
                      <Trash2 className="h-3 w-3" />
                    </button>
                  </span>
                ))}
              </div>
            </div>
          ))}
        </div>
      )}
    </Card>
  );
}
