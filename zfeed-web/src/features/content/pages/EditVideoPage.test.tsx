import { fireEvent, render, screen } from "@testing-library/react";
import { createMemoryRouter, RouterProvider } from "react-router-dom";
import { vi } from "vitest";

import { AppProviders } from "@/app/providers/AppProviders";
import { useSessionStore } from "@/entities/session/model/session.store";
import { EditVideoPage } from "@/features/content/pages/EditVideoPage";

const getContentDetailMock = vi.fn();
const editVideoMock = vi.fn();

vi.mock("@/features/content/api/content.api", () => ({
  getContentDetail: (...args: unknown[]) => getContentDetailMock(...args),
  editVideo: (...args: unknown[]) => editVideoMock(...args),
}));

vi.mock("@/features/content/lib/upload", () => ({
  uploadContentAsset: vi.fn(),
}));

describe("EditVideoPage", () => {
  beforeEach(() => {
    getContentDetailMock.mockReset();
    editVideoMock.mockReset();

    useSessionStore.setState({
      token: "token",
      expiredAt: 1,
      user: { userId: 7, nickname: "我", avatar: "" },
    });

    getContentDetailMock.mockResolvedValue({
      detail: {
        content_id: 101,
        content_type: 20,
        author_id: 7,
        author_name: "我",
        author_avatar: "",
        title: "旧视频标题",
        description: "旧视频描述",
        cover_url: "https://example.com/old-cover.jpg",
        article_content: "",
        video_url: "https://example.com/old-video.mp4",
        video_duration: 120,
        published_at: 1,
        like_count: 1,
        favorite_count: 1,
        comment_count: 1,
        is_liked: false,
        is_favorited: false,
        is_following_author: false,
      },
    });

    editVideoMock.mockResolvedValue({ content_id: 101 });
  });

  it("opens refreshed upload pickers from keyboard-focusable buttons", async () => {
    const inputClickSpy = vi.spyOn(HTMLInputElement.prototype, "click");
    const router = createMemoryRouter(
      [
        { path: "/studio/video/:contentId/edit", element: <EditVideoPage /> },
        { path: "/content/:contentId", element: <div>detail</div> },
      ],
      {
        initialEntries: ["/studio/video/101/edit"],
      },
    );

    render(
      <AppProviders>
        <RouterProvider router={router} />
      </AppProviders>,
    );

    await screen.findByDisplayValue("旧视频标题");

    fireEvent.click(screen.getByRole("button", { name: "上传视频封面" }));
    fireEvent.click(screen.getByRole("button", { name: "上传视频文件" }));

    expect(inputClickSpy).toHaveBeenCalledTimes(2);
    inputClickSpy.mockRestore();
  });
});
