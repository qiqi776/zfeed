import type { Config } from "tailwindcss";

export default {
  content: ["./index.html", "./src/**/*.{ts,tsx}"],
  theme: {
    extend: {
      colors: {
        ink: "#0B1220",
        mist: "#EFF4FA",
        accent: "#00A6A6",
        ember: "#FF7A59",
      },
      boxShadow: {
        card: "0 20px 40px rgba(11, 18, 32, 0.08)",
      },
    },
  },
  plugins: [],
} satisfies Config;
