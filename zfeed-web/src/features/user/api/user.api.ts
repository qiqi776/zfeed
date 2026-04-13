import { request } from "@/shared/lib/http/request";

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

export function getUserProfile(userId: number) {
  return request<UserProfileRes>({
    method: "GET",
    url: `/v1/user/profile/${userId}`,
  });
}
