export const DEFAULT_FEED_PAGE_SIZE = 10;

function normalizeViewerId(viewerId?: number) {
  return viewerId && viewerId > 0 ? viewerId : 0;
}

export const feedKeys = {
  recommend: (viewerId?: number, pageSize = DEFAULT_FEED_PAGE_SIZE) =>
    ["feed", "recommend", normalizeViewerId(viewerId), pageSize] as const,
  recommendPrefix: (viewerId?: number) =>
    viewerId === undefined
      ? (["feed", "recommend"] as const)
      : (["feed", "recommend", normalizeViewerId(viewerId)] as const),
  follow: (viewerId?: number, pageSize = DEFAULT_FEED_PAGE_SIZE) =>
    ["feed", "follow", normalizeViewerId(viewerId), pageSize] as const,
  followPrefix: (viewerId?: number) =>
    viewerId === undefined
      ? (["feed", "follow"] as const)
      : (["feed", "follow", normalizeViewerId(viewerId)] as const),
  favorites: (userId: number, pageSize = DEFAULT_FEED_PAGE_SIZE) =>
    ["feed", "favorites", userId, pageSize] as const,
  favoritesPrefix: (userId?: number) =>
    userId && userId > 0
      ? (["feed", "favorites", userId] as const)
      : (["feed", "favorites"] as const),
  userPublish: (userId: number, viewerId?: number, pageSize = DEFAULT_FEED_PAGE_SIZE) =>
    ["user", "publish", userId, normalizeViewerId(viewerId), pageSize] as const,
  userPublishPrefix: (userId?: number, viewerId?: number) => {
    if (userId && userId > 0 && viewerId !== undefined) {
      return ["user", "publish", userId, normalizeViewerId(viewerId)] as const;
    }
    if (userId && userId > 0) {
      return ["user", "publish", userId] as const;
    }
    return ["user", "publish"] as const;
  },
  studioPublish: (userId: number, pageSize = DEFAULT_FEED_PAGE_SIZE) =>
    ["studio", "publish", userId, pageSize] as const,
  studioPublishPrefix: (userId?: number) =>
    userId && userId > 0 ? (["studio", "publish", userId] as const) : (["studio", "publish"] as const),
};

export const userKeys = {
  me: (viewerId?: number) => ["user", "me", normalizeViewerId(viewerId)] as const,
  mePrefix: () => ["user", "me"] as const,
  profile: (userId: number, viewerId?: number) =>
    ["user", "profile", userId, normalizeViewerId(viewerId)] as const,
  profilePrefix: (userId?: number) =>
    userId && userId > 0 ? (["user", "profile", userId] as const) : (["user", "profile"] as const),
  followers: (userId: number, viewerId?: number, pageSize = DEFAULT_FEED_PAGE_SIZE) =>
    ["user", "followers", userId, normalizeViewerId(viewerId), pageSize] as const,
  followersPrefix: (userId?: number) =>
    userId && userId > 0 ? (["user", "followers", userId] as const) : (["user", "followers"] as const),
};

export const contentKeys = {
  detail: (contentId: number, viewerId?: number) =>
    ["content", "detail", contentId, normalizeViewerId(viewerId)] as const,
  detailPrefix: () => ["content", "detail"] as const,
  comments: (contentId: number) => ["content", "comments", contentId] as const,
  replies: (contentId: number, rootCommentId: number) =>
    ["content", "comments", contentId, "replies", rootCommentId] as const,
};

export const searchKeys = {
  users: (query: string, viewerId?: number, pageSize = DEFAULT_FEED_PAGE_SIZE) =>
    ["search", "users", query, normalizeViewerId(viewerId), pageSize] as const,
  usersPrefix: () => ["search", "users"] as const,
  contents: (query: string, viewerId?: number, pageSize = DEFAULT_FEED_PAGE_SIZE) =>
    ["search", "contents", query, normalizeViewerId(viewerId), pageSize] as const,
  contentsPrefix: () => ["search", "contents"] as const,
};
