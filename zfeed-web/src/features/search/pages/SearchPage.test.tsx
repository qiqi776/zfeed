import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import { vi } from "vitest";

import { AppProviders } from "@/app/providers/AppProviders";
import { useSessionStore } from "@/entities/session/model/session.store";
import { SearchPage } from "@/features/search/pages/SearchPage";

const searchUsersMock = vi.fn();
const searchContentsMock = vi.fn();

vi.mock("@/features/search/api/search.api", () => ({
  searchUsers: (...args: unknown[]) => searchUsersMock(...args),
  searchContents: (...args: unknown[]) => searchContentsMock(...args),
}));

vi.mock("@/features/interaction/api/interaction.api", () => ({
  followUser: vi.fn(),
  unfollowUser: vi.fn(),
}));

describe("SearchPage", () => {
  beforeEach(() => {
    searchUsersMock.mockReset();
    searchContentsMock.mockReset();

    useSessionStore.setState({
      token: "token",
      expiredAt: 1,
      user: { userId: 1, nickname: "我", avatar: "" },
    });
  });

  it("loads user search results from query string", async () => {
    searchUsersMock.mockResolvedValue({
      items: [
        {
          user_id: 8,
          nickname: "Alice",
          avatar: "",
          bio: "design notes",
          is_following: false,
        },
      ],
      next_cursor: 0,
      has_more: false,
    });

    render(
      <AppProviders>
        <MemoryRouter initialEntries={["/search?tab=users&q=Alice"]}>
          <Routes>
            <Route path="/search" element={<SearchPage />} />
          </Routes>
        </MemoryRouter>
      </AppProviders>,
    );

    expect(await screen.findByText("Alice")).toBeInTheDocument();
    expect(screen.getByText("design notes")).toBeInTheDocument();
    expect(screen.getByLabelText("搜索关键词")).toHaveValue("Alice");
    expect(screen.getByRole("tablist", { name: "搜索类型" })).toBeInTheDocument();
    expect(screen.getByRole("tab", { name: "搜索用户" })).toHaveAttribute("aria-selected", "true");
    expect(screen.getByRole("tabpanel", { name: "搜索用户" })).toBeInTheDocument();
  });

  it("shows paged loading feedback when fetching more search results", async () => {
    let resolveNextPage: ((value: unknown) => void) | undefined;
    const nextPagePromise = new Promise((resolve) => {
      resolveNextPage = resolve;
    });

    searchUsersMock
      .mockResolvedValueOnce({
        items: [
          {
            user_id: 8,
            nickname: "Alice",
            avatar: "",
            bio: "design notes",
            is_following: false,
          },
        ],
        next_cursor: 9,
        has_more: true,
      })
      .mockReturnValueOnce(nextPagePromise);

    render(
      <AppProviders>
        <MemoryRouter initialEntries={["/search?tab=users&q=Alice"]}>
          <Routes>
            <Route path="/search" element={<SearchPage />} />
          </Routes>
        </MemoryRouter>
      </AppProviders>,
    );

    await screen.findByText("Alice");

    fireEvent.click(screen.getByRole("button", { name: "加载更多" }));

    expect(await screen.findByText("正在继续加载搜索结果")).toBeInTheDocument();
    expect(screen.getByText("下一页搜索结果正在拼接到当前列表中。")).toBeInTheDocument();

    resolveNextPage?.({
      items: [
        {
          user_id: 9,
          nickname: "Alice B",
          avatar: "",
          bio: "more notes",
          is_following: false,
        },
      ],
      next_cursor: 0,
      has_more: false,
    });

    await waitFor(() => {
      expect(screen.queryByText("正在继续加载搜索结果")).not.toBeInTheDocument();
    });
  });

  it("supports switching search tabs with keyboard arrows", async () => {
    searchContentsMock.mockResolvedValue({
      items: [],
      next_cursor: 0,
      has_more: false,
    });
    searchUsersMock.mockResolvedValue({
      items: [],
      next_cursor: 0,
      has_more: false,
    });

    render(
      <AppProviders>
        <MemoryRouter initialEntries={["/search?tab=contents&q=Alice"]}>
          <Routes>
            <Route path="/search" element={<SearchPage />} />
          </Routes>
        </MemoryRouter>
      </AppProviders>,
    );

    expect(await screen.findByRole("tab", { name: "搜索内容" })).toHaveAttribute(
      "aria-selected",
      "true",
    );

    fireEvent.keyDown(screen.getByRole("tab", { name: "搜索内容" }), { key: "ArrowRight" });

    await waitFor(() => {
      expect(screen.getByRole("tab", { name: "搜索用户" })).toHaveAttribute(
        "aria-selected",
        "true",
      );
    });
    expect(screen.getByRole("tabpanel", { name: "搜索用户" })).toBeInTheDocument();
  });
});
