import { useInfiniteQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { type FormEvent, type KeyboardEvent, useEffect, useMemo, useState } from "react";
import { Link, useSearchParams } from "react-router-dom";

import { useSessionStore } from "@/entities/session/model/session.store";
import { followUser, unfollowUser } from "@/features/interaction/api/interaction.api";
import { searchContents, searchUsers } from "@/features/search/api/search.api";
import {
  captureQuerySnapshots,
  getFollowSyncQueryKeys,
  patchAuthorFollowStateAcrossPages,
  restoreQuerySnapshots,
} from "@/shared/lib/query/cacheSync";
import { feedKeys, searchKeys, userKeys } from "@/shared/lib/query/queryKeys";
import { FeedGridSkeleton } from "@/shared/ui/FeedGridSkeleton";
import { ImageFallback } from "@/shared/ui/ImageFallback";
import { InlineNotice } from "@/shared/ui/InlineNotice";
import { PagedQueryFeedback } from "@/shared/ui/PagedQueryFeedback";
import { PageHeader } from "@/shared/ui/PageHeader";
import { StatePanel } from "@/shared/ui/StatePanel";
import { UserCardSkeletonGrid } from "@/shared/ui/UserCardSkeletonGrid";
import { useToast } from "@/shared/ui/toast/toast.store";

const defaultPageSize = 20;
const searchTabs = ["contents", "users"] as const;

type SearchTab = (typeof searchTabs)[number];

const searchTabMeta: Record<SearchTab, { tabId: string; panelId: string; label: string }> = {
  contents: {
    tabId: "search-tab-contents",
    panelId: "search-panel-contents",
    label: "搜索内容",
  },
  users: {
    tabId: "search-tab-users",
    panelId: "search-panel-users",
    label: "搜索用户",
  },
};

function formatPublishedAt(timestamp: number) {
  if (!timestamp) {
    return "刚刚";
  }
  return new Intl.DateTimeFormat("zh-CN", {
    month: "2-digit",
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
  }).format(timestamp * 1000);
}

export function SearchPage() {
  const [searchParams, setSearchParams] = useSearchParams();
  const queryClient = useQueryClient();
  const { showToast } = useToast();
  const currentUserId = useSessionStore((state) => state.user?.userId ?? 0);

  const currentTab: SearchTab = searchParams.get("tab") === "users" ? "users" : "contents";
  const keyword = searchParams.get("q")?.trim() ?? "";
  const [draftKeyword, setDraftKeyword] = useState(keyword);
  const activeTabMeta = searchTabMeta[currentTab];

  useEffect(() => {
    setDraftKeyword(keyword);
  }, [keyword]);

  const userQuery = useInfiniteQuery({
    queryKey: searchKeys.users(keyword, currentUserId, defaultPageSize),
    enabled: currentTab === "users" && keyword.length > 0,
    initialPageParam: { cursor: undefined as number | undefined },
    queryFn: ({ pageParam }) =>
      searchUsers({
        query: keyword,
        cursor: pageParam.cursor,
        page_size: defaultPageSize,
      }),
    getNextPageParam: (lastPage) =>
      lastPage.has_more ? { cursor: lastPage.next_cursor } : undefined,
  });

  const contentQuery = useInfiniteQuery({
    queryKey: searchKeys.contents(keyword, currentUserId, defaultPageSize),
    enabled: currentTab === "contents" && keyword.length > 0,
    initialPageParam: { cursor: undefined as number | undefined },
    queryFn: ({ pageParam }) =>
      searchContents({
        query: keyword,
        cursor: pageParam.cursor,
        page_size: defaultPageSize,
      }),
    getNextPageParam: (lastPage) =>
      lastPage.has_more ? { cursor: lastPage.next_cursor } : undefined,
  });

  const userItems = useMemo(
    () => userQuery.data?.pages.flatMap((page) => page.items) ?? [],
    [userQuery.data],
  );
  const contentItems = useMemo(
    () => contentQuery.data?.pages.flatMap((page) => page.items) ?? [],
    [contentQuery.data],
  );

  const followMutation = useMutation({
    mutationFn: async (payload: { userId: number; isFollowing: boolean }) => {
      if (payload.isFollowing) {
        const res = await unfollowUser({ target_user_id: payload.userId });
        return res.is_followed;
      }
      const res = await followUser({ target_user_id: payload.userId });
      return res.is_followed;
    },
    onMutate: async (payload) => {
      const queryKeys = getFollowSyncQueryKeys(payload.userId, currentUserId);
      await Promise.all(queryKeys.map((queryKey) => queryClient.cancelQueries({ queryKey })));

      const previousSnapshots = captureQuerySnapshots(queryClient, queryKeys);
      patchAuthorFollowStateAcrossPages(queryClient, payload.userId, currentUserId, !payload.isFollowing);

      return { previousSnapshots };
    },
    onSuccess: async (isFollowed, payload) => {
      if (isFollowed !== !payload.isFollowing) {
        patchAuthorFollowStateAcrossPages(queryClient, payload.userId, currentUserId, isFollowed);
      }

      await Promise.all([
        queryClient.invalidateQueries({ queryKey: feedKeys.followPrefix(currentUserId) }),
        queryClient.invalidateQueries({ queryKey: userKeys.profilePrefix(payload.userId) }),
        queryClient.invalidateQueries({ queryKey: userKeys.profilePrefix(currentUserId) }),
        queryClient.invalidateQueries({ queryKey: userKeys.mePrefix() }),
      ]);
    },
    onError: (error: Error, _payload, context) => {
      if (context?.previousSnapshots) {
        restoreQuerySnapshots(queryClient, context.previousSnapshots);
      }
      showToast({
        tone: "error",
        title: "关注状态更新失败",
        description: error.message || "请稍后重试。",
      });
    },
  });

  function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const next = draftKeyword.trim();
    setSearchParams((current) => {
      const params = new URLSearchParams(current);
      if (next) {
        params.set("q", next);
      } else {
        params.delete("q");
      }
      params.set("tab", currentTab);
      return params;
    });
  }

  function switchTab(tab: SearchTab) {
    setSearchParams((current) => {
      const params = new URLSearchParams(current);
      params.set("tab", tab);
      if (draftKeyword.trim()) {
        params.set("q", draftKeyword.trim());
      }
      return params;
    });
  }

  function handleTabKeyDown(event: KeyboardEvent<HTMLButtonElement>, tab: SearchTab) {
    let nextTab: SearchTab | null = null;

    if (event.key === "ArrowRight" || event.key === "ArrowDown") {
      nextTab = searchTabs[(searchTabs.indexOf(tab) + 1) % searchTabs.length];
    }

    if (event.key === "ArrowLeft" || event.key === "ArrowUp") {
      nextTab = searchTabs[(searchTabs.indexOf(tab) - 1 + searchTabs.length) % searchTabs.length];
    }

    if (event.key === "Home") {
      nextTab = searchTabs[0];
    }

    if (event.key === "End") {
      nextTab = searchTabs[searchTabs.length - 1];
    }

    if (!nextTab || nextTab === tab) {
      return;
    }

    event.preventDefault();
    switchTab(nextTab);

    const focusNextTab = () => {
      document.getElementById(searchTabMeta[nextTab].tabId)?.focus();
    };

    if (typeof window !== "undefined" && typeof window.requestAnimationFrame === "function") {
      window.requestAnimationFrame(focusNextTab);
      return;
    }

    focusNextTab();
  }

  const activeQuery = currentTab === "users" ? userQuery : contentQuery;
  const hasKeyword = keyword.length > 0;
  const hasActiveResults =
    currentTab === "users" ? userItems.length > 0 : contentItems.length > 0;

  return (
    <section className="space-y-6">
      <PageHeader
        eyebrow="Search"
        title="搜索"
        description="先提供基础用户 / 内容搜索，当前实现基于后端最小可用检索接口。"
      />

      <form
        onSubmit={handleSubmit}
        className="rounded-[28px] border border-slate-200 bg-white p-5 shadow-card"
      >
        <div className="flex flex-col gap-4 lg:flex-row lg:items-center">
          <label htmlFor="search-keyword-input" className="sr-only">
            搜索关键词
          </label>
          <input
            id="search-keyword-input"
            value={draftKeyword}
            onChange={(event) => setDraftKeyword(event.target.value)}
            aria-describedby="search-keyword-hint"
            placeholder={currentTab === "users" ? "搜索昵称、简介或手机号" : "搜索标题和描述"}
            className="w-full rounded-2xl border border-slate-200 px-4 py-3 outline-none ring-accent transition focus:ring"
          />
          <p id="search-keyword-hint" className="sr-only">
            {currentTab === "users"
              ? "当前为用户搜索，可按昵称、简介或手机号查找用户。"
              : "当前为内容搜索，可按标题或描述查找内容。"}
          </p>
          <button
            type="submit"
            className="rounded-full bg-ink px-5 py-2.5 text-sm font-medium text-white transition hover:bg-slate-800"
          >
            搜索
          </button>
        </div>

        <div className="mt-4 flex flex-wrap gap-3" role="tablist" aria-label="搜索类型">
          <button
            type="button"
            onClick={() => switchTab("contents")}
            onKeyDown={(event) => handleTabKeyDown(event, "contents")}
            role="tab"
            id={searchTabMeta.contents.tabId}
            aria-selected={currentTab === "contents"}
            aria-controls={searchTabMeta.contents.panelId}
            tabIndex={currentTab === "contents" ? 0 : -1}
            className={[
              "rounded-full px-4 py-2 text-sm transition",
              currentTab === "contents"
                ? "bg-[#eef7fb] text-accent"
                : "border border-slate-200 bg-white text-slate-600 hover:border-accent hover:text-accent",
            ].join(" ")}
          >
            {searchTabMeta.contents.label}
          </button>
          <button
            type="button"
            onClick={() => switchTab("users")}
            onKeyDown={(event) => handleTabKeyDown(event, "users")}
            role="tab"
            id={searchTabMeta.users.tabId}
            aria-selected={currentTab === "users"}
            aria-controls={searchTabMeta.users.panelId}
            tabIndex={currentTab === "users" ? 0 : -1}
            className={[
              "rounded-full px-4 py-2 text-sm transition",
              currentTab === "users"
                ? "bg-[#eef7fb] text-accent"
                : "border border-slate-200 bg-white text-slate-600 hover:border-accent hover:text-accent",
            ].join(" ")}
          >
            {searchTabMeta.users.label}
          </button>
        </div>
      </form>

      <InlineNotice
        title="当前搜索是基础版本"
        description="内容搜索目前基于 MySQL 模糊匹配，适合补齐页面闭环，但还不是独立检索索引。"
        tone="soft"
      />

      <section
        id={activeTabMeta.panelId}
        role="tabpanel"
        aria-labelledby={activeTabMeta.tabId}
        className="space-y-6"
      >
        {!hasKeyword ? (
          <StatePanel
            title="输入关键词开始搜索"
            description="你可以搜索内容标题、描述，或直接按昵称、简介查找用户。"
          />
        ) : null}

        <PagedQueryFeedback
          hasItems={hasActiveResults}
          isRefreshing={activeQuery.isRefetching && !activeQuery.isLoading && !activeQuery.isFetchingNextPage}
          isFetchingNextPage={activeQuery.isFetchingNextPage}
          refreshingTitle="搜索结果正在同步"
          refreshingDescription="当前关键词对应的结果正在后台刷新，你可以继续浏览已加载内容。"
          fetchingNextPageTitle="正在继续加载搜索结果"
          fetchingNextPageDescription="下一页搜索结果正在拼接到当前列表中。"
        />

        {hasKeyword && !activeQuery.isLoading && !activeQuery.isError ? (
          <p className="sr-only" role="status">
            {currentTab === "users"
              ? `当前已加载 ${userItems.length} 个用户搜索结果。`
              : `当前已加载 ${contentItems.length} 条内容搜索结果。`}
          </p>
        ) : null}

        {activeQuery.isLoading ? (
          currentTab === "users" ? <UserCardSkeletonGrid /> : <FeedGridSkeleton />
        ) : null}

        {activeQuery.isError ? (
          <StatePanel
            title="搜索失败"
            description={(activeQuery.error as Error)?.message || "请稍后重试"}
            tone="error"
          />
        ) : null}

        {currentTab === "users" && userItems.length > 0 ? (
          <ul className="grid gap-4 md:grid-cols-2">
            {userItems.map((item) => (
              <li key={item.user_id}>
                <article className="rounded-[28px] border border-slate-200 bg-white p-5 shadow-card">
                  <div className="flex items-start gap-4">
                    <Link to={`/users/${item.user_id}`} className="shrink-0">
                      <ImageFallback
                        src={item.avatar}
                        alt={item.nickname || `用户 ${item.user_id}`}
                        name={item.nickname || `用户 ${item.user_id}`}
                        variant="avatar"
                        containerClassName="h-16 w-16 overflow-hidden rounded-full bg-slate-100"
                        imageClassName="h-full w-full object-cover"
                      />
                    </Link>

                    <div className="min-w-0 flex-1">
                      <Link
                        to={`/users/${item.user_id}`}
                        className="block text-lg font-semibold text-slate-900"
                      >
                        {item.nickname || `用户 ${item.user_id}`}
                      </Link>
                      <p className="mt-2 line-clamp-3 text-sm leading-6 text-slate-500">
                        {item.bio || "这个人还没有留下简介。"}
                      </p>
                      <div className="mt-4 flex flex-wrap gap-3">
                        <Link
                          to={`/users/${item.user_id}`}
                          className="rounded-full border border-slate-200 bg-white px-4 py-2 text-sm text-slate-600 transition hover:border-accent hover:text-accent"
                        >
                          查看主页
                        </Link>
                        {currentUserId > 0 && currentUserId !== item.user_id ? (
                          <button
                            type="button"
                            disabled={followMutation.isPending}
                            onClick={() =>
                              followMutation.mutate({
                                userId: item.user_id,
                                isFollowing: item.is_following,
                              })
                            }
                            className={[
                              "rounded-full px-4 py-2 text-sm font-medium transition",
                              item.is_following
                                ? "border border-slate-200 bg-white text-slate-700 hover:border-accent hover:text-accent"
                                : "bg-ink text-white hover:bg-slate-800",
                              followMutation.isPending &&
                              followMutation.variables?.userId === item.user_id
                                ? "opacity-70"
                                : "",
                            ].join(" ")}
                          >
                            {followMutation.isPending &&
                            followMutation.variables?.userId === item.user_id
                              ? "处理中..."
                              : item.is_following
                                ? "已关注"
                                : "关注"}
                          </button>
                        ) : null}
                      </div>
                    </div>
                  </div>
                </article>
              </li>
            ))}
          </ul>
        ) : null}

        {currentTab === "contents" && contentItems.length > 0 ? (
          <ul className="grid gap-4 md:grid-cols-2">
            {contentItems.map((item) => (
              <li key={item.content_id}>
                <Link
                  to={`/content/${item.content_id}`}
                  className="block overflow-hidden rounded-[28px] border border-slate-200 bg-white shadow-card transition hover:-translate-y-0.5 hover:border-accent"
                >
                  <ImageFallback
                    src={item.cover_url}
                    alt={item.title || "内容封面"}
                    containerClassName="aspect-[16/9] bg-slate-100"
                    imageClassName="h-full w-full object-cover"
                  />
                  <div className="space-y-3 p-5">
                    <div className="flex items-center justify-between gap-3 text-xs text-slate-400">
                      <span>{item.content_type === 20 ? "视频" : "文章"}</span>
                      <span>{formatPublishedAt(item.published_at)}</span>
                    </div>
                    <p className="line-clamp-2 text-lg font-semibold text-slate-900">
                      {item.title || "未命名内容"}
                    </p>
                    <p className="text-sm text-slate-500">
                      作者：{item.author_name || `用户 ${item.author_id}`}
                    </p>
                  </div>
                </Link>
              </li>
            ))}
          </ul>
        ) : null}

        {hasKeyword &&
        !activeQuery.isLoading &&
        !activeQuery.isError &&
        ((currentTab === "users" && userItems.length === 0) ||
          (currentTab === "contents" && contentItems.length === 0)) ? (
          <StatePanel
            title={currentTab === "users" ? "没有搜索到匹配用户" : "没有搜索到内容结果"}
            description="换一个关键词，或者切换到另一个搜索标签再试试。"
          />
        ) : null}

        {activeQuery.hasNextPage ? (
          <button
            type="button"
            onClick={() => activeQuery.fetchNextPage()}
            disabled={activeQuery.isFetchingNextPage}
            className="rounded-full border border-slate-200 bg-white px-5 py-2.5 text-sm text-slate-600 transition hover:border-accent hover:text-accent disabled:opacity-60"
          >
            {activeQuery.isFetchingNextPage ? "加载中..." : "加载更多"}
          </button>
        ) : null}
      </section>
    </section>
  );
}
