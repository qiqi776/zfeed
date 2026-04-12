import { useEffect } from "react";

import { type ToastItem, type ToastTone, useToastStore } from "@/shared/ui/toast/toast.store";

const toneClassName: Record<ToastTone, string> = {
  info: "border-slate-200 bg-white text-slate-700",
  success: "border-[#cfece5] bg-[linear-gradient(180deg,#f6fffc,#edf9f5)] text-slate-700",
  error: "border-[#ffd7cf] bg-[#fff6f3] text-slate-700",
};

const toneBadgeClassName: Record<ToastTone, string> = {
  info: "bg-[#eef7fb] text-slate-600",
  success: "bg-[#e9fbf7] text-accent",
  error: "bg-[#ffe3db] text-ember",
};

const toneLabel: Record<ToastTone, string> = {
  info: "提示",
  success: "成功",
  error: "错误",
};

function ToastCard({ toast }: { toast: ToastItem }) {
  const dismissToast = useToastStore((state) => state.dismissToast);

  useEffect(() => {
    const timer = window.setTimeout(() => {
      dismissToast(toast.id);
    }, toast.durationMs ?? 3600);

    return () => {
      window.clearTimeout(timer);
    };
  }, [dismissToast, toast.durationMs, toast.id]);

  return (
    <article
      role={toast.tone === "error" ? "alert" : "status"}
      className={[
        "pointer-events-auto rounded-[24px] border px-4 py-4 shadow-card backdrop-blur-sm",
        toneClassName[toast.tone ?? "info"],
      ].join(" ")}
    >
      <div className="flex items-start justify-between gap-4">
        <div className="min-w-0">
          <span
            className={[
              "inline-flex rounded-full px-2.5 py-1 text-[11px] font-semibold uppercase tracking-[0.18em]",
              toneBadgeClassName[toast.tone ?? "info"],
            ].join(" ")}
          >
            {toneLabel[toast.tone ?? "info"]}
          </span>
          <p className="mt-3 text-sm font-semibold text-slate-900">{toast.title}</p>
          {toast.description ? (
            <p className="mt-1 text-sm leading-6 text-slate-600">{toast.description}</p>
          ) : null}
        </div>

        <button
          type="button"
          onClick={() => dismissToast(toast.id)}
          className="rounded-full border border-slate-200 bg-white px-2.5 py-1 text-xs text-slate-500 transition hover:border-slate-300 hover:text-slate-700"
        >
          关闭
        </button>
      </div>
    </article>
  );
}

export function ToastViewport() {
  const toasts = useToastStore((state) => state.toasts);

  if (toasts.length === 0) {
    return null;
  }

  return (
    <section
      aria-live="polite"
      className="pointer-events-none fixed inset-x-4 top-4 z-50 flex flex-col gap-3 sm:left-auto sm:right-6 sm:top-6 sm:w-full sm:max-w-sm"
    >
      {toasts.map((toast) => (
        <ToastCard key={toast.id} toast={toast} />
      ))}
    </section>
  );
}
