import type { Config } from 'tailwindcss';

const config: Config = {
  content: ['./src/**/*.{ts,tsx}'],
  theme: {
    extend: {
      colors: {
        brand: {
          50: '#fff8f1', 100: '#f6eadc', 200: '#ead5bd',
          500: '#c87941', 600: '#b5652f', 700: '#8f4e27', 800: '#6f3d22', 900: '#3f2a1f',
        },
      },
      fontFamily: {
        mono: ['"JetBrains Mono"', '"Fira Code"', 'Consolas', 'monospace'],
      },
    },
  },
  plugins: [],
};

export default config;
