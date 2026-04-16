import { render, screen, waitFor } from "@testing-library/react";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import { vi } from "vitest";

import { AppProviders } from "@/app/providers/AppProviders";
import { useSessionStore } from "@/entities/session/model/session.store";
import { UserProfilePage } from "@/features/user/pages/UserProfilePage";

const getUserProfileMock = vi.fn();
const getUserPublishMock = vi.fn();

vi.mock("@/features/user/api/user.api", () => ({
  getUserProfile: (...args: unknown[]) => getUserProfileMock(...args),
}));

vi.mock("@/features/feed/api/feed.api", () => ({
  getUserPublish: (...args: unknown[]) => getUserPublishMock(...args),
}));

vi.mock("@/features/interaction/api/interaction.api", () => ({
  followUser: vi.fn(),
  unfollowUser: vi.fn(),
}));

describe("UserProfilePage", () => {
  beforeEach(() => {
    getUserProfileMock.mockReset();
    getUserPublishMock.mockReset();

    useSessionStore.setState({
      token: "token",
      expiredAt: 1,
      user: { userId: 1, nickname: "访客", avatar: "" },
    });

    getUserProfileMock.mockResolvedValue({
      user_profile: {
        user_id: 9,
        nickname: "作者",
        avatar: "",
        bio: "公开简介",
        gender: 1,
      },
      counts: {
        followee_count: 2,
        follower_count: 3,
        like_received_count: 4,
        favorite_received_count: 5,
        content_count: 3,
      },
      viewer: {
        is_following: false,
      },
    });

    getUserPublishMock.mockResolvedValue({
      items: [
        {
          content_id: 101,
          content_type: 10,
          author_id: 9,
          author_name: "作者",
          author_avatar: "",
          title: "第一篇公开内容",
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

  it("clarifies aggregate counts versus loaded public items", async () => {
    render(
      <AppProviders>
        <MemoryRouter initialEntries={["/users/9"]}>
          <Routes>
            <Route path="/users/:userId" element={<UserProfilePage />} />
          </Routes>
        </MemoryRouter>
      </AppProviders>,
    );

    await screen.findByText("当前只展示公开资料和公开内容");

    await waitFor(() => {
      expect(
        screen.getByText(
          (_, node) =>
            node?.textContent === "公开总数 3 · 已加载 1 条" &&
            node.tagName.toLowerCase() === "span",
        ),
      ).toBeInTheDocument();
    });
    expect(screen.getByText("公开总量和当前列表是两套口径")).toBeInTheDocument();
    expect(
      screen.getByText("这里不会暴露草稿、私密内容或未开放的后台状态；当前列表 badge 里的已加载只代表前端页数。"),
    ).toBeInTheDocument();
  });
});
