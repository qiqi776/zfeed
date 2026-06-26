import { defineConfig, devices } from "@playwright/test";

export default defineConfig({
    testDir: "./e2e",
    timeout: 90_000,
    expect: {
        timeout: 5_000
    },
    use: {
        baseURL: "http://127.0.0.1:55002",
        channel: "chrome",
        trace: "on-first-retry"
    },
    webServer: {
        command: "npm run dev -- --port 55002",
        url: "http://127.0.0.1:55002",
        reuseExistingServer: false,
        timeout: 120_000
    },
    projects: [
        {
            name: "chromium-375",
            use: { ...devices["Pixel 5"], viewport: { width: 375, height: 812 } }
        },
        {
            name: "chromium-768",
            use: { ...devices["Desktop Chrome"], viewport: { width: 768, height: 1024 } }
        },
        {
            name: "chromium-1440",
            use: { ...devices["Desktop Chrome"], viewport: { width: 1440, height: 1100 } }
        }
    ]
});
