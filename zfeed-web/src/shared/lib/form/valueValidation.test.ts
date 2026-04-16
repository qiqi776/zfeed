import { isValidEmail, isValidHttpUrl } from "@/shared/lib/form/valueValidation";

describe("valueValidation", () => {
  it("accepts http and https urls", () => {
    expect(isValidHttpUrl("https://example.com/image.png")).toBe(true);
    expect(isValidHttpUrl("http://example.com/video.mp4")).toBe(true);
    expect(isValidHttpUrl("ftp://example.com/file")).toBe(false);
  });

  it("validates email shape", () => {
    expect(isValidEmail("demo@example.com")).toBe(true);
    expect(isValidEmail(" bad@example.com ")).toBe(true);
    expect(isValidEmail("missing-at-symbol")).toBe(false);
  });
});
