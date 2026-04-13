import { fireEvent, render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { vi } from "vitest";

import { AppProviders } from "@/app/providers/AppProviders";
import { PublishVideoPage } from "@/features/publish/pages/PublishVideoPage";

const publishVideoMock = vi.fn();

vi.mock("@/features/content/api/content.api", () => ({
  publishVideo: (...args: unknown[]) => publishVideoMock(...args),
}));

describe("PublishVideoPage", () => {
  it("blocks submit when duration is invalid", () => {
    render(
      <AppProviders>
        <MemoryRouter>
          <PublishVideoPage />
        </MemoryRouter>
      </AppProviders>,
    );

    fireEvent.change(screen.getByLabelText("标题"), { target: { value: "测试视频" } });
    fireEvent.change(screen.getByRole("textbox", { name: /封面 URL/ }), {
      target: { value: "https://example.com/cover.jpg" },
    });
    fireEvent.change(screen.getByRole("textbox", { name: /视频 URL/ }), {
      target: { value: "https://example.com/video.mp4" },
    });
    fireEvent.change(screen.getByRole("textbox", { name: /时长（秒）/ }), {
      target: { value: "abc" },
    });

    fireEvent.click(screen.getByRole("button", { name: "发布视频" }));

    expect(screen.getByRole("alert")).toHaveTextContent("视频时长无效");
    expect(screen.getByRole("alert")).toHaveTextContent("请输入有效的秒数。");
    expect(publishVideoMock).not.toHaveBeenCalled();
  });
});
