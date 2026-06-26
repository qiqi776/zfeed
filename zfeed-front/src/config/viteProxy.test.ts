import { readFileSync } from "node:fs";
import { join } from "node:path";
import { describe, expect, it } from "vitest";

describe("vite dev proxy", () => {
    it("proxies backend API calls during local front-back integration", () => {
        const source = readFileSync(join(process.cwd(), "vite.config.ts"), "utf8");

        expect(source).toContain("server:");
        expect(source).toContain('"/v1"');
        expect(source).toContain('"http://127.0.0.1:5000"');
        expect(source).toContain("changeOrigin: true");
    });
});
