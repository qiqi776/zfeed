import { InlineNotice } from "@/shared/ui/InlineNotice";

type PagedQueryFeedbackProps = {
  hasItems: boolean;
  isRefreshing?: boolean;
  isFetchingNextPage?: boolean;
  refreshingTitle: string;
  refreshingDescription: string;
  fetchingNextPageTitle: string;
  fetchingNextPageDescription: string;
};

export function PagedQueryFeedback({
  hasItems,
  isRefreshing = false,
  isFetchingNextPage = false,
  refreshingTitle,
  refreshingDescription,
  fetchingNextPageTitle,
  fetchingNextPageDescription,
}: PagedQueryFeedbackProps) {
  if (!hasItems) {
    return null;
  }

  if (isFetchingNextPage) {
    return (
      <div aria-live="polite">
        <InlineNotice
          title={fetchingNextPageTitle}
          description={fetchingNextPageDescription}
          tone="soft"
        />
      </div>
    );
  }

  if (isRefreshing) {
    return (
      <div aria-live="polite">
        <InlineNotice
          title={refreshingTitle}
          description={refreshingDescription}
          tone="soft"
        />
      </div>
    );
  }

  return null;
}
