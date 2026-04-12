import { request } from "@/shared/lib/http/request";

export type ContentDetail = {
  content_id: number;
  content_type: number;
  author_id: number;
  author_name: string;
  author_avatar: string;
  title: string;
  description: string;
  cover_url: string;
  article_content: string;
  video_url: string;
  video_duration: number;
  published_at: number;
  like_count: number;
  favorite_count: number;
  comment_count: number;
  is_liked: boolean;
  is_favorited: boolean;
  is_following_author: boolean;
};

export type GetContentDetailReq = {
  content_id: number;
};

export type GetContentDetailRes = {
  detail: ContentDetail;
};

export type PublishArticleReq = {
  title: string;
  description?: string;
  cover: string;
  content: string;
  visibility: number;
};

export type PublishArticleRes = {
  content_id: number;
};

export type PublishVideoReq = {
  title: string;
  description?: string;
  video_url: string;
  cover_url: string;
  duration?: number;
  visibility: number;
};

export type PublishVideoRes = {
  content_id: number;
};

export function getContentDetail(payload: GetContentDetailReq) {
  return request<GetContentDetailRes, GetContentDetailReq>({
    method: "POST",
    url: "/v1/content/detail",
    data: payload,
  });
}

export function publishArticle(payload: PublishArticleReq) {
  return request<PublishArticleRes, PublishArticleReq>({
    method: "POST",
    url: "/v1/content/article/publish",
    data: payload,
  });
}

export function publishVideo(payload: PublishVideoReq) {
  return request<PublishVideoRes, PublishVideoReq>({
    method: "POST",
    url: "/v1/content/video/publish",
    data: payload,
  });
}
