import { resolvePageRoute } from "./pageRoutes";
import type { PageId } from "./pageRoutes";

export type AdminPageId =
    | "admin-login"
    | "admin-dashboard"
    | "admin-users"
    | "admin-contents"
    | "admin-comments"
    | "admin-settings"
    | "admin-not-found";

const adminRouteMap: Record<string, AdminPageId> = {
    "/admin": "admin-dashboard",
    "/admin/login": "admin-login",
    "/admin/users": "admin-users",
    "/admin/contents": "admin-contents",
    "/admin/comments": "admin-comments",
    "/admin/settings": "admin-settings",
};

export function resolveAdminPageRoute(pathname: string): { isAdmin: true; page: AdminPageId } | { isAdmin: false; page: PageId } {
    if (pathname.startsWith("/admin")) {
        return {
            isAdmin: true,
            page: adminRouteMap[pathname] ?? "admin-not-found",
        };
    }

    return {
        isAdmin: false,
        page: resolvePageRoute(pathname),
    };
}
