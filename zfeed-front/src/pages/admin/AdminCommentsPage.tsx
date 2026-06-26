import { createElement, useEffect, useState } from "react";
import type { ChangeEvent, KeyboardEvent } from "react";
import { adminListComments, adminUpdateCommentStatus, searchContents } from "../../runtime/apiClient";
import { showToast } from "../../runtime/toast";
import { readAuthSession } from "../../runtime/authStore";
import { AdminLayout } from "./AdminLayout";
import { ConfirmDialog } from "./ConfirmDialog";

type AdminCommentItem = {
    comment_id: number;
    content_id: number;
    user_id: number;
    user_name: string;
    comment: string;
    status: number;
    created_at: number;
};

type ContentItem = { content_id: number; title: string; author_name: string };

type CommentsState =
    | { status: "loading" }
    | { status: "ready"; comments: AdminCommentItem[]; total: number }
    | { status: "error"; message: string };

export function AdminCommentsPage() {
    const session = readAuthSession();
    const [searchInput, setSearchInput] = useState("");
    const [filterContentId, setFilterContentId] = useState<number | null>(null);
    const [page, setPage] = useState(0);
    const pageSize = 20;
    const [state, setState] = useState<CommentsState>({ status: "loading" });

    // Content search (for finding a specific content to filter by)
    const [contentSearch, setContentSearch] = useState("");
    const [contents, setContents] = useState<ContentItem[]>([]);
    const [contentSearching, setContentSearching] = useState(false);

    // Confirm dialog for comment actions
    const [confirmOpen, setConfirmOpen] = useState(false);
    const [confirmCommentId, setConfirmCommentId] = useState<number | null>(null);
    const [confirmAction, setConfirmAction] = useState<"hide" | "restore" | "delete">("hide");

    const loadComments = (q: string, pageNum: number, contentId: number | null) => {
        setState({ status: "loading" });
        adminListComments<{ comments: AdminCommentItem[]; total_count: number }>({
            query: q.trim(),
            page: pageNum,
            page_size: pageSize,
            ...(contentId && contentId > 0 ? { content_id: contentId } : {}),
        }).then((res) => {
            setState({ status: "ready", comments: res.comments ?? [], total: res.total_count ?? 0 });
        }).catch((err) => {
            setState({ status: "error", message: err instanceof Error ? err.message : "加载失败" });
        });
    };

    useEffect(() => { loadComments(searchInput, page, filterContentId); }, [searchInput, page, filterContentId]);

    const handleSearch = () => { setPage(0); };

    const handleSearchContent = () => {
        if (!contentSearch.trim()) return;
        setContentSearching(true);
        searchContents<{ items: ContentItem[] }>({ query: contentSearch.trim(), page_size: 10, mode: "latest" })
            .then((r) => setContents(r.items ?? []))
            .catch(() => setContents([]))
            .finally(() => setContentSearching(false));
    };

    const handleFilterByContent = (contentId: number) => {
        setFilterContentId(contentId);
        setPage(0);
    };

    const clearContentFilter = () => {
        setFilterContentId(null);
        setPage(0);
    };

    const handleCommentAction = (commentId: number, action: "hide" | "restore" | "delete") => {
        setConfirmCommentId(commentId);
        setConfirmAction(action);
        setConfirmOpen(true);
    };

    const executeCommentAction = async () => {
        if (!confirmCommentId) return;
        const statusMap = { hide: 20, restore: 10, delete: 40 };
        try {
            await adminUpdateCommentStatus({ comment_id: confirmCommentId, status: statusMap[confirmAction] });
            loadComments(searchInput, page, filterContentId);
        } catch (err) {
            showToast(err instanceof Error ? err.message : "操作失败");
        } finally {
            setConfirmOpen(false);
        }
    };

    const actionLabels = {
        hide: { title: "隐藏评论", msg: "确定要隐藏该评论吗？用户将无法看到此评论。", label: "隐藏", danger: false },
        restore: { title: "恢复评论", msg: "确定要恢复该评论吗？评论将重新对用户可见。", label: "恢复", danger: false },
        delete: { title: "删除评论", msg: "确定要删除该评论吗？此操作不可撤销。", label: "删除", danger: true },
    };

    return createElement(AdminLayout, { currentPath: "/admin/comments", adminName: session?.user?.nickname },
        createElement(ConfirmDialog, {
            open: confirmOpen,
            title: actionLabels[confirmAction].title,
            message: actionLabels[confirmAction].msg,
            confirmLabel: actionLabels[confirmAction].label,
            danger: actionLabels[confirmAction].danger,
            onConfirm: executeCommentAction,
            onCancel: () => setConfirmOpen(false),
        }),
        createElement("div", { className: "space-y-5" },
            createElement("h2", { className: "text-xl font-bold text-slate-800" }, "💬 评论管理"),

            // Search bar
            createElement("div", { className: "flex gap-3 flex-wrap" },
                createElement("input", {
                    type: "text",
                    value: searchInput,
                    onChange: (e: ChangeEvent<HTMLInputElement>) => setSearchInput(e.target.value),
                    onKeyDown: (e: KeyboardEvent<HTMLInputElement>) => { if (e.key === "Enter") handleSearch(); },
                    placeholder: "搜索评论内容 / 用户名...",
                    className: "flex-1 max-w-md px-4 py-2 border border-slate-300 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-blue-500",
                }),
                createElement("button", {
                    onClick: handleSearch,
                    className: "px-5 py-2 bg-blue-600 text-white text-sm font-medium rounded-lg hover:bg-blue-700 transition-colors",
                }, "搜索"),
                filterContentId &&
                    createElement("button", {
                        onClick: clearContentFilter,
                        className: "px-3 py-2 text-sm text-slate-600 bg-slate-100 rounded-lg hover:bg-slate-200",
                    }, `✕ 内容 #${filterContentId}`),
            ),

            // Content filter section
            createElement("div", { className: "bg-white rounded-xl shadow-sm border border-slate-200 p-4" },
                createElement("p", { className: "text-xs font-medium text-slate-500 mb-2" }, "按内容筛选评论"),
                createElement("div", { className: "flex gap-2" },
                    createElement("input", {
                        type: "text",
                        value: contentSearch,
                        onChange: (e: ChangeEvent<HTMLInputElement>) => setContentSearch(e.target.value),
                        onKeyDown: (e: KeyboardEvent<HTMLInputElement>) => { if (e.key === "Enter") handleSearchContent(); },
                        placeholder: "搜索内容标题...",
                        className: "flex-1 max-w-xs px-3 py-1.5 border border-slate-300 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-blue-500",
                    }),
                    createElement("button", {
                        onClick: handleSearchContent,
                        disabled: contentSearching,
                        className: "px-3 py-1.5 text-xs bg-slate-100 text-slate-600 rounded-lg hover:bg-slate-200 disabled:opacity-50",
                    }, contentSearching ? "搜索中..." : "查找"),
                ),
                contents.length > 0 &&
                    createElement("div", { className: "mt-3 flex flex-wrap gap-1.5" },
                        contents.slice(0, 8).map((c) =>
                            createElement("button", {
                                key: c.content_id,
                                onClick: () => handleFilterByContent(c.content_id),
                                className: `px-2.5 py-1 text-xs rounded-full transition-colors ${filterContentId === c.content_id ? "bg-blue-100 text-blue-700 font-medium" : "bg-slate-50 text-slate-600 hover:bg-slate-100"}`,
                            }, `${c.title?.slice(0, 20) || "无标题"} (#${c.content_id})`),
                        ),
                    ),
            ),

            state.status === "ready" &&
                createElement("p", { className: "text-sm text-slate-500" }, `共 ${state.total} 条评论`),

            // Comment table
            createElement("div", { className: "bg-white rounded-xl shadow-sm border border-slate-200 overflow-hidden" },
                createElement("table", { className: "w-full text-sm" },
                    createElement("thead", { className: "bg-slate-50 text-slate-500 text-xs uppercase" },
                        createElement("tr", null,
                            ...[ "评论ID", "内容ID", "用户", "评论内容", "状态", "操作" ].map((h) =>
                                createElement("th", { key: h, className: "px-4 py-3 text-left font-medium" }, h),
                            ),
                        ),
                    ),
                    createElement("tbody", null,
                        state.status === "loading" &&
                            createElement("tr", null,
                                createElement("td", { colSpan: 6, className: "px-4 py-8 text-center text-slate-400" }, "加载中..."),
                            ),
                        state.status === "error" &&
                            createElement("tr", null,
                                createElement("td", { colSpan: 6, className: "px-4 py-8 text-center text-red-500" }, state.message),
                            ),
                        state.status === "ready" && state.comments.length === 0 &&
                            createElement("tr", null,
                                createElement("td", { colSpan: 6, className: "px-4 py-8 text-center text-slate-400" }, "暂无评论数据"),
                            ),
                        state.status === "ready" && state.comments.map((c) =>
                            createElement("tr", {
                                key: c.comment_id,
                                className: "border-t border-slate-100 hover:bg-slate-50 transition-colors",
                            },
                                createElement("td", { className: "px-4 py-3 text-slate-500 font-mono text-xs" }, c.comment_id),
                                createElement("td", { className: "px-4 py-3" },
                                    createElement("a", {
                                        href: `/content/${c.content_id}`,
                                        target: "_blank",
                                        className: "text-blue-600 hover:underline font-mono text-xs",
                                    }, `#${c.content_id}`),
                                ),
                                createElement("td", { className: "px-4 py-3" },
                                    createElement("span", { className: "text-sm font-medium text-slate-700" }, c.user_name || `用户${c.user_id}`),
                                ),
                                createElement("td", { className: "px-4 py-3" },
                                    createElement("p", {
                                        className: "text-sm text-slate-600 max-w-[300px] truncate",
                                        title: c.comment,
                                    }, c.comment || "-"),
                                ),
                                createElement("td", { className: "px-4 py-3" },
                                    createElement("span", {
                                        className: `inline-block px-2 py-0.5 rounded-full text-xs font-medium ${c.status === 10 ? "bg-green-100 text-green-700" : c.status === 20 ? "bg-yellow-100 text-yellow-700" : "bg-red-100 text-red-700"}`,
                                    }, c.status === 10 ? "正常" : c.status === 20 ? "已隐藏" : "已删除"),
                                ),
                                createElement("td", { className: "px-4 py-3" },
                                    createElement("div", { className: "flex gap-1.5" },
                                        c.status !== 20 && createElement("button", {
                                            onClick: () => handleCommentAction(c.comment_id, "hide"),
                                            className: "text-xs px-2.5 py-1 rounded-lg bg-orange-50 text-orange-600 hover:bg-orange-100 font-medium",
                                        }, "隐藏"),
                                        c.status === 20 && createElement("button", {
                                            onClick: () => handleCommentAction(c.comment_id, "restore"),
                                            className: "text-xs px-2.5 py-1 rounded-lg bg-green-50 text-green-600 hover:bg-green-100 font-medium",
                                        }, "恢复"),
                                        createElement("button", {
                                            onClick: () => handleCommentAction(c.comment_id, "delete"),
                                            className: "text-xs px-2.5 py-1 rounded-lg bg-red-50 text-red-600 hover:bg-red-100 font-medium",
                                        }, "删除"),
                                    ),
                                ),
                            ),
                        ),
                    ),
                ),
            ),

            // Pagination
            state.status === "ready" &&
                createElement("div", { className: "flex justify-center gap-2" },
                    createElement("button", {
                        disabled: page <= 0,
                        onClick: () => setPage(Math.max(0, page - 1)),
                        className: "px-3 py-1.5 text-sm border border-slate-300 rounded-lg disabled:opacity-40 hover:bg-slate-50",
                    }, "上一页"),
                    createElement("span", { className: "px-3 py-1.5 text-sm text-slate-600" }, `第 ${page + 1} 页`),
                    createElement("button", {
                        onClick: () => setPage(page + 1),
                        className: "px-3 py-1.5 text-sm border border-slate-300 rounded-lg hover:bg-slate-50",
                    }, "下一页"),
                ),
        ),
    );
}
