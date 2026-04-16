import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { createMemoryRouter, RouterProvider } from "react-router-dom";
import { vi } from "vitest";

import { AppProviders } from "@/app/providers/AppProviders";
import { useSessionStore } from "@/entities/session/model/session.store";
import { SettingsPage } from "@/features/user/pages/SettingsPage";

const getMeMock = vi.fn();
const updateProfileMock = vi.fn();
const uploadAvatarMock = vi.fn();

vi.mock("@/features/auth/api/auth.api", () => ({
  getMe: (...args: unknown[]) => getMeMock(...args),
}));

vi.mock("@/features/user/api/user.api", () => ({
  updateProfile: (...args: unknown[]) => updateProfileMock(...args),
  uploadAvatar: (...args: unknown[]) => uploadAvatarMock(...args),
}));

describe("SettingsPage", () => {
  beforeEach(() => {
    getMeMock.mockReset();
    updateProfileMock.mockReset();
    uploadAvatarMock.mockReset();

    useSessionStore.setState({
      token: "token",
      expiredAt: 10,
      user: { userId: 7, nickname: "旧昵称", avatar: "" },
    });

    getMeMock.mockResolvedValue({
      user_info: {
        user_id: 7,
        mobile: "+8613800000000",
        nickname: "旧昵称",
        avatar: "https://example.com/old.png",
        bio: "旧简介",
        gender: 1,
        status: 10,
        email: "old@example.com",
        birthday: 946684800,
      },
      followee_count: 1,
      follower_count: 2,
      like_received_count: 3,
      favorite_received_count: 4,
      content_count: 5,
    });

    updateProfileMock.mockResolvedValue({
      user_info: {
        user_id: 7,
        mobile: "+8613800000000",
        nickname: "新昵称",
        avatar: "https://example.com/new.png",
        bio: "新简介",
        gender: 2,
        status: 10,
        email: "new@example.com",
        birthday: 978307200,
      },
    });

    uploadAvatarMock.mockResolvedValue({
      url: "https://example.com/new.png",
      object_key: "avatar/1.png",
      mime: "image/png",
      size: 1,
    });
  });

  it("loads profile data and submits changed fields", async () => {
    const router = createMemoryRouter(
      [
        { path: "/me/settings", element: <SettingsPage /> },
        { path: "/me", element: <div>me</div> },
      ],
      {
        initialEntries: ["/me/settings"],
      },
    );

    render(
      <AppProviders>
        <RouterProvider router={router} />
      </AppProviders>,
    );

    await screen.findByDisplayValue("旧昵称");

    fireEvent.change(screen.getByLabelText(/昵称/), { target: { value: "新昵称" } });
    fireEvent.change(screen.getByLabelText(/简介/), { target: { value: "新简介" } });
    fireEvent.change(screen.getByLabelText(/邮箱/), { target: { value: "new@example.com" } });
    fireEvent.change(screen.getByLabelText(/生日/), { target: { value: "2001-01-01" } });

    fireEvent.click(screen.getByRole("button", { name: "保存资料" }));

    await waitFor(() => {
      expect(updateProfileMock.mock.calls[0]?.[0]).toEqual({
        nickname: "新昵称",
        bio: "新简介",
        email: "new@example.com",
        birthday: 978307200,
      });
    });
  });

  it("opens avatar picker from an actual button", async () => {
    const inputClickSpy = vi.spyOn(HTMLInputElement.prototype, "click");
    const router = createMemoryRouter(
      [
        { path: "/me/settings", element: <SettingsPage /> },
        { path: "/me", element: <div>me</div> },
      ],
      {
        initialEntries: ["/me/settings"],
      },
    );

    render(
      <AppProviders>
        <RouterProvider router={router} />
      </AppProviders>,
    );

    await screen.findByDisplayValue("旧昵称");

    fireEvent.click(screen.getByRole("button", { name: "上传头像" }));

    expect(inputClickSpy).toHaveBeenCalledTimes(1);
    inputClickSpy.mockRestore();
  });

  it("blocks submit when email is invalid", async () => {
    const router = createMemoryRouter(
      [
        { path: "/me/settings", element: <SettingsPage /> },
        { path: "/me", element: <div>me</div> },
      ],
      {
        initialEntries: ["/me/settings"],
      },
    );

    render(
      <AppProviders>
        <RouterProvider router={router} />
      </AppProviders>,
    );

    await screen.findByDisplayValue("旧昵称");

    fireEvent.change(screen.getByLabelText(/邮箱/), { target: { value: "bad-email" } });
    fireEvent.click(screen.getByRole("button", { name: "保存资料" }));

    expect(screen.getByText("请输入有效邮箱地址，例如 name@example.com。")).toBeInTheDocument();
    expect(updateProfileMock).not.toHaveBeenCalled();
  });
});
