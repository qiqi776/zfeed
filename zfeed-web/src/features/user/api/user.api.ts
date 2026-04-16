import { request } from "@/shared/lib/http/request";
import { avatarUploadRule, validateSelectedFile } from "@/shared/lib/media/fileValidation";

export type UserProfileRes = {
  user_profile: {
    user_id: number;
    nickname: string;
    avatar: string;
    bio: string;
    gender: number;
  };
  counts: {
    followee_count: number;
    follower_count: number;
    like_received_count: number;
    favorite_received_count: number;
    content_count: number;
  };
  viewer: {
    is_following: boolean;
  };
};

export type UpdateProfileReq = {
  nickname?: string;
  avatar?: string;
  bio?: string;
  gender?: number;
  email?: string;
  birthday?: number;
};

export type UpdateProfileRes = {
  user_info: {
    user_id: number;
    mobile: string;
    nickname: string;
    avatar: string;
    bio: string;
    gender: number;
    status: number;
    email: string;
    birthday: number;
  };
};

export type UploadAvatarRes = {
  url: string;
  object_key: string;
  mime: string;
  size: number;
};

export type FollowerItem = {
  user_id: number;
  nickname: string;
  avatar: string;
  bio: string;
  is_following: boolean;
};

export type QueryFollowersReq = {
  user_id: number;
  cursor?: number;
  page_size: number;
};

export type QueryFollowersRes = {
  items: FollowerItem[];
  next_cursor: number;
  has_more: boolean;
};

export function getUserProfile(userId: number) {
  return request<UserProfileRes>({
    method: "GET",
    url: `/v1/user/profile/${userId}`,
  });
}

export function updateProfile(payload: UpdateProfileReq) {
  return request<UpdateProfileRes, UpdateProfileReq>({
    method: "PUT",
    url: "/v1/users/me/profile",
    data: payload,
  });
}

export function uploadAvatar(file: File) {
  validateSelectedFile(file, avatarUploadRule);
  const formData = new FormData();
  formData.append("file", file);
  return request<UploadAvatarRes, FormData>({
    method: "POST",
    url: "/v1/users/avatar/upload",
    data: formData,
  });
}

export function queryFollowers(payload: QueryFollowersReq) {
  return request<QueryFollowersRes, QueryFollowersReq>({
    method: "POST",
    url: "/v1/user/followers",
    data: payload,
  });
}
