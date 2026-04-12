import { FormEvent, useMemo, useState } from "react";
import { useMutation } from "@tanstack/react-query";
import { Link, useNavigate } from "react-router-dom";

import { useSessionStore } from "@/entities/session/model/session.store";
import { register } from "@/features/auth/api/auth.api";
import { useToast } from "@/shared/ui/toast/toast.store";

function todayToUnix() {
  return Math.floor(Date.now() / 1000);
}

export function RegisterPage() {
  const navigate = useNavigate();
  const setSession = useSessionStore((state) => state.setSession);
  const { showToast } = useToast();

  const [mobile, setMobile] = useState("13800000000");
  const [password, setPassword] = useState("123456");
  const [nickname, setNickname] = useState("新用户");
  const [email, setEmail] = useState("demo@example.com");

  const birthday = useMemo(() => todayToUnix(), []);

  const mutation = useMutation({
    mutationFn: register,
    onSuccess: (res) => {
      setSession({
        token: res.token,
        expiredAt: res.expired_at,
        user: { userId: res.user_id, nickname, avatar: "" },
      });
      showToast({
        tone: "success",
        title: "注册成功",
        description: "账号已创建，正在进入首页。",
      });
      navigate("/", { replace: true });
    },
    onError: (e: Error) => {
      showToast({
        tone: "error",
        title: "注册失败",
        description: e.message && e.message !== "注册失败" ? e.message : undefined,
      });
    },
  });

  function onSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();

    mutation.mutate({
      mobile: mobile.trim(),
      password,
      nickname: nickname.trim(),
      avatar: "https://dummyimage.com/300x300/ccd6e5/0b1220.png&text=Z",
      bio: "",
      gender: 0,
      email: email.trim(),
      birthday,
    });
  }

  return (
    <div className="grid min-h-screen place-items-center bg-[radial-gradient(circle_at_80%_15%,#ffe5dc,transparent_40%),radial-gradient(circle_at_20%_80%,#d8f4ff,transparent_40%),linear-gradient(170deg,#f9fcff,#edf5fb)] px-5">
      <section className="w-full max-w-md rounded-3xl border border-white/60 bg-white/90 p-7 shadow-card backdrop-blur">
        <h1 className="font-display text-2xl font-semibold tracking-tight">注册 ZFeed</h1>
        <p className="mt-1 text-sm text-slate-500">先完成基础字段，后续可接入头像上传能力。</p>

        <form className="mt-6 space-y-4" onSubmit={onSubmit}>
          <label className="block text-sm">
            手机号
            <input
              value={mobile}
              onChange={(e) => setMobile(e.target.value)}
              className="mt-1 w-full rounded-xl border border-slate-200 px-3 py-2 outline-none ring-accent transition focus:ring"
            />
          </label>

          <label className="block text-sm">
            密码
            <input
              type="password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              className="mt-1 w-full rounded-xl border border-slate-200 px-3 py-2 outline-none ring-accent transition focus:ring"
            />
          </label>

          <label className="block text-sm">
            昵称
            <input
              value={nickname}
              onChange={(e) => setNickname(e.target.value)}
              className="mt-1 w-full rounded-xl border border-slate-200 px-3 py-2 outline-none ring-accent transition focus:ring"
            />
          </label>

          <label className="block text-sm">
            邮箱
            <input
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              className="mt-1 w-full rounded-xl border border-slate-200 px-3 py-2 outline-none ring-accent transition focus:ring"
            />
          </label>

          <button
            type="submit"
            disabled={mutation.isPending}
            className="w-full rounded-xl bg-ink px-4 py-2.5 text-sm font-medium text-white transition hover:bg-slate-800 disabled:opacity-60"
          >
            {mutation.isPending ? "注册中..." : "注册"}
          </button>
        </form>

        <p className="mt-4 text-sm text-slate-600">
          已有账号？
          <Link to="/login" className="ml-1 text-accent hover:underline">
            去登录
          </Link>
        </p>
      </section>
    </div>
  );
}
