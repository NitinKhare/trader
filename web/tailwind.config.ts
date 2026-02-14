import type { Config } from "tailwindcss";

const config: Config = {
  content: [
    "./app/**/*.{js,ts,jsx,tsx,mdx}",
    "./components/**/*.{js,ts,jsx,tsx,mdx}",
  ],
  theme: {
    extend: {
      colors: {
        profit: "#10b981",
        loss: "#ef4444",
        neutral: "#6b7280",
        primary: "#3b82f6",
        dark: "#1f2937",
      },
    },
  },
  plugins: [],
};
export default config;
