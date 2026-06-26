import { createElement } from "react";

type PageStateKind = "loading" | "empty" | "error" | "auth-required";

type PageStateProps = {
    state: PageStateKind;
    title?: string;
    description: string;
    actionHref?: string;
    actionLabel?: string;
    className?: string;
};

const defaultClassName = "mt-6 min-w-0 rounded-2xl border border-white/40 bg-white/35 px-4 py-4 text-[14px] leading-7 text-on-surface-variant";

export function PageState({ state, title, description, actionHref, actionLabel, className = defaultClassName }: PageStateProps) {
    return createElement("div", {
        className,
        "data-page-state": state
    },
        title ? createElement("div", { className: "break-words font-label-sm text-on-surface" }, title) : null,
        title
            ? createElement("p", { className: "mt-1 break-words text-[14px] leading-7 text-on-surface-variant" }, description)
            : createElement("span", { className: "break-words" }, description),
        actionHref && actionLabel
            ? createElement("a", { className: "ml-2 text-primary font-label-sm", href: actionHref }, actionLabel)
            : null
    );
}
