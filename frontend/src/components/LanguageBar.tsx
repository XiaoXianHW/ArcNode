import { LanguageStat } from '../lib/api';
import { formatDuration } from '../lib/format';
import { languageColor } from '../lib/colors';

export function LanguageBar({ items }: { items: LanguageStat[] }) {
  const total = items.reduce((s, i) => s + i.duration, 0) || 1;
  return (
    <div className="space-y-3">
      <div className="flex h-3 w-full overflow-hidden rounded-sm bg-elevated">
        {items.map((it) => (
          <div
            key={it.language}
            title={`${it.language}: ${formatDuration(it.duration)}`}
            className="h-full"
            style={{
              width: `${(it.duration / total) * 100}%`,
              background: languageColor(it.language),
            }}
          />
        ))}
      </div>
      <ul className="grid grid-cols-2 md:grid-cols-3 gap-x-4 gap-y-2 text-sm">
        {items.map((it) => (
          <li key={it.language} className="flex items-center gap-2">
            <span
              className="h-2.5 w-2.5 rounded-sm shrink-0"
              style={{ background: languageColor(it.language) }}
            />
            <span className="flex-1 truncate" title={it.language}>
              {it.language}
            </span>
            <span className="font-mono text-xs text-fg/70">{formatDuration(it.duration)}</span>
            <span className="font-mono text-[10px] text-muted w-9 text-right">
              {((it.duration / total) * 100).toFixed(0)}%
            </span>
          </li>
        ))}
      </ul>
    </div>
  );
}
