import { createElement, useEffect, useState } from "react";
import type { ChangeEvent, KeyboardEvent } from "react";
import { adminListUsers, adminUpdateUserStatus } from "../../runtime/apiClient";
import { showToast } from "../../runtime/toast";
import { readAuthSession } from "../../runtime/authStore";
import { AdminLayout } from "./AdminLayout";
import { ConfirmDialog } from "./ConfirmDialog";
import { UserDetailDrawer } from "./UserDetailDrawer";

type AdminUserItem = { user_id: number; nickname: string; avatar: string; mobile: string; status: number; role: number; created_at: number };

type UsersState =
    | { status: "loading" }
    | { status: "ready"; users: AdminUserItem[]; total: number }
    | { status: "error"; message: string };

export function AdminUsersPage() {
    const [state, setState] = useState<UsersState>({ status: "loading" });
    const [searchInput, setSearchInput] = useState("");
    const [selectedUserId, setSelectedUserId] = useState<number | null>(null);
    const [page, setPage] = useState(0);
    const pageSize = 20;
    const session = readAuthSession();

    const loadUsers = (q: string, pageNum: number) => {
        setState({ status: "loading" });
        adminListUsers<{ users: AdminUserItem[]; total_count: number }>({
            query: q.trim(),
            page: pageNum,
            page_size: pageSize,
        }).then((res) => {
            setState({ status: "ready", users: res.users ?? [], total: res.total_count ?? 0 });
        }).catch((err) => {
            setState({ status: "error", message: err instanceof Error ? err.message : "加载失败" });
        });
    };

    useEffect(() => { loadUsers(searchInput, page); }, [searchInput, page]);

    const handleSearch = () => { setPage(0); };

    // User status toggle
    const [confirmOpen, setConfirmOpen] = useState(false);
    const [confirmUserId, setConfirmUserId] = useState<number | null>(null);
    const [confirmAction, setConfirmAction] = useState<"disable" | "enable">("disable");

    const handleToggleStatus = (userId: number, action: "disable" | "enable") => {
        setConfirmUserId(userId);
        setConfirmAction(action);
        setConfirmOpen(true);
    };

    const executeUserAction = async () => {
        if (!confirmUserId) return;
        const status = confirmAction === "disable" ? 20 : 10;
        try {
            await adminUpdateUserStatus({ user_id: confirmUserId, status });
            loadUsers(searchInput, page);
        } catch (err) {
            showToast(err instanceof Error ? err.message : "操作失败");
        } finally {
            setConfirmOpen(false);
        }
    };

    return createElement(AdminLayout, { currentPath: "/admin/users", adminName: session?.user?.nickname },
        createElement(UserDetailDrawer, {
            userId: selectedUserId ?? 0,
            open: selectedUserId !== null,
            onClose: () => setSelectedUserId(null),
        }),
        createElement(ConfirmDialog, {
            open: confirmOpen,
            title: confirmAction === "disable" ? "禁用用户" : "启用用户",
            message: `确定要${confirmAction === "disable" ? "禁用" : "启用"}该用户吗？${confirmAction === "disable" ? "用户将无法登录和使用任何功能。" : "用户将恢复正常使用。"}`,
            confirmLabel: confirmAction === "disable" ? "禁用" : "启用",
            danger: confirmAction === "disable",
            onConfirm: executeUserAction,
            onCancel: () => setConfirmOpen(false),
        }),
        createElement("div", { className: "space-y-5" },
            createElement("h2", { className: "text-xl font-bold text-slate-800" }, "👤 用户管理"),

            // Search bar
            createElement("div", { className: "flex gap-3" },
                createElement("input", {
                    type: "text",
                    value: searchInput,
                    onChange: (e: ChangeEvent<HTMLInputElement>) => setSearchInput(e.target.value),
                    onKeyDown: (e: KeyboardEvent<HTMLInputElement>) => { if (e.key === "Enter") handleSearch(); },
                    placeholder: "搜索用户（昵称 / 手机号）...",
                    className: "flex-1 max-w-md px-4 py-2 border border-slate-300 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-blue-500",
                }),
                createElement("button", {
                    onClick: handleSearch,
                    className: "px-5 py-2 bg-blue-600 text-white text-sm font-medium rounded-lg hover:bg-blue-700 transition-colors",
                }, "搜索"),
            ),

            // User count
            state.status === "ready" &&
                createElement("p", { className: "text-sm text-slate-500" }, `共 ${state.total} 个用户`),

            // User table
            createElement("div", { className: "bg-white rounded-xl shadow-sm border border-slate-200 overflow-hidden" },
                createElement("table", { className: "w-full text-sm" },
                    createElement("thead", { className: "bg-slate-50 text-slate-500 text-xs uppercase" },
                        createElement("tr", null,
                            ...[ "ID", "昵称", "手机号", "状态", "角色", "操作" ].map((h) =>
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
                        state.status === "ready" && state.users.length === 0 &&
                            createElement("tr", null,
                                createElement("td", { colSpan: 6, className: "px-4 py-8 text-center text-slate-400" }, "暂无用户数据"),
                            ),
                        state.status === "ready" && state.users.map((u) =>
                            createElement("tr", {
                                key: u.user_id,
                                className: "border-t border-slate-100 hover:bg-slate-50 transition-colors cursor-pointer",
                                onClick: () => setSelectedUserId(u.user_id),
                            },
                                createElement("td", { className: "px-4 py-3 text-slate-500 font-mono text-xs" }, u.user_id),
                                createElement("td", { className: "px-4 py-3 font-medium text-slate-800" },
                                    createElement("div", { className: "flex items-center gap-2" },
                                        u.avatar
                                            ? createElement("img", { src: u.avatar, className: "w-7 h-7 rounded-full object-cover" })
                                            : createElement("div", { className: "w-7 h-7 rounded-full bg-slate-200 flex items-center justify-center text-xs text-slate-500" }, "U"),
                                        u.nickname || "未设置",
                                    ),
                                ),
                                createElement("td", { className: "px-4 py-3 text-slate-500 text-xs font-mono" }, u.mobile || "-"),
                                createElement("td", { className: "px-4 py-3" },
                                    createElement("span", {
                                        className: `inline-block px-2 py-0.5 rounded-full text-xs font-medium ${u.status === 10 ? "bg-green-100 text-green-700" : "bg-red-100 text-red-700"}`,
                                    }, u.status === 10 ? "正常" : "已禁用"),
                                ),
                                createElement("td", { className: "px-4 py-3" },
                                    createElement("span", {
                                        className: `inline-block px-2 py-0.5 rounded-full text-xs font-medium ${u.role >= 2 ? "bg-purple-100 text-purple-700" : u.role >= 1 ? "bg-blue-100 text-blue-700" : "bg-slate-100 text-slate-600"}`,
                                    }, u.role >= 2 ? "超级管理员" : u.role >= 1 ? "管理员" : "普通用户"),
                                ),
                                createElement("td", { className: "px-4 py-3" },
                                    createElement("div", { className: "flex gap-1.5" },
                                        createElement("button", {
                                            onClick: (e: Event) => { e.stopPropagation(); handleToggleStatus(u.user_id, "disable"); },
                                            className: "text-xs font-medium px-2.5 py-1 rounded-lg text-red-600 bg-red-50 hover:bg-red-100 transition-colors",
                                        }, "禁用"),
                                        createElement("button", {
                                            onClick: (e: Event) => { e.stopPropagation(); handleToggleStatus(u.user_id, "enable"); },
                                            className: "text-xs font-medium px-2.5 py-1 rounded-lg text-green-600 bg-green-50 hover:bg-green-100 transition-colors",
                                        }, "启用"),
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
