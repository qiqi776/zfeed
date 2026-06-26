import { createElement, useEffect, useState } from "react";
import { adminDashboard } from "../../runtime/apiClient";
import { readAuthSession, getAdminRole } from "../../runtime/authStore";
import { AdminLayout } from "./AdminLayout";

type DashboardData = {
    total_users: number;
    total_contents: number;
    total_comments: number;
    today_new_users: number;
    today_new_contents: number;
    pending_review: number;
};

type DashboardState =
    | { status: "loading" }
    | { status: "ready"; data: DashboardData }
    | { status: "error"; message: string };

export function AdminDashboardPage() {
    const [state, setState] = useState<DashboardState>({ status: "loading" });
    const session = readAuthSession();
    const role = getAdminRole();

    useEffect(() => {
        adminDashboard<DashboardData>()
            .then((data) => setState({ status: "ready", data }))
            .catch((err) => setState({ status: "error", message: err instanceof Error ? err.message : "获取看板数据失败" }));
    }, []);

    const statCards = [
        { label: "总用户数", value: state.status === "ready" ? state.data.total_users : "-", color: "bg-blue-500" },
        { label: "总内容数", value: state.status === "ready" ? state.data.total_contents : "-", color: "bg-green-500" },
        { label: "总评论数", value: state.status === "ready" ? state.data.total_comments : "-", color: "bg-purple-500" },
        { label: "今日新增用户", value: state.status === "ready" ? state.data.today_new_users : "-", color: "bg-orange-500" },
        { label: "今日新增内容", value: state.status === "ready" ? state.data.today_new_contents : "-", color: "bg-cyan-500" },
        { label: "待审核", value: state.status === "ready" ? state.data.pending_review : "-", color: "bg-red-500" },
    ];

    return createElement(AdminLayout, {
        currentPath: "/admin",
        adminName: session?.user?.nickname,
    },
        createElement("div", { className: "space-y-6" },
            createElement("h2", { className: "text-xl font-bold text-slate-800" }, "📊 数据看板"),

            state.status === "error" &&
                createElement("div", { className: "bg-red-50 border border-red-200 rounded-xl p-4 text-sm text-red-600" }, state.message),

            // Stat cards
            createElement("div", {
                className: "grid gap-4",
                style: { gridTemplateColumns: "repeat(auto-fill, minmax(200px, 1fr))" }
            },
                statCards.map((card) =>
                    createElement("div", {
                        key: card.label,
                        className: "bg-white rounded-xl shadow-sm border border-slate-200 p-5",
                    },
                        createElement("div", {
                            className: `inline-block w-10 h-10 rounded-lg ${card.color} flex items-center justify-center text-white text-lg mb-3`
                        }, card.label.slice(0, 1)),
                        createElement("p", { className: "text-2xl font-bold text-slate-800" },
                            state.status === "loading" ? "..." : String(card.value),
                        ),
                        createElement("p", { className: "text-sm text-slate-500 mt-0.5" }, card.label),
                    )
                ),
            ),

            // System status + quick links
            createElement("div", { className: "grid gap-6", style: { gridTemplateColumns: "1fr 1fr" } },
                // System overview
                createElement("div", { className: "bg-white rounded-xl shadow-sm border border-slate-200 p-5" },
                    createElement("h3", { className: "text-sm font-semibold text-slate-700 mb-3" }, "🖥️ 系统状态"),
                    createElement("div", { className: "space-y-2 text-sm" },
                        createElement("div", { className: "flex justify-between py-1 border-b border-slate-100" },
                            createElement("span", { className: "text-slate-500" }, "管理员角色"),
                            createElement("span", { className: "font-medium text-slate-700" }, role >= 2 ? "超级管理员" : "管理员"),
                        ),
                        createElement("div", { className: "flex justify-between py-1 border-b border-slate-100" },
                            createElement("span", { className: "text-slate-500" }, "待审核内容"),
                            createElement("span", { className: `font-medium ${state.status === "ready" && state.data.pending_review > 0 ? "text-red-600" : "text-slate-700"}` },
                                state.status === "ready" ? state.data.pending_review : "-"
                            ),
                        ),
                        createElement("div", { className: "flex justify-between py-1" },
                            createElement("span", { className: "text-slate-500" }, "数据来源"),
                            createElement("span", { className: "font-medium text-green-600 text-xs" }, "Admin API"),
                        ),
                    ),
                ),

                // Quick links
                createElement("div", { className: "space-y-3" },
                    createElement(QuickLink, { href: "/admin/users", title: "👤 用户管理", desc: "查看、搜索、管理用户账号" }),
                    createElement(QuickLink, { href: "/admin/contents", title: "📝 内容审核", desc: "审核内容，处理违规" }),
                    createElement(QuickLink, { href: "/admin/comments", title: "💬 评论管理", desc: "查看和管理评论" }),
                    createElement(QuickLink, { href: "/admin/settings", title: "⚙️ 系统设置", desc: "功能开关与工具链接" }),
                ),
            ),
        ),
    );
}

function QuickLink({ href, title, desc }: { href: string; title: string; desc: string }) {
    return createElement("a", {
        href,
        className: "block bg-white rounded-xl shadow-sm border border-slate-200 p-4 hover:border-blue-300 hover:shadow-md transition-all",
    },
        createElement("h3", { className: "font-semibold text-slate-800 text-sm" }, title),
        createElement("p", { className: "mt-1 text-xs text-slate-500" }, desc),
    );
}
