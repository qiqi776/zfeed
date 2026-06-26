import { resolvePageRoute, type PageId } from "../routes/pageRoutes";

const safeNextRoutes = new Set<PageId>([
    "home",
    "following",
    "profile",
    "detail",
    "edit-profile",
    "search",
    "compose",
    "settings",
]);

export function resolveSafeAuthNextPath(defaultPath: string) {
    const next = readSafeAuthNextPath();
    return next ?? defaultPath;
}

export function buildAuthSiblingHref(pathname: "/login" | "/register") {
    const next = readSafeAuthNextPath();
    return next ? `${pathname}?next=${encodeURIComponent(next)}` : pathname;
}

function readSafeAuthNextPath() {
    const next = new URLSearchParams(window.location.search).get("next");
    if (!next || !next.startsWith("/") || next.startsWith("//")) {
        return null;
    }

    try {
        const url = new URL(next, window.location.origin);
        if (url.origin !== window.location.origin) {
            return null;
        }

        const route = resolvePageRoute(url.pathname);
        if (!safeNextRoutes.has(route)) {
            return null;
        }

        return `${url.pathname}${url.search}${url.hash}`;
    } catch {
        return null;
    }
}
