import { createElement } from "react";

type ConfirmDialogProps = {
    open: boolean;
    title: string;
    message: string;
    confirmLabel?: string;
    danger?: boolean;
    onConfirm: () => void;
    onCancel: () => void;
};

export function ConfirmDialog({ open, title, message, confirmLabel = "确定", danger = false, onConfirm, onCancel }: ConfirmDialogProps) {
    if (!open) return null;

    return createElement("div", {
        className: "fixed inset-0 z-50 flex items-center justify-center bg-black/40",
        onClick: onCancel,
    },
        createElement("div", {
            className: "bg-white rounded-xl shadow-2xl p-6 w-full max-w-sm mx-4",
            onClick: (e: Event) => e.stopPropagation(),
        },
            createElement("h3", { className: "text-lg font-semibold text-slate-800" }, title),
            createElement("p", { className: "mt-2 text-sm text-slate-600" }, message),
            createElement("div", { className: "mt-5 flex justify-end gap-3" },
                createElement("button", {
                    onClick: onCancel,
                    className: "px-4 py-2 text-sm font-medium text-slate-600 bg-slate-100 rounded-lg hover:bg-slate-200 transition-colors",
                }, "取消"),
                createElement("button", {
                    onClick: onConfirm,
                    className: `px-4 py-2 text-sm font-medium text-white rounded-lg transition-colors ${danger ? "bg-red-600 hover:bg-red-700" : "bg-blue-600 hover:bg-blue-700"}`,
                }, confirmLabel),
            ),
        ),
    );
}
