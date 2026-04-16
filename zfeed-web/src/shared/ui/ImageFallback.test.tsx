import { fireEvent, render, screen } from "@testing-library/react";
import { vi } from "vitest";

import { ImageFallback } from "@/shared/ui/ImageFallback";

describe("ImageFallback", () => {
  it("shows fallback when image loading fails", () => {
    render(
      <ImageFallback
        src="https://example.com/broken.png"
        alt="测试封面"
        containerClassName="h-20 w-20"
        imageClassName="h-full w-full object-cover"
      />,
    );

    fireEvent.error(screen.getByAltText("测试封面"));

    expect(screen.getByLabelText("测试封面 占位图")).toBeInTheDocument();
    expect(screen.getByText("暂无封面")).toBeInTheDocument();
  });

  it("notifies callers when image load state changes", () => {
    const onErrorChange = vi.fn();

    render(
      <ImageFallback
        src="https://example.com/broken.png"
        alt="测试封面"
        onErrorChange={onErrorChange}
        containerClassName="h-20 w-20"
        imageClassName="h-full w-full object-cover"
      />,
    );

    expect(onErrorChange).toHaveBeenLastCalledWith(false);

    fireEvent.error(screen.getByAltText("测试封面"));

    expect(onErrorChange).toHaveBeenLastCalledWith(true);
  });
});
