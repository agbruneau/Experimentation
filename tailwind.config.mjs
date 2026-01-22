/** @type {import('tailwindcss').Config} */
export default {
  content: ['./src/**/*.{astro,html,js,jsx,md,mdx,svelte,ts,tsx,vue}'],
  theme: {
    extend: {
      colors: {
        bg: {
          primary: '#0f0f12',
          secondary: '#16161a',
          tertiary: '#1e1e24',
          hover: '#26262e',
        },
        text: {
          primary: '#e8e8ed',
          secondary: '#9898a6',
          tertiary: '#6b6b78',
        },
        accent: {
          verb: '#7c3aed',      // Le Verbe - Applications (violet)
          noun: '#06b6d4',      // Le Nom - Données (cyan)
          signal: '#f59e0b',    // Le Signal - Événements (ambre)
          link: '#818cf8',
        },
        border: {
          subtle: '#2a2a36',
          DEFAULT: '#3a3a48',
        }
      },
      fontFamily: {
        sans: ['Inter', 'system-ui', 'sans-serif'],
        serif: ['Source Serif 4', 'Georgia', 'serif'],
        mono: ['JetBrains Mono', 'Consolas', 'monospace'],
      },
      typography: {
        DEFAULT: {
          css: {
            '--tw-prose-body': '#e8e8ed',
            '--tw-prose-headings': '#ffffff',
            '--tw-prose-lead': '#9898a6',
            '--tw-prose-links': '#818cf8',
            '--tw-prose-bold': '#ffffff',
            '--tw-prose-counters': '#9898a6',
            '--tw-prose-bullets': '#6b6b78',
            '--tw-prose-hr': '#3a3a48',
            '--tw-prose-quotes': '#e8e8ed',
            '--tw-prose-quote-borders': '#7c3aed',
            '--tw-prose-captions': '#9898a6',
            '--tw-prose-code': '#e8e8ed',
            '--tw-prose-pre-code': '#e8e8ed',
            '--tw-prose-pre-bg': '#1a1a22',
            '--tw-prose-th-borders': '#3a3a48',
            '--tw-prose-td-borders': '#2a2a36',
            maxWidth: '72ch',
            lineHeight: '1.75',
          },
        },
      },
    },
  },
  plugins: [
    require('@tailwindcss/typography'),
  ],
};
