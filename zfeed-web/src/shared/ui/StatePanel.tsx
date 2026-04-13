import type { ReactNode } from "react";

type StateTone = "neutral" | "soft" | "error";

type StatePanelProps = {
  title: string;
  description?: string;
  tone?: StateTone;
  action?: ReactNode;
};

const toneClassName: Record<StateTone, string> = {
  neutral: "border-dashed border-slate-200 bg-slate-50 text-slate-700",
  soft: "border-slate-200 bg-[linear-gradient(180deg,#f8fcff,#eef7fb)] text-slate-700",
  error: "border-[#ffd7cf] bg-[#fff6f3] text-ember",
};

export function StatePanel({
  title,
  description,
  tone = "neutral",
  action,
}: StatePanelProps) {
  return (
    <section
      className={[
        "rounded-[28px] border px-6 py-10 text-center shadow-card",
        toneClassName[tone],
      ].join(" ")}
    >
      <h2 className="font-display text-2xl font-semibold text-slate-900">{title}</h2>
      {description ? <p className="mt-2 text-sm">{description}</p> : null}
      {action ? <div className="mt-5">{action}</div> : null}
    </section>
  );
}
