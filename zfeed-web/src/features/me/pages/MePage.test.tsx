import { render, screen } from "@testing-library/react";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import { vi } from "vitest";

import { AppProviders } from "@/app/providers/AppProviders";
import { useSessionStore } from "@/entities/session/model/session.store";
import { MePage } from "@/features/me/pages/MePage";

const getMeMock = vi.fn();

vi.mock("@/features/auth/api/auth.api", () => ({
  getMe: (...args: unknown[]) => getMeMock(...args),
}));

describe("MePage", () => {
  beforeEach(() => {
    getMeMock.mockReset();

    useSessionStore.setState({
      token: "token",
      expiredAt: 1,
      user: { userId: 7, nickname: "我", avatar: "" },
    });

    getMeMock.mockResolvedValue({
      user_info: {
        user_id: 7,
        mobile: "+8613800000000",
        nickname: "我的名字",
        avatar: "",
        bio: "这是我的简介",
        gender: 0,
        status: 10,
        email: "me@example.com",
        birthday: 0,
      },
      followee_count: 11,
      follower_count: 12,
      like_received_count: 13,
      favorite_received_count: 14,
      content_count: 15,
    });
  });

  it("shows aggregate boundary guidance for the personal space", async () => {
    render(
      <AppProviders>
        <MemoryRouter initialEntries={["/me"]}>
          <Routes>
            <Route path="/me" element={<MePage />} />
          </Routes>
        </MemoryRouter>
      </AppProviders>,
    );

    await screen.findByText("我的名字");

    expect(screen.getByText("这些数字是聚合口径")).toBeInTheDocument();
    expect(screen.getByText("聚合计数优先")).toBeInTheDocument();
    expect(screen.getByText("Studio / 公开主页只看公开内容")).toBeInTheDocument();
  });
});
