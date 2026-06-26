import { defineConfig } from "vitest/config";
import react from "@vitejs/plugin-react";

export default defineConfig({
    plugins: [react()],
    server: {
        proxy: {
            "/v1/admin": {
                target: "http://127.0.0.1:5011",
                changeOrigin: true
            },
            "/v1": {
                target: "http://127.0.0.1:5000",
                changeOrigin: true
            }
        }
    },
    test: {
        environment: "jsdom",
        include: ["src/**/*.{test,spec}.{ts,tsx}"],
        setupFiles: "./src/test/setup.ts",
        globals: true
    }
});
