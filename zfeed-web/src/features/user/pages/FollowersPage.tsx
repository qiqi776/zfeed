import { useInfiniteQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { Link, useParams } from "react-router-dom";

import { useSessionStore } from "@/entities/session/model/session.store";
import { followUser, unfollowUser } from "@/features/interaction/api/interaction.api";
import { queryFollowers } from "@/features/user/api/user.api";
import {
  captureQuerySnapshots,
  getFollowSyncQueryKeys,
  patchAuthorFollowStateAcrossPages,
  restoreQuerySnapshots,
} from "@/shared/lib/query/cacheSync";
import { feedKeys, userKeys } from "@/shared/lib/query/queryKeys";
import { ImageFallback } from "@/shared/ui/ImageFallback";
import { PageHeader } from "@/shared/ui/PageHeader";
import { PagedQueryFeedback } from "@/shared/ui/PagedQueryFeedback";
import { StatePanel } from "@/shared/ui/StatePanel";
import { UserCardSkeletonGrid } from "@/shared/ui/UserCardSkeletonGrid";
import { useToast } from "@/shared/ui/toast/toast.store";

const defaultPageSize = 20;

export function FollowersPage() {
  const params = useParams();
  const queryClient = useQueryClient();
  const { showToast } = useToast();
  const currentUserId = useSessionStore((state) => state.user?.userId ?? 0);

  const userId = Number(params.userId);
  const isValidUserId = Number.isInteger(userId) && userId > 0;
  const isOwnPage = currentUserId > 0 && currentUserId === userId;

  const query = useInfiniteQuery({
    queryKey: userKeys.followers(userId, currentUserId, defaultPageSize),
    enabled: isValidUserId,
    initialPageParam: { cursor: undefined as number | undefined },
    queryFn: ({ pageParam }) =>
      queryFollowers({
        user_id: userId,
        cursor: pageParam.cursor,
        page_size: defaultPageSize,
      }),
    getNextPageParam: (lastPage) =>
      lastPage.has_more ? { cursor: lastPage.next_cursor } : undefined,
  });

  const items = query.data?.pages.flatMap((page) => page.items) ?? [];

  const followMutation = useMutation({
    mutationFn: async (payload: { targetUserId: number; isFollowing: boolean }) => {
      if (payload.isFollowing) {
        const res = await unfollowUser({ target_user_id: payload.targetUserId });
        return res.is_followed;
      }
      const res = await followUser({ target_user_id: payload.targetUserId });
      return res.is_followed;
    },
    onMutate: async (payload) => {
      const queryKeys = getFollowSyncQueryKeys(payload.targetUserId, currentUserId);
      await Promise.all(queryKeys.map((queryKey) => queryClient.cancelQueries({ queryKey })));

      const previousSnapshots = captureQuerySnapshots(queryClient, queryKeys);
      patchAuthorFollowStateAcrossPages(
        queryClient,
        payload.targetUserId,
        currentUserId,
        !payload.isFollowing,
      );

      return { previousSnapshots };
    },
    onError: (error, _payload, context) => {
      if (context?.previousSnapshots) {
        restoreQuerySnapshots(queryClient, context.previousSnapshots);
      }
      showToast({
        tone: "error",
        title: "关注状态更新失败",
        description: error.message || "请稍后重试。",
      });
    },
    onSuccess: async (isFollowed, payload) => {
      if (isFollowed !== !payload.isFollowing) {
        patchAuthorFollowStateAcrossPages(
          queryClient,
          payload.targetUserId,
          currentUserId,
          isFollowed,
        );
      }

      await Promise.all([
        queryClient.invalidateQueries({ queryKey: userKeys.followersPrefix(userId) }),
        queryClient.invalidateQueries({ queryKey: userKeys.profilePrefix(payload.targetUserId) }),
        queryClient.invalidateQueries({ queryKey: userKeys.profilePrefix(currentUserId) }),
        queryClient.invalidateQueries({ queryKey: userKeys.mePrefix() }),
        queryClient.invalidateQueries({ queryKey: feedKeys.followPrefix(currentUserId) }),
      ]);
    },
  });

  if (!isValidUserId) {
    return <StatePanel title="用户 ID 无效" description="请检查当前链接。" tone="error" />;
  }

  return (
    <section className="space-y-6">
      <PageHeader
        eyebrow="Followers"
        title={isOwnPage ? "我的粉丝" : `用户 ${userId} 的粉丝`}
        description="按最近关注关系倒序查看粉丝，并可直接在列表里处理互相关注。"
        aside={
          <Link
            to={isOwnPage ? "/me" : `/users/${userId}`}
            className="inline-flex rounded-full border border-slate-200 bg-white px-4 py-2 text-sm text-slate-600 transition hover:border-accent hover:text-accent"
          >
            {isOwnPage ? "返回我的主页" : "返回用户主页"}
          </Link>
        }
      />

      <PagedQueryFeedback
        hasItems={items.length > 0}
        isRefreshing={query.isRefetching && !query.isLoading && !query.isFetchingNextPage}
        isFetchingNextPage={query.isFetchingNextPage}
        refreshingTitle="粉丝列表正在同步最新关系"
        refreshingDescription="互相关注和回关状态会在后台同步到当前列表。"
        fetchingNextPageTitle="正在继续加载粉丝"
        fetchingNextPageDescription="下一页粉丝资料正在追加到当前列表。"
      />

      {query.isLoading ? <UserCardSkeletonGrid /> : null}

      {query.isError ? (
        <StatePanel
          title="粉丝列表加载失败"
          description={(query.error as Error)?.message || "请稍后重试"}
          tone="error"
        />
      ) : null}

      {!query.isLoading && !query.isError && items.length === 0 ? (
        <StatePanel
          title={isOwnPage ? "你还没有粉丝" : "这个用户还没有粉丝"}
          description="当别人开始持续关注你时，这里会出现粉丝列表。"
        />
      ) : null}

      {items.length > 0 ? (
        <ul className="grid gap-4 md:grid-cols-2">
          {items.map((item) => (
            <li key={item.user_id}>
              <article className="rounded-[28px] border border-slate-200 bg-white p-5 shadow-card">
                <div className="flex items-start gap-4">
                  <Link to={`/users/${item.user_id}`} className="shrink-0">
                    <ImageFallback
                      src={item.avatar}
                      alt={item.nickname || `用户 ${item.user_id}`}
                      name={item.nickname || `用户 ${item.user_id}`}
                      variant="avatar"
                      containerClassName="h-16 w-16 overflow-hidden rounded-full bg-slate-100"
                      imageClassName="h-full w-full object-cover"
                    />
                  </Link>

                  <div className="min-w-0 flex-1">
                    <Link to={`/users/${item.user_id}`} className="block">
                      <p className="truncate text-lg font-semibold text-slate-900">
                        {item.nickname || `用户 ${item.user_id}`}
                      </p>
                    </Link>
                    <p className="mt-2 line-clamp-3 text-sm leading-6 text-slate-500">
                      {item.bio || "这个人还没有留下简介。"}
                    </p>

                    <div className="mt-4 flex flex-wrap gap-3">
                      <Link
                        to={`/users/${item.user_id}`}
                        className="rounded-full border border-slate-200 bg-white px-4 py-2 text-sm text-slate-600 transition hover:border-accent hover:text-accent"
                      >
                        查看主页
                      </Link>

                      {currentUserId > 0 && currentUserId !== item.user_id ? (
                        <button
                          type="button"
                          disabled={followMutation.isPending}
                          onClick={() =>
                            followMutation.mutate({
                              targetUserId: item.user_id,
                              isFollowing: item.is_following,
                            })
                          }
                          className={[
                            "rounded-full px-4 py-2 text-sm font-medium transition",
                            item.is_following
                              ? "border border-slate-200 bg-white text-slate-700 hover:border-accent hover:text-accent"
                              : "bg-ink text-white hover:bg-slate-800",
                            followMutation.isPending &&
                            followMutation.variables?.targetUserId === item.user_id
                              ? "opacity-60"
                              : "",
                          ].join(" ")}
                        >
                          {followMutation.isPending &&
                          followMutation.variables?.targetUserId === item.user_id
                            ? "处理中..."
                            : item.is_following
                              ? "已关注"
                              : "回关"}
                        </button>
                      ) : null}
                    </div>
                  </div>
                </div>
              </article>
            </li>
          ))}
        </ul>
      ) : null}

      {query.hasNextPage ? (
        <button
          type="button"
          onClick={() => query.fetchNextPage()}
          disabled={query.isFetchingNextPage}
          className="rounded-full border border-slate-200 bg-white px-5 py-2.5 text-sm text-slate-600 transition hover:border-accent hover:text-accent disabled:opacity-60"
        >
          {query.isFetchingNextPage ? "加载中..." : "加载更多"}
        </button>
      ) : null}
    </section>
  );
}
