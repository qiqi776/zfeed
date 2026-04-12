import { useInfiniteQuery } from "@tanstack/react-query";

import { useSessionStore } from "@/entities/session/model/session.store";
import { getRecommend } from "@/features/feed/api/feed.api";
import { FeedCard } from "@/features/feed/ui/FeedCard";
import { feedKeys, DEFAULT_FEED_PAGE_SIZE } from "@/shared/lib/query/queryKeys";
import { FeedGridSkeleton } from "@/shared/ui/FeedGridSkeleton";
import { PageHeader } from "@/shared/ui/PageHeader";
import { StatePanel } from "@/shared/ui/StatePanel";

export function RecommendPage() {
  const currentUserId = useSessionStore((state) => state.user?.userId ?? 0);
  const query = useInfiniteQuery({
    queryKey: feedKeys.recommend(currentUserId, DEFAULT_FEED_PAGE_SIZE),
    initialPageParam: { cursor: "0", snapshotId: "" },
    queryFn: ({ pageParam }) =>
      getRecommend({
        cursor: pageParam.cursor,
        page_size: DEFAULT_FEED_PAGE_SIZE,
        snapshot_id: pageParam.snapshotId || undefined,
      }),
    getNextPageParam: (lastPage) => {
      if (!lastPage.has_more) {
        return undefined;
      }
      return {
        cursor: lastPage.next_cursor,
        snapshotId: lastPage.snapshot_id,
      };
    },
  });

  const items = query.data?.pages.flatMap((p) => p.items) ?? [];

  return (
    <section className="space-y-5">
      <PageHeader
        eyebrow="Discover"
        title="推荐流"
        description="已接入游标分页和 snapshot 续读。"
        aside={
          <span className="rounded-full bg-[#eef7fb] px-4 py-2 text-sm font-medium text-slate-700">
            已加载 {items.length} 条
          </span>
        }
      />

      {query.isLoading ? <FeedGridSkeleton /> : null}
      {query.isError ? (
        <StatePanel
          title="推荐流加载失败"
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
      ) : null}

      {!query.isLoading && !query.isError && items.length === 0 ? (
        <StatePanel
          title="推荐流暂时为空"
          description="热榜快照当前没有返回内容，稍后再试。"
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
