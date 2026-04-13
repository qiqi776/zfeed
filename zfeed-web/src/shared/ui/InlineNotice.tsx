import type { ReactNode } from "react";

type InlineNoticeTone = "neutral" | "soft" | "error";

type InlineNoticeProps = {
  title: string;
  description?: string;
  tone?: InlineNoticeTone;
  action?: ReactNode;
};

const toneClassName: Record<InlineNoticeTone, string> = {
  neutral: "border-dashed border-slate-200 bg-slate-50 text-slate-600",
  soft: "border-slate-200 bg-[linear-gradient(180deg,#f8fcff,#eef7fb)] text-slate-600",
  error: "border-[#ffd7cf] bg-[#fff6f3] text-ember",
};

export function InlineNotice({
  title,
  description,
  tone = "neutral",
  action,
}: InlineNoticeProps) {
  return (
    <section className={["rounded-[24px] border px-4 py-4", toneClassName[tone]].join(" ")}>
      <p className="text-sm font-semibold text-slate-900">{title}</p>
      {description ? <p className="mt-1 text-sm leading-6">{description}</p> : null}
      {action ? <div className="mt-3">{action}</div> : null}
    </section>
  );
}
