import { useMutation } from "@tanstack/react-query";
import { Link, NavLink, Outlet, useNavigate } from "react-router-dom";

import { useSessionStore } from "@/entities/session/model/session.store";
import { logout } from "@/features/auth/api/auth.api";
import { queryClient } from "@/shared/lib/query/queryClient";

function getNavLinkClass(isActive: boolean) {
  return [
    "rounded-full px-3 py-1.5 transition",
    isActive ? "bg-[#eef7fb] text-accent" : "text-slate-600 hover:text-accent",
  ].join(" ");
}

export function AppLayout() {
  const navigate = useNavigate();
  const user = useSessionStore((state) => state.user);
  const clearSession = useSessionStore((state) => state.clearSession);

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
      <header className="sticky top-0 z-10 border-b border-white/70 bg-white/90 backdrop-blur">
        <div className="mx-auto flex h-16 w-full max-w-6xl items-center justify-between px-5">
          <Link to="/" className="font-display text-xl font-semibold tracking-tight text-ink">
            ZFeed Web
          </Link>

          <nav className="flex items-center gap-5 text-sm font-medium">
            <NavLink to="/" end className={({ isActive }) => getNavLinkClass(isActive)}>
              推荐
            </NavLink>
            <NavLink to="/following" className={({ isActive }) => getNavLinkClass(isActive)}>
              关注
            </NavLink>
            <NavLink to="/publish" className={({ isActive }) => getNavLinkClass(isActive)}>
              发布
            </NavLink>
            <NavLink to="/me" className={({ isActive }) => getNavLinkClass(isActive)}>
              我的
            </NavLink>
          </nav>

          <div className="flex items-center gap-3">
            <Link
              to={user?.userId ? `/users/${user.userId}` : "/me"}
              className="max-w-28 truncate text-sm text-slate-600 transition hover:text-accent"
            >
              {user?.nickname ?? "用户"}
            </Link>
            <button
              type="button"
              onClick={() => logoutMutation.mutate()}
              className="rounded-xl border border-slate-200 px-3 py-1.5 text-sm transition hover:border-ember hover:text-ember"
            >
              退出
            </button>
          </div>
        </div>
      </header>

      <main className="mx-auto w-full max-w-6xl px-5 py-8">
        <Outlet />
      </main>
    </div>
  );
}
