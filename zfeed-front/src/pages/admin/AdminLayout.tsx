import { createElement, type ReactNode } from "react";

const sidebarItems = [
    { href: "/admin", label: "📊 数据看板", icon: "dashboard" },
    { href: "/admin/users", label: "👤 用户管理", icon: "users" },
    { href: "/admin/contents", label: "📝 内容审核", icon: "contents" },
    { href: "/admin/comments", label: "💬 评论管理", icon: "comments" },
    { href: "/admin/settings", label: "⚙️ 系统设置", icon: "settings" },
];

type Props = {
    children?: ReactNode;
    currentPath: string;
    adminName?: string;
};

export function AdminLayout({ children, currentPath, adminName }: Props) {
    return createElement("div", {
        className: "flex min-h-screen bg-slate-50",
        style: { fontFamily: "-apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif" }
    },
        // Sidebar
        createElement("aside", {
            className: "w-56 bg-slate-900 text-white flex flex-col shrink-0",
        },
            createElement("div", { className: "px-5 py-4 border-b border-slate-700" },
                createElement("span", { className: "text-lg font-bold tracking-wide" }, "ZFeed 后台"),
            ),
            createElement("nav", { className: "flex-1 py-3" },
                sidebarItems.map((item) =>
                    createElement("a", {
                        key: item.href,
                        href: item.href,
                        className: `flex items-center gap-3 px-5 py-2.5 text-sm transition-colors ${
                            currentPath === item.href
                                ? "bg-slate-700 text-white font-medium border-l-3 border-blue-400"
                                : "text-slate-300 hover:bg-slate-800 hover:text-white"
                        }`,
                    }, item.label)
                ),
            ),
            createElement("div", { className: "px-5 py-3 border-t border-slate-700 text-xs text-slate-400" },
                createElement("a", { href: "/home", className: "hover:text-white transition-colors" }, "← 返回前台"),
            ),
        ),

        // Main content
        createElement("div", { className: "flex-1 flex flex-col min-w-0" },
            // Top bar
            createElement("header", {
                className: "h-14 bg-white border-b border-slate-200 flex items-center justify-between px-6 shrink-0",
            },
                createElement("span", {
                    className: "text-sm text-slate-400"
                }, "管理后台"),

                adminName
                    ? createElement("span", { className: "text-sm text-slate-600 font-medium" },
                        createElement("span", { className: "mr-2" }, "👤"),
                        adminName,
                    )
                    : null,
            ),

            // Page content
            createElement("main", { className: "flex-1 p-6 overflow-auto" },
                children,
            ),
        ),
    );
};
