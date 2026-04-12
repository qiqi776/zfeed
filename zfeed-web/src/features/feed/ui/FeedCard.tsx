import { Link } from "react-router-dom";

import type { FeedItem } from "@/features/feed/api/feed.api";
import { ImageFallback } from "@/shared/ui/ImageFallback";

function formatPublishedAt(timestamp: number) {
  if (!timestamp) {
    return "刚刚";
  }

  return new Intl.DateTimeFormat("zh-CN", {
    month: "2-digit",
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
  }).format(timestamp * 1000);
}

export function FeedCard({ item }: { item: FeedItem }) {
  const authorName = item.author_name || `用户${item.author_id}`;

  return (
    <article className="overflow-hidden rounded-[28px] border border-slate-200 bg-white shadow-card transition hover:-translate-y-0.5">
      <div className="flex items-center justify-between gap-3 border-b border-slate-100 px-4 py-3">
        <Link
          to={`/users/${item.author_id}`}
          className="flex min-w-0 items-center gap-3 transition hover:text-accent"
        >
          <ImageFallback
            src={item.author_avatar}
            alt={authorName}
            name={authorName}
            variant="avatar"
            containerClassName="h-11 w-11 overflow-hidden rounded-full bg-slate-100"
            imageClassName="h-full w-full object-cover"
          />
          <div className="min-w-0">
            <p className="truncate text-sm font-semibold text-slate-900">{authorName}</p>
            <p className="text-xs text-slate-500">{formatPublishedAt(item.published_at)}</p>
          </div>
        </Link>

        <span className="rounded-full bg-[#eef7fb] px-3 py-1 text-[11px] font-medium uppercase tracking-[0.18em] text-slate-500">
          {item.content_type === 20 ? "Video" : "Article"}
        </span>
      </div>

      <Link to={`/content/${item.content_id}`} className="block">
        <ImageFallback
          src={item.cover_url}
          alt={item.title || "内容封面"}
          containerClassName="aspect-[16/9] bg-slate-100"
          imageClassName="h-full w-full object-cover"
        />

        <div className="space-y-3 p-4">
          <h2 className="line-clamp-2 text-lg font-semibold text-slate-900">
            {item.title || "未命名内容"}
          </h2>
          <div className="flex items-center justify-between text-sm text-slate-500">
            <span>{item.is_liked ? "你已点赞" : `点赞 ${item.like_count}`}</span>
            <span className="font-medium text-accent">查看详情</span>
          </div>
        </div>
      </Link>
    </article>
  );
}
