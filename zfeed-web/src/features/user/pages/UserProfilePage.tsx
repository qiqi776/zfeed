import { useInfiniteQuery, useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Link, useParams } from "react-router-dom";

import { useSessionStore } from "@/entities/session/model/session.store";
import { getUserPublish } from "@/features/feed/api/feed.api";
import { FeedCard } from "@/features/feed/ui/FeedCard";
import { followUser, unfollowUser } from "@/features/interaction/api/interaction.api";
import { getUserProfile, type UserProfileRes } from "@/features/user/api/user.api";
import {
  captureQuerySnapshots,
  getFollowSyncQueryKeys,
  patchAuthorFollowStateAcrossPages,
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

function genderText(gender: number) {
  if (gender === 1) {
    return "男";
  }
  if (gender === 2) {
    return "女";
  }
  return "未设置";
}

export function UserProfilePage() {
  const params = useParams();
  const queryClient = useQueryClient();
  const currentUserId = useSessionStore((state) => state.user?.userId ?? 0);
  const { showToast } = useToast();

  const userId = Number(params.userId);
  const isValidUserId = Number.isInteger(userId) && userId > 0;

  const profileQuery = useQuery({
    queryKey: userKeys.profile(userId, currentUserId),
    enabled: isValidUserId,
    queryFn: () => getUserProfile(userId),
  });

  const publishQuery = useInfiniteQuery({
    queryKey: feedKeys.userPublish(userId, currentUserId, DEFAULT_FEED_PAGE_SIZE),
    enabled: isValidUserId,
    initialPageParam: { cursor: "0" },
    queryFn: ({ pageParam }) =>
      getUserPublish({
        user_id: userId,
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

  const publishItems = publishQuery.data?.pages.flatMap((page) => page.items) ?? [];
  const profile = profileQuery.data;
  const profileInfo = profile?.user_profile;
  const profileCounts = profile?.counts;
  const isOwnProfile = currentUserId > 0 && currentUserId === userId;

  const followMutation = useMutation({
    mutationFn: async (current: UserProfileRes) => {
      if (current.viewer.is_following) {
        const res = await unfollowUser({ target_user_id: current.user_profile.user_id });
        return res.is_followed;
      }
      const res = await followUser({ target_user_id: current.user_profile.user_id });
      return res.is_followed;
    },
    onMutate: async (current) => {
      const queryKeys = getFollowSyncQueryKeys(current.user_profile.user_id, currentUserId);
      await Promise.all(queryKeys.map((queryKey) => queryClient.cancelQueries({ queryKey })));

      const previousSnapshots = captureQuerySnapshots(queryClient, queryKeys);
      const nextIsFollowing = !current.viewer.is_following;

      patchAuthorFollowStateAcrossPages(
        queryClient,
        current.user_profile.user_id,
        currentUserId,
        nextIsFollowing,
      );

      return { previousSnapshots };
    },
    onError: (error, _current, context) => {
      if (context?.previousSnapshots) {
        restoreQuerySnapshots(queryClient, context.previousSnapshots);
      }
      showToast({
        tone: "error",
        title: "关注状态更新失败",
        description:
          error.message && error.message !== "关注操作失败" ? error.message : "请稍后重试。",
      });
    },
    onSuccess: async (isFollowed, current) => {
      if (isFollowed !== !current.viewer.is_following) {
        patchAuthorFollowStateAcrossPages(
          queryClient,
          current.user_profile.user_id,
          currentUserId,
          isFollowed,
        );
      }

      await Promise.all([
        queryClient.invalidateQueries({ queryKey: feedKeys.followPrefix(currentUserId) }),
        queryClient.invalidateQueries({
          queryKey: userKeys.profilePrefix(current.user_profile.user_id),
        }),
        queryClient.invalidateQueries({ queryKey: userKeys.profilePrefix(currentUserId) }),
        queryClient.invalidateQueries({ queryKey: userKeys.mePrefix() }),
      ]);
    },
  });

  if (!isValidUserId) {
    return <StatePanel title="用户 ID 无效" description="请检查链接里的用户编号。" tone="error" />;
  }

  if (profileQuery.isLoading) {
    return (
      <section className="space-y-4">
        <div className="h-12 w-44 rounded-full bg-white shadow-card" />
        <div className="h-48 rounded-[32px] bg-white shadow-card" />
        <FeedGridSkeleton />
      </section>
    );
  }

  if (profileQuery.isError || !profile || !profileInfo || !profileCounts) {
    return (
      <StatePanel
        title="用户主页加载失败"
        description={(profileQuery.error as Error)?.message || "请稍后重试"}
        tone="error"
        action={
          <Link
            to="/"
            className="inline-flex rounded-full bg-ink px-5 py-2.5 text-sm font-medium text-white transition hover:bg-slate-800"
          >
            返回推荐流
          </Link>
        }
      />
    );
  }

  return (
    <section className="space-y-6">
      <PageHeader
        eyebrow="Profile"
        title={isOwnProfile ? "公开主页预览" : "用户主页"}
        description={
          isOwnProfile
            ? "这里展示其他用户看到的公开资料和公开发布内容。"
            : "查看公开资料、关系状态和公开发布内容。"
        }
        aside={
          <Link
            to={isOwnProfile ? "/me" : "/"}
            className="inline-flex items-center rounded-full border border-slate-200 bg-white px-4 py-2 text-sm text-slate-600 transition hover:border-accent hover:text-accent"
          >
            {isOwnProfile ? "返回我的主页" : "返回推荐流"}
          </Link>
        }
      />

      <PersonalSpaceHero
        eyebrow={isOwnProfile ? "Public Preview" : "Profile"}
        identity={`ID ${profileInfo.user_id}`}
        title={profileInfo.nickname || `用户 ${profileInfo.user_id}`}
        description={profileInfo.bio || "这个人还没有留下公开简介。"}
        media={
          <ImageFallback
            src={profileInfo.avatar}
            alt={profileInfo.nickname || `用户 ${profileInfo.user_id}`}
            name={profileInfo.nickname || `用户 ${profileInfo.user_id}`}
            variant="avatar"
            containerClassName="h-20 w-20 overflow-hidden rounded-full border border-white/70 bg-white/80"
            imageClassName="h-full w-full object-cover"
          />
        }
        aside={
          <div className="flex flex-wrap gap-3 lg:max-w-sm lg:justify-end">
            {isOwnProfile ? (
              <>
                <span className="rounded-full border border-slate-200 bg-white px-4 py-2 text-sm text-slate-600">
                  这是你的公开主页
                </span>
                <Link
                  to="/me/settings"
                  className="rounded-full border border-slate-200 bg-white px-4 py-2 text-sm text-slate-600 transition hover:border-accent hover:text-accent"
                >
                  编辑资料
                </Link>
              </>
            ) : (
              <button
                type="button"
                onClick={() => followMutation.mutate(profile)}
                disabled={followMutation.isPending}
                className={[
                  "rounded-full px-5 py-2.5 text-sm font-medium transition",
                  profile.viewer.is_following
                    ? "border border-slate-200 bg-white text-slate-700 hover:border-accent hover:text-accent"
                    : "bg-ink text-white hover:bg-slate-800",
                  followMutation.isPending ? "opacity-70" : "",
                ].join(" ")}
              >
                {followMutation.isPending
                  ? "处理中..."
                  : profile.viewer.is_following
                    ? "已关注"
                    : "关注"}
              </button>
            )}
          </div>
        }
      />

      <PersonalMetricGrid
        items={[
          { label: "关注", value: profileCounts.followee_count },
          { label: "粉丝", value: profileCounts.follower_count },
          { label: "内容", value: profileCounts.content_count },
          { label: "获赞", value: profileCounts.like_received_count },
          { label: "被收藏", value: profileCounts.favorite_received_count },
        ]}
        columns={5}
      />

      <InlineNotice
        title="当前只展示公开资料和公开内容"
        description="关注、粉丝、内容、获赞和被收藏来自对方公开聚合；下方发布列表只覆盖当前开放读取的公开内容，不代表对方全部后台数据。"
        tone="soft"
      />

      <div className="flex flex-wrap gap-3">
        <Link
          to={`/users/${profileInfo.user_id}/followers`}
          className="rounded-full border border-slate-200 bg-white px-4 py-2 text-sm text-slate-600 transition hover:border-accent hover:text-accent"
        >
          查看粉丝列表
        </Link>
      </div>

      <div className="grid gap-6 lg:grid-cols-[1.25fr_0.75fr]">
        <PersonalSpaceSection
          eyebrow="Publish Feed"
          title="公开发布"
          description="这里展示当前公开可见的内容，用于继续回访作者和作品；总量和当前页列表分开表达。"
          badge={
            <span className="rounded-full bg-[#fff8ef] px-4 py-2 text-sm text-slate-600">
              公开总数 {profileCounts.content_count} · 已加载 {publishItems.length} 条
            </span>
          }
        >
          <InlineNotice
            title="公开总量和当前列表是两套口径"
            description="badge 里的公开总数来自资料聚合；已加载只代表前端当前拉到的公开 feed 条数。"
            tone="soft"
          />

          <PagedQueryFeedback
            hasItems={publishItems.length > 0}
            isRefreshing={
              publishQuery.isRefetching && !publishQuery.isLoading && !publishQuery.isFetchingNextPage
            }
            isFetchingNextPage={publishQuery.isFetchingNextPage}
            refreshingTitle="公开发布列表正在同步"
            refreshingDescription="资料页和其他公开读取链路的变化会在后台刷新到这里。"
            fetchingNextPageTitle="正在继续加载公开发布"
            fetchingNextPageDescription="下一页公开作品正在追加到当前主页列表。"
          />

          {publishQuery.isLoading ? <FeedGridSkeleton /> : null}
          {publishQuery.isError ? (
            <StatePanel
              title="公开发布加载失败"
              description={(publishQuery.error as Error)?.message || "请稍后重试"}
              tone="error"
            />
          ) : null}

          {publishItems.length > 0 ? (
            <ul className="grid gap-4 md:grid-cols-2">
              {publishItems.map((item) => (
                <li key={`${item.content_id}-${item.published_at}`}>
                  <FeedCard item={item} />
                </li>
              ))}
            </ul>
          ) : !publishQuery.isLoading && !publishQuery.isError ? (
            <StatePanel
              title="还没有公开发布内容"
              description="这个用户当前没有可展示的公开内容。"
            />
          ) : null}

          {publishQuery.hasNextPage ? (
            <button
              type="button"
              onClick={() => publishQuery.fetchNextPage()}
              disabled={publishQuery.isFetchingNextPage}
              className="rounded-xl border border-slate-300 bg-white px-4 py-2 text-sm transition hover:border-accent hover:text-accent disabled:opacity-60"
            >
              {publishQuery.isFetchingNextPage ? "加载中..." : "加载更多"}
            </button>
          ) : null}
        </PersonalSpaceSection>

        <PersonalSpaceSection
          eyebrow="Profile Meta"
          title="主页信息"
          description="这部分强调作者身份、数据口径和当前公开主页的边界。"
        >
          <div className="space-y-3">
            <PersonalSpaceInfoCard
              label="数据口径"
              value={`公开聚合 ${profileCounts.content_count} 条`}
              description="这里不会暴露草稿、私密内容或未开放的后台状态；当前列表 badge 里的已加载只代表前端页数。"
            />
            <PersonalSpaceInfoCard
              label="关系状态"
              value={
                isOwnProfile
                  ? "这是你的公开主页"
                  : profile.viewer.is_following
                    ? "你已经关注这个用户"
                    : "你还没有关注这个用户"
              }
              description={
                isOwnProfile
                  ? "这里用于预览其他用户当前看到的版本。"
                  : "关注状态会在推荐流、详情页和用户主页之间保持同步。"
              }
            />
            <PersonalSpaceInfoCard
              label="性别"
              value={genderText(profileInfo.gender)}
              description="当前为公开资料中的展示信息。"
            />
            <PersonalSpaceInfoCard
              label="主页状态"
              value={profileCounts.content_count > 0 ? "有公开内容" : "暂时空白"}
              description="分页继续拉取的也是公开发布 feed，不会推断未公开内容。"
            />
          </div>
        </PersonalSpaceSection>
      </div>
    </section>
  );
}
