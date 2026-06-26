import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import {
    ApiError,
    apiRequest,
    authorizedFetch,
    commentContent,
    deleteComment,
    editArticle,
    editVideo,
    favoriteContent,
    getApiBaseUrl,
    getContentDetail,
    getEditableContentDetail,
    getFollowFeed,
    getRecommendFeed,
    getUserFavoriteFeed,
    getUserFollowers,
    getUserProfile,
    getUserPublishedFeed,
    likeContent,
    searchContents,
    searchUsers,
    unlikeContent,
    unfavoriteContent
} from "./apiClient";
import { saveAuthSession } from "./authStore";

function jsonResponse(body: unknown, init?: ResponseInit) {
    return new Response(JSON.stringify(body), {
        headers: { "Content-Type": "application/json" },
        status: 200,
        ...init
    });
}

describe("apiClient", () => {
    beforeEach(() => {
        window.localStorage.clear();
        vi.restoreAllMocks();
        vi.unstubAllGlobals();
    });

    afterEach(() => {
        window.localStorage.clear();
        vi.unstubAllGlobals();
        vi.unstubAllEnvs();
    });

    it("uses a configurable API base URL for JSON requests", async () => {
        vi.stubEnv("VITE_API_BASE_URL", "https://api.example.test");
        const fetchMock = vi.fn(async () => jsonResponse({ ok: true }));
        vi.stubGlobal("fetch", fetchMock);

        await apiRequest("/v1/feed/recommend", {
            method: "POST",
            body: { cursor: 0 }
        });

        expect(getApiBaseUrl()).toBe("https://api.example.test");
        expect(fetchMock).toHaveBeenCalledWith("https://api.example.test/v1/feed/recommend", expect.objectContaining({
            body: JSON.stringify({ cursor: 0 }),
            method: "POST"
        }));
    });

    it("authorizedFetch adds Bearer token and clears it on auth failure", async () => {
        saveAuthSession({
            token: "secret-token",
            expiredAt: Math.floor(Date.now() / 1000) + 3600
        });
        const fetchMock = vi.fn(async () => jsonResponse({ message: "raw token failure" }, { status: 403 }));
        vi.stubGlobal("fetch", fetchMock);

        await expect(authorizedFetch("/v1/private", { method: "POST", body: { ok: true } })).rejects.toMatchObject({
            status: 403,
            message: "登录状态已失效"
        });

        expect(fetchMock).toHaveBeenCalledWith("/v1/private", expect.objectContaining({
            headers: expect.objectContaining({
                Authorization: "Bearer secret-token"
            })
        }));
        expect(window.localStorage.getItem("zfeed.auth.session")).toBeNull();
    });

    it("blocks authenticated absolute URLs before sending the Bearer token", async () => {
        saveAuthSession({
            token: "secret-token",
            expiredAt: Math.floor(Date.now() / 1000) + 3600
        });
        const fetchMock = vi.fn();
        vi.stubGlobal("fetch", fetchMock);

        await expect(authorizedFetch("https://evil.example/v1/private")).rejects.toMatchObject({
            status: 400,
            message: "认证请求只能发送到站内 API"
        });

        expect(fetchMock).not.toHaveBeenCalled();
    });

    it("blocks authenticated write operations before fetch when there is no valid session", async () => {
        const fetchMock = vi.fn();
        vi.stubGlobal("fetch", fetchMock);

        await expect(likeContent({ contentId: "article-1" })).rejects.toEqual(new ApiError(401, "请先登录后再操作"));
        await expect(favoriteContent({ contentId: "article-1" })).rejects.toMatchObject({
            status: 401,
            message: "请先登录后再操作"
        });
        expect(fetchMock).not.toHaveBeenCalled();
    });

    it("uses optional Bearer auth for search helpers when a session exists", async () => {
        const fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
            void input;
            void init;
            return jsonResponse({
                items: [],
                next_cursor: 0,
                has_more: false,
                next_page_token: "",
                snapshot_id: ""
            });
        });
        vi.stubGlobal("fetch", fetchMock);

        await searchContents({ query: "AI 创作", page_size: 10, mode: "hybrid" });
        expect(fetchMock).toHaveBeenNthCalledWith(1, "/v1/search/contents", expect.objectContaining({
            method: "POST",
            body: JSON.stringify({ query: "AI 创作", page_size: 10, mode: "hybrid" })
        }));
        expect((fetchMock.mock.calls[0][1] as RequestInit).headers).not.toHaveProperty("Authorization");

        saveAuthSession({
            token: "search-token",
            expiredAt: Math.floor(Date.now() / 1000) + 3600
        });
        await searchUsers({ query: "Mira", page_size: 10, mode: "relevance" });

        expect(fetchMock).toHaveBeenNthCalledWith(2, "/v1/search/users", expect.objectContaining({
            method: "POST",
            headers: expect.objectContaining({
                Authorization: "Bearer search-token"
            }),
            body: JSON.stringify({ query: "Mira", page_size: 10, mode: "relevance" })
        }));
    });

    it("retries the recommend feed as a guest when optional auth is rejected", async () => {
        saveAuthSession({
            token: "reader-token",
            expiredAt: Math.floor(Date.now() / 1000) + 3600
        });
        const fetchMock = vi.fn(async (_input: RequestInfo | URL, init?: RequestInit) => {
            if ((init?.headers as Record<string, string> | undefined)?.Authorization) {
                return jsonResponse({ message: "raw auth failure" }, { status: 403 });
            }

            return jsonResponse({ items: [{ content_id: 1001 }], has_more: false });
        });
        vi.stubGlobal("fetch", fetchMock);

        await expect(getRecommendFeed({ cursor: "", page_size: 20 })).resolves.toEqual({
            items: [{ content_id: 1001 }],
            has_more: false
        });
        expect(window.localStorage.getItem("zfeed.auth.session")).toBeNull();
        expect(fetchMock).toHaveBeenNthCalledWith(1, "/v1/feed/recommend", expect.objectContaining({
            headers: expect.objectContaining({
                Authorization: "Bearer reader-token"
            })
        }));
        expect(fetchMock).toHaveBeenNthCalledWith(2, "/v1/feed/recommend", expect.objectContaining({
            headers: expect.not.objectContaining({
                Authorization: expect.any(String)
            })
        }));
    });

    it("retries public optional reads as a guest when optional auth is rejected", async () => {
        saveAuthSession({
            token: "public-token",
            expiredAt: Math.floor(Date.now() / 1000) + 3600
        });
        const fetchMock = vi.fn(async (_input: RequestInfo | URL, init?: RequestInit) => {
            if ((init?.headers as Record<string, string> | undefined)?.Authorization) {
                return jsonResponse({ message: "raw auth failure" }, { status: 401 });
            }

            return jsonResponse({ ok: true });
        });
        vi.stubGlobal("fetch", fetchMock);

        await expect(searchContents({ query: "AI", page_size: 10 })).resolves.toEqual({ ok: true });
        await expect(getContentDetail({ content_id: "1001" })).resolves.toEqual({ ok: true });
        await expect(getUserProfile("1001")).resolves.toEqual({ ok: true });

        expect(window.localStorage.getItem("zfeed.auth.session")).toBeNull();
        expect(fetchMock).toHaveBeenCalledTimes(4);
        expect(fetchMock).toHaveBeenNthCalledWith(1, "/v1/search/contents", expect.objectContaining({
            headers: expect.objectContaining({
                Authorization: "Bearer public-token"
            })
        }));
        expect(fetchMock).toHaveBeenNthCalledWith(2, "/v1/search/contents", expect.objectContaining({
            headers: expect.not.objectContaining({
                Authorization: expect.any(String)
            })
        }));
        expect(fetchMock).toHaveBeenNthCalledWith(3, "/v1/content/detail", expect.objectContaining({
            headers: expect.not.objectContaining({
                Authorization: expect.any(String)
            })
        }));
        expect(fetchMock).toHaveBeenNthCalledWith(4, "/v1/user/profile/1001", expect.objectContaining({
            headers: expect.not.objectContaining({
                Authorization: expect.any(String)
            })
        }));
    });

    it("retries the user published feed as a guest when optional auth is rejected", async () => {
        saveAuthSession({
            token: "published-token",
            expiredAt: Math.floor(Date.now() / 1000) + 3600
        });
        const fetchMock = vi.fn(async (_input: RequestInfo | URL, init?: RequestInit) => {
            if ((init?.headers as Record<string, string> | undefined)?.Authorization) {
                return jsonResponse({ message: "raw auth failure" }, { status: 403 });
            }

            return jsonResponse({ items: [{ content_id: 9101 }], has_more: false });
        });
        vi.stubGlobal("fetch", fetchMock);

        await expect(getUserPublishedFeed({ user_id: "1001", cursor: "", page_size: 20 })).resolves.toEqual({
            items: [{ content_id: 9101 }],
            has_more: false
        });
        expect(window.localStorage.getItem("zfeed.auth.session")).toBeNull();
        expect(fetchMock).toHaveBeenNthCalledWith(1, "/v1/feed/user/publish", expect.objectContaining({
            headers: expect.objectContaining({
                Authorization: "Bearer published-token"
            }),
            body: JSON.stringify({ user_id: "1001", cursor: "", page_size: 20 })
        }));
        expect(fetchMock).toHaveBeenNthCalledWith(2, "/v1/feed/user/publish", expect.objectContaining({
            headers: expect.not.objectContaining({
                Authorization: expect.any(String)
            }),
            body: JSON.stringify({ user_id: "1001", cursor: "", page_size: 20 })
        }));
    });

    it("keeps follow feed auth failures as auth-required instead of retrying as a guest", async () => {
        saveAuthSession({
            token: "follow-token",
            expiredAt: Math.floor(Date.now() / 1000) + 3600
        });
        const fetchMock = vi.fn(async () => jsonResponse({ message: "raw auth failure" }, { status: 403 }));
        vi.stubGlobal("fetch", fetchMock);

        await expect(getFollowFeed({ cursor: "", page_size: 20 })).rejects.toMatchObject({
            status: 403,
            message: "登录状态已失效"
        });
        expect(window.localStorage.getItem("zfeed.auth.session")).toBeNull();
        expect(fetchMock).toHaveBeenCalledTimes(1);
    });

    it("serializes optional read helpers for feed and content detail", async () => {
        saveAuthSession({
            token: "reader-token",
            expiredAt: Math.floor(Date.now() / 1000) + 3600
        });
        const fetchMock = vi.fn(async () => jsonResponse({ items: [], has_more: false }));
        vi.stubGlobal("fetch", fetchMock);

        await getRecommendFeed({ cursor: "", page_size: 20, snapshot_id: "snap-1" });
        await getFollowFeed({ cursor: "", page_size: 20 });
        await getContentDetail({ content_id: "1001" });
        await getUserProfile("1001");
        await getUserPublishedFeed({ user_id: "1001", cursor: "", page_size: 20 });
        await getUserFavoriteFeed({ user_id: "1001", cursor: "", page_size: 20 });
        await getUserFollowers({ user_id: "1001", cursor: 0, page_size: 20 });

        expect(fetchMock).toHaveBeenNthCalledWith(1, "/v1/feed/recommend", expect.objectContaining({
            method: "POST",
            headers: expect.objectContaining({
                Authorization: "Bearer reader-token"
            }),
            body: JSON.stringify({ cursor: "", page_size: 20, snapshot_id: "snap-1" })
        }));
        expect(fetchMock).toHaveBeenNthCalledWith(2, "/v1/feed/follow", expect.objectContaining({
            method: "POST",
            headers: expect.objectContaining({
                Authorization: "Bearer reader-token"
            }),
            body: JSON.stringify({ cursor: "", page_size: 20 })
        }));
        expect(fetchMock).toHaveBeenNthCalledWith(3, "/v1/content/detail", expect.objectContaining({
            method: "POST",
            headers: expect.objectContaining({
                Authorization: "Bearer reader-token"
            }),
            body: JSON.stringify({ content_id: "1001" })
        }));
        expect(fetchMock).toHaveBeenNthCalledWith(4, "/v1/user/profile/1001", expect.objectContaining({
            method: "GET",
            headers: expect.objectContaining({
                Authorization: "Bearer reader-token"
            })
        }));
        expect(fetchMock).toHaveBeenNthCalledWith(5, "/v1/feed/user/publish", expect.objectContaining({
            method: "POST",
            headers: expect.objectContaining({
                Authorization: "Bearer reader-token"
            }),
            body: JSON.stringify({ user_id: "1001", cursor: "", page_size: 20 })
        }));
        expect(fetchMock).toHaveBeenNthCalledWith(6, "/v1/feed/user/favorite", expect.objectContaining({
            method: "POST",
            headers: expect.objectContaining({
                Authorization: "Bearer reader-token"
            }),
            body: JSON.stringify({ user_id: "1001", cursor: "", page_size: 20 })
        }));
        expect(fetchMock).toHaveBeenNthCalledWith(7, "/v1/user/followers", expect.objectContaining({
            method: "POST",
            headers: expect.objectContaining({
                Authorization: "Bearer reader-token"
            }),
            body: JSON.stringify({ user_id: "1001", cursor: 0, page_size: 20 })
        }));
    });

    it("serializes authenticated content edit helpers", async () => {
        saveAuthSession({
            token: "editor-token",
            expiredAt: Math.floor(Date.now() / 1000) + 3600
        });
        const fetchMock = vi.fn(async () => jsonResponse({ content_id: 1001 }));
        vi.stubGlobal("fetch", fetchMock);

        await getEditableContentDetail({ content_id: "1001" });
        await editArticle("1001", {
            title: "更新后的标题",
            description: "更新后的摘要",
            cover: "https://example.com/cover.png",
            content: "更新后的正文"
        });
        await editVideo("1002", {
            title: "更新后的视频",
            description: "更新后的视频摘要",
            video_url: "https://example.com/video.mp4",
            cover_url: "https://example.com/video-cover.png",
            duration: 120
        });

        expect(fetchMock).toHaveBeenNthCalledWith(1, "/v1/content/detail", expect.objectContaining({
            method: "POST",
            headers: expect.objectContaining({
                Authorization: "Bearer editor-token"
            }),
            body: JSON.stringify({ content_id: "1001" })
        }));
        expect(fetchMock).toHaveBeenNthCalledWith(2, "/v1/content/article/1001", expect.objectContaining({
            method: "PUT",
            headers: expect.objectContaining({
                Authorization: "Bearer editor-token"
            }),
            body: JSON.stringify({
                title: "更新后的标题",
                description: "更新后的摘要",
                cover: "https://example.com/cover.png",
                content: "更新后的正文"
            })
        }));
        expect(fetchMock).toHaveBeenNthCalledWith(3, "/v1/content/video/1002", expect.objectContaining({
            method: "PUT",
            headers: expect.objectContaining({
                Authorization: "Bearer editor-token"
            }),
            body: JSON.stringify({
                title: "更新后的视频",
                description: "更新后的视频摘要",
                video_url: "https://example.com/video.mp4",
                cover_url: "https://example.com/video-cover.png",
                duration: 120
            })
        }));
    });

    it("serializes interaction write helpers with documented backend field names", async () => {
        saveAuthSession({
            token: "write-token",
            expiredAt: Math.floor(Date.now() / 1000) + 3600
        });
        const fetchMock = vi.fn(async () => jsonResponse({}));
        vi.stubGlobal("fetch", fetchMock);

        await likeContent({ contentId: "1001", contentUserId: "2001", scene: "ARTICLE" });
        await unlikeContent({ contentId: "1001", scene: "ARTICLE" });
        await favoriteContent({ contentId: "1001", scene: "ARTICLE" });
        await unfavoriteContent({ contentId: "1001", scene: "ARTICLE" });
        await commentContent({ content_id: "1001", scene: "ARTICLE", comment: "hello", content_user_id: "2001" });
        await deleteComment({ comment_id: "3001", content_id: "1001", scene: "ARTICLE" });

        expect(fetchMock).toHaveBeenNthCalledWith(1, "/v1/interaction/like", expect.objectContaining({
            body: JSON.stringify({ content_id: "1001", content_user_id: "2001", scene: "ARTICLE" })
        }));
        expect(fetchMock).toHaveBeenNthCalledWith(2, "/v1/interaction/unlike", expect.objectContaining({
            body: JSON.stringify({ content_id: "1001", scene: "ARTICLE" })
        }));
        expect(fetchMock).toHaveBeenNthCalledWith(3, "/v1/interaction/favorite", expect.objectContaining({
            body: JSON.stringify({ content_id: "1001", scene: "ARTICLE" })
        }));
        expect(fetchMock).toHaveBeenNthCalledWith(4, "/v1/interaction/favorite", expect.objectContaining({
            method: "DELETE",
            body: JSON.stringify({ content_id: "1001", scene: "ARTICLE" })
        }));
        expect(fetchMock).toHaveBeenNthCalledWith(5, "/v1/interaction/comment", expect.objectContaining({
            body: JSON.stringify({ content_id: "1001", scene: "ARTICLE", comment: "hello", content_user_id: "2001" })
        }));
        expect(fetchMock).toHaveBeenNthCalledWith(6, "/v1/interaction/comment", expect.objectContaining({
            method: "DELETE",
            body: JSON.stringify({ comment_id: "3001", content_id: "1001", scene: "ARTICLE" })
        }));
    });

    it("defaults content interaction scene to the backend ARTICLE enum", async () => {
        saveAuthSession({
            token: "write-token",
            expiredAt: Math.floor(Date.now() / 1000) + 3600
        });
        const fetchMock = vi.fn(async () => jsonResponse({}));
        vi.stubGlobal("fetch", fetchMock);

        await likeContent({ contentId: "1001", contentUserId: "2001" });

        expect(fetchMock).toHaveBeenCalledWith("/v1/interaction/like", expect.objectContaining({
            body: JSON.stringify({ content_id: "1001", content_user_id: "2001", scene: "ARTICLE" })
        }));
    });

    it("normalizes backend and network errors without exposing raw backend messages", async () => {
        vi.stubGlobal("fetch", vi.fn(async () => jsonResponse({ message: "database password leaked" }, { status: 500 })));

        await expect(apiRequest("/v1/feed/recommend")).rejects.toMatchObject({
            status: 500,
            message: "请求失败，请稍后重试"
        });

        vi.stubGlobal("fetch", vi.fn(async () => {
            throw new Error("connect ECONNREFUSED 10.0.0.1");
        }));

        await expect(apiRequest("/v1/feed/recommend")).rejects.toMatchObject({
            status: 0,
            message: "网络连接失败，请稍后重试"
        });
    });

    it("normalizes invalid JSON responses instead of leaking parser errors", async () => {
        vi.stubGlobal("fetch", vi.fn(async () => new Response("{bad json", {
            headers: { "Content-Type": "application/json" },
            status: 200
        })));

        await expect(apiRequest("/v1/feed/recommend")).rejects.toMatchObject({
            status: 200,
            message: "响应解析失败，请稍后重试"
        });
    });
});
