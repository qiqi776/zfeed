import { act, fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { App } from "./App";
import { resetContentInteractionState } from "./runtime/contentInteractionStore";

function jsonResponse(body: unknown, init?: ResponseInit) {
    return new Response(JSON.stringify(body), {
        headers: { "Content-Type": "application/json" },
        status: 200,
        ...init
    });
}

function recommendFeedPayload(items: unknown[] = [defaultRecommendFeedItem()]) {
    return {
        items,
        next_cursor: "",
        has_more: false,
        snapshot_id: "recommend-snapshot"
    };
}

function defaultRecommendFeedItem(overrides: Record<string, unknown> = {}) {
    return {
        content_id: 1001,
        content_type: 1,
        author_id: 1001,
        author_name: "Jax Lee",
        author_avatar: "https://example.com/jax.png",
        title: "用 AI 构建产品：30 天从 0 到 1",
        description: "记录从 0 到 1 的完整过程，包括技术栈选择、产品思考和踩坑经验。",
        cover_url: "https://example.com/cover.png",
        published_at: 1765670400,
        like_count: 128,
        favorite_count: 36,
        comment_count: 12,
        is_liked: true,
        is_favorited: false,
        ...overrides
    };
}

function isRecommendFeedRequest(input: RequestInfo | URL) {
    return input === "/v1/feed/recommend";
}

function meProfilePayload(overrides: Record<string, unknown> = {}) {
    return {
        user_info: {
            user_id: 7,
            mobile: "13800138000",
            nickname: "Mira Chen",
            avatar: "https://example.com/avatar.png",
            bio: "关注创作者工具、液态玻璃界面和高质量信息流体验。",
            gender: 0,
            status: 1,
            email: "mira@example.com",
            birthday: 0,
            ...(overrides.user_info as Record<string, unknown> | undefined)
        },
        followee_count: 346,
        follower_count: 18600,
        like_received_count: 9200,
        favorite_received_count: 912,
        content_count: 128,
        ...overrides
    };
}

function userProfilePayload(overrides: Record<string, unknown> = {}) {
    return {
        user_info: {
            user_id: 1001,
            nickname: "Jax Lee",
            avatar: "https://example.com/jax.png",
            bio: "AI 产品作者",
            mobile: "",
            email: "",
            ...(overrides.user_info as Record<string, unknown> | undefined)
        },
        followee_count: 18,
        follower_count: 2400,
        like_received_count: 1200,
        favorite_received_count: 300,
        content_count: 16,
        is_following: false,
        ...overrides
    };
}

function backendUserProfilePayload(overrides: Record<string, unknown> = {}) {
    return {
        user_profile: {
            user_id: 2001,
            nickname: "后端作者",
            avatar: "https://example.com/backend-author.png",
            bio: "后端真实用户简介",
            gender: 0,
            ...(overrides.user_profile as Record<string, unknown> | undefined)
        },
        counts: {
            followee_count: 6,
            follower_count: 21,
            like_received_count: 34,
            favorite_received_count: 8,
            content_count: 13,
            ...(overrides.counts as Record<string, unknown> | undefined)
        },
        viewer: {
            is_following: true,
            ...(overrides.viewer as Record<string, unknown> | undefined)
        },
        ...overrides
    };
}

function userPublishedFeedPayload(items: unknown[] = [defaultRecommendFeedItem({
    content_id: 9101,
    author_id: 1001,
    author_name: "Jax Lee",
    title: "用户主页真实发布内容",
    description: "这条内容来自用户发布流接口。",
    like_count: 9,
    favorite_count: 4,
    comment_count: 2
})]) {
    return {
        items,
        cursor: "",
        has_more: false
    };
}

function userFavoriteFeedPayload(items: unknown[] = [defaultRecommendFeedItem({
    content_id: 9201,
    author_id: 1002,
    author_name: "收藏作者",
    title: "我的真实收藏内容",
    description: "这条内容来自我的收藏流接口。",
    like_count: 19,
    favorite_count: 7,
    comment_count: 3
})]) {
    return {
        items,
        next_cursor: "",
        has_more: false
    };
}

function userFollowersPayload(items: unknown[] = [{
    user_id: 3001,
    nickname: "粉丝用户",
    avatar: "3",
    bio: "关注我的用户",
    is_following: false
}]) {
    return {
        items,
        next_cursor: 0,
        has_more: false
    };
}

function contentDetailPayload(overrides: Record<string, unknown> = {}) {
    return {
        detail: {
            content_id: "1001",
            content_type: 1,
            author_id: "7",
            author_name: "Mira Chen",
            author_avatar: "https://example.com/avatar.png",
            title: "原始文章标题",
            description: "原始文章摘要",
            cover_url: "https://example.com/cover.png",
            article_content: "第一段正文。\n第二段正文。",
            video_url: "",
            video_duration: 0,
            published_at: 1765670400,
            like_count: 12,
            favorite_count: 5,
            comment_count: 3,
            is_liked: false,
            is_favorited: false,
            is_following_author: false,
            ...overrides
        }
    };
}

function defaultCommentItem(overrides: Record<string, unknown> = {}) {
    return {
        comment_id: "3001",
        content_id: "1001",
        user_id: "2001",
        nickname: "测试1",
        avatar: "1",
        comment: "后端评论1",
        root_id: "3001",
        parent_id: "0",
        reply_to_user_id: "0",
        created_at: 1765670400,
        reply_count: 0,
        ...overrides
    };
}

function commentListPayload(items: unknown[] = [defaultCommentItem()]) {
    return {
        comments: items,
        next_cursor: 0,
        has_more: false
    };
}

function isContentDetailRequest(input: RequestInfo | URL) {
    return input === "/v1/content/detail";
}

function isCommentListRequest(input: RequestInfo | URL) {
    return input === "/v1/interaction/comment/list";
}

function respondToProfileListRequest(input: RequestInfo | URL) {
    if (input === "/v1/feed/user/favorite") {
        return jsonResponse(userFavoriteFeedPayload([]));
    }

    if (input === "/v1/user/followers") {
        return jsonResponse(userFollowersPayload([]));
    }

    return null;
}

describe("App routes", () => {
    beforeEach(() => {
        window.localStorage.clear();
        resetContentInteractionState();
        vi.restoreAllMocks();
        vi.unstubAllGlobals();
    });

    afterEach(() => {
        window.localStorage.clear();
        resetContentInteractionState();
        vi.unstubAllGlobals();
    });

    it("redirects unauthenticated root visits to the login entry", async () => {
        window.history.pushState({}, "", "/");

        render(<App />);

        await waitFor(() => expect(window.location.pathname).toBe("/login"));
        expect(await screen.findByRole("heading", { name: "登录 zfeed" })).toBeInTheDocument();
        expect(screen.getByTitle("zfeed 首页雾化背景")).toBeInTheDocument();
    });

    it("restores a valid session at the root path and enters the home feed", async () => {
        window.localStorage.setItem("zfeed.auth.session", JSON.stringify({
            token: "valid-token",
            expiredAt: Math.floor(Date.now() / 1000) + 3600,
            user: { userId: 7 }
        }));
        const fetchMock = vi.fn(async () => jsonResponse({
            user_info: {
                user_id: 7,
                mobile: "13800138000",
                nickname: "Mira Chen",
                avatar: "https://example.com/avatar.png",
                bio: "",
                gender: 0,
                status: 1,
                email: "",
                birthday: 0
            },
            followee_count: 1,
            follower_count: 2,
            like_received_count: 3,
            favorite_received_count: 4,
            content_count: 5
        }));
        vi.stubGlobal("fetch", fetchMock);
        window.history.pushState({}, "", "/");

        render(<App />);

        await waitFor(() => expect(window.location.pathname).toBe("/home"));
        expect(fetchMock).toHaveBeenCalledWith("/v1/users/me", expect.objectContaining({
            headers: expect.objectContaining({
                Authorization: "Bearer valid-token"
            })
        }));
        expect(await screen.findByText("用 AI 构建产品：30 天从 0 到 1")).toBeInTheDocument();
    });

    it("clears an invalid restored session and enters the guest home feed", async () => {
        window.localStorage.setItem("zfeed.auth.session", JSON.stringify({
            token: "expired-token",
            expiredAt: Math.floor(Date.now() / 1000) + 3600
        }));
        vi.stubGlobal("fetch", vi.fn(async () => jsonResponse({ message: "expired" }, { status: 401 })));
        window.history.pushState({}, "", "/");

        render(<App />);

        await waitFor(() => expect(window.location.pathname).toBe("/home"));
        expect(window.localStorage.getItem("zfeed.auth.session")).toBeNull();
    });

    it("keeps the restored session when validation fails for a non-auth reason", async () => {
        window.localStorage.setItem("zfeed.auth.session", JSON.stringify({
            token: "valid-token",
            expiredAt: Math.floor(Date.now() / 1000) + 3600,
            user: { userId: 7 }
        }));
        const fetchMock = vi.fn(async (input: RequestInfo | URL) => {
            if (input === "/v1/users/me") {
                return jsonResponse({ message: "server unavailable" }, { status: 500 });
            }

            return jsonResponse(recommendFeedPayload());
        });
        vi.stubGlobal("fetch", fetchMock);
        window.history.pushState({}, "", "/");

        render(<App />);

        await waitFor(() => expect(window.location.pathname).toBe("/home"));
        expect(window.localStorage.getItem("zfeed.auth.session")).not.toBeNull();
        expect(await screen.findByText("用 AI 构建产品：30 天从 0 到 1")).toBeInTheDocument();
        await waitFor(() => expect(fetchMock).toHaveBeenCalledWith("/v1/feed/recommend", expect.objectContaining({
            headers: expect.objectContaining({
                Authorization: "Bearer valid-token"
            })
        })));
    });

    it("clears a locally expired root session and enters the guest home feed", async () => {
        window.localStorage.setItem("zfeed.auth.session", JSON.stringify({
            token: "locally-expired-token",
            expiredAt: Math.floor(Date.now() / 1000) - 1
        }));
        const fetchMock = vi.fn(async () => jsonResponse(recommendFeedPayload()));
        vi.stubGlobal("fetch", fetchMock);
        window.history.pushState({}, "", "/");

        render(<App />);

        await waitFor(() => expect(window.location.pathname).toBe("/home"));
        expect(window.localStorage.getItem("zfeed.auth.session")).toBeNull();
        expect(fetchMock).not.toHaveBeenCalledWith("/v1/users/me", expect.any(Object));
        expect(await screen.findByText("用 AI 构建产品：30 天从 0 到 1")).toBeInTheDocument();
    });

    it("falls back to the guest home feed when session restore never resolves", async () => {
        vi.useFakeTimers();
        try {
            window.localStorage.setItem("zfeed.auth.session", JSON.stringify({
                token: "hanging-token",
                expiredAt: Math.floor(Date.now() / 1000) + 3600
            }));
            vi.stubGlobal("fetch", vi.fn(() => new Promise(() => undefined)));
            window.history.pushState({}, "", "/");

            render(<App />);

            const routeLoading = screen.queryByText("正在加载...");
            if (routeLoading) {
                expect(routeLoading).toHaveAttribute("data-page-state", "loading");
            }

            await act(async () => {
                await vi.dynamicImportSettled();
            });
            const restoreState = screen.getByRole("heading", { name: "正在恢复 zfeed 会话" });
            expect(restoreState.closest("[data-page-state]")).toHaveAttribute("data-page-state", "loading");

            await act(async () => {
                vi.advanceTimersByTime(5_000);
            });

            expect(window.location.pathname).toBe("/home");
            expect(window.localStorage.getItem("zfeed.auth.session")).toBeNull();
        } finally {
            vi.useRealTimers();
        }
    });

    it("loads the home recommend feed from the API without executing user content", async () => {
        const fetchMock = vi.fn(async () => jsonResponse(recommendFeedPayload([
            defaultRecommendFeedItem({
                author_name: "Jax <owner>",
                title: "接口推荐内容 <script>alert(1)</script>",
                description: "接口返回的推荐流摘要"
            })
        ])));
        vi.stubGlobal("fetch", fetchMock);
        window.history.pushState({}, "", "/home");

        render(<App />);

        expect(await screen.findByText("接口推荐内容 <script>alert(1)</script>")).toBeInTheDocument();
        expect(screen.getAllByText("Jax <owner>").length).toBeGreaterThan(0);
        expect(screen.getAllByText("接口返回的推荐流摘要").length).toBeGreaterThan(0);
        expect(document.querySelector("script")).toBeNull();
        expect(fetchMock).toHaveBeenCalledWith("/v1/feed/recommend", expect.objectContaining({
            method: "POST",
            body: JSON.stringify({ cursor: "", page_size: 20 })
        }));
    });

    it("renders home recommendations from backend feed authors", async () => {
        const fetchMock = vi.fn(async () => jsonResponse(recommendFeedPayload([
            defaultRecommendFeedItem({
                author_id: 1,
                author_name: "测试1",
                author_avatar: "1",
                title: "测试内容1",
                description: "测试内容1"
            }),
            defaultRecommendFeedItem({
                content_id: 1002,
                author_id: 2,
                author_name: "测试2",
                author_avatar: "2",
                title: "测试内容2",
                description: "测试内容2"
            })
        ])));
        vi.stubGlobal("fetch", fetchMock);
        window.history.pushState({}, "", "/home");

        render(<App />);

        const suggestedPanel = await screen.findByRole("heading", { name: "为你推荐" }).then((heading) => heading.closest(".glass-panel"));
        expect(suggestedPanel).not.toBeNull();
        await waitFor(() => expect(within(suggestedPanel as HTMLElement).getByText("测试1")).toBeInTheDocument());
        expect(within(suggestedPanel as HTMLElement).getByText("测试内容1")).toBeInTheDocument();
        expect(within(suggestedPanel as HTMLElement).getByText("测试2")).toBeInTheDocument();
        expect(document.querySelector('img[src="1"]')).toBeNull();
        expect(screen.queryByText("Zhang Xiaolong")).not.toBeInTheDocument();
        expect(screen.queryByText("AI 前线")).not.toBeInTheDocument();
    });

    it("shows an empty state when the home recommend feed has no items", async () => {
        vi.stubGlobal("fetch", vi.fn(async () => jsonResponse(recommendFeedPayload([]))));
        window.history.pushState({}, "", "/home");

        render(<App />);

        const emptyState = await screen.findByText("暂时没有内容");
        expect(emptyState.closest("[data-page-state]")).toHaveAttribute("data-page-state", "empty");
    });

    it("shows an error state when the home recommend feed cannot be loaded", async () => {
        vi.stubGlobal("fetch", vi.fn(async () => jsonResponse({}, { status: 500 })));
        window.history.pushState({}, "", "/home");

        render(<App />);

        const errorState = await screen.findByText("推荐流加载失败");
        expect(errorState.closest("[data-page-state]")).toHaveAttribute("data-page-state", "error");
        expect(screen.queryByText("用 AI 构建产品：30 天从 0 到 1")).not.toBeInTheDocument();
    });

    it("returns the home feed to guest mode when optional auth is rejected", async () => {
        window.localStorage.setItem("zfeed.auth.session", JSON.stringify({
            token: "expired-home-token",
            expiredAt: Math.floor(Date.now() / 1000) + 3600,
            user: { userId: 7 }
        }));
        const fetchMock = vi.fn(async (_input: RequestInfo | URL, init?: RequestInit) => {
            if ((init?.headers as Record<string, string> | undefined)?.Authorization) {
                return jsonResponse({ message: "raw token rejected" }, { status: 403 });
            }

            return jsonResponse(recommendFeedPayload([
                defaultRecommendFeedItem({
                    title: "游客模式推荐内容"
                })
            ]));
        });
        vi.stubGlobal("fetch", fetchMock);
        window.history.pushState({}, "", "/home");

        render(<App />);

        expect(await screen.findByRole("heading", { name: "游客模式推荐内容" })).toBeInTheDocument();
        expect(window.localStorage.getItem("zfeed.auth.session")).toBeNull();
        expect(fetchMock).toHaveBeenCalledTimes(2);
        expect(fetchMock).toHaveBeenNthCalledWith(1, "/v1/feed/recommend", expect.objectContaining({
            headers: expect.objectContaining({
                Authorization: "Bearer expired-home-token"
            })
        }));
        expect(fetchMock).toHaveBeenNthCalledWith(2, "/v1/feed/recommend", expect.objectContaining({
            headers: expect.not.objectContaining({
                Authorization: expect.any(String)
            })
        }));
        expect(screen.queryByText("raw token rejected")).not.toBeInTheDocument();
    });

    it("loads the signed-in following feed from the API", async () => {
        window.localStorage.setItem("zfeed.auth.session", JSON.stringify({
            token: "following-token",
            expiredAt: Math.floor(Date.now() / 1000) + 3600,
            user: { userId: 7 }
        }));
        const fetchMock = vi.fn(async () => jsonResponse({
            items: [{
                content_id: 1001,
                content_type: 1,
                author_id: 1002,
                author_name: "关注作者 <safe>",
                author_avatar: "https://example.com/author.png",
                title: "接口关注内容 <script>alert(1)</script>",
                description: "来自关注作者的新内容",
                cover_url: "https://example.com/follow-cover.png",
                published_at: 1765670400,
                like_count: 6,
                favorite_count: 2,
                comment_count: 1,
                is_liked: false,
                is_favorited: false
            }],
            next_cursor: "",
            has_more: false,
            snapshot_id: "follow-snapshot"
        }));
        vi.stubGlobal("fetch", fetchMock);
        window.history.pushState({}, "", "/following");

        render(<App />);

        expect(await screen.findByText("接口关注内容 <script>alert(1)</script>")).toBeInTheDocument();
        expect(within(screen.getByRole("main")).getByText("关注作者 <safe>")).toBeInTheDocument();
        expect(within(screen.getByRole("main")).getByText("来自关注作者的新内容")).toBeInTheDocument();
        expect(document.querySelector("script")).toBeNull();
        expect(fetchMock).toHaveBeenCalledWith("/v1/feed/follow", expect.objectContaining({
            method: "POST",
            headers: expect.objectContaining({
                Authorization: "Bearer following-token"
            }),
            body: JSON.stringify({ cursor: "", page_size: 20 })
        }));
    });

    it("renders following feed cards with the same visual classes as recommendation cards", async () => {
        const homeFetchMock = vi.fn(async () => jsonResponse(recommendFeedPayload([
            defaultRecommendFeedItem({
                author_name: "首页样式作者",
                title: "首页样式基准卡片",
                description: "推荐流卡片样式基准"
            })
        ])));
        vi.stubGlobal("fetch", homeFetchMock);
        window.history.pushState({}, "", "/home");

        const { unmount } = render(<App />);

        const homeHeading = await screen.findByRole("heading", { name: "首页样式基准卡片" });
        const homeArticle = homeHeading.closest("article");
        expect(homeArticle).not.toBeNull();
        const homeActionButton = within(homeArticle as HTMLElement).getByRole("button", { name: "取消点赞" });
        const homeBookmarkButton = within(homeArticle as HTMLElement).getByRole("button", { name: "收藏" });
        const expectedArticleClass = homeArticle?.className;
        const expectedHeadingClass = homeHeading.className;
        const expectedLikeButtonClass = homeActionButton.className;
        const expectedBookmarkButtonClass = homeBookmarkButton.className;
        unmount();

        window.localStorage.setItem("zfeed.auth.session", JSON.stringify({
            token: "following-token",
            expiredAt: Math.floor(Date.now() / 1000) + 3600,
            user: { userId: 7 }
        }));
        const followingFetchMock = vi.fn(async () => jsonResponse({
            items: [{
                content_id: 2001,
                content_type: 1,
                author_id: 2002,
                author_name: "关注样式作者",
                title: "关注样式卡片",
                description: "关注流卡片应该复用推荐流样式",
                published_at: 1765670400,
                like_count: 6,
                favorite_count: 2,
                comment_count: 1,
                is_liked: true,
                is_favorited: false
            }],
            next_cursor: "",
            has_more: false
        }));
        vi.stubGlobal("fetch", followingFetchMock);
        window.history.pushState({}, "", "/following");

        render(<App />);

        const followingHeading = await screen.findByRole("heading", { name: "关注样式卡片" });
        const followingArticle = followingHeading.closest("article");
        expect(followingArticle).not.toBeNull();
        expect(followingArticle?.className).toBe(expectedArticleClass);
        expect(followingHeading.className).toBe(expectedHeadingClass);
        expect(within(followingArticle as HTMLElement).getByRole("button", { name: "取消点赞" }).className).toBe(expectedLikeButtonClass);
        expect(within(followingArticle as HTMLElement).getByRole("button", { name: "收藏" }).className).toBe(expectedBookmarkButtonClass);
    });

    it("keeps the home side rails when navigating to the following feed", async () => {
        window.localStorage.setItem("zfeed.auth.session", JSON.stringify({
            token: "following-token",
            expiredAt: Math.floor(Date.now() / 1000) + 3600,
            user: { userId: 7 }
        }));
        const fetchMock = vi.fn(async (input: RequestInfo | URL) => {
            if (input === "/v1/feed/recommend") {
                return jsonResponse(recommendFeedPayload([
                    defaultRecommendFeedItem({
                        author_name: "首页推荐作者",
                        title: "首页推荐内容",
                        description: "首页推荐摘要"
                    })
                ]));
            }

            if (input === "/v1/feed/follow") {
                return jsonResponse({
                    items: [{
                        content_id: 2001,
                        content_type: 1,
                        author_id: 2002,
                        author_name: "关注流作者",
                        title: "关注流主栏内容",
                        description: "只切换中间信息流",
                        published_at: 1765670400,
                        like_count: 6,
                        favorite_count: 2,
                        comment_count: 1,
                        is_liked: false,
                        is_favorited: false
                    }],
                    next_cursor: "",
                    has_more: false
                });
            }

            return jsonResponse({});
        });
        vi.stubGlobal("fetch", fetchMock);
        window.history.pushState({}, "", "/home");

        render(<App />);

        expect(await screen.findByRole("heading", { name: "首页推荐内容" })).toBeInTheDocument();
        expect(screen.getByText("频道").closest("aside")).toHaveClass("lg:col-span-2");
        expect(screen.getByRole("heading", { name: "今日数据" }).closest("aside")).toHaveClass("lg:col-span-3");
        const initialLeftRail = screen.getByText("频道").closest("aside") as HTMLElement;
        expect(within(initialLeftRail).getByRole("link", { name: /推荐/ })).toHaveClass("text-primary");
        expect(within(initialLeftRail).getByRole("link", { name: /关注/ })).not.toHaveClass("text-primary");
        const initialSuggestedPanel = screen.getByRole("heading", { name: "为你推荐" }).closest(".glass-panel");
        expect(initialSuggestedPanel).not.toBeNull();
        expect(within(initialSuggestedPanel as HTMLElement).getByText("首页推荐作者")).toBeInTheDocument();

        const header = document.querySelector("header");
        expect(header).not.toBeNull();
        fireEvent.click(within(header as HTMLElement).getByRole("link", { name: "关注" }));

        expect(await screen.findByRole("heading", { name: "关注流主栏内容" })).toBeInTheDocument();
        expect(window.location.pathname).toBe("/following");
        expect(screen.getByText("频道").closest("aside")).toHaveClass("lg:col-span-2");
        expect(screen.getByRole("heading", { name: "今日数据" }).closest("aside")).toHaveClass("lg:col-span-3");
        const followingLeftRail = screen.getByText("频道").closest("aside") as HTMLElement;
        expect(within(followingLeftRail).getByRole("link", { name: /推荐/ })).not.toHaveClass("text-primary");
        expect(within(followingLeftRail).getByRole("link", { name: /关注/ })).toHaveClass("text-primary");
        const suggestedPanel = screen.getByRole("heading", { name: "为你推荐" }).closest(".glass-panel");
        expect(suggestedPanel).not.toBeNull();
        expect(within(suggestedPanel as HTMLElement).getByText("首页推荐作者")).toBeInTheDocument();
        expect(within(suggestedPanel as HTMLElement).queryByText("关注流作者")).not.toBeInTheDocument();
        expect(within(screen.getByRole("main")).getByText("只切换中间信息流")).toBeInTheDocument();
    });

    it("shows an empty state when the following feed has no items", async () => {
        window.localStorage.setItem("zfeed.auth.session", JSON.stringify({
            token: "following-token",
            expiredAt: Math.floor(Date.now() / 1000) + 3600,
            user: { userId: 7 }
        }));
        vi.stubGlobal("fetch", vi.fn(async () => jsonResponse({
            items: [],
            next_cursor: "",
            has_more: false
        })));
        window.history.pushState({}, "", "/following");

        render(<App />);

        const emptyState = await screen.findByText("关注一些作者后，这里会出现他们的新内容");
        expect(emptyState.closest("[data-page-state]")).toHaveAttribute("data-page-state", "empty");
        expect(screen.getByRole("link", { name: "去搜索" })).toHaveAttribute("href", "/search");
    });

    it("shows an error state when the following feed cannot be loaded", async () => {
        window.localStorage.setItem("zfeed.auth.session", JSON.stringify({
            token: "following-token",
            expiredAt: Math.floor(Date.now() / 1000) + 3600,
            user: { userId: 7 }
        }));
        vi.stubGlobal("fetch", vi.fn(async () => jsonResponse({}, { status: 500 })));
        window.history.pushState({}, "", "/following");

        render(<App />);

        const errorState = await screen.findByText("关注流加载失败");
        expect(errorState.closest("[data-page-state]")).toHaveAttribute("data-page-state", "error");
        expect(screen.queryByText("我关注的创作者今天都在用 AI 重构工作流")).not.toBeInTheDocument();
    });

    it("returns the following feed to auth-required when the token is rejected", async () => {
        window.localStorage.setItem("zfeed.auth.session", JSON.stringify({
            token: "following-token",
            expiredAt: Math.floor(Date.now() / 1000) + 3600,
            user: { userId: 7 }
        }));
        vi.stubGlobal("fetch", vi.fn(async () => jsonResponse({}, { status: 401 })));
        window.history.pushState({}, "", "/following");

        render(<App />);

        const authState = await screen.findByText("登录后才能查看关注流。");
        expect(authState.closest("[data-page-state]")).toHaveAttribute("data-page-state", "auth-required");
        expect(window.localStorage.getItem("zfeed.auth.session")).toBeNull();
    });

    it("shows an auth-required state on following when signed out", async () => {
        window.history.pushState({}, "", "/following");

        render(<App />);

        const authState = await screen.findByText("登录后才能查看关注流。");
        expect(authState.closest("[data-page-state]")).toHaveAttribute("data-page-state", "auth-required");
        expect(screen.getByRole("link", { name: "去登录" })).toHaveAttribute("href", "/login?next=%2Ffollowing");
        expect(screen.queryByText("我关注的创作者今天都在用 AI 重构工作流")).not.toBeInTheDocument();
    });

    it("loads the me profile from the API without executing user content", async () => {
        window.localStorage.setItem("zfeed.auth.session", JSON.stringify({
            token: "me-token",
            expiredAt: Math.floor(Date.now() / 1000) + 3600,
            user: { userId: 7 }
        }));
        const fetchMock = vi.fn(async (input: RequestInfo | URL) => {
            if (input === "/v1/users/me") {
                return jsonResponse(meProfilePayload({
                    user_info: {
                        nickname: "接口用户 <script>alert(1)</script>",
                        avatar: "https://example.com/me.png",
                        bio: "接口返回的个人简介"
                    },
                    followee_count: 3,
                    follower_count: 9,
                    like_received_count: 12,
                    favorite_received_count: 4,
                    content_count: 42
                }));
            }

            if (input === "/v1/feed/user/publish") {
                return jsonResponse(userPublishedFeedPayload([]));
            }

            const listResponse = respondToProfileListRequest(input);
            if (listResponse) {
                return listResponse;
            }

            return jsonResponse({});
        });
        vi.stubGlobal("fetch", fetchMock);
        window.history.pushState({}, "", "/me");

        render(<App />);

        expect(await screen.findByRole("heading", { name: "接口用户 <script>alert(1)</script>" })).toBeInTheDocument();
        expect(screen.getByText("接口返回的个人简介")).toBeInTheDocument();
        expect(screen.getAllByText("42").length).toBeGreaterThan(0);
        expect(screen.getAllByText("3").length).toBeGreaterThan(0);
        expect(screen.getAllByText("9").length).toBeGreaterThan(0);
        expect(screen.getAllByText("12").length).toBeGreaterThan(0);
        expect(document.querySelector("script")).toBeNull();
        expect(fetchMock).toHaveBeenCalledWith("/v1/users/me", expect.objectContaining({
            headers: expect.objectContaining({
                Authorization: "Bearer me-token"
            })
        }));
        expect(screen.getByText("编辑资料")).toBeInTheDocument();
    });

    it("renders numeric backend avatars as avatar text on my profile", async () => {
        window.localStorage.setItem("zfeed.auth.session", JSON.stringify({
            token: "me-token",
            expiredAt: Math.floor(Date.now() / 1000) + 3600,
            user: { userId: 20 }
        }));
        const fetchMock = vi.fn(async (input: RequestInfo | URL) => {
            if (input === "/v1/users/me") {
                return jsonResponse(meProfilePayload({
                    user_info: {
                        user_id: 20,
                        mobile: "13800000019",
                        nickname: "测试19",
                        avatar: "19",
                        bio: "测试内容19",
                        email: "test19@zfeed.local"
                    }
                }));
            }

            if (input === "/v1/feed/user/publish") {
                return jsonResponse(userPublishedFeedPayload([]));
            }

            const listResponse = respondToProfileListRequest(input);
            if (listResponse) {
                return listResponse;
            }

            return jsonResponse({});
        });
        vi.stubGlobal("fetch", fetchMock);
        window.history.pushState({}, "", "/me");

        render(<App />);

        expect(await screen.findByRole("heading", { name: "测试19" })).toBeInTheDocument();
        expect(screen.getAllByText("19").length).toBeGreaterThan(0);
        expect(document.querySelector('img[src="19"]')).toBeNull();
    });

    it("loads my published feed from the signed-in user id", async () => {
        window.localStorage.setItem("zfeed.auth.session", JSON.stringify({
            token: "me-token",
            expiredAt: Math.floor(Date.now() / 1000) + 3600,
            user: { userId: 7 }
        }));
        const fetchMock = vi.fn(async (input: RequestInfo | URL) => {
            if (input === "/v1/users/me") {
                return jsonResponse(meProfilePayload());
            }

            if (input === "/v1/feed/user/publish") {
                return jsonResponse(userPublishedFeedPayload([
                    defaultRecommendFeedItem({
                        content_id: 7701,
                        author_id: 7,
                        author_name: "Mira Chen",
                        title: "我的真实发布内容",
                        description: "这条内容来自我的发布流接口。"
                    })
                ]));
            }

            const listResponse = respondToProfileListRequest(input);
            if (listResponse) {
                return listResponse;
            }

            return jsonResponse({});
        });
        vi.stubGlobal("fetch", fetchMock);
        window.history.pushState({}, "", "/me");

        render(<App />);

        expect(await screen.findByRole("heading", { name: "我的真实发布内容" })).toBeInTheDocument();
        expect(screen.getByText("这条内容来自我的发布流接口。")).toBeInTheDocument();
        expect(fetchMock).toHaveBeenCalledWith("/v1/feed/user/publish", expect.objectContaining({
            method: "POST",
            headers: expect.objectContaining({
                Authorization: "Bearer me-token"
            }),
            body: JSON.stringify({ user_id: "7", cursor: "", page_size: 20 })
        }));
        expect(screen.queryByText("个人发布流会在后续接入 `/v1/feed/user/publish`。")).not.toBeInTheDocument();
    });

    it("uses the profile stat row to switch favorites and followers while followings stays display-only", async () => {
        window.localStorage.setItem("zfeed.auth.session", JSON.stringify({
            token: "me-token",
            expiredAt: Math.floor(Date.now() / 1000) + 3600,
            user: { userId: 7 }
        }));
        const fetchMock = vi.fn(async (input: RequestInfo | URL) => {
            if (input === "/v1/users/me") {
                return jsonResponse(meProfilePayload());
            }

            if (input === "/v1/feed/user/publish") {
                return jsonResponse(userPublishedFeedPayload([
                    defaultRecommendFeedItem({
                        content_id: 7701,
                        author_id: 7,
                        author_name: "Mira Chen",
                        title: "我的真实发布内容",
                        description: "这条内容来自我的发布流接口。"
                    })
                ]));
            }

            if (input === "/v1/feed/user/favorite") {
                return jsonResponse(userFavoriteFeedPayload());
            }

            if (input === "/v1/user/followers") {
                return jsonResponse(userFollowersPayload());
            }

            return jsonResponse({});
        });
        vi.stubGlobal("fetch", fetchMock);
        window.history.pushState({}, "", "/me");

        render(<App />);

        expect(screen.queryByRole("tablist", { name: "个人主页内容" })).not.toBeInTheDocument();
        expect(await screen.findByRole("button", { name: "128 动态" })).toHaveAttribute("aria-pressed", "true");
        expect(screen.getByRole("button", { name: "912 收藏" })).toHaveAttribute("aria-pressed", "false");
        expect(screen.getByRole("button", { name: "18600 粉丝" })).toHaveAttribute("aria-pressed", "false");
        expect(screen.queryByRole("button", { name: "346 关注" })).not.toBeInTheDocument();
        expect(screen.getByText("346").closest("button")).toBeNull();

        fireEvent.click(screen.getByRole("button", { name: "912 收藏" }));
        expect(await screen.findByRole("heading", { name: "我的真实收藏内容" })).toBeInTheDocument();
        expect(screen.getByText("这条内容来自我的收藏流接口。")).toBeInTheDocument();
        const favoriteArticle = screen.getByRole("heading", { name: "我的真实收藏内容" }).closest("article");
        expect(favoriteArticle).not.toBeNull();
        const contentFavoriteButton = within(favoriteArticle as HTMLElement).getByRole("button", { name: "收藏" });
        expect(within(contentFavoriteButton).getByText("7")).toBeInTheDocument();
        fireEvent.click(contentFavoriteButton);
        expect(within(contentFavoriteButton).getByText("8")).toBeInTheDocument();

        fireEvent.click(screen.getByRole("button", { name: "18600 粉丝" }));
        expect(await screen.findByText("粉丝用户")).toBeInTheDocument();
        expect(screen.getByText("关注我的用户")).toBeInTheDocument();

        expect(fetchMock).toHaveBeenCalledWith("/v1/feed/user/favorite", expect.objectContaining({
            method: "POST",
            headers: expect.objectContaining({
                Authorization: "Bearer me-token"
            }),
            body: JSON.stringify({ user_id: "7", cursor: "", page_size: 20 })
        }));
        expect(fetchMock).toHaveBeenCalledWith("/v1/user/followers", expect.objectContaining({
            method: "POST",
            headers: expect.objectContaining({
                Authorization: "Bearer me-token"
            }),
            body: JSON.stringify({ user_id: "7", cursor: 0, page_size: 20 })
        }));
        expect(fetchMock).not.toHaveBeenCalledWith("/v1/user/followings", expect.any(Object));
    });

    it("shows an auth-required state on me when signed out", async () => {
        window.history.pushState({}, "", "/me");

        render(<App />);

        const authState = await screen.findByText("登录后才能查看我的主页。");
        expect(authState.closest("[data-page-state]")).toHaveAttribute("data-page-state", "auth-required");
        expect(screen.getByRole("link", { name: "去登录" })).toHaveAttribute("href", "/login?next=%2Fme");
        expect(screen.queryByText("编辑资料")).not.toBeInTheDocument();
    });

    it("shows an error state when the me profile cannot be loaded", async () => {
        window.localStorage.setItem("zfeed.auth.session", JSON.stringify({
            token: "me-token",
            expiredAt: Math.floor(Date.now() / 1000) + 3600,
            user: { userId: 7 }
        }));
        vi.stubGlobal("fetch", vi.fn(async () => jsonResponse({ message: "raw user failure" }, { status: 500 })));
        window.history.pushState({}, "", "/me");

        render(<App />);

        const errorState = await screen.findByText("我的主页加载失败");
        expect(errorState.closest("[data-page-state]")).toHaveAttribute("data-page-state", "error");
        expect(screen.queryByText("raw user failure")).not.toBeInTheDocument();
        expect(screen.queryByText("Mira Chen")).not.toBeInTheDocument();
    });

    it("returns the me profile to auth-required when the token is rejected", async () => {
        window.localStorage.setItem("zfeed.auth.session", JSON.stringify({
            token: "me-token",
            expiredAt: Math.floor(Date.now() / 1000) + 3600,
            user: { userId: 7 }
        }));
        vi.stubGlobal("fetch", vi.fn(async () => jsonResponse({}, { status: 401 })));
        window.history.pushState({}, "", "/me");

        render(<App />);

        const authState = await screen.findByText("登录后才能查看我的主页。");
        expect(authState.closest("[data-page-state]")).toHaveAttribute("data-page-state", "auth-required");
        expect(window.localStorage.getItem("zfeed.auth.session")).toBeNull();
    });

    it("points the profile edit action at the modern edit profile route", async () => {
        window.localStorage.setItem("zfeed.auth.session", JSON.stringify({
            token: "me-token",
            expiredAt: Math.floor(Date.now() / 1000) + 3600,
            user: { userId: 7 }
        }));
        vi.stubGlobal("fetch", vi.fn(async (input: RequestInfo | URL) => {
            if (input === "/v1/users/me") {
                return jsonResponse(meProfilePayload());
            }

            if (input === "/v1/feed/user/publish") {
                return jsonResponse(userPublishedFeedPayload([]));
            }

            return jsonResponse({});
        }));
        window.history.pushState({}, "", "/me");

        render(<App />);

        const editProfileLink = await screen.findByRole("link", { name: "编辑资料" });

        expect(editProfileLink).toHaveAttribute("href", "/me/edit");
    });

    it("renders user, content, search, compose, settings, and edit routes", async () => {
        vi.stubGlobal("fetch", vi.fn(async (input: RequestInfo | URL) => {
            if (input === "/v1/user/profile/jax") {
                return jsonResponse(userProfilePayload());
            }

            return jsonResponse({});
        }));
        window.history.pushState({}, "", "/user/jax");
        const { unmount } = render(<App />);
        expect(await screen.findByRole("heading", { name: "Jax Lee" })).toBeInTheDocument();
        unmount();

        window.history.pushState({}, "", "/user/unknown");
        vi.stubGlobal("fetch", vi.fn(async () => jsonResponse({}, { status: 404 })));
        render(<App />);
        const unknownUserState = await screen.findByText("用户不存在");
        expect(unknownUserState.closest("[data-page-state]")).toHaveAttribute("data-page-state", "error");
        expect(screen.queryByText("编辑资料")).not.toBeInTheDocument();
        unmount();

        window.history.pushState({}, "", "/content/1001");
        vi.stubGlobal("fetch", vi.fn(async (input: RequestInfo | URL) => {
            if (isContentDetailRequest(input)) {
                return jsonResponse(contentDetailPayload({
                    content_id: "1001",
                    author_id: "1001",
                    author_name: "测试作者",
                    author_avatar: "https://example.com/author.png",
                    title: "真实后端标题",
                    description: "真实后端摘要",
                    article_content: "真实后端正文"
                }));
            }

            if (isCommentListRequest(input)) {
                return jsonResponse(commentListPayload([]));
            }

            return jsonResponse({});
        }));
        render(<App />);
        expect(await screen.findByRole("heading", { name: "真实后端标题" })).toBeInTheDocument();
        unmount();

        window.localStorage.setItem("zfeed.auth.session", JSON.stringify({
            token: "edit-token",
            expiredAt: Math.floor(Date.now() / 1000) + 3600,
            user: { userId: 7 }
        }));
        vi.stubGlobal("fetch", vi.fn(async () => jsonResponse(contentDetailPayload())));
        window.history.pushState({}, "", "/content/1001/edit");
        render(<App />);
        expect(await screen.findByRole("heading", { name: "编辑内容" })).toBeInTheDocument();
        unmount();
        window.localStorage.clear();

        window.history.pushState({}, "", "/search");
        render(<App />);
        expect(await screen.findByRole("heading", { name: "搜索" })).toBeInTheDocument();
        expect(screen.getByText("输入关键词开始搜索")).toBeInTheDocument();
        unmount();

        window.history.pushState({}, "", "/compose");
        render(<App />);
        expect(await screen.findByRole("heading", { name: "发布" })).toBeInTheDocument();
        unmount();

        window.history.pushState({}, "", "/settings");
        render(<App />);
        expect(await screen.findByRole("heading", { name: "设置" })).toBeInTheDocument();
        unmount();

        window.history.pushState({}, "", "/me/edit");
        render(<App />);
        expect(await screen.findByRole("heading", { name: "编辑资料" })).toBeInTheDocument();
    });

    it("loads a user profile from the API without executing user content", async () => {
        window.localStorage.setItem("zfeed.auth.session", JSON.stringify({
            token: "reader-token",
            expiredAt: Math.floor(Date.now() / 1000) + 3600,
            user: { userId: 7 }
        }));
        const fetchMock = vi.fn(async () => jsonResponse(userProfilePayload({
            user_info: {
                user_id: 1001,
                nickname: "接口作者 <script>alert(1)</script>",
                avatar: "https://example.com/author.png",
                bio: "接口返回的作者简介"
            },
            followee_count: 5,
            follower_count: 21,
            like_received_count: 34,
            favorite_received_count: 8,
            content_count: 13,
            is_following: true
        })));
        vi.stubGlobal("fetch", fetchMock);
        window.history.pushState({}, "", "/user/jax");

        render(<App />);

        expect(await screen.findByRole("heading", { name: "接口作者 <script>alert(1)</script>" })).toBeInTheDocument();
        expect(screen.getByText("接口返回的作者简介")).toBeInTheDocument();
        expect(screen.getAllByText("13").length).toBeGreaterThan(0);
        expect(screen.getAllByText("5").length).toBeGreaterThan(0);
        expect(screen.getAllByText("21").length).toBeGreaterThan(0);
        expect(screen.getByRole("button", { name: "已关注" })).toHaveAttribute("data-user-id", "1001");
        expect(document.querySelector("script")).toBeNull();
        expect(fetchMock).toHaveBeenCalledWith("/v1/user/profile/jax", expect.objectContaining({
            method: "GET",
            headers: expect.objectContaining({
                Authorization: "Bearer reader-token"
            })
        }));
    });

    it("keeps supporting the legacy user_info envelope on the profile page", async () => {
        window.localStorage.setItem("zfeed.auth.session", JSON.stringify({
            token: "reader-token",
            expiredAt: Math.floor(Date.now() / 1000) + 3600,
            user: { userId: 7 }
        }));
        vi.stubGlobal("fetch", vi.fn(async () => jsonResponse(userProfilePayload({
            user_info: {
                user_id: 1002,
                nickname: "旧接口作者",
                avatar: "https://example.com/legacy.png",
                bio: "旧接口仍可读",
                mobile: "13800000077"
            },
            followee_count: 2,
            follower_count: 4,
            like_received_count: 6,
            favorite_received_count: 8,
            content_count: 10,
            is_following: true
        }))));
        window.history.pushState({}, "", "/user/legacy");

        render(<App />);

        expect(await screen.findByRole("heading", { name: "旧接口作者" })).toBeInTheDocument();
        expect(screen.getByText("旧接口仍可读")).toBeInTheDocument();
        expect(screen.getAllByText("10").length).toBeGreaterThan(0);
        expect(screen.getAllByText("2").length).toBeGreaterThan(0);
        expect(screen.getAllByText("4").length).toBeGreaterThan(0);
        expect(screen.getAllByText("6").length).toBeGreaterThan(0);
        expect(screen.getAllByText("8").length).toBeGreaterThan(0);
        expect(screen.getByRole("button", { name: "已关注" })).toHaveAttribute("data-user-id", "1002");
    });

    it("maps the backend user profile envelope into the profile page", async () => {
        window.localStorage.setItem("zfeed.auth.session", JSON.stringify({
            token: "reader-token",
            expiredAt: Math.floor(Date.now() / 1000) + 3600,
            user: { userId: 7 }
        }));
        const fetchMock = vi.fn(async (input: RequestInfo | URL) => {
            if (input === "/v1/user/profile/2001") {
                return jsonResponse(backendUserProfilePayload());
            }

            return jsonResponse(userPublishedFeedPayload([]));
        });
        vi.stubGlobal("fetch", fetchMock);
        window.history.pushState({}, "", "/user/2001");

        render(<App />);

        expect(await screen.findByRole("heading", { name: "后端作者" })).toBeInTheDocument();
        expect(screen.getByText("后端真实用户简介")).toBeInTheDocument();
        expect(screen.getAllByText("13").length).toBeGreaterThan(0);
        expect(screen.getAllByText("6").length).toBeGreaterThan(0);
        expect(screen.getAllByText("21").length).toBeGreaterThan(0);
        expect(screen.getAllByText("34").length).toBeGreaterThan(0);
        expect(screen.getAllByText("8").length).toBeGreaterThan(0);
        expect(screen.getByRole("button", { name: "已关注" })).toHaveAttribute("data-user-id", "2001");
        expect(fetchMock).toHaveBeenCalledWith("/v1/feed/user/publish", expect.objectContaining({
            body: JSON.stringify({ user_id: "2001", cursor: "", page_size: 20 })
        }));
    });

    it("loads a user published feed after the user profile", async () => {
        window.localStorage.setItem("zfeed.auth.session", JSON.stringify({
            token: "reader-token",
            expiredAt: Math.floor(Date.now() / 1000) + 3600,
            user: { userId: 7 }
        }));
        const fetchMock = vi.fn(async (input: RequestInfo | URL) => {
            if (input === "/v1/user/profile/jax") {
                return jsonResponse(userProfilePayload());
            }

            return jsonResponse(userPublishedFeedPayload());
        });
        vi.stubGlobal("fetch", fetchMock);
        window.history.pushState({}, "", "/user/jax");

        render(<App />);

        expect(await screen.findByRole("heading", { name: "用户主页真实发布内容" })).toBeInTheDocument();
        expect(screen.getByText("这条内容来自用户发布流接口。")).toBeInTheDocument();
        expect(fetchMock).toHaveBeenCalledWith("/v1/feed/user/publish", expect.objectContaining({
            method: "POST",
            headers: expect.objectContaining({
                Authorization: "Bearer reader-token"
            }),
            body: JSON.stringify({ user_id: "1001", cursor: "", page_size: 20 })
        }));
    });

    it("shows an empty state when a user has no published content", async () => {
        vi.stubGlobal("fetch", vi.fn(async (input: RequestInfo | URL) => {
            if (input === "/v1/user/profile/jax") {
                return jsonResponse(userProfilePayload());
            }

            return jsonResponse(userPublishedFeedPayload([]));
        }));
        window.history.pushState({}, "", "/user/jax");

        render(<App />);

        expect(await screen.findByText("还没有公开发布内容")).toBeInTheDocument();
    });

    it("shows a contained error state when the user published feed fails", async () => {
        vi.stubGlobal("fetch", vi.fn(async (input: RequestInfo | URL) => {
            if (input === "/v1/user/profile/jax") {
                return jsonResponse(userProfilePayload());
            }

            return jsonResponse({ message: "raw feed failure" }, { status: 500 });
        }));
        window.history.pushState({}, "", "/user/jax");

        render(<App />);

        expect(await screen.findByRole("heading", { name: "Jax Lee" })).toBeInTheDocument();
        const errorState = await screen.findByText("发布内容加载失败");
        expect(errorState.closest("[data-page-state]")).toHaveAttribute("data-page-state", "error");
        expect(screen.queryByText("raw feed failure")).not.toBeInTheDocument();
    });

    it("renders user published content as safe text", async () => {
        vi.stubGlobal("fetch", vi.fn(async (input: RequestInfo | URL) => {
            if (input === "/v1/user/profile/jax") {
                return jsonResponse(userProfilePayload());
            }

            return jsonResponse(userPublishedFeedPayload([
                defaultRecommendFeedItem({
                    content_id: 9102,
                    author_id: 1001,
                    author_name: "Jax Lee",
                    title: "发布 <script>alert(1)</script>",
                    description: "摘要 <img src=x onerror=alert(1)>"
                })
            ]));
        }));
        window.history.pushState({}, "", "/user/jax");

        render(<App />);

        expect(await screen.findByRole("heading", { name: "发布 <script>alert(1)</script>" })).toBeInTheDocument();
        expect(screen.getByText("摘要 <img src=x onerror=alert(1)>")).toBeInTheDocument();
        expect(document.querySelector("script")).toBeNull();
    });

    it("shows an error state when a user profile cannot be loaded", async () => {
        vi.stubGlobal("fetch", vi.fn(async () => jsonResponse({ message: "raw profile failure" }, { status: 500 })));
        window.history.pushState({}, "", "/user/jax");

        render(<App />);

        const errorState = await screen.findByText("用户主页加载失败");
        expect(errorState.closest("[data-page-state]")).toHaveAttribute("data-page-state", "error");
        expect(screen.queryByText("raw profile failure")).not.toBeInTheDocument();
        expect(screen.queryByText("Jax Lee")).not.toBeInTheDocument();
    });

    it("loads numeric content detail from the API without executing user content", async () => {
        const fetchMock = vi.fn(async () => jsonResponse({
            detail: {
                content_id: "1001",
                content_type: 1,
                author_id: "1002",
                author_name: "Nora <admin>",
                author_avatar: "https://example.com/nora.png",
                title: "真实详情 <script>alert(1)</script>",
                description: "接口返回的正文摘要",
                cover_url: "https://example.com/cover.png",
                article_content: "第一段真实正文。\n第二段保持安全文本。",
                video_url: "",
                video_duration: 0,
                published_at: 1765670400,
                like_count: 12,
                favorite_count: 5,
                comment_count: 3,
                is_liked: false,
                is_favorited: true,
                is_following_author: false
            }
        }));
        vi.stubGlobal("fetch", fetchMock);
        window.history.pushState({}, "", "/content/1001");

        render(<App />);

        expect(await screen.findByText("真实详情 <script>alert(1)</script>")).toBeInTheDocument();
        expect(screen.getByText("Nora <admin>")).toBeInTheDocument();
        expect(screen.getByText("第一段真实正文。")).toBeInTheDocument();
        expect(screen.getByText("第二段保持安全文本。")).toBeInTheDocument();
        expect(document.querySelector("script")).toBeNull();
        expect(fetchMock).toHaveBeenCalledWith("/v1/content/detail", expect.objectContaining({
            method: "POST",
            body: JSON.stringify({ content_id: "1001" })
        }));
    });

    it("keeps the feed side rails visible on content detail", async () => {
        const fetchMock = vi.fn(async (input: RequestInfo | URL) => {
            if (isContentDetailRequest(input)) {
                return jsonResponse(contentDetailPayload({
                    content_id: "1001",
                    title: "三栏详情内容"
                }));
            }

            if (isCommentListRequest(input)) {
                return jsonResponse(commentListPayload([]));
            }

            return jsonResponse({});
        });
        vi.stubGlobal("fetch", fetchMock);
        window.history.pushState({}, "", "/content/1001");

        render(<App />);

        expect(await screen.findByRole("heading", { name: "三栏详情内容" })).toBeInTheDocument();
        expect(screen.getByText("频道").closest("aside")).toHaveClass("lg:col-span-2");
        expect(screen.getByRole("heading", { name: "今日数据" }).closest("aside")).toHaveClass("lg:col-span-3");
    });

    it("uses loaded comments to keep refreshed detail comment counts in sync", async () => {
        const fetchMock = vi.fn(async (input: RequestInfo | URL) => {
            if (isContentDetailRequest(input)) {
                return jsonResponse(contentDetailPayload({
                    content_id: "1001",
                    title: "评论数补齐内容",
                    comment_count: 0
                }));
            }

            if (isCommentListRequest(input)) {
                return jsonResponse(commentListPayload([
                    defaultCommentItem({
                        comment_id: "5001",
                        nickname: "测试19",
                        comment: "你好啊"
                    })
                ]));
            }

            return jsonResponse({});
        });
        vi.stubGlobal("fetch", fetchMock);
        window.history.pushState({}, "", "/content/1001");

        render(<App />);

        expect(await screen.findByRole("heading", { name: "评论数补齐内容" })).toBeInTheDocument();
        expect(await screen.findByText("你好啊")).toBeInTheDocument();
        expect(screen.getByText("1 条讨论")).toBeInTheDocument();
        const detailArticle = screen.getByRole("heading", { name: "评论数补齐内容" }).closest("article");
        expect(detailArticle).not.toBeNull();
        expect(within(detailArticle as HTMLElement).getByText("1")).toBeInTheDocument();
    });

    it("shows an error state when numeric content detail cannot be loaded", async () => {
        vi.stubGlobal("fetch", vi.fn(async () => jsonResponse({}, { status: 500 })));
        window.history.pushState({}, "", "/content/1001");

        render(<App />);

        const errorState = await screen.findByText("内容加载失败");
        expect(errorState.closest("[data-page-state]")).toHaveAttribute("data-page-state", "error");
        expect(screen.queryByText("用 AI 构建产品：30 天从 0 到 1")).not.toBeInTheDocument();
    });

    it("shows an auth-required state on edit content when signed out", async () => {
        window.history.pushState({}, "", "/content/1001/edit");

        render(<App />);

        const authState = await screen.findByText("登录后才能编辑内容。");
        expect(authState.closest("[data-page-state]")).toHaveAttribute("data-page-state", "auth-required");
        expect(screen.getByRole("link", { name: "去登录" })).toHaveAttribute("href", "/login?next=%2Fcontent%2F1001%2Fedit");
        expect(screen.queryByLabelText("标题")).not.toBeInTheDocument();
    });

    it("loads editable article content from the API without executing user content", async () => {
        window.localStorage.setItem("zfeed.auth.session", JSON.stringify({
            token: "edit-token",
            expiredAt: Math.floor(Date.now() / 1000) + 3600,
            user: { userId: 7 }
        }));
        const fetchMock = vi.fn(async () => jsonResponse(contentDetailPayload({
            title: "可编辑标题 <script>alert(1)</script>",
            description: "可编辑摘要",
            article_content: "可编辑正文 <img src=x onerror=alert(1)>"
        })));
        vi.stubGlobal("fetch", fetchMock);
        window.history.pushState({}, "", "/content/1001/edit");

        render(<App />);

        expect(await screen.findByRole("heading", { name: "编辑内容" })).toBeInTheDocument();
        expect(screen.getByLabelText("标题")).toHaveValue("可编辑标题 <script>alert(1)</script>");
        expect(screen.getByLabelText("摘要")).toHaveValue("可编辑摘要");
        expect(screen.getByLabelText("正文")).toHaveValue("可编辑正文 <img src=x onerror=alert(1)>");
        expect(document.querySelector("script")).toBeNull();
        expect(fetchMock).toHaveBeenCalledWith("/v1/content/detail", expect.objectContaining({
            method: "POST",
            headers: expect.objectContaining({
                Authorization: "Bearer edit-token"
            }),
            body: JSON.stringify({ content_id: "1001" })
        }));
    });

    it("submits edited article content with Bearer auth and returns to detail", async () => {
        window.localStorage.setItem("zfeed.auth.session", JSON.stringify({
            token: "edit-token",
            expiredAt: Math.floor(Date.now() / 1000) + 3600,
            user: { userId: 7 }
        }));
        const fetchMock = vi.fn(async (input: RequestInfo | URL) => {
            if (input === "/v1/content/detail") {
                return jsonResponse(contentDetailPayload());
            }

            return jsonResponse({ content_id: 1001 });
        });
        vi.stubGlobal("fetch", fetchMock);
        window.history.pushState({}, "", "/content/1001/edit");

        render(<App />);

        fireEvent.change(await screen.findByLabelText("标题"), { target: { value: "更新后的文章标题" } });
        fireEvent.change(screen.getByLabelText("摘要"), { target: { value: "更新后的文章摘要" } });
        fireEvent.change(screen.getByLabelText("正文"), { target: { value: "更新后的文章正文" } });
        fireEvent.click(screen.getByRole("button", { name: "保存修改" }));

        expect(await screen.findByRole("button", { name: "保存中" })).toBeDisabled();
        await waitFor(() => expect(fetchMock).toHaveBeenCalledWith("/v1/content/article/1001", expect.objectContaining({
            method: "PUT",
            headers: expect.objectContaining({
                Authorization: "Bearer edit-token"
            }),
            body: JSON.stringify({
                title: "更新后的文章标题",
                description: "更新后的文章摘要",
                cover: "https://example.com/cover.png",
                content: "更新后的文章正文"
            })
        })));
        await waitFor(() => expect(window.location.pathname).toBe("/content/1001"));
    });

    it("shows a safe error when editable content cannot be loaded", async () => {
        window.localStorage.setItem("zfeed.auth.session", JSON.stringify({
            token: "edit-token",
            expiredAt: Math.floor(Date.now() / 1000) + 3600,
            user: { userId: 7 }
        }));
        vi.stubGlobal("fetch", vi.fn(async () => jsonResponse({ message: "raw edit failure" }, { status: 500 })));
        window.history.pushState({}, "", "/content/1001/edit");

        render(<App />);

        const errorState = await screen.findByText("编辑内容加载失败");
        expect(errorState.closest("[data-page-state]")).toHaveAttribute("data-page-state", "error");
        expect(screen.queryByText("raw edit failure")).not.toBeInTheDocument();
        expect(screen.queryByLabelText("标题")).not.toBeInTheDocument();
    });

    it("shows an auth-required state on edit profile when signed out", async () => {
        window.history.pushState({}, "", "/me/edit");

        render(<App />);

        const authState = await screen.findByText("登录后才能编辑资料。");
        expect(authState.closest("[data-page-state]")).toHaveAttribute("data-page-state", "auth-required");
        expect(screen.getByRole("link", { name: "去登录" })).toHaveAttribute("href", "/login?next=%2Fme%2Fedit");
        expect(screen.queryByLabelText("昵称")).not.toBeInTheDocument();
    });

    it("loads edit profile fields and recommendations from backend data", async () => {
        window.localStorage.setItem("zfeed.auth.session", JSON.stringify({
            token: "profile-token",
            expiredAt: Math.floor(Date.now() / 1000) + 3600,
            user: { userId: 20 }
        }));
        const fetchMock = vi.fn(async (input: RequestInfo | URL) => {
            if (input === "/v1/users/me") {
                return jsonResponse(meProfilePayload({
                    user_info: {
                        user_id: 20,
                        mobile: "13800000019",
                        nickname: "测试19",
                        avatar: "19",
                        bio: "测试内容19",
                        email: "test19@zfeed.local",
                        gender: 0,
                        birthday: 0
                    }
                }));
            }

            if (input === "/v1/search/users") {
                return jsonResponse({
                    items: [
                        { user_id: 1, nickname: "测试1", avatar: "1", bio: "测试内容1", is_following: false },
                        { user_id: 2, nickname: "测试2", avatar: "2", bio: "测试内容2", is_following: true },
                        { user_id: 20, nickname: "测试19", avatar: "19", bio: "测试内容19", is_following: false }
                    ],
                    next_cursor: 0,
                    has_more: false
                });
            }

            return jsonResponse({});
        });
        vi.stubGlobal("fetch", fetchMock);
        window.history.pushState({}, "", "/me/edit");

        render(<App />);

        expect(await screen.findByDisplayValue("测试19")).toBeInTheDocument();
        expect(screen.getByDisplayValue("13800000019")).toBeInTheDocument();
        expect(screen.getByDisplayValue("19")).toBeInTheDocument();
        expect(screen.getByDisplayValue("测试内容19")).toBeInTheDocument();
        expect(screen.getByText("测试1")).toBeInTheDocument();
        expect(screen.getByText("测试内容1")).toBeInTheDocument();
        expect(screen.getAllByText("19").length).toBeGreaterThan(0);
        expect(document.querySelector('img[src="19"]')).toBeNull();
        expect(screen.queryByDisplayValue("Mira Chen")).not.toBeInTheDocument();
        expect(screen.queryByText("Zhang Xiaolong")).not.toBeInTheDocument();
        expect(screen.queryByText("AI 前线")).not.toBeInTheDocument();
        expect(fetchMock).toHaveBeenCalledWith("/v1/search/users", expect.objectContaining({
            method: "POST",
            body: JSON.stringify({ query: "测试", page_size: 4, mode: "relevance" })
        }));
    });

    it("renders the login route with the required fields", async () => {
        window.history.pushState({}, "", "/login");

        render(<App />);

        expect(await screen.findByRole("heading", { name: "登录 zfeed" })).toBeInTheDocument();
        expect(screen.getByTitle("zfeed 首页雾化背景")).toBeInTheDocument();
        expect(screen.getByLabelText("手机号")).toBeInTheDocument();
        expect(screen.getByLabelText("密码")).toBeInTheDocument();
        expect(screen.getByRole("link", { name: "去注册" })).toHaveAttribute("href", "/register");
    });

    it("validates login fields before submitting", async () => {
        const fetchMock = vi.fn();
        vi.stubGlobal("fetch", fetchMock);
        window.history.pushState({}, "", "/login");

        render(<App />);

        fireEvent.click(await screen.findByRole("button", { name: "登录" }));

        expect(await screen.findByText("请输入手机号")).toBeInTheDocument();
        expect(screen.getByText("请输入密码")).toBeInTheDocument();
        expect(fetchMock).not.toHaveBeenCalled();
    });

    it("validates login phone format and password length before submitting", async () => {
        const fetchMock = vi.fn();
        vi.stubGlobal("fetch", fetchMock);
        window.history.pushState({}, "", "/login");

        render(<App />);

        fireEvent.change(await screen.findByLabelText("手机号"), { target: { value: "12345" } });
        fireEvent.change(screen.getByLabelText("密码"), { target: { value: "12345" } });
        fireEvent.click(screen.getByRole("button", { name: "登录" }));

        expect(await screen.findByText("请输入有效手机号")).toBeInTheDocument();
        expect(screen.getByText("密码至少 6 位")).toBeInTheDocument();
        expect(fetchMock).not.toHaveBeenCalled();
    });

    it("accepts backend-compatible login mobile formats", async () => {
        const fetchMock = vi.fn(async () => jsonResponse({
            user_id: 8,
            token: "login-token",
            expired_at: Math.floor(Date.now() / 1000) + 3600,
            nickname: "Obs Author",
            avatar: ""
        }));
        vi.stubGlobal("fetch", fetchMock);
        window.history.pushState({}, "", "/login");

        render(<App />);

        fireEvent.change(await screen.findByLabelText("手机号"), { target: { value: "+8611775980087" } });
        fireEvent.change(screen.getByLabelText("密码"), { target: { value: "123456Aa!" } });
        fireEvent.click(screen.getByRole("button", { name: "登录" }));

        await waitFor(() => expect(fetchMock).toHaveBeenCalledWith("/v1/login", expect.objectContaining({
            method: "POST",
            body: JSON.stringify({ mobile: "+8611775980087", password: "123456Aa!" })
        })));
        expect(screen.queryByText("请输入有效手机号")).not.toBeInTheDocument();
    });

    it("submits login, stores the session, and enters the home feed", async () => {
        const fetchMock = vi.fn(async () => jsonResponse({
            user_id: 7,
            token: "login-token",
            expired_at: Math.floor(Date.now() / 1000) + 3600,
            nickname: "Mira Chen",
            avatar: "https://example.com/avatar.png"
        }));
        vi.stubGlobal("fetch", fetchMock);
        window.history.pushState({}, "", "/login");

        render(<App />);

        fireEvent.change(await screen.findByLabelText("手机号"), { target: { value: "13800138000" } });
        fireEvent.change(screen.getByLabelText("密码"), { target: { value: "password123" } });
        fireEvent.click(screen.getByRole("button", { name: "登录" }));

        expect(await screen.findByRole("button", { name: "登录中" })).toBeDisabled();
        await waitFor(() => expect(window.location.pathname).toBe("/home"));
        expect(fetchMock).toHaveBeenCalledWith("/v1/login", expect.objectContaining({
            method: "POST",
            body: JSON.stringify({ mobile: "13800138000", password: "password123" })
        }));
        expect(JSON.parse(window.localStorage.getItem("zfeed.auth.session") ?? "{}")).toMatchObject({
            token: "login-token",
            user: { userId: 7, nickname: "Mira Chen" }
        });
    });

    it("prevents duplicate login submissions while the request is pending", async () => {
        let resolveLogin!: (response: Response) => void;
        const fetchMock = vi.fn(() => new Promise<Response>((resolve) => {
            resolveLogin = resolve;
        }));
        vi.stubGlobal("fetch", fetchMock);
        window.history.pushState({}, "", "/login");

        render(<App />);

        fireEvent.change(await screen.findByLabelText("手机号"), { target: { value: "13800138000" } });
        fireEvent.change(screen.getByLabelText("密码"), { target: { value: "password123" } });
        const form = screen.getByRole("button", { name: "登录" }).closest("form");
        expect(form).not.toBeNull();

        fireEvent.submit(form!);
        expect(await screen.findByRole("button", { name: "登录中" })).toBeDisabled();
        fireEvent.submit(form!);

        expect(fetchMock).toHaveBeenCalledTimes(1);
        resolveLogin(jsonResponse({
            user_id: 7,
            token: "login-token",
            expired_at: Math.floor(Date.now() / 1000) + 3600,
            nickname: "Mira Chen",
            avatar: "https://example.com/avatar.png"
        }));
        await waitFor(() => expect(window.location.pathname).toBe("/home"));
    });

    it("returns to a safe next path after login", async () => {
        const fetchMock = vi.fn(async () => jsonResponse({
            user_id: 7,
            token: "login-token",
            expired_at: Math.floor(Date.now() / 1000) + 3600,
            nickname: "Mira Chen",
            avatar: "https://example.com/avatar.png"
        }));
        vi.stubGlobal("fetch", fetchMock);
        window.history.pushState({}, "", "/login?next=%2Fcompose");

        render(<App />);

        fireEvent.change(await screen.findByLabelText("手机号"), { target: { value: "13800138000" } });
        fireEvent.change(screen.getByLabelText("密码"), { target: { value: "password123" } });
        fireEvent.click(screen.getByRole("button", { name: "登录" }));

        await waitFor(() => expect(window.location.pathname).toBe("/compose"));
    });

    it("ignores unsafe external next paths after login", async () => {
        const fetchMock = vi.fn(async () => jsonResponse({
            user_id: 7,
            token: "login-token",
            expired_at: Math.floor(Date.now() / 1000) + 3600,
            nickname: "Mira Chen",
            avatar: "https://example.com/avatar.png"
        }));
        vi.stubGlobal("fetch", fetchMock);
        window.history.pushState({}, "", "/login?next=https%3A%2F%2Fevil.example%2Fsteal");

        render(<App />);

        fireEvent.change(await screen.findByLabelText("手机号"), { target: { value: "13800138000" } });
        fireEvent.change(screen.getByLabelText("密码"), { target: { value: "password123" } });
        fireEvent.click(screen.getByRole("button", { name: "登录" }));

        await waitFor(() => expect(window.location.pathname).toBe("/home"));
    });

    it("ignores removed auth next paths after login", async () => {
        const fetchMock = vi.fn(async () => jsonResponse({
            user_id: 7,
            token: "login-token",
            expired_at: Math.floor(Date.now() / 1000) + 3600,
            nickname: "Mira Chen",
            avatar: "https://example.com/avatar.png"
        }));
        vi.stubGlobal("fetch", fetchMock);
        window.history.pushState({}, "", "/login?next=%2Ffollowing.html");

        render(<App />);

        expect(await screen.findByRole("heading", { name: "登录 zfeed" })).toBeInTheDocument();
        fireEvent.change(screen.getByLabelText("手机号"), { target: { value: "13800138000" } });
        fireEvent.change(screen.getByLabelText("密码"), { target: { value: "password123" } });
        fireEvent.click(screen.getByRole("button", { name: "登录" }));

        await waitFor(() => expect(window.location.pathname).toBe("/home"));
    });

    it("shows a sanitized login error without storing a session", async () => {
        vi.stubGlobal("fetch", vi.fn(async () => jsonResponse({ message: "raw backend failure" }, { status: 401 })));
        window.history.pushState({}, "", "/login");

        render(<App />);

        fireEvent.change(await screen.findByLabelText("手机号"), { target: { value: "13800138000" } });
        fireEvent.change(screen.getByLabelText("密码"), { target: { value: "bad-password" } });
        fireEvent.click(screen.getByRole("button", { name: "登录" }));

        expect(await screen.findByText("手机号或密码不正确")).toBeInTheDocument();
        expect(screen.queryByText("raw backend failure")).not.toBeInTheDocument();
        expect(window.localStorage.getItem("zfeed.auth.session")).toBeNull();
    });

    it("renders the register route with profile fields", async () => {
        window.history.pushState({}, "", "/register");

        render(<App />);

        expect(await screen.findByRole("heading", { name: "创建 zfeed 账号" })).toBeInTheDocument();
        expect(screen.getByTitle("zfeed 首页雾化背景")).toBeInTheDocument();
        expect(screen.getByLabelText("手机号")).toBeInTheDocument();
        expect(screen.getByLabelText("昵称")).toBeInTheDocument();
        expect(screen.getByRole("link", { name: "去登录" })).toHaveAttribute("href", "/login");
    });

    it("keeps auth next when switching between login and register", async () => {
        window.history.pushState({}, "", "/login?next=%2Fcompose");

        const { unmount } = render(<App />);

        expect(await screen.findByRole("heading", { name: "登录 zfeed" })).toBeInTheDocument();
        expect(screen.getByRole("link", { name: "注册" })).toHaveAttribute("href", "/register?next=%2Fcompose");
        expect(screen.getByRole("link", { name: "去注册" })).toHaveAttribute("href", "/register?next=%2Fcompose");
        unmount();

        window.history.pushState({}, "", "/register?next=%2Fcompose");
        render(<App />);

        expect(await screen.findByRole("heading", { name: "创建 zfeed 账号" })).toBeInTheDocument();
        expect(screen.getByRole("link", { name: "登录" })).toHaveAttribute("href", "/login?next=%2Fcompose");
        expect(screen.getByRole("link", { name: "去登录" })).toHaveAttribute("href", "/login?next=%2Fcompose");
    });

    it("validates register fields before submitting", async () => {
        const fetchMock = vi.fn();
        vi.stubGlobal("fetch", fetchMock);
        window.history.pushState({}, "", "/register");

        render(<App />);

        fireEvent.change(await screen.findByLabelText("昵称"), { target: { value: "a".repeat(65) } });
        fireEvent.change(screen.getByLabelText("头像 URL"), { target: { value: "javascript:alert(1)" } });
        fireEvent.change(screen.getByLabelText("邮箱"), { target: { value: "bad-email" } });
        fireEvent.change(screen.getByLabelText("简介"), { target: { value: "b".repeat(256) } });
        fireEvent.click(screen.getByRole("button", { name: "注册" }));

        expect(await screen.findByText("请输入手机号")).toBeInTheDocument();
        expect(screen.getByText("请输入密码")).toBeInTheDocument();
        expect(screen.getByText("昵称最多 64 字")).toBeInTheDocument();
        expect(screen.getByText("请输入有效头像链接")).toBeInTheDocument();
        expect(screen.getByText("请输入有效邮箱")).toBeInTheDocument();
        expect(screen.getByText("简介最多 255 字")).toBeInTheDocument();
        expect(fetchMock).not.toHaveBeenCalled();
    });

    it("validates register phone format and password length before submitting", async () => {
        const fetchMock = vi.fn();
        vi.stubGlobal("fetch", fetchMock);
        window.history.pushState({}, "", "/register");

        render(<App />);

        fireEvent.change(await screen.findByLabelText("手机号"), { target: { value: "12345" } });
        fireEvent.change(screen.getByLabelText("密码"), { target: { value: "12345" } });
        fireEvent.click(screen.getByRole("button", { name: "注册" }));

        expect(await screen.findByText("请输入有效手机号")).toBeInTheDocument();
        expect(screen.getByText("密码至少 6 位")).toBeInTheDocument();
        expect(fetchMock).not.toHaveBeenCalled();
    });

    it("accepts backend-compatible register mobile formats", async () => {
        const fetchMock = vi.fn(async () => jsonResponse({
            user_id: 10,
            token: "register-token",
            expired_at: Math.floor(Date.now() / 1000) + 3600
        }));
        vi.stubGlobal("fetch", fetchMock);
        window.history.pushState({}, "", "/register");

        render(<App />);

        fireEvent.change(await screen.findByLabelText("手机号"), { target: { value: "+8611775980088" } });
        fireEvent.change(screen.getByLabelText("密码"), { target: { value: "123456Aa!" } });
        fireEvent.click(screen.getByRole("button", { name: "注册" }));

        await waitFor(() => expect(fetchMock).toHaveBeenCalledWith("/v1/users", expect.objectContaining({
            method: "POST",
            body: JSON.stringify({
                mobile: "+8611775980088",
                password: "123456Aa!"
            })
        })));
        expect(screen.queryByText("请输入有效手机号")).not.toBeInTheDocument();
    });

    it("submits register, stores the session, and enters me", async () => {
        const fetchMock = vi.fn(async () => jsonResponse({
            user_id: 9,
            token: "register-token",
            expired_at: Math.floor(Date.now() / 1000) + 3600
        }));
        vi.stubGlobal("fetch", fetchMock);
        window.history.pushState({}, "", "/register");

        render(<App />);

        fireEvent.change(await screen.findByLabelText("手机号"), { target: { value: "13900139000" } });
        fireEvent.change(screen.getByLabelText("密码"), { target: { value: "password123" } });
        fireEvent.change(screen.getByLabelText("昵称"), { target: { value: "New User" } });
        fireEvent.change(screen.getByLabelText("头像 URL"), { target: { value: "https://example.com/a.png" } });
        fireEvent.change(screen.getByLabelText("邮箱"), { target: { value: "new@example.com" } });
        fireEvent.change(screen.getByLabelText("简介"), { target: { value: "hello" } });
        fireEvent.click(screen.getByRole("button", { name: "注册" }));

        expect(await screen.findByRole("button", { name: "注册中" })).toBeDisabled();
        await waitFor(() => expect(window.location.pathname).toBe("/me"));
        expect(fetchMock).toHaveBeenCalledWith("/v1/users", expect.objectContaining({
            method: "POST",
            body: JSON.stringify({
                mobile: "13900139000",
                password: "password123",
                nickname: "New User",
                avatar: "https://example.com/a.png",
                email: "new@example.com",
                bio: "hello"
            })
        }));
        expect(JSON.parse(window.localStorage.getItem("zfeed.auth.session") ?? "{}")).toMatchObject({
            token: "register-token",
            user: { userId: 9, nickname: "New User" }
        });
    });

    it("returns to auth next after register", async () => {
        const fetchMock = vi.fn(async () => jsonResponse({
            user_id: 9,
            token: "register-token",
            expired_at: Math.floor(Date.now() / 1000) + 3600
        }));
        vi.stubGlobal("fetch", fetchMock);
        window.history.pushState({}, "", "/register?next=%2Fcompose");

        render(<App />);

        expect(await screen.findByRole("heading", { name: "创建 zfeed 账号" })).toBeInTheDocument();
        fireEvent.change(screen.getByLabelText("手机号"), { target: { value: "13900139000" } });
        fireEvent.change(screen.getByLabelText("密码"), { target: { value: "password123" } });
        fireEvent.change(screen.getByLabelText("昵称"), { target: { value: "New User" } });
        fireEvent.click(screen.getByRole("button", { name: "注册" }));

        await waitFor(() => expect(window.location.pathname).toBe("/compose"));
    });

    it("prevents duplicate register submissions while the request is pending", async () => {
        let resolveRegister!: (response: Response) => void;
        const fetchMock = vi.fn(() => new Promise<Response>((resolve) => {
            resolveRegister = resolve;
        }));
        vi.stubGlobal("fetch", fetchMock);
        window.history.pushState({}, "", "/register");

        render(<App />);

        fireEvent.change(await screen.findByLabelText("手机号"), { target: { value: "13900139000" } });
        fireEvent.change(screen.getByLabelText("密码"), { target: { value: "password123" } });
        fireEvent.change(screen.getByLabelText("昵称"), { target: { value: "New User" } });
        const form = screen.getByRole("button", { name: "注册" }).closest("form");
        expect(form).not.toBeNull();

        fireEvent.submit(form!);
        expect(await screen.findByRole("button", { name: "注册中" })).toBeDisabled();
        fireEvent.submit(form!);

        expect(fetchMock).toHaveBeenCalledTimes(1);
        resolveRegister(jsonResponse({
            user_id: 9,
            token: "register-token",
            expired_at: Math.floor(Date.now() / 1000) + 3600
        }));
        await waitFor(() => expect(window.location.pathname).toBe("/me"));
    });

    it("opens compose from the primary publish button", async () => {
        vi.stubGlobal("fetch", vi.fn(async () => jsonResponse(recommendFeedPayload())));
        window.history.pushState({}, "", "/home");

        render(<App />);

        fireEvent.click(await screen.findByRole("button", { name: /发布/ }));

        await waitFor(() => expect(window.location.pathname).toBe("/compose"));
        expect(await screen.findByRole("heading", { name: "发布" })).toBeInTheDocument();
    });

    it("opens compose from the home composer input instead of treating it as search", async () => {
        vi.stubGlobal("fetch", vi.fn(async () => jsonResponse(recommendFeedPayload())));
        window.history.pushState({}, "", "/home");

        render(<App />);

        const composer = await screen.findByPlaceholderText("分享你的想法、灵感或最新动态...");
        fireEvent.change(composer, { target: { value: "准备发布一条动态" } });
        fireEvent.keyDown(composer, { key: "Enter" });

        await waitFor(() => expect(window.location.pathname).toBe("/compose"));
        expect(window.location.search).toBe("");
    });

    it("shows an auth-required state on compose with a next-aware login link", async () => {
        window.history.pushState({}, "", "/compose");

        render(<App />);

        const authState = await screen.findByText("发布需要登录。");
        expect(authState.closest("[data-page-state]")).toHaveAttribute("data-page-state", "auth-required");
        expect(screen.getByRole("link", { name: "去登录" })).toHaveAttribute("href", "/login?next=%2Fcompose");
        expect(screen.queryByPlaceholderText("标题")).not.toBeInTheDocument();
    });

    it("shows an auth-required state on settings when signed out", async () => {
        window.history.pushState({}, "", "/settings");

        render(<App />);

        const authState = await screen.findByText("登录后才能管理设置。");
        expect(authState.closest("[data-page-state]")).toHaveAttribute("data-page-state", "auth-required");
        expect(screen.getByRole("link", { name: "去登录" })).toHaveAttribute("href", "/login?next=%2Fsettings");
        expect(screen.queryByText("账号与安全")).not.toBeInTheDocument();
    });

    it("logs out from settings by clearing the session and returning to login", async () => {
        window.localStorage.setItem("zfeed.auth.session", JSON.stringify({
            token: "settings-token",
            expiredAt: Math.floor(Date.now() / 1000) + 3600,
            user: { userId: 7, nickname: "Mira Chen" }
        }));
        window.history.pushState({}, "", "/settings");

        render(<App />);

        expect(await screen.findByText("账号与安全")).toBeInTheDocument();
        fireEvent.click(screen.getByRole("button", { name: "退出登录" }));

        await waitFor(() => expect(window.location.pathname).toBe("/login"));
        expect(window.localStorage.getItem("zfeed.auth.session")).toBeNull();
        expect(await screen.findByRole("heading", { name: "登录 zfeed" })).toBeInTheDocument();
    });

    it("opens search from the global search box", async () => {
        vi.stubGlobal("fetch", vi.fn(async () => jsonResponse(recommendFeedPayload())));
        window.history.pushState({}, "", "/home");

        render(<App />);

        const searchBox = await screen.findByPlaceholderText("搜索内容、创作者或话题");
        fireEvent.change(searchBox, { target: { value: "AI 创作" } });
        fireEvent.keyDown(searchBox, { key: "Enter" });

        await waitFor(() => expect(window.location.pathname).toBe("/search"));
        expect(window.location.search).toBe("?q=AI+%E5%88%9B%E4%BD%9C");
    });

    it("loads content results from the search query by default", async () => {
        const fetchMock = vi.fn(async (input: RequestInfo | URL) => {
            if (input === "/v1/search/contents") {
                return jsonResponse({
                    items: [{
                        content_id: 1001,
                        content_type: 1,
                        author_id: 1001,
                        author_name: "Jax Lee",
                        author_avatar: "https://example.com/jax.png",
                        title: "AI 搜索结果 <script>alert(1)</script>",
                        cover_url: "https://example.com/cover.png",
                        published_at: 1765670400
                    }],
                    next_cursor: 0,
                    has_more: false,
                    next_page_token: "",
                    snapshot_id: "content-snapshot"
                });
            }

            if (input === "/v1/search/users") {
                return jsonResponse({
                    items: [{
                        user_id: 1002,
                        nickname: "Nora <admin>",
                        avatar: "https://example.com/nora.png",
                        bio: "研究 AI 工具",
                        is_following: false
                    }],
                    next_cursor: 0,
                    has_more: false,
                    next_page_token: "",
                    snapshot_id: "user-snapshot"
                });
            }

            return jsonResponse({});
        });
        vi.stubGlobal("fetch", fetchMock);
        window.history.pushState({}, "", "/search?q=AI+%E5%88%9B%E4%BD%9C");

        render(<App />);

        expect(await screen.findByText("AI 搜索结果 <script>alert(1)</script>")).toBeInTheDocument();
        expect(screen.queryByText("Nora <admin>")).not.toBeInTheDocument();
        expect(screen.queryByText("输入关键词开始搜索")).not.toBeInTheDocument();
        expect(document.querySelector("script")).toBeNull();
        expect(screen.getByRole("button", { name: "内容" })).toHaveAttribute("aria-pressed", "true");
        expect(screen.getByRole("button", { name: "作者" })).toHaveAttribute("aria-pressed", "false");
        expect(fetchMock).toHaveBeenCalledWith("/v1/search/contents", expect.objectContaining({
            method: "POST",
            body: JSON.stringify({ query: "AI 创作", page_size: 10, mode: "latest" })
        }));
        expect(fetchMock).not.toHaveBeenCalledWith("/v1/search/users", expect.anything());
    });

    it("switches the search target between content and authors", async () => {
        const fetchMock = vi.fn(async (input: RequestInfo | URL) => {
            if (input === "/v1/search/contents") {
                return jsonResponse({
                    items: [{
                        content_id: 1001,
                        content_type: 1,
                        author_id: 1001,
                        author_name: "Jax Lee",
                        author_avatar: "",
                        title: "不应该出现的内容结果",
                        cover_url: "",
                        published_at: 1765670400
                    }]
                });
            }

            if (input === "/v1/search/users") {
                return jsonResponse({
                    items: [{
                        user_id: 1002,
                        nickname: "作者搜索结果",
                        avatar: "",
                        bio: "这个结果来自作者搜索。",
                        is_following: false
                    }]
                });
            }

            return jsonResponse({});
        });
        vi.stubGlobal("fetch", fetchMock);
        window.history.pushState({}, "", "/search?q=AI&type=author");

        render(<App />);

        expect(await screen.findByText("作者搜索结果")).toBeInTheDocument();
        expect(screen.queryByText("不应该出现的内容结果")).not.toBeInTheDocument();
        expect(screen.getByRole("button", { name: "内容" })).toHaveAttribute("aria-pressed", "false");
        expect(screen.getByRole("button", { name: "作者" })).toHaveAttribute("aria-pressed", "true");
        expect(fetchMock).not.toHaveBeenCalledWith("/v1/search/contents", expect.anything());
        expect(fetchMock).toHaveBeenCalledWith("/v1/search/users", expect.objectContaining({
            method: "POST",
            body: JSON.stringify({ query: "AI", page_size: 10, mode: "latest" })
        }));
    });

    it("keeps the selected search target when submitting the search box", async () => {
        const fetchMock = vi.fn(async (input: RequestInfo | URL) => {
            if (input === "/v1/search/users") {
                return jsonResponse({ items: [] });
            }

            return jsonResponse({ items: [] });
        });
        vi.stubGlobal("fetch", fetchMock);
        window.history.pushState({}, "", "/search?q=AI&type=author");

        render(<App />);

        const searchBox = await screen.findByPlaceholderText("搜索内容、创作者或话题");
        fireEvent.change(searchBox, { target: { value: "Nora" } });
        fireEvent.keyDown(searchBox, { key: "Enter" });

        await waitFor(() => expect(window.location.pathname).toBe("/search"));
        expect(window.location.search).toBe("?q=Nora&type=author");
    });

    it("loads content search without depending on author search", async () => {
        const fetchMock = vi.fn(async (input: RequestInfo | URL) => {
            if (input === "/v1/search/contents") {
                return jsonResponse({
                    items: [{
                        content_id: 1001,
                        content_type: 1,
                        author_id: 1001,
                        author_name: "Jax Lee",
                        author_avatar: "https://example.com/jax.png",
                        title: "可用的内容搜索结果",
                        description: "用户搜索接口失败时也应该展示内容。",
                        cover_url: "",
                        published_at: 1765670400
                    }]
                });
            }

            if (input === "/v1/search/users") {
                return jsonResponse({ message: "搜索快照未开启" }, { status: 400 });
            }

            return jsonResponse({});
        });
        vi.stubGlobal("fetch", fetchMock);
        window.history.pushState({}, "", "/search?q=AI");

        render(<App />);

        expect(await screen.findByRole("heading", { name: "可用的内容搜索结果" })).toBeInTheDocument();
        expect(document.querySelector('[data-page-state="error"]')).toBeNull();
        expect(screen.queryByText("搜索失败")).not.toBeInTheDocument();
        expect(fetchMock).not.toHaveBeenCalledWith("/v1/search/users", expect.anything());
    });

    it("renders search results as an actionable feed stream", async () => {
        window.localStorage.setItem("zfeed.auth.session", JSON.stringify({
            token: "search-feed-token",
            expiredAt: Math.floor(Date.now() / 1000) + 3600,
            user: { userId: 7 }
        }));
        const fetchMock = vi.fn(async (input: RequestInfo | URL) => {
            if (input === "/v1/search/contents") {
                return jsonResponse({
                    items: [{
                        content_id: 1001,
                        content_type: 1,
                        author_id: 1001,
                        author_name: "Jax Lee",
                        author_avatar: "https://example.com/jax.png",
                        title: "搜索流内容",
                        description: "这条内容来自搜索信息流。",
                        cover_url: "",
                        published_at: 1765670400,
                        like_count: 6,
                        favorite_count: 2,
                        comment_count: 1,
                        is_liked: false,
                        is_favorited: false
                    }]
                });
            }

            if (input === "/v1/search/users") {
                return jsonResponse({ items: [] });
            }

            return jsonResponse({});
        });
        vi.stubGlobal("fetch", fetchMock);
        window.history.pushState({}, "", "/search?q=AI");

        render(<App />);

        const heading = await screen.findByRole("heading", { name: "搜索流内容" });
        const article = heading.closest("article");
        expect(article).not.toBeNull();
        expect(screen.getByText("频道").closest("aside")).toHaveClass("lg:col-span-2");
        expect(screen.getByRole("heading", { name: "今日数据" }).closest("aside")).toHaveClass("lg:col-span-3");
        const leftRail = screen.getByText("频道").closest("aside") as HTMLElement;
        expect(within(leftRail).getByRole("link", { name: /搜索/ })).toHaveClass("text-primary");

        const likeButton = within(article as HTMLElement).getByRole("button", { name: "点赞" });
        fireEvent.click(likeButton);

        expect(likeButton).toHaveAttribute("aria-label", "取消点赞");
        expect(within(likeButton).getByText("7")).toBeInTheDocument();
        await waitFor(() => expect(fetchMock).toHaveBeenCalledWith("/v1/interaction/like", expect.objectContaining({
            method: "POST",
            headers: expect.objectContaining({
                Authorization: "Bearer search-feed-token"
            }),
            body: JSON.stringify({ content_id: "1001", content_user_id: "1001", scene: "ARTICLE" })
        })));
    });

    it("keeps long search result metadata from overflowing cards", async () => {
        const longAuthorName = "VeryLongAuthorNameWithoutSpaces".repeat(4);
        const fetchMock = vi.fn(async (input: RequestInfo | URL) => {
            if (input === "/v1/search/contents") {
                return jsonResponse({
                    items: [{
                        content_id: 1001,
                        content_type: 1,
                        author_id: 1001,
                        author_name: longAuthorName,
                        author_avatar: "",
                        title: "搜索结果标题",
                        cover_url: "",
                        published_at: 1765670400
                    }]
                });
            }

            return jsonResponse({});
        });
        vi.stubGlobal("fetch", fetchMock);
        window.history.pushState({}, "", "/search?q=long");

        render(<App />);

        expect(await screen.findByText(longAuthorName)).toHaveClass("break-words");
    });

    it("validates search query length before requesting results", async () => {
        const fetchMock = vi.fn();
        vi.stubGlobal("fetch", fetchMock);
        window.history.pushState({}, "", `/search?q=${"a".repeat(51)}`);

        render(<App />);

        expect(await screen.findByText("关键词最多 50 字")).toBeInTheDocument();
        expect(fetchMock).not.toHaveBeenCalled();
    });

    it("guides unauthenticated write actions to login without calling the API", async () => {
        const fetchMock = vi.fn(async (input: RequestInfo | URL) => {
            if (isRecommendFeedRequest(input)) {
                return jsonResponse(recommendFeedPayload());
            }

            return jsonResponse({});
        });
        vi.stubGlobal("fetch", fetchMock);
        window.history.pushState({}, "", "/home");

        render(<App />);

        const likeButton = (await screen.findAllByText("favorite"))[0].closest("button");
        expect(likeButton).not.toBeNull();
        fireEvent.click(likeButton!);

        await waitFor(() => expect(window.location.pathname).toBe("/login"));
        expect(fetchMock).toHaveBeenCalledTimes(1);
        expect(fetchMock).toHaveBeenCalledWith("/v1/feed/recommend", expect.any(Object));
    });

    it("likes recommendation content with count and accessible state updates", async () => {
        window.localStorage.setItem("zfeed.auth.session", JSON.stringify({
            token: "like-token",
            expiredAt: Math.floor(Date.now() / 1000) + 3600,
            user: { userId: 7 }
        }));
        const fetchMock = vi.fn(async (input: RequestInfo | URL) => {
            if (isRecommendFeedRequest(input)) {
                return jsonResponse(recommendFeedPayload([
                    defaultRecommendFeedItem({
                        content_id: 3001,
                        author_id: 4001,
                        author_name: "交互作者",
                        title: "可点赞推荐内容",
                        like_count: 6,
                        is_liked: false,
                        is_favorited: false
                    })
                ]));
            }

            return jsonResponse({});
        });
        vi.stubGlobal("fetch", fetchMock);
        window.history.pushState({}, "", "/home");

        render(<App />);

        const heading = await screen.findByRole("heading", { name: "可点赞推荐内容" });
        const article = heading.closest("article");
        expect(article).not.toBeNull();
        const likeButton = within(article as HTMLElement).getByRole("button", { name: "点赞" });

        fireEvent.click(likeButton);

        expect(likeButton).toHaveAttribute("aria-label", "取消点赞");
        expect(within(likeButton).getByText("7")).toBeInTheDocument();
        await waitFor(() => expect(fetchMock).toHaveBeenCalledWith("/v1/interaction/like", expect.objectContaining({
            method: "POST",
            headers: expect.objectContaining({
                Authorization: "Bearer like-token"
            }),
            body: JSON.stringify({ content_id: "3001", content_user_id: "4001", scene: "ARTICLE" })
        })));
    });

    it("keeps liked recommendation count synced when opening stale detail data", async () => {
        window.localStorage.setItem("zfeed.auth.session", JSON.stringify({
            token: "like-sync-token",
            expiredAt: Math.floor(Date.now() / 1000) + 3600,
            user: { userId: 7 }
        }));
        const fetchMock = vi.fn(async (input: RequestInfo | URL) => {
            if (isRecommendFeedRequest(input)) {
                return jsonResponse(recommendFeedPayload([
                    defaultRecommendFeedItem({
                        content_id: 3001,
                        author_id: 4001,
                        author_name: "同步作者",
                        title: "需要同步点赞的内容",
                        like_count: 6,
                        is_liked: false,
                        is_favorited: false
                    })
                ]));
            }

            if (isContentDetailRequest(input)) {
                return jsonResponse(contentDetailPayload({
                    content_id: "3001",
                    author_id: "4001",
                    author_name: "同步作者",
                    title: "需要同步点赞的内容",
                    like_count: 6,
                    is_liked: false
                }));
            }

            if (isCommentListRequest(input)) {
                return jsonResponse(commentListPayload([]));
            }

            return jsonResponse({});
        });
        vi.stubGlobal("fetch", fetchMock);
        window.history.pushState({}, "", "/home");

        render(<App />);

        const heading = await screen.findByRole("heading", { name: "需要同步点赞的内容" });
        const article = heading.closest("article");
        expect(article).not.toBeNull();
        const likeButton = within(article as HTMLElement).getByRole("button", { name: "点赞" });

        fireEvent.click(likeButton);

        expect(likeButton).toHaveAttribute("aria-label", "取消点赞");
        expect(within(likeButton).getByText("7")).toBeInTheDocument();

        fireEvent.click(heading);

        await waitFor(() => expect(window.location.pathname).toBe("/content/3001"));
        const detailArticle = (await screen.findByRole("heading", { name: "需要同步点赞的内容" })).closest("article");
        expect(detailArticle).not.toBeNull();
        const detailLikeButton = within(detailArticle as HTMLElement).getByRole("button", { name: "取消点赞" });
        expect(within(detailLikeButton).getByText("7")).toBeInTheDocument();
    });

    it("uses Bearer writes for liked content and rolls back when the API fails", async () => {
        window.localStorage.setItem("zfeed.auth.session", JSON.stringify({
            token: "write-token",
            expiredAt: Math.floor(Date.now() / 1000) + 3600,
            user: { userId: 7 }
        }));
        const fetchMock = vi.fn(async (input: RequestInfo | URL) => {
            if (isRecommendFeedRequest(input)) {
                return jsonResponse(recommendFeedPayload());
            }

            return jsonResponse({ message: "raw failure" }, { status: 500 });
        });
        vi.stubGlobal("fetch", fetchMock);
        window.history.pushState({}, "", "/home");

        render(<App />);

        const likeButton = (await screen.findAllByText("favorite"))[0].closest("button");
        expect(likeButton).not.toBeNull();
        expect(likeButton).toHaveClass("text-error");

        fireEvent.click(likeButton!);

        await waitFor(() => expect(fetchMock).toHaveBeenCalledWith("/v1/interaction/unlike", expect.objectContaining({
            method: "POST",
            headers: expect.objectContaining({
                Authorization: "Bearer write-token"
            }),
            body: JSON.stringify({ content_id: "1001", scene: "ARTICLE" })
        })));
        await waitFor(() => expect(likeButton).toHaveClass("text-error"));
        expect(screen.queryByText("raw failure")).not.toBeInTheDocument();
    });

    it("disables content action buttons while the write request is pending", async () => {
        window.localStorage.setItem("zfeed.auth.session", JSON.stringify({
            token: "write-token",
            expiredAt: Math.floor(Date.now() / 1000) + 3600,
            user: { userId: 7 }
        }));
        let resolveWrite!: (response: Response) => void;
        const fetchMock = vi.fn((input: RequestInfo | URL) => {
            if (isRecommendFeedRequest(input)) {
                return Promise.resolve(jsonResponse(recommendFeedPayload()));
            }

            return new Promise<Response>((resolve) => {
                resolveWrite = resolve;
            });
        });
        vi.stubGlobal("fetch", fetchMock);
        window.history.pushState({}, "", "/home");

        render(<App />);

        const likeButton = (await screen.findAllByText("favorite"))[0].closest("button");
        expect(likeButton).not.toBeNull();
        fireEvent.click(likeButton!);

        expect(likeButton).toBeDisabled();
        expect(fetchMock).toHaveBeenCalledTimes(2);

        resolveWrite(jsonResponse({}));
        await waitFor(() => expect(likeButton).not.toBeDisabled());
    });

    it("favorites recommendation content with saved label and accessible state updates", async () => {
        window.localStorage.setItem("zfeed.auth.session", JSON.stringify({
            token: "favorite-token",
            expiredAt: Math.floor(Date.now() / 1000) + 3600,
            user: { userId: 7 }
        }));
        const fetchMock = vi.fn(async (input: RequestInfo | URL) => {
            if (isRecommendFeedRequest(input)) {
                return jsonResponse(recommendFeedPayload([
                    defaultRecommendFeedItem({
                        content_id: 3101,
                        author_id: 4101,
                        author_name: "收藏作者",
                        title: "可收藏推荐内容",
                        favorite_count: 2,
                        is_favorited: false
                    })
                ]));
            }

            return jsonResponse({});
        });
        vi.stubGlobal("fetch", fetchMock);
        window.history.pushState({}, "", "/home");

        render(<App />);

        const heading = await screen.findByRole("heading", { name: "可收藏推荐内容" });
        const article = heading.closest("article");
        expect(article).not.toBeNull();
        const favoriteButton = within(article as HTMLElement).getByRole("button", { name: "收藏" });

        fireEvent.click(favoriteButton);

        expect(favoriteButton).toHaveAttribute("aria-label", "取消收藏");
        expect(within(favoriteButton).getByText("3")).toBeInTheDocument();
        await waitFor(() => expect(fetchMock).toHaveBeenCalledWith("/v1/interaction/favorite", expect.objectContaining({
            method: "POST",
            headers: expect.objectContaining({
                Authorization: "Bearer favorite-token"
            }),
            body: JSON.stringify({ content_id: "3101", scene: "ARTICLE" })
        })));
    });

    it("uses Bearer writes for favorite content and rolls back when the API fails", async () => {
        window.localStorage.setItem("zfeed.auth.session", JSON.stringify({
            token: "favorite-token",
            expiredAt: Math.floor(Date.now() / 1000) + 3600,
            user: { userId: 7 }
        }));
        const fetchMock = vi.fn(async (input: RequestInfo | URL) => {
            if (isRecommendFeedRequest(input)) {
                return jsonResponse(recommendFeedPayload());
            }

            return jsonResponse({ message: "raw favorite failure" }, { status: 500 });
        });
        vi.stubGlobal("fetch", fetchMock);
        window.history.pushState({}, "", "/home");

        render(<App />);

        const favoriteButton = await screen.findByRole("button", { name: "收藏" });
        expect(favoriteButton).toHaveClass("text-on-surface-variant");

        fireEvent.click(favoriteButton);

        await waitFor(() => expect(fetchMock).toHaveBeenCalledWith("/v1/interaction/favorite", expect.objectContaining({
            method: "POST",
            headers: expect.objectContaining({
                Authorization: "Bearer favorite-token"
            }),
            body: JSON.stringify({ content_id: "1001", scene: "ARTICLE" })
        })));
        await waitFor(() => expect(favoriteButton).toHaveClass("text-on-surface-variant"));
        expect(screen.queryByText("raw favorite failure")).not.toBeInTheDocument();
    });

    it("follows a profile author with Bearer auth and optimistic feedback", async () => {
        window.localStorage.setItem("zfeed.auth.session", JSON.stringify({
            token: "follow-token",
            expiredAt: Math.floor(Date.now() / 1000) + 3600,
            user: { userId: 7 }
        }));
        const fetchMock = vi.fn(async (input: RequestInfo | URL) => {
            if (input === "/v1/user/profile/jax") {
                return jsonResponse(userProfilePayload());
            }

            return jsonResponse({ is_followed: true });
        });
        vi.stubGlobal("fetch", fetchMock);
        window.history.pushState({}, "", "/user/jax");

        render(<App />);

        const followButton = (await screen.findAllByRole("button", { name: /关注/ }))[0];
        fireEvent.click(followButton);

        expect(await screen.findByRole("button", { name: "已关注" })).toBeInTheDocument();
        expect(fetchMock).toHaveBeenCalledWith("/v1/user/profile/jax", expect.objectContaining({
            headers: expect.objectContaining({
                Authorization: "Bearer follow-token"
            })
        }));
        await waitFor(() => expect(fetchMock).toHaveBeenCalledWith("/v1/interaction/followings", expect.objectContaining({
            method: "POST",
            headers: expect.objectContaining({
                Authorization: "Bearer follow-token"
            }),
            body: JSON.stringify({ target_user_id: "1001" })
        })));
    });

    it("rolls back a failed profile follow without exposing raw backend errors", async () => {
        window.localStorage.setItem("zfeed.auth.session", JSON.stringify({
            token: "follow-token",
            expiredAt: Math.floor(Date.now() / 1000) + 3600,
            user: { userId: 7 }
        }));
        const fetchMock = vi.fn(async (input: RequestInfo | URL) => {
            if (input === "/v1/user/profile/jax") {
                return jsonResponse(userProfilePayload());
            }
            if (input === "/v1/feed/user/publish") {
                return jsonResponse(userPublishedFeedPayload());
            }

            return jsonResponse({ message: "raw follow failure" }, { status: 500 });
        });
        vi.stubGlobal("fetch", fetchMock);
        window.history.pushState({}, "", "/user/jax");

        render(<App />);

        const followButton = (await screen.findAllByRole("button", { name: "关注" }))[0];
        fireEvent.click(followButton);

        await waitFor(() => expect(fetchMock).toHaveBeenCalledWith("/v1/interaction/followings", expect.objectContaining({
            method: "POST",
            body: JSON.stringify({ target_user_id: "1001" })
        })));
        await waitFor(() => expect(followButton.textContent?.trim()).toBe("关注"));
        expect(followButton).not.toBeDisabled();
        expect(screen.queryByText("raw follow failure")).not.toBeInTheDocument();
    });

    it("follows the real detail author with the backend user id", async () => {
        window.localStorage.setItem("zfeed.auth.session", JSON.stringify({
            token: "detail-follow-token",
            expiredAt: Math.floor(Date.now() / 1000) + 3600,
            user: { userId: 7 }
        }));
        window.history.pushState({}, "", "/content/1001");
        const fetchMock = vi.fn(async (input: RequestInfo | URL) => {
            if (isContentDetailRequest(input)) {
                return jsonResponse(contentDetailPayload({
                    content_id: "1001",
                    author_id: "1",
                    author_name: "测试1",
                    author_avatar: "https://example.com/author1.png",
                    title: "真实后端标题",
                    description: "真实后端摘要",
                    article_content: "真实后端正文",
                    is_following_author: false
                }));
            }

            if (isCommentListRequest(input)) {
                return jsonResponse(commentListPayload([]));
            }

            if (input === "/v1/interaction/followings") {
                return jsonResponse({ is_followed: true });
            }

            return jsonResponse({});
        });
        vi.stubGlobal("fetch", fetchMock);

        render(<App />);

        await screen.findByRole("heading", { name: "真实后端标题" });
        const followButton = screen.getByRole("button", { name: "关注作者" });
        fireEvent.click(followButton);

        expect(await screen.findByRole("button", { name: "已关注" })).toBeInTheDocument();
        await waitFor(() => expect(fetchMock).toHaveBeenCalledWith("/v1/interaction/followings", expect.objectContaining({
            method: "POST",
            headers: expect.objectContaining({
                Authorization: "Bearer detail-follow-token"
            }),
            body: JSON.stringify({ target_user_id: "1" })
        })));
    });

    it("likes and favorites detail content with backend interaction APIs", async () => {
        window.localStorage.setItem("zfeed.auth.session", JSON.stringify({
            token: "detail-action-token",
            expiredAt: Math.floor(Date.now() / 1000) + 3600,
            user: { userId: 7 }
        }));
        const fetchMock = vi.fn(async (input: RequestInfo | URL) => {
            if (isContentDetailRequest(input)) {
                return jsonResponse(contentDetailPayload({
                    content_id: "1001",
                    author_id: "1",
                    author_name: "测试1",
                    title: "详情页交互内容",
                    like_count: 12,
                    favorite_count: 5,
                    is_liked: false,
                    is_favorited: false
                }));
            }

            if (isCommentListRequest(input)) {
                return jsonResponse(commentListPayload([]));
            }

            return jsonResponse({});
        });
        vi.stubGlobal("fetch", fetchMock);
        window.history.pushState({}, "", "/content/1001");

        render(<App />);

        await screen.findByRole("heading", { name: "详情页交互内容" });
        const likeButton = screen.getByRole("button", { name: "点赞" });
        const favoriteButton = screen.getByRole("button", { name: "收藏" });

        fireEvent.click(likeButton);
        fireEvent.click(favoriteButton);

        expect(likeButton).toHaveAttribute("aria-label", "取消点赞");
        expect(within(likeButton).getByText("13")).toBeInTheDocument();
        expect(favoriteButton).toHaveAttribute("aria-label", "取消收藏");
        expect(within(favoriteButton).getByText("6")).toBeInTheDocument();
        await waitFor(() => expect(fetchMock).toHaveBeenCalledWith("/v1/interaction/like", expect.objectContaining({
            method: "POST",
            headers: expect.objectContaining({
                Authorization: "Bearer detail-action-token"
            }),
            body: JSON.stringify({ content_id: "1001", content_user_id: "1", scene: "ARTICLE" })
        })));
        await waitFor(() => expect(fetchMock).toHaveBeenCalledWith("/v1/interaction/favorite", expect.objectContaining({
            method: "POST",
            headers: expect.objectContaining({
                Authorization: "Bearer detail-action-token"
            }),
            body: JSON.stringify({ content_id: "1001", scene: "ARTICLE" })
        })));
    });

    it("guides unauthenticated detail author follows to login without calling the write API", async () => {
        const fetchMock = vi.fn(async (input: RequestInfo | URL) => {
            if (isContentDetailRequest(input)) {
                return jsonResponse(contentDetailPayload({
                    content_id: "1001",
                    author_id: "1",
                    author_name: "测试1",
                    author_avatar: "https://example.com/author1.png",
                    title: "真实后端标题",
                    description: "真实后端摘要",
                    article_content: "真实后端正文",
                    is_following_author: false
                }));
            }

            if (isCommentListRequest(input)) {
                return jsonResponse(commentListPayload([]));
            }

            return jsonResponse({});
        });
        vi.stubGlobal("fetch", fetchMock);
        window.history.pushState({}, "", "/content/1001");

        render(<App />);

        await screen.findByRole("heading", { name: "真实后端标题" });
        const followButton = screen.getByRole("button", { name: "关注作者" });
        fireEvent.click(followButton);

        await waitFor(() => expect(window.location.pathname).toBe("/login"));
        expect(window.location.search).toBe("?next=%2Fcontent%2F1001");
        expect(fetchMock).toHaveBeenCalledWith("/v1/content/detail", expect.anything());
    });

    it("follows a recommended home author with Bearer auth and optimistic feedback", async () => {
        window.localStorage.setItem("zfeed.auth.session", JSON.stringify({
            token: "follow-token",
            expiredAt: Math.floor(Date.now() / 1000) + 3600,
            user: { userId: 7 }
        }));
        const fetchMock = vi.fn(async (input: RequestInfo | URL) => {
            if (isRecommendFeedRequest(input)) {
                return jsonResponse(recommendFeedPayload([
                    defaultRecommendFeedItem({
                        author_id: 1,
                        author_name: "测试1",
                        author_avatar: "1",
                        title: "测试内容1",
                        description: "测试内容1"
                    })
                ]));
            }

            return jsonResponse({ is_followed: true });
        });
        vi.stubGlobal("fetch", fetchMock);
        window.history.pushState({}, "", "/home");

        render(<App />);

        const suggestedPanel = await screen.findByRole("heading", { name: "为你推荐" }).then((heading) => heading.closest(".glass-panel"));
        expect(suggestedPanel).not.toBeNull();
        const suggestedAuthor = within(suggestedPanel as HTMLElement).getByText("测试1").closest("[data-suggested-user-id]");
        expect(suggestedAuthor).not.toBeNull();
        const followButton = within(suggestedAuthor as HTMLElement).getByRole("button", { name: "关注" });
        fireEvent.click(followButton);

        expect(await screen.findByRole("button", { name: "已关注" })).toBeInTheDocument();
        await waitFor(() => expect(fetchMock).toHaveBeenCalledWith("/v1/interaction/followings", expect.objectContaining({
            method: "POST",
            headers: expect.objectContaining({
                Authorization: "Bearer follow-token"
            }),
            body: JSON.stringify({ target_user_id: "1" })
        })));
    });

    it("guides unauthenticated recommended author follows to login without calling the write API", async () => {
        const fetchMock = vi.fn(async (input: RequestInfo | URL) => {
            if (isRecommendFeedRequest(input)) {
                return jsonResponse(recommendFeedPayload());
            }

            return jsonResponse({});
        });
        vi.stubGlobal("fetch", fetchMock);
        window.history.pushState({}, "", "/home");

        render(<App />);

        const suggestedFollowButtons = await screen.findAllByRole("button", { name: "关注" });
        fireEvent.click(suggestedFollowButtons[0]);

        await waitFor(() => expect(window.location.pathname).toBe("/login"));
        expect(window.location.search).toBe("?next=%2Fhome");
        expect(fetchMock).toHaveBeenCalledTimes(1);
        expect(fetchMock).toHaveBeenCalledWith("/v1/feed/recommend", expect.any(Object));
    });

    it("disables follow buttons while the request is pending", async () => {
        window.localStorage.setItem("zfeed.auth.session", JSON.stringify({
            token: "follow-token",
            expiredAt: Math.floor(Date.now() / 1000) + 3600,
            user: { userId: 7 }
        }));
        let resolveFollow!: (response: Response) => void;
        const fetchMock = vi.fn((input: RequestInfo | URL) => {
            if (input === "/v1/user/profile/jax") {
                return Promise.resolve(jsonResponse(userProfilePayload()));
            }
            if (input === "/v1/feed/user/publish") {
                return Promise.resolve(jsonResponse(userPublishedFeedPayload()));
            }
            const listResponse = respondToProfileListRequest(input);
            if (listResponse) {
                return Promise.resolve(listResponse);
            }

            return new Promise<Response>((resolve) => {
                resolveFollow = resolve;
            });
        });
        vi.stubGlobal("fetch", fetchMock);
        window.history.pushState({}, "", "/user/jax");

        render(<App />);

        const followButton = (await screen.findAllByRole("button", { name: /关注/ }))[0];
        fireEvent.click(followButton);

        expect(await screen.findByRole("button", { name: "已关注" })).toBeDisabled();
        expect(fetchMock).toHaveBeenCalledTimes(5);

        resolveFollow(jsonResponse({ is_followed: true }));
        await waitFor(() => expect(followButton).not.toBeDisabled());
    });

    it("posts a content comment with optimistic insertion and clears the composer", async () => {
        window.localStorage.setItem("zfeed.auth.session", JSON.stringify({
            token: "comment-token",
            expiredAt: Math.floor(Date.now() / 1000) + 3600,
            user: { userId: 7, nickname: "Mira Chen" }
        }));
        const fetchMock = vi.fn(async (input: RequestInfo | URL) => {
            if (isContentDetailRequest(input)) {
                return jsonResponse(contentDetailPayload({
                    content_id: "1001",
                    author_id: "1001",
                    author_name: "测试作者",
                    author_avatar: "https://example.com/author.png",
                    title: "真实后端标题",
                    description: "真实后端摘要",
                    article_content: "真实后端正文"
                }));
            }
            if (isCommentListRequest(input)) {
                return jsonResponse(commentListPayload([]));
            }
            if (input === "/v1/interaction/comment") {
                return jsonResponse({ comment_id: "5001" });
            }
            return jsonResponse({});
        });
        vi.stubGlobal("fetch", fetchMock);
        window.history.pushState({}, "", "/content/1001");

        render(<App />);

        const composer = await screen.findByPlaceholderText("写下你的观点，补充或提问...");
        expect(screen.getByText("3 条讨论")).toBeInTheDocument();
        fireEvent.change(composer, { target: { value: "这条评论来自真实接口" } });
        fireEvent.click(screen.getByRole("button", { name: "发送" }));

        expect(await screen.findByText("这条评论来自真实接口")).toBeInTheDocument();
        expect(await screen.findByText("4 条讨论")).toBeInTheDocument();
        expect(screen.getByText("还没有评论")).not.toBeVisible();
        await waitFor(() => expect(composer).toHaveValue(""));
        await waitFor(() => expect(fetchMock).toHaveBeenCalledWith("/v1/interaction/comment", expect.objectContaining({
            method: "POST",
            headers: expect.objectContaining({
                Authorization: "Bearer comment-token"
            }),
            body: JSON.stringify({
                content_id: "1001",
                scene: "ARTICLE",
                comment: "这条评论来自真实接口",
                content_user_id: "1001"
            })
        })));
    });

    it("validates content comment length before calling the API", async () => {
        window.localStorage.setItem("zfeed.auth.session", JSON.stringify({
            token: "comment-token",
            expiredAt: Math.floor(Date.now() / 1000) + 3600,
            user: { userId: 7, nickname: "Mira Chen" }
        }));
        const fetchMock = vi.fn(async (input: RequestInfo | URL) => {
            if (isContentDetailRequest(input)) {
                return jsonResponse(contentDetailPayload({
                    content_id: "1001",
                    author_id: "1001",
                    author_name: "测试作者",
                    author_avatar: "https://example.com/author.png",
                    title: "真实后端标题",
                    description: "真实后端摘要",
                    article_content: "真实后端正文"
                }));
            }
            if (isCommentListRequest(input)) {
                return jsonResponse(commentListPayload([]));
            }
            return jsonResponse({});
        });
        vi.stubGlobal("fetch", fetchMock);
        window.history.pushState({}, "", "/content/1001");

        render(<App />);

        const composer = await screen.findByPlaceholderText("写下你的观点，补充或提问...");
        fireEvent.change(composer, { target: { value: "评".repeat(256) } });
        fireEvent.click(screen.getByRole("button", { name: "发送" }));

        expect(await screen.findByText("评论最多 255 字")).toBeInTheDocument();
        expect(fetchMock).not.toHaveBeenCalledWith("/v1/interaction/comment", expect.anything());
    });

    it("disables the comment submit button while the request is pending", async () => {
        window.localStorage.setItem("zfeed.auth.session", JSON.stringify({
            token: "comment-token",
            expiredAt: Math.floor(Date.now() / 1000) + 3600,
            user: { userId: 7, nickname: "Mira Chen" }
        }));
        let resolveComment!: (response: Response) => void;
        const fetchMock = vi.fn((input: RequestInfo | URL) => {
            if (isContentDetailRequest(input)) {
                return Promise.resolve(jsonResponse(contentDetailPayload({
                    content_id: "1001",
                    author_id: "1001",
                    author_name: "测试作者",
                    author_avatar: "https://example.com/author.png",
                    title: "真实后端标题",
                    description: "真实后端摘要",
                    article_content: "真实后端正文"
                })));
            }
            if (isCommentListRequest(input)) {
                return Promise.resolve(jsonResponse(commentListPayload([])));
            }
            return new Promise<Response>((resolve) => {
                resolveComment = resolve;
            });
        });
        vi.stubGlobal("fetch", fetchMock);
        window.history.pushState({}, "", "/content/1001");

        render(<App />);

        const composer = await screen.findByPlaceholderText("写下你的观点，补充或提问...");
        fireEvent.change(composer, { target: { value: "请求中禁用评论按钮" } });
        fireEvent.click(screen.getByRole("button", { name: "发送" }));

        expect(await screen.findByRole("button", { name: "发送中" })).toBeDisabled();
        expect(fetchMock).toHaveBeenCalledWith("/v1/interaction/comment", expect.objectContaining({
            method: "POST"
        }));

        resolveComment(jsonResponse({ comment_id: "5003" }));
        await waitFor(() => expect(screen.getByRole("button", { name: "发送" })).not.toBeDisabled());
    });

    it("posts a reply from an inline reply composer with Bearer auth", async () => {
        window.localStorage.setItem("zfeed.auth.session", JSON.stringify({
            token: "reply-token",
            expiredAt: Math.floor(Date.now() / 1000) + 3600,
            user: { userId: 7, nickname: "Mira Chen" }
        }));
        const fetchMock = vi.fn(async (input: RequestInfo | URL) => {
            if (isContentDetailRequest(input)) {
                return jsonResponse(contentDetailPayload({
                    content_id: "1001",
                    author_id: "1001",
                    author_name: "测试作者",
                    author_avatar: "https://example.com/author.png",
                    title: "真实后端标题",
                    description: "真实后端摘要",
                    article_content: "真实后端正文"
                }));
            }
            if (isCommentListRequest(input)) {
                return jsonResponse(commentListPayload([defaultCommentItem({
                    comment_id: "3001",
                    user_id: "2001",
                    nickname: "测试1",
                    comment: "后端评论1"
                })]));
            }
            if (input === "/v1/interaction/comment") {
                return jsonResponse({ comment_id: "5002" });
            }
            return jsonResponse({});
        });
        vi.stubGlobal("fetch", fetchMock);
        window.history.pushState({}, "", "/content/1001");

        render(<App />);

        const replyButtons = await screen.findAllByRole("button", { name: "回复" });
        expect(screen.getByText("3 条讨论")).toBeInTheDocument();
        fireEvent.click(replyButtons[0]);

        const replyInput = await screen.findByPlaceholderText("回复 测试1...");
        fireEvent.change(replyInput, { target: { value: "补充一下这条回复" } });
        const replyComposer = replyInput.closest("[data-reply-composer]");
        expect(replyComposer).not.toBeNull();
        fireEvent.click(within(replyComposer as HTMLElement).getByRole("button", { name: "发送" }));

        expect(await screen.findByText("补充一下这条回复")).toBeInTheDocument();
        expect(await screen.findByText("4 条讨论")).toBeInTheDocument();
        await waitFor(() => expect(fetchMock).toHaveBeenCalledWith("/v1/interaction/comment", expect.objectContaining({
            method: "POST",
            headers: expect.objectContaining({
                Authorization: "Bearer reply-token"
            }),
            body: JSON.stringify({
                content_id: "1001",
                scene: "ARTICLE",
                comment: "补充一下这条回复",
                parent_id: "3001",
                root_id: "3001",
                reply_to_user_id: "2001",
                content_user_id: "1001"
            })
        })));
    });

    it("disables the reply submit button while the request is pending", async () => {
        window.localStorage.setItem("zfeed.auth.session", JSON.stringify({
            token: "reply-token",
            expiredAt: Math.floor(Date.now() / 1000) + 3600,
            user: { userId: 7, nickname: "Mira Chen" }
        }));
        let resolveReply!: (response: Response) => void;
        const fetchMock = vi.fn((input: RequestInfo | URL) => {
            if (isContentDetailRequest(input)) {
                return Promise.resolve(jsonResponse(contentDetailPayload({
                    content_id: "1001",
                    author_id: "1001",
                    author_name: "测试作者",
                    author_avatar: "https://example.com/author.png",
                    title: "真实后端标题",
                    description: "真实后端摘要",
                    article_content: "真实后端正文"
                })));
            }
            if (isCommentListRequest(input)) {
                return Promise.resolve(jsonResponse(commentListPayload([defaultCommentItem({
                    comment_id: "3001",
                    user_id: "2001",
                    nickname: "测试1",
                    comment: "后端评论1"
                })])));
            }
            return new Promise<Response>((resolve) => {
                resolveReply = resolve;
            });
        });
        vi.stubGlobal("fetch", fetchMock);
        window.history.pushState({}, "", "/content/1001");

        render(<App />);

        const replyButtons = await screen.findAllByRole("button", { name: "回复" });
        fireEvent.click(replyButtons[0]);

        const replyInput = await screen.findByPlaceholderText("回复 测试1...");
        fireEvent.change(replyInput, { target: { value: "请求中禁用回复按钮" } });
        const replyComposer = replyInput.closest("[data-reply-composer]");
        expect(replyComposer).not.toBeNull();
        fireEvent.click(within(replyComposer as HTMLElement).getByRole("button", { name: "发送" }));

        expect(await within(replyComposer as HTMLElement).findByRole("button", { name: "发送中" })).toBeDisabled();
        expect(fetchMock).toHaveBeenCalledWith("/v1/interaction/comment", expect.objectContaining({
            method: "POST"
        }));

        resolveReply(jsonResponse({ comment_id: "5004" }));
        await waitFor(() => expect(within(replyComposer as HTMLElement).getByRole("button", { name: "发送" })).not.toBeDisabled());
    });

    it("deletes my own comment optimistically with Bearer auth", async () => {
        window.localStorage.setItem("zfeed.auth.session", JSON.stringify({
            token: "delete-token",
            expiredAt: Math.floor(Date.now() / 1000) + 3600,
            user: { userId: 7, nickname: "Mira Chen" }
        }));
        const fetchMock = vi.fn(async (input: RequestInfo | URL) => {
            if (isContentDetailRequest(input)) {
                return jsonResponse(contentDetailPayload({
                    content_id: "1001",
                    author_id: "1001",
                    author_name: "测试作者",
                    author_avatar: "https://example.com/author.png",
                    title: "真实后端标题",
                    description: "真实后端摘要",
                    article_content: "真实后端正文"
                }));
            }
            if (isCommentListRequest(input)) {
                return jsonResponse(commentListPayload([defaultCommentItem({
                    comment_id: "3003",
                    root_id: "3003",
                    user_id: "7",
                    nickname: "测试7",
                    comment: "评论区也需要保持阅读节奏，信息密度不能太吵。喜欢这种评论卡片和正文之间的层级。"
                })]));
            }
            return jsonResponse({});
        });
        vi.stubGlobal("fetch", fetchMock);
        window.history.pushState({}, "", "/content/1001");

        render(<App />);

        expect(await screen.findByText("评论区也需要保持阅读节奏，信息密度不能太吵。喜欢这种评论卡片和正文之间的层级。")).toBeInTheDocument();
        expect(screen.getByText("3 条讨论")).toBeInTheDocument();
        fireEvent.click(screen.getByRole("button", { name: "删除评论" }));

        await waitFor(() => expect(screen.queryByText("评论区也需要保持阅读节奏，信息密度不能太吵。喜欢这种评论卡片和正文之间的层级。")).not.toBeInTheDocument());
        expect(await screen.findByText("2 条讨论")).toBeInTheDocument();
        await waitFor(() => expect(fetchMock).toHaveBeenCalledWith("/v1/interaction/comment", expect.objectContaining({
            method: "DELETE",
            headers: expect.objectContaining({
                Authorization: "Bearer delete-token"
            }),
            body: JSON.stringify({
                comment_id: "3003",
                content_id: "1001",
                scene: "ARTICLE",
                root_id: "3003"
            })
        })));
    });

    it("restores a deleted comment and shows a safe error when delete fails", async () => {
        window.localStorage.setItem("zfeed.auth.session", JSON.stringify({
            token: "delete-token",
            expiredAt: Math.floor(Date.now() / 1000) + 3600,
            user: { userId: 7, nickname: "Mira Chen" }
        }));
        const fetchMock = vi.fn(async (input: RequestInfo | URL) => {
            if (isContentDetailRequest(input)) {
                return jsonResponse(contentDetailPayload({
                    content_id: "1001",
                    author_id: "1001",
                    author_name: "测试作者",
                    author_avatar: "https://example.com/author.png",
                    title: "真实后端标题",
                    description: "真实后端摘要",
                    article_content: "真实后端正文"
                }));
            }
            if (isCommentListRequest(input)) {
                return jsonResponse(commentListPayload([defaultCommentItem({
                    comment_id: "3003",
                    root_id: "3003",
                    user_id: "7",
                    nickname: "测试7",
                    comment: "评论区也需要保持阅读节奏，信息密度不能太吵。喜欢这种评论卡片和正文之间的层级。"
                })]));
            }
            return jsonResponse({ message: "raw delete failure" }, { status: 500 });
        });
        vi.stubGlobal("fetch", fetchMock);
        window.history.pushState({}, "", "/content/1001");

        render(<App />);

        const comment = "评论区也需要保持阅读节奏，信息密度不能太吵。喜欢这种评论卡片和正文之间的层级。";
        expect(await screen.findByText(comment)).toBeInTheDocument();
        expect(screen.getByText("3 条讨论")).toBeInTheDocument();
        fireEvent.click(screen.getByRole("button", { name: "删除评论" }));

        await waitFor(() => expect(fetchMock).toHaveBeenCalledWith("/v1/interaction/comment", expect.objectContaining({
            method: "DELETE"
        })));
        expect(await screen.findByText(comment)).toBeInTheDocument();
        expect(await screen.findByText("3 条讨论")).toBeInTheDocument();
        expect(await screen.findByText("删除失败，请重试")).toBeInTheDocument();
        expect(screen.queryByText("raw delete failure")).not.toBeInTheDocument();
    });

    it("saves edited profile fields with validation and Bearer auth", async () => {
        window.localStorage.setItem("zfeed.auth.session", JSON.stringify({
            token: "profile-token",
            expiredAt: Math.floor(Date.now() / 1000) + 3600,
            user: { userId: 7 }
        }));
        const fetchMock = vi.fn(async () => jsonResponse({
            user_info: {
                user_id: 7,
                mobile: "13800138000",
                nickname: "Mira Updated",
                avatar: "",
                bio: "新的简介",
                gender: 0,
                status: 1,
                email: "mira.updated@example.com",
                birthday: 0
            }
        }));
        vi.stubGlobal("fetch", fetchMock);
        window.history.pushState({}, "", "/me/edit");

        render(<App />);

        fireEvent.change(await screen.findByLabelText("昵称"), { target: { value: "Mira Updated" } });
        fireEvent.change(screen.getByLabelText("简介"), { target: { value: "新的简介" } });
        fireEvent.change(screen.getByLabelText("邮箱"), { target: { value: "mira.updated@example.com" } });
        fireEvent.click(screen.getByRole("button", { name: /保存/ }));

        expect(await screen.findByText("资料已保存")).toBeInTheDocument();
        await waitFor(() => expect(fetchMock).toHaveBeenCalledWith("/v1/users/me/profile", expect.objectContaining({
            method: "PUT",
            headers: expect.objectContaining({
                Authorization: "Bearer profile-token"
            }),
            body: JSON.stringify({
                nickname: "Mira Updated",
                avatar: undefined,
                bio: "新的简介",
                email: "mira.updated@example.com",
                gender: 0,
                birthday: undefined
            })
        })));
    });

    it("disables the edit profile save button while the request is pending", async () => {
        window.localStorage.setItem("zfeed.auth.session", JSON.stringify({
            token: "profile-token",
            expiredAt: Math.floor(Date.now() / 1000) + 3600,
            user: { userId: 7 }
        }));
        let resolveProfileSave!: (response: Response) => void;
        const fetchMock = vi.fn((input: RequestInfo | URL) => {
            if (input === "/v1/users/me") {
                return Promise.resolve(jsonResponse(meProfilePayload()));
            }

            if (input === "/v1/search/users") {
                return Promise.resolve(jsonResponse({ items: [] }));
            }

            return new Promise<Response>((resolve) => {
                resolveProfileSave = resolve;
            });
        });
        vi.stubGlobal("fetch", fetchMock);
        window.history.pushState({}, "", "/me/edit");

        render(<App />);

        fireEvent.change(await screen.findByLabelText("昵称"), { target: { value: "Mira Pending" } });
        fireEvent.click(screen.getByRole("button", { name: /保存/ }));

        expect(await screen.findByRole("button", { name: /保存中/ })).toBeDisabled();
        expect(fetchMock).toHaveBeenCalledWith("/v1/users/me/profile", expect.objectContaining({
            method: "PUT"
        }));

        resolveProfileSave(jsonResponse({}));
        expect(await screen.findByText("资料已保存")).toBeInTheDocument();
    });

    it("publishes an article from compose and routes to the new content", async () => {
        window.localStorage.setItem("zfeed.auth.session", JSON.stringify({
            token: "publish-token",
            expiredAt: Math.floor(Date.now() / 1000) + 3600,
            user: { userId: 7 }
        }));
        const fetchMock = vi.fn(async () => jsonResponse({ content_id: 6789 }));
        vi.stubGlobal("fetch", fetchMock);
        window.history.pushState({}, "", "/compose");

        render(<App />);

        fireEvent.change(await screen.findByPlaceholderText("标题"), { target: { value: "一篇真实发布的文章" } });
        fireEvent.change(screen.getByPlaceholderText("写一段简介，方便读者快速了解内容..."), { target: { value: "这是独立文章简介" } });
        fireEvent.change(screen.getByPlaceholderText("封面 URL，可选"), { target: { value: "https://example.com/article-cover.png" } });
        fireEvent.change(screen.getByPlaceholderText("写下你的想法..."), { target: { value: "这是正文内容" } });
        fireEvent.click(screen.getByRole("button", { name: "发布" }));

        expect(await screen.findByRole("button", { name: "发布中" })).toBeDisabled();
        await waitFor(() => expect(fetchMock).toHaveBeenCalledWith("/v1/content/article/publish", expect.objectContaining({
            method: "POST",
            headers: expect.objectContaining({
                Authorization: "Bearer publish-token"
            }),
            body: JSON.stringify({
                title: "一篇真实发布的文章",
                description: "这是独立文章简介",
                cover: "https://example.com/article-cover.png",
                content: "这是正文内容",
                visibility: 10
            })
        })));
        await waitFor(() => expect(window.location.pathname).toBe("/content/6789"));
    });

    it("omits optional article fields when publishing without summary or cover", async () => {
        window.localStorage.setItem("zfeed.auth.session", JSON.stringify({
            token: "publish-token",
            expiredAt: Math.floor(Date.now() / 1000) + 3600,
            user: { userId: 7 }
        }));
        const fetchMock = vi.fn(async () => jsonResponse({ content_id: 6790 }));
        vi.stubGlobal("fetch", fetchMock);
        window.history.pushState({}, "", "/compose");

        render(<App />);

        fireEvent.change(await screen.findByPlaceholderText("标题"), { target: { value: "没有简介的文章" } });
        fireEvent.change(screen.getByPlaceholderText("写下你的想法..."), { target: { value: "正文仍然会发布" } });
        fireEvent.click(screen.getByRole("button", { name: "发布" }));

        await waitFor(() => expect(fetchMock).toHaveBeenCalledWith("/v1/content/article/publish", expect.objectContaining({
            method: "POST",
            body: JSON.stringify({
                title: "没有简介的文章",
                content: "正文仍然会发布",
                visibility: 10
            })
        })));
        await waitFor(() => expect(window.location.pathname).toBe("/content/6790"));
    });

    it("validates compose title and content before publishing", async () => {
        window.localStorage.setItem("zfeed.auth.session", JSON.stringify({
            token: "publish-token",
            expiredAt: Math.floor(Date.now() / 1000) + 3600,
            user: { userId: 7 }
        }));
        const fetchMock = vi.fn();
        vi.stubGlobal("fetch", fetchMock);
        window.history.pushState({}, "", "/compose");

        render(<App />);

        const titleInput = await screen.findByPlaceholderText("标题");
        const composeForm = titleInput.closest('[data-compose-form="true"]');
        expect(composeForm).not.toBeNull();
        fireEvent.click(within(composeForm as HTMLElement).getByRole("button", { name: "发布" }));

        expect(await screen.findByText("请输入标题")).toBeInTheDocument();
        expect(fetchMock).not.toHaveBeenCalled();
        expect(window.location.pathname).toBe("/compose");
    });

    it("keeps compose recoverable when publish succeeds without a content id", async () => {
        window.localStorage.setItem("zfeed.auth.session", JSON.stringify({
            token: "publish-token",
            expiredAt: Math.floor(Date.now() / 1000) + 3600,
            user: { userId: 7 }
        }));
        const fetchMock = vi.fn(async () => jsonResponse({}));
        vi.stubGlobal("fetch", fetchMock);
        window.history.pushState({}, "", "/compose");

        render(<App />);

        fireEvent.change(await screen.findByPlaceholderText("标题"), { target: { value: "缺少 ID 的发布响应" } });
        fireEvent.change(screen.getByPlaceholderText("写下你的想法..."), { target: { value: "后端响应成功但没有内容 ID。" } });
        fireEvent.click(screen.getByRole("button", { name: "发布" }));

        await waitFor(() => expect(fetchMock).toHaveBeenCalledWith("/v1/content/article/publish", expect.objectContaining({
            method: "POST"
        })));
        expect(await screen.findByText("发布失败，请重试")).toBeInTheDocument();
        expect(screen.getByRole("button", { name: "发布" })).not.toBeDisabled();
        expect(window.location.pathname).toBe("/compose");
    });

    it("submits the search page input back into the product search route", async () => {
        window.history.pushState({}, "", "/search");

        render(<App />);

        const searchInput = await screen.findByPlaceholderText("搜索内容、创作者或话题");
        expect(screen.getByText("输入关键词开始搜索").closest("[data-page-state]")).toHaveAttribute("data-page-state", "empty");
        fireEvent.change(searchInput, { target: { value: "设计 系统" } });
        fireEvent.keyDown(searchInput, { key: "Enter" });

        await waitFor(() => expect(window.location.pathname).toBe("/search"));
        expect(window.location.search).toBe("?q=%E8%AE%BE%E8%AE%A1+%E7%B3%BB%E7%BB%9F");
    });

    it("opens internal links inside the React app without a full page navigation", async () => {
        window.localStorage.setItem("zfeed.auth.session", JSON.stringify({
            token: "me-token",
            expiredAt: Math.floor(Date.now() / 1000) + 3600,
            user: { userId: 7 }
        }));
        vi.stubGlobal("fetch", vi.fn(async () => jsonResponse(recommendFeedPayload())));
        window.history.pushState({}, "", "/home");

        render(<App />);

        const profileLink = await screen.findByLabelText("进入我的主页");
        fireEvent.click(profileLink);

        await waitFor(() => {
            expect(window.location.pathname).toBe("/me");
        });
        expect(window.location.search).toBe("");
        expect(await screen.findByText("编辑资料")).toBeInTheDocument();
    });

    it("shows not found for old .html URLs", async () => {
        window.history.pushState({}, "", "/following.html");

        render(<App />);

        const errorState = await screen.findByText("页面不存在");
        expect(errorState.closest("[data-page-state]")).toHaveAttribute("data-page-state", "error");
    });
});
