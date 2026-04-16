import { render, screen } from "@testing-library/react";

import { PagedQueryFeedback } from "@/shared/ui/PagedQueryFeedback";

describe("PagedQueryFeedback", () => {
  it("renders nothing when there are no items", () => {
    const { container } = render(
      <PagedQueryFeedback
        hasItems={false}
        refreshingTitle="刷新中"
        refreshingDescription="正在同步"
        fetchingNextPageTitle="加载更多"
        fetchingNextPageDescription="正在拼接"
      />,
    );

    expect(container).toBeEmptyDOMElement();
  });

  it("prefers next-page loading feedback when appending items", () => {
    render(
      <PagedQueryFeedback
        hasItems
        isRefreshing
        isFetchingNextPage
        refreshingTitle="刷新中"
        refreshingDescription="正在同步"
        fetchingNextPageTitle="加载更多"
        fetchingNextPageDescription="正在拼接"
      />,
    );

    expect(screen.getByText("加载更多")).toBeInTheDocument();
    expect(screen.queryByText("刷新中")).not.toBeInTheDocument();
  });

  it("renders refresh feedback when background sync is running", () => {
    render(
      <PagedQueryFeedback
        hasItems
        isRefreshing
        refreshingTitle="刷新中"
        refreshingDescription="正在同步"
        fetchingNextPageTitle="加载更多"
        fetchingNextPageDescription="正在拼接"
      />,
    );

    expect(screen.getByText("刷新中")).toBeInTheDocument();
    expect(screen.getByText("正在同步")).toBeInTheDocument();
  });
});
