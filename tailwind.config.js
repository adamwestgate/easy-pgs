// tailwind.config.js
module.exports = {
  content: [
    "./src/**/*.{js,jsx,ts,tsx}",
    "./public/index.html",
  ],
theme: {
    extend: {
      keyframes: {
        breathe: {
          '0%, 100%': { transform: 'scale(1)', boxShadow: '0 8px 24px 0 rgba(0,0,0,0.07)' },
          '50%':      { transform: 'scale(1.02)', boxShadow: '0 12px 32px 0 rgba(0,0,0,0.12)' },
        },
      },
      animation: {
        breathe: 'breathe 1s cubic-bezier(0.77,0,0.175,1) infinite',
      },
    },
  },
  plugins: [],
};
