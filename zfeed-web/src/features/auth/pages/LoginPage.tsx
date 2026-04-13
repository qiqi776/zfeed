import { FormEvent, useState } from "react";
import { useMutation } from "@tanstack/react-query";
import { Link, useLocation, useNavigate } from "react-router-dom";

import { useSessionStore } from "@/entities/session/model/session.store";
import { login } from "@/features/auth/api/auth.api";
import { useToast } from "@/shared/ui/toast/toast.store";

export function LoginPage() {
  const navigate = useNavigate();
  const location = useLocation();
  const setSession = useSessionStore((state) => state.setSession);
  const { showToast } = useToast();

  const [mobile, setMobile] = useState("13800000000");
  const [password, setPassword] = useState("123456");

  const mutation = useMutation({
    mutationFn: login,
    onSuccess: (res) => {
      setSession({
        token: res.token,
        expiredAt: res.expired_at,
        user: {
          userId: res.user_id,
          nickname: res.nickname,
          avatar: res.avatar,
        },
      });
      const next = new URLSearchParams(location.search).get("next") || "/";
      showToast({
        tone: "success",
        title: "登录成功",
        description: "正在进入下一页。",
      });
      navigate(next, { replace: true });
    },
    onError: (e: Error) => {
      showToast({
        tone: "error",
        title: "登录失败",
        description: e.message && e.message !== "登录失败" ? e.message : undefined,
      });
    },
  });

  function onSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    mutation.mutate({ mobile: mobile.trim(), password });
  }

  return (
    <div className="grid min-h-screen place-items-center bg-[radial-gradient(circle_at_15%_20%,#dff7f3,transparent_40%),radial-gradient(circle_at_85%_15%,#ffe5dc,transparent_38%),linear-gradient(160deg,#f8fbff,#ecf3fa)] px-5">
      <section className="w-full max-w-md rounded-3xl border border-white/60 bg-white/85 p-7 shadow-card backdrop-blur">
        <h1 className="font-display text-2xl font-semibold tracking-tight">登录 ZFeed</h1>
        <p className="mt-1 text-sm text-slate-500">先打通登录态，后续页面可直接复用鉴权。</p>

        <form className="mt-6 space-y-4" onSubmit={onSubmit}>
          <label className="block text-sm">
            手机号
            <input
              value={mobile}
              onChange={(e) => setMobile(e.target.value)}
              className="mt-1 w-full rounded-xl border border-slate-200 px-3 py-2 outline-none ring-accent transition focus:ring"
              placeholder="13800000000"
            />
          </label>

          <label className="block text-sm">
            密码
            <input
              type="password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              className="mt-1 w-full rounded-xl border border-slate-200 px-3 py-2 outline-none ring-accent transition focus:ring"
              placeholder="请输入密码"
            />
          </label>

          <button
            type="submit"
            disabled={mutation.isPending}
            className="w-full rounded-xl bg-ink px-4 py-2.5 text-sm font-medium text-white transition hover:bg-slate-800 disabled:opacity-60"
          >
            {mutation.isPending ? "登录中..." : "登录"}
          </button>
        </form>

        <p className="mt-4 text-sm text-slate-600">
          没有账号？
          <Link to="/register" className="ml-1 text-accent hover:underline">
            去注册
          </Link>
        </p>
      </section>
    </div>
  );
}
