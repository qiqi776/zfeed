import { createElement, useEffect, useState } from "react";
import { getRecommendFeed } from "../runtime/apiClient";
import { readAuthSession } from "../runtime/authStore";
import { useContentInteractionVersion } from "../runtime/contentInteractionStore";
import { PageShell } from "../runtime/PageShell";
import { renderUserAvatar } from "./avatar";
import { cacheHomeRailItems, homeStyles, renderFeedCard, renderHomeHeader, renderLeftRail, renderRightRail } from "./feedShell";
import { PageState } from "./PageState";
import { sharedGlassBodyClass } from "./sharedGlassStyles";

type RecommendFeedItem = {
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

type RecommendFeedResponse = {
    items?: RecommendFeedItem[];
    snapshot_id?: string;
};

type RecommendFeedState =
    | { status: "loading" }
    | { status: "ready"; items: RecommendFeedItem[]; snapshotId?: string }
    | { status: "empty" }
    | { status: "error" };

const recommendFeedPageSize = 20;

export function HomePage() {
    const [state, setState] = useState<RecommendFeedState>({ status: "loading" });
    useContentInteractionVersion();

    useEffect(() => {
        let isCurrent = true;

        getRecommendFeed<RecommendFeedResponse>({ cursor: "", page_size: recommendFeedPageSize })
            .then((response) => {
                if (!isCurrent) {
                    return;
                }

                const items = response.items ?? [];
                cacheHomeRailItems(items);
                setState(items.length > 0
                    ? { status: "ready", items, snapshotId: response.snapshot_id }
                    : { status: "empty" });
            })
            .catch(() => {
                if (isCurrent) {
                    setState({ status: "error" });
                }
            });

        return () => {
            isCurrent = false;
        };
    }, []);

    return createElement(
        PageShell,
        { title: "zfeed - 推荐信息流", htmlClass: "light", bodyClass: sharedGlassBodyClass, styles: homeStyles },
        createElement("div", { className: "page-root" },
            renderHomeHeader("home"),
            createElement("div", { className: "pt-24 px-4 md:px-6 max-w-[1600px] mx-auto pb-safe" },
                createElement("div", { className: "grid grid-cols-4 md:grid-cols-8 lg:grid-cols-12 gap-5" },
                    renderLeftRail("home"),
                    createElement("main", { className: "feed-transition col-span-4 md:col-span-8 lg:col-span-7 flex flex-col gap-6 pb-24 feed-ready" },
                        renderComposer(),
                        renderRecommendState(state)
                    ),
                    renderRightRail(state.status === "ready" ? state.items : [])
                )
            )
        )
    );
}

function renderComposer() {
    return createElement("div", { className: "glass-panel rounded-2xl p-5 hover-lift shine-effect" },
        createElement("div", { className: "flex gap-4 items-start mb-4" },
            renderCurrentUserAvatar("w-10 h-10 rounded-full border border-white bg-white/55 shadow-sm object-cover", "text-[13px]"),
            createElement("div", { className: "composer-shell" },
                createElement("input", { className: "w-full min-h-11 bg-transparent border-none text-body-lg focus:ring-0 placeholder:text-on-surface-variant/60 transition-all duration-300", placeholder: "分享你的想法、灵感或最新动态...", type: "text" })
            )
        ),
        createElement("div", { className: "flex items-center justify-between mt-2 pt-4 border-t border-white/30" },
            createElement("div", { className: "flex flex-wrap gap-2" },
                renderComposerButton("article", "文字", "text-primary"),
                renderComposerButton("image", "图片", "text-emerald-600"),
                renderComposerButton("link", "链接", "text-on-surface-variant"),
                renderComposerButton("play_circle", "视频", "text-purple-600")
            )
        )
    );
}

function renderComposerButton(icon: string, label: string, colorClass: string) {
    return createElement("button", { className: `glass-button-ghost rounded-full px-4 py-2 flex items-center gap-2 ${colorClass} font-label-sm active:scale-95`, type: "button" },
        createElement("span", { className: "material-symbols-outlined text-[18px]" }, icon),
        label
    );
}

function renderRecommendState(state: RecommendFeedState) {
    if (state.status === "loading") {
        return createElement(PageState, {
            state: "loading",
            title: "正在加载推荐流",
            description: "正在获取最新推荐内容。"
        });
    }

    if (state.status === "empty") {
        return createElement(PageState, {
            state: "empty",
            title: "暂时没有内容",
            description: "稍后刷新，或去搜索页看看大家正在讨论什么。",
            actionHref: "/search",
            actionLabel: "去搜索"
        });
    }

    if (state.status === "error") {
        return createElement(PageState, {
            state: "error",
            title: "推荐流加载失败",
            description: "请稍后重试。"
        });
    }

    return createElement("div", { className: "flex flex-col gap-6", "data-snapshot-id": state.snapshotId ?? "" },
        ...state.items.map(renderFeedCard)
    );
}

function renderCurrentUserAvatar(className: string, textClassName: string) {
    const session = readAuthSession();
    if (!session?.user) {
        return createElement("div", { className });
    }

    return renderUserAvatar(
        { avatar: session.user.avatar, nickname: session.user.nickname, userId: session.user.userId },
        className,
        { alt: "用户头像", textClassName }
    );
}
