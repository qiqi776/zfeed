import { createElement, useEffect, useState } from "react";

type ToastType = "success" | "error" | "info";

type ToastItem = {
    id: number;
    message: string;
    type: ToastType;
};

let toastId = 0;
let addToastFn: ((item: ToastItem) => void) | null = null;

export function showToast(message: string, type: ToastType = "error") {
    if (addToastFn) {
        addToastFn({ id: ++toastId, message, type });
    }
}

const typeStyles: Record<ToastType, string> = {
    success: "bg-green-600 text-white",
    error: "bg-red-600 text-white",
    info: "bg-slate-800 text-white",
};

const typeIcons: Record<ToastType, string> = {
    success: "check_circle",
    error: "error",
    info: "info",
};

function ToastItemView({ item, onDone }: { item: ToastItem; onDone: () => void }) {
    useEffect(() => {
        const timer = setTimeout(onDone, 3000);
        return () => clearTimeout(timer);
    }, [onDone]);

    return createElement("div", {
        className: `flex items-center gap-3 px-5 py-3 rounded-xl shadow-lg ${typeStyles[item.type]} animate-slide-up`,
        style: { animation: "toastSlideUp 0.3s ease-out", minWidth: "280px", maxWidth: "420px" },
    },
        createElement("span", { className: "material-symbols-outlined text-[20px] shrink-0" }, typeIcons[item.type]),
        createElement("span", { className: "text-sm font-medium flex-1" }, item.message),
    );
}

const toastStyles = `
@keyframes toastSlideUp {
    from { opacity: 0; transform: translateY(16px); }
    to { opacity: 1; transform: translateY(0); }
}
`;

export function ToastContainer() {
    const [toasts, setToasts] = useState<ToastItem[]>([]);

    useEffect(() => {
        addToastFn = (item: ToastItem) => setToasts((prev) => [...prev.slice(-4), item]);
        return () => { addToastFn = null; };
    }, []);

    const removeToast = (id: number) => setToasts((prev) => prev.filter((t) => t.id !== id));

    if (toasts.length === 0) return null;

    return createElement("div", null,
        createElement("style", null, toastStyles),
        createElement("div", {
            className: "fixed bottom-6 right-6 z-[9999] flex flex-col gap-3",
            style: { pointerEvents: "none" },
        },
            toasts.map((t) =>
                createElement("div", { key: t.id, style: { pointerEvents: "auto" } },
                    createElement(ToastItemView, { item: t, onDone: () => removeToast(t.id) }),
                )
            ),
        ),
    );
}
