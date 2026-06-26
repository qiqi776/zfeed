import { createElement } from "react";

export function AuthHomeBackdrop() {
    return createElement("div", {
        "aria-hidden": "true",
        className: "auth-home-backdrop",
        title: "zfeed 首页雾化背景"
    },
        createElement("div", { className: "auth-home-shell" },
            createElement("div", { className: "auth-home-topbar" },
                createElement("div", { className: "auth-logo-dot" }),
                createElement("div", { className: "auth-home-wordmark" }, "zfeed"),
                createElement("div", { className: "auth-home-search" }),
                createElement("div", { className: "auth-home-icon" }),
                createElement("div", { className: "auth-home-icon" })
            ),
            createElement("div", { className: "auth-home-grid" },
                createElement("div", { className: "auth-home-left" },
                    renderBackdropLine("w-24"),
                    renderBackdropLine("w-32"),
                    renderBackdropLine("w-28"),
                    renderBackdropLine("w-36")
                ),
                createElement("div", { className: "auth-home-feed" },
                    renderBackdropComposer(),
                    renderBackdropCard("Jax Lee", "用 AI 构建产品：30 天从 0 到 1", "w-5/6"),
                    renderBackdropCard("Lin Xia", "创作工具的未来：AI 成为协作副驾", "w-3/4")
                ),
                createElement("div", { className: "auth-home-right" },
                    renderBackdropLine("w-32"),
                    renderBackdropLine("w-28"),
                    renderBackdropLine("w-36"),
                    renderBackdropLine("w-24")
                )
            )
        )
    );
}

function renderBackdropLine(widthClass: string) {
    return createElement("div", { className: `h-9 rounded-2xl bg-white/45 ${widthClass}` });
}

function renderBackdropComposer() {
    return createElement("div", { className: "auth-backdrop-card" },
        createElement("div", { className: "flex items-center gap-3" },
            createElement("div", { className: "h-10 w-10 rounded-full bg-white/55" }),
            createElement("div", { className: "h-11 flex-1 rounded-2xl bg-white/50" })
        ),
        createElement("div", { className: "mt-5 grid grid-cols-4 gap-3" },
            createElement("div", { className: "h-14 rounded-2xl bg-white/45" }),
            createElement("div", { className: "h-14 rounded-2xl bg-white/42" }),
            createElement("div", { className: "h-14 rounded-2xl bg-white/40" }),
            createElement("div", { className: "h-14 rounded-2xl bg-white/38" })
        )
    );
}

function renderBackdropCard(author: string, title: string, widthClass: string) {
    return createElement("article", { className: "auth-backdrop-card" },
        createElement("div", { className: "flex items-center gap-3" },
            createElement("div", { className: "h-10 w-10 rounded-full bg-white/55" }),
            createElement("div", null,
                createElement("div", { className: "font-label-sm text-on-surface" }, author),
                createElement("div", { className: "mt-1 h-3 w-20 rounded-full bg-white/45" })
            )
        ),
        createElement("div", { className: "mt-5 font-title text-[18px] leading-7 text-on-surface" }, title),
        createElement("div", { className: `mt-4 h-3 rounded-full bg-white/55 ${widthClass}` }),
        createElement("div", { className: "mt-3 h-3 w-2/3 rounded-full bg-white/45" }),
        createElement("div", { className: "mt-5 h-36 rounded-[24px] bg-white/42" })
    );
}
