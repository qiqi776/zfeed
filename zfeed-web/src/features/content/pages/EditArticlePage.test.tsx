import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { createMemoryRouter, RouterProvider } from "react-router-dom";
import { vi } from "vitest";

import { AppProviders } from "@/app/providers/AppProviders";
import { useSessionStore } from "@/entities/session/model/session.store";
import { EditArticlePage } from "@/features/content/pages/EditArticlePage";

const getContentDetailMock = vi.fn();
const editArticleMock = vi.fn();

vi.mock("@/features/content/api/content.api", () => ({
  getContentDetail: (...args: unknown[]) => getContentDetailMock(...args),
  editArticle: (...args: unknown[]) => editArticleMock(...args),
}));

vi.mock("@/features/content/lib/upload", () => ({
  uploadContentAsset: vi.fn(),
}));

describe("EditArticlePage", () => {
  beforeEach(() => {
    getContentDetailMock.mockReset();
    editArticleMock.mockReset();

    useSessionStore.setState({
      token: "token",
      expiredAt: 1,
      user: { userId: 7, nickname: "我", avatar: "" },
    });

    getContentDetailMock.mockResolvedValue({
      detail: {
        content_id: 101,
        content_type: 10,
        author_id: 7,
        author_name: "我",
        author_avatar: "",
        title: "旧标题",
        description: "旧描述",
        cover_url: "https://example.com/old-cover.jpg",
        article_content: "旧正文",
        video_url: "",
        video_duration: 0,
        published_at: 1,
        like_count: 1,
        favorite_count: 1,
        comment_count: 1,
        is_liked: false,
        is_favorited: false,
        is_following_author: false,
      },
    });

    editArticleMock.mockResolvedValue({ content_id: 101 });
  });

  it("opens the refreshed cover picker from a keyboard-focusable button", async () => {
    const inputClickSpy = vi.spyOn(HTMLInputElement.prototype, "click");
    const router = createMemoryRouter(
      [
        { path: "/studio/article/:contentId/edit", element: <EditArticlePage /> },
        { path: "/content/:contentId", element: <div>detail</div> },
      ],
      {
        initialEntries: ["/studio/article/101/edit"],
      },
    );

    render(
      <AppProviders>
        <RouterProvider router={router} />
      </AppProviders>,
    );

    await screen.findByDisplayValue("旧标题");

    fireEvent.click(screen.getByRole("button", { name: "上传新封面" }));

    expect(inputClickSpy).toHaveBeenCalledTimes(1);
    inputClickSpy.mockRestore();
  });

  it("submits changed article fields", async () => {
    const router = createMemoryRouter(
      [
        { path: "/studio/article/:contentId/edit", element: <EditArticlePage /> },
        { path: "/content/:contentId", element: <div>detail</div> },
      ],
      {
        initialEntries: ["/studio/article/101/edit"],
      },
    );

    render(
      <AppProviders>
        <RouterProvider router={router} />
      </AppProviders>,
    );

    await screen.findByDisplayValue("旧标题");

    fireEvent.change(screen.getByLabelText(/标题/), { target: { value: "新标题" } });
    fireEvent.change(screen.getByLabelText(/正文/), { target: { value: "新正文" } });

    fireEvent.click(screen.getByRole("button", { name: "保存文章" }));

    await waitFor(() => {
      expect(editArticleMock).toHaveBeenCalledWith(101, {
        title: "新标题",
        content: "新正文",
      });
    });
  });

  it("blocks submit when cover url is invalid", async () => {
    const router = createMemoryRouter(
      [
        { path: "/studio/article/:contentId/edit", element: <EditArticlePage /> },
        { path: "/content/:contentId", element: <div>detail</div> },
      ],
      {
        initialEntries: ["/studio/article/101/edit"],
      },
    );

    render(
      <AppProviders>
        <RouterProvider router={router} />
      </AppProviders>,
    );

    await screen.findByDisplayValue("旧标题");

    fireEvent.change(screen.getByLabelText(/封面 URL/), {
      target: { value: "ftp://example.com/bad-cover.jpg" },
    });

    fireEvent.click(screen.getByRole("button", { name: "保存文章" }));

    expect(screen.getByRole("alert")).toHaveTextContent("封面 URL 无效");
    expect(editArticleMock).not.toHaveBeenCalled();
  });
});
