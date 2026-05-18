export const CATEGORY_COLORS: Record<string, string> = {
  coding: '#22c55e',
  ai_tools: '#a78bfa',
  design: '#f472b6',
  gaming: '#ef4444',
  video: '#8b5cf6',
  music: '#10b981',
  communication: '#f59e0b',
  browsing: '#3b82f6',
  productivity: '#ec4899',
  reading: '#06b6d4',
  social: '#fb7185',
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
