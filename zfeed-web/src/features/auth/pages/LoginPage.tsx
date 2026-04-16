import { FormEvent, useState } from "react";
import { useMutation } from "@tanstack/react-query";
import { Link, useLocation, useNavigate } from "react-router-dom";

import { useSessionStore } from "@/entities/session/model/session.store";
import { login } from "@/features/auth/api/auth.api";
import { isLikelyE164Mobile, normalizeMobileInput } from "@/features/auth/lib/mobile";
import { InlineNotice } from "@/shared/ui/InlineNotice";
import { useToast } from "@/shared/ui/toast/toast.store";

export function LoginPage() {
  const navigate = useNavigate();
  const location = useLocation();
  const setSession = useSessionStore((state) => state.setSession);
  const { showToast } = useToast();

  const [mobile, setMobile] = useState("");
  const [password, setPassword] = useState("");
  const normalizedMobile = normalizeMobileInput(mobile);
  const mobileWillNormalize = Boolean(mobile.trim()) && normalizedMobile !== mobile.trim();

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

    if (!normalizedMobile || !isLikelyE164Mobile(normalizedMobile)) {
      showToast({
        tone: "error",
        title: "手机号格式不正确",
        description: "请输入有效手机号，支持直接输入 13800000000，会自动按 +86 规范提交。",
      });
      return;
    }

    if (!password.trim()) {
      showToast({
        tone: "error",
        title: "密码不能为空",
        description: "请输入密码后再登录。",
      });
      return;
    }

    mutation.mutate({ mobile: normalizedMobile, password });
  }

  return (
    <div className="min-h-screen bg-[radial-gradient(circle_at_14%_18%,#dff7f3,transparent_34%),radial-gradient(circle_at_88%_16%,#ffe5dc,transparent_34%),linear-gradient(160deg,#f7fbff,#edf4fa)] px-5 py-10">
      <div className="mx-auto grid w-full max-w-6xl gap-6 lg:grid-cols-[0.96fr_1.04fr]">
        <section className="overflow-hidden rounded-[36px] border border-white/70 bg-[linear-gradient(180deg,rgba(255,255,255,0.9),rgba(247,251,255,0.9))] shadow-card backdrop-blur">
          <div className="border-b border-white/70 px-6 py-6 md:px-8">
            <p className="text-xs uppercase tracking-[0.24em] text-slate-500">Welcome Back</p>
            <h1 className="mt-3 font-display text-4xl font-semibold tracking-tight text-slate-900">
              回到 ZFeed
            </h1>
            <p className="mt-3 max-w-xl text-sm leading-7 text-slate-600">
              在一个冷静、柔和、温暖的社区里继续表达、回访和共同成长。
            </p>
          </div>

          <div className="grid gap-4 px-6 py-6 md:px-8">
            <FeatureCard
              title="安静回访"
              description="从推荐、关注到收藏，继续回到那些真正值得二次停留的内容。"
            />
            <FeatureCard
              title="稳定登录态"
              description="登录后会保留你的会话、关系状态和内容回访路径。"
            />
            <FeatureCard
              title="共同成长"
              description="这里不是喧闹流量场，而是沉淀经验和日常观察的高质量社区。"
            />
          </div>
        </section>

        <section className="rounded-[36px] border border-white/70 bg-white/88 p-6 shadow-card backdrop-blur md:p-8">
          <h2 className="font-display text-3xl font-semibold tracking-tight text-slate-900">
            登录账号
          </h2>
          <p className="mt-2 text-sm leading-7 text-slate-500">
            支持直接输入 `13800000000`，前端会自动按 `+86` 规范手机号提交。
          </p>

          <form className="mt-6 space-y-5" onSubmit={onSubmit}>
            <InlineNotice
              title="当前建议使用手机号登录"
              description="为了和后端注册链路保持一致，登录时也会优先按规范手机号格式处理。"
              tone="soft"
            />

            <label className="block text-sm text-slate-700">
              手机号
              <input
                value={mobile}
                onChange={(event) => setMobile(event.target.value)}
                className="mt-1 w-full rounded-2xl border border-slate-200 px-4 py-3 outline-none ring-accent transition focus:ring"
                placeholder="+8613800000000"
              />
            </label>

            {mobileWillNormalize ? (
              <p className="text-xs text-slate-500">将按 {normalizedMobile} 提交</p>
            ) : null}

            <label className="block text-sm text-slate-700">
              密码
              <input
                type="password"
                value={password}
                onChange={(event) => setPassword(event.target.value)}
                className="mt-1 w-full rounded-2xl border border-slate-200 px-4 py-3 outline-none ring-accent transition focus:ring"
                placeholder="请输入密码"
              />
            </label>

            <button
              type="submit"
              disabled={mutation.isPending}
              className="w-full rounded-2xl bg-ink px-4 py-3 text-sm font-medium text-white transition hover:bg-slate-800 disabled:opacity-60"
            >
              {mutation.isPending ? "登录中..." : "登录"}
            </button>
          </form>

          <p className="mt-5 text-sm text-slate-600">
            还没有账号？
            <Link to="/register" className="ml-1 text-accent hover:underline">
              去注册
            </Link>
          </p>
        </section>
      </div>
    </div>
  );
}

function FeatureCard({ title, description }: { title: string; description: string }) {
  return (
    <article className="rounded-[28px] border border-slate-200/70 bg-white/70 p-5">
      <p className="font-display text-xl font-semibold tracking-tight text-slate-900">{title}</p>
      <p className="mt-2 text-sm leading-7 text-slate-600">{description}</p>
    </article>
  );
}
