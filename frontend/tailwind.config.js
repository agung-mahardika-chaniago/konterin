/** @type {import('tailwindcss').Config} */
export default {
  darkMode: ['class'],
  content: ['./index.html', './src/**/*.{ts,tsx}'],
  theme: {
    extend: {
      colors: {
        brand: {
          50: '#EFF6FF',
          100: '#DBEAFE',
          600: '#1D5AA6',
          700: '#164278',
          800: '#10325F',
          900: '#0B2447',
          950: '#071A33',
        },
        accent: {
          50: '#ECFEFF',
          500: '#06B6D4',
          600: '#0891B2',
        },
        surface: '#FFFFFF',
        background: '#F6F8FB',
        border: '#E2E8F0',
        ink: '#0F172A',
        muted: '#64748B',
        success: '#10B981',
        warning: '#F59E0B',
        danger: '#EF4444',
      },
      fontFamily: {
        sans: ['"Plus Jakarta Sans"', 'system-ui', 'sans-serif'],
        mono: ['"JetBrains Mono"', 'ui-monospace', 'monospace'],
      },
      boxShadow: {
        card: '0 1px 2px rgba(15,23,42,.04), 0 1px 3px rgba(15,23,42,.06)',
        modal: '0 24px 48px rgba(15,23,42,.18)',
      },
      borderRadius: {
        card: '12px',
        control: '8px',
      },
    },
  },
  plugins: [],
};
