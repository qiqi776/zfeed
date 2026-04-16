import { QueryClient, type InfiniteData } from "@tanstack/react-query";

import { type MeRes } from "@/features/auth/api/auth.api";
import {
  type ContentDetail,
  type GetContentDetailRes,
} from "@/features/content/api/content.api";
import {
  type CursorFeedRes,
  type FeedItem,
  type RecommendRes,
} from "@/features/feed/api/feed.api";
import {
  type FollowerItem,
  type QueryFollowersRes,
  type UserProfileRes,
} from "@/features/user/api/user.api";
import {
  type SearchUsersRes,
  type SearchUserItem,
} from "@/features/search/api/search.api";
import {
  patchAuthorFollowStateAcrossPages,
  patchLikeStateAcrossCollections,
  removeContentAcrossCollections,
} from "@/shared/lib/query/cacheSync";
import { contentKeys, feedKeys, searchKeys, userKeys } from "@/shared/lib/query/queryKeys";

function createFeedItem(overrides: Partial<FeedItem> = {}): FeedItem {
  return {
    content_id: 101,
    content_type: 10,
    author_id: 9,
    author_name: "作者",
    author_avatar: "",
    title: "标题",
    cover_url: "",
    published_at: 1,
    is_liked: false,
    like_count: 3,
    ...overrides,
  };
}

function createCursorFeedData(items: FeedItem[]): InfiniteData<CursorFeedRes> {
  return {
    pageParams: [{ cursor: "0" }],
    pages: [
      {
        items,
        next_cursor: "0",
        has_more: false,
      },
    ],
  };
}

function createRecommendData(items: FeedItem[]): InfiniteData<RecommendRes> {
  return {
    pageParams: [{ cursor: "0", snapshotId: "" }],
    pages: [
      {
        items,
        next_cursor: "0",
        has_more: false,
        snapshot_id: "snapshot-1",
      },
    ],
  };
}

function createProfileData(userId: number): UserProfileRes {
  return {
    user_profile: {
      user_id: userId,
      nickname: "作者",
      avatar: "",
      bio: "",
      gender: 0,
    },
    counts: {
      followee_count: 4,
      follower_count: 8,
      like_received_count: 0,
      favorite_received_count: 0,
      content_count: 2,
    },
    viewer: {
      is_following: false,
    },
  };
}

function createSearchUserItem(overrides: Partial<SearchUserItem> = {}): SearchUserItem {
  return {
    user_id: 9,
    nickname: "作者",
    avatar: "",
    bio: "",
    is_following: false,
    ...overrides,
  };
}

function createSearchUsersData(items: SearchUserItem[]): InfiniteData<SearchUsersRes> {
  return {
    pageParams: [{ cursor: undefined }],
    pages: [
      {
        items,
        next_cursor: 0,
        has_more: false,
      },
    ],
  };
}

function createFollowerItem(overrides: Partial<FollowerItem> = {}): FollowerItem {
  return {
    user_id: 9,
    nickname: "作者",
    avatar: "",
    bio: "",
    is_following: false,
    ...overrides,
  };
}

function createFollowersData(items: FollowerItem[]): InfiniteData<QueryFollowersRes> {
  return {
    pageParams: [{ cursor: undefined }],
    pages: [
      {
        items,
        next_cursor: 0,
        has_more: false,
      },
    ],
  };
}

function createMeData(): MeRes {
  return {
    user_info: {
      user_id: 1,
      mobile: "13800000000",
      nickname: "我",
      avatar: "",
      bio: "",
      gender: 0,
      status: 1,
      email: "me@example.com",
      birthday: 946684800,
    },
    followee_count: 6,
    follower_count: 0,
    like_received_count: 0,
    favorite_received_count: 0,
    content_count: 2,
  };
}

function createDetail(authorId: number, overrides: Partial<ContentDetail> = {}): GetContentDetailRes {
  return {
    detail: {
      content_id: 101,
      content_type: 10,
      author_id: authorId,
      author_name: "作者",
      author_avatar: "",
      title: "标题",
      description: "",
      cover_url: "",
      article_content: "",
      video_url: "",
      video_duration: 0,
      published_at: 1,
      like_count: 3,
      favorite_count: 0,
      comment_count: 0,
      is_liked: false,
      is_favorited: false,
      is_following_author: false,
      ...overrides,
    },
  };
}

describe("cacheSync", () => {
  it("patches like state across cached feed collections", () => {
    const queryClient = new QueryClient();
    const viewerId = 1;
    const item = createFeedItem();

    queryClient.setQueryData(feedKeys.recommend(viewerId), createRecommendData([item]));
    queryClient.setQueryData(feedKeys.follow(viewerId), createCursorFeedData([item]));
    queryClient.setQueryData(feedKeys.favorites(viewerId), createCursorFeedData([item]));
    queryClient.setQueryData(feedKeys.userPublish(9, viewerId), createCursorFeedData([item]));
    queryClient.setQueryData(feedKeys.studioPublish(viewerId), createCursorFeedData([item]));

    patchLikeStateAcrossCollections(queryClient, item.content_id, true, 1);

    expect(
      queryClient.getQueryData<InfiniteData<RecommendRes>>(feedKeys.recommend(viewerId))?.pages[0].items[0],
    ).toMatchObject({ is_liked: true, like_count: 4 });
    expect(
      queryClient.getQueryData<InfiniteData<CursorFeedRes>>(feedKeys.follow(viewerId))?.pages[0].items[0],
    ).toMatchObject({ is_liked: true, like_count: 4 });
    expect(
      queryClient.getQueryData<InfiniteData<CursorFeedRes>>(feedKeys.favorites(viewerId))?.pages[0].items[0],
    ).toMatchObject({ is_liked: true, like_count: 4 });
    expect(
      queryClient.getQueryData<InfiniteData<CursorFeedRes>>(feedKeys.userPublish(9, viewerId))
        ?.pages[0].items[0],
    ).toMatchObject({ is_liked: true, like_count: 4 });
    expect(
      queryClient.getQueryData<InfiniteData<CursorFeedRes>>(feedKeys.studioPublish(viewerId))
        ?.pages[0].items[0],
    ).toMatchObject({ is_liked: true, like_count: 4 });
  });

  it("patches follow state across detail, profile and me caches", () => {
    const queryClient = new QueryClient();
    const viewerId = 1;
    const authorId = 9;

    queryClient.setQueryData(contentKeys.detail(101, viewerId), createDetail(authorId));
    queryClient.setQueryData(contentKeys.detail(102, viewerId), createDetail(authorId, { content_id: 102 }));
    queryClient.setQueryData(userKeys.profile(authorId, viewerId), createProfileData(authorId));
    queryClient.setQueryData(userKeys.profile(viewerId, viewerId), createProfileData(viewerId));
    queryClient.setQueryData(userKeys.me(viewerId), createMeData());
    queryClient.setQueryData(
      searchKeys.users("作者", viewerId),
      createSearchUsersData([createSearchUserItem(), createSearchUserItem({ user_id: 10 })]),
    );
    queryClient.setQueryData(
      userKeys.followers(authorId, viewerId),
      createFollowersData([createFollowerItem(), createFollowerItem({ user_id: 8 })]),
    );

    patchAuthorFollowStateAcrossPages(queryClient, authorId, viewerId, true);

    expect(queryClient.getQueryData<GetContentDetailRes>(contentKeys.detail(101, viewerId))?.detail)
      .toMatchObject({ is_following_author: true });
    expect(queryClient.getQueryData<GetContentDetailRes>(contentKeys.detail(102, viewerId))?.detail)
      .toMatchObject({ is_following_author: true });
    expect(queryClient.getQueryData<UserProfileRes>(userKeys.profile(authorId, viewerId)))
      .toMatchObject({
        viewer: { is_following: true },
        counts: { follower_count: 9 },
      });
    expect(queryClient.getQueryData<UserProfileRes>(userKeys.profile(viewerId, viewerId)))
      .toMatchObject({
        counts: { followee_count: 5 },
      });
    expect(queryClient.getQueryData<MeRes>(userKeys.me(viewerId)))
      .toMatchObject({ followee_count: 7 });
    expect(
      queryClient.getQueryData<InfiniteData<SearchUsersRes>>(searchKeys.users("作者", viewerId))
        ?.pages[0].items[0],
    ).toMatchObject({ is_following: true });
    expect(
      queryClient.getQueryData<InfiniteData<QueryFollowersRes>>(userKeys.followers(authorId, viewerId))
        ?.pages[0].items[0],
    ).toMatchObject({ is_following: true });
  });

  it("removes deleted content across cached feed collections", () => {
    const queryClient = new QueryClient();
    const viewerId = 1;
    const retained = createFeedItem({ content_id: 202, title: "保留内容" });
    const deleted = createFeedItem({ content_id: 101, title: "待删除内容" });

    queryClient.setQueryData(feedKeys.recommend(viewerId), createRecommendData([deleted, retained]));
    queryClient.setQueryData(feedKeys.follow(viewerId), createCursorFeedData([deleted, retained]));
    queryClient.setQueryData(feedKeys.favorites(viewerId), createCursorFeedData([retained, deleted]));
    queryClient.setQueryData(feedKeys.userPublish(9, viewerId), createCursorFeedData([deleted]));
    queryClient.setQueryData(feedKeys.studioPublish(viewerId), createCursorFeedData([deleted, retained]));

    removeContentAcrossCollections(queryClient, deleted.content_id);

    expect(
      queryClient.getQueryData<InfiniteData<RecommendRes>>(feedKeys.recommend(viewerId))?.pages[0].items,
    ).toEqual([retained]);
    expect(
      queryClient.getQueryData<InfiniteData<CursorFeedRes>>(feedKeys.follow(viewerId))?.pages[0].items,
    ).toEqual([retained]);
    expect(
      queryClient.getQueryData<InfiniteData<CursorFeedRes>>(feedKeys.favorites(viewerId))?.pages[0].items,
    ).toEqual([retained]);
    expect(
      queryClient.getQueryData<InfiniteData<CursorFeedRes>>(feedKeys.userPublish(9, viewerId))?.pages[0].items,
    ).toEqual([]);
    expect(
      queryClient.getQueryData<InfiniteData<CursorFeedRes>>(feedKeys.studioPublish(viewerId))?.pages[0].items,
    ).toEqual([retained]);
  });
});
