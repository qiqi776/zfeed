import { type InfiniteData, type QueryClient } from "@tanstack/react-query";

import { type MeRes } from "@/features/auth/api/auth.api";
import { type ContentDetail, type GetContentDetailRes } from "@/features/content/api/content.api";
import { type CursorFeedRes, type FeedItem, type RecommendRes } from "@/features/feed/api/feed.api";
import { type UserProfileRes } from "@/features/user/api/user.api";
import { contentKeys, feedKeys, userKeys } from "@/shared/lib/query/queryKeys";

type InfiniteFeedPage = CursorFeedRes | RecommendRes;

export type QuerySnapshot = readonly [readonly unknown[], unknown];

function patchInfiniteFeedItems(
  current: InfiniteData<InfiniteFeedPage> | undefined,
  contentId: number,
  updater: (item: FeedItem) => FeedItem,
) {
  if (!current) {
    return current;
  }

  let touched = false;

  const pages = current.pages.map((page) => {
    let pageTouched = false;
    const items = page.items.map((item) => {
      if (item.content_id !== contentId) {
        return item;
      }
      pageTouched = true;
      touched = true;
      return updater(item);
    });

    return pageTouched ? { ...page, items } : page;
  });

  return touched ? { ...current, pages } : current;
}

export function captureQuerySnapshots(
  queryClient: QueryClient,
  queryKeys: readonly (readonly unknown[])[],
) {
  return queryKeys.flatMap(
    (queryKey) => queryClient.getQueriesData({ queryKey }) as QuerySnapshot[],
  );
}

export function restoreQuerySnapshots(queryClient: QueryClient, snapshots: readonly QuerySnapshot[]) {
  snapshots.forEach(([queryKey, data]) => {
    queryClient.setQueryData(queryKey, data);
  });
}

export function patchFeedItemAcrossCollections(
  queryClient: QueryClient,
  contentId: number,
  updater: (item: FeedItem) => FeedItem,
) {
  [
    feedKeys.recommendPrefix(),
    feedKeys.followPrefix(),
    feedKeys.favoritesPrefix(),
    feedKeys.userPublishPrefix(),
    feedKeys.studioPublishPrefix(),
  ].forEach((queryKey) => {
    queryClient.setQueriesData<InfiniteData<InfiniteFeedPage>>({ queryKey }, (current) =>
      patchInfiniteFeedItems(current, contentId, updater),
    );
  });
}

export function patchLikeStateAcrossCollections(
  queryClient: QueryClient,
  contentId: number,
  nextIsLiked: boolean,
  delta: number,
) {
  patchFeedItemAcrossCollections(queryClient, contentId, (item) => ({
    ...item,
    is_liked: nextIsLiked,
    like_count: Math.max(0, item.like_count + delta),
  }));
}

export function patchContentDetail(
  queryClient: QueryClient,
  contentId: number,
  viewerId: number,
  updater: (detail: ContentDetail) => ContentDetail,
) {
  queryClient.setQueryData<GetContentDetailRes>(contentKeys.detail(contentId, viewerId), (current) => {
    if (!current) {
      return current;
    }
    return { detail: updater(current.detail) };
  });
}

export function patchContentDetailsByAuthor(
  queryClient: QueryClient,
  authorId: number,
  updater: (detail: ContentDetail) => ContentDetail,
) {
  queryClient.setQueriesData<GetContentDetailRes>({ queryKey: contentKeys.detailPrefix() }, (current) => {
    if (!current || current.detail.author_id !== authorId) {
      return current;
    }

    return { detail: updater(current.detail) };
  });
}

export function patchAuthorFollowStateAcrossPages(
  queryClient: QueryClient,
  authorId: number,
  viewerId: number,
  nextIsFollowing: boolean,
) {
  const followerDelta = nextIsFollowing ? 1 : -1;
  const followeeDelta = nextIsFollowing ? 1 : -1;

  patchContentDetailsByAuthor(queryClient, authorId, (detail) => ({
    ...detail,
    is_following_author: nextIsFollowing,
  }));

  queryClient.setQueryData<UserProfileRes>(userKeys.profile(authorId, viewerId), (current) => {
    if (!current) {
      return current;
    }

    return {
      ...current,
      counts: {
        ...current.counts,
        follower_count: Math.max(0, current.counts.follower_count + followerDelta),
      },
      viewer: {
        ...current.viewer,
        is_following: nextIsFollowing,
      },
    };
  });

  queryClient.setQueryData<MeRes>(userKeys.me(viewerId), (current) => {
    if (!current) {
      return current;
    }

    return {
      ...current,
      followee_count: Math.max(0, current.followee_count + followeeDelta),
    };
  });
}

export async function invalidatePublishSurfaces(queryClient: QueryClient, userId: number) {
  if (userId <= 0) {
    return;
  }

  await Promise.all([
    queryClient.invalidateQueries({ queryKey: feedKeys.userPublishPrefix(userId) }),
    queryClient.invalidateQueries({ queryKey: feedKeys.studioPublishPrefix(userId) }),
    queryClient.invalidateQueries({ queryKey: userKeys.profilePrefix(userId) }),
  ]);
}
