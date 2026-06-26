import { Component, createElement, lazy, Suspense, useEffect, useState } from "react";
import type { ComponentType, LazyExoticComponent, ReactNode } from "react";
import { resolveAdminPageRoute, type AdminPageId } from "./routes/adminRoutes";
import type { PageId } from "./routes/pageRoutes";
import { ROUTES } from "./routes/constants";
import { isAdmin } from "./runtime/authStore";
import { ToastContainer } from "./runtime/toast";
import "./styles/index.css";

const pageComponents: Record<PageId, LazyExoticComponent<ComponentType>> = {
    "auth-gateway": lazy(() => import("./pages/AuthGatewayPage").then((module) => ({ default: module.AuthGatewayPage }))),
    home: lazy(() => import("./pages/HomePage").then((module) => ({ default: module.HomePage }))),
    following: lazy(() => import("./pages/FollowingPage").then((module) => ({ default: module.FollowingPage }))),
    profile: lazy(() => import("./pages/ProfilePage").then((module) => ({ default: module.ProfilePage }))),
    detail: lazy(() => import("./pages/DetailPage").then((module) => ({ default: module.DetailPage }))),
    "edit-content": lazy(() => import("./pages/EditContentPage").then((module) => ({ default: module.EditContentPage }))),
    "edit-profile": lazy(() => import("./pages/EditProfilePage").then((module) => ({ default: module.EditProfilePage }))),
    search: lazy(() => import("./pages/SearchPage").then((module) => ({ default: module.SearchPage }))),
    compose: lazy(() => import("./pages/ComposePage").then((module) => ({ default: module.ComposePage }))),
    settings: lazy(() => import("./pages/SettingsPage").then((module) => ({ default: module.SettingsPage }))),
    login: lazy(() => import("./pages/LoginPage").then((module) => ({ default: module.LoginPage }))),
    register: lazy(() => import("./pages/RegisterPage").then((module) => ({ default: module.RegisterPage }))),
    "not-found": lazy(() => import("./pages/NotFoundPage").then((module) => ({ default: module.NotFoundPage })))
};

const adminPageComponents: Record<AdminPageId, LazyExoticComponent<ComponentType>> = {
    "admin-login": lazy(() => import("./pages/admin/AdminLoginPage").then((module) => ({ default: module.AdminLoginPage }))),
    "admin-dashboard": lazy(() => import("./pages/admin/AdminDashboardPage").then((module) => ({ default: module.AdminDashboardPage }))),
    "admin-users": lazy(() => import("./pages/admin/AdminUsersPage").then((module) => ({ default: module.AdminUsersPage }))),
    "admin-contents": lazy(() => import("./pages/admin/AdminContentsPage").then((module) => ({ default: module.AdminContentsPage }))),
    "admin-comments": lazy(() => import("./pages/admin/AdminCommentsPage").then((module) => ({ default: module.AdminCommentsPage }))),
    "admin-settings": lazy(() => import("./pages/admin/AdminSettingsPage").then((module) => ({ default: module.AdminSettingsPage }))),
    "admin-not-found": lazy(() => import("./pages/NotFoundPage").then((module) => ({ default: module.NotFoundPage })))
};

const adminPublicPages: AdminPageId[] = ["admin-login", "admin-not-found"];

export function App() {
    const [revision, setRevision] = useState(0);
    const resolved = resolveAdminPageRoute(window.location.pathname);

    // Auth guard: redirect to admin login if accessing protected admin page without admin role
    useEffect(() => {
        if (
            resolved.isAdmin &&
            !adminPublicPages.includes(resolved.page) &&
            !isAdmin()
        ) {
            window.location.href = ROUTES.ADMIN_LOGIN;
        }
    }, [resolved.isAdmin, resolved.page]);

    const Page = resolved.isAdmin
        ? adminPageComponents[resolved.page]
        : pageComponents[resolved.page];
    const routeKey = resolved.isAdmin ? `admin-${resolved.page}` : resolved.page;
    const pageKey = `${routeKey}-${window.location.search}-${revision}`;

    useEffect(() => {
        const onPopState = () => setRevision((current) => current + 1);
        window.addEventListener("popstate", onPopState);
        return () => window.removeEventListener("popstate", onPopState);
    }, []);

    useEffect(() => {
        const onClick = (event: MouseEvent) => {
            const link = findClosestLink(event.target);
            if (!link || !shouldHandleInternalNavigation(event, link)) {
                return;
            }

            event.preventDefault();
            event.stopImmediatePropagation();

            const nextUrl = new URL(link.href);
            if (nextUrl.href !== window.location.href) {
                window.history.pushState(getNavigationState(link), "", `${nextUrl.pathname}${nextUrl.search}${nextUrl.hash}`);
                window.dispatchEvent(new PopStateEvent("popstate"));
            }
        };

        document.addEventListener("click", onClick, true);
        return () => document.removeEventListener("click", onClick, true);
    }, []);

    return (
        <ErrorBoundary>
            <Suspense fallback={<RouteLoading />}>
                <Page key={pageKey} />
            </Suspense>
            <ToastContainer />
        </ErrorBoundary>
    );
}

function RouteLoading() {
    return <div className="min-h-screen bg-[#eef2f6] px-6 py-8 text-[14px] text-slate-500" data-page-state="loading">正在加载...</div>;
}

class ErrorBoundary extends Component<{ children: ReactNode }, { hasError: boolean; error: string }> {
    constructor(props: { children: ReactNode }) {
        super(props);
        this.state = { hasError: false, error: "" };
    }

    static getDerivedStateFromError(error: Error) {
        return { hasError: true, error: error.message || "未知错误" };
    }

    componentDidCatch(error: Error) {
        console.error("ErrorBoundary caught:", error);
    }

    render() {
        if (this.state.hasError) {
            return createElement("div", {
                className: "min-h-screen bg-[#eef2f6] flex items-center justify-center px-6"
            },
                createElement("div", { className: "text-center max-w-md" },
                    createElement("h1", { className: "text-2xl font-bold text-slate-700 mb-3" }, "页面出现错误"),
                    createElement("p", { className: "text-sm text-slate-500 mb-6" }, this.state.error || "请刷新页面重试"),
                    createElement("a", {
                        href: "/home",
                        className: "inline-flex items-center gap-2 px-5 py-2.5 bg-blue-600 text-white rounded-full text-sm font-medium hover:bg-blue-700 transition-colors"
                    }, "返回首页"),
                )
            );
        }
        return this.props.children;
    }
}

function findClosestLink(target: EventTarget | null) {
    return target instanceof Element ? target.closest<HTMLAnchorElement>("a[href]") : null;
}

function shouldHandleInternalNavigation(event: MouseEvent, link: HTMLAnchorElement) {
    if (event.defaultPrevented || event.metaKey || event.ctrlKey || event.shiftKey || event.altKey || event.button !== 0) {
        return false;
    }

    if (link.target === "_blank" || link.hasAttribute("download")) {
        return false;
    }

    const href = link.getAttribute("href");
    if (!href || href === "#" || href.startsWith("#")) {
        return false;
    }

    const url = new URL(link.href);
    return (
        url.origin === window.location.origin &&
        (url.pathname === "/" ||
            url.pathname === "/home" ||
            url.pathname === "/me" ||
            url.pathname === "/me/edit" ||
            url.pathname === "/following" ||
            url.pathname.startsWith("/user/") ||
            url.pathname.startsWith("/content/") ||
            url.pathname.startsWith("/admin") ||
            url.pathname === "/search" ||
            url.pathname === "/compose" ||
            url.pathname === "/settings" ||
            url.pathname === "/login" ||
            url.pathname === "/register")
    );
}

function getNavigationState(link: HTMLAnchorElement) {
    return link.dataset.preserveFeedRails === "true" ? { preserveFeedRails: true } : {};
}
