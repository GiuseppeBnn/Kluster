/** @type {import('tailwindcss').Config} */
module.exports = {
  content: [
    './views/**/**/*.ejs',
  ],
  theme: {
    extend: {},
    colors: {
      "primary": "#e10600",
      "secondary": "#1e1e1e",
      "content": "#fff",
      "content-dark": "#1e1e1e",
      "primary-focus": "#9B0400",
      "secondary-focus": "#72728a",
      "base-100": "#EDEDED",
    }
  },
  plugins: [],
}