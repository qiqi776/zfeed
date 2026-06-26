import { createElement } from "react";
import { PageShell } from "../runtime/PageShell";
import { clearAuthSession, readAuthSession } from "../runtime/authStore";
import { navigateTo } from "../runtime/navigation";
import { PageState } from "./PageState";
import { sharedGlassBodyClass, sharedGlassStyles } from "./sharedGlassStyles";

export function SettingsPage() {
    const session = readAuthSession();

    return createElement(
        PageShell,
        {
            title: "zfeed - 设置",
            htmlClass: "light",
            bodyClass: sharedGlassBodyClass,
            styles: sharedGlassStyles
        },
        createElement("div", { className: "page-root" },
            createElement("main", { className: "min-h-screen px-4 py-6 md:px-6 md:py-10" },
                createElement("section", { className: "auth-sheet feed-transition feed-ready mx-auto w-full max-w-2xl rounded-[28px] p-6 md:p-8" },
                    createElement("a", { className: "text-label-sm text-primary", href: "/home" }, "返回首页"),
                    createElement("h1", { className: "mt-5 font-display text-[34px] leading-tight text-on-surface" }, "设置"),
                    session
                        ? createElement("div", { className: "mt-6 grid gap-3" },
                        createElement("div", { className: "rounded-2xl border border-white/40 bg-white/35 px-4 py-4" },
                            createElement("div", { className: "font-label-sm text-on-surface" }, "账号与安全"),
                            createElement("p", { className: "mt-1 text-[14px] leading-6 text-on-surface-variant" }, "管理登录状态和基础账号信息。"),
                            createElement("button", {
                                className: "glass-button-ghost mt-4 inline-flex min-h-11 items-center justify-center rounded-full px-4 py-2.5 text-red-700 font-label-sm active:scale-95",
                                type: "button",
                                onClick: () => {
                                    clearAuthSession();
                                    navigateTo("/login");
                                }
                            }, "退出登录")
                        ),
                        createElement("div", { className: "rounded-2xl border border-white/40 bg-white/35 px-4 py-4" },
                            createElement("div", { className: "font-label-sm text-on-surface" }, "内容偏好"),
                            createElement("p", { className: "mt-1 text-[14px] leading-6 text-on-surface-variant" }, "控制推荐流和通知偏好。")
                        )
                    )
                        : createElement(PageState, {
                            state: "auth-required",
                            description: "登录后才能管理设置。",
                            actionHref: `/login?next=${encodeURIComponent("/settings")}`,
                            actionLabel: "去登录"
                        })
                )
            )
        )
    );
}
