import { Link } from "react-router-dom";

import { PageHeader } from "@/shared/ui/PageHeader";
import { StatePanel } from "@/shared/ui/StatePanel";

const publishChoices = [
  {
    to: "/publish/article",
    eyebrow: "Article",
    title: "发布文章",
    description: "适合长文本、图文说明、复盘总结和结构化表达。",
  },
  {
    to: "/publish/video",
    eyebrow: "Video",
    title: "发布视频",
    description: "支持视频源文件直传，也支持继续使用现成 URL 兜底发布。",
  },
];

export function PublishPage() {
  return (
    <section className="space-y-6">
      <PageHeader
        eyebrow="Create"
        title="发布内容"
        description="先选择内容类型。当前版本优先把公开文章和公开视频发布主链打通。"
      />

      <StatePanel
        title="当前版本仍聚焦公开发布"
        description="内容详情和我的发布链路已经支持公开内容稳定读取；封面和视频源文件也可以走上传签名链，手填 URL 作为兜底方式保留。"
        tone="soft"
      />

      <div className="grid gap-4 lg:grid-cols-2">
        {publishChoices.map((choice) => (
          <Link
            key={choice.to}
            to={choice.to}
            className="group overflow-hidden rounded-[32px] border border-slate-200 bg-white p-6 shadow-card transition hover:-translate-y-0.5 hover:border-accent"
          >
            <div className="rounded-[28px] bg-[radial-gradient(circle_at_top_left,#dff7f3,transparent_35%),radial-gradient(circle_at_bottom_right,#ffe5dc,transparent_35%),linear-gradient(160deg,#f8fbff,#edf5fb)] p-6">
              <p className="text-xs uppercase tracking-[0.24em] text-slate-500">{choice.eyebrow}</p>
              <h2 className="mt-3 font-display text-3xl font-semibold tracking-tight text-slate-900">
                {choice.title}
              </h2>
              <p className="mt-2 text-sm leading-7 text-slate-600">{choice.description}</p>
              <span className="mt-6 inline-flex rounded-full bg-ink px-5 py-2.5 text-sm font-medium text-white transition group-hover:bg-slate-800">
                进入
              </span>
            </div>
          </Link>
        ))}
      </div>

      <div className="flex flex-wrap gap-3">
        <Link
          to="/studio"
          className="rounded-full border border-slate-200 bg-white px-4 py-2 text-sm text-slate-600 transition hover:border-accent hover:text-accent"
        >
          去我的发布
        </Link>
        <Link
          to="/favorites"
          className="rounded-full border border-slate-200 bg-white px-4 py-2 text-sm text-slate-600 transition hover:border-accent hover:text-accent"
        >
          去我的收藏
        </Link>
      </div>
    </section>
  );
}
