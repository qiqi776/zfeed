import { useInfiniteQuery } from "@tanstack/react-query";
import { Link } from "react-router-dom";

import { useSessionStore } from "@/entities/session/model/session.store";
import { getUserFavorite } from "@/features/feed/api/feed.api";
import { FeedCard } from "@/features/feed/ui/FeedCard";
import { feedKeys, DEFAULT_FEED_PAGE_SIZE } from "@/shared/lib/query/queryKeys";
import { FeedGridSkeleton } from "@/shared/ui/FeedGridSkeleton";
import { PageHeader } from "@/shared/ui/PageHeader";
import {
  PersonalMetricGrid,
  PersonalSpaceHero,
  PersonalSpaceInfoCard,
  PersonalSpaceSection,
} from "@/shared/ui/PersonalSpace";
import { StatePanel } from "@/shared/ui/StatePanel";

export function FavoritesPage() {
  const currentUserId = useSessionStore((state) => state.user?.userId ?? 0);

  const query = useInfiniteQuery({
    queryKey: feedKeys.favorites(currentUserId, DEFAULT_FEED_PAGE_SIZE),
    enabled: currentUserId > 0,
    initialPageParam: { cursor: "0" },
    queryFn: ({ pageParam }) =>
      getUserFavorite({
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
        description="需要先拿到当前用户 ID，才能加载收藏内容。"
        tone="error"
        action={
          <Link
            to="/login"
            className="inline-flex rounded-full bg-ink px-5 py-2.5 text-sm font-medium text-white transition hover:bg-slate-800"
          >
            去登录
          </Link>
        }
      />
    );
  }

  return (
    <section className="space-y-6">
      <PageHeader
        eyebrow="Favorites"
        title="我的收藏"
        description="这里保留你想回看的内容，统一回访收藏和详情状态。"
      />

      <PersonalSpaceHero
        eyebrow="Collection"
        identity={`当前用户 ID ${currentUserId}`}
        title="我的收藏架"
        description="把想回看的文章和视频放进同一个空间里，当前按游标分页持续加载。"
        aside={
          <div className="flex flex-wrap gap-3 lg:max-w-sm lg:justify-end">
            <Link
              to="/"
              className="rounded-full bg-ink px-4 py-2 text-sm font-medium text-white transition hover:bg-slate-800"
            >
              去推荐流
            </Link>
            <Link
              to="/following"
              className="rounded-full border border-slate-200 bg-white px-4 py-2 text-sm text-slate-600 transition hover:border-accent hover:text-accent"
            >
              看关注流
            </Link>
          </div>
        }
      />

      <PersonalMetricGrid
        items={[
          { label: "已加载", value: items.length, hint: "当前已经加载到页面里的收藏内容数量。" },
          { label: "可见范围", value: "仅自己", hint: "收藏列表当前只服务登录用户本人查看。" },
          {
            label: "同步状态",
            value: query.isFetching ? "同步中" : "已同步",
            hint: "详情页取消收藏后，这里会跟随更新。",
          },
        ]}
        columns={3}
      />

      <div className="grid gap-6 lg:grid-cols-[1.25fr_0.75fr]">
        <PersonalSpaceSection
          eyebrow="Collection Feed"
          title="收藏内容"
          description="这里优先展示你已经收藏并可继续回访的内容。"
          badge={
            <span className="rounded-full bg-[#eef7fb] px-4 py-2 text-sm font-medium text-slate-700">
              已加载 {items.length} 条
            </span>
          }
        >
          {query.isLoading ? <FeedGridSkeleton /> : null}

          {query.isError ? (
            <StatePanel
              title="收藏列表加载失败"
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
              title="你还没有收藏内容"
              description="先去推荐流或详情页收藏一些作品，这里会自动积累。"
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
          ) : null}
        </PersonalSpaceSection>

        <PersonalSpaceSection
          eyebrow="Notes"
          title="收藏说明"
          description="这一列用来明确收藏语义和后续回访路径。"
        >
          <div className="space-y-3">
            <PersonalSpaceInfoCard
              label="页面语义"
              value="个人收藏架"
              description="这里不是操作日志，而是你为后续回看保留的内容空间。"
            />
            <PersonalSpaceInfoCard
              label="状态同步"
              value="跨页保持一致"
              description="在详情页取消收藏后，收藏页和其他相关列表会同步更新。"
            />
            <PersonalSpaceInfoCard
              label="下一步"
              value="回到流或详情页继续挑选"
              description="如果收藏为空，可以从推荐流和关注流继续补充内容。"
            />
          </div>
        </PersonalSpaceSection>
      </div>
    </section>
  );
}
