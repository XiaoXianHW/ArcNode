export const CATEGORY_COLORS: Record<string, string> = {
  coding: '#ffffff',
  gaming: '#ff6b6b',
  video: '#a78bfa',
  music: '#34d399',
  communication: '#fbbf24',
  browsing: '#60a5fa',
  productivity: '#f472b6',
  uncategorized: '#525252',
};

export function categoryColor(name: string): string {
  return CATEGORY_COLORS[name] ?? '#94a3b8';
}

export const CHART_PALETTE = [
  '#ffffff',
  '#a78bfa',
  '#60a5fa',
  '#34d399',
  '#fbbf24',
  '#f472b6',
  '#ff6b6b',
  '#525252',
];
