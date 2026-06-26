# User Published Feed Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the `/user/:userId` profile publish placeholder with a real published content feed backed by `POST /v1/feed/user/publish`.

**Architecture:** Add a small optional-auth API helper, then let `ProfilePage` load the user profile first and fetch the published feed for the normalized backend `user_id`. Render the feed inside the existing profile main column using the same glass panel and feed card class patterns already used in Home/Following pages.

**Tech Stack:** React `createElement`, TypeScript, Vite, Vitest, Testing Library, Playwright.

---

## File Structure

- Modify `src/runtime/apiClient.ts`: add `UserPublishedFeedBody` and `getUserPublishedFeed`.
- Modify `src/runtime/apiClient.test.ts`: verify the new helper serializes `POST /v1/feed/user/publish` with optional Bearer auth.
- Modify `src/pages/ProfilePage.tsx`: add published feed state, fetch flow, state rendering, and feed card rendering without changing existing profile header or rail classes.
- Modify `src/App.test.tsx`: cover ready, empty, error, and XSS-safe published feed behavior on `/user/jax`.
- Modify `e2e/pages.spec.ts`: mock `/v1/feed/user/publish` for `/user/jax` screenshot route coverage.

## Task 1: API Helper

**Files:**
- Modify: `src/runtime/apiClient.ts`
- Modify: `src/runtime/apiClient.test.ts`

- [ ] **Step 1: Write the failing test**

In `src/runtime/apiClient.test.ts`, add `getUserPublishedFeed` to the import list and update `serializes optional read helpers for feed and content detail`:

```ts
await getUserPublishedFeed({ user_id: "1001", cursor: "", page_size: 20 });

expect(fetchMock).toHaveBeenNthCalledWith(5, "/v1/feed/user/publish", expect.objectContaining({
    method: "POST",
    headers: expect.objectContaining({
        Authorization: "Bearer reader-token"
    }),
    body: JSON.stringify({ user_id: "1001", cursor: "", page_size: 20 })
}));
```

- [ ] **Step 2: Run test to verify it fails**

Run: `npm test -- src/runtime/apiClient.test.ts`

Expected: FAIL because `getUserPublishedFeed` is not exported from `src/runtime/apiClient.ts`.

- [ ] **Step 3: Write minimal implementation**

In `src/runtime/apiClient.ts`, add:

```ts
type UserPublishedFeedBody = {
    user_id: string;
    cursor: string;
    page_size: number;
};

export function getUserPublishedFeed<T>(body: UserPublishedFeedBody) {
    return apiRequest<T>("/v1/feed/user/publish", { method: "POST", body, optionalAuth: true });
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `npm test -- src/runtime/apiClient.test.ts`

Expected: PASS.

## Task 2: Profile Page Published Feed

**Files:**
- Modify: `src/App.test.tsx`
- Modify: `src/pages/ProfilePage.tsx`

- [ ] **Step 1: Write failing ready-state test**

In `src/App.test.tsx`, add a helper near `userProfilePayload`:

```ts
function userPublishedFeedPayload(items: unknown[] = [defaultRecommendFeedItem({
    content_id: 9101,
    author_id: 1001,
    author_name: "Jax Lee",
    title: "用户主页真实发布内容",
    description: "这条内容来自用户发布流接口。",
    like_count: 9,
    favorite_count: 4,
    comment_count: 2
})]) {
    return {
        items,
        cursor: "",
        has_more: false
    };
}
```

Then add a test:

```ts
it("loads a user published feed after the user profile", async () => {
    window.localStorage.setItem("zfeed.auth.session", JSON.stringify({
        token: "reader-token",
        expiredAt: Math.floor(Date.now() / 1000) + 3600,
        user: { userId: 7 }
    }));
    const fetchMock = vi.fn(async (input: RequestInfo | URL) => {
        if (input === "/v1/user/profile/jax") {
            return jsonResponse(userProfilePayload());
        }

        return jsonResponse(userPublishedFeedPayload());
    });
    vi.stubGlobal("fetch", fetchMock);
    window.history.pushState({}, "", "/user/jax");

    render(<App />);

    expect(await screen.findByRole("heading", { name: "用户主页真实发布内容" })).toBeInTheDocument();
    expect(screen.getByText("这条内容来自用户发布流接口。")).toBeInTheDocument();
    expect(fetchMock).toHaveBeenCalledWith("/v1/feed/user/publish", expect.objectContaining({
        method: "POST",
        headers: expect.objectContaining({
            Authorization: "Bearer reader-token"
        }),
        body: JSON.stringify({ user_id: "1001", cursor: "", page_size: 20 })
    }));
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `npm test -- src/App.test.tsx`

Expected: FAIL because `/user/jax` still renders the publish placeholder and never calls `/v1/feed/user/publish`.

- [ ] **Step 3: Write minimal implementation**

In `src/pages/ProfilePage.tsx`:

- Import `getUserPublishedFeed`.
- Add feed item, response, and state types.
- Add `publishedFeed` state to `UserProfilePage`.
- After `getUserProfile` resolves, call `getUserPublishedFeed({ user_id: profile.userId, cursor: "", page_size: 20 })`.
- Pass the feed state into `renderUserReadyPage(profile, variant, publishedFeed)`.
- Replace the placeholder section with `renderUserPublishedFeed(publishedFeed)`.
- Add `renderUserPublishedFeed`, `renderUserPublishedCard`, `formatPublishedAt`, `formatCount`, and `getAvatarFallback` helpers using existing glass classes.

- [ ] **Step 4: Run test to verify it passes**

Run: `npm test -- src/App.test.tsx`

Expected: PASS.

## Task 3: Published Feed Empty, Error, and Safe Text

**Files:**
- Modify: `src/App.test.tsx`
- Modify: `src/pages/ProfilePage.tsx`

- [ ] **Step 1: Write failing coverage tests**

Add tests in `src/App.test.tsx`:

```ts
it("shows an empty state when a user has no published content", async () => {
    vi.stubGlobal("fetch", vi.fn(async (input: RequestInfo | URL) => {
        if (input === "/v1/user/profile/jax") {
            return jsonResponse(userProfilePayload());
        }

        return jsonResponse(userPublishedFeedPayload([]));
    }));
    window.history.pushState({}, "", "/user/jax");

    render(<App />);

    expect(await screen.findByText("还没有公开发布内容")).toBeInTheDocument();
});

it("shows a contained error state when the user published feed fails", async () => {
    vi.stubGlobal("fetch", vi.fn(async (input: RequestInfo | URL) => {
        if (input === "/v1/user/profile/jax") {
            return jsonResponse(userProfilePayload());
        }

        return jsonResponse({ message: "raw feed failure" }, { status: 500 });
    }));
    window.history.pushState({}, "", "/user/jax");

    render(<App />);

    expect(await screen.findByRole("heading", { name: "Jax Lee" })).toBeInTheDocument();
    const errorState = await screen.findByText("发布内容加载失败");
    expect(errorState.closest("[data-page-state]")).toHaveAttribute("data-page-state", "error");
    expect(screen.queryByText("raw feed failure")).not.toBeInTheDocument();
});

it("renders user published content as safe text", async () => {
    vi.stubGlobal("fetch", vi.fn(async (input: RequestInfo | URL) => {
        if (input === "/v1/user/profile/jax") {
            return jsonResponse(userProfilePayload());
        }

        return jsonResponse(userPublishedFeedPayload([
            defaultRecommendFeedItem({
                content_id: 9102,
                author_id: 1001,
                author_name: "Jax Lee",
                title: "发布 <script>alert(1)</script>",
                description: "摘要 <img src=x onerror=alert(1)>"
            })
        ]));
    }));
    window.history.pushState({}, "", "/user/jax");

    render(<App />);

    expect(await screen.findByRole("heading", { name: "发布 <script>alert(1)</script>" })).toBeInTheDocument();
    expect(screen.getByText("摘要 <img src=x onerror=alert(1)>")).toBeInTheDocument();
    expect(document.querySelector("script")).toBeNull();
});
```

- [ ] **Step 2: Run tests to verify they fail if state rendering is incomplete**

Run: `npm test -- src/App.test.tsx`

Expected before Task 2 implementation: FAIL. Expected after Task 2 minimal implementation: any missing empty/error/safe behavior fails clearly.

- [ ] **Step 3: Complete state rendering**

Ensure `renderUserPublishedFeed` handles:

- `loading`: `PageState` title `正在加载发布内容`
- `empty`: `PageState` title `还没有公开发布内容`
- `error`: `PageState` title `发布内容加载失败`
- `ready`: mapped published feed cards

- [ ] **Step 4: Run tests to verify they pass**

Run: `npm test -- src/App.test.tsx`

Expected: PASS.

## Task 4: E2E Mock and Final Verification

**Files:**
- Modify: `e2e/pages.spec.ts`

- [ ] **Step 1: Update E2E mock**

In `e2e/pages.spec.ts`, add a route mock for `**/v1/feed/user/publish` in the same branch that mocks `/user/jax`, returning one published item with title `用户主页真实发布内容`.

- [ ] **Step 2: Run verification**

Run:

```bash
npm test
npm run lint
npm run build
npm run test:e2e
```

Expected:

- Vitest reports all test files passing.
- ESLint exits 0.
- Vite build exits 0.
- Playwright reports all route screenshot tests passing.

- [ ] **Step 3: Commit implementation**

Run:

```bash
git status --short --branch -uall
git add src/runtime/apiClient.ts src/runtime/apiClient.test.ts src/App.test.tsx src/pages/ProfilePage.tsx e2e/pages.spec.ts
git commit -m "接入用户发布流"
```

Expected: commit includes only the five implementation/test files, not `spec/README.md`.
