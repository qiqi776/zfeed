import { createElement, useEffect, useState } from "react";
import { searchContents, searchUsers } from "../runtime/apiClient";
import { useContentInteractionVersion } from "../runtime/contentInteractionStore";
import { PageShell } from "../runtime/PageShell";
import { renderUserAvatar } from "./avatar";
import { type FeedCardItem, getCachedHomeRailItems, homeStyles, renderFeedCard, renderHomeHeader, renderLeftRail, renderRightRail } from "./feedShell";
import { PageState } from "./PageState";
import { sharedGlassBodyClass } from "./sharedGlassStyles";

type SearchContentItem = FeedCardItem;

type SearchUserItem = {
    user_id: string | number;
    nickname: string;
    avatar?: string;
    bio?: string;
    is_following?: boolean;
};

type SearchContentsResponse = {
    items?: SearchContentItem[];
};

type SearchUsersResponse = {
    items?: SearchUserItem[];
};

type SearchViewState =
    | { status: "idle" }
    | { status: "loading" }
    | { status: "invalid" }
    | { status: "error" }
    | { status: "ready"; contents: SearchContentItem[]; users: SearchUserItem[] };

type SearchTarget = "content" | "author";

const maxSearchQueryLength = 50;
const searchPageSize = 10;

export function SearchPage() {
    const searchParams = new URLSearchParams(window.location.search);
    const query = searchParams.get("q")?.trim() ?? "";
    const searchTarget = getSearchTarget(searchParams.get("type"));
    const [viewState, setViewState] = useState<SearchViewState>(() => getInitialSearchState(query));
    useContentInteractionVersion();

    useEffect(() => {
        if (!query || query.length > maxSearchQueryLength) {
            return;
        }

        let isCurrent = true;

        const request = searchTarget === "author"
            ? searchUsers<SearchUsersResponse>({ query, page_size: searchPageSize, mode: "latest" })
            : searchContents<SearchContentsResponse>({ query, page_size: searchPageSize, mode: "latest" });

        request.then((result) => {
            if (!isCurrent) {
                return;
            }

            setViewState({
                status: "ready",
                contents: searchTarget === "content" ? (result as SearchContentsResponse).items ?? [] : [],
                users: searchTarget === "author" ? (result as SearchUsersResponse).items ?? [] : []
            });
        }).catch(() => {
            if (isCurrent) {
                setViewState({ status: "error" });
            }
        });

        return () => {
            isCurrent = false;
        };
    }, [query, searchTarget]);

    return createElement(
        PageShell,
        {
            title: "zfeed - 搜索",
            htmlClass: "light",
            bodyClass: sharedGlassBodyClass,
            styles: homeStyles
        },
        createElement("div", { className: "page-root" },
            renderHomeHeader("search", { showSearch: false }),
            createElement("div", { className: "pt-24 px-4 md:px-6 max-w-[1600px] mx-auto pb-safe" },
                createElement("div", { className: "grid grid-cols-4 md:grid-cols-8 lg:grid-cols-12 gap-5" },
                    renderLeftRail("search"),
                    createElement("main", { className: "feed-transition col-span-4 md:col-span-8 lg:col-span-7 flex flex-col gap-6 pb-24 feed-ready" },
                        renderSearchPanel(query, searchTarget),
                        renderSearchState(query, searchTarget, viewState)
                    ),
                    renderRightRail(getCachedHomeRailItems())
                )
            )
        )
    );
}

function getInitialSearchState(query: string): SearchViewState {
    if (!query) {
        return { status: "idle" };
    }

    if (query.length > maxSearchQueryLength) {
        return { status: "invalid" };
    }

    return { status: "loading" };
}

function renderSearchPanel(query: string, searchTarget: SearchTarget) {
    return createElement("section", { className: "glass-panel rounded-2xl p-5 hover-lift shine-effect" },
        createElement("div", { className: "flex flex-wrap items-end justify-between gap-3" },
            createElement("div", null,
                createElement("p", { className: "font-meta-xs uppercase tracking-wider text-on-surface-variant/70" }, "Search"),
                createElement("h1", { className: "mt-1 font-display text-[30px] leading-tight text-on-surface" }, "搜索")
            ),
            createElement("a", { className: "glass-button-ghost rounded-full px-4 py-2 text-primary font-label-sm active:scale-95", href: "/home" }, "返回推荐")
        ),
        createElement("label", { className: "mt-5 block" },
            createElement("span", { className: "sr-only" }, "关键词"),
            createElement("div", { className: "composer-shell" },
                createElement("input", {
                    autoFocus: true,
                    className: "w-full bg-transparent border-none text-body-md focus:ring-0 placeholder:text-on-surface-variant/60 transition-all duration-300",
                    defaultValue: query,
                    "data-search-target": searchTarget,
                    maxLength: 50,
                    placeholder: "搜索内容、创作者或话题",
                    type: "search"
                })
            )
        ),
        createElement("div", { className: "mt-4 inline-flex rounded-full border border-white/60 bg-white/30 p-1 shadow-sm" },
            renderSearchTargetButton("content", "article", "内容", query, searchTarget),
            renderSearchTargetButton("author", "person", "作者", query, searchTarget)
        )
    );
}

function renderSearchTargetButton(target: SearchTarget, icon: string, label: string, query: string, activeTarget: SearchTarget) {
    const active = target === activeTarget;

    return createElement("a", {
        "aria-label": label,
        "aria-pressed": active ? "true" : "false",
        className: `inline-flex min-h-9 items-center gap-2 rounded-full px-4 text-[13px] font-semibold transition-all duration-200 ${active ? "bg-white/70 text-primary shadow-sm" : "text-on-surface-variant hover:bg-white/45 hover:text-primary"}`,
        href: getSearchHref(query, target),
        role: "button"
    },
        createElement("span", { className: "material-symbols-outlined text-[18px]" }, icon),
        label
    );
}

function renderSearchState(query: string, searchTarget: SearchTarget, viewState: SearchViewState) {
    if (!query) {
        return createElement(PageState, {
            state: "empty",
            title: "输入关键词开始搜索",
            description: "支持搜索内容、作者或话题。",
            className: "glass-panel rounded-3xl p-5"
        });
    }

    if (viewState.status === "invalid") {
        return createElement(PageState, {
            state: "error",
            title: "关键词最多 50 字",
            description: "请缩短关键词后重试。",
            className: "glass-panel rounded-3xl p-5"
        });
    }

    if (viewState.status === "loading") {
        return createElement(PageState, {
            state: "loading",
            title: "正在搜索",
            description: searchTarget === "author" ? "正在检索作者。" : "正在检索内容。",
            className: "glass-panel rounded-3xl p-5"
        });
    }

    if (viewState.status === "error") {
        return createElement(PageState, {
            state: "error",
            title: "搜索失败",
            description: "请稍后重试。",
            className: "glass-panel rounded-3xl p-5"
        });
    }

    if (viewState.status !== "ready") {
        return null;
    }

    if (viewState.contents.length === 0 && viewState.users.length === 0) {
        return createElement(PageState, {
            state: "empty",
            title: "没有找到相关结果",
            description: `换个关键词搜索${searchTarget === "author" ? "作者" : "内容"}“${query}”。`,
            className: "glass-panel rounded-3xl p-5"
        });
    }

    return createElement("div", { className: "flex flex-col gap-6" },
        viewState.contents.length > 0 ? createElement("section", { className: "flex flex-col gap-6" },
            createElement("div", { className: "glass-panel rounded-3xl p-5 hover-lift shine-effect" },
                createElement("h2", { className: "font-headline-md text-on-surface" }, "内容"),
                createElement("p", { className: "mt-1 text-[14px] leading-6 text-on-surface-variant" }, `与“${query}”相关的内容。`)
            ),
            ...viewState.contents.map(renderFeedCard)
        ) : null,
        viewState.users.length > 0 ? createElement("section", { className: "grid gap-3" },
            createElement("div", { className: "glass-panel rounded-3xl p-5 hover-lift shine-effect" },
                createElement("h2", { className: "font-headline-md text-on-surface" }, "作者"),
                createElement("p", { className: "mt-1 text-[14px] leading-6 text-on-surface-variant" }, "可能相关的作者。")
            ),
            ...viewState.users.map(renderUserResult)
        ) : null
    );
}

function getSearchTarget(type: string | null): SearchTarget {
    return type === "author" ? "author" : "content";
}

function getSearchHref(query: string, target: SearchTarget) {
    const params = new URLSearchParams();
    if (query) {
        params.set("q", query);
    }
    if (target !== "content") {
        params.set("type", target);
    }

    return `/search?${params.toString()}`;
}

function renderUserResult(item: SearchUserItem) {
    const userId = String(item.user_id);
    const nickname = item.nickname || `用户 ${userId}`;

    return createElement("article", {
        className: "glass-panel hover-lift shine-effect rounded-3xl p-5",
        key: `user-${item.user_id}`
    },
        createElement("div", { className: "relative z-20 flex min-w-0 items-center justify-between gap-4" },
            createElement("a", { className: "flex min-w-0 items-center gap-3", href: `/user/${userId}` },
                renderUserAvatar(
                    { avatar: item.avatar, nickname, userId },
                    "h-12 w-12 shrink-0 rounded-full border border-white object-cover shadow-sm",
                    { alt: nickname, textClassName: "text-[15px]" }
                ),
                createElement("span", { className: "min-w-0" },
                    createElement("span", { className: "block break-words font-headline-md text-[15px] text-on-surface" }, nickname),
                    createElement("span", { className: "mt-1 block line-clamp-2 break-words text-[13px] leading-5 text-on-surface-variant" }, item.bio || "这个创作者还没有简介。")
                )
            ),
            createElement("button", {
                className: `glass-button-ghost rounded-full px-4 py-2 font-label-sm active:scale-95 transition-all duration-300 ${item.is_following ? "text-on-surface-variant" : "text-primary"}`,
                "data-user-id": userId,
                type: "button"
            }, item.is_following ? "已关注" : "关注")
        )
    );
}
