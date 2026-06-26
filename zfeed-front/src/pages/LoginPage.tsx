import { createElement, useState } from "react";
import { PageShell } from "../runtime/PageShell";
import { login as loginRequest } from "../runtime/apiClient";
import { saveAuthSession } from "../runtime/authStore";
import { navigateTo } from "../runtime/navigation";
import { AuthHomeBackdrop } from "./AuthHomeBackdrop";
import { buildAuthSiblingHref, resolveSafeAuthNextPath } from "./authNavigation";
import { isValidMobile } from "./mobile";
import { sharedGlassBodyClass, sharedGlassStyles } from "./sharedGlassStyles";

type LoginErrors = {
    phone?: string;
    password?: string;
    form?: string;
};

export function LoginPage() {
    const [phone, setPhone] = useState("");
    const [password, setPassword] = useState("");
    const [errors, setErrors] = useState<LoginErrors>({});
    const [isSubmitting, setIsSubmitting] = useState(false);

    return createElement(
        PageShell,
        {
            title: "zfeed - 登录",
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
                        "auth-sheet w-full max-w-[420px] min-h-[calc(100dvh-2rem)] overflow-hidden rounded-t-[28px] rounded-b-none md:min-h-0 md:rounded-[28px] p-6 md:p-8"
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
                                createElement("div", { className: "mt-1 font-meta-xs text-on-surface-variant" }, "继续浏览")
                            )
                        ),
                        createElement("a", {
                            className:
                                "glass-button-ghost inline-flex min-h-11 items-center rounded-full px-4 py-2.5 text-primary font-label-sm",
                            href: buildAuthSiblingHref("/register")
                        }, "注册")
                    ),
                    createElement("div", { className: "mt-6" },
                        createElement("h1", {
                            className: "font-display text-[34px] leading-tight text-on-surface"
                        }, "登录 zfeed"),
                        createElement("p", {
                            className: "mt-2 text-[15px] leading-7 text-on-surface-variant"
                        }, "返回内容流，继续你上次停下的地方。")
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

                            const nextErrors: LoginErrors = {};

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

                            if (Object.keys(nextErrors).length > 0) {
                                setErrors(nextErrors);
                                return;
                            }

                            setIsSubmitting(true);
                            setErrors({});

                            try {
                                const response = await loginRequest<{
                                    user_id: number;
                                    token: string;
                                    expired_at: number;
                                    nickname: string;
                                    avatar: string;
                                }>({
                                    mobile: phone.trim(),
                                    password
                                });

                                saveAuthSession({
                                    token: response.token,
                                    expiredAt: response.expired_at,
                                    user: {
                                        userId: response.user_id,
                                        nickname: response.nickname,
                                        avatar: response.avatar
                                    }
                                });
                                navigateTo(resolveSafeAuthNextPath("/home"));
                            } catch {
                                setErrors({ form: "手机号或密码不正确" });
                            } finally {
                                setIsSubmitting(false);
                            }
                        }
                    },
                        renderField("手机号", "phone", "tel", "请输入手机号", "tel", phone, setPhone, errors.phone),
                        renderField("密码", "password", "password", "请输入密码", "current-password", password, setPassword, errors.password),
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
                            }, isSubmitting ? "登录中" : "登录")
                        )
                    ),
                    createElement("div", { className: "mt-6 flex items-center justify-between gap-4 border-t border-white/30 pt-4" },
                        createElement("span", { className: "text-label-sm text-on-surface-variant" }, "还没有账号？"),
                        createElement("a", {
                            className:
                                "glass-button-ghost inline-flex min-h-11 items-center rounded-full px-4 py-2.5 text-primary font-label-sm",
                            href: buildAuthSiblingHref("/register")
                        }, "去注册")
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
