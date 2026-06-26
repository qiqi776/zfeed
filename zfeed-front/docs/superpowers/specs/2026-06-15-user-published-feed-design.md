# User Published Feed Design

## Goal

把 `/user/:userId` 用户主页的发布区域从占位文案升级为真实发布内容列表，同时保持当前用户主页的玻璃质感、布局比例和卡片样式不漂移。

## Scope

- 本次只实现用户主页的“发布”内容流。
- 不在本次加入收藏 Tab、粉丝 Tab 或分页加载更多。
- 不改 ProfileHeader、左右 rail、统计区和现有按钮视觉。
- 不抽全局 FeedCard 组件；先用局部渲染函数复用现有卡片类名，降低样式风险。

## API

- Endpoint: `POST /v1/feed/user/publish`
- Auth: optional Bearer token
- Request body:

```json
{
  "user_id": "1001",
  "cursor": "",
  "page_size": 20
}
```

- Response shape follows existing feed pages:

```json
{
  "items": [
    {
      "content_id": 1001,
      "content_type": 1,
      "author_id": 1001,
      "author_name": "Jax Lee",
      "author_avatar": "",
      "title": "内容标题",
      "description": "内容摘要",
      "cover_url": "",
      "published_at": 1765670400,
      "like_count": 12,
      "favorite_count": 3,
      "comment_count": 2,
      "is_liked": false,
      "is_favorited": false
    }
  ],
  "cursor": "next"
}
```

## Data Flow

1. `/user/:userId` 先加载 `GET /v1/user/profile/:userId`。
2. 用户资料 ready 后，用资料里的 `user_id` 请求 `POST /v1/feed/user/publish`。
3. 发布流请求失败不影响用户资料展示，只在发布区域显示失败态。
4. 发布流返回空数组时显示玻璃风格空态，引导继续探索。
5. 发布流里的标题、摘要、作者名继续由 React 文本渲染，避免执行用户内容。

## UI

- 发布区域保留当前主内容栏位置。
- 标题仍为“发布内容”。
- 加载、空、错误态使用现有 `PageState`，包在 `glass-panel rounded-3xl p-5 hover-lift shine-effect` 容器里。
- 有内容时使用和现有 feed 一致的 `glass-panel rounded-3xl p-5 md:p-6 hover-lift shine-effect` 卡片。
- 长标题和摘要使用 `break-words`，摘要限制三行，避免 375px 宽度撑破。

## Tests

- `apiClient` 测 `getUserPublishedFeed` 序列化请求体并携带 optional Bearer。
- `App` 测 `/user/jax` 会先加载 profile，再加载发布流，并渲染发布内容。
- `App` 测发布流空态、错误态和 XSS 安全渲染。
- E2E 为 `/user/jax` mock 发布流，保持 375px、768px、1440px 路由截图回归稳定。

## Non-Goals

- 不实现发布流分页。
- 不实现收藏和粉丝 Tab。
- 不新增视觉风格。
- 不改变关注按钮动作和资料接口。
