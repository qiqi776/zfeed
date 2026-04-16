import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { createMemoryRouter, RouterProvider } from "react-router-dom";
import { vi } from "vitest";

import { AppProviders } from "@/app/providers/AppProviders";
import { RegisterPage } from "@/features/auth/pages/RegisterPage";

const registerMock = vi.fn();
const uploadAvatarMock = vi.fn();

vi.mock("@/features/auth/api/auth.api", async () => {
  const actual = await vi.importActual<typeof import("@/features/auth/api/auth.api")>(
    "@/features/auth/api/auth.api",
  );

  return {
    ...actual,
    register: (...args: unknown[]) => registerMock(...args),
  };
});

vi.mock("@/features/user/api/user.api", () => ({
  uploadAvatar: (...args: unknown[]) => uploadAvatarMock(...args),
}));

describe("RegisterPage", () => {
  beforeEach(() => {
    registerMock.mockReset();
    uploadAvatarMock.mockReset();
    registerMock.mockResolvedValue({
      user_id: 9,
      token: "token",
      expired_at: 123,
    });
    uploadAvatarMock.mockResolvedValue({
      url: "https://cdn.example.com/avatar.png",
      object_key: "avatar/1.png",
      mime: "image/png",
      size: 1,
    });
  });

  it("opens avatar picker from a keyboard-focusable button", async () => {
    const router = createMemoryRouter([{ path: "/register", element: <RegisterPage /> }], {
      initialEntries: ["/register"],
    });
    const inputClickSpy = vi.spyOn(HTMLInputElement.prototype, "click");

    render(
      <AppProviders>
        <RouterProvider router={router} />
      </AppProviders>,
    );

    fireEvent.click(screen.getByRole("button", { name: "上传头像" }));

    expect(inputClickSpy).toHaveBeenCalledTimes(1);
    inputClickSpy.mockRestore();
  });

  it("submits the completed register payload and normalizes mobile input", async () => {
    const router = createMemoryRouter(
      [
        { path: "/register", element: <RegisterPage /> },
        { path: "/", element: <div>home</div> },
      ],
      {
        initialEntries: ["/register"],
      },
    );

    render(
      <AppProviders>
        <RouterProvider router={router} />
      </AppProviders>,
    );

    fireEvent.change(screen.getByLabelText(/手机号/), {
      target: { value: "13800000000" },
    });
    fireEvent.change(screen.getByLabelText(/密码/), {
      target: { value: "123456" },
    });
    fireEvent.change(screen.getByLabelText(/昵称/), {
      target: { value: "晨风" },
    });
    fireEvent.change(screen.getByLabelText(/邮箱/), {
      target: { value: "chenfeng@example.com" },
    });
    fireEvent.change(screen.getByLabelText(/个人简介/), {
      target: { value: "分享日常与成长记录" },
    });
    fireEvent.change(screen.getByLabelText(/生日/), {
      target: { value: "2000-05-06" },
    });
    fireEvent.click(screen.getByRole("button", { name: /晨雾暖阳/ }));
    fireEvent.click(screen.getByRole("button", { name: "女" }));
    fireEvent.click(screen.getByRole("button", { name: "注册并进入社区" }));

    await waitFor(() => expect(registerMock).toHaveBeenCalledTimes(1));
    expect(registerMock).toHaveBeenCalledWith(
      {
        mobile: "+8613800000000",
        password: "123456",
        nickname: "晨风",
        avatar: "https://dummyimage.com/320x320/ffe5dc/0b1220.png&text=W",
        bio: "分享日常与成长记录",
        gender: 2,
        email: "chenfeng@example.com",
        birthday: Math.floor(Date.UTC(2000, 4, 6) / 1000),
      },
      expect.anything(),
    );
  });

  it("uploads avatar file and reuses returned url", async () => {
    const router = createMemoryRouter([{ path: "/register", element: <RegisterPage /> }], {
      initialEntries: ["/register"],
    });

    render(
      <AppProviders>
        <RouterProvider router={router} />
      </AppProviders>,
    );

    const file = new File(["avatar"], "avatar.png", { type: "image/png" });
    fireEvent.change(screen.getByLabelText(/上传头像文件/, { selector: "input" }), {
      target: { files: [file] },
    });

    await waitFor(() => expect(uploadAvatarMock.mock.calls[0]?.[0]).toBe(file));
    expect(screen.getByLabelText(/自定义头像地址/)).toHaveValue(
      "https://cdn.example.com/avatar.png",
    );
  });

  it("blocks submit when email is invalid", async () => {
    const router = createMemoryRouter([{ path: "/register", element: <RegisterPage /> }], {
      initialEntries: ["/register"],
    });

    render(
      <AppProviders>
        <RouterProvider router={router} />
      </AppProviders>,
    );

    fireEvent.change(screen.getByLabelText(/手机号/), {
      target: { value: "13800000000" },
    });
    fireEvent.change(screen.getByLabelText(/密码/), {
      target: { value: "123456" },
    });
    fireEvent.change(screen.getByLabelText(/昵称/), {
      target: { value: "晨风" },
    });
    fireEvent.change(screen.getByLabelText(/邮箱/), {
      target: { value: "bad-email" },
    });
    fireEvent.change(screen.getByLabelText(/生日/), {
      target: { value: "2000-05-06" },
    });

    fireEvent.click(screen.getByRole("button", { name: "注册并进入社区" }));

    expect(screen.getByText("请输入有效邮箱地址，例如 name@example.com。")).toBeInTheDocument();
    expect(registerMock).not.toHaveBeenCalled();
  });
});
