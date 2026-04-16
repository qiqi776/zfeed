import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { type ChangeEvent, type FormEvent, useEffect, useRef, useState } from "react";
import { Link, useNavigate } from "react-router-dom";

import { useSessionStore } from "@/entities/session/model/session.store";
import { getMe } from "@/features/auth/api/auth.api";
import { useUnsavedChangesGuard } from "@/features/publish/lib/publishDraft";
import { updateProfile, uploadAvatar } from "@/features/user/api/user.api";
import { isValidEmail, isValidHttpUrl } from "@/shared/lib/form/valueValidation";
import { avatarUploadRule, describeFileValidationRule } from "@/shared/lib/media/fileValidation";
import { userKeys } from "@/shared/lib/query/queryKeys";
import { ImageFallback } from "@/shared/ui/ImageFallback";
import { InlineNotice } from "@/shared/ui/InlineNotice";
import { PageHeader } from "@/shared/ui/PageHeader";
import { StatePanel } from "@/shared/ui/StatePanel";
import { useToast } from "@/shared/ui/toast/toast.store";

type SettingsFormState = {
  nickname: string;
  avatar: string;
  bio: string;
  gender: string;
  email: string;
  birthday: string;
};

function unixToDateInput(value: number) {
  if (!value) {
    return "";
  }
  const date = new Date(value * 1000);
  const year = date.getUTCFullYear();
  const month = String(date.getUTCMonth() + 1).padStart(2, "0");
  const day = String(date.getUTCDate()).padStart(2, "0");
  return `${year}-${month}-${day}`;
}

function dateInputToUnix(value: string) {
  if (!value) {
    return undefined;
  }
  const [year, month, day] = value.split("-").map((item) => Number.parseInt(item, 10));
  if (!year || !month || !day) {
    return undefined;
  }
  return Math.floor(Date.UTC(year, month - 1, day) / 1000);
}

function buildInitialForm(data: Awaited<ReturnType<typeof getMe>>): SettingsFormState {
  return {
    nickname: data.user_info.nickname || "",
    avatar: data.user_info.avatar || "",
    bio: data.user_info.bio || "",
    gender: String(data.user_info.gender || 0),
    email: data.user_info.email || "",
    birthday: unixToDateInput(data.user_info.birthday),
  };
}

function hasSettingsFormChanges(current: SettingsFormState, initial: SettingsFormState | null) {
  if (!initial) {
    return false;
  }

  return (
    current.nickname.trim() !== initial.nickname.trim() ||
    current.avatar.trim() !== initial.avatar.trim() ||
    current.bio.trim() !== initial.bio.trim() ||
    current.gender !== initial.gender ||
    current.email.trim() !== initial.email.trim() ||
    current.birthday !== initial.birthday
  );
}

export function SettingsPage() {
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const { showToast } = useToast();
  const avatarFileInputRef = useRef<HTMLInputElement | null>(null);
  const token = useSessionStore((state) => state.token);
  const expiredAt = useSessionStore((state) => state.expiredAt);
  const sessionUser = useSessionStore((state) => state.user);
  const setSession = useSessionStore((state) => state.setSession);
  const currentUserId = sessionUser?.userId ?? 0;

  const query = useQuery({
    queryKey: userKeys.me(currentUserId),
    queryFn: getMe,
    enabled: currentUserId > 0,
  });

  const [form, setForm] = useState<SettingsFormState>({
    nickname: "",
    avatar: "",
    bio: "",
    gender: "0",
    email: "",
    birthday: "",
  });
  const [initialForm, setInitialForm] = useState<SettingsFormState | null>(null);
  const hasUnsavedChanges = hasSettingsFormChanges(form, initialForm);
  const trimmedNickname = form.nickname.trim();
  const trimmedAvatar = form.avatar.trim();
  const trimmedBio = form.bio.trim();
  const trimmedEmail = form.email.trim();
  const hasValidAvatarUrl = !trimmedAvatar || isValidHttpUrl(trimmedAvatar);
  const hasValidEmailValue = !trimmedEmail || isValidEmail(trimmedEmail);

  useEffect(() => {
    if (!query.data) {
      return;
    }
    const next = buildInitialForm(query.data);
    setForm(next);
    setInitialForm(next);
  }, [query.data]);

  const avatarMutation = useMutation({
    mutationFn: uploadAvatar,
    onSuccess: (res) => {
      setForm((current) => ({ ...current, avatar: res.url }));
      showToast({
        tone: "success",
        title: "头像上传成功",
        description: "新头像地址已经回填到当前表单。",
      });
    },
    onError: (error: Error) => {
      showToast({
        tone: "error",
        title: "头像上传失败",
        description: error.message || "请稍后重试。",
      });
    },
  });

  const updateMutation = useMutation({
    mutationFn: updateProfile,
    onSuccess: async (res) => {
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: userKeys.me(currentUserId) }),
        queryClient.invalidateQueries({ queryKey: userKeys.profilePrefix(currentUserId) }),
      ]);

      if (token && expiredAt) {
        setSession({
          token,
          expiredAt,
          user: {
            userId: res.user_info.user_id,
            nickname: res.user_info.nickname,
            avatar: res.user_info.avatar,
          },
        });
      }

      showToast({
        tone: "success",
        title: "资料已更新",
        description: "公开资料和登录态缓存都已经同步刷新。",
      });
      allowExit(() => navigate("/me", { replace: true }));
    },
    onError: (error: Error) => {
      showToast({
        tone: "error",
        title: "资料更新失败",
        description: error.message || "请稍后重试。",
      });
    },
  });

  const allowExit = useUnsavedChangesGuard(hasUnsavedChanges && !updateMutation.isPending);

  function updateField<Key extends keyof SettingsFormState>(key: Key, value: SettingsFormState[Key]) {
    setForm((current) => ({ ...current, [key]: value }));
  }

  function handleAvatarFileChange(event: ChangeEvent<HTMLInputElement>) {
    const file = event.target.files?.[0];
    event.target.value = "";
    if (!file) {
      return;
    }
    avatarMutation.mutate(file);
  }

  function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!initialForm) {
      return;
    }

    const payload: Parameters<typeof updateProfile>[0] = {};
    if (form.nickname.trim() !== initialForm.nickname.trim()) {
      payload.nickname = trimmedNickname;
    }
    if (form.avatar.trim() !== initialForm.avatar.trim()) {
      payload.avatar = trimmedAvatar;
    }
    if (form.bio.trim() !== initialForm.bio.trim()) {
      payload.bio = trimmedBio;
    }
    if (form.gender !== initialForm.gender) {
      payload.gender = Number(form.gender);
    }
    if (form.email.trim() !== initialForm.email.trim()) {
      payload.email = trimmedEmail;
    }

    if (!trimmedNickname) {
      showToast({
        tone: "error",
        title: "昵称不能为空",
        description: "请至少保留一个可识别的公开昵称。",
      });
      return;
    }

    if (trimmedAvatar && !hasValidAvatarUrl) {
      showToast({
        tone: "error",
        title: "头像地址无效",
        description: "请输入可访问的 http / https 图片地址，或直接重新上传头像。",
      });
      return;
    }

    if (trimmedEmail && !hasValidEmailValue) {
      showToast({
        tone: "error",
        title: "邮箱格式不正确",
        description: "请输入有效邮箱地址，例如 name@example.com。",
      });
      return;
    }

    const nextBirthday = dateInputToUnix(form.birthday);
    const prevBirthday = dateInputToUnix(initialForm.birthday);
    if (nextBirthday !== prevBirthday && nextBirthday !== undefined) {
      payload.birthday = nextBirthday;
    }

    if (Object.keys(payload).length === 0) {
      showToast({
        tone: "info",
        title: "没有变更",
        description: "当前表单内容和已保存资料一致。",
      });
      return;
    }

    updateMutation.mutate(payload);
  }

  if (currentUserId <= 0) {
    return <StatePanel title="当前没有可编辑的登录态" description="请先登录。" tone="error" />;
  }

  if (query.isLoading || !initialForm) {
    return (
      <section className="space-y-6">
        <div className="h-12 w-40 rounded-full bg-white shadow-card" />
        <div className="h-80 rounded-[32px] bg-white shadow-card" />
      </section>
    );
  }

  if (query.isError || !query.data) {
    return (
      <StatePanel
        title="资料设置加载失败"
        description={(query.error as Error)?.message || "请稍后重试"}
        tone="error"
      />
    );
  }

  return (
    <section className="space-y-6">
      <PageHeader
        eyebrow="Settings"
        title="资料设置"
        description="集中维护当前账号的公开头像、昵称、简介和基础资料。"
        aside={
          <Link
            to="/me"
            className="inline-flex rounded-full border border-slate-200 bg-white px-4 py-2 text-sm text-slate-600 transition hover:border-accent hover:text-accent"
          >
            返回我的主页
          </Link>
        }
      />

      <div className="grid gap-6 xl:grid-cols-[0.88fr_1.12fr]">
        <section className="space-y-5 rounded-[32px] border border-slate-200 bg-white p-6 shadow-card">
          <div className="flex items-center gap-4">
            <ImageFallback
              src={form.avatar}
              alt={form.nickname || "当前头像"}
              name={form.nickname || "ZFeed"}
              variant="avatar"
              containerClassName="h-24 w-24 overflow-hidden rounded-full border border-slate-200 bg-slate-50"
              imageClassName="h-full w-full object-cover"
            />
            <div className="space-y-2">
              <p className="font-display text-2xl font-semibold text-slate-900">
                {form.nickname.trim() || "你的公开昵称"}
              </p>
              <p className="text-sm text-slate-500">上传头像后会自动回填到下方表单。</p>
              <button
                type="button"
                onClick={() => avatarFileInputRef.current?.click()}
                aria-describedby="settings-avatar-upload-hint"
                className="inline-flex rounded-full bg-ink px-4 py-2 text-sm font-medium text-white transition hover:bg-slate-800"
              >
                {avatarMutation.isPending ? "上传中..." : "上传头像"}
              </button>
              <input
                ref={avatarFileInputRef}
                type="file"
                accept="image/png,image/jpeg,image/webp"
                tabIndex={-1}
                className="hidden"
                onChange={handleAvatarFileChange}
              />
              <button
                type="button"
                onClick={() => updateField("avatar", "")}
                disabled={!form.avatar.trim()}
                className="rounded-full border border-slate-200 bg-white px-4 py-2 text-sm text-slate-600 transition hover:border-accent hover:text-accent disabled:opacity-60"
              >
                清空头像
              </button>
            </div>
          </div>

          <InlineNotice
            title="当前头像上传会走应用后端"
            description={`这条链路已经可用，适合账号头像；上传文件支持 ${describeFileValidationRule(avatarUploadRule)}。`}
            tone="soft"
          />
          <p id="settings-avatar-upload-hint" className="sr-only">
            当前头像上传会走应用后端，上传文件支持 {describeFileValidationRule(avatarUploadRule)}。
          </p>

          <InlineNotice
            title={hasUnsavedChanges ? "离开当前页面前会提醒你" : "当前没有未保存修改"}
            description={
              hasUnsavedChanges
                ? "如果你刷新、关闭标签页或切换到站内其他页面，都会先弹出确认。"
                : "保存成功后会自动返回我的主页。"
            }
            tone={hasUnsavedChanges ? "soft" : "neutral"}
          />

          <div className="space-y-3 rounded-[24px] bg-slate-50 p-4">
            <div>
              <p className="text-sm text-slate-500">手机号</p>
              <p className="mt-1 text-sm font-medium text-slate-900">{query.data.user_info.mobile}</p>
            </div>
            <div>
              <p className="text-sm text-slate-500">当前公开预览</p>
              <Link
                to={`/users/${currentUserId}`}
                className="mt-1 inline-flex text-sm font-medium text-accent hover:underline"
              >
                查看我的公开主页
              </Link>
            </div>
          </div>
        </section>

        <form className="space-y-5 rounded-[32px] border border-slate-200 bg-white p-6 shadow-card" onSubmit={handleSubmit}>
          <div className="grid gap-5 md:grid-cols-2">
            <label className="block text-sm text-slate-700">
              昵称
              <input
                value={form.nickname}
                onChange={(event) => updateField("nickname", event.target.value)}
                maxLength={64}
                className="mt-1 w-full rounded-2xl border border-slate-200 px-4 py-3 outline-none ring-accent transition focus:ring"
              />
              <span className="mt-2 block text-xs text-slate-400">{trimmedNickname.length}/64</span>
            </label>

            <label className="block text-sm text-slate-700">
              头像地址
              <input
                value={form.avatar}
                onChange={(event) => updateField("avatar", event.target.value)}
                className="mt-1 w-full rounded-2xl border border-slate-200 px-4 py-3 outline-none ring-accent transition focus:ring"
              />
              <span
                className={[
                  "mt-2 block text-xs",
                  trimmedAvatar && !hasValidAvatarUrl ? "text-ember" : "text-slate-400",
                ].join(" ")}
              >
                {trimmedAvatar && !hasValidAvatarUrl
                  ? "请输入可访问的 http / https 图片地址。"
                  : "如果不想重新上传，也可以直接粘贴公开头像地址。"}
              </span>
            </label>
          </div>

          <label className="block text-sm text-slate-700">
            简介
            <textarea
              value={form.bio}
              onChange={(event) => updateField("bio", event.target.value)}
              rows={4}
              maxLength={255}
              className="mt-1 w-full resize-none rounded-[24px] border border-slate-200 px-4 py-3 outline-none ring-accent transition focus:ring"
            />
            <span className="mt-2 block text-xs text-slate-400">{trimmedBio.length}/255</span>
          </label>

          <div className="grid gap-5 md:grid-cols-3">
            <label className="block text-sm text-slate-700">
              性别
              <select
                value={form.gender}
                onChange={(event) => updateField("gender", event.target.value)}
                className="mt-1 w-full rounded-2xl border border-slate-200 px-4 py-3 outline-none ring-accent transition focus:ring"
              >
                <option value="0">未设置</option>
                <option value="1">男</option>
                <option value="2">女</option>
              </select>
            </label>

            <label className="block text-sm text-slate-700 md:col-span-2">
              邮箱
              <input
                value={form.email}
                onChange={(event) => updateField("email", event.target.value)}
                type="email"
                className="mt-1 w-full rounded-2xl border border-slate-200 px-4 py-3 outline-none ring-accent transition focus:ring"
              />
              <span
                className={[
                  "mt-2 block text-xs",
                  trimmedEmail && !hasValidEmailValue ? "text-ember" : "text-slate-400",
                ].join(" ")}
              >
                {trimmedEmail && !hasValidEmailValue
                  ? "请输入有效邮箱地址，例如 name@example.com。"
                  : "邮箱会作为个人资料的一部分保存。"}
              </span>
            </label>
          </div>

          <label className="block text-sm text-slate-700">
            生日
            <input
              value={form.birthday}
              onChange={(event) => updateField("birthday", event.target.value)}
              type="date"
              className="mt-1 w-full rounded-2xl border border-slate-200 px-4 py-3 outline-none ring-accent transition focus:ring"
            />
          </label>

          <div className="flex flex-wrap gap-3">
            <button
              type="submit"
              disabled={updateMutation.isPending}
              className="rounded-full bg-ink px-5 py-2.5 text-sm font-medium text-white transition hover:bg-slate-800 disabled:opacity-60"
            >
              {updateMutation.isPending ? "保存中..." : "保存资料"}
            </button>
            <button
              type="button"
              onClick={() => {
                if (initialForm) {
                  setForm(initialForm);
                }
              }}
              disabled={!hasUnsavedChanges}
              className="rounded-full border border-slate-200 bg-white px-5 py-2.5 text-sm text-slate-600 transition hover:border-accent hover:text-accent"
            >
              重置未保存修改
            </button>
          </div>
        </form>
      </div>
    </section>
  );
}
