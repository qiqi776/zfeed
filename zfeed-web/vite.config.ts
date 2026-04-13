import path from "node:path";
import react from "@vitejs/plugin-react-swc";
import { loadEnv } from "vite";
import { defineConfig } from "vitest/config";

export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, process.cwd(), "");
  const proxyTarget =
    env.VITE_DEV_PROXY_TARGET?.trim() ||
    env.VITE_API_BASE_URL?.trim() ||
    "http://127.0.0.1:5000";

  return {
    plugins: [react()],
    resolve: {
      alias: {
        "@": path.resolve(__dirname, "./src"),
      },
    },
    server: {
      port: 5173,
      proxy: {
        "/v1": {
          target: proxyTarget,
          changeOrigin: true,
        },
      },
    },
    test: {
      globals: true,
      environment: "jsdom",
      setupFiles: "./src/test/setup.ts",
    },
  };
});
