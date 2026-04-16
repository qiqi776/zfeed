import { useInfiniteQuery, useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Link } from "react-router-dom";

import { useSessionStore } from "@/entities/session/model/session.store";
import { getMe } from "@/features/auth/api/auth.api";
import { deleteContent } from "@/features/content/api/content.api";
import { getUserPublish, type FeedItem } from "@/features/feed/api/feed.api";
import { contentDeletionCopy } from "@/shared/lib/content/actionCopy";
import {
  captureQuerySnapshots,
  removeContentAcrossCollections,
  restoreQuerySnapshots,
} from "@/shared/lib/query/cacheSync";
import { feedKeys, userKeys, DEFAULT_FEED_PAGE_SIZE } from "@/shared/lib/query/queryKeys";
import { FeedGridSkeleton } from "@/shared/ui/FeedGridSkeleton";
import { ImageFallback } from "@/shared/ui/ImageFallback";
import { InlineNotice } from "@/shared/ui/InlineNotice";
import { PagedQueryFeedback } from "@/shared/ui/PagedQueryFeedback";
import { PageHeader } from "@/shared/ui/PageHeader";
import {
  PersonalMetricGrid,
  PersonalSpaceHero,
  PersonalSpaceInfoCard,
  PersonalSpaceSection,
} from "@/shared/ui/PersonalSpace";
import { StatePanel } from "@/shared/ui/StatePanel";
import { useToast } from "@/shared/ui/toast/toast.store";

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
  const queryClient = useQueryClient();
  const { showToast } = useToast();
  const meQuery = useQuery({
    queryKey: userKeys.me(currentUserId),
    queryFn: getMe,
    enabled: currentUserId > 0,
  });

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
  const aggregateContentCount = meQuery.data?.content_count ?? items.length;

  const deleteMutation = useMutation({
    mutationFn: deleteContent,
    onMutate: async (contentId) => {
      await Promise.all([
        queryClient.cancelQueries({ queryKey: feedKeys.recommendPrefix() }),
        queryClient.cancelQueries({ queryKey: feedKeys.followPrefix() }),
        queryClient.cancelQueries({ queryKey: feedKeys.favoritesPrefix() }),
        queryClient.cancelQueries({ queryKey: feedKeys.userPublishPrefix() }),
        queryClient.cancelQueries({ queryKey: feedKeys.studioPublishPrefix() }),
      ]);

      const previousSnapshots = captureQuerySnapshots(queryClient, [
        feedKeys.recommendPrefix(),
        feedKeys.followPrefix(),
        feedKeys.favoritesPrefix(),
        feedKeys.userPublishPrefix(),
        feedKeys.studioPublishPrefix(),
      ]);

      removeContentAcrossCollections(queryClient, contentId);
      return { previousSnapshots };
    },
    onError: (error, _contentId, context) => {
      if (context?.previousSnapshots) {
        restoreQuerySnapshots(queryClient, context.previousSnapshots);
      }
      showToast({
        tone: "error",
        title: "删除内容失败",
        description: error.message && error.message !== "删除内容失败" ? error.message : "请稍后重试。",
      });
    },
    onSuccess: async () => {
      showToast({
        tone: "info",
        title: contentDeletionCopy.successTitle,
        description: contentDeletionCopy.successDescription,
      });
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: feedKeys.recommendPrefix() }),
        queryClient.invalidateQueries({ queryKey: feedKeys.followPrefix() }),
        queryClient.invalidateQueries({ queryKey: feedKeys.favoritesPrefix() }),
        queryClient.invalidateQueries({ queryKey: feedKeys.userPublishPrefix(currentUserId) }),
        queryClient.invalidateQueries({ queryKey: feedKeys.studioPublishPrefix(currentUserId) }),
        queryClient.invalidateQueries({ queryKey: userKeys.profilePrefix(currentUserId) }),
      ]);
    },
  });

  function handleDeleteContent(contentId: number) {
    if (deleteMutation.isPending) {
      return;
    }

    if (
      typeof window !== "undefined" &&
      !window.confirm(contentDeletionCopy.confirm)
    ) {
      return;
    }

    deleteMutation.mutate(contentId);
  }

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
        description="这里已经具备公开作品回访、基础统计、编辑和删除能力。"
      />

      <PersonalSpaceHero
        eyebrow="Public Studio"
        identity={`当前用户 ID ${currentUserId}`}
        title="我的公开作品空间"
        description="这里集中展示当前可被他人看见的公开内容，并提供基础创作者统计，便于回访、自检和发布后确认。"
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
          {
            label: "公开内容",
            value: aggregateContentCount,
            hint: "优先显示后端聚合的公开内容统计，列表只负责回放当前页数据。",
          },
          {
            label: "粉丝",
            value: meQuery.data?.follower_count ?? "加载中",
            hint: "用于快速判断公开关系面的变化。",
          },
          {
            label: "获赞",
            value: meQuery.data?.like_received_count ?? "加载中",
            hint: "当前使用个人聚合计数，适合做第一版创作者面板。",
          },
          {
            label: "被收藏",
            value: meQuery.data?.favorite_received_count ?? "加载中",
            hint: "用于观察内容被保存和回访的强度。",
          },
          {
            label: "管理能力",
            value: deleteMutation.isPending ? "处理中" : "可编辑 / 可删除",
            hint: "你现在可以继续编辑已发布内容，也可以从公开列表移除它。",
          },
        ]}
        columns={5}
      />

      <div className="grid gap-6 lg:grid-cols-[1.25fr_0.75fr]">
        <PersonalSpaceSection
          eyebrow="Publish Feed"
          title="公开发布"
          description="按公开发布时间回看已经对外展示的作品，并把聚合总数和当前页列表区分开。"
          badge={
            <span className="rounded-full bg-[#eef7fb] px-4 py-2 text-sm font-medium text-slate-700">
              公开总数 {aggregateContentCount} · 已加载 {items.length} 条
            </span>
          }
        >
          <InlineNotice
            title="当前总量和当前页列表分开表达"
            description="badge 里的公开总数优先来自我的主页聚合；已加载只代表前端当前拿到的公开 feed 条数，不代表全部后台内容。"
            tone="soft"
          />

          <PagedQueryFeedback
            hasItems={items.length > 0}
            isRefreshing={query.isRefetching && !query.isLoading && !query.isFetchingNextPage}
            isFetchingNextPage={query.isFetchingNextPage}
            refreshingTitle="公开发布空间正在同步最新状态"
            refreshingDescription="编辑、删除或重新进入页面后，当前公开列表会在后台刷新。"
            fetchingNextPageTitle="正在继续加载公开内容"
            fetchingNextPageDescription="下一页公开作品正在追加到当前 Studio 列表。"
          />

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
                  <StudioContentCard
                    item={item}
                    deleting={deleteMutation.isPending && deleteMutation.variables === item.content_id}
                    onDelete={() => handleDeleteContent(item.content_id)}
                  />
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
              label="数据口径"
              value="聚合总数优先"
              description="列表 badge 里的“已加载”只代表当前拉到前端的页数，总数口径仍以后端聚合统计为准。"
            />
            <PersonalSpaceInfoCard
              label="内容范围"
              value="只展示公开内容"
              description="当前数据来自 `user publish feed`，不会把私密内容伪装成已可管理。"
            />
            <PersonalSpaceInfoCard
              label="当前用途"
              value="浏览、回访、编辑、自检"
              description="你可以确认作品是否已进入公开可见链路，并继续编辑或清理它。"
            />
            <PersonalSpaceInfoCard
              label="编辑与删除"
              value="都已接通"
              description="编辑会刷新详情页与发布列表，删除则会把内容从公开读取链路中移除。"
            />
            <PersonalSpaceInfoCard
              label="统计面板"
              value="已接基础聚合"
              description="当前复用我的主页计数来展示内容、粉丝、获赞和被收藏，先完成第一版 studio 面板。"
            />
            <PersonalSpaceInfoCard
              label="未接能力"
              value="私密内容 / Studio Summary 还没接前端"
              description="这页先保持公开内容回访、编辑和删除语义清楚，不冒充更完整的后台能力。"
            />
          </div>
        </PersonalSpaceSection>
      </div>
    </section>
  );
}

function StudioContentCard({
  item,
  deleting,
  onDelete,
}: {
  item: FeedItem;
  deleting: boolean;
  onDelete: () => void;
}) {
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
            to={
              item.content_type === 20
                ? `/studio/video/${item.content_id}/edit`
                : `/studio/article/${item.content_id}/edit`
            }
            className="rounded-full border border-slate-200 bg-white px-4 py-2 text-sm text-slate-600 transition hover:border-accent hover:text-accent"
          >
            编辑内容
          </Link>
          <Link
            to={`/users/${item.author_id}`}
            className="rounded-full border border-slate-200 bg-white px-4 py-2 text-sm text-slate-600 transition hover:border-accent hover:text-accent"
          >
            查看主页
          </Link>
          <button
            type="button"
            onClick={onDelete}
            disabled={deleting}
            className="rounded-full border border-[#ffd7cf] bg-[#fff6f3] px-4 py-2 text-sm text-ember transition hover:border-ember disabled:cursor-not-allowed disabled:opacity-60"
          >
            {deleting ? contentDeletionCopy.pendingActionLabel : contentDeletionCopy.actionLabel}
          </button>
        </div>
      </div>
    </article>
  );
}
