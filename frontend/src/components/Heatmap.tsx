import { useMemo, useState } from 'react';
import { DailyBucket } from '../lib/api';
import { formatDuration } from '../lib/format';

interface Props {
  days: DailyBucket[];
  start: number;
  end: number;
  color?: string;
  weeks?: number;
}

const DAY_MS = 86400 * 1000;
const CELL = 12;
const GAP = 3;

export function Heatmap({ days, end, color = '#22c55e', weeks = 53 }: Props) {
  const [hover, setHover] = useState<{ date: string; duration: number; x: number; y: number } | null>(null);

  const buckets = useMemo(() => {
    const m = new Map<string, number>();
    days.forEach((d) => m.set(d.date, d.duration));
    return m;
  }, [days]);

  const cells = useMemo(() => {
    const endDate = new Date(end * 1000);
    endDate.setHours(0, 0, 0, 0);
    const dayOfWeek = endDate.getDay();
    const trailing = 6 - dayOfWeek;
    const startDate = new Date(endDate.getTime() - (weeks * 7 - 1 - trailing) * DAY_MS);
    startDate.setHours(0, 0, 0, 0);

    const out: { date: string; duration: number; col: number; row: number }[] = [];
    for (let i = 0; i < weeks * 7; i++) {
      const d = new Date(startDate.getTime() + i * DAY_MS);
      const iso = isoDate(d);
      const col = Math.floor(i / 7);
      const row = i % 7;
      out.push({ date: iso, duration: buckets.get(iso) ?? 0, col, row });
    }
    return { cells: out, startDate, endDate };
  }, [buckets, end, weeks]);

  const max = useMemo(() => Math.max(...cells.cells.map((c) => c.duration), 1), [cells]);

  const monthLabels = useMemo(() => {
    const out: { col: number; label: string }[] = [];
    let lastMonth = -1;
    cells.cells.forEach((c) => {
      if (c.row !== 0) return;
      const m = new Date(c.date + 'T00:00:00').getMonth();
      if (m !== lastMonth) {
        out.push({ col: c.col, label: new Date(c.date + 'T00:00:00').toLocaleString(undefined, { month: 'short' }) });
        lastMonth = m;
      }
    });
    return out;
  }, [cells]);

  const width = weeks * (CELL + GAP) + 28;
  const height = 7 * (CELL + GAP) + 26;

  return (
    <div className="relative">
      <svg viewBox={`0 0 ${width} ${height}`} className="w-full" preserveAspectRatio="xMinYMin meet">
        {monthLabels.map((m) => (
          <text key={m.col} x={28 + m.col * (CELL + GAP)} y={10} fontSize={9} fill="#666">
            {m.label}
          </text>
        ))}
        {['Mon', 'Wed', 'Fri'].map((d, i) => (
          <text key={d} x={0} y={26 + (i * 2 + 1) * (CELL + GAP)} fontSize={9} fill="#666">
            {d}
          </text>
        ))}
        {cells.cells.map((c) => {
          const t = c.duration > 0 ? Math.min(1, Math.log(c.duration + 1) / Math.log(max + 1)) : 0;
          const fill = c.duration > 0 ? heatColor(color, t) : '#171717';
          return (
            <rect
              key={c.date}
              x={28 + c.col * (CELL + GAP)}
              y={16 + c.row * (CELL + GAP)}
              width={CELL}
              height={CELL}
              rx={2}
              fill={fill}
              stroke="rgba(255,255,255,0.04)"
              onMouseEnter={(e) =>
                setHover({
                  date: c.date,
                  duration: c.duration,
                  x: e.clientX,
                  y: e.clientY,
                })
              }
              onMouseLeave={() => setHover(null)}
            />
          );
        })}
      </svg>
      <div className="flex items-center gap-2 mt-2 text-xs text-muted">
        <span>Less</span>
        {[0, 0.25, 0.5, 0.75, 1].map((t, i) => (
          <span
            key={i}
            className="inline-block"
            style={{
              width: 10,
              height: 10,
              borderRadius: 2,
              background: t === 0 ? '#171717' : heatColor(color, t),
            }}
          />
        ))}
        <span>More</span>
      </div>
      {hover && (
        <div
          className="fixed pointer-events-none z-50 rounded-md border border-border bg-elevated px-2 py-1 text-xs"
          style={{ left: hover.x + 12, top: hover.y + 12 }}
        >
          <div className="font-mono">{hover.date}</div>
          <div className="text-fg/80">{hover.duration > 0 ? formatDuration(hover.duration) : 'No activity'}</div>
        </div>
      )}
    </div>
  );
}

function isoDate(d: Date) {
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`;
}

function heatColor(base: string, t: number) {
  const [r, g, b] = hexToRgb(base);
  const min = 0.18;
  const a = min + (1 - min) * t;
  return `rgba(${r}, ${g}, ${b}, ${a.toFixed(3)})`;
}

function hexToRgb(hex: string): [number, number, number] {
  const h = hex.replace('#', '');
  const n = h.length === 3 ? h.split('').map((c) => c + c).join('') : h;
  return [parseInt(n.slice(0, 2), 16), parseInt(n.slice(2, 4), 16), parseInt(n.slice(4, 6), 16)];
}
