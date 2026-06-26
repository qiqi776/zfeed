import { createElement } from "react";
import { PageShell } from "../runtime/PageShell";
import { readAuthSession } from "../runtime/authStore";
import { PageState } from "./PageState";
import { sharedGlassBodyClass, sharedGlassStyles } from "./sharedGlassStyles";

export function ComposePage() {
    const session = readAuthSession();

    return createElement(
        PageShell,
        {
            title: "zfeed - 发布",
            htmlClass: "light",
            bodyClass: sharedGlassBodyClass,
            styles: sharedGlassStyles
        },
        createElement("div", { className: "page-root" },
            createElement("main", { className: "min-h-screen px-4 py-6 md:px-6 md:py-10" },
                createElement("section", { className: "auth-sheet feed-transition feed-ready mx-auto w-full max-w-3xl rounded-[28px] p-6 md:p-8" },
                    createElement("a", { className: "text-label-sm text-primary", href: "/home" }, "返回首页"),
                    createElement("h1", { className: "mt-5 font-display text-[34px] leading-tight text-on-surface" }, "发布"),
                    session
                        ? createElement("div", { className: "mt-6 grid gap-4", "data-compose-form": "true" },
                            createElement("input", {
                                className: "auth-field rounded-2xl px-4 py-3 text-[15px] text-on-surface",
                                maxLength: 80,
                                name: "title",
                                placeholder: "标题",
                                type: "text"
                            }),
                            createElement("textarea", {
                                className: "auth-field min-h-28 resize-none rounded-2xl px-4 py-3 text-[15px] text-on-surface",
                                maxLength: 255,
                                name: "description",
                                placeholder: "写一段简介，方便读者快速了解内容...",
                                rows: 4
                            }),
                            createElement("input", {
                                className: "auth-field rounded-2xl px-4 py-3 text-[15px] text-on-surface",
                                name: "cover",
                                placeholder: "封面 URL，可选",
                                type: "url"
                            }),
                            createElement("textarea", {
                                className: "auth-field min-h-56 resize-none rounded-2xl px-4 py-3 text-[15px] text-on-surface",
                                maxLength: 5000,
                                name: "content",
                                placeholder: "写下你的想法...",
                                rows: 10
                            }),
                            createElement("button", {
                                className: "glass-button-primary inline-flex min-h-11 items-center justify-center rounded-full px-5 py-3 text-white font-label-sm",
                                type: "button"
                            }, "发布")
                        )
                        : createElement(PageState, {
                            state: "auth-required",
                            description: "发布需要登录。",
                            actionHref: `/login?next=${encodeURIComponent("/compose")}`,
                            actionLabel: "去登录"
                        })
                )
            )
        )
    );
}
