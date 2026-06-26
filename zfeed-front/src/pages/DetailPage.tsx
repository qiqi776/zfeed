import { createElement, useEffect, useState } from "react";
import { getContentDetail, listComments } from "../runtime/apiClient";
import { readAuthSession } from "../runtime/authStore";
import { mergeContentInteractionState, useContentInteractionVersion } from "../runtime/contentInteractionStore";
import { PageShell } from "../runtime/PageShell";
import { renderUserAvatar } from "./avatar";
import { getContentScene } from "./contentScene";
import { getCachedHomeRailItems, homeStyles, renderHomeHeader, renderLeftRail, renderRightRail } from "./feedShell";
import { PageState } from "./PageState";

type ApiContentDetail = {
    content_id: string | number;
    content_type: number;
    author_id: string | number;
    author_name: string;
    author_avatar?: string;
    title: string;
    description?: string;
    cover_url?: string;
    article_content?: string;
    video_url?: string;
    video_duration?: number;
    published_at?: number;
    like_count?: number;
    favorite_count?: number;
    comment_count?: number;
    is_liked?: boolean;
    is_favorited?: boolean;
    is_following_author?: boolean;
};

type ContentDetailResponse = {
    detail?: ApiContentDetail;
};

type ApiCommentItem = {
    comment_id: string | number;
    content_id?: string | number;
    user_id: string | number;
    reply_to_user_id?: string | number;
    parent_id?: string | number;
    root_id?: string | number;
    comment: string;
    created_at?: number;
    user_name?: string;
    user_avatar?: string;
    nickname?: string;
    avatar?: string;
    reply_count?: number;
};

type CommentListResponse = {
    comments?: ApiCommentItem[];
};

type DynamicDetailState =
    | { status: "loading" }
    | { status: "error" }
    | { status: "not-found" }
    | { status: "ready"; detail: ApiContentDetail };

type CommentListState =
    | { status: "loading" }
    | { status: "ready"; comments: ApiCommentItem[] }
    | { status: "error" };

const detailBodyClass = "text-on-surface font-body-md antialiased overflow-x-hidden selection:bg-primary-container selection:text-on-primary-container";
const styles = "/* Liquid Glass Utility Classes */\r\n        .glass-panel {\r\n            background: rgba(255, 255, 255, 0.4);\r\n            backdrop-filter: blur(40px);\r\n            -webkit-backdrop-filter: blur(40px);\r\n            border: 1px solid rgba(255, 255, 255, 0.6);\r\n            border-top: 1px solid rgba(255, 255, 255, 0.8); /* Specular highlight */\r\n            box-shadow: inset 0 1px 2px rgba(255, 255, 255, 0.8), \r\n                        0 4px 20px rgba(0, 0, 0, 0.05);\r\n        }\r\n        \r\n        .glass-input {\r\n            background: rgba(255, 255, 255, 0.5);\r\n            backdrop-filter: blur(20px);\r\n            border: 1px solid rgba(255, 255, 255, 0.7);\r\n            box-shadow: inset 0 1px 3px rgba(0,0,0,0.02);\r\n            transition: all 0.3s ease-out;\r\n        }\r\n        \r\n        .glass-input:focus {\n            outline: none;\n            border-color: var(--color-primary, #1f53c9);\n            box-shadow: 0 0 0 2px rgba(31, 83, 201, 0.2), inset 0 1px 3px rgba(0,0,0,0.02);\n        }\n\n        .composer-shell {\n            position: relative;\n            flex: 1;\n            overflow: hidden;\n            border-radius: 1.25rem;\n            background: rgba(255, 255, 255, 0.34);\n            backdrop-filter: blur(18px);\n            -webkit-backdrop-filter: blur(18px);\n            border: 1px solid rgba(255, 255, 255, 0.62);\n            box-shadow: inset 0 1px 2px rgba(255, 255, 255, 0.72), 0 4px 16px rgba(0, 0, 0, 0.035);\n            transition: transform 0.3s cubic-bezier(0.4, 0, 0.2, 1), background 0.3s ease-out, border-color 0.3s ease-out, box-shadow 0.3s ease-out;\n        }\n\n        .composer-shell:hover,\n        .composer-shell:focus-within {\n            transform: translateY(-2px);\n            background: rgba(255, 255, 255, 0.58);\n            border-color: rgba(255, 255, 255, 0.9);\n            box-shadow: inset 0 1px 2px rgba(255, 255, 255, 0.9), 0 10px 28px rgba(31, 83, 201, 0.12);\n        }\n\n        .composer-shell:focus-within {\n            border-color: rgba(31, 83, 201, 0.50);\n            box-shadow: 0 0 0 3px rgba(31, 83, 201, 0.14), inset 0 1px 2px rgba(255, 255, 255, 0.95), 0 12px 30px rgba(31, 83, 201, 0.16);\n        }\n\n        .composer-shell input {\n            position: relative;\n            z-index: 20;\n            min-height: 44px;\n            padding: 0.65rem 0.95rem;\n        }\n\n        .composer-shell input:focus {\n            outline: none;\n            box-shadow: none;\n        }\n\n        .glass-button-primary {\n            background: linear-gradient(135deg, rgba(64, 109, 228, 0.9) 0%, rgba(31, 83, 201, 0.9) 100%);\r\n            backdrop-filter: blur(10px);\r\n            box-shadow: 0 4px 15px rgba(31, 83, 201, 0.3), inset 0 1px 1px rgba(255, 255, 255, 0.4);\r\n            border: 1px solid rgba(255, 255, 255, 0.2);\r\n            transition: all 0.3s ease-out;\r\n        }\r\n        \r\n        .glass-button-primary:hover {\r\n            transform: translateY(-1px);\r\n            box-shadow: 0 6px 20px rgba(31, 83, 201, 0.4), inset 0 1px 1px rgba(255, 255, 255, 0.5);\r\n        }\r\n\r\n        .glass-button-ghost {\r\n            background: rgba(255, 255, 255, 0.3);\r\n            backdrop-filter: blur(10px);\r\n            border: 1px solid rgba(255, 255, 255, 0.6);\r\n            transition: all 0.3s ease-out;\r\n        }\r\n\r\n        .glass-button-ghost:hover {\r\n            background: rgba(255, 255, 255, 0.5);\r\n            border-color: rgba(255, 255, 255, 0.8);\r\n        }\r\n\r\n        .hover-lift {\r\n            transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);\r\n        }\r\n        .hover-lift:hover {\r\n            transform: translateY(-4px);\r\n            box-shadow: inset 0 1px 2px rgba(255, 255, 255, 0.9), 0 12px 30px rgba(0, 0, 0, 0.08);\r\n            border-color: rgba(255, 255, 255, 0.9);\r\n        }\r\n\r\n        .shine-effect {\r\n            position: relative;\r\n            overflow: hidden;\r\n        }\r\n        .shine-effect::after {\r\n            content: '';\r\n            position: absolute;\r\n            top: 0;\r\n            left: -150%;\r\n            width: 50%;\r\n            height: 100%;\r\n            background: linear-gradient(to right, rgba(255,255,255,0) 0%, rgba(255,255,255,0.4) 50%, rgba(255,255,255,0) 100%);\r\n            transform: skewX(-25deg);\r\n            transition: left 0.7s ease;\r\n            z-index: 10;\r\n            pointer-events: none;\r\n        }\r\n        .shine-effect:hover::after {\r\n            left: 200%;\r\n        }\r\n\r\n        body {\n            background-color: #eef2f6;\n            background-image: \n                radial-gradient(circle at 15% 50%, rgba(180, 197, 255, 0.4) 0%, transparent 50%),\n                radial-gradient(circle at 85% 30%, rgba(219, 226, 250, 0.5) 0%, transparent 50%);\n            background-attachment: fixed;\n            min-height: 100vh;\n        }\n\n        .feed-transition {\n            opacity: 0;\n            transform: translateY(8px) scale(0.995);\n            filter: blur(8px);\n            transition:\n                opacity 0.28s ease,\n                transform 0.34s cubic-bezier(0.22, 1, 0.36, 1),\n                filter 0.34s ease;\n        }\n\n        .feed-transition.feed-ready {\n            opacity: 1;\n            transform: none;\n            filter: none;\n        }\n\n        .detail-prose p {\n            margin-bottom: 1rem;\n            color: rgba(67, 70, 84, 0.96);\n            font-size: 16px;\n            line-height: 1.82;\n        }\n\n        .detail-video-shell {\n            background:\n                radial-gradient(circle at 24% 18%, rgba(255,255,255,0.42), transparent 30%),\n                linear-gradient(135deg, rgba(31,83,201,0.82), rgba(88,96,128,0.78));\n            box-shadow: inset 0 1px 2px rgba(255,255,255,0.46), 0 20px 48px rgba(31,83,201,0.22);\n        }\n\n        .comment-row {\n            background: rgba(255,255,255,0.28);\n            border: 1px solid rgba(255,255,255,0.42);\n            box-shadow: inset 0 1px 1px rgba(255,255,255,0.62);\n        }";

export function DetailPage() {
    const contentId = parseContentId(window.location.pathname);
    return createElement(DynamicContentDetailPage, { contentId, key: contentId });
}

function DynamicContentDetailPage({ contentId }: { contentId: string }) {
    const [state, setState] = useState<DynamicDetailState>({ status: "loading" });
    const [commentsState, setCommentsState] = useState<CommentListState>({ status: "loading" });
    useContentInteractionVersion();

    useEffect(() => {
        let isCurrent = true;

        getContentDetail<ContentDetailResponse>({ content_id: contentId })
            .then((response) => {
                if (!isCurrent) {
                    return;
                }

                if (!response.detail) {
                    setState({ status: "not-found" });
                    setCommentsState({ status: "ready", comments: [] });
                    return;
                }

                setState({ status: "ready", detail: response.detail });

                return listComments<CommentListResponse>({
                    content_id: String(response.detail.content_id),
                    scene: getContentScene(response.detail.content_type),
                    cursor: 0,
                    page_size: 20
                })
                    .then((commentResponse) => {
                        if (isCurrent) {
                            setCommentsState({ status: "ready", comments: commentResponse.comments ?? [] });
                        }
                    })
                    .catch(() => {
                        if (isCurrent) {
                            setCommentsState({ status: "error" });
                        }
                    });
            })
            .catch(() => {
                if (isCurrent) {
                    setState({ status: "error" });
                    setCommentsState({ status: "ready", comments: [] });
                }
            });

        return () => {
            isCurrent = false;
        };
    }, [contentId]);

    const title = state.status === "ready" ? `zfeed - ${state.detail.title}` : "zfeed - 内容详情";

    return createElement(
        PageShell,
        { title, htmlClass: "light", bodyClass: detailBodyClass, styles: `${homeStyles}\n${styles}` },
        createElement("div", { className: "page-root" },
            renderHomeHeader("home"),
            createElement("div", { className: "pt-24 px-4 md:px-6 max-w-[1600px] mx-auto pb-safe" },
                createElement("div", { className: "grid grid-cols-4 md:grid-cols-8 lg:grid-cols-12 gap-5" },
                    renderLeftRail("home"),
                    createElement("main", { className: "feed-transition col-span-4 md:col-span-8 lg:col-span-7 flex flex-col gap-6 pb-24 feed-ready" },
                        createElement("section", { className: "glass-panel rounded-3xl p-5 md:p-8" },
                            createElement("a", { className: "font-label-sm text-primary", href: "/home" }, "返回首页"),
                            renderDynamicDetailState(state, commentsState)
                        )
                    ),
                    renderRightRail(getCachedHomeRailItems())
                )
            )
        )
    );
}

function renderDynamicDetailState(state: DynamicDetailState, commentsState: CommentListState) {
    if (state.status === "loading") {
        return createElement(PageState, {
            state: "loading",
            title: "正在加载内容",
            description: "正在获取内容详情。"
        });
    }

    if (state.status === "error") {
        return createElement(PageState, {
            state: "error",
            title: "内容加载失败",
            description: "请稍后重试。"
        });
    }

    if (state.status === "not-found") {
        return createElement(PageState, {
            state: "empty",
            title: "内容不存在",
            description: "这篇内容可能已删除或不可见。"
        });
    }

    const detail = mergeContentInteractionState(withSyncedCommentCount(state.detail, commentsState));

    return createElement("div", { className: "mt-6" },
        renderDynamicDetail(detail),
        renderCommentSection(detail, commentsState)
    );
}

function withSyncedCommentCount(detail: ApiContentDetail, commentsState: CommentListState): ApiContentDetail {
    if (commentsState.status !== "ready") {
        return detail;
    }

    const syncedCount = Math.max(detail.comment_count ?? 0, commentsState.comments.length);
    return syncedCount === detail.comment_count ? detail : { ...detail, comment_count: syncedCount };
}

function renderDynamicDetail(detail: ApiContentDetail) {
    const contentId = String(detail.content_id);
    const authorId = String(detail.author_id);
    const scene = getContentScene(detail.content_type);
    const paragraphs = getArticleParagraphs(detail);
    const isVideo = scene === "VIDEO";

    return createElement("article", { "data-content-scene": scene },
        detail.cover_url
            ? createElement("img", {
                alt: detail.title,
                className: "mb-6 aspect-[16/9] w-full rounded-3xl border border-white/50 object-cover shadow-sm",
                src: detail.cover_url
            })
            : null,
        createElement("div", { className: "flex flex-wrap items-center justify-between gap-4" },
            createElement("div", { className: "flex min-w-0 items-center gap-3" },
                renderUserAvatar(
                    { avatar: detail.author_avatar, nickname: detail.author_name, userId: detail.author_id },
                    "h-11 w-11 shrink-0 rounded-full border border-white object-cover shadow-sm",
                    { alt: detail.author_name, textClassName: "text-[14px]" }
                ),
                createElement("div", { className: "min-w-0" },
                    createElement("div", { className: "truncate font-headline-md text-[15px] text-on-surface" }, detail.author_name),
                    createElement("div", { className: "font-meta-xs text-on-surface-variant" }, formatDetailDate(detail.published_at))
                )
            ),
            createElement("button", {
                className: `glass-button-ghost rounded-full px-4 py-2 font-label-sm active:scale-95 transition-all duration-300 ${detail.is_following_author ? "text-on-surface-variant" : "text-primary"}`,
                "data-user-id": authorId,
                type: "button"
            }, detail.is_following_author ? "已关注" : "关注作者")
        ),
        createElement("h1", { className: "mt-6 break-words font-display text-[34px] font-bold leading-tight tracking-normal text-on-surface md:text-[44px]" }, detail.title),
        detail.description
            ? createElement("p", { className: "mt-4 break-words text-[16px] leading-8 text-on-surface-variant" }, detail.description)
            : null,
        isVideo && detail.video_url
            ? createElement("a", {
                className: "detail-video-shell mt-6 flex min-h-[220px] items-center justify-center rounded-3xl text-white shadow-sm",
                href: detail.video_url
            },
                createElement("span", { className: "material-symbols-outlined text-[48px]" }, "play_circle"),
                createElement("span", { className: "sr-only" }, "播放视频")
            )
            : null,
        paragraphs.length > 0
            ? createElement("div", { className: "detail-prose mt-6 break-words" },
                paragraphs.map((paragraph, index) => createElement("p", { key: `${contentId}-${index}` }, paragraph))
            )
            : null,
        createElement("div", { className: "mt-8 flex flex-wrap items-center gap-3" },
            createElement("button", {
                "aria-label": detail.is_liked ? "取消点赞" : "点赞",
                className: `glass-button-ghost rounded-full px-4 py-2 flex items-center gap-2 font-label-sm active:scale-95 transition-all duration-300 ${detail.is_liked ? "text-error" : "text-on-surface-variant"}`,
                "data-content-id": contentId,
                "data-content-scene": scene,
                "data-content-user-id": authorId,
                type: "button"
            },
                createElement("span", { className: "material-symbols-outlined text-[18px]" }, "favorite"),
                createElement("span", null, formatCount(detail.like_count))
            ),
            createElement("button", {
                "aria-label": detail.is_favorited ? "取消收藏" : "收藏",
                className: `glass-button-ghost rounded-full px-4 py-2 flex items-center gap-2 font-label-sm active:scale-95 transition-all duration-300 ${detail.is_favorited ? "text-primary" : "text-on-surface-variant"}`,
                "data-content-id": contentId,
                "data-content-scene": scene,
                type: "button"
            },
                createElement("span", { className: "material-symbols-outlined text-[18px]" }, "bookmark"),
                createElement("span", null, formatCount(detail.favorite_count))
            ),
            createElement("div", { className: "glass-button-ghost rounded-full px-4 py-2 flex items-center gap-2 font-label-sm text-on-surface-variant" },
                createElement("span", { className: "material-symbols-outlined text-[18px]" }, "chat_bubble"),
                createElement("span", { "data-content-comment-count": contentId }, formatCount(detail.comment_count))
            )
        )
    );
}

function renderCommentSection(detail: ApiContentDetail, state: CommentListState) {
    const contentUserId = String(detail.author_id);
    const scene = getContentScene(detail.content_type);
    const comments = state.status === "ready" ? state.comments : [];
    const session = readAuthSession();
    const currentUser = session?.user;

    return createElement("section", { className: "mt-6 glass-panel rounded-3xl p-5", "data-content-scene": scene, "data-content-user-id": contentUserId },
        createElement("div", { className: "mb-4 flex items-center justify-between gap-4" },
            createElement("div", null,
                createElement("h2", { className: "font-headline-md text-on-surface" }, "评论"),
                createElement("p", { className: "mt-1 text-[14px] leading-6 text-on-surface-variant", "data-comment-discussion-count": String(detail.content_id) }, `${formatCount(detail.comment_count)} 条讨论`)
            )
        ),
        createElement("div", { className: "comment-row rounded-2xl p-4 mb-4", "data-content-user-id": contentUserId },
            createElement("div", { className: "flex gap-3" },
                renderUserAvatar(
                    { avatar: currentUser?.avatar, nickname: currentUser?.nickname ?? "我", userId: currentUser?.userId },
                    "h-10 w-10 shrink-0 rounded-full border border-white object-cover shadow-sm",
                    { alt: "我的头像", textClassName: "text-[13px]" }
                ),
                createElement("div", { className: "flex-1 min-w-0" },
                    createElement("div", { className: "composer-shell" },
                        createElement("input", {
                            className: "w-full bg-transparent border-none text-body-md focus:ring-0 placeholder:text-on-surface-variant/60 transition-all duration-300",
                            maxLength: 255,
                            placeholder: "写下你的观点，补充或提问...",
                            type: "text"
                        })
                    ),
                    createElement("div", { className: "mt-3 flex justify-end" },
                        createElement("button", { className: "glass-button-primary text-white font-label-sm px-4 py-2 rounded-full active:scale-95 transition-all duration-300", type: "button" }, "发送")
                    )
                )
            )
        ),
        createElement("div", { className: "flex flex-col gap-3" },
            state.status === "loading"
                ? createElement(PageState, { state: "loading", title: "正在加载评论", description: "正在获取讨论内容。" })
                : null,
            state.status === "error"
                ? createElement(PageState, { state: "error", title: "评论加载失败", description: "请稍后重试。" })
                : null,
            state.status === "ready" && comments.length === 0
                ? createElement(PageState, { state: "empty", title: "还没有评论", description: "写下第一条评论。" })
                : null,
            ...comments.map((comment) => renderCommentRow(comment, String(currentUser?.userId ?? "")))
        )
    );
}

function renderCommentRow(comment: ApiCommentItem, currentUserId: string) {
    const commentId = cleanId(comment.comment_id);
    const rootId = cleanId(comment.root_id) || commentId;
    const userId = cleanId(comment.user_id);
    const authorName = cleanText(comment.user_name) || cleanText(comment.nickname) || `用户 ${userId}`;
    const avatar = cleanText(comment.user_avatar) || cleanText(comment.avatar);
    const canDelete = Boolean(currentUserId && userId === currentUserId);

    return createElement("div", {
        className: "comment-row rounded-2xl p-4",
        "data-comment-author-name": authorName,
        "data-comment-id": commentId,
        "data-comment-user-id": userId,
        "data-root-id": rootId,
        key: commentId || `${userId}-${comment.created_at ?? comment.comment}`
    },
        createElement("div", { className: "flex gap-3" },
            renderUserAvatar(
                { avatar, nickname: authorName, userId },
                "h-10 w-10 shrink-0 rounded-full border border-white object-cover shadow-sm",
                { alt: authorName, textClassName: "text-[13px]" }
            ),
            createElement("div", { className: "min-w-0 flex-1" },
                createElement("div", { className: "flex flex-wrap items-center gap-2" },
                    createElement("div", { className: "font-headline-md text-[15px] text-on-surface" }, authorName),
                    createElement("span", { className: "font-meta-xs text-on-surface-variant" }, "·"),
                    createElement("span", { className: "font-meta-xs text-on-surface-variant" }, formatDetailDate(comment.created_at))
                ),
                createElement("p", { className: "mt-3 break-words font-body-md text-on-surface-variant" }, comment.comment),
                createElement("div", { className: "mt-3 flex flex-wrap items-center gap-2" },
                    createElement("button", {
                        "aria-label": "回复",
                        className: "glass-button-ghost rounded-full px-3 py-1.5 text-[12px] font-semibold text-primary active:scale-95",
                        type: "button"
                    },
                        createElement("span", { className: "material-symbols-outlined mr-1 align-[-3px] text-[15px]" }, "reply"),
                        "回复"
                    ),
                    canDelete
                        ? createElement("button", {
                            "aria-label": "删除评论",
                            className: "glass-button-ghost rounded-full px-3 py-1.5 text-[12px] font-semibold text-red-600 active:scale-95",
                            type: "button"
                        },
                            createElement("span", { className: "material-symbols-outlined mr-1 align-[-3px] text-[15px]" }, "delete"),
                            "删除"
                        )
                        : null
                )
            )
        )
    );
}

function parseContentId(pathname: string) {
    const raw = pathname.split("/")[2] ?? "";
    try {
        return decodeURIComponent(raw);
    } catch {
        return raw;
    }
}

function getArticleParagraphs(detail: ApiContentDetail) {
    return (detail.article_content ?? "")
        .split(/\r?\n/)
        .map((paragraph) => paragraph.trim())
        .filter(Boolean);
}

function formatDetailDate(publishedAt?: number) {
    if (!publishedAt) {
        return "刚刚";
    }

    return new Intl.DateTimeFormat("zh-CN", {
        year: "numeric",
        month: "2-digit",
        day: "2-digit"
    }).format(new Date(publishedAt * 1000));
}

function formatCount(count?: number) {
    return String(count ?? 0);
}

function cleanId(value: unknown) {
    return typeof value === "string" || typeof value === "number" ? String(value).trim() : "";
}

function cleanText(value: unknown) {
    return typeof value === "string" || typeof value === "number" ? String(value).trim() : "";
}
