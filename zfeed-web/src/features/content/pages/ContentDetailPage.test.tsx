import { fireEvent, render, screen } from "@testing-library/react";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import { vi } from "vitest";

import { AppProviders } from "@/app/providers/AppProviders";
import { useSessionStore } from "@/entities/session/model/session.store";
import { ContentDetailPage } from "@/features/content/pages/ContentDetailPage";

const getContentDetailMock = vi.fn();
const queryCommentListMock = vi.fn();
const queryReplyCommentListMock = vi.fn();

vi.mock("@/features/content/api/content.api", () => ({
  getContentDetail: (...args: unknown[]) => getContentDetailMock(...args),
}));

vi.mock("@/features/interaction/api/interaction.api", () => ({
  likeContent: vi.fn(),
  unlikeContent: vi.fn(),
  favoriteContent: vi.fn(),
  removeFavorite: vi.fn(),
  followUser: vi.fn(),
  unfollowUser: vi.fn(),
  commentContent: vi.fn(),
  deleteComment: vi.fn(),
  queryCommentList: (...args: unknown[]) => queryCommentListMock(...args),
  queryReplyCommentList: (...args: unknown[]) => queryReplyCommentListMock(...args),
}));

describe("ContentDetailPage", () => {
  beforeEach(() => {
    useSessionStore.setState({
      token: "token",
      expiredAt: 1,
      user: { userId: 1, nickname: "我", avatar: "" },
    });

    getContentDetailMock.mockResolvedValue({
      detail: {
        content_id: 101,
        content_type: 10,
        author_id: 9,
        author_name: "作者",
        author_avatar: "",
        title: "测试内容",
        description: "测试描述",
        cover_url: "",
        article_content: "正文内容",
        video_url: "",
        video_duration: 0,
        published_at: 1,
        like_count: 2,
        favorite_count: 1,
        comment_count: 1,
        is_liked: false,
        is_favorited: false,
        is_following_author: false,
      },
    });

    queryCommentListMock.mockResolvedValue({
      comments: [
        {
          comment_id: 201,
          content_id: 101,
          user_id: 7,
          reply_to_user_id: 0,
          parent_id: 0,
          root_id: 0,
          comment: "第一条评论",
          created_at: 1,
          status: 1,
          user_name: "评论者",
          user_avatar: "",
          reply_count: 0,
        },
      ],
      next_cursor: 0,
      has_more: false,
    });

    queryReplyCommentListMock.mockResolvedValue({
      comments: [],
      next_cursor: 0,
      has_more: false,
    });
  });

  it("shows clearer reply context after clicking reply", async () => {
    render(
      <AppProviders>
        <MemoryRouter initialEntries={["/content/101"]}>
          <Routes>
            <Route path="/content/:contentId" element={<ContentDetailPage />} />
          </Routes>
        </MemoryRouter>
      </AppProviders>,
    );

    await screen.findByText("第一条评论");

    fireEvent.click(screen.getByRole("button", { name: "回复" }));

    expect(screen.getByText("这条回复会进入 @评论者 这条评论所在的楼层。")).toBeInTheDocument();
    expect(screen.getByText("发送回复")).toBeInTheDocument();
  });
});
