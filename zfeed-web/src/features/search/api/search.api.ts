import { request } from "@/shared/lib/http/request";

export type SearchUserItem = {
  user_id: number;
  nickname: string;
  avatar: string;
  bio: string;
  is_following: boolean;
};

export type SearchUsersReq = {
  query: string;
  cursor?: number;
  page_size: number;
};

export type SearchUsersRes = {
  items: SearchUserItem[];
  next_cursor: number;
  has_more: boolean;
};

export type SearchContentItem = {
  content_id: number;
  content_type: number;
  author_id: number;
  author_name: string;
  author_avatar: string;
  title: string;
  cover_url: string;
  published_at: number;
};

export type SearchContentsReq = {
  query: string;
  cursor?: number;
  page_size: number;
};

export type SearchContentsRes = {
  items: SearchContentItem[];
  next_cursor: number;
  has_more: boolean;
};

export function searchUsers(payload: SearchUsersReq) {
  return request<SearchUsersRes, SearchUsersReq>({
    method: "POST",
    url: "/v1/search/users",
    data: payload,
  });
}

export function searchContents(payload: SearchContentsReq) {
  return request<SearchContentsRes, SearchContentsReq>({
    method: "POST",
    url: "/v1/search/contents",
    data: payload,
  });
}
