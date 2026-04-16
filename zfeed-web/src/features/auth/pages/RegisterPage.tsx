import { type ChangeEvent, type FormEvent, useMemo, useRef, useState } from "react";
import { useMutation } from "@tanstack/react-query";
import { Link, useNavigate } from "react-router-dom";

import { useSessionStore } from "@/entities/session/model/session.store";
import { register } from "@/features/auth/api/auth.api";
import { isLikelyE164Mobile, normalizeMobileInput } from "@/features/auth/lib/mobile";
import { uploadAvatar } from "@/features/user/api/user.api";
import { isValidEmail, isValidHttpUrl } from "@/shared/lib/form/valueValidation";
import { avatarUploadRule, describeFileValidationRule } from "@/shared/lib/media/fileValidation";
import { ImageFallback } from "@/shared/ui/ImageFallback";
import { InlineNotice } from "@/shared/ui/InlineNotice";
import { useToast } from "@/shared/ui/toast/toast.store";

const avatarPresets = [
  {
    label: "薄荷海盐",
    url: "https://dummyimage.com/320x320/dff7f3/0b1220.png&text=M",
  },
  {
    label: "晨雾暖阳",
    url: "https://dummyimage.com/320x320/ffe5dc/0b1220.png&text=W",
  },
  {
    label: "湖面微光",
    url: "https://dummyimage.com/320x320/eef7fb/0b1220.png&text=L",
  },
  {
    label: "夜色书页",
    url: "https://dummyimage.com/320x320/cfd8e6/0b1220.png&text=Z",
  },
] as const;

const genderOptions = [
  { value: "0", label: "未设置" },
  { value: "1", label: "男" },
  { value: "2", label: "女" },
] as const;

const communityNotes = [
  {
    title: "日常可被认真看见",
    description: "这里鼓励稳定表达，而不是用高刺激内容换取短暂注意力。",
  },
  {
    title: "经验可以留下痕迹",
    description: "发布不是一次性动作，更重要的是后续回访、回应和共同成长。",
  },
  {
    title: "关系以温暖为前提",
    description: "界面会优先营造可信、柔和、有人味的社区氛围。",
  },
] as const;

function getSuggestedBirthday() {
  const now = new Date();
  const year = now.getFullYear() - 20;
  const month = String(now.getMonth() + 1).padStart(2, "0");
  const day = String(now.getDate()).padStart(2, "0");
  return `${year}-${month}-${day}`;
}

function birthdayToUnix(value: string) {
  const [year, month, day] = value.split("-").map((part) => Number.parseInt(part, 10));
  if (!year || !month || !day) {
    return NaN;
  }
  return Math.floor(Date.UTC(year, month - 1, day) / 1000);
}

export function RegisterPage() {
  const navigate = useNavigate();
  const setSession = useSessionStore((state) => state.setSession);
  const { showToast } = useToast();
  const avatarFileInputRef = useRef<HTMLInputElement | null>(null);

  const [mobile, setMobile] = useState("");
  const [password, setPassword] = useState("");
  const [nickname, setNickname] = useState("");
  const [email, setEmail] = useState("");
  const [bio, setBio] = useState("");
  const [gender, setGender] = useState<(typeof genderOptions)[number]["value"]>("0");
  const [birthday, setBirthday] = useState(getSuggestedBirthday);
  const [avatar, setAvatar] = useState<string>(avatarPresets[0].url);

  const normalizedMobile = useMemo(() => normalizeMobileInput(mobile), [mobile]);
  const mobileWillNormalize = Boolean(mobile.trim()) && normalizedMobile !== mobile.trim();
  const trimmedNickname = nickname.trim();
  const trimmedEmail = email.trim();
  const trimmedBio = bio.trim();
  const trimmedAvatar = avatar.trim();
  const avatarPresetLabel =
    avatarPresets.find((item) => item.url === trimmedAvatar)?.label ?? "自定义头像";
  const hasValidAvatarUrl = !trimmedAvatar || isValidHttpUrl(trimmedAvatar);
  const hasValidEmailValue = !trimmedEmail || isValidEmail(trimmedEmail);

  const mutation = useMutation({
    mutationFn: register,
    onSuccess: (res) => {
      setSession({
        token: res.token,
        expiredAt: res.expired_at,
        user: { userId: res.user_id, nickname: trimmedNickname, avatar: trimmedAvatar },
      });
      showToast({
        tone: "success",
        title: "注册成功",
        description: "欢迎加入 ZFeed，正在进入首页。",
      });
      navigate("/", { replace: true });
    },
    onError: (error: Error) => {
      showToast({
        tone: "error",
        title: "注册失败",
        description: error.message && error.message !== "注册失败" ? error.message : undefined,
      });
    },
  });

  const avatarUploadMutation = useMutation({
    mutationFn: uploadAvatar,
    onSuccess: (res) => {
      setAvatar(res.url);
      showToast({
        tone: "success",
        title: "头像上传成功",
        description: "上传结果已经回填到当前注册表单。",
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

  function handleAvatarFileChange(event: ChangeEvent<HTMLInputElement>) {
    const file = event.target.files?.[0];
    event.target.value = "";
    if (!file) {
      return;
    }
    avatarUploadMutation.mutate(file);
  }

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
        description: "请输入密码后再创建账号。",
      });
      return;
    }

    if (!trimmedNickname) {
      showToast({
        tone: "error",
        title: "昵称还没有填写",
        description: "先给自己一个社区里可识别的名字。",
      });
      return;
    }

    if (!trimmedEmail) {
      showToast({
        tone: "error",
        title: "邮箱还没有填写",
        description: "请输入一个有效邮箱地址。",
      });
      return;
    }

    if (!hasValidEmailValue) {
      showToast({
        tone: "error",
        title: "邮箱格式不正确",
        description: "请输入一个有效邮箱地址，例如 name@example.com。",
      });
      return;
    }

    if (!trimmedAvatar || !hasValidAvatarUrl) {
      showToast({
        tone: "error",
        title: "头像地址无效",
        description: "请上传头像、选择系统头像，或填写可访问的 http / https 图片地址。",
      });
      return;
    }

    const birthdayUnix = birthdayToUnix(birthday);
    if (!Number.isFinite(birthdayUnix) || birthdayUnix <= 0) {
      showToast({
        tone: "error",
        title: "生日无效",
        description: "请选择一个有效日期。",
      });
      return;
    }

    mutation.mutate({
      mobile: normalizedMobile,
      password,
      nickname: trimmedNickname,
      avatar: trimmedAvatar,
      bio: trimmedBio,
      gender: Number(gender),
      email: trimmedEmail,
      birthday: birthdayUnix,
    });
  }

  return (
    <div className="min-h-screen bg-[radial-gradient(circle_at_12%_16%,#dff7f3,transparent_33%),radial-gradient(circle_at_86%_14%,#ffe5dc,transparent_32%),linear-gradient(165deg,#f8fbff,#edf4fa)] px-5 py-10">
      <div className="mx-auto grid w-full max-w-6xl gap-6 lg:grid-cols-[1.08fr_0.92fr]">
        <section className="overflow-hidden rounded-[36px] border border-white/70 bg-[linear-gradient(180deg,rgba(255,255,255,0.92),rgba(247,251,255,0.88))] shadow-card backdrop-blur">
          <div className="border-b border-white/70 px-6 py-6 md:px-8">
            <p className="text-xs uppercase tracking-[0.24em] text-slate-500">Join The Community</p>
            <h1 className="mt-3 font-display text-4xl font-semibold tracking-tight text-slate-900">
              加入一个共同成长的社区
            </h1>
            <p className="mt-3 max-w-xl text-sm leading-7 text-slate-600">
              在这里分享日常、沉淀经验、回访彼此的表达。界面会保持冷静、柔和和温暖，
              让内容与关系都有耐心地生长。
            </p>
          </div>

          <div className="grid gap-4 px-6 py-6 md:px-8">
            {communityNotes.map((note) => (
              <article
                key={note.title}
                className="rounded-[28px] border border-slate-200/80 bg-white/70 p-5"
              >
                <p className="font-display text-xl font-semibold tracking-tight text-slate-900">
                  {note.title}
                </p>
                <p className="mt-2 text-sm leading-7 text-slate-600">{note.description}</p>
              </article>
            ))}
          </div>
        </section>

        <section className="rounded-[36px] border border-white/70 bg-white/88 p-6 shadow-card backdrop-blur md:p-8">
          <h2 className="font-display text-3xl font-semibold tracking-tight text-slate-900">
            创建账号
          </h2>
          <p className="mt-2 text-sm leading-7 text-slate-500">
            先补齐一份温和、可信的个人名片，后续发布与回访都会从这里开始。
          </p>

          <form className="mt-6 space-y-6" onSubmit={onSubmit}>
            <InlineNotice
              title="注册阶段已支持头像上传"
              description="你可以直接上传头像文件，也可以继续使用系统头像或填写公开图片地址。"
              tone="soft"
            />

            <div className="grid gap-6 xl:grid-cols-[0.96fr_1.04fr]">
              <section className="space-y-5 rounded-[32px] border border-slate-200 bg-[linear-gradient(180deg,#ffffff,#f8fbfd)] p-5">
                <div className="flex items-center gap-4">
                  <ImageFallback
                    src={trimmedAvatar}
                    alt={trimmedNickname || "新用户头像"}
                    name={trimmedNickname || "ZFeed"}
                    variant="avatar"
                    loading="eager"
                    containerClassName="h-20 w-20 overflow-hidden rounded-full border border-white/80 bg-white shadow-card"
                    imageClassName="h-full w-full object-cover"
                  />
                  <div>
                    <p className="text-xs uppercase tracking-[0.2em] text-slate-400">Preview</p>
                    <p className="mt-2 font-display text-2xl font-semibold tracking-tight text-slate-900">
                      {trimmedNickname || "你的社区名片"}
                    </p>
                    <p className="mt-1 text-sm text-slate-500">{avatarPresetLabel}</p>
                  </div>
                </div>

                <div className="grid grid-cols-2 gap-3">
                  {avatarPresets.map((preset) => {
                    const active = preset.url === trimmedAvatar;
                    return (
                      <button
                        key={preset.label}
                        type="button"
                        aria-pressed={active}
                        onClick={() => setAvatar(preset.url)}
                        className={[
                          "overflow-hidden rounded-[24px] border text-left transition",
                          active
                            ? "border-accent bg-[#eef7fb]"
                            : "border-slate-200 bg-white hover:border-accent/50",
                        ].join(" ")}
                      >
                        <ImageFallback
                          src={preset.url}
                          alt={preset.label}
                          name={preset.label}
                          variant="cover"
                          containerClassName="aspect-square bg-slate-100"
                          imageClassName="h-full w-full object-cover"
                        />
                        <span className="block px-3 py-3 text-sm font-medium text-slate-700">
                          {preset.label}
                        </span>
                      </button>
                    );
                  })}
                </div>

                <label className="block text-sm text-slate-700">
                  自定义头像地址
                  <input
                    value={avatar}
                    onChange={(event) => setAvatar(event.target.value)}
                    className="mt-1 w-full rounded-2xl border border-slate-200 px-4 py-3 outline-none ring-accent transition focus:ring"
                    placeholder="https://example.com/avatar.jpg"
                  />
                  <p
                    className={[
                      "mt-2 text-xs",
                      trimmedAvatar && !hasValidAvatarUrl ? "text-ember" : "text-slate-500",
                    ].join(" ")}
                  >
                    {trimmedAvatar && !hasValidAvatarUrl
                      ? "请输入可访问的 http / https 图片地址，或改用系统头像。"
                      : "你可以使用系统头像、上传文件，或填写公开可访问的头像地址。"}
                  </p>
                </label>

                <button
                  type="button"
                  onClick={() => avatarFileInputRef.current?.click()}
                  aria-describedby="register-avatar-upload-hint"
                  className="inline-flex items-center justify-center rounded-full border border-slate-200 bg-white px-4 py-2 text-sm text-slate-600 transition hover:border-accent hover:text-accent"
                >
                  {avatarUploadMutation.isPending ? "上传中..." : "上传头像"}
                </button>
                <input
                  ref={avatarFileInputRef}
                  type="file"
                  accept="image/png,image/jpeg,image/webp"
                  className="hidden"
                  tabIndex={-1}
                  aria-label="上传头像文件"
                  onChange={handleAvatarFileChange}
                />
                <p id="register-avatar-upload-hint" className="text-xs text-slate-500">
                  上传文件支持 {describeFileValidationRule(avatarUploadRule)}。
                </p>
              </section>

              <section className="space-y-5 rounded-[32px] border border-slate-200 bg-white p-5">
                <div className="grid gap-5 md:grid-cols-2">
                  <label className="block text-sm text-slate-700">
                    手机号
                    <input
                      value={mobile}
                      onChange={(event) => setMobile(event.target.value)}
                      className="mt-1 w-full rounded-2xl border border-slate-200 px-4 py-3 outline-none ring-accent transition focus:ring"
                      placeholder="+8613800000000"
                    />
                  </label>

                  <label className="block text-sm text-slate-700">
                    密码
                    <input
                      type="password"
                      value={password}
                      onChange={(event) => setPassword(event.target.value)}
                      className="mt-1 w-full rounded-2xl border border-slate-200 px-4 py-3 outline-none ring-accent transition focus:ring"
                      placeholder="设置一个登录密码"
                    />
                  </label>

                  <label className="block text-sm text-slate-700">
                    昵称
                    <input
                      value={nickname}
                      onChange={(event) => setNickname(event.target.value)}
                      maxLength={64}
                      className="mt-1 w-full rounded-2xl border border-slate-200 px-4 py-3 outline-none ring-accent transition focus:ring"
                      placeholder="你希望被怎么称呼"
                    />
                    <span className="mt-2 block text-xs text-slate-400">{trimmedNickname.length}/64</span>
                  </label>

                  <label className="block text-sm text-slate-700">
                    邮箱
                    <input
                      value={email}
                      onChange={(event) => setEmail(event.target.value)}
                      type="email"
                      className="mt-1 w-full rounded-2xl border border-slate-200 px-4 py-3 outline-none ring-accent transition focus:ring"
                      placeholder="name@example.com"
                    />
                    <span
                      className={[
                        "mt-2 block text-xs",
                        trimmedEmail && !hasValidEmailValue ? "text-ember" : "text-slate-400",
                      ].join(" ")}
                    >
                      {trimmedEmail && !hasValidEmailValue
                        ? "请输入有效邮箱地址，例如 name@example.com。"
                        : "注册成功后会作为联系邮箱保存在个人资料中。"}
                    </span>
                  </label>
                </div>

                {mobileWillNormalize ? (
                  <p className="text-xs text-slate-500">将按 {normalizedMobile} 提交</p>
                ) : null}

                <label className="block text-sm text-slate-700">
                  个人简介
                  <textarea
                    value={bio}
                    onChange={(event) => setBio(event.target.value)}
                    rows={4}
                    maxLength={255}
                    className="mt-1 w-full resize-none rounded-[24px] border border-slate-200 px-4 py-3 outline-none ring-accent transition focus:ring"
                    placeholder="介绍一下你想在这里分享什么，或者你正在经历怎样的成长阶段。"
                  />
                  <span className="mt-2 block text-xs text-slate-400">{trimmedBio.length}/255</span>
                </label>

                <div className="grid gap-5 md:grid-cols-2">
                  <div>
                    <span className="text-sm text-slate-700">性别</span>
                    <div className="mt-2 flex flex-wrap gap-2">
                      {genderOptions.map((option) => {
                        const active = option.value === gender;
                        return (
                          <button
                            key={option.value}
                            type="button"
                            aria-pressed={active}
                            onClick={() => setGender(option.value)}
                            className={[
                              "rounded-full px-4 py-2 text-sm transition",
                              active
                                ? "bg-ink text-white"
                                : "border border-slate-200 bg-white text-slate-600 hover:border-accent hover:text-accent",
                            ].join(" ")}
                          >
                            {option.label}
                          </button>
                        );
                      })}
                    </div>
                  </div>

                  <label className="block text-sm text-slate-700">
                    生日
                    <input
                      value={birthday}
                      onChange={(event) => setBirthday(event.target.value)}
                      type="date"
                      className="mt-2 w-full rounded-2xl border border-slate-200 px-4 py-3 outline-none ring-accent transition focus:ring"
                    />
                  </label>
                </div>

                <button
                  type="submit"
                  disabled={mutation.isPending}
                  className="w-full rounded-2xl bg-ink px-4 py-3 text-sm font-medium text-white transition hover:bg-slate-800 disabled:opacity-60"
                >
                  {mutation.isPending ? "注册中..." : "注册并进入社区"}
                </button>
              </section>
            </div>
          </form>

          <p className="mt-5 text-sm text-slate-600">
            已有账号？
            <Link to="/login" className="ml-1 text-accent hover:underline">
              去登录
            </Link>
          </p>
        </section>
      </div>
    </div>
  );
}
