import { createElement, useEffect, useState } from "react";
import { getContentDetail, listComments } from "../../runtime/apiClient";
import { getContentScene } from "../contentScene";

type Props = {
    contentId: number;
    open: boolean;
    onClose: () => void;
};

type ContentInfo = {
    detail?: {
        content_id?: number;
        content_type?: number;
        author_id?: number;
        author_name?: string;
        author_avatar?: string;
        title?: string;
        description?: string;
        cover_url?: string;
        article_content?: string;
        video_url?: string;
        video_duration?: number;
        published_at?: number;
        like_count?: number;
        favorite_count?: number;
        comment_count?: number;
    };
};

type CommentItem = {
    comment_id?: number;
    user_id?: number;
    user_name?: string;
    user_avatar?: string;
    comment?: string;
    created_at?: number;
    status?: number;
};

type CommentsRes = {
    comments?: CommentItem[];
    has_more?: boolean;
};

export function ContentPreviewModal({ contentId, open, onClose }: Props) {
    const [detail, setDetail] = useState<ContentInfo | null>(null);
    const [comments, setComments] = useState<CommentItem[]>([]);
    const [loading, setLoading] = useState(true);
    const [tab, setTab] = useState<"detail" | "comments">("detail");

    useEffect(() => {
        if (!open) return;
        setLoading(true);
        setTab("detail");
        Promise.all([
            getContentDetail<ContentInfo>({ content_id: String(contentId) }).catch(() => null),
            listComments<CommentsRes>({ content_id: String(contentId), scene: getContentScene(), cursor: 0, page_size: 30 })
                .then((r) => r?.comments ?? []).catch(() => []),
        ]).then(([d, c]) => {
            setDetail(d);
            setComments(c);
            setLoading(false);
        });
    }, [contentId, open]);

    if (!open) return null;

    const d = detail?.detail;
    const typeLabel = d?.content_type === 10 ? "文章" : d?.content_type === 20 ? "视频" : "其他";

    return createElement("div", {
        className: "fixed inset-0 z-50 flex items-center justify-center bg-black/50",
        onClick: onClose,
    },
        createElement("div", {
            className: "bg-white rounded-xl shadow-2xl w-full max-w-2xl mx-4 max-h-[85vh] flex flex-col",
            onClick: (e: Event) => e.stopPropagation(),
        },
            // Header
            createElement("div", { className: "flex items-center justify-between px-5 py-4 border-b border-slate-200 shrink-0" },
                createElement("div", { className: "flex gap-3" },
                    createElement("button", {
                        onClick: () => setTab("detail"),
                        className: `text-sm font-medium pb-1 border-b-2 transition-colors ${tab === "detail" ? "text-blue-600 border-blue-600" : "text-slate-500 border-transparent hover:text-slate-700"}`,
                    }, "内容详情"),
                    createElement("button", {
                        onClick: () => setTab("comments"),
                        className: `text-sm font-medium pb-1 border-b-2 transition-colors ${tab === "comments" ? "text-blue-600 border-blue-600" : "text-slate-500 border-transparent hover:text-slate-700"}`,
                    }, `评论 (${comments.length})`),
                ),
                createElement("button", {
                    onClick: onClose,
                    className: "text-slate-400 hover:text-slate-600 text-xl leading-none",
                }, "✕"),
            ),

            // Body
            createElement("div", { className: "flex-1 overflow-y-auto p-5 space-y-4" },
                loading
                    ? createElement("p", { className: "text-sm text-slate-400" }, "加载中...")
                    : tab === "detail"
                        ? createElement("div", { className: "space-y-4" },
                            // Meta
                            createElement("div", { className: "flex items-center gap-3" },
                                d?.author_avatar
                                    ? createElement("img", { src: d.author_avatar, className: "w-8 h-8 rounded-full" })
                                    : createElement("div", { className: "w-8 h-8 rounded-full bg-slate-200" }),
                                createElement("div", null,
                                    createElement("span", { className: "text-sm font-medium text-slate-800" }, d?.author_name || "未知作者"),
                                    createElement("span", { className: "ml-2 text-xs text-slate-400" },
                                        typeLabel,
                                        d?.published_at ? ` · ${new Date(d.published_at * 1000).toLocaleDateString("zh-CN")}` : "",
                                    ),
                                ),
                            ),

                            d?.title && createElement("h3", { className: "text-lg font-semibold text-slate-800" }, d.title),
                            d?.description && createElement("p", { className: "text-sm text-slate-600" }, d.description),

                            d?.cover_url && createElement("img", {
                                src: d.cover_url,
                                className: "w-full max-h-48 object-cover rounded-lg",
                            }),

                            d?.article_content && createElement("div", {
                                className: "text-sm text-slate-700 leading-relaxed whitespace-pre-wrap max-h-64 overflow-y-auto bg-slate-50 rounded-lg p-4",
                            }, d.article_content),

                            d?.video_url && createElement("video", {
                                src: d.video_url,
                                controls: true,
                                className: "w-full rounded-lg max-h-64",
                            }),

                            // Stats
                            createElement("div", { className: "flex gap-4 text-sm text-slate-500" },
                                createElement("span", null, `👍 ${d?.like_count ?? 0}`),
                                createElement("span", null, `⭐ ${d?.favorite_count ?? 0}`),
                                createElement("span", null, `💬 ${d?.comment_count ?? 0}`),
                            ),
                        )
                        : createElement("div", { className: "space-y-3" },
                            comments.length === 0
                                ? createElement("p", { className: "text-sm text-slate-400" }, "暂无评论")
                                : comments.map((c, i) =>
                                    createElement("div", {
                                        key: c.comment_id ?? i,
                                        className: "bg-slate-50 rounded-lg p-3",
                                    },
                                        createElement("div", { className: "flex items-center gap-2 mb-2" },
                                            c.user_avatar
                                                ? createElement("img", { src: c.user_avatar, className: "w-6 h-6 rounded-full" })
                                                : createElement("div", { className: "w-6 h-6 rounded-full bg-slate-300" }),
                                            createElement("span", { className: "text-sm font-medium text-slate-700" }, c.user_name || `用户${c.user_id}`),
                                            c.status === 20 &&
                                                createElement("span", { className: "text-xs text-red-500 bg-red-50 px-1.5 py-0.5 rounded" }, "已删除"),
                                        ),
                                        createElement("p", { className: "text-sm text-slate-600" }, c.comment),
                                    ),
                                ),
                        ),
            ),

            // Footer
            createElement("div", { className: "flex gap-2 px-5 py-3 border-t border-slate-200 shrink-0" },
                createElement("a", {
                    href: `/content/${contentId}`,
                    target: "_blank",
                    className: "flex-1 text-center px-4 py-2 text-sm font-medium text-blue-600 bg-blue-50 rounded-lg hover:bg-blue-100 transition-colors",
                }, "前台查看"),
                createElement("button", {
                    onClick: onClose,
                    className: "px-4 py-2 text-sm font-medium text-slate-600 bg-slate-100 rounded-lg hover:bg-slate-200 transition-colors",
                }, "关闭"),
            ),
        ),
    );
}
