import { AxiosError } from "axios";
import { describe, expect, it } from "vitest";

import { readMessage } from "@/shared/lib/http/request";

type ErrorPayload = {
  message?: string;
  msg?: string;
};

function createAxiosError(data: ErrorPayload | string) {
  return new AxiosError<ErrorPayload | string>("Request failed with status code 500", undefined, undefined, undefined, {
    data,
    status: 500,
    statusText: "Internal Server Error",
    headers: {},
    config: { headers: {} } as never,
  });
}

describe("readMessage", () => {
  it("prefers backend message field", () => {
    const error = createAxiosError({ message: "手机号已注册" });

    expect(readMessage(error)).toBe("手机号已注册");
  });

  it("falls back to plain text response bodies", () => {
    const error = createAxiosError("手机号已注册");

    expect(readMessage(error)).toBe("手机号已注册");
  });
});
