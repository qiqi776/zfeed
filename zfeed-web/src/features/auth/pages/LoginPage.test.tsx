import { render, screen } from "@testing-library/react";
import { createMemoryRouter, RouterProvider } from "react-router-dom";

import { AppProviders } from "@/app/providers/AppProviders";
import { LoginPage } from "@/features/auth/pages/LoginPage";

describe("LoginPage", () => {
  it("renders login button", () => {
    const router = createMemoryRouter([{ path: "/", element: <LoginPage /> }], {
      initialEntries: ["/"],
    });

    render(
      <AppProviders>
        <RouterProvider router={router} />
      </AppProviders>,
    );

    expect(screen.getByRole("button", { name: "登录" })).toBeInTheDocument();
  });
});
