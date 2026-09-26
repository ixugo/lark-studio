/** @type {import('tailwindcss').Config} */
export default {
  content: [
    "./index.html",
    "./src/**/*.{js,ts,jsx,tsx}",
  ],
  darkMode: 'class',
  theme: {
    extend: {
      colors: {
        apple: {
          bg: '#F5F5F7',
          darkBg: '#121215',
          card: '#FFFFFF',
          darkCard: '#1C1C1E',
          sidebar: '#FBFBFD',
          darkSidebar: '#18181B',
          accent: '#0071E3',
          accentHover: '#0077ED',
          border: '#E5E5EA',
          darkBorder: '#2C2C2E',
          text: '#1D1D1F',
          darkText: '#F5F5F7',
          muted: '#86868B',
        }
      }
    },
  },
  plugins: [],
}
