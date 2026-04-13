import { AxiosError, type AxiosRequestConfig } from "axios";

import { httpClient } from "@/shared/lib/http/client";

type ErrorPayload = {
  message?: string;
  msg?: string;
};

export function readMessage(error: AxiosError<ErrorPayload | string>): string {
  const payload = error.response?.data;
  if (typeof payload === "string" && payload.trim()) {
    return payload.trim();
  }

  if (payload && typeof payload === "object") {
    const text = String(payload.message ?? payload.msg ?? "").trim();
    if (text) {
      return text;
    }
  }

  return String(error.message);
}

export async function request<TResponse, TBody = unknown>(config: AxiosRequestConfig<TBody>) {
  try {
    const response = await httpClient.request<TResponse, { data: TResponse }, TBody>(config);
    return response.data;
  } catch (error) {
    if (error instanceof AxiosError) {
      throw new Error(readMessage(error));
    }
    throw error;
  }
}
