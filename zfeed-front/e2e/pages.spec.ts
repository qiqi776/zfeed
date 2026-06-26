import { expect, test, type Page } from "@playwright/test";

const routes = [
    ["/", "登录 zfeed"],
    ["/home", "用 AI 构建产品：30 天从 0 到 1"],
    ["/following", "我关注的创作者今天都在用 AI 重构工作流"],
    ["/login", "登录 zfeed"],
    ["/register", "创建 zfeed 账号"],
    ["/me", "Mira Chen"],
    ["/user/jax", "用户主页真实发布内容"],
    ["/content/1001", "真实后端标题"],
    ["/content/1001/edit", "编辑内容"],
    ["/me/edit", "编辑资料"],
    ["/search", "搜索"],
    ["/compose", "发布"],
    ["/settings", "设置"]
] as const;

const routeReadyTimeoutMs = 15_000;

for (const [route, text] of routes) {
    test(`renders ${route}`, async ({ page }) => {
        if (route === "/following" || route === "/me") {
            await seedAuthSession(page);
        }
        if (route === "/content/1001/edit") {
            await seedAuthSession(page);
            await mockEditableContentDetail(page);
        }
        if (route === "/content/1001") {
            await mockContentDetail(page);
            await mockCommentList(page);
        }
        if (route === "/me") {
            await mockMeProfile(page);
            await mockUserPublishedFeed(page);
        }
        if (route === "/user/jax") {
            await mockUserProfile(page);
            await mockUserPublishedFeed(page);
        }
        if (route === "/home") {
            await mockRecommendFeed(page);
        }
        if (route === "/following") {
            await mockFollowingFeed(page);
        }

        await page.goto(route, { waitUntil: "domcontentloaded" });
        await expect(page.getByText(text).first()).toBeVisible({ timeout: routeReadyTimeoutMs });
        await expect(page.locator("body")).toHaveCSS("overflow-x", "hidden");
    });
}

test("uses the frosted home backdrop on auth entry pages", async ({ page }) => {
    await page.goto("/login", { waitUntil: "domcontentloaded" });
    await expect(page.getByTitle("zfeed 首页雾化背景")).toBeVisible();

    await page.goto("/register", { waitUntil: "domcontentloaded" });
    await expect(page.getByTitle("zfeed 首页雾化背景")).toBeVisible();
});

test("captures stable desktop and mobile feed screenshots", async ({ page }, testInfo) => {
    await mockRecommendFeed(page);
    await page.goto("/home", { waitUntil: "domcontentloaded" });
    await expect(page.getByText("用 AI 构建产品：30 天从 0 到 1").first()).toBeVisible();

    await page.screenshot({
        path: testInfo.outputPath(`feed-${testInfo.project.name}.png`),
        fullPage: false
    });
});

test("keeps migrated edit profile form values visible", async ({ page }) => {
    await seedAuthSession(page);
    await page.goto("/me/edit", { waitUntil: "domcontentloaded" });

    await expect(page.getByLabel("昵称")).toHaveValue("Mira Chen");
    await expect(page.getByLabel("简介")).toContainText("关注创作者工具");
});

test("keeps migrated edit content form values visible", async ({ page }) => {
    await seedAuthSession(page);
    await mockEditableContentDetail(page);
    await page.goto("/content/1001/edit", { waitUntil: "domcontentloaded" });

    await expect(page.getByLabel("标题")).toHaveValue("原始文章标题");
    await expect(page.getByLabel("正文")).toContainText("第一段正文。");
});

test("does not serve old .html URLs", async ({ page }) => {
    await page.goto("/following.html", { waitUntil: "domcontentloaded" });

    await expect(page.getByText("页面不存在")).toBeVisible();
});

test("does not serve removed legacy product URLs", async ({ page }) => {
    await page.goto("/profile?user=me", { waitUntil: "domcontentloaded" });
    await expect(page.getByText("页面不存在")).toBeVisible();

    await page.goto("/detail?type=article", { waitUntil: "domcontentloaded" });
    await expect(page.getByText("页面不存在")).toBeVisible();

    await page.goto("/edit-profile?user=me", { waitUntil: "domcontentloaded" });
    await expect(page.getByText("页面不存在")).toBeVisible();

    await page.goto("/liquid-glass-feed", { waitUntil: "domcontentloaded" });
    await expect(page.getByText("页面不存在")).toBeVisible();
});

test("requires auth for the me route", async ({ page }) => {
    await page.goto("/me", { waitUntil: "domcontentloaded" });

    await expect(page.getByText("登录后才能查看我的主页。")).toBeVisible();
    await expect(page.locator("[data-page-state='auth-required']")).toBeVisible();
});

test("requires auth for the following route", async ({ page }) => {
    await page.goto("/following", { waitUntil: "domcontentloaded" });

    await expect(page.getByText("登录后才能查看关注流。")).toBeVisible();
    await expect(page.locator("[data-page-state='auth-required']")).toBeVisible();
});

test("does not fall back unknown user routes to the me profile", async ({ page }) => {
    await mockMissingUserProfile(page);
    await page.goto("/user/unknown", { waitUntil: "domcontentloaded" });

    await expect(page.getByText("用户不存在", { exact: true })).toBeVisible();
    await expect(page.locator("[data-page-state='error']")).toBeVisible();
    await expect(page.getByText("编辑资料")).toHaveCount(0);
});

async function seedAuthSession(page: Page) {
    await page.addInitScript(() => {
        window.localStorage.setItem("zfeed.auth.session", JSON.stringify({
            token: "e2e-token",
            expiredAt: Math.floor(Date.now() / 1000) + 3600,
            user: { userId: 7 }
        }));
    });
}

async function mockFollowingFeed(page: Page) {
    await page.route("**/v1/feed/follow", async (route) => {
        await route.fulfill({
            contentType: "application/json",
            body: JSON.stringify({
                items: [{
                    content_id: 1001,
                    content_type: 1,
                    author_id: 1002,
                    author_name: "Mira Chen",
                    author_avatar: "",
                    title: "我关注的创作者今天都在用 AI 重构工作流",
                    description: "接口返回的关注流内容。",
                    cover_url: "",
                    published_at: 1765670400,
                    like_count: 12,
                    favorite_count: 4,
                    comment_count: 2,
                    is_liked: false,
                    is_favorited: false
                }],
                next_cursor: "",
                has_more: false
            })
        });
    });
}

async function mockRecommendFeed(page: Page) {
    await page.route("**/v1/feed/recommend", async (route) => {
        await route.fulfill({
            contentType: "application/json",
            body: JSON.stringify({
                items: [{
                    content_id: 1001,
                    content_type: 1,
                    author_id: 1001,
                    author_name: "Jax Lee",
                    author_avatar: "",
                    title: "用 AI 构建产品：30 天从 0 到 1",
                    description: "记录从 0 到 1 的完整过程，包括技术栈选择、产品思考和踩坑经验。",
                    cover_url: "",
                    published_at: 1765670400,
                    like_count: 128,
                    favorite_count: 36,
                    comment_count: 12,
                    is_liked: true,
                    is_favorited: false
                }],
                next_cursor: "",
                has_more: false,
                snapshot_id: "recommend-snapshot"
            })
        });
    });
}

async function mockMeProfile(page: Page) {
    await page.route("**/v1/users/me", async (route) => {
        await route.fulfill({
            contentType: "application/json",
            body: JSON.stringify({
                user_info: {
                    user_id: 7,
                    mobile: "13800138000",
                    nickname: "Mira Chen",
                    avatar: "",
                    bio: "关注创作者工具、液态玻璃界面和高质量信息流体验。",
                    gender: 0,
                    status: 1,
                    email: "mira@example.com",
                    birthday: 0
                },
                followee_count: 346,
                follower_count: 18600,
                like_received_count: 9200,
                favorite_received_count: 912,
                content_count: 128
            })
        });
    });
}

async function mockUserProfile(page: Page) {
    await page.route("**/v1/user/profile/jax", async (route) => {
        await route.fulfill({
            contentType: "application/json",
            body: JSON.stringify({
                user_info: {
                    user_id: 1001,
                    mobile: "",
                    nickname: "Jax Lee",
                    avatar: "",
                    bio: "AI 产品作者，关注创作者工具和高质量信息流体验。",
                    gender: 0,
                    status: 1,
                    email: "",
                    birthday: 0
                },
                followee_count: 18,
                follower_count: 2400,
                like_received_count: 1200,
                favorite_received_count: 300,
                content_count: 16,
                is_following: false
            })
        });
    });
}

async function mockUserPublishedFeed(page: Page) {
    await page.route("**/v1/feed/user/publish", async (route) => {
        await route.fulfill({
            contentType: "application/json",
            body: JSON.stringify({
                items: [{
                    content_id: 9101,
                    content_type: 1,
                    author_id: 1001,
                    author_name: "Jax Lee",
                    author_avatar: "",
                    title: "用户主页真实发布内容",
                    description: "这条内容来自用户发布流接口。",
                    cover_url: "",
                    published_at: 1765670400,
                    like_count: 9,
                    favorite_count: 4,
                    comment_count: 2,
                    is_liked: false,
                    is_favorited: false
                }],
                cursor: "",
                has_more: false
            })
        });
    });
}

async function mockEditableContentDetail(page: Page) {
    await page.route("**/v1/content/detail", async (route) => {
        await route.fulfill({
            contentType: "application/json",
            body: JSON.stringify({
                detail: {
                    content_id: "1001",
                    content_type: 1,
                    author_id: "7",
                    author_name: "Mira Chen",
                    author_avatar: "",
                    title: "原始文章标题",
                    description: "原始文章摘要",
                    cover_url: "",
                    article_content: "第一段正文。\n第二段正文。",
                    video_url: "",
                    video_duration: 0,
                    published_at: 1765670400,
                    like_count: 12,
                    favorite_count: 5,
                    comment_count: 3,
                    is_liked: false,
                    is_favorited: false,
                    is_following_author: false
                }
            })
        });
    });
}

async function mockContentDetail(page: Page) {
    await page.route("**/v1/content/detail", async (route) => {
        await route.fulfill({
            contentType: "application/json",
            body: JSON.stringify({
                detail: {
                    content_id: "1001",
                    content_type: 1,
                    author_id: "1001",
                    author_name: "测试作者",
                    author_avatar: "1",
                    title: "真实后端标题",
                    description: "真实后端摘要",
                    cover_url: "",
                    article_content: "真实后端正文",
                    video_url: "",
                    video_duration: 0,
                    published_at: 1765670400,
                    like_count: 12,
                    favorite_count: 5,
                    comment_count: 1,
                    is_liked: false,
                    is_favorited: false,
                    is_following_author: false
                }
            })
        });
    });
}

async function mockCommentList(page: Page) {
    await page.route("**/v1/interaction/comment/list", async (route) => {
        await route.fulfill({
            contentType: "application/json",
            body: JSON.stringify({
                comments: [{
                    comment_id: 3001,
                    content_id: 1001,
                    user_id: 2001,
                    reply_to_user_id: 0,
                    parent_id: 0,
                    root_id: 3001,
                    comment: "真实评论内容",
                    created_at: 1765670400,
                    status: 1,
                    user_name: "测试1",
                    user_avatar: "1",
                    reply_count: 0
                }],
                next_cursor: 0,
                has_more: false
            })
        });
    });
}

async function mockMissingUserProfile(page: Page) {
    await page.route("**/v1/user/profile/unknown", async (route) => {
        await route.fulfill({
            contentType: "application/json",
            status: 404,
            body: JSON.stringify({})
        });
    });
}
