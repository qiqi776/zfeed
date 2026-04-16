import { vi } from "vitest";

import {
  inferUploadExt,
  postSignedUpload,
  uploadContentAsset,
} from "@/features/content/lib/upload";

const getContentUploadCredentialsMock = vi.fn();

vi.mock("@/features/content/api/content.api", () => ({
  getContentUploadCredentials: (...args: unknown[]) => getContentUploadCredentialsMock(...args),
}));

describe("upload helper", () => {
  beforeEach(() => {
    getContentUploadCredentialsMock.mockReset();
  });

  it("infers extension from file name", () => {
    const file = new File(["cover"], "cover.PNG", { type: "image/png" });
    expect(inferUploadExt(file)).toBe(".png");
  });

  it("posts signed upload form data", async () => {
    const fetchMock = vi.fn().mockResolvedValue({ ok: true });
    const file = new File(["cover"], "cover.png", { type: "image/png" });

    await postSignedUpload(
      file,
      {
        object_key: "uploads/article-cover/cover.png",
        url: "https://cdn.example.com/uploads/article-cover/cover.png",
        expired_at: 1,
        form_data: {
          host: "https://oss.example.com",
          policy: "policy",
          signature: "signature",
          "x-oss-security-token": "",
          "x-oss-signature-version": "OSS4-HMAC-SHA256",
          "x-oss-credential": "cred",
          "x-oss-date": "20260414T000000Z",
          key: "uploads/article-cover/cover.png",
        },
      },
      fetchMock as unknown as typeof fetch,
    );

    expect(fetchMock).toHaveBeenCalledTimes(1);
    expect(fetchMock.mock.calls[0]?.[0]).toBe("https://oss.example.com");
    expect(fetchMock.mock.calls[0]?.[1]).toMatchObject({ method: "POST" });
  });

  it("rejects unsupported image file before requesting upload credentials", async () => {
    const file = new File(["gif"], "cover.gif", { type: "image/gif" });

    await expect(uploadContentAsset(file, "article-cover")).rejects.toThrow(
      "图片文件类型不支持",
    );
    expect(getContentUploadCredentialsMock).not.toHaveBeenCalled();
  });
});
