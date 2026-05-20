import { useEffect, useState } from 'react';
import { useTheme } from '../state/themeContext';

export interface ChartTokens {
  surface: string;
  border: string;
  muted: string;
  fg: string;
  cellEmpty: string;
  tooltipBg: string;
}

function readTokens(): ChartTokens {
  if (typeof window === 'undefined') {
    return {
      surface: '#0a0a0a',
      border: '#1f1f1f',
      muted: '#666',
      fg: '#fafafa',
      cellEmpty: '#171717',
      tooltipBg: '#0a0a0a',
    };
  }
  const cs = getComputedStyle(document.documentElement);
  const v = (name: string) => `rgb(${cs.getPropertyValue(name).trim() || '0 0 0'})`;
  return {
    surface: v('--c-surface'),
    border: v('--c-border'),
    muted: v('--c-muted'),
    fg: v('--c-fg'),
    cellEmpty: v('--c-cell-empty'),
    tooltipBg: v('--c-tooltip-bg'),
  };
}

export function useChartTokens(): ChartTokens {
  const { theme } = useTheme();
  const [tokens, setTokens] = useState(readTokens);
  useEffect(() => {
    setTokens(readTokens());
  }, [theme]);
  return tokens;
}
