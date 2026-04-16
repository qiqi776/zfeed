import { act, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { createMemoryRouter, RouterProvider } from "react-router-dom";
import { vi } from "vitest";

import { AppLayout } from "@/app/layout/AppLayout";
import { AppProviders } from "@/app/providers/AppProviders";
import { useSessionStore } from "@/entities/session/model/session.store";

const logoutMock = vi.fn();

vi.mock("@/features/auth/api/auth.api", () => ({
  logout: (...args: unknown[]) => logoutMock(...args),
}));

describe("AppLayout", () => {
  beforeEach(() => {
    Object.defineProperty(window, "scrollY", {
      configurable: true,
      writable: true,
      value: 0,
    });
    window.sessionStorage.clear();
    window.scrollTo = vi.fn();

    useSessionStore.setState({
      token: "token",
      expiredAt: 1,
      user: { userId: 7, nickname: "测试用户", avatar: "" },
    });
    logoutMock.mockResolvedValue({});
  });

  it("renders primary shell and clears session on logout", async () => {
    const router = createMemoryRouter(
      [
        {
          path: "/",
          element: <AppLayout />,
          children: [{ index: true, element: <div>首页内容</div> }],
        },
        { path: "/login", element: <div>登录页</div> },
      ],
      {
        initialEntries: ["/"],
      },
    );

    render(
      <AppProviders>
        <RouterProvider router={router} />
      </AppProviders>,
    );

    expect(screen.getByText("ZFeed Web")).toBeInTheDocument();
    expect(screen.getByText("首页内容")).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "跳到主要内容" })).toHaveAttribute("href", "#app-main");
    expect(screen.getByRole("main")).toHaveAttribute("id", "app-main");
    expect(screen.getByRole("button", { name: "退出" })).toBeInTheDocument();
    expect(screen.getByText("测试用户")).toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: "退出" }));

    await waitFor(() => {
      expect(screen.getByText("登录页")).toBeInTheDocument();
    });
    expect(useSessionStore.getState().token).toBeNull();
    expect(logoutMock).toHaveBeenCalledTimes(1);
  });

  it("restores list scroll position when navigating back from detail", async () => {
    const router = createMemoryRouter(
      [
        {
          path: "/",
          element: <AppLayout />,
          children: [
            { index: true, element: <div>推荐列表</div> },
            { path: "content/:contentId", element: <div>内容详情</div> },
          ],
        },
      ],
      {
        initialEntries: ["/"],
      },
    );

    render(
      <AppProviders>
        <RouterProvider router={router} />
      </AppProviders>,
    );

    await screen.findByText("推荐列表");

    window.scrollTo = vi.fn();
    window.scrollY = 640;

    await act(async () => {
      await router.navigate("/content/101");
    });
    await screen.findByText("内容详情");

    expect(window.scrollTo).toHaveBeenLastCalledWith(0, 0);

    window.scrollTo = vi.fn();
    await act(async () => {
      await router.navigate(-1);
    });

    await waitFor(() => {
      expect(screen.getByText("推荐列表")).toBeInTheDocument();
    });
    expect(window.scrollTo).toHaveBeenLastCalledWith(0, 640);
  });
});
