import type { Config } from 'tailwindcss';

const config: Config = {
  content: ['./index.html', './src/**/*.{ts,tsx}'],
  theme: {
    extend: {
      fontFamily: {
        sans: ['Geist', 'Inter', 'ui-sans-serif', 'system-ui', 'sans-serif'],
        mono: ['Geist Mono', 'JetBrains Mono', 'ui-monospace', 'monospace'],
      },
      colors: {
        bg: '#000000',
        surface: '#0a0a0a',
        elevated: '#111111',
        border: '#1f1f1f',
        muted: '#666666',
        fg: '#fafafa',
        accent: '#ffffff',
      },
      boxShadow: {
        soft: '0 1px 0 0 rgba(255,255,255,0.04), 0 0 0 1px rgba(255,255,255,0.06)',
      },
    },
  },
  plugins: [],
};

export default config;
