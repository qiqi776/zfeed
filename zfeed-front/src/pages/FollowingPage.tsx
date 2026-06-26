import { createElement, useEffect, useState } from "react";
import { getFollowFeed } from "../runtime/apiClient";
import { readAuthSession } from "../runtime/authStore";
import { useContentInteractionVersion } from "../runtime/contentInteractionStore";
import { PageShell } from "../runtime/PageShell";
import { type FeedRailContentItem, getCachedHomeRailItems, homeStyles, renderFeedCard, renderHomeHeader, renderLeftRail, renderRightRail } from "./feedShell";
import { PageState } from "./PageState";
import { sharedGlassBodyClass } from "./sharedGlassStyles";

type FollowFeedItem = {
    content_id: string | number;
    content_type?: number;
    author_id: string | number;
    author_name: string;
    author_avatar?: string;
    title: string;
    description?: string;
    cover_url?: string;
    published_at?: number;
    like_count?: number;
    favorite_count?: number;
    comment_count?: number;
    is_liked?: boolean;
    is_favorited?: boolean;
};

type FollowFeedResponse = {
    items?: FollowFeedItem[];
};

type FollowFeedState =
    | { status: "auth-required" }
    | { status: "loading" }
    | { status: "ready"; items: FollowFeedItem[] }
    | { status: "empty" }
    | { status: "error" };

const followFeedPageSize = 20;

export function FollowingPage() {
    const [state, setState] = useState<FollowFeedState>(() => readAuthSession() ? { status: "loading" } : { status: "auth-required" });
    useContentInteractionVersion();

    useEffect(() => {
        if (!readAuthSession()) {
            return;
        }

        let isCurrent = true;

        getFollowFeed<FollowFeedResponse>({ cursor: "", page_size: followFeedPageSize })
            .then((response) => {
                if (!isCurrent) {
                    return;
                }

                const items = response.items ?? [];
                setState(items.length > 0 ? { status: "ready", items } : { status: "empty" });
            })
            .catch(() => {
                if (!isCurrent) {
                    return;
                }

                setState(readAuthSession() ? { status: "error" } : { status: "auth-required" });
            });

        return () => {
            isCurrent = false;
        };
    }, []);

    const preserveFeedRails = shouldPreserveFeedRails();
    const railItems = preserveFeedRails ? getCachedHomeRailItems() : state.status === "ready" ? state.items : [];
    return renderFollowingPageShell(renderFollowingState(state), railItems);
}

function renderFollowingPageShell(mainContent: ReturnType<typeof createElement>, railItems: FeedRailContentItem[]) {
    return createElement(
        PageShell,
        { title: "zfeed - 关注信息流", htmlClass: "light", bodyClass: sharedGlassBodyClass, styles: homeStyles },
        createElement("div", { className: "page-root" },
            renderHomeHeader("following"),
            createElement("div", { className: "pt-24 px-4 md:px-6 max-w-[1600px] mx-auto pb-safe" },
                createElement("div", { className: "grid grid-cols-4 md:grid-cols-8 lg:grid-cols-12 gap-5" },
                    renderLeftRail("following"),
                    createElement("main", { className: "feed-transition col-span-4 md:col-span-8 lg:col-span-7 flex flex-col gap-6 pb-24 feed-ready" },
                        mainContent
                    ),
                    renderRightRail(railItems)
                )
            )
        )
    );
}

function shouldPreserveFeedRails() {
    const state = window.history.state as { preserveFeedRails?: boolean } | null;
    return state?.preserveFeedRails === true;
}

function renderFollowingState(state: FollowFeedState) {
    if (state.status === "ready") {
        return createElement("div", { className: "flex flex-col gap-6" },
            createElement("div", { className: "glass-panel rounded-2xl p-5" },
                createElement("div", { className: "flex flex-wrap items-end justify-between gap-3" },
                    createElement("div", null,
                        createElement("p", { className: "font-meta-xs uppercase tracking-wider text-on-surface-variant/70" }, "Following"),
                        createElement("h1", { className: "mt-1 font-display text-[30px] leading-tight text-on-surface" }, "关注流")
                    ),
                    createElement("a", { className: "glass-button-ghost rounded-full px-4 py-2 text-primary font-label-sm active:scale-95", href: "/search" }, "发现作者")
                )
            ),
            ...state.items.map(renderFeedCard)
        );
    }

    return createElement(PageState, getFollowingStateView(state.status));
}

function getFollowingStateView(status: Exclude<FollowFeedState["status"], "ready">) {
    if (status === "auth-required") {
        return {
            state: "auth-required" as const,
            description: "登录后才能查看关注流。",
            actionHref: `/login?next=${encodeURIComponent("/following")}`,
            actionLabel: "去登录"
        };
    }

    if (status === "loading") {
        return {
            state: "loading" as const,
            title: "正在加载关注流",
            description: "正在获取你关注作者的新内容。"
        };
    }

    if (status === "empty") {
        return {
            state: "empty" as const,
            title: "关注一些作者后，这里会出现他们的新内容",
            description: "去搜索页发现值得关注的创作者。",
            actionHref: "/search",
            actionLabel: "去搜索"
        };
    }

    return {
        state: "error" as const,
        title: "关注流加载失败",
        description: "请稍后重试。"
    };
}
