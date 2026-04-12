import type { ReactNode } from "react";

type PersonalSpaceHeroProps = {
  eyebrow?: string;
  identity?: string;
  title: string;
  description?: string;
  media?: ReactNode;
  aside?: ReactNode;
};

type PersonalMetricItem = {
  label: string;
  value: ReactNode;
  hint?: string;
};

type PersonalMetricGridProps = {
  items: PersonalMetricItem[];
  columns?: 3 | 4 | 5;
};

type PersonalSpaceSectionProps = {
  eyebrow?: string;
  title: string;
  description?: string;
  badge?: ReactNode;
  actions?: ReactNode;
  children: ReactNode;
};

type PersonalSpaceInfoCardProps = {
  label: string;
  value: ReactNode;
  description?: string;
};

const metricGridClassName: Record<NonNullable<PersonalMetricGridProps["columns"]>, string> = {
  3: "md:grid-cols-3",
  4: "md:grid-cols-4",
  5: "md:grid-cols-5",
};

export function PersonalSpaceHero({
  eyebrow,
  identity,
  title,
  description,
  media,
  aside,
}: PersonalSpaceHeroProps) {
  return (
    <section className="overflow-hidden rounded-[32px] border border-white/70 bg-white shadow-card">
      <div className="bg-[radial-gradient(circle_at_top_left,#dff7f3,transparent_30%),radial-gradient(circle_at_bottom_right,#ffe5dc,transparent_38%),linear-gradient(180deg,#fbfdff,#f2f7fb)] px-6 py-8 md:px-8">
        <div className="flex flex-col gap-6 lg:flex-row lg:items-end lg:justify-between">
          <div className="flex flex-col gap-4 sm:flex-row sm:items-center">
            {media ? <div className="shrink-0">{media}</div> : null}

            <div>
              {eyebrow ? (
                <p className="text-xs uppercase tracking-[0.24em] text-slate-500">{eyebrow}</p>
              ) : null}
              {identity ? (
                <p className="mt-2 text-xs uppercase tracking-[0.2em] text-slate-400">
                  {identity}
                </p>
              ) : null}
              <h2 className="mt-2 font-display text-3xl font-semibold tracking-tight text-slate-900">
                {title}
              </h2>
              {description ? (
                <p className="mt-3 max-w-2xl text-sm leading-7 text-slate-600">{description}</p>
              ) : null}
            </div>
          </div>

          {aside ? <div className="shrink-0">{aside}</div> : null}
        </div>
      </div>
    </section>
  );
}

export function PersonalMetricGrid({
  items,
  columns = 4,
}: PersonalMetricGridProps) {
  return (
    <div className={["grid gap-3", metricGridClassName[columns]].join(" ")}>
      {items.map((item) => (
        <article
          key={item.label}
          className="rounded-[24px] border border-slate-200 bg-white p-4 shadow-card"
        >
          <p className="text-sm text-slate-500">{item.label}</p>
          <p className="mt-2 text-2xl font-semibold text-slate-900">{item.value}</p>
          {item.hint ? <p className="mt-2 text-sm leading-6 text-slate-500">{item.hint}</p> : null}
        </article>
      ))}
    </div>
  );
}

export function PersonalSpaceSection({
  eyebrow,
  title,
  description,
  badge,
  actions,
  children,
}: PersonalSpaceSectionProps) {
  return (
    <section className="space-y-5 rounded-[32px] border border-slate-200 bg-white p-6 shadow-card">
      <div className="flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
        <div>
          {eyebrow ? (
            <p className="text-xs uppercase tracking-[0.22em] text-slate-500">{eyebrow}</p>
          ) : null}
          <h2 className="mt-2 font-display text-2xl font-semibold text-slate-900">{title}</h2>
          {description ? <p className="mt-2 text-sm leading-6 text-slate-500">{description}</p> : null}
        </div>

        {badge || actions ? (
          <div className="flex shrink-0 flex-wrap items-center gap-3">
            {badge}
            {actions}
          </div>
        ) : null}
      </div>

      {children}
    </section>
  );
}

export function PersonalSpaceInfoCard({
  label,
  value,
  description,
}: PersonalSpaceInfoCardProps) {
  return (
    <article className="rounded-[24px] border border-slate-200 bg-[linear-gradient(180deg,#ffffff,#f8fbfd)] p-4">
      <p className="text-sm text-slate-500">{label}</p>
      <p className="mt-2 text-lg font-semibold text-slate-900">{value}</p>
      {description ? <p className="mt-2 text-sm leading-6 text-slate-500">{description}</p> : null}
    </article>
  );
}
