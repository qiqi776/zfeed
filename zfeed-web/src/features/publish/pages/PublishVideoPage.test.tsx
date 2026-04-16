import { fireEvent, render, screen } from "@testing-library/react";
import { createMemoryRouter, RouterProvider } from "react-router-dom";
import { vi } from "vitest";

import { AppProviders } from "@/app/providers/AppProviders";
import { PublishVideoPage } from "@/features/publish/pages/PublishVideoPage";

const publishVideoMock = vi.fn();

vi.mock("@/features/content/api/content.api", () => ({
  publishVideo: (...args: unknown[]) => publishVideoMock(...args),
}));

function renderPage() {
  const router = createMemoryRouter(
    [
      { path: "/publish/video", element: <PublishVideoPage /> },
      { path: "/publish", element: <div>publish</div> },
      { path: "/studio", element: <div>studio</div> },
      { path: "/content/:contentId", element: <div>detail</div> },
    ],
    {
      initialEntries: ["/publish/video"],
    },
  );

  return render(
    <AppProviders>
      <RouterProvider router={router} />
    </AppProviders>,
  );
}

describe("PublishVideoPage", () => {
  it("opens file pickers from keyboard-focusable upload buttons", () => {
    const inputClickSpy = vi.spyOn(HTMLInputElement.prototype, "click");

    renderPage();

    fireEvent.click(screen.getByRole("button", { name: "上传视频封面" }));
    fireEvent.click(screen.getByRole("button", { name: "上传视频文件" }));

    expect(inputClickSpy).toHaveBeenCalledTimes(2);
    inputClickSpy.mockRestore();
  });

  it("blocks submit when duration is invalid", () => {
    renderPage();

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
