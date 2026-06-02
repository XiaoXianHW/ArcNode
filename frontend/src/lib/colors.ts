export const CATEGORY_COLORS: Record<string, string> = {
  coding: '#22c55e',
  terminal: '#14b8a6',
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

const DEVICE_PALETTE = [
  '#60a5fa',
  '#34d399',
  '#fbbf24',
  '#f472b6',
  '#a78bfa',
  '#ff6b6b',
  '#22d3ee',
  '#facc15',
];

// Stable color for a device id, used when merging multiple devices.
export function deviceColor(id: string): string {
  let hash = 0;
  for (let i = 0; i < id.length; i++) hash = (hash * 31 + id.charCodeAt(i)) >>> 0;
  return DEVICE_PALETTE[hash % DEVICE_PALETTE.length];
}

export const LANGUAGE_COLORS: Record<string, string> = {
  Go: '#00ADD8',
  Rust: '#dea584',
  TypeScript: '#3178c6',
  JavaScript: '#f7df1e',
  Python: '#3572A5',
  Java: '#b07219',
  Kotlin: '#A97BFF',
  Swift: '#F05138',
  'Objective-C': '#438eff',
  C: '#555555',
  'C++': '#f34b7d',
  'C#': '#178600',
  'F#': '#b845fc',
  'VB.NET': '#945db7',
  Ruby: '#701516',
  PHP: '#4F5D95',
  Perl: '#0298c3',
  Lua: '#000080',
  Dart: '#00B4AB',
  R: '#198CE7',
  Julia: '#a270ba',
  Scala: '#c22d40',
  Clojure: '#db5855',
  Elixir: '#6e4a7e',
  Erlang: '#B83998',
  Haskell: '#5e5086',
  Elm: '#60B5CC',
  OCaml: '#3be133',
  Ada: '#02f88c',
  D: '#ba595e',
  Nim: '#ffc200',
  V: '#4f87c4',
  Zig: '#ec915c',
  Crystal: '#000100',
  Groovy: '#4298b8',
  Solidity: '#AA6746',
  Shell: '#89e051',
  PowerShell: '#012456',
  Batch: '#C1F12E',
  SQL: '#e38c00',
  HTML: '#e34c26',
  CSS: '#563d7c',
  Sass: '#a53b70',
  Less: '#1d365d',
  Stylus: '#ff6347',
  Vue: '#41b883',
  Svelte: '#ff3e00',
  Astro: '#ff5d01',
  Markdown: '#083fa1',
  JSON: '#cbcb41',
  YAML: '#cb171e',
  TOML: '#9c4221',
  XML: '#0060ac',
  Docker: '#384d54',
  Terraform: '#623CE4',
  Protobuf: '#3F51B5',
  GraphQL: '#e10098',
  LaTeX: '#3D6117',
  Assembly: '#6E4C13',
  Fortran: '#4d41b1',
  COBOL: '#2350a9',
  MATLAB: '#e16737',
  VHDL: '#adb2cb',
  SystemVerilog: '#DAE1C2',
  Other: '#525252',
};

export function languageColor(name: string): string {
  return LANGUAGE_COLORS[name] ?? '#94a3b8';
}
