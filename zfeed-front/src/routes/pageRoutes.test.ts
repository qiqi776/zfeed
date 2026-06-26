import { describe, expect, it } from "vitest";
import { resolvePageRoute } from "./pageRoutes";

describe("resolvePageRoute", () => {
    it("maps modern URLs to the intended source page", () => {
        expect(resolvePageRoute("/")).toBe("auth-gateway");
        expect(resolvePageRoute("/home")).toBe("home");
        expect(resolvePageRoute("/me")).toBe("profile");
        expect(resolvePageRoute("/user/jax")).toBe("profile");
        expect(resolvePageRoute("/following")).toBe("following");
        expect(resolvePageRoute("/content/1001")).toBe("detail");
        expect(resolvePageRoute("/content/1001/edit")).toBe("edit-content");
        expect(resolvePageRoute("/me/edit")).toBe("edit-profile");
        expect(resolvePageRoute("/search")).toBe("search");
        expect(resolvePageRoute("/compose")).toBe("compose");
        expect(resolvePageRoute("/settings")).toBe("settings");
        expect(resolvePageRoute("/login")).toBe("login");
        expect(resolvePageRoute("/register")).toBe("register");
    });

    it("does not map old .html URLs", () => {
        expect(resolvePageRoute("/pasted-html-original-copy.html")).toBe("not-found");
        expect(resolvePageRoute("/following.html")).toBe("not-found");
        expect(resolvePageRoute("/profile.html")).toBe("not-found");
        expect(resolvePageRoute("/detail.html")).toBe("not-found");
        expect(resolvePageRoute("/edit-profile.html")).toBe("not-found");
        expect(resolvePageRoute("/liquid-glass-feed.html")).toBe("not-found");
        expect(resolvePageRoute("/liquid-glass-feed")).toBe("not-found");
        expect(resolvePageRoute("/profile")).toBe("not-found");
        expect(resolvePageRoute("/detail")).toBe("not-found");
        expect(resolvePageRoute("/content/article-1")).toBe("not-found");
        expect(resolvePageRoute("/content/video-1")).toBe("not-found");
        expect(resolvePageRoute("/edit-profile")).toBe("not-found");
        expect(resolvePageRoute("/unknown")).toBe("not-found");
    });
});
