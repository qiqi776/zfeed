import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { clearAuthSession, readAuthSession, saveAuthSession } from "./authStore";

describe("authStore", () => {
    beforeEach(() => {
        window.localStorage.clear();
        vi.restoreAllMocks();
        vi.useFakeTimers();
        vi.setSystemTime(new Date("2026-06-14T12:00:00Z"));
    });

    afterEach(() => {
        window.localStorage.clear();
        vi.useRealTimers();
    });

    it("stores and restores token, expiry, and basic user info", () => {
        saveAuthSession({
            token: "token-1",
            expiredAt: Math.floor(Date.now() / 1000) + 3600,
            user: {
                userId: 7,
                nickname: "Mira Chen",
                avatar: "https://example.com/avatar.png"
            }
        });

        expect(readAuthSession()).toMatchObject({
            token: "token-1",
            user: { userId: 7, nickname: "Mira Chen" }
        });
    });

    it("clears expired, malformed, or incomplete sessions", () => {
        window.localStorage.setItem("zfeed.auth.session", JSON.stringify({
            token: "expired",
            expiredAt: Math.floor(Date.now() / 1000) - 1
        }));
        expect(readAuthSession()).toBeNull();
        expect(window.localStorage.getItem("zfeed.auth.session")).toBeNull();

        window.localStorage.setItem("zfeed.auth.session", "{bad json");
        expect(readAuthSession()).toBeNull();
        expect(window.localStorage.getItem("zfeed.auth.session")).toBeNull();

        window.localStorage.setItem("zfeed.auth.session", JSON.stringify({ token: "missing-expiry" }));
        expect(readAuthSession()).toBeNull();
        expect(window.localStorage.getItem("zfeed.auth.session")).toBeNull();

        window.localStorage.setItem("zfeed.auth.session", JSON.stringify({
            token: "bad-expiry",
            expiredAt: "not-a-number"
        }));
        expect(readAuthSession()).toBeNull();
        expect(window.localStorage.getItem("zfeed.auth.session")).toBeNull();

        window.localStorage.setItem("zfeed.auth.session", JSON.stringify({
            token: 123,
            expiredAt: Math.floor(Date.now() / 1000) + 3600
        }));
        expect(readAuthSession()).toBeNull();
        expect(window.localStorage.getItem("zfeed.auth.session")).toBeNull();
    });

    it("clears sessions with malformed cached user info", () => {
        window.localStorage.setItem("zfeed.auth.session", JSON.stringify({
            token: "token-1",
            expiredAt: Math.floor(Date.now() / 1000) + 3600,
            user: {
                userId: "7",
                nickname: "Mira Chen",
                avatar: "https://example.com/avatar.png"
            }
        }));

        expect(readAuthSession()).toBeNull();
        expect(window.localStorage.getItem("zfeed.auth.session")).toBeNull();
    });

    it("clears saved sessions explicitly", () => {
        saveAuthSession({
            token: "token-1",
            expiredAt: Math.floor(Date.now() / 1000) + 3600
        });

        clearAuthSession();

        expect(readAuthSession()).toBeNull();
    });

    it("rejects invalid sessions before saving them", () => {
        expect(() => saveAuthSession({
            token: "",
            expiredAt: Math.floor(Date.now() / 1000) + 3600
        })).toThrow("Invalid auth session");

        expect(window.localStorage.getItem("zfeed.auth.session")).toBeNull();
    });

    it("rejects invalid user info before saving a session", () => {
        expect(() => saveAuthSession({
            token: "token-1",
            expiredAt: Math.floor(Date.now() / 1000) + 3600,
            user: {
                userId: Number.NaN,
                nickname: "Mira Chen",
                avatar: "https://example.com/avatar.png"
            }
        })).toThrow("Invalid auth session");

        expect(window.localStorage.getItem("zfeed.auth.session")).toBeNull();
    });
});
