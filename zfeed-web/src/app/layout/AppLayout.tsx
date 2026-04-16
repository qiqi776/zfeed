import { useMutation } from "@tanstack/react-query";
import { Link, NavLink, Outlet, useNavigate } from "react-router-dom";

import { RouteScrollManager } from "@/app/layout/RouteScrollManager";
import { useSessionStore } from "@/entities/session/model/session.store";
import { logout } from "@/features/auth/api/auth.api";
import { queryClient } from "@/shared/lib/query/queryClient";

type PrimaryNavItem = {
  to: string;
  label: string;
  end?: boolean;
};

type SecondaryNavItem = {
  to: string;
  label: string;
};

const primaryNavItems: PrimaryNavItem[] = [
  { to: "/", label: "推荐", end: true },
  { to: "/following", label: "关注" },
  { to: "/search", label: "搜索" },
  { to: "/publish", label: "发布" },
  { to: "/me", label: "我的" },
];

const secondaryNavItems: SecondaryNavItem[] = [
  { to: "/favorites", label: "收藏" },
  { to: "/studio", label: "创作台" },
];

function getDesktopNavLinkClass(isActive: boolean) {
  return [
    "rounded-full px-3 py-1.5 transition",
    isActive ? "bg-[#eef7fb] text-accent" : "text-slate-600 hover:text-accent",
  ].join(" ");
}

function getShortcutLinkClass(isActive: boolean) {
  return [
    "rounded-full border px-4 py-2 text-sm transition",
    isActive
      ? "border-[#cfe6f2] bg-[#eef7fb] text-accent"
      : "border-slate-200 bg-white text-slate-600 hover:border-accent hover:text-accent",
  ].join(" ");
}

function getBottomNavLinkClass(isActive: boolean) {
  return [
    "flex min-h-[3.5rem] flex-col items-center justify-center rounded-[20px] px-2 text-xs font-medium transition",
    isActive
      ? "bg-[linear-gradient(180deg,#eef7fb,#e3f0f7)] text-accent shadow-card"
      : "text-slate-500 hover:bg-white/80 hover:text-accent",
  ].join(" ");
}

export function AppLayout() {
  const navigate = useNavigate();
  const user = useSessionStore((state) => state.user);
  const clearSession = useSessionStore((state) => state.clearSession);
  const profileHref = user?.userId ? `/users/${user.userId}` : "/me";

  const logoutMutation = useMutation({
    mutationFn: logout,
    onSettled: () => {
      queryClient.clear();
      clearSession();
      navigate("/login", { replace: true });
    },
  });

  return (
    <div className="min-h-screen bg-gradient-to-b from-[#f6fbff] to-[#edf2f7] text-ink">
      <a href="#app-main" className="skip-link">
        跳到主要内容
      </a>
      <header className="sticky top-0 z-30 border-b border-white/70 bg-white/90 backdrop-blur">
        <div className="mx-auto max-w-6xl px-4 sm:px-5">
          <div className="flex min-h-[4.5rem] items-center justify-between gap-3 py-3">
            <div className="min-w-0">
              <Link to="/" className="font-display text-xl font-semibold tracking-tight text-ink">
                ZFeed Web
              </Link>
              <p className="mt-1 text-xs text-slate-500 sm:text-sm">
                安静地发布、回访和整理自己的内容关系。
              </p>
            </div>

            <nav className="hidden items-center gap-5 text-sm font-medium md:flex" aria-label="桌面主导航">
              {primaryNavItems.map((item) => (
                <NavLink
                  key={item.to}
                  to={item.to}
                  end={item.end}
                  className={({ isActive }) => getDesktopNavLinkClass(isActive)}
                >
                  {item.label}
                </NavLink>
              ))}
            </nav>

            <div className="flex shrink-0 items-center gap-2 sm:gap-3">
              <nav className="hidden items-center gap-2 lg:flex" aria-label="桌面快捷入口">
                {secondaryNavItems.map((item) => (
                  <NavLink
                    key={item.to}
                    to={item.to}
                    className={({ isActive }) => getShortcutLinkClass(isActive)}
                  >
                    {item.label}
                  </NavLink>
                ))}
              </nav>
              <Link
                to={profileHref}
                className="max-w-28 truncate rounded-full border border-slate-200 bg-white px-3 py-1.5 text-sm text-slate-600 transition hover:border-accent hover:text-accent"
              >
                {user?.nickname ?? "用户"}
              </Link>
              <button
                type="button"
                onClick={() => logoutMutation.mutate()}
                disabled={logoutMutation.isPending}
                className="rounded-xl border border-slate-200 px-3 py-1.5 text-sm transition hover:border-ember hover:text-ember disabled:opacity-60"
              >
                {logoutMutation.isPending ? "退出中..." : "退出"}
              </button>
            </div>
          </div>

          <nav className="flex flex-wrap gap-2 pb-3 md:hidden" aria-label="快捷入口">
            {secondaryNavItems.map((item) => (
              <NavLink
                key={item.to}
                to={item.to}
                className={({ isActive }) => getShortcutLinkClass(isActive)}
              >
                {item.label}
              </NavLink>
            ))}
          </nav>
        </div>
      </header>

      <main
        id="app-main"
        tabIndex={-1}
        className="mx-auto w-full max-w-6xl px-4 py-6 pb-28 sm:px-5 md:py-8 md:pb-8"
      >
        <RouteScrollManager />
        <Outlet />
      </main>

      <nav
        className="fixed inset-x-0 bottom-0 z-30 border-t border-white/80 bg-white/95 backdrop-blur md:hidden"
        aria-label="主导航"
        style={{ paddingBottom: "max(0.75rem, env(safe-area-inset-bottom))" }}
      >
        <div className="mx-auto grid max-w-lg grid-cols-5 gap-2 px-4 pt-3">
          {primaryNavItems.map((item) => (
            <NavLink
              key={item.to}
              to={item.to}
              end={item.end}
              className={({ isActive }) => getBottomNavLinkClass(isActive)}
            >
              {item.label}
            </NavLink>
          ))}
        </div>
      </nav>
    </div>
  );
}
