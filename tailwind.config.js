/** @type {import('tailwindcss').Config} */
module.exports = {
  content: [
    "./templates/**/*.html",
    "./static/**/*.{js,css}",
    "./**/*.go"
  ],
  theme: {
    extend: {
      colors: {
        accent: {
          lavender: '#c4b5fd',
          gold: '#fbbf24',
        }
      },
      animation: {
        'diagonal-drift': 'diagonal-drift 5s linear infinite',
      },
      keyframes: {
        'diagonal-drift': {
          '0%': { 'background-position': '0 0' },
          '100%': { 'background-position': '40px 40px' }
        }
      }
    },
  },
  plugins: [],
  safelist: [
    'animate-diagonal-drift',
    'tech-badge',
    'lavender-accent',
    'lavender-text',
    'gold-accent'
  ]
}