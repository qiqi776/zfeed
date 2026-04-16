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

export type EditArticleReq = {
  title?: string;
  description?: string;
  cover?: string;
  content?: string;
};

export type EditArticleRes = {
  content_id: number;
};

export type EditVideoReq = {
  title?: string;
  description?: string;
  video_url?: string;
  cover_url?: string;
  duration?: number;
};

export type EditVideoRes = {
  content_id: number;
};

export type ContentUploadScene =
  | "avatar"
  | "article-cover"
  | "video-cover"
  | "video-source";

export type ContentUploadCredentialsReq = {
  scene: ContentUploadScene;
  file_ext: string;
  file_size: number;
  file_name: string;
};

export type ContentUploadCredentialsRes = {
  object_key: string;
  url: string;
  expired_at: number;
  form_data: {
    host: string;
    policy: string;
    signature: string;
    "x-oss-security-token": string;
    "x-oss-signature-version": string;
    "x-oss-credential": string;
    "x-oss-date": string;
    key: string;
  };
};

export function deleteContent(contentId: number) {
  return request<Record<string, never>>({
    method: "DELETE",
    url: `/v1/content/${contentId}`,
  });
}

export function editArticle(contentId: number, payload: EditArticleReq) {
  return request<EditArticleRes, EditArticleReq>({
    method: "PUT",
    url: `/v1/content/article/${contentId}`,
    data: payload,
  });
}

export function editVideo(contentId: number, payload: EditVideoReq) {
  return request<EditVideoRes, EditVideoReq>({
    method: "PUT",
    url: `/v1/content/video/${contentId}`,
    data: payload,
  });
}

export function getContentUploadCredentials(payload: ContentUploadCredentialsReq) {
  return request<ContentUploadCredentialsRes, ContentUploadCredentialsReq>({
    method: "POST",
    url: "/v1/content/upload-credentials",
    data: payload,
  });
}

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
