import { request } from "@/shared/lib/http/request";

export type FeedItem = {
  content_id: number;
  content_type: number;
  author_id: number;
  author_name: string;
  author_avatar: string;
  title: string;
  cover_url: string;
  published_at: number;
  is_liked: boolean;
  like_count: number;
};

export type CursorFeedRes = {
  items: FeedItem[];
  next_cursor: string;
  has_more: boolean;
};

export type RecommendReq = {
  cursor: string;
  page_size: number;
  snapshot_id?: string;
};

export type FollowReq = {
  cursor: string;
  page_size: number;
};

export type UserPublishReq = {
  user_id: number;
  cursor: string;
  page_size: number;
};

export type UserFavoriteReq = {
  user_id: number;
  cursor: string;
  page_size: number;
};

export type RecommendRes = CursorFeedRes & {
  snapshot_id: string;
};

export function getRecommend(payload: RecommendReq) {
  return request<RecommendRes, RecommendReq>({
    method: "POST",
    url: "/v1/feed/recommend",
    data: payload,
  });
}

export function getFollow(payload: FollowReq) {
  return request<CursorFeedRes, FollowReq>({
    method: "POST",
    url: "/v1/feed/follow",
    data: payload,
  });
}

export function getUserPublish(payload: UserPublishReq) {
  return request<CursorFeedRes, UserPublishReq>({
    method: "POST",
    url: "/v1/feed/user/publish",
    data: payload,
  });
}

export function getUserFavorite(payload: UserFavoriteReq) {
  return request<CursorFeedRes, UserFavoriteReq>({
    method: "POST",
    url: "/v1/feed/user/favorite",
    data: payload,
  });
}
