import axios, { AxiosError } from "axios";

import { useSessionStore } from "@/entities/session/model/session.store";
import { env } from "@/shared/config/env";
import { queryClient } from "@/shared/lib/query/queryClient";

type ErrorPayload = {
  code?: number;
  msg?: string;
  message?: string;
};

function getErrorText(payload: ErrorPayload | string | undefined): string {
  if (!payload) {
    return "";
  }
  if (typeof payload === "string") {
    return payload.trim();
  }
  return String(payload.message ?? payload.msg ?? "").trim();
}

function shouldDropSession(error: AxiosError<ErrorPayload | string>): boolean {
  const status = error.response?.status ?? 0;
  if (status === 401 || status === 403) {
    return true;
  }

  const text = getErrorText(error.response?.data);
  if (text.includes("未登录") || text.includes("token") || text.includes("登录态")) {
    return true;
  }

  const code = typeof error.response?.data === "object" ? error.response?.data?.code : undefined;
  return code === 401 || code === 403;
}

export const httpClient = axios.create({
  baseURL: env.apiBaseUrl || undefined,
  timeout: 12_000,
});

httpClient.interceptors.request.use((config) => {
  const token = useSessionStore.getState().token;
  if (token) {
    config.headers.Authorization = `Bearer ${token}`;
  }
  return config;
});

httpClient.interceptors.response.use(
  (response) => response,
  (error: AxiosError<ErrorPayload | string>) => {
    if (shouldDropSession(error)) {
      queryClient.clear();
      useSessionStore.getState().clearSession();
      if (typeof window !== "undefined" && !window.location.pathname.startsWith("/login")) {
        const next = `${window.location.pathname}${window.location.search}`;
        window.location.href = `/login?next=${encodeURIComponent(next)}`;
      }
    }
    return Promise.reject(error);
  },
);
