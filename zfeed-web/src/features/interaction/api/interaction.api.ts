import { request } from "@/shared/lib/http/request";

export type InteractionScene = "ARTICLE" | "VIDEO";

export type CommentItem = {
  comment_id: number;
  content_id: number;
  user_id: number;
  reply_to_user_id: number;
  parent_id: number;
  root_id: number;
  comment: string;
  created_at: number;
  status: number;
  user_name: string;
  user_avatar: string;
  reply_count: number;
};

export type LikePayload = {
  content_id: number;
  content_user_id: number;
  scene: InteractionScene;
};

export type UnlikePayload = {
  content_id: number;
  scene: InteractionScene;
};

export type FavoritePayload = {
  content_id: number;
  content_user_id: number;
  scene: InteractionScene;
};

export type RemoveFavoritePayload = {
  content_id: number;
  scene: InteractionScene;
};

export type CommentPayload = {
  content_id: number;
  content_user_id: number;
  scene: InteractionScene;
  comment: string;
  parent_id?: number;
  root_id?: number;
  reply_to_user_id?: number;
};

export type CommentRes = {
  comment_id: number;
};

export type DeleteCommentPayload = {
  comment_id: number;
  content_id: number;
  scene: InteractionScene;
  root_id?: number;
  parent_id?: number;
};

export type QueryCommentListPayload = {
  content_id: number;
  scene: InteractionScene;
  cursor: number;
  page_size: number;
};

export type QueryCommentListRes = {
  comments: CommentItem[];
  next_cursor: number;
  has_more: boolean;
};

export type QueryReplyCommentListPayload = {
  comment_id: number;
  cursor: number;
  page_size: number;
};

export type QueryReplyCommentListRes = {
  comments: CommentItem[];
  next_cursor: number;
  has_more: boolean;
};

export type FollowUserPayload = {
  target_user_id: number;
};

export type FollowUserRes = {
  is_followed: boolean;
};

export function likeContent(payload: LikePayload) {
  return request<Record<string, never>, LikePayload>({
    method: "POST",
    url: "/v1/interaction/like",
    data: payload,
  });
}

export function unlikeContent(payload: UnlikePayload) {
  return request<Record<string, never>, UnlikePayload>({
    method: "POST",
    url: "/v1/interaction/unlike",
    data: payload,
  });
}

export function favoriteContent(payload: FavoritePayload) {
  return request<Record<string, never>, FavoritePayload>({
    method: "POST",
    url: "/v1/interaction/favorite",
    data: payload,
  });
}

export function removeFavorite(payload: RemoveFavoritePayload) {
  return request<Record<string, never>, RemoveFavoritePayload>({
    method: "DELETE",
    url: "/v1/interaction/favorite",
    data: payload,
  });
}

export function commentContent(payload: CommentPayload) {
  return request<CommentRes, CommentPayload>({
    method: "POST",
    url: "/v1/interaction/comment",
    data: payload,
  });
}

export function deleteComment(payload: DeleteCommentPayload) {
  return request<Record<string, never>, DeleteCommentPayload>({
    method: "DELETE",
    url: "/v1/interaction/comment",
    data: payload,
  });
}

export function queryCommentList(payload: QueryCommentListPayload) {
  return request<QueryCommentListRes, QueryCommentListPayload>({
    method: "POST",
    url: "/v1/interaction/comment/list",
    data: payload,
  });
}

export function queryReplyCommentList(payload: QueryReplyCommentListPayload) {
  return request<QueryReplyCommentListRes, QueryReplyCommentListPayload>({
    method: "POST",
    url: "/v1/interaction/comment/reply/list",
    data: payload,
  });
}

export function followUser(payload: FollowUserPayload) {
  return request<FollowUserRes, FollowUserPayload>({
    method: "POST",
    url: "/v1/interaction/followings",
    data: payload,
  });
}

export function unfollowUser(payload: FollowUserPayload) {
  return request<FollowUserRes, FollowUserPayload>({
    method: "DELETE",
    url: "/v1/interaction/followings",
    data: payload,
  });
}
