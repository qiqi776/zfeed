import {
  avatarUploadRule,
  contentVideoUploadRule,
  describeFileValidationRule,
  validateSelectedFile,
} from "@/shared/lib/media/fileValidation";

describe("fileValidation", () => {
  it("describes upload rule with type and size", () => {
    expect(describeFileValidationRule(avatarUploadRule)).toBe("PNG / JPG / JPEG / WEBP，最大 5 MB");
  });

  it("rejects oversized file", () => {
    const file = new File(["video"], "demo.mp4", { type: "video/mp4" });
    Object.defineProperty(file, "size", { value: 201 * 1024 * 1024 });

    expect(() => validateSelectedFile(file, contentVideoUploadRule)).toThrow(
      "视频文件过大，请选择不超过 200 MB 的文件。",
    );
  });
});
