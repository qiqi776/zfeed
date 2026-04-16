import { request } from "@/shared/lib/http/request";

export type LoginReq = {
  mobile: string;
  password: string;
};

export type LoginRes = {
  user_id: number;
  token: string;
  expired_at: number;
  nickname: string;
  avatar: string;
};

export type RegisterReq = {
  mobile: string;
  password: string;
  nickname: string;
  avatar: string;
  bio: string;
  gender: number;
  email: string;
  birthday: number;
};

export type RegisterRes = {
  user_id: number;
  token: string;
  expired_at: number;
};

export type MeRes = {
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
  followee_count: number;
  follower_count: number;
  like_received_count: number;
  favorite_received_count: number;
  content_count: number;
};

export function login(payload: LoginReq) {
  return request<LoginRes, LoginReq>({
    method: "POST",
    url: "/v1/login",
    data: payload,
  });
}

export function register(payload: RegisterReq) {
  return request<RegisterRes, RegisterReq>({
    method: "POST",
    url: "/v1/users",
    data: payload,
  });
}

export function logout() {
  return request<Record<string, never>>({
    method: "POST",
    url: "/v1/logout",
  });
}

export function getMe() {
  return request<MeRes>({
    method: "GET",
    url: "/v1/users/me",
  });
}
