import { fireEvent, render, screen } from "@testing-library/react";
import { createMemoryRouter, RouterProvider } from "react-router-dom";
import { vi } from "vitest";

import { AppProviders } from "@/app/providers/AppProviders";
import { PublishArticlePage } from "@/features/publish/pages/PublishArticlePage";

const ARTICLE_DRAFT_STORAGE_KEY = "zfeed-web-publish-article-draft";
const publishArticleMock = vi.fn();

vi.mock("@/features/content/api/content.api", () => ({
  publishArticle: (...args: unknown[]) => publishArticleMock(...args),
}));

function renderPage() {
  const router = createMemoryRouter(
    [
      { path: "/publish/article", element: <PublishArticlePage /> },
      { path: "/publish", element: <div>publish</div> },
      { path: "/studio", element: <div>studio</div> },
      { path: "/content/:contentId", element: <div>detail</div> },
    ],
    {
      initialEntries: ["/publish/article"],
    },
  );

  return render(
    <AppProviders>
      <RouterProvider router={router} />
    </AppProviders>,
  );
}

describe("PublishArticlePage", () => {
  it("opens the cover file picker from a keyboard-focusable button", () => {
    const inputClickSpy = vi.spyOn(HTMLInputElement.prototype, "click");

    renderPage();

    fireEvent.click(screen.getByRole("button", { name: "上传文章封面" }));

    expect(inputClickSpy).toHaveBeenCalledTimes(1);
    inputClickSpy.mockRestore();
  });

  it("restores local draft and allows clearing it", () => {
    window.localStorage.setItem(
      ARTICLE_DRAFT_STORAGE_KEY,
      JSON.stringify({
        value: {
          title: "草稿标题",
          description: "草稿描述",
          cover: "https://example.com/cover.jpg",
          content: "草稿正文",
          visibility: "10",
        },
      }),
    );

    renderPage();

    expect(screen.getByText("已恢复上次未发布的文章草稿")).toBeInTheDocument();
    expect(screen.getByLabelText("标题")).toHaveValue("草稿标题");
    expect(screen.getByRole("textbox", { name: /封面 URL/ })).toHaveValue(
      "https://example.com/cover.jpg",
    );
    expect(screen.getByLabelText("正文")).toHaveValue("草稿正文");

    fireEvent.click(screen.getByRole("button", { name: "清空草稿" }));

    expect(screen.getByLabelText("标题")).toHaveValue("");
    expect(screen.getByRole("textbox", { name: /封面 URL/ })).toHaveValue("");
    expect(screen.getByLabelText("正文")).toHaveValue("");
    expect(window.localStorage.getItem(ARTICLE_DRAFT_STORAGE_KEY)).toBeNull();
    expect(screen.getByRole("status")).toHaveTextContent("文章草稿已清空");
    expect(publishArticleMock).not.toHaveBeenCalled();
  });

  it("blocks submit when cover url is invalid", () => {
    renderPage();

    fireEvent.change(screen.getByLabelText("标题"), { target: { value: "测试文章" } });
    fireEvent.change(screen.getByRole("textbox", { name: /封面 URL/ }), {
      target: { value: "ftp://example.com/cover.jpg" },
    });
    fireEvent.change(screen.getByLabelText("正文"), { target: { value: "正文内容" } });

    fireEvent.click(screen.getByRole("button", { name: "发布文章" }));

    expect(screen.getByRole("alert")).toHaveTextContent("封面 URL 无效");
    expect(screen.getByRole("alert")).toHaveTextContent("请输入可访问的 http 或 https 封面地址。");
    expect(publishArticleMock).not.toHaveBeenCalled();
  });
});
