import { type InfiniteData, type QueryClient } from "@tanstack/react-query";

import { type MeRes } from "@/features/auth/api/auth.api";
import { type ContentDetail, type GetContentDetailRes } from "@/features/content/api/content.api";
import { type CursorFeedRes, type FeedItem, type RecommendRes } from "@/features/feed/api/feed.api";
import { type SearchUserItem, type SearchUsersRes } from "@/features/search/api/search.api";
import { type QueryFollowersRes, type UserProfileRes, type FollowerItem } from "@/features/user/api/user.api";
import { contentKeys, feedKeys, searchKeys, userKeys } from "@/shared/lib/query/queryKeys";

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

function removeInfiniteFeedItems(
  current: InfiniteData<InfiniteFeedPage> | undefined,
  contentId: number,
) {
  if (!current) {
    return current;
  }

  let touched = false;

  const pages = current.pages.map((page) => {
    const items = page.items.filter((item) => item.content_id !== contentId);
    if (items.length !== page.items.length) {
      touched = true;
      return { ...page, items };
    }
    return page;
  });

  return touched ? { ...current, pages } : current;
}

function patchSearchUserItems(
  current: InfiniteData<SearchUsersRes> | undefined,
  userId: number,
  updater: (item: SearchUserItem) => SearchUserItem,
) {
  if (!current) {
    return current;
  }

  let touched = false;

  const pages = current.pages.map((page) => {
    let pageTouched = false;
    const items = page.items.map((item) => {
      if (item.user_id !== userId) {
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

function patchFollowerItems(
  current: InfiniteData<QueryFollowersRes> | undefined,
  userId: number,
  updater: (item: FollowerItem) => FollowerItem,
) {
  if (!current) {
    return current;
  }

  let touched = false;

  const pages = current.pages.map((page) => {
    let pageTouched = false;
    const items = page.items.map((item) => {
      if (item.user_id !== userId) {
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

export function removeContentAcrossCollections(queryClient: QueryClient, contentId: number) {
  [
    feedKeys.recommendPrefix(),
    feedKeys.followPrefix(),
    feedKeys.favoritesPrefix(),
    feedKeys.userPublishPrefix(),
    feedKeys.studioPublishPrefix(),
  ].forEach((queryKey) => {
    queryClient.setQueriesData<InfiniteData<InfiniteFeedPage>>({ queryKey }, (current) =>
      removeInfiniteFeedItems(current, contentId),
    );
  });
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
  const delta = nextIsFollowing ? 1 : -1;

  patchContentDetailsByAuthor(queryClient, authorId, (detail) => ({
    ...detail,
    is_following_author: nextIsFollowing,
  }));

  queryClient.setQueriesData<UserProfileRes>({ queryKey: userKeys.profilePrefix(authorId) }, (current) => {
    if (!current || current.user_profile.user_id !== authorId) {
      return current;
    }

    return {
      ...current,
      counts: {
        ...current.counts,
        follower_count: Math.max(0, current.counts.follower_count + delta),
      },
      viewer: {
        ...current.viewer,
        is_following: nextIsFollowing,
      },
    };
  });

  if (viewerId > 0) {
    queryClient.setQueryData<MeRes>(userKeys.me(viewerId), (current) => {
      if (!current) {
        return current;
      }

      return {
        ...current,
        followee_count: Math.max(0, current.followee_count + delta),
      };
    });

    queryClient.setQueriesData<UserProfileRes>(
      { queryKey: userKeys.profilePrefix(viewerId) },
      (current) => {
        if (!current || current.user_profile.user_id !== viewerId) {
          return current;
        }

        return {
          ...current,
          counts: {
            ...current.counts,
            followee_count: Math.max(0, current.counts.followee_count + delta),
          },
        };
      },
    );
  }

  queryClient.setQueriesData<InfiniteData<SearchUsersRes>>(
    { queryKey: searchKeys.usersPrefix() },
    (current) =>
      patchSearchUserItems(current, authorId, (item) => ({
        ...item,
        is_following: nextIsFollowing,
      })),
  );

  queryClient.setQueriesData<InfiniteData<QueryFollowersRes>>(
    { queryKey: userKeys.followersPrefix() },
    (current) =>
      patchFollowerItems(current, authorId, (item) => ({
        ...item,
        is_following: nextIsFollowing,
      })),
  );
}

export function getFollowSyncQueryKeys(targetUserId: number, viewerId: number) {
  return [
    contentKeys.detailPrefix(),
    userKeys.profilePrefix(targetUserId),
    searchKeys.usersPrefix(),
    userKeys.followersPrefix(),
    ...(viewerId > 0 ? [userKeys.mePrefix(), userKeys.profilePrefix(viewerId)] : []),
  ] as const;
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
