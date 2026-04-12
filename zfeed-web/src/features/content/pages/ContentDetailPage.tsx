import {
  type InfiniteData,
  useInfiniteQuery,
  useMutation,
  useQuery,
  useQueryClient,
} from "@tanstack/react-query";
import { type FormEvent, useMemo, useRef, useState } from "react";
import { Link, useParams } from "react-router-dom";

import { useSessionStore } from "@/entities/session/model/session.store";
import {
  getContentDetail,
  type ContentDetail,
  type GetContentDetailRes,
} from "@/features/content/api/content.api";
import {
  commentContent,
  deleteComment,
  favoriteContent,
  followUser,
  likeContent,
  queryCommentList,
  queryReplyCommentList,
  removeFavorite,
  type CommentItem as InteractionCommentItem,
  type InteractionScene,
  type QueryCommentListRes,
  type QueryReplyCommentListRes,
  unfollowUser,
  unlikeContent,
} from "@/features/interaction/api/interaction.api";
import {
  captureQuerySnapshots,
  patchAuthorFollowStateAcrossPages,
  patchContentDetail,
  patchLikeStateAcrossCollections,
  restoreQuerySnapshots,
} from "@/shared/lib/query/cacheSync";
import { contentKeys, feedKeys, userKeys } from "@/shared/lib/query/queryKeys";
import { ImageFallback } from "@/shared/ui/ImageFallback";
import { InlineNotice } from "@/shared/ui/InlineNotice";
import { StatePanel } from "@/shared/ui/StatePanel";
import { useToast } from "@/shared/ui/toast/toast.store";

const topLevelCommentPageSize = 10;
const replyCommentPageSize = 8;

type ReplyTarget = {
  parentId: number;
  rootId: number;
  replyToUserId: number;
  userName: string;
  threadOwnerName: string;
  isNested: boolean;
};

function resolveScene(contentType: number): InteractionScene {
  return contentType === 20 ? "VIDEO" : "ARTICLE";
}

function formatDateTime(timestamp: number) {
  if (!timestamp) {
    return "刚刚";
  }
  return new Intl.DateTimeFormat("zh-CN", {
    year: "numeric",
    month: "2-digit",
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
  }).format(timestamp * 1000);
}

function formatDuration(seconds: number) {
  if (!seconds) {
    return "00:00";
  }
  const hours = Math.floor(seconds / 3600);
  const minutes = Math.floor((seconds % 3600) / 60);
  const remain = seconds % 60;

  if (hours > 0) {
    return [hours, minutes, remain].map((value) => String(value).padStart(2, "0")).join(":");
  }

  return [minutes, remain].map((value) => String(value).padStart(2, "0")).join(":");
}

function flattenComments(
  data: InfiniteData<QueryCommentListRes> | InfiniteData<QueryReplyCommentListRes> | undefined,
) {
  return data?.pages.flatMap((page) => page.comments ?? []) ?? [];
}

function createReplyTarget(
  rootComment: InteractionCommentItem,
  targetComment: InteractionCommentItem,
): ReplyTarget {
  const threadOwnerName = rootComment.user_name || `用户 ${rootComment.user_id}`;

  return {
    parentId: targetComment.comment_id,
    rootId: rootComment.root_id > 0 ? rootComment.root_id : rootComment.comment_id,
    replyToUserId: targetComment.user_id,
    userName: targetComment.user_name || `用户 ${targetComment.user_id}`,
    threadOwnerName,
    isNested: targetComment.comment_id !== rootComment.comment_id,
  };
}

function optionalCommentId(value: number) {
  return value > 0 ? value : undefined;
}

export function ContentDetailPage() {
  const params = useParams();
  const queryClient = useQueryClient();
  const currentUserId = useSessionStore((state) => state.user?.userId ?? 0);
  const commentInputRef = useRef<HTMLTextAreaElement | null>(null);
  const { showToast } = useToast();

  const [commentDraft, setCommentDraft] = useState("");
  const [replyTarget, setReplyTarget] = useState<ReplyTarget | null>(null);

  const contentId = Number(params.contentId);
  const isValidContentId = Number.isInteger(contentId) && contentId > 0;

  const detailQuery = useQuery({
    queryKey: contentKeys.detail(contentId, currentUserId),
    enabled: isValidContentId,
    queryFn: () => getContentDetail({ content_id: contentId }),
  });

  const detail = detailQuery.data?.detail;
  const scene = detail ? resolveScene(detail.content_type) : null;
  const trimmedComment = commentDraft.trim();

  const commentsQuery = useInfiniteQuery({
    queryKey: contentKeys.comments(contentId),
    enabled: isValidContentId && Boolean(scene),
    initialPageParam: 0,
    queryFn: ({ pageParam }) =>
      queryCommentList({
        content_id: contentId,
        scene: scene ?? "ARTICLE",
        cursor: Number(pageParam) || 0,
        page_size: topLevelCommentPageSize,
      }),
    getNextPageParam: (lastPage) => (lastPage.has_more ? lastPage.next_cursor : undefined),
  });

  const comments = useMemo(() => flattenComments(commentsQuery.data), [commentsQuery.data]);

  const likeMutation = useMutation({
    mutationFn: async (current: ContentDetail) => {
      const currentScene = resolveScene(current.content_type);
      if (current.is_liked) {
        await unlikeContent({ content_id: current.content_id, scene: currentScene });
        return false;
      }
      await likeContent({
        content_id: current.content_id,
        content_user_id: current.author_id,
        scene: currentScene,
      });
      return true;
    },
    onMutate: async (current) => {
      await Promise.all([
        queryClient.cancelQueries({ queryKey: contentKeys.detail(current.content_id, currentUserId) }),
        queryClient.cancelQueries({ queryKey: feedKeys.recommendPrefix(currentUserId) }),
        queryClient.cancelQueries({ queryKey: feedKeys.followPrefix(currentUserId) }),
        queryClient.cancelQueries({ queryKey: feedKeys.favoritesPrefix(currentUserId) }),
        queryClient.cancelQueries({ queryKey: feedKeys.userPublishPrefix() }),
        queryClient.cancelQueries({ queryKey: feedKeys.studioPublishPrefix() }),
      ]);

      const previousSnapshots = captureQuerySnapshots(queryClient, [
        contentKeys.detail(current.content_id, currentUserId),
        feedKeys.recommendPrefix(currentUserId),
        feedKeys.followPrefix(currentUserId),
        feedKeys.favoritesPrefix(currentUserId),
        feedKeys.userPublishPrefix(),
        feedKeys.studioPublishPrefix(),
      ]);

      const nextIsLiked = !current.is_liked;
      const delta = nextIsLiked ? 1 : -1;

      patchContentDetail(queryClient, current.content_id, currentUserId, (detailState) => ({
        ...detailState,
        is_liked: nextIsLiked,
        like_count: Math.max(0, detailState.like_count + delta),
      }));
      patchLikeStateAcrossCollections(queryClient, current.content_id, nextIsLiked, delta);

      return { previousSnapshots };
    },
    onError: (error, _current, context) => {
      if (context?.previousSnapshots) {
        restoreQuerySnapshots(queryClient, context.previousSnapshots);
      }
      showToast({
        tone: "error",
        title: "点赞状态更新失败",
        description: error.message && error.message !== "点赞操作失败" ? error.message : "请稍后重试。",
      });
    },
  });

  const favoriteMutation = useMutation({
    mutationFn: async (current: ContentDetail) => {
      const currentScene = resolveScene(current.content_type);
      if (current.is_favorited) {
        await removeFavorite({ content_id: current.content_id, scene: currentScene });
        return false;
      }
      await favoriteContent({
        content_id: current.content_id,
        content_user_id: current.author_id,
        scene: currentScene,
      });
      return true;
    },
    onMutate: async (current) => {
      await queryClient.cancelQueries({ queryKey: contentKeys.detail(current.content_id, currentUserId) });

      const previousDetail = queryClient.getQueryData<GetContentDetailRes>(
        contentKeys.detail(current.content_id, currentUserId),
      );
      const nextIsFavorited = !current.is_favorited;
      const delta = nextIsFavorited ? 1 : -1;

      patchContentDetail(queryClient, current.content_id, currentUserId, (detailState) => ({
        ...detailState,
        is_favorited: nextIsFavorited,
        favorite_count: Math.max(0, detailState.favorite_count + delta),
      }));

      return { previousDetail };
    },
    onError: (error, current, context) => {
      if (context?.previousDetail) {
        queryClient.setQueryData(contentKeys.detail(current.content_id, currentUserId), context.previousDetail);
      }
      showToast({
        tone: "error",
        title: "收藏状态更新失败",
        description: error.message && error.message !== "收藏操作失败" ? error.message : "请稍后重试。",
      });
    },
    onSuccess: () => {
      if (currentUserId > 0) {
        void queryClient.invalidateQueries({ queryKey: feedKeys.favoritesPrefix(currentUserId) });
      }
    },
  });

  const followMutation = useMutation({
    mutationFn: async (current: ContentDetail) => {
      if (current.is_following_author) {
        const res = await unfollowUser({ target_user_id: current.author_id });
        return res.is_followed;
      }
      const res = await followUser({ target_user_id: current.author_id });
      return res.is_followed;
    },
    onMutate: async (current) => {
      await Promise.all([
        queryClient.cancelQueries({ queryKey: contentKeys.detailPrefix() }),
        queryClient.cancelQueries({ queryKey: userKeys.profile(current.author_id, currentUserId) }),
      ]);

      const previousSnapshots = captureQuerySnapshots(queryClient, [
        contentKeys.detailPrefix(),
        userKeys.profile(current.author_id, currentUserId),
        userKeys.mePrefix(),
      ]);
      const nextIsFollowing = !current.is_following_author;

      patchAuthorFollowStateAcrossPages(queryClient, current.author_id, currentUserId, nextIsFollowing);

      return { previousSnapshots };
    },
    onError: (error, _current, context) => {
      if (context?.previousSnapshots) {
        restoreQuerySnapshots(queryClient, context.previousSnapshots);
      }
      showToast({
        tone: "error",
        title: "关注状态更新失败",
        description: error.message && error.message !== "关注操作失败" ? error.message : "请稍后重试。",
      });
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: feedKeys.followPrefix(currentUserId) });
    },
  });

  const commentMutation = useMutation({
    mutationFn: async (current: {
      detail: ContentDetail;
      comment: string;
      replyTarget: ReplyTarget | null;
    }) =>
      commentContent({
        content_id: current.detail.content_id,
        content_user_id: current.detail.author_id,
        scene: resolveScene(current.detail.content_type),
        comment: current.comment,
        parent_id: current.replyTarget?.parentId,
        root_id: current.replyTarget?.rootId,
        reply_to_user_id: current.replyTarget?.replyToUserId,
      }),
    onSuccess: (_, variables) => {
      patchContentDetail(queryClient, variables.detail.content_id, currentUserId, (detailState) => ({
        ...detailState,
        comment_count: detailState.comment_count + 1,
      }));
      setCommentDraft("");
      setReplyTarget(null);
      showToast({
        tone: "success",
        title: variables.replyTarget ? "回复已发送" : "评论已发布",
        description: "评论区内容已刷新。",
      });
      void queryClient.invalidateQueries({
        queryKey: contentKeys.comments(variables.detail.content_id),
      });
    },
    onError: (error) => {
      showToast({
        tone: "error",
        title: "评论发送失败",
        description: error.message && error.message !== "发表评论失败" ? error.message : "请稍后重试。",
      });
    },
  });

  const deleteCommentMutation = useMutation({
    mutationFn: async (current: { detail: ContentDetail; comment: InteractionCommentItem }) =>
      deleteComment({
        comment_id: current.comment.comment_id,
        content_id: current.detail.content_id,
        scene: resolveScene(current.detail.content_type),
        root_id: optionalCommentId(current.comment.root_id),
        parent_id: optionalCommentId(current.comment.parent_id),
      }),
    onSuccess: (_, variables) => {
      patchContentDetail(queryClient, variables.detail.content_id, currentUserId, (detailState) => ({
        ...detailState,
        comment_count: Math.max(0, detailState.comment_count - 1),
      }));
      showToast({
        tone: "info",
        title: "评论已删除",
        description: "评论区内容已刷新。",
      });
      void queryClient.invalidateQueries({
        queryKey: contentKeys.comments(variables.detail.content_id),
      });
    },
    onError: (error) => {
      showToast({
        tone: "error",
        title: "删除评论失败",
        description: error.message && error.message !== "删除评论失败" ? error.message : "请稍后重试。",
      });
    },
  });

  function handleReply(target: ReplyTarget) {
    setReplyTarget(target);
    commentInputRef.current?.focus();
    commentInputRef.current?.scrollIntoView?.({
      behavior: "smooth",
      block: "center",
    });
  }

  function handleCommentSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!detail || !trimmedComment) {
      return;
    }

    commentMutation.mutate({
      detail,
      comment: trimmedComment,
      replyTarget,
    });
  }

  if (!isValidContentId) {
    return <StatePanel title="内容 ID 无效" description="请检查链接里的内容编号。" tone="error" />;
  }

  if (detailQuery.isLoading) {
    return (
      <section className="space-y-4">
        <div className="h-10 w-40 rounded-full bg-slate-200" />
        <div className="h-[340px] rounded-[32px] bg-slate-200" />
        <div className="grid gap-4 lg:grid-cols-[1.8fr_0.8fr]">
          <div className="h-72 rounded-[28px] bg-white" />
          <div className="h-72 rounded-[28px] bg-white" />
        </div>
      </section>
    );
  }

  if (detailQuery.isError || !detail) {
    return (
      <StatePanel
        title="内容详情加载失败"
        description={(detailQuery.error as Error)?.message || "请稍后重试"}
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

  const isVideo = detail.content_type === 20;
  const isOwnContent = currentUserId === detail.author_id;
  const authorName = detail.author_name || `用户 ${detail.author_id}`;
  const composerHint = replyTarget
    ? replyTarget.isNested
      ? `这条回复会挂在 @${replyTarget.threadOwnerName} 的楼层下，并默认指向 @${replyTarget.userName}。`
      : `这条回复会进入 @${replyTarget.userName} 这条评论所在的楼层。`
    : "直接发布到当前评论区，所有浏览者都能看到。";
  const isCommentsRefreshing = commentsQuery.isRefetching && !commentsQuery.isLoading;

  return (
    <section className="space-y-6">
      <div className="flex items-center justify-between gap-3">
        <Link
          to="/"
          className="inline-flex items-center rounded-full border border-slate-200 bg-white px-4 py-2 text-sm text-slate-600 transition hover:border-accent hover:text-accent"
        >
          返回推荐流
        </Link>
        <span className="rounded-full bg-[#e9fbf7] px-4 py-2 text-xs font-semibold uppercase tracking-[0.22em] text-accent">
          {isVideo ? "Video Story" : "Article Story"}
        </span>
      </div>

      <article className="overflow-hidden rounded-[32px] border border-white/70 bg-white shadow-card">
        <div className="grid gap-0 lg:grid-cols-[1.15fr_0.85fr]">
          <div className="relative min-h-[320px] overflow-hidden bg-ink">
            <ImageFallback
              src={detail.cover_url}
              alt={detail.title || "内容封面"}
              containerClassName="h-full w-full"
              imageClassName="h-full w-full object-cover opacity-90"
              fallbackClassName="h-full"
            />
            <div className="absolute inset-0 bg-[linear-gradient(160deg,rgba(11,18,32,0.16),rgba(11,18,32,0.82))]" />
            <div className="absolute inset-x-0 bottom-0 space-y-4 p-6 text-white md:p-8">
              <p className="text-xs uppercase tracking-[0.3em] text-white/70">{authorName}</p>
              <h1 className="font-display text-3xl font-semibold leading-tight md:text-5xl">
                {detail.title || "未命名内容"}
              </h1>
              <p className="max-w-2xl text-sm text-white/80 md:text-base">
                {detail.description || "这条内容还没有额外描述，直接进入正文。"}
              </p>
            </div>
          </div>

          <div className="space-y-6 bg-[linear-gradient(180deg,#fffdf7,#fff)] p-6 md:p-8">
            <Link
              to={`/users/${detail.author_id}`}
              className="flex items-center gap-4 transition hover:text-accent"
            >
              <ImageFallback
                src={detail.author_avatar}
                alt={authorName}
                name={authorName}
                variant="avatar"
                containerClassName="h-14 w-14 overflow-hidden rounded-full bg-slate-100"
                imageClassName="h-full w-full object-cover"
              />
              <div>
                <p className="text-lg font-semibold text-slate-900">{authorName}</p>
                <p className="text-sm text-slate-500">
                  发布于 {formatDateTime(detail.published_at)}
                </p>
              </div>
            </Link>

            <div className="grid grid-cols-3 gap-3">
              <MetricCard label="点赞" value={detail.like_count} />
              <MetricCard label="收藏" value={detail.favorite_count} />
              <MetricCard label="评论" value={detail.comment_count} />
            </div>

            <div className="grid gap-3">
              <ActionButton
                active={detail.is_liked}
                busy={likeMutation.isPending}
                label={detail.is_liked ? "已点赞" : "点赞"}
                meta={`${detail.like_count} 次互动`}
                onClick={() => likeMutation.mutate(detail)}
              />
              <ActionButton
                active={detail.is_favorited}
                busy={favoriteMutation.isPending}
                label={detail.is_favorited ? "已收藏" : "收藏"}
                meta={`${detail.favorite_count} 人收藏`}
                onClick={() => favoriteMutation.mutate(detail)}
              />
              {isOwnContent ? (
                <div className="rounded-3xl border border-dashed border-slate-200 bg-slate-50 px-4 py-4 text-sm text-slate-500">
                  这是你发布的内容，当前不显示关注按钮。
                </div>
              ) : (
                <ActionButton
                  active={detail.is_following_author}
                  busy={followMutation.isPending}
                  label={detail.is_following_author ? "已关注作者" : "关注作者"}
                  meta={detail.is_following_author ? "之后会同步到关注流" : "建立内容到作者的关系"}
                  onClick={() => followMutation.mutate(detail)}
                />
              )}
            </div>

          </div>
        </div>
      </article>

      <div className="grid gap-6 lg:grid-cols-[1.5fr_0.5fr]">
        <section className="rounded-[28px] border border-slate-200 bg-white p-6 shadow-card">
          <div className="mb-4 flex items-center justify-between">
            <h2 className="font-display text-2xl font-semibold text-slate-900">
              {isVideo ? "播放内容" : "正文内容"}
            </h2>
            {isVideo ? (
              <span className="rounded-full bg-slate-100 px-3 py-1 text-xs text-slate-600">
                时长 {formatDuration(detail.video_duration)}
              </span>
            ) : null}
          </div>

          {isVideo ? (
            detail.video_url ? (
              <video
                controls
                playsInline
                poster={detail.cover_url}
                className="w-full rounded-3xl bg-black"
                src={detail.video_url}
              />
            ) : (
              <InlineNotice
                title="视频暂时无法播放"
                description="当前内容没有可用的视频地址，稍后可以再试，或先回到作者主页查看其他公开内容。"
                tone="soft"
              />
            )
          ) : (
            detail.article_content ? (
              <div className="whitespace-pre-wrap text-[15px] leading-8 text-slate-700">
                {detail.article_content}
              </div>
            ) : (
              <InlineNotice
                title="正文暂未提供"
                description="这条内容已经发布，但正文还没有可展示的公开文本。"
                tone="soft"
              />
            )
          )}
        </section>

        <aside className="space-y-4 rounded-[28px] border border-slate-200 bg-[linear-gradient(180deg,#f8fcff,#eef7fb)] p-6 shadow-card">
          <div>
            <p className="text-xs uppercase tracking-[0.22em] text-slate-500">阅读状态</p>
            <p className="mt-2 text-lg font-semibold text-slate-900">
              {detail.is_liked || detail.is_favorited || detail.is_following_author
                ? "你已经参与过互动"
                : "还没有留下动作"}
            </p>
          </div>
          <div className="rounded-3xl bg-white p-4">
            <p className="text-sm text-slate-500">作者 ID</p>
            <Link
              to={`/users/${detail.author_id}`}
              className="mt-1 inline-flex text-base font-semibold text-slate-900 transition hover:text-accent"
            >
              {detail.author_id}
            </Link>
          </div>
          <div className="rounded-3xl bg-white p-4">
            <p className="text-sm text-slate-500">内容 ID</p>
            <p className="mt-1 text-base font-semibold text-slate-900">{detail.content_id}</p>
          </div>
        </aside>
      </div>

      <section className="rounded-[28px] border border-slate-200 bg-white p-6 shadow-card">
        <div className="flex flex-col gap-3 sm:flex-row sm:items-end sm:justify-between">
          <div>
            <p className="text-xs uppercase tracking-[0.24em] text-slate-500">Discussion</p>
            <h2 className="mt-2 font-display text-2xl font-semibold text-slate-900">评论区</h2>
            <p className="mt-1 text-sm text-slate-500">
              先把根评论、回复查看和删除自己的评论打通。
            </p>
          </div>
          <div className="rounded-full bg-[#eef7fb] px-4 py-2 text-sm font-medium text-slate-700">
            {detail.comment_count} 条评论
          </div>
        </div>

        <form
          className="mt-6 rounded-[28px] border border-slate-200 bg-[#fbfdff] p-4"
          onSubmit={handleCommentSubmit}
        >
          {replyTarget ? (
            <div className="mb-3 flex flex-wrap items-center justify-between gap-3 rounded-2xl bg-[#fff8ef] px-4 py-3 text-sm text-slate-700">
              <p>
                正在回复{" "}
                <span className="font-semibold text-slate-900">@{replyTarget.userName}</span>
              </p>
              <button
                type="button"
                onClick={() => setReplyTarget(null)}
                className="text-sm text-slate-500 transition hover:text-accent"
              >
                取消回复
              </button>
            </div>
          ) : null}

          <p className="mb-3 text-sm leading-6 text-slate-500">{composerHint}</p>

          <textarea
            ref={commentInputRef}
            value={commentDraft}
            onChange={(event) => setCommentDraft(event.target.value)}
            maxLength={255}
            rows={4}
            placeholder={
              replyTarget ? `回复 ${replyTarget.userName}...` : "写下你的看法，最多 255 字"
            }
            className="w-full resize-none rounded-3xl border border-slate-200 bg-white px-4 py-3 text-sm leading-7 text-slate-800 outline-none ring-accent transition focus:ring"
          />

          <div className="mt-4 flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
            <p className="text-xs text-slate-500">{trimmedComment.length}/255</p>
            <div className="flex items-center gap-3">
              <button
                type="submit"
                disabled={!trimmedComment || commentMutation.isPending}
                className="rounded-full bg-ink px-5 py-2.5 text-sm font-medium text-white transition hover:bg-slate-800 disabled:cursor-not-allowed disabled:opacity-60"
              >
                {commentMutation.isPending ? "发送中..." : replyTarget ? "发送回复" : "发表评论"}
              </button>
            </div>
          </div>
        </form>

        {isCommentsRefreshing ? (
          <div className="mt-4">
            <InlineNotice
              title="评论区正在同步最新内容"
              description="刚刚的互动已经提交，列表会自动刷新。"
              tone="soft"
            />
          </div>
        ) : null}

        <div className="mt-6 space-y-4">
          {commentsQuery.isLoading ? (
            Array.from({ length: 3 }).map((_, index) => (
              <div key={index} className="h-28 rounded-[28px] bg-slate-100" />
            ))
          ) : commentsQuery.isError ? (
            <StatePanel
              title="评论列表加载失败"
              description={(commentsQuery.error as Error)?.message || "请稍后重试"}
              tone="error"
            />
          ) : comments.length > 0 ? (
            comments.map((comment) => (
              <CommentThread
                key={comment.comment_id}
                contentId={detail.content_id}
                rootComment={comment}
                authorId={detail.author_id}
                currentUserId={currentUserId}
                deletingCommentId={deleteCommentMutation.variables?.comment.comment_id}
                deletePending={deleteCommentMutation.isPending}
                onDelete={(targetComment) =>
                  deleteCommentMutation.mutate({ detail, comment: targetComment })
                }
                onReply={handleReply}
              />
            ))
          ) : (
            <StatePanel
              title="还没有评论"
              description="发一条评论，做第一个参与互动的人。"
            />
          )}
        </div>

        {commentsQuery.hasNextPage ? (
          <div className="mt-5 flex justify-center">
            <button
              type="button"
              onClick={() => commentsQuery.fetchNextPage()}
              disabled={commentsQuery.isFetchingNextPage}
              className="rounded-full border border-slate-200 bg-white px-5 py-2.5 text-sm text-slate-600 transition hover:border-accent hover:text-accent disabled:opacity-60"
            >
              {commentsQuery.isFetchingNextPage ? "加载中..." : "查看更多评论"}
            </button>
          </div>
        ) : comments.length > 0 ? (
          <p className="mt-5 text-center text-sm text-slate-500">评论已全部展示</p>
        ) : null}
      </section>
    </section>
  );
}

function CommentThread({
  contentId,
  rootComment,
  authorId,
  currentUserId,
  onReply,
  onDelete,
  deletePending,
  deletingCommentId,
}: {
  contentId: number;
  rootComment: InteractionCommentItem;
  authorId: number;
  currentUserId: number;
  onReply: (target: ReplyTarget) => void;
  onDelete: (comment: InteractionCommentItem) => void;
  deletePending: boolean;
  deletingCommentId: number | undefined;
}) {
  const [expanded, setExpanded] = useState(false);
  const isDeletingRootComment = deletePending && deletingCommentId === rootComment.comment_id;

  const repliesQuery = useInfiniteQuery({
    queryKey: contentKeys.replies(contentId, rootComment.comment_id),
    enabled: expanded && rootComment.reply_count > 0,
    initialPageParam: 0,
    queryFn: ({ pageParam }) =>
      queryReplyCommentList({
        comment_id: rootComment.comment_id,
        cursor: Number(pageParam) || 0,
        page_size: replyCommentPageSize,
      }),
    getNextPageParam: (lastPage) => (lastPage.has_more ? lastPage.next_cursor : undefined),
  });

  const replies = useMemo(() => flattenComments(repliesQuery.data), [repliesQuery.data]);
  const rootReplyTarget = createReplyTarget(rootComment, rootComment);
  const loadedReplyCount = replies.length;

  if (isDeletingRootComment) {
    return (
      <article className="rounded-[28px] border border-dashed border-slate-200 bg-slate-50 px-5 py-5 text-sm text-slate-500 shadow-sm">
        这条评论正在删除，评论区会自动刷新。
      </article>
    );
  }

  return (
    <article className="rounded-[28px] border border-slate-200 bg-white p-4 shadow-sm">
      <CommentEntry
        comment={rootComment}
        authorId={authorId}
        currentUserId={currentUserId}
        deleting={false}
        onDelete={() => onDelete(rootComment)}
        onReply={() => onReply(rootReplyTarget)}
      />

      {rootComment.reply_count > 0 ? (
        <div className="mt-4 border-t border-slate-100 pt-4">
          <div className="flex flex-wrap items-center justify-between gap-3">
            <button
              type="button"
              onClick={() => setExpanded((value) => !value)}
              className="text-sm font-medium text-accent transition hover:text-ink"
            >
              {expanded ? "收起回复" : `查看 ${rootComment.reply_count} 条回复`}
            </button>
            {expanded ? (
              <span className="rounded-full bg-[#eef7fb] px-3 py-1 text-xs text-slate-600">
                已展开 {loadedReplyCount} / {rootComment.reply_count}
              </span>
            ) : null}
          </div>

          {expanded ? (
            <div className="mt-4 space-y-3 border-l border-slate-200 pl-4">
              {repliesQuery.isLoading ? (
                Array.from({ length: Math.min(rootComment.reply_count, 2) }).map((_, index) => (
                  <div key={index} className="h-24 rounded-3xl bg-slate-100" />
                ))
              ) : repliesQuery.isError ? (
                <InlineNotice
                  title="回复列表加载失败"
                  description={(repliesQuery.error as Error)?.message || "请稍后重试"}
                  tone="error"
                />
              ) : replies.length === 0 ? (
                <InlineNotice
                  title="当前没有可显示的回复"
                  description="这层讨论可能刚被清理，或列表还在同步。"
                />
              ) : (
                replies.map((reply) => (
                  <CommentEntry
                    key={reply.comment_id}
                    comment={reply}
                    authorId={authorId}
                    currentUserId={currentUserId}
                    deleting={deletePending && deletingCommentId === reply.comment_id}
                    onDelete={() => onDelete(reply)}
                    onReply={() => onReply(createReplyTarget(rootComment, reply))}
                  />
                ))
              )}

              {repliesQuery.hasNextPage ? (
                <button
                  type="button"
                  onClick={() => repliesQuery.fetchNextPage()}
                  disabled={repliesQuery.isFetchingNextPage}
                  className="rounded-full border border-slate-200 bg-white px-4 py-2 text-sm text-slate-600 transition hover:border-accent hover:text-accent disabled:opacity-60"
                >
                  {repliesQuery.isFetchingNextPage ? "加载中..." : "查看更多回复"}
                </button>
              ) : replies.length > 0 ? (
                <p className="text-sm text-slate-500">这层回复已全部展示</p>
              ) : null}
            </div>
          ) : null}
        </div>
      ) : null}
    </article>
  );
}

function CommentEntry({
  comment,
  authorId,
  currentUserId,
  deleting,
  onReply,
  onDelete,
}: {
  comment: InteractionCommentItem;
  authorId: number;
  currentUserId: number;
  deleting: boolean;
  onReply: () => void;
  onDelete: () => void;
}) {
  const isAuthor = comment.user_id === authorId;
  const isOwner = comment.user_id === currentUserId;
  const displayName = comment.user_name || `用户 ${comment.user_id}`;

  if (deleting) {
    return (
      <div className="rounded-[24px] border border-dashed border-slate-200 bg-slate-50 px-4 py-4 text-sm text-slate-500">
        这条{comment.parent_id > 0 ? "回复" : "评论"}正在删除，列表会自动刷新。
      </div>
    );
  }

  return (
    <div className="rounded-[24px] bg-slate-50 p-4">
      <div className="flex items-start gap-3">
        <ImageFallback
          src={comment.user_avatar}
          alt={displayName}
          name={displayName}
          variant="avatar"
          containerClassName="h-11 w-11 overflow-hidden rounded-full bg-white ring-1 ring-slate-200"
          imageClassName="h-full w-full object-cover"
        />

        <div className="min-w-0 flex-1">
          <div className="flex flex-wrap items-center gap-2">
            <p className="text-sm font-semibold text-slate-900">{displayName}</p>
            {isAuthor ? (
              <span className="rounded-full bg-[#e9fbf7] px-2 py-0.5 text-[11px] font-medium text-accent">
                作者
              </span>
            ) : null}
            {isOwner ? (
              <span className="rounded-full bg-slate-200 px-2 py-0.5 text-[11px] font-medium text-slate-600">
                我
              </span>
            ) : null}
            <span className="text-xs text-slate-400">{formatDateTime(comment.created_at)}</span>
          </div>

          <p className="mt-2 whitespace-pre-wrap break-words text-sm leading-7 text-slate-700">
            {comment.comment}
          </p>

          <div className="mt-3 flex items-center gap-4 text-sm">
            <button
              type="button"
              onClick={onReply}
              className="text-slate-500 transition hover:text-accent"
            >
              回复
            </button>
            {isOwner ? (
              <button
                type="button"
                onClick={onDelete}
                disabled={deleting}
                className="text-slate-500 transition hover:text-ember disabled:opacity-60"
              >
                {deleting ? "删除中..." : "删除"}
              </button>
            ) : null}
          </div>
        </div>
      </div>
    </div>
  );
}

function MetricCard({ label, value }: { label: string; value: number }) {
  return (
    <article className="rounded-3xl border border-slate-200 bg-white p-4">
      <p className="text-sm text-slate-500">{label}</p>
      <p className="mt-2 text-2xl font-semibold text-slate-900">{value}</p>
    </article>
  );
}

function ActionButton({
  active,
  busy,
  label,
  meta,
  onClick,
}: {
  active: boolean;
  busy: boolean;
  label: string;
  meta: string;
  onClick: () => void;
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      disabled={busy}
      className={[
        "flex w-full items-center justify-between rounded-3xl border px-4 py-4 text-left transition",
        active
          ? "border-accent bg-[#e9fbf7] text-ink"
          : "border-slate-200 bg-white text-slate-900 hover:border-accent hover:text-accent",
        busy ? "opacity-70" : "",
      ].join(" ")}
    >
      <span className="font-medium">{busy ? "处理中..." : label}</span>
      <span className="text-sm text-slate-500">{meta}</span>
    </button>
  );
}
