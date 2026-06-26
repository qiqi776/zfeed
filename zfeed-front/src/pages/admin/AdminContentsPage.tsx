import { createElement, useEffect, useState } from "react";
import type { ChangeEvent, KeyboardEvent } from "react";
import { adminListContents, adminUpdateContentStatus } from "../../runtime/apiClient";
import { showToast } from "../../runtime/toast";
import { readAuthSession } from "../../runtime/authStore";
import { AdminLayout } from "./AdminLayout";
import { ConfirmDialog } from "./ConfirmDialog";
import { ContentPreviewModal } from "./ContentPreviewModal";

type AdminContentItem = {
    content_id: number;
    content_type: number;
    author_id: number;
    author_name: string;
    title: string;
    cover_url: string;
    status: number;
    published_at: number;
    like_count: number;
    comment_count: number;
};

type ContentsState =
    | { status: "loading" }
    | { status: "ready"; contents: AdminContentItem[]; total: number }
    | { status: "error"; message: string };

export function AdminContentsPage() {
    const [state, setState] = useState<ContentsState>({ status: "loading" });
    const [searchInput, setSearchInput] = useState("");
    const [page, setPage] = useState(0);
    const [previewContentId, setPreviewContentId] = useState<number | null>(null);
    const [confirmOpen, setConfirmOpen] = useState(false);
    const [confirmContentId, setConfirmContentId] = useState<number | null>(null);
    const [confirmAction, setConfirmAction] = useState<"hide" | "show">("hide");
    const pageSize = 20;
    const session = readAuthSession();

    const loadContents = (q: string, pageNum: number) => {
        setState({ status: "loading" });
        adminListContents<{ contents: AdminContentItem[]; total_count: number }>({
            query: q.trim(),
            page: pageNum,
            page_size: pageSize,
        }).then((res) => {
            setState({ status: "ready", contents: res.contents ?? [], total: res.total_count ?? 0 });
        }).catch((err) => {
            setState({ status: "error", message: err instanceof Error ? err.message : "加载失败" });
        });
    };

    useEffect(() => { loadContents(searchInput, page); }, [searchInput, page]);

    const handleSearch = () => { setPage(0); };

    const handleAction = (contentId: number, action: "hide" | "show") => {
        setConfirmContentId(contentId);
        setConfirmAction(action);
        setConfirmOpen(true);
    };

    const executeAction = async () => {
        if (!confirmContentId) return;
        const status = confirmAction === "hide" ? 20 : 30;
        try {
            await adminUpdateContentStatus({ content_id: confirmContentId, status });
            loadContents(searchInput, page);
        } catch (err) {
            showToast(err instanceof Error ? err.message : "操作失败");
        } finally {
            setConfirmOpen(false);
        }
    };

    return createElement(AdminLayout, { currentPath: "/admin/contents", adminName: session?.user?.nickname },
        createElement(ContentPreviewModal, {
            contentId: previewContentId ?? 0,
            open: previewContentId !== null,
            onClose: () => setPreviewContentId(null),
        }),
        createElement(ConfirmDialog, {
            open: confirmOpen,
            title: confirmAction === "hide" ? "隐藏内容" : "恢复内容",
            message: `确定要${confirmAction === "hide" ? "隐藏" : "恢复"}这条内容吗？${confirmAction === "hide" ? "用户将无法看到此内容。" : "内容将重新对用户可见。"}`,
            confirmLabel: confirmAction === "hide" ? "隐藏" : "恢复",
            danger: confirmAction === "hide",
            onConfirm: executeAction,
            onCancel: () => setConfirmOpen(false),
        }),
        createElement("div", { className: "space-y-5" },
            createElement("h2", { className: "text-xl font-bold text-slate-800" }, "📝 内容审核"),

            // Search bar
            createElement("div", { className: "flex gap-3" },
                createElement("input", {
                    type: "text",
                    value: searchInput,
                    onChange: (e: ChangeEvent<HTMLInputElement>) => setSearchInput(e.target.value),
                    onKeyDown: (e: KeyboardEvent<HTMLInputElement>) => { if (e.key === "Enter") handleSearch(); },
                    placeholder: "搜索内容（标题 / 作者）...",
                    className: "flex-1 max-w-md px-4 py-2 border border-slate-300 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-blue-500",
                }),
                createElement("button", {
                    onClick: handleSearch,
                    className: "px-5 py-2 bg-blue-600 text-white text-sm font-medium rounded-lg hover:bg-blue-700 transition-colors",
                }, "搜索"),
            ),

            state.status === "ready" &&
                createElement("p", { className: "text-sm text-slate-500" }, `共 ${state.total} 条内容`),

            // Content table
            createElement("div", { className: "bg-white rounded-xl shadow-sm border border-slate-200 overflow-hidden" },
                createElement("table", { className: "w-full text-sm" },
                    createElement("thead", { className: "bg-slate-50 text-slate-500 text-xs uppercase" },
                        createElement("tr", null,
                            ...[ "ID", "标题", "作者", "状态", "互动", "操作" ].map((h) =>
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
                        state.status === "ready" && state.contents.length === 0 &&
                            createElement("tr", null,
                                createElement("td", { colSpan: 6, className: "px-4 py-8 text-center text-slate-400" }, "暂无内容"),
                            ),
                        state.status === "ready" && state.contents.map((c) =>
                            createElement("tr", {
                                key: c.content_id,
                                className: "border-t border-slate-100 hover:bg-slate-50 transition-colors cursor-pointer",
                                onClick: () => setPreviewContentId(c.content_id),
                            },
                                createElement("td", { className: "px-4 py-3 text-slate-500 font-mono text-xs" }, c.content_id),
                                createElement("td", { className: "px-4 py-3" },
                                    createElement("div", { className: "flex items-center gap-2" },
                                        c.cover_url
                                            ? createElement("img", { src: c.cover_url, className: "w-10 h-10 rounded-lg object-cover shrink-0" })
                                            : createElement("div", { className: "w-10 h-10 rounded-lg bg-slate-100 flex items-center justify-center text-slate-400 text-xs shrink-0" }, "无图"),
                                        createElement("div", { className: "min-w-0" },
                                            createElement("span", {
                                                className: "font-medium text-slate-800 hover:text-blue-600 truncate block max-w-[300px]",
                                            }, c.title || "无标题"),
                                            createElement("span", { className: "text-xs text-slate-400" },
                                                c.content_type === 10 ? "文章" : c.content_type === 20 ? "视频" : "其他",
                                            ),
                                        ),
                                    ),
                                ),
                                createElement("td", { className: "px-4 py-3" },
                                    createElement("a", {
                                        href: `/user/${c.author_id}`,
                                        target: "_blank",
                                        className: "text-slate-600 hover:text-blue-600 text-sm",
                                        onClick: (e: Event) => e.stopPropagation(),
                                    }, c.author_name || `用户${c.author_id}`),
                                ),
                                createElement("td", { className: "px-4 py-3" },
                                    createElement("span", {
                                        className: `inline-block px-2 py-0.5 rounded-full text-xs font-medium ${c.status === 30 ? "bg-green-100 text-green-700" : c.status === 20 ? "bg-yellow-100 text-yellow-700" : "bg-slate-100 text-slate-600"}`,
                                    }, c.status === 30 ? "已发布" : c.status === 20 ? "已隐藏" : "草稿"),
                                ),
                                createElement("td", { className: "px-4 py-3 text-xs text-slate-500" },
                                    createElement("span", { className: "mr-3" }, `👍 ${c.like_count ?? 0}`),
                                    createElement("span", null, `💬 ${c.comment_count ?? 0}`),
                                ),
                                createElement("td", { className: "px-4 py-3" },
                                    createElement("div", { className: "flex gap-1.5" },
                                        createElement("button", {
                                            onClick: (e: Event) => { e.stopPropagation(); setPreviewContentId(c.content_id); },
                                            className: "text-xs px-2.5 py-1 rounded-lg bg-blue-50 text-blue-600 hover:bg-blue-100 font-medium",
                                        }, "预览"),
                                        createElement("button", {
                                            onClick: (e: Event) => { e.stopPropagation(); handleAction(c.content_id, "hide"); },
                                            className: "text-xs px-2.5 py-1 rounded-lg bg-orange-50 text-orange-600 hover:bg-orange-100 font-medium",
                                        }, "隐藏"),
                                        createElement("button", {
                                            onClick: (e: Event) => { e.stopPropagation(); handleAction(c.content_id, "show"); },
                                            className: "text-xs px-2.5 py-1 rounded-lg bg-green-50 text-green-600 hover:bg-green-100 font-medium",
                                        }, "恢复"),
                                    ),
                                ),
                            ),
                        ),
                    ),
                ),
            ),

            state.status === "ready" && state.total > 20 &&
                createElement("div", { className: "flex justify-center gap-2" },
                    createElement("button", {
                        disabled: page <= 1,
                        onClick: () => setPage(page - 1),
                        className: "px-3 py-1.5 text-sm border border-slate-300 rounded-lg disabled:opacity-40 hover:bg-slate-50",
                    }, "上一页"),
                    createElement("span", { className: "px-3 py-1.5 text-sm text-slate-600" }, `第 ${page} 页`),
                    createElement("button", {
                        disabled: page * 20 >= state.total,
                        onClick: () => setPage(page + 1),
                        className: "px-3 py-1.5 text-sm border border-slate-300 rounded-lg disabled:opacity-40 hover:bg-slate-50",
                    }, "下一页"),
                ),
        ),
    );
}
