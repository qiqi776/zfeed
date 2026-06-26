import { createElement, type ChangeEvent, useState } from "react";
import { PageShell } from "../runtime/PageShell";
import { register as registerRequest } from "../runtime/apiClient";
import { saveAuthSession } from "../runtime/authStore";
import { navigateTo } from "../runtime/navigation";
import { AuthHomeBackdrop } from "./AuthHomeBackdrop";
import { buildAuthSiblingHref, resolveSafeAuthNextPath } from "./authNavigation";
import { isValidMobile } from "./mobile";
import { sharedGlassBodyClass, sharedGlassStyles } from "./sharedGlassStyles";

type RegisterErrors = {
    phone?: string;
    password?: string;
    nickname?: string;
    avatarUrl?: string;
    email?: string;
    bio?: string;
    form?: string;
};

export function RegisterPage() {
    const [phone, setPhone] = useState("");
    const [password, setPassword] = useState("");
    const [nickname, setNickname] = useState("");
    const [avatarUrl, setAvatarUrl] = useState("");
    const [email, setEmail] = useState("");
    const [bio, setBio] = useState("");
    const [errors, setErrors] = useState<RegisterErrors>({});
    const [isSubmitting, setIsSubmitting] = useState(false);

    return createElement(
        PageShell,
        {
            title: "zfeed - 注册",
            htmlClass: "light",
            bodyClass: sharedGlassBodyClass,
            styles: sharedGlassStyles
        },
        createElement("div", { className: "page-root auth-gateway-root" },
            createElement(AuthHomeBackdrop),
            createElement("main", {
                className: "relative z-10 min-h-screen px-4 py-4 md:px-6 md:py-8 flex items-end md:items-center justify-center"
            },
                createElement("section", {
                    className:
                        "auth-sheet w-full max-w-[460px] min-h-[calc(100dvh-2rem)] overflow-hidden rounded-t-[28px] rounded-b-none md:min-h-0 md:rounded-[28px] p-6 md:p-8"
                },
                    createElement("div", { className: "flex items-center justify-between gap-4" },
                        createElement("a", {
                            className: "flex items-center gap-3",
                            href: "/"
                        },
                            createElement("div", {
                                className:
                                    "flex h-11 w-11 items-center justify-center rounded-2xl bg-gradient-to-br from-primary-container to-primary text-white shadow-md"
                            }, "z"),
                            createElement("div", null,
                                createElement("div", { className: "font-display text-[24px] leading-none text-on-surface" }, "zfeed"),
                                createElement("div", { className: "mt-1 font-meta-xs text-on-surface-variant" }, "创建账号")
                            )
                        ),
                        createElement("a", {
                            className:
                                "glass-button-ghost inline-flex min-h-11 items-center rounded-full px-4 py-2.5 text-primary font-label-sm",
                            href: buildAuthSiblingHref("/login")
                        }, "登录")
                    ),
                    createElement("div", { className: "mt-6" },
                        createElement("h1", {
                            className: "font-display text-[34px] leading-tight text-on-surface"
                        }, "创建 zfeed 账号"),
                        createElement("p", {
                            className: "mt-2 text-[15px] leading-7 text-on-surface-variant"
                        }, "注册后就能继续浏览、关注和发布。")
                    ),
                    errors.form
                        ? createElement("div", { className: "auth-alert auth-alert-error mt-4 rounded-2xl px-4 py-3 text-[13px]" }, errors.form)
                        : null,
                    createElement("form", {
                        className: "mt-6 grid gap-4",
                        noValidate: true,
                        onSubmit: async (event) => {
                            event.preventDefault();
                            if (isSubmitting) {
                                return;
                            }

                            const nextErrors: RegisterErrors = {};

                            if (!phone.trim()) {
                                nextErrors.phone = "请输入手机号";
                            } else if (!isValidMobile(phone.trim())) {
                                nextErrors.phone = "请输入有效手机号";
                            }

                            if (!password.trim()) {
                                nextErrors.password = "请输入密码";
                            } else if (password.length < 6) {
                                nextErrors.password = "密码至少 6 位";
                            }

                            if (nickname.length > 64) {
                                nextErrors.nickname = "昵称最多 64 字";
                            }

                            if (bio.length > 255) {
                                nextErrors.bio = "简介最多 255 字";
                            }

                            if (email && !isValidEmail(email)) {
                                nextErrors.email = "请输入有效邮箱";
                            }

                            if (avatarUrl && !isLikelyUrl(avatarUrl)) {
                                nextErrors.avatarUrl = "请输入有效头像链接";
                            }

                            if (Object.keys(nextErrors).length > 0) {
                                setErrors(nextErrors);
                                return;
                            }

                            setIsSubmitting(true);
                            setErrors({});

                            try {
                                const response = await registerRequest<{
                                    user_id: number;
                                    token: string;
                                    expired_at: number;
                                }>({
                                    mobile: phone.trim(),
                                    password,
                                    nickname: nickname.trim() || undefined,
                                    avatar: avatarUrl.trim() || undefined,
                                    email: email.trim() || undefined,
                                    bio: bio.trim() || undefined
                                });

                                saveAuthSession({
                                    token: response.token,
                                    expiredAt: response.expired_at,
                                    user: {
                                        userId: response.user_id,
                                        nickname: nickname.trim() || undefined,
                                        avatar: avatarUrl.trim() || undefined
                                    }
                                });
                                navigateTo(resolveSafeAuthNextPath("/me"));
                            } catch {
                                setErrors({ form: "注册失败，请检查信息后重试" });
                            } finally {
                                setIsSubmitting(false);
                            }
                        }
                    },
                        renderField("手机号", "phone", "tel", "请输入手机号", "tel", phone, setPhone, errors.phone),
                        renderField("密码", "password", "password", "请输入密码", "new-password", password, setPassword, errors.password),
                        renderField("昵称", "nickname", "text", "可选", "nickname", nickname, setNickname, errors.nickname),
                        renderField("头像 URL", "avatarUrl", "url", "可选", "url", avatarUrl, setAvatarUrl, errors.avatarUrl),
                        renderField("邮箱", "email", "email", "可选", "email", email, setEmail, errors.email),
                        createElement("label", { className: "flex flex-col gap-2" },
                            createElement("span", { className: "auth-label" }, "简介"),
                            createElement("textarea", {
                                className:
                                    "auth-field min-h-28 resize-none rounded-2xl px-4 py-3 text-[15px] text-on-surface",
                                id: "bio",
                                name: "bio",
                                placeholder: "可选",
                                rows: 4,
                                "aria-invalid": Boolean(errors.bio),
                                value: bio,
                                onChange: (event: ChangeEvent<HTMLTextAreaElement>) => setBio(event.target.value)
                            }),
                            errors.bio ? createElement("span", { className: "text-[12px] text-red-600" }, errors.bio) : null
                        ),
                        createElement("div", { className: "flex items-center justify-between gap-3 pt-1" },
                            createElement("a", {
                                className: "text-label-sm text-primary",
                                href: "/home"
                            }, "先浏览内容"),
                            createElement("button", {
                                className:
                                    "glass-button-primary inline-flex min-h-11 flex-1 items-center justify-center rounded-full px-5 py-3 text-white font-label-sm disabled:opacity-70 disabled:cursor-not-allowed",
                                disabled: isSubmitting,
                                type: "submit"
                            }, isSubmitting ? "注册中" : "注册")
                        )
                    ),
                    createElement("div", { className: "mt-6 border-t border-white/30 pt-4" },
                        createElement("a", {
                            className:
                                "glass-button-ghost inline-flex min-h-11 w-full items-center justify-center rounded-full px-4 py-2.5 text-primary font-label-sm",
                            href: buildAuthSiblingHref("/login")
                        }, "去登录")
                    )
                )
            )
        )
    );
}

function renderField(
    label: string,
    id: string,
    type: string,
    placeholder: string,
    autoComplete: string,
    value: string,
    onValueChange: (value: string) => void,
    error?: string
) {
    return createElement("label", { className: "flex flex-col gap-2" },
        createElement("span", { className: "auth-label" }, label),
        createElement("input", {
            className: "auth-field rounded-2xl px-4 py-3 text-[15px] text-on-surface",
            id,
            name: id,
            placeholder,
            type,
            autoComplete,
            inputMode: type === "tel" ? "tel" : undefined,
            "aria-invalid": Boolean(error),
            value,
            onChange: (event) => onValueChange(event.target.value)
        }),
        error ? createElement("span", { className: "text-[12px] text-red-600" }, error) : null
    );
}

function isValidEmail(value: string) {
    return /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(value);
}

function isLikelyUrl(value: string) {
    try {
        const url = new URL(value);
        return url.protocol === "http:" || url.protocol === "https:";
    } catch {
        return false;
    }
}
