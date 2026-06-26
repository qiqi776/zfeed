import { beforeEach, describe, expect, it } from "vitest";
import { buildAuthSiblingHref, resolveSafeAuthNextPath } from "./authNavigation";

describe("authNavigation", () => {
    beforeEach(() => {
        window.history.pushState({}, "", "/login");
    });

    it("keeps safe modern next paths for auth entry switching and completion", () => {
        window.history.pushState({}, "", "/login?next=%2Fsearch%3Fq%3DAI%23results");

        expect(buildAuthSiblingHref("/register")).toBe("/register?next=%2Fsearch%3Fq%3DAI%23results");
        expect(resolveSafeAuthNextPath("/home")).toBe("/search?q=AI#results");
    });

    it("drops unsafe, removed, and auth-entry next paths", () => {
        for (const next of [
            "https%3A%2F%2Fevil.example%2Fsteal",
            "%2F%2Fevil.example%2Fsteal",
            "%2Ffollowing.html",
            "%2Fprofile",
            "%2Funknown",
            "%2Flogin",
            "%2Fregister"
        ]) {
            window.history.pushState({}, "", `/login?next=${next}`);

            expect(buildAuthSiblingHref("/register")).toBe("/register");
            expect(resolveSafeAuthNextPath("/home")).toBe("/home");
        }
    });
});
