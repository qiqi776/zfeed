import { useInfiniteQuery } from "@tanstack/react-query";
import { Link } from "react-router-dom";

import { useSessionStore } from "@/entities/session/model/session.store";
import { getUserPublish, type FeedItem } from "@/features/feed/api/feed.api";
import { feedKeys, DEFAULT_FEED_PAGE_SIZE } from "@/shared/lib/query/queryKeys";
import { FeedGridSkeleton } from "@/shared/ui/FeedGridSkeleton";
import { ImageFallback } from "@/shared/ui/ImageFallback";
import { PageHeader } from "@/shared/ui/PageHeader";
import {
  PersonalMetricGrid,
  PersonalSpaceHero,
  PersonalSpaceInfoCard,
  PersonalSpaceSection,
} from "@/shared/ui/PersonalSpace";
import { StatePanel } from "@/shared/ui/StatePanel";

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

export function StudioPage() {
  const currentUserId = useSessionStore((state) => state.user?.userId ?? 0);

  const query = useInfiniteQuery({
    queryKey: feedKeys.studioPublish(currentUserId, DEFAULT_FEED_PAGE_SIZE),
    enabled: currentUserId > 0,
    initialPageParam: { cursor: "0" },
    queryFn: ({ pageParam }) =>
      getUserPublish({
        user_id: currentUserId,
        cursor: pageParam.cursor,
        page_size: DEFAULT_FEED_PAGE_SIZE,
      }),
    getNextPageParam: (lastPage) =>
      lastPage.has_more ? { cursor: lastPage.next_cursor } : undefined,
  });

  const items = query.data?.pages.flatMap((page) => page.items) ?? [];

  if (currentUserId <= 0) {
    return (
      <StatePanel
        title="当前登录态不可用"
        description="需要先拿到当前用户 ID，才能加载我的发布。"
        tone="error"
      />
    );
  }

  return (
    <section className="space-y-6">
      <PageHeader
        eyebrow="Studio"
        title="我的公开发布"
        description="当前先聚焦公开作品浏览与回访，删除链路仍等待后端补齐。"
      />

      <PersonalSpaceHero
        eyebrow="Public Studio"
        identity={`当前用户 ID ${currentUserId}`}
        title="我的公开作品空间"
        description="这里集中展示当前可被他人看见的公开内容，用来做回访、自检和发布后确认。"
        aside={
          <div className="flex flex-wrap gap-3 lg:max-w-sm lg:justify-end">
            <Link
              to="/publish"
              className="rounded-full bg-ink px-4 py-2 text-sm font-medium text-white transition hover:bg-slate-800"
            >
              去发布
            </Link>
            <Link
              to="/me"
              className="rounded-full border border-slate-200 bg-white px-4 py-2 text-sm text-slate-600 transition hover:border-accent hover:text-accent"
            >
              返回我的主页
            </Link>
          </div>
        }
      />

      <PersonalMetricGrid
        items={[
          { label: "公开内容", value: items.length, hint: "当前已加载到页面里的公开作品数量。" },
          { label: "读取范围", value: "仅公开", hint: "列表目前只读取公开发布内容，不含私密内容。" },
          { label: "删除能力", value: "待补齐", hint: "删除接口未打通前，这里只提供浏览和回访。" },
        ]}
        columns={3}
      />

      <div className="grid gap-6 lg:grid-cols-[1.25fr_0.75fr]">
        <PersonalSpaceSection
          eyebrow="Publish Feed"
          title="公开发布"
          description="按公开发布时间回看已经对外展示的作品。"
          badge={
            <span className="rounded-full bg-[#eef7fb] px-4 py-2 text-sm font-medium text-slate-700">
              已加载 {items.length} 条
            </span>
          }
        >
          {query.isLoading ? <FeedGridSkeleton /> : null}

          {query.isError ? (
            <StatePanel
              title="发布列表加载失败"
              description={(query.error as Error).message || "请稍后重试"}
              tone="error"
            />
          ) : null}

          {items.length > 0 ? (
            <ul className="grid gap-4 lg:grid-cols-2">
              {items.map((item) => (
                <li key={`${item.content_id}-${item.published_at}`}>
                  <StudioContentCard item={item} />
                </li>
              ))}
            </ul>
          ) : null}

          {!query.isLoading && !query.isError && items.length === 0 ? (
            <StatePanel
              title="你还没有公开发布内容"
              description="可以先去发布文章或视频，随后回到这里查看公开内容列表。"
              action={
                <Link
                  to="/publish"
                  className="inline-flex rounded-full bg-ink px-5 py-2.5 text-sm font-medium text-white transition hover:bg-slate-800"
                >
                  去发布
                </Link>
              }
            />
          ) : null}

          {query.hasNextPage ? (
            <button
              type="button"
              onClick={() => query.fetchNextPage()}
              disabled={query.isFetchingNextPage}
              className="rounded-xl border border-slate-300 bg-white px-4 py-2 text-sm transition hover:border-accent hover:text-accent disabled:opacity-60"
            >
              {query.isFetchingNextPage ? "加载中..." : "加载更多"}
            </button>
          ) : null}
        </PersonalSpaceSection>

        <PersonalSpaceSection
          eyebrow="Boundary"
          title="当前边界"
          description="在删除和私密读取链路补齐前，这里先保持语义清楚。"
        >
          <div className="space-y-3">
            <PersonalSpaceInfoCard
              label="内容范围"
              value="只展示公开内容"
              description="当前数据来自 `user publish feed`，不会把私密内容伪装成已可管理。"
            />
            <PersonalSpaceInfoCard
              label="当前用途"
              value="浏览、回访、自检"
              description="你可以确认作品是否已进入公开可见链路，并回访详情页或作者主页。"
            />
            <PersonalSpaceInfoCard
              label="删除能力"
              value="后端未就绪"
              description="删除接口真正可用之前，这里不会提供假删除按钮。"
            />
          </div>
        </PersonalSpaceSection>
      </div>
    </section>
  );
}

function StudioContentCard({ item }: { item: FeedItem }) {
  const contentType = item.content_type === 20 ? "视频" : "文章";
  const authorName = item.author_name || `用户 ${item.author_id}`;

  return (
    <article className="overflow-hidden rounded-[28px] border border-slate-200 bg-white shadow-card">
      <ImageFallback
        src={item.cover_url}
        alt={item.title || "内容封面"}
        containerClassName="aspect-[16/9] bg-slate-100"
        imageClassName="h-full w-full object-cover"
      />

      <div className="space-y-4 p-5">
        <div className="flex items-center justify-between gap-3">
          <span className="rounded-full bg-[#eef7fb] px-3 py-1 text-xs font-medium text-slate-600">
            {contentType}
          </span>
          <span className="text-xs text-slate-400">{formatPublishedAt(item.published_at)}</span>
        </div>

        <div>
          <h2 className="line-clamp-2 text-xl font-semibold text-slate-900">
            {item.title || "未命名内容"}
          </h2>
          <p className="mt-2 text-sm text-slate-500">作者：{authorName}</p>
        </div>

        <div className="grid grid-cols-2 gap-3 rounded-3xl bg-slate-50 p-4">
          <div>
            <p className="text-sm text-slate-500">点赞</p>
            <p className="mt-1 text-xl font-semibold text-slate-900">{item.like_count}</p>
          </div>
          <div>
            <p className="text-sm text-slate-500">状态</p>
            <p className="mt-1 text-xl font-semibold text-slate-900">公开发布</p>
          </div>
        </div>

        <div className="flex flex-wrap gap-3">
          <Link
            to={`/content/${item.content_id}`}
            className="rounded-full bg-ink px-4 py-2 text-sm font-medium text-white transition hover:bg-slate-800"
          >
            查看详情
          </Link>
          <Link
            to={`/users/${item.author_id}`}
            className="rounded-full border border-slate-200 bg-white px-4 py-2 text-sm text-slate-600 transition hover:border-accent hover:text-accent"
          >
            查看主页
          </Link>
        </div>
      </div>
    </article>
  );
}
