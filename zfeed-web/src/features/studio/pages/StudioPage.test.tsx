import { render, screen, waitFor } from "@testing-library/react";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import { vi } from "vitest";

import { AppProviders } from "@/app/providers/AppProviders";
import { useSessionStore } from "@/entities/session/model/session.store";
import { StudioPage } from "@/features/studio/pages/StudioPage";

const getMeMock = vi.fn();
const getUserPublishMock = vi.fn();
const deleteContentMock = vi.fn();

vi.mock("@/features/auth/api/auth.api", () => ({
  getMe: (...args: unknown[]) => getMeMock(...args),
}));

vi.mock("@/features/feed/api/feed.api", () => ({
  getUserPublish: (...args: unknown[]) => getUserPublishMock(...args),
}));

vi.mock("@/features/content/api/content.api", () => ({
  deleteContent: (...args: unknown[]) => deleteContentMock(...args),
}));

describe("StudioPage", () => {
  beforeEach(() => {
    getMeMock.mockReset();
    getUserPublishMock.mockReset();
    deleteContentMock.mockReset();

    useSessionStore.setState({
      token: "token",
      expiredAt: 1,
      user: { userId: 7, nickname: "我", avatar: "" },
    });

    getMeMock.mockResolvedValue({
      user_info: {
        user_id: 7,
        mobile: "+8613800000000",
        nickname: "我",
        avatar: "",
        bio: "",
        gender: 0,
        status: 10,
        email: "me@example.com",
        birthday: 0,
      },
      followee_count: 1,
      follower_count: 2,
      like_received_count: 3,
      favorite_received_count: 4,
      content_count: 4,
    });

    getUserPublishMock.mockResolvedValue({
      items: [
        {
          content_id: 101,
          content_type: 10,
          author_id: 7,
          author_name: "我",
          author_avatar: "",
          title: "公开文章",
          cover_url: "",
          published_at: 1,
          is_liked: false,
          like_count: 2,
        },
      ],
      next_cursor: "0",
      has_more: false,
    });
  });

  it("shows aggregate-versus-list guidance for the studio", async () => {
    render(
      <AppProviders>
        <MemoryRouter initialEntries={["/studio"]}>
          <Routes>
            <Route path="/studio" element={<StudioPage />} />
          </Routes>
        </MemoryRouter>
      </AppProviders>,
    );

    await screen.findByText("当前总量和当前页列表分开表达");

    await waitFor(() => {
      expect(
        screen.getByText(
          (_, node) =>
            node?.textContent === "公开总数 4 · 已加载 1 条" &&
            node.tagName.toLowerCase() === "span",
        ),
      ).toBeInTheDocument();
    });
    expect(screen.getByText("当前总量和当前页列表分开表达")).toBeInTheDocument();
    expect(screen.getByText("私密内容 / Studio Summary 还没接前端")).toBeInTheDocument();
  });
});
