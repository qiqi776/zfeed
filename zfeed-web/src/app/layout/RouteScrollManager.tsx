import { useEffect, useLayoutEffect } from "react";
import { useLocation, useNavigationType } from "react-router-dom";

const routeScrollStorageKey = "zfeed-web:route-scroll";

function isScrollableReplayRoute(pathname: string) {
  return (
    pathname === "/" ||
    pathname === "/following" ||
    pathname === "/favorites" ||
    pathname === "/search" ||
    pathname === "/studio" ||
    /^\/users\/\d+$/.test(pathname) ||
    /^\/users\/\d+\/followers$/.test(pathname)
  );
}

function buildRouteScrollKey(pathname: string, search: string) {
  return `${pathname}${search}`;
}

function readRouteScrollMap() {
  if (typeof window === "undefined") {
    return {} as Record<string, number>;
  }

  try {
    const raw = window.sessionStorage.getItem(routeScrollStorageKey);
    if (!raw) {
      return {} as Record<string, number>;
    }

    const parsed = JSON.parse(raw) as Record<string, number>;
    return parsed && typeof parsed === "object" ? parsed : ({} as Record<string, number>);
  } catch {
    return {} as Record<string, number>;
  }
}

function writeRouteScrollMap(next: Record<string, number>) {
  if (typeof window === "undefined") {
    return;
  }

  try {
    window.sessionStorage.setItem(routeScrollStorageKey, JSON.stringify(next));
  } catch {
    // ignore storage failures and fall back to default browser behavior
  }
}

function saveRouteScrollPosition(key: string, top: number) {
  const current = readRouteScrollMap();
  current[key] = Math.max(0, Math.round(top));
  writeRouteScrollMap(current);
}

function readRouteScrollPosition(key: string) {
  const current = readRouteScrollMap();
  const value = current[key];
  return typeof value === "number" ? value : undefined;
}

export function RouteScrollManager() {
  const location = useLocation();
  const navigationType = useNavigationType();
  const routeKey = buildRouteScrollKey(location.pathname, location.search);
  const shouldReplay = isScrollableReplayRoute(location.pathname);

  useLayoutEffect(() => {
    return () => {
      if (!shouldReplay || typeof window === "undefined") {
        return;
      }

      saveRouteScrollPosition(routeKey, window.scrollY);
    };
  }, [routeKey, shouldReplay]);

  useLayoutEffect(() => {
    if (typeof window === "undefined") {
      return;
    }

    if (shouldReplay && navigationType === "POP") {
      const restoredTop = readRouteScrollPosition(routeKey);
      if (typeof restoredTop === "number") {
        window.scrollTo(0, restoredTop);
        return;
      }
    }

    window.scrollTo(0, 0);
  }, [navigationType, routeKey, shouldReplay]);

  useEffect(() => {
    if (!shouldReplay || typeof window === "undefined") {
      return;
    }

    const handlePageHide = () => {
      saveRouteScrollPosition(routeKey, window.scrollY);
    };

    window.addEventListener("pagehide", handlePageHide);
    return () => {
      window.removeEventListener("pagehide", handlePageHide);
    };
  }, [routeKey, shouldReplay]);

  return null;
}
