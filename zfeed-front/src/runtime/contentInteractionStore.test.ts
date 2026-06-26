import { beforeEach, describe, expect, it, vi } from "vitest";

const session = {
    token: "interaction-token",
    expiredAt: Math.floor(Date.now() / 1000) + 3600,
    user: { userId: 7 }
};

describe("contentInteractionStore", () => {
    beforeEach(() => {
        window.localStorage.clear();
        vi.resetModules();
    });

    it("restores liked content counts after a browser refresh", async () => {
        window.localStorage.setItem("zfeed.auth.session", JSON.stringify(session));
        const firstLoad = await import("./contentInteractionStore");
        firstLoad.setContentReactionOverride("1001", "like", true, 1);

        vi.resetModules();

        const refreshedLoad = await import("./contentInteractionStore");
        expect(refreshedLoad.mergeContentInteractionState({
            content_id: "1001",
            is_liked: false,
            like_count: 0
        })).toMatchObject({
            is_liked: true,
            like_count: 1
        });
    });
});
