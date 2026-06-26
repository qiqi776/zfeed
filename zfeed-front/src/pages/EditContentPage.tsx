import { createElement, useEffect, useState } from "react";
import { getEditableContentDetail } from "../runtime/apiClient";
import { PageShell } from "../runtime/PageShell";
import { readAuthSession } from "../runtime/authStore";
import type { AuthSession } from "../runtime/authStore";
import { PageState } from "./PageState";
import { sharedGlassBodyClass, sharedGlassStyles } from "./sharedGlassStyles";

type ApiContentDetail = {
    content_id: string | number;
    content_type: number;
    title: string;
    description?: string;
    cover_url?: string;
    article_content?: string;
    video_url?: string;
    video_duration?: number;
};

type ContentDetailResponse = {
    detail?: ApiContentDetail;
};

type EditContentState =
    | { status: "auth-required" }
    | { status: "loading" }
    | { status: "ready"; detail: ApiContentDetail }
    | { status: "error" };

export function EditContentPage() {
    const [session] = useState<AuthSession | null>(() => readAuthSession());
    const contentId = getEditableContentId(window.location.pathname);
    const [state, setState] = useState<EditContentState>(() => {
        if (!session) {
            return { status: "auth-required" };
        }

        return contentId ? { status: "loading" } : { status: "error" };
    });

    useEffect(() => {
        if (!session) {
            return;
        }

        if (!contentId) {
            return;
        }

        let cancelled = false;
        getEditableContentDetail<ContentDetailResponse>({ content_id: contentId })
            .then((response) => {
                if (cancelled) {
                    return;
                }

                if (!response.detail) {
                    setState({ status: "error" });
                    return;
                }

                setState({ status: "ready", detail: response.detail });
            })
            .catch(() => {
                if (!cancelled) {
                    setState({ status: "error" });
                }
            });

        return () => {
            cancelled = true;
        };
    }, [contentId, session]);

    return createElement(
        PageShell,
        {
            title: "zfeed - 编辑内容",
            htmlClass: "light",
            bodyClass: sharedGlassBodyClass,
            styles: sharedGlassStyles
        },
        createElement("div", { className: "page-root" },
            createElement("main", { className: "min-h-screen px-4 py-6 md:px-6 md:py-10" },
                createElement("section", { className: "auth-sheet feed-transition feed-ready mx-auto w-full max-w-3xl rounded-[28px] p-6 md:p-8" },
                    createElement("a", { className: "text-label-sm text-primary", href: contentId ? `/content/${contentId}` : "/home" }, contentId ? "返回详情" : "返回首页"),
                    createElement("h1", { className: "mt-5 break-words font-display text-[34px] leading-tight text-on-surface" }, "编辑内容"),
                    renderState(state, contentId)
                )
            )
        )
    );
}

function renderState(state: EditContentState, contentId: string) {
    if (state.status === "auth-required") {
        return createElement(PageState, {
            state: "auth-required",
            description: "登录后才能编辑内容。",
            actionHref: `/login?next=${encodeURIComponent(window.location.pathname)}`,
            actionLabel: "去登录"
        });
    }

    if (state.status === "loading") {
        return createElement(PageState, {
            state: "loading",
            title: "正在加载编辑内容",
            description: "正在读取原文内容，请稍候。"
        });
    }

    if (state.status === "error") {
        return createElement(PageState, {
            state: "error",
            title: "编辑内容加载失败",
            description: "内容不存在，或你没有编辑权限。",
            actionHref: contentId ? `/content/${contentId}` : "/home",
            actionLabel: contentId ? "返回详情" : "返回首页"
        });
    }

    return renderEditForm(state.detail);
}

function renderEditForm(detail: ApiContentDetail) {
    const contentId = String(detail.content_id);
    const isVideo = detail.content_type === 2;

    return createElement("form", {
        className: "mt-6 grid gap-4",
        "data-content-edit": "true",
        "data-content-id": contentId,
        "data-content-type": isVideo ? "video" : "article"
    },
        createElement("input", { defaultValue: detail.cover_url ?? "", name: "cover", type: "hidden" }),
        createElement("input", { defaultValue: detail.video_url ?? "", name: "videoUrl", type: "hidden" }),
        createElement("input", { defaultValue: String(detail.video_duration ?? 0), name: "duration", type: "hidden" }),
        createElement("label", { className: "grid gap-2" },
            createElement("span", { className: "auth-label" }, "标题"),
            createElement("input", {
                className: "auth-field rounded-2xl px-4 py-3 text-[15px] text-on-surface",
                defaultValue: detail.title,
                maxLength: 100,
                name: "title",
                type: "text"
            })
        ),
        createElement("label", { className: "grid gap-2" },
            createElement("span", { className: "auth-label" }, "摘要"),
            createElement("textarea", {
                className: "auth-field min-h-28 resize-none rounded-2xl px-4 py-3 text-[15px] text-on-surface",
                defaultValue: detail.description ?? "",
                maxLength: isVideo ? 500 : 255,
                name: "description",
                rows: 4
            })
        ),
        createElement("label", { className: "grid gap-2" },
            createElement("span", { className: "auth-label" }, isVideo ? "视频说明" : "正文"),
            createElement("textarea", {
                className: "auth-field min-h-56 resize-none rounded-2xl px-4 py-3 text-[15px] text-on-surface",
                defaultValue: isVideo ? detail.video_url ?? "" : detail.article_content ?? "",
                maxLength: isVideo ? 500 : 1000000,
                name: "content",
                rows: 10
            })
        ),
        createElement("div", { className: "flex flex-wrap items-center justify-end gap-3" },
            createElement("a", {
                className: "glass-button-ghost rounded-full px-5 py-3 text-primary font-label-sm",
                href: `/content/${contentId}`
            }, "取消"),
            createElement("button", {
                className: "glass-button-primary inline-flex min-h-11 items-center justify-center rounded-full px-5 py-3 text-white font-label-sm",
                type: "button"
            }, "保存修改")
        )
    );
}

function getEditableContentId(pathname: string) {
    const match = pathname.match(/^\/content\/([^/]+)\/edit$/);
    return match?.[1] ?? "";
}
