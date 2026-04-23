import tokens from './src/styles/design-tokens.json';

function mapThemeToCssVars(theme) {
  const vars = {};
  for (const [key, value] of Object.entries(theme)) {
    vars[`--color-${key}`] = value;
  }
  return vars;
}

export default {
  content: ["./index.html", "./src/**/*.{js,ts,jsx,tsx}"],
  darkMode: 'class',
  theme: {
    extend: {
      colors: {
        background: 'var(--color-background)',
        surface: { DEFAULT: 'var(--color-surface)', alt: 'var(--color-surfaceAlt)' },
        border: 'var(--color-border)',
        primary: { DEFAULT: 'var(--color-primary)', muted: 'var(--color-primaryMuted)' },
        success: 'var(--color-success)',
        warning: 'var(--color-warning)',
        danger: 'var(--color-danger)',
        text: { DEFAULT: 'var(--color-text)', muted: 'var(--color-textMuted)', dim: 'var(--color-textDim)' },
      },
      fontFamily: {
        sans: tokens.typography.fontFamily.main,
        mono: tokens.typography.fontFamily.mono,
      },
      animation: {
        'in': 'in 250ms cubic-bezier(0.16, 1, 0.3, 1)',
        'slide-in-from-right': 'slide-in-from-right 250ms cubic-bezier(0.16, 1, 0.3, 1)',
        'fade-in': 'fade-in 300ms ease-out',
        'zoom-in-95': 'zoom-in-95 250ms cubic-bezier(0.16, 1, 0.3, 1)',
      },
      keyframes: {
        in: {
          '0%': { opacity: 0, transform: 'scale(0.95)' },
          '100%': { opacity: 1, transform: 'scale(1)' },
        },
        'slide-in-from-right': {
          '0%': { transform: 'translateX(100%)' },
          '100%': { transform: 'translateX(0)' },
        },
        'fade-in': {
          '0%': { opacity: 0 },
          '100%': { opacity: 1 },
        },
        'zoom-in-95': {
          '0%': { opacity: 0, transform: 'scale(0.95)' },
          '100%': { opacity: 1, transform: 'scale(1)' },
        },
      },
      transitionTimingFunction: {
        'terminal': 'cubic-bezier(0.16, 1, 0.3, 1)',
      },
      transitionDuration: {
        'micro': '150ms',
        'standard': '250ms',
        'long': '300ms',
      },
    },
  },
  plugins: [],
}