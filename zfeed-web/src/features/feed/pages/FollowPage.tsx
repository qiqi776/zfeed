import { useInfiniteQuery } from "@tanstack/react-query";
import { Link } from "react-router-dom";

import { useSessionStore } from "@/entities/session/model/session.store";
import { getFollow } from "@/features/feed/api/feed.api";
import { FeedCard } from "@/features/feed/ui/FeedCard";
import { feedKeys, DEFAULT_FEED_PAGE_SIZE } from "@/shared/lib/query/queryKeys";
import { FeedGridSkeleton } from "@/shared/ui/FeedGridSkeleton";
import { PagedQueryFeedback } from "@/shared/ui/PagedQueryFeedback";
import { PageHeader } from "@/shared/ui/PageHeader";
import { StatePanel } from "@/shared/ui/StatePanel";

export function FollowPage() {
  const currentUserId = useSessionStore((state) => state.user?.userId ?? 0);
  const query = useInfiniteQuery({
    queryKey: feedKeys.follow(currentUserId, DEFAULT_FEED_PAGE_SIZE),
    initialPageParam: { cursor: "0" },
    queryFn: ({ pageParam }) =>
      getFollow({
        cursor: pageParam.cursor,
        page_size: DEFAULT_FEED_PAGE_SIZE,
      }),
    getNextPageParam: (lastPage) => {
      if (!lastPage.has_more) {
        return undefined;
      }
      return { cursor: lastPage.next_cursor };
    },
  });

  const items = query.data?.pages.flatMap((page) => page.items) ?? [];

  return (
    <section className="space-y-5">
      <PageHeader
        eyebrow="Following"
        title="关注流"
        description="只展示你关注用户最近发布的公开内容。"
        aside={
          <span className="rounded-full bg-[#eef7fb] px-4 py-2 text-sm text-slate-600">
            当前 {items.length} 条
          </span>
        }
      />

      <PagedQueryFeedback
        hasItems={items.length > 0}
        isRefreshing={query.isRefetching && !query.isLoading && !query.isFetchingNextPage}
        isFetchingNextPage={query.isFetchingNextPage}
        refreshingTitle="关注流正在同步最新发布"
        refreshingDescription="你关注的作者有新状态时，这里会在后台自动刷新。"
        fetchingNextPageTitle="正在继续加载关注内容"
        fetchingNextPageDescription="下一页关注内容正在追加到当前列表。"
      />

      {query.isLoading ? <FeedGridSkeleton /> : null}
      {query.isError ? (
        <StatePanel
          title="关注流加载失败"
          description={(query.error as Error).message || "请稍后重试"}
          tone="error"
        />
      ) : null}

      {items.length > 0 ? (
        <ul className="grid gap-4 md:grid-cols-2">
          {items.map((item) => (
            <li key={`${item.content_id}-${item.published_at}`}>
              <FeedCard item={item} />
            </li>
          ))}
        </ul>
      ) : !query.isLoading && !query.isError ? (
        <StatePanel
          title="你的关注流还是空的"
          description="先去推荐流或作者主页关注一些人，这里就会开始积累内容。"
          action={
            <Link
              to="/"
              className="inline-flex rounded-full bg-ink px-5 py-2.5 text-sm font-medium text-white transition hover:bg-slate-800"
            >
              去推荐流
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
      ) : items.length > 0 ? (
        <p className="text-sm text-slate-500">没有更多了</p>
      ) : null}
    </section>
  );
}
