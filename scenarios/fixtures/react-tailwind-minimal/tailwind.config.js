/** @type {import('tailwindcss').Config} */
export default {
  content: ["./index.html", "./src/**/*.{js,ts,jsx,tsx}"],
  darkMode: "class",
  theme: {
    extend: {
      colors: {
        background: {
          light: '#ffffff',
          dark: '#000000'
        },
        text: {
          light: '#000000',
          dark: '#ffffff'
        }
      }
    },
  },
  plugins: [],
};