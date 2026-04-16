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
    getContentDetailMock.mockReset();
    queryCommentListMock.mockReset();
    queryReplyCommentListMock.mockReset();

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

    expect(screen.getByLabelText("发表评论")).toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: "回复" }));

    expect(screen.getByText("这条回复会进入 @评论者 这条评论所在的楼层。")).toBeInTheDocument();
    expect(screen.getByLabelText("回复评论")).toBeInTheDocument();
    expect(screen.getByText("发送回复")).toBeInTheDocument();
  });

  it("announces reply thread expand state with aria attributes", async () => {
    queryCommentListMock.mockResolvedValueOnce({
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
          reply_count: 2,
        },
      ],
      next_cursor: 0,
      has_more: false,
    });

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

    const toggleButton = screen.getByRole("button", { name: "查看 2 条回复" });
    expect(toggleButton).toHaveAttribute("aria-expanded", "false");
    expect(toggleButton).toHaveAttribute("aria-controls", "comment-thread-replies-201");

    fireEvent.click(toggleButton);

    expect(screen.getByRole("button", { name: "收起回复" })).toHaveAttribute(
      "aria-expanded",
      "true",
    );
  });

  it("shows explicit cover fallback status when cover is unavailable", async () => {
    render(
      <AppProviders>
        <MemoryRouter initialEntries={["/content/101"]}>
          <Routes>
            <Route path="/content/:contentId" element={<ContentDetailPage />} />
          </Routes>
        </MemoryRouter>
      </AppProviders>,
    );

    await screen.findByText("测试内容");

    expect(screen.getByText("当前没有可用封面，已用占位图兜底")).toBeInTheDocument();
    expect(screen.getByText("当前内容没有可用封面地址，已降级为占位图。")).toBeInTheDocument();
  });

  it("falls back to cover preview when video playback fails", async () => {
    getContentDetailMock.mockResolvedValueOnce({
      detail: {
        content_id: 101,
        content_type: 20,
        author_id: 9,
        author_name: "作者",
        author_avatar: "",
        title: "视频内容",
        description: "测试视频描述",
        cover_url: "https://example.com/cover.jpg",
        article_content: "",
        video_url: "https://example.com/video.mp4",
        video_duration: 65,
        published_at: 1,
        like_count: 2,
        favorite_count: 1,
        comment_count: 1,
        is_liked: false,
        is_favorited: false,
        is_following_author: false,
      },
    });

    const { container } = render(
      <AppProviders>
        <MemoryRouter initialEntries={["/content/101"]}>
          <Routes>
            <Route path="/content/:contentId" element={<ContentDetailPage />} />
          </Routes>
        </MemoryRouter>
      </AppProviders>,
    );

    await screen.findByText("视频内容");

    const video = container.querySelector("video");
    expect(video).not.toBeNull();

    fireEvent.error(video as HTMLVideoElement);

    expect(screen.getByText("视频加载失败，已切到封面预览")).toBeInTheDocument();
    expect(
      screen.getByText("当前视频地址暂时不可用，你仍然可以先查看封面、简介和评论区，稍后再试。"),
    ).toBeInTheDocument();
    expect(screen.getByText("当前先展示封面预览，你仍然可以继续阅读简介和评论。")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "重试播放" })).toBeInTheDocument();
    expect(screen.getByText("当前视频地址暂时不可用，已回退到封面预览。")).toBeInTheDocument();
  });
});
