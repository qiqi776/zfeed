import { createElement } from "react";

export function NotFoundPage() {
    return createElement(
        "main",
        {
            className:
                "min-h-screen flex items-center justify-center text-on-surface font-body-md antialiased overflow-x-hidden px-6",
            "data-page-state": "error"
        },
        createElement("div", { className: "text-center" },
            createElement("h1", { className: "font-display text-[40px] leading-tight text-on-surface" }, "页面不存在"),
            createElement("p", { className: "mt-3 text-on-surface-variant" }, "你访问的地址已经不再使用旧的 .html 路径。"),
            createElement("a", { className: "inline-flex mt-6 text-primary font-label-sm", href: "/home" }, "返回首页")
        )
    );
}
