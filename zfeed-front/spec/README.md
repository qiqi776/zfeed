# Specification: zfeed 液态毛玻璃 Feed 流内容系统

## Metadata
- **Version**: 1.1.2
- **Status**: Draft
- **Author**: Codex
- **Created**: 2026-06-11
- **Last Updated**: 2026-06-15

## Overview
本文档定义 `zfeed` 的 Web 端和 App 端前端体验方案。方案基于当前 `front-api` 已提供的用户、内容、Feed、互动、搜索能力，目标是把项目从后端接口演示升级为一个可真实浏览、搜索、发布、互动的现代内容社区产品。

后端接口以 `/home/zz/workspace/projects/zfeed/app/front/doc/front.api` 为总入口，实际端点来自 `base/base.api`、`user/user.api`、`content/content.api`、`feed/feed.api`、`interaction/interaction.api` 和 `search/search.api`。本文所有接口路径都以这些 `.api` 文件为准。

设计方向锁定为「液态毛玻璃 Feed 流内容系统」。这不是营销落地页，也不是普通后台模板。首屏必须直接展示真实可用的产品主界面：固定顶部导航、左侧频道导航、中央信息流、右侧趋势辅助栏，以及能反映真实内容生态的多类型 Feed 卡片。

视觉语言以当前 `zfeed-front` 已落地的页面为准：浅色 Material 3 质感、蓝色主操作色、白色半透明玻璃面板、固定的冷白到冰蓝环境光、Hanken Grotesk 与 Inter 字体、Material Symbols 和 Lucide 线性图标。界面保留半透明磨砂、边缘高光、柔和投影、轻微景深和漂浮层次，但不改成新的品牌方向。

发布页采用本文档定义的新方案，旧版发布页线框不作为视觉参考。

## Implementation Status

标记说明：

- `[x] 已完成`：当前 Web React 项目已经落地，且有单元测试或 E2E 覆盖主要路径。
- `[~] 部分完成`：已经有页面、接口封装或核心交互，但还缺少规格中的部分状态、分页、媒体上传、独立页面或 App 实现。
- `[ ] 未完成`：仍停留在规格阶段，当前项目未接入对应页面或接口。

当前快照：

- `[x] 已完成`：Web 端已经覆盖启动和会话恢复、登录、注册、推荐流、关注流、内容详情、搜索、发布文章基础流、内容编辑、个人主页、资料编辑、设置和 404。
- `[x] 已完成`：认证 token、Bearer header、过期会话清理、Optional login 降级、Required login 阻断、旧 `.html` URL 移除和后端错误脱敏已经接入。
- `[x] 已完成`：点赞、取消点赞、收藏、取消收藏、关注、取消关注、评论、回复、删除评论、发布文章、编辑内容、资料保存等写操作已经接入 Bearer 请求和失败回滚。
- `[~] 部分完成`：推荐流、关注流、用户发布流、搜索、个人页、发布页和设置页已经可用，但还缺少连续分页、完整频道状态、独立收藏/粉丝/发布列表、主题设置和离线缓存。
- `[~] 部分完成`：发布和编辑已经覆盖文章基础流和内容编辑流，视频发布、媒体上传凭证、上传进度、草稿恢复、上传失败恢复仍未完成。
- `[ ] 未完成`：App 端目前仍是产品和交互规格，没有 iOS、Android 或跨端代码实现。

## Requirements

### Functional Requirements
- FR-1: Web 端必须覆盖首页推荐流、关注流、内容详情、搜索、发布、编辑、个人主页、资料编辑、粉丝列表、收藏列表、登录注册和错误页。`[~] 部分完成`
- FR-2: App 端必须覆盖与 Web 端同等的核心业务闭环，并明确原生导航、手势、底部 Sheet、沉浸媒体和安全区规则。`[ ] 未完成`
- FR-3: Web 端和 App 端必须共享同一套产品信息架构、视觉 token、状态模型和接口映射。`[~] 部分完成`
- FR-4: 推荐流必须支持 `cursor` 和 `snapshot_id` 的连续分页。`[~] 部分完成`
- FR-5: 搜索必须支持内容和用户两个结果面，并兼容 `latest`、`relevance`、`hybrid` 三类模式。`[~] 部分完成`
- FR-6: 发布必须支持文章、视频、编辑、媒体上传凭证、上传进度、发布前校验和发布失败恢复。`[~] 部分完成`
- FR-7: 互动必须支持点赞、取消点赞、收藏、取消收藏、关注、取消关注、评论、回复、删除评论和对应的乐观更新。`[x] 已完成`
- FR-8: 未登录用户必须能浏览公开内容、搜索和打开详情；触发写操作时使用登录引导，不直接丢失当前操作上下文。`[x] 已完成`
- FR-9: 每个页面必须有 loading、empty、error 和 auth required 状态。`[~] 部分完成`
- FR-10: 当前 `front-api` 未暴露关注列表接口，因此不设计“我关注的人列表”为已支持页面。只展示关注数，或在后续接口补齐后再进入页面。`[x] 已完成`
- FR-11: Web 首屏必须是完整产品界面，不能出现营销 hero、宣传 CTA、孤立卡片或空白概念稿。`[x] 已完成`
- FR-12: Web 桌面端必须包含顶部固定导航栏、左侧导航栏、中央 Feed 和右侧辅助栏。`[x] 已完成`
- FR-13: Feed 卡片必须覆盖图文、长文摘要、图片组、视频预览、链接分享等内容类型的视觉结构。`[~] 部分完成`
- FR-14: 交互状态必须明确展示当前频道高亮、点赞选中、收藏选中、搜索聚焦、通知角标、趋势榜排名和 Feed 卡片 hover 上浮。`[~] 部分完成`
- FR-15: 移动端必须简化顶部导航，使用底部 Tab Bar，Feed 单列显示。`[~] 部分完成`

### Non-functional Requirements
- NFR-1: 视觉上使用液态毛玻璃，但不能牺牲正文、评论、表单和按钮的可读性。`[x] 已完成`
- NFR-2: Web 端默认使用 React 或 Next.js、Tailwind、Motion、Phosphor Icons。若项目后续选择其他栈，必须保持本文档定义的设计 token 和交互模型。`[~] 部分完成`
- NFR-3: App 端不复刻 Web CSS。iOS 使用原生模糊材质和安全区，Android 使用原生模糊能力或半透明实体降级。`[ ] 未完成`
- NFR-4: 全部动画必须尊重 reduced motion。关键动效只使用 transform 和 opacity。`[~] 部分完成`
- NFR-5: 可访问性必须覆盖键盘焦点、语义按钮、图片 alt、表单 label、错误提示和触控目标。`[~] 部分完成`
- NFR-6: Web 端 LCP 目标小于 2.5 秒，INP 目标小于 200ms，CLS 目标小于 0.1。`[ ] 未完成`
- NFR-7: App 端首屏必须在弱网下优先呈现骨架屏和本地缓存，不用空白页等待接口。`[ ] 未完成`
- NFR-8: 认证 token 使用 `Authorization: Bearer <token>`，与当前 `front-api` middleware 保持一致。`[x] 已完成`
- NFR-9: 背景使用冷白、冰蓝、银灰、淡青和轻微紫色点缀，紫色只能作为环境光点缀，不能成为主操作色。`[x] 已完成`
- NFR-10: 所有玻璃面板必须有 1px 半透明描边、内高光、细腻阴影和实体降级方案。`[~] 部分完成`
- NFR-11: 信息密度保持适中，适合长时间阅读，不做密集仪表盘，也不做空旷画廊式页面。`[x] 已完成`

## Design Read

Reading this as: 高保真内容社区产品主界面，为创作者、知识社区、趋势资讯和社交内容分发场景服务。视觉语言以当前 `zfeed-front` 为准，是浅色 Material 3 玻璃界面加蓝色主操作色，而不是重新起一套新的品牌色或深色赛博风格。首屏直接呈现真实可用的 Feed 系统。

三项设计拨盘：

| Dial | Value | Reason |
| --- | ---: | --- |
| DESIGN_VARIANCE | 7 | 主界面需要漂浮层次、玻璃面板和内容形态差异，但 Feed 阅读仍要稳定 |
| MOTION_INTENSITY | 5 | hover 上浮、频道切换、搜索聚焦、Sheet 展开和点赞反馈需要细腻动效 |
| VISUAL_DENSITY | 6 | 信息密度适中，适合长时间阅读，也能体现真实内容生态 |

## Product Principles

1. 首屏即产品。打开 Web 端后直接看到可用的 Feed 流、导航、趋势栏和内容操作，不出现营销 hero。
2. 内容优先。玻璃材质服务于框架和操作层，标题、正文、评论和表单必须清晰可读。
3. 真实生态。Feed 中必须同时出现图文、长文摘要、图片组、视频预览和链接分享，不只摆放同一种卡片。
4. 玻璃克制。液态高光、边缘折射和景深用于提升层次，不用廉价彩色光效堆满页面。
5. 游客可浏览。未登录不阻断公开内容，只在写操作时触发登录引导。
6. 写操作即时反馈。点赞、收藏、关注、评论先给本地反馈，失败再回滚。
7. 发布可恢复。上传、草稿、校验失败、会话过期都不能让用户丢内容。
8. 接口诚实。只设计当前接口能支撑的能力，缺失接口明确标注为后续扩展。
9. 平台一致但不生硬统一。Web、iOS、Android 使用同一产品语言，但遵守各自平台交互习惯。

## High Fidelity UI Contract

Web 端高保真稿必须表现一个完整的信息流系统，而不是几张孤立卡片。第一屏至少包含：

- 顶部固定导航栏，含品牌 Logo、全局搜索、频道切换、通知、私信、用户头像和发布按钮。
- 左侧导航栏，含关注、推荐、热门、收藏、频道分类、话题标签和设置入口。
- 中央 Feed，至少展示 3 张不同内容类型的卡片，首屏可见图文、视频或长文摘要中的两类。
- 右侧辅助栏，含趋势话题、推荐关注、实时热榜、今日数据和社区活跃度中的至少 3 个模块。
- 当前频道高亮、搜索框聚焦态、通知角标、点赞或收藏选中态、趋势榜排名状态。

视觉上必须满足：

- 背景不是纯白，而是柔和多层渐变和模糊光影。
- 主要颜色是冷白、冰蓝、银灰、淡青，蓝色是主操作色，轻微紫色只能做环境光。
- 所有导航、按钮、标签、搜索框、侧边栏和辅助栏都使用统一玻璃材质，保持现在项目里的白色半透明面板感。
- Feed 卡片有 1px 半透明描边、内阴影、边缘高光和 hover 上浮。
- 文字层级清楚，标题、正文、元信息、辅助信息不能混成同一灰度。
- 图标为统一线性风格，按钮目标至少 40px，移动端至少 44px。

禁止：

- 不做营销首页、宣传 hero、价格区、客户 logo 墙。
- 不做普通后台管理模板。
- 不做深色赛博朋克。
- 不用大面积紫蓝霓虹渐变。
- 不用纯白卡片堆叠成普通 SaaS 仪表盘。
- 不伪造接口没有提供的精确计数。

## Visual System

### Theme

默认以浅色液态玻璃主题为主，深色主题作为后续兼容。Web 首屏不能是纯白平面，也不能是深色赛博朋克。页面背景延续当前项目的 `#eef2f6` 冷白底、蓝色径向环境光和白色半透明玻璃面板，所有内容面板悬浮在柔和的多层模糊光影之上。

主题在根节点统一切换，禁止单个页面随机反色。用户未设置时跟随系统偏好。

### Color Tokens

| Token | Light | Dark | Role |
| --- | --- | --- | --- |
| `--z-canvas` | `#EEF6F8` | `#071012` | 页面背景，冷白冰蓝底 |
| `--z-canvas-2` | `#E8F0F5` | `#101A20` | 背景二级色场 |
| `--z-surface` | `rgba(255, 255, 255, 0.72)` | `rgba(17, 27, 32, 0.72)` | Feed 卡片、内容面板 |
| `--z-surface-strong` | `rgba(255, 255, 255, 0.88)` | `rgba(20, 30, 36, 0.88)` | 表单、正文、评论 |
| `--z-surface-soft` | `rgba(232, 243, 246, 0.58)` | `rgba(34, 48, 56, 0.58)` | 骨架、次级信息块 |
| `--z-text` | `#10171A` | `#EEF7FA` | 主文字 |
| `--z-text-muted` | `#61737B` | `#9CB0B8` | 次级文字 |
| `--z-text-faint` | `#8AA0A8` | `#738990` | 元信息 |
| `--z-line` | `rgba(77, 113, 125, 0.18)` | `rgba(255, 255, 255, 0.11)` | 半透明描边 |
| `--z-line-strong` | `rgba(255, 255, 255, 0.72)` | `rgba(255, 255, 255, 0.18)` | 玻璃边缘高光 |
| `--z-accent` | `#1F53C9` | `#8FB3FF` | 发布按钮、选中态、关键操作 |
| `--z-accent-soft` | `rgba(31, 83, 201, 0.12)` | `rgba(143, 179, 255, 0.16)` | 频道高亮、标签底 |
| `--z-ice` | `#DDF3FA` | `#17313A` | 冰蓝环境光 |
| `--z-lilac` | `#D8D1FF` | `#37305F` | 轻微紫色点缀，只用于背景光 |
| `--z-danger` | `#C8423A` | `#FF8A7C` | 删除、失败 |
| `--z-warning` | `#9A650D` | `#F0C15A` | 上传等待、会话提醒 |

颜色策略：

- 主操作色使用蓝色，不使用紫蓝渐变按钮。
- 轻微紫色只能出现在远景环境光，不用于 CTA、激活态和标签。
- 背景由 3 到 5 层柔和径向色场构成，并固定在视口，不跟随滚动容器重绘。
- 内容图片不叠加装饰标签。
- danger 和 warning 只表达语义状态，不用于品牌装饰。

### Typography

| Purpose | Web | App |
| --- | --- | --- |
| Latin display | Hanken Grotesk | iOS SF Pro, Android Roboto |
| Chinese text | PingFang SC, Noto Sans SC | 平台系统中文字体 |
| Number | Inter or tabular numbers | tabular numbers |

字号建议：

| Token | Web | App | Usage |
| --- | --- | --- | --- |
| `display` | 40/48, weight 650 | 34/41, weight 700 | 空态、认证页、重点标题 |
| `title` | 24/32, weight 650 | 22/28, weight 700 | 页面标题 |
| `headline` | 18/26, weight 650 | 17/24, weight 650 | Feed 标题、详情标题 |
| `body` | 15/24, weight 400 | 16/24, weight 400 | 正文、评论 |
| `caption` | 13/18, weight 450 | 13/18, weight 450 | 时间、计数、说明 |
| `control` | 14/20, weight 600 | 15/20, weight 600 | 按钮、Tab |

排版规则：

- Feed 标题最多 2 行，详情标题最多 3 行。
- 正文最大阅读宽度 Web 为 680px。
- 按钮文字不换行。
- 数字计数使用 tabular numbers，避免点赞数变化导致抖动。

### Radius

| Token | Value | Usage |
| --- | ---: | --- |
| `radius-xs` | 8px | 输入框、图片角 |
| `radius-sm` | 12px | 小按钮、标签 |
| `radius-md` | 18px | Feed 卡片、表单块 |
| `radius-lg` | 28px | 玻璃面板、Sheet |
| `radius-pill` | 999px | 搜索框、Tab、主按钮 |

### Liquid Glass Material

Web 端液态毛玻璃是 Web 近似实现，不是 Apple 官方 Liquid Glass 包。视觉目标是半透明磨砂、液态高光、边缘折射、柔和阴影和轻微景深，当前项目更接近浅色 Material 3 玻璃界面。玻璃用于导航、工具、浮层和操作反馈，Feed 正文区域使用半透明强表面，避免文字被背景干扰。

| Layer | Usage | Visual Rule |
| --- | --- | --- |
| `glass-nav` | 顶部固定导航、左侧导航、移动底部 Tab | blur 24px, saturation 170%, 1px 亮边，内高光 |
| `glass-control` | 搜索框、频道切换、按钮、标签 | blur 18px, 胶囊形态，hover 时亮边增强 |
| `glass-card` | Feed 卡片、右侧趋势卡片 | blur 22px, 背景更实，边缘有折射高光和内阴影 |
| `glass-panel` | 登录面板、评论 Sheet、搜索浮层 | blur 30px, 阴影更深，适合覆盖式浮层 |
| `glass-popover` | Toast、菜单、确认框 | blur 18px, 背景最实，优先保证文字可读 |
| `solid-reader` | 长文正文、评论正文、表单输入区 | 近实体 surface，只保留轻微透明和内描边 |

材质细节：

- 每个玻璃面板必须有 1px 半透明描边。
- 顶部边缘使用轻微白色内高光，底部边缘使用更低透明度暗边。
- 面板阴影使用冷灰蓝，不使用纯黑重阴影。
- Feed 卡片 hover 时上浮 4px 到 6px，阴影扩散增加，边缘高光增强。
- 激活态按钮使用蓝色 accent tint 和内发光，不使用外部霓虹光。

降级规则：

- `prefers-reduced-transparency` 或设备性能不足时，所有 glass layer 变为 `rgba` 实体面。
- App 端 Android 低版本或低性能设备使用半透明 surface，不强制模糊。
- 内容图片背后可以有轻微动态色采样，但不能影响文本对比度。

### Motion

| Interaction | Motion |
| --- | --- |
| 页面进入 | opacity 0 -> 1, translateY 12px -> 0, 180ms |
| Feed 卡片出现 | 每项延迟 35ms，最多对首屏 8 项做 stagger |
| Feed 卡片 hover | translateY -5px, shadow spread +12px, 180ms |
| 搜索框聚焦 | scale 1.01, border opacity +24%, 背景透明度 +8% |
| 按钮按压 | scale 0.98 或 translateY 1px |
| 频道切换 | 胶囊选中态 layout transition, 180ms |
| Sheet 展开 | translateY 100% -> 0, opacity 0 -> 1, 240ms |
| 点赞 | 图标 scale 0.85 -> 1.08 -> 1，计数淡入 |
| 收藏 | icon fill 淡入，边缘高光短暂增强 |
| 通知角标 | scale 0.8 -> 1，120ms |
| 删除确认 | 背景 scrim 淡入，确认面板 scale 0.96 -> 1 |

禁止：

- 不监听 `window.scrollY` 写 React state。
- 不动画 height、width、top、left。
- 不做全站自定义鼠标。
- 不对每张 Feed 卡片加无限循环动画。

## Shared Information Architecture

| Area | Web Route | App Screen | Auth | Web Status | App Status |
| --- | --- | --- | --- | --- | --- |
| 启动和会话恢复 | `/` | Launch | Optional | `[x] 已完成` | `[ ] 未完成` |
| 登录 | `/login` | Auth stack, login | No | `[x] 已完成` | `[ ] 未完成` |
| 注册 | `/register` | Auth stack, register | No | `[x] 已完成` | `[ ] 未完成` |
| 推荐流 | `/home` | Home tab | Optional | `[~] 部分完成` | `[ ] 未完成` |
| 关注流 | `/following` | Follow tab | Optional, empty state requires login | `[~] 部分完成` | `[ ] 未完成` |
| 内容详情 | `/content/:contentId` | Content detail | Optional | `[~] 部分完成` | `[ ] 未完成` |
| 搜索 | `/search` | Search tab | Optional | `[~] 部分完成` | `[ ] 未完成` |
| 发布入口 | `/compose` | Compose tab | Required | `[~] 部分完成` | `[ ] 未完成` |
| 编辑内容 | `/content/:contentId/edit` | Edit content | Required | `[x] 已完成` | `[ ] 未完成` |
| 我的主页 | `/me` | Me tab | Required | `[~] 部分完成` | `[ ] 未完成` |
| 用户主页 | `/user/:userId` | User profile | Optional | `[~] 部分完成` | `[ ] 未完成` |
| 编辑资料 | `/me/edit` | Edit profile | Required | `[~] 部分完成` | `[ ] 未完成` |
| 粉丝列表 | `/user/:userId/followers` | Followers list | Optional | `[ ] 未完成` | `[ ] 未完成` |
| 收藏列表 | `/user/:userId/favorites` | Favorites list | Optional | `[ ] 未完成` | `[ ] 未完成` |
| 发布列表 | `/user/:userId/posts` | Published list | Optional | `[ ] 未完成` | `[ ] 未完成` |
| 设置 | `/settings` | Settings | Required | `[~] 部分完成` | `[ ] 未完成` |
| 404 | `*` | Not found | Optional | `[x] 已完成` | `[ ] 未完成` |

## Shared Components

### AppShell `[~] 部分完成`

职责：

- 控制 Web 桌面顶部固定导航、左侧导航、中央信息流、右侧辅助栏，以及移动端底部 Tab。
- 读取当前登录态，决定是否显示头像、登录按钮和写操作入口。
- 提供全局 Toast、Auth Sheet、网络离线提示。

Web 桌面：

- 顶部固定导航 72px：品牌 Logo、全局搜索、频道切换、通知、私信、用户头像、发布按钮。
- 左栏 248px：`glass-nav`，包含关注、推荐、热门、收藏、频道分类、话题标签、设置入口。
- 中栏 720px：多类型 Feed 流或详情主内容。
- 右栏 336px：趋势话题、推荐关注、实时热榜、今日数据、社区活跃度。
- 所有栏位都有轻微漂浮层次，主 Feed 不被侧栏玻璃覆盖。

Web 移动：

- 顶部 `glass-nav` 简化导航，包含品牌、搜索入口、通知。
- 底部 `glass-nav` Tab Bar。
- Feed 单列显示，页面内容有底部安全留白，避免被 Tab Bar 挡住。

App：

- 原生 Tab Bar。
- iOS 使用 large title 和 scroll edge material。
- Android 使用 edge-to-edge 布局，底栏遵守手势导航安全区。

### TopNavigation `[~] 部分完成`

Web 桌面顶部固定导航必须完整呈现真实产品操作，而不是营销站导航。

结构：

- 左侧：zfeed 品牌 Logo 和当前 workspace 名称。
- 中左：全局搜索框，支持聚焦态、快捷键提示和清除按钮。
- 中部：频道切换，推荐、关注、热门、长文、视频。
- 右侧：通知按钮、私信按钮、用户头像、发布按钮。

视觉：

- 整体为 `glass-nav`，高度 72px，顶部固定，背景模糊但文字清晰。
- 搜索框使用 `glass-control` 胶囊，聚焦时边缘高光增强，背景透明度略提高。
- 当前频道用 `--z-accent-soft` 胶囊高亮，文字使用 `--z-accent`。
- 通知和私信使用统一线性图标，通知按钮右上显示小角标。
- 发布按钮突出但不突兀，使用当前项目的蓝色 accent 胶囊和轻内高光。

交互：

- 搜索框 focus 后进入搜索建议浮层。
- 频道切换使用 layout transition。
- 通知角标只在有未读时显示。
- 发布按钮 hover 上浮 2px，active 轻微下压。

### LeftNavigation `[~] 部分完成`

左侧导航承担内容分发系统的频道结构，不做后台菜单感。

分组：

- 主频道：推荐、关注、热门、收藏。
- 内容频道：长文、视频、图片、链接。
- 话题标签：AI、后端架构、产品设计、开源、趋势观察。
- 底部：设置、主题切换、登录或当前用户入口。

视觉：

- 面板使用 `glass-nav`，宽度 248px，圆角 28px。
- 分组标题使用小号 muted 文本，不使用大写装饰标签。
- 当前导航项使用半透明胶囊高亮和细腻内阴影。
- 图标使用同一线性图标族，尺寸 20px，stroke 1.75。
- 话题标签使用轻玻璃 chip，最多展示 5 个，更多进入话题页。

交互：

- hover 时导航项背景透明度提高，图标和文字轻微提亮。
- 当前项不可再次触发重复请求。
- 收藏入口未登录时仍可见，点击后触发 AuthSheet。

### RightAssistRail `[~] 部分完成`

右侧辅助栏必须体现趋势资讯和社区活跃度，不能只是空白广告位。

模块：

- 趋势话题：话题名、热度变化、内容数量。
- 推荐关注：头像、昵称、简介、关注按钮。
- 实时热榜：排名、标题、热度状态。
- 今日数据：新增内容、活跃作者、互动次数。
- 社区活跃度：轻量曲线或时间段活跃提示。

视觉：

- 每个模块使用 `glass-card`，卡片之间保持 16px 间距。
- 趋势榜排名使用 1 到 10 的 tabular numbers。
- 上升、下降、持平状态用小箭头和语义色，不使用夸张进度条。
- 推荐关注头像使用真实图片或明确占位，不使用通用头像图标。

交互：

- 趋势话题点击进入搜索结果。
- 推荐关注按钮可乐观更新。
- 热榜条目 hover 时背景轻微提亮，点击进入详情。
- 今日数据只展示当前产品可解释的数据，不编造精准业务指标。

### FeedCard `[~] 部分完成`

字段来源：

- `content_id`
- `content_type`
- `author_id`
- `author_name`
- `author_avatar`
- `title`
- `cover_url`
- `published_at`
- `is_liked`
- `like_count`
- 评论数、收藏数和分享数当前可在详情或互动信息补偿后展示，Feed 列表没有真实字段时使用隐藏或延迟加载，不伪造精确数字。

视觉：

- 卡片使用 `glass-card`，背景半透明但正文区域必须清晰可读。
- 顶部作者行包含头像、昵称、认证标识、发布时间和更多菜单。
- 认证标识为小型蓝色线性徽标，不使用醒目的彩色大 badge。
- 标题最多 2 行，正文摘要最多 4 行。
- 内容预览区根据类型变化：图文、长文摘要、图片组、视频预览、链接分享。
- 话题标签位于正文和操作条之间，使用 `glass-control` chip。
- 底部操作条包含点赞、评论、转发、收藏、分享。
- 激活态按钮使用填充图标或 accent 文本，未激活态保持线性图标。
- 卡片之间保持 18px 到 22px 垂直间距，适合长时间阅读。

内容类型：

| Type | Preview |
| --- | --- |
| 图文 | 单张 16:10 图片或无图摘要 |
| 长文摘要 | 标题、摘要、阅读时间、文章封面可选 |
| 图片组 | 2 到 4 张图片网格，最后一张可显示数量叠层 |
| 视频预览 | 16:9 封面、播放按钮、时长 |
| 链接分享 | 域名、标题、摘要、站点缩略图 |

交互：

- 点击卡片主体进入详情。
- 点击作者进入用户主页。
- 点赞走乐观更新。失败回滚，并提示“点赞失败，请重试”。
- 收藏走乐观更新。失败回滚，并提示“收藏失败，请重试”。
- 评论按钮进入详情并打开评论。
- 分享按钮调用 Web Clipboard 或 App 系统分享。
- 未登录点赞弹出登录引导，登录成功后继续点赞。
- 图片加载失败显示内容类型占位，不隐藏整张卡片。
- hover 时卡片上浮 4px 到 6px，玻璃边缘高光增强，阴影更柔。
- active 时卡片轻微下压，不能产生布局跳动。

### ContentActionBar `[~] 部分完成`

用于详情页底部或桌面右侧 sticky 操作区。

操作：

- 点赞或取消点赞。
- 收藏或取消收藏。
- 打开评论。
- 分享链接，Web 使用 Clipboard，App 使用系统分享。
- 作者关注或取消关注，只有非本人详情显示。

状态：

- loading: 禁用重复点击，但保留按钮位置。
- error: 回滚按钮视觉状态。
- auth required: 打开登录 Sheet。

### CommentThread `[~] 部分完成`

结构：

- 一级评论列表。
- 每条评论可展开回复。
- 回复区显示父评论摘要、回复对象和输入框。

交互：

- 评论输入上限 255 字。
- 发送后插入 pending 评论，接口成功后替换真实 `comment_id`。
- 失败时 pending 项显示“发送失败，重试”。
- 删除评论需要确认。
- 回复列表分页使用 `cursor`，不一次拉完。

### PublishComposer `[~] 部分完成`

模式：

- article publish
- video publish
- article edit
- video edit

共同字段：

- title
- description
- visibility
- cover

文章字段：

- content

视频字段：

- video_url
- cover_url
- duration

交互：

- 标题必填，1 到 100 字。
- 文章正文必填。
- 视频地址必填，封面必填。
- 上传凭证先调 `/v1/content/upload-credentials`。
- 上传不阻塞正文编辑。
- 发布中按钮进入 loading，防止重复提交。
- 会话过期时打开登录 Sheet，登录后继续保留草稿。

### AuthSheet `[~] 部分完成`

触发场景：

- 点赞、收藏、关注、评论、发布、编辑、删除、资料保存。

交互：

- Web 桌面居中 glass panel。
- Web 移动和 App 使用底部 Sheet。
- 登录成功后回到原动作，不跳走。
- 关闭登录 Sheet 后保留当前页面状态。

### State Components `[~] 部分完成`

| State | Web | App |
| --- | --- | --- |
| loading | 与最终布局一致的骨架屏 | 原生骨架屏，保留导航 |
| empty | 明确下一步按钮 | 空态插画可用简单品牌形状，不用复杂 SVG |
| error | 页面内错误面板或 Toast | Inline error + Toast |
| offline | 顶部小条提示，保留缓存内容 | 原生网络状态提示 |
| unauthorized | Auth Sheet | Auth Sheet |

## Web End Design

### Web Global Layout

#### Desktop

适用宽度：`>= 1024px`。

布局：

```text
fixed glass top nav
left glass rail | central feed | right assist rail
248px           | 720px        | 336px
```

行为：

- 顶部导航固定在视口顶部，高度 72px，横向贯穿主界面。
- 左侧导航从顶部导航下方开始 sticky，宽 248px，高度为剩余视口高度。
- 中央 Feed 宽 720px，作为唯一主滚动阅读区域。
- 右侧辅助栏宽 336px，sticky，展示趋势和社区状态。
- 页面最大内容宽度 1400px 到 1480px，整体居中。
- 页面背景使用固定柔和色场和模糊光影，不跟随滚动容器重绘。
- 玻璃侧栏、顶栏和右栏不覆盖 Feed 文本，不制造阅读遮挡。

首屏必须可见：

- 顶部导航完整可见。
- 左侧导航至少可见主频道和部分话题标签。
- 中央 Feed 至少可见 2 张完整卡片和第 3 张的开头。
- 右侧辅助栏至少可见趋势话题、推荐关注和实时热榜。
- Feed 中至少有一张卡片处于点赞或收藏选中态。
- 顶部搜索框展示 focus 或可聚焦视觉状态。
- 通知按钮展示未读角标。

背景：

- 冷白到冰蓝的基础渐变。
- 左上角淡青色模糊光影。
- 右上角轻微紫色远景光影。
- 中央 Feed 背后有非常轻的银灰景深，不形成脏灰。
- 所有背景光影固定在页面底层，不能随 Feed 滚动闪烁。

#### Tablet

适用宽度：`768px` 到 `1023px`。

- 顶部导航保留品牌、搜索入口、通知、发布按钮。
- 左侧栏缩成 72px icon rail。
- 右侧辅助栏隐藏。
- 中央 Feed 最大 720px 居中。
- 频道切换移到 Feed 顶部的玻璃分段控件。

#### Mobile Web

适用宽度：`< 768px`。

- 顶部 56px 简化 glass nav，包含品牌、搜索入口、通知。
- 底部 64px glass tab bar，包含首页、关注、发布、搜索、我的。
- Feed 单列显示。
- 页面左右 padding 16px。
- 主按钮和输入目标至少 44px 高。
- 右侧趋势内容移到 Feed 中的“趋势快照”模块，位于前 6 张卡片之后。
- 顶部频道切换使用横向滚动 chip，当前频道始终高亮。

### Web Launch `[x] 已完成`

目标：

- 恢复会话并尽快进入可浏览状态。

流程：

1. 从本地安全存储读取 token。
2. 若有 token，调用 `GET /v1/users/me`。
3. 成功后进入 `/home` 的登录态。
4. 失败后清理 token，进入游客态 `/home`。
5. 无 token 直接进入游客态 `/home`。

界面：

- 背景为 `--z-canvas`。
- 中央 zfeed wordmark 和 3 条 Feed 骨架。
- 不显示长时间 spinner。

### Web Login `[x] 已完成`

Route: `/login`

Layout:

- 桌面端居中 420px `glass-panel`。
- 移动端从底部进入全屏 Sheet。
- 背景保留模糊的产品色场，不使用宣传 hero。

Fields:

- 手机号。
- 密码。

Actions:

- 登录。
- 去注册。

API:

- `POST /v1/login`

Interaction:

- 手机号和密码为空时，字段下方显示错误。
- 提交中禁用按钮，按钮文字显示“登录中”。
- 登录成功保存 `token`、`expired_at`、基础用户信息。
- 登录失败显示“手机号或密码不正确”。
- 如果从写操作触发登录，成功后自动继续原动作。

### Web Register `[x] 已完成`

Route: `/register`

Fields:

- 手机号。
- 密码。
- 昵称，可选。
- 头像 URL，可选。
- 简介，可选。
- 邮箱，可选。

API:

- `POST /v1/users`

Interaction:

- 昵称最多 64 字。
- 简介最多 255 字。
- 邮箱格式错误就近提示。
- 注册成功保存 token 并进入 `/me`。
- 如果用户从发布入口进入注册，成功后进入 `/compose`。

### Web Home Recommend Feed `[~] 部分完成`

Route: `/home`

API:

- `POST /v1/feed/recommend`

Request:

- `cursor`
- `page_size`
- `snapshot_id`

Layout:

- 首屏直接显示完整主界面，不出现营销 hero。
- 顶部固定导航使用 `TopNavigation`。
- 左侧使用 `LeftNavigation`，当前频道为“推荐”。
- 中央主列顶部使用玻璃频道切换，推荐、关注、热门、长文、视频。
- 中央 Feed 由多类型 `FeedCard` 组成，首屏至少混排图文、视频预览、长文摘要。
- 右侧使用 `RightAssistRail`，显示趋势话题、推荐关注、实时热榜和今日数据。

首屏内容建议：

1. 第一张卡片：长文摘要，展示创作者头像、认证标识、标题、摘要、话题标签和互动状态。
2. 第二张卡片：图片组，展示 2 到 4 张图片网格和收藏选中态。
3. 第三张卡片：视频预览，展示播放按钮、时长和点赞选中态。
4. 第四张卡片：链接分享，展示站点域名、标题、摘要和缩略图。

频道状态：

- 当前频道“推荐”使用 accent 胶囊高亮。
- hover 频道只提高玻璃透明度和文字亮度。
- 切换频道时保留当前滚动位置到本地内存，返回频道时恢复。

Interaction:

- 首屏请求 `cursor=""`，`page_size=20`。
- 返回 `snapshot_id` 后保存在当前 Feed session。
- 滚动到底加载 `next_cursor`。
- `has_more=false` 显示轻量结束提示。
- 下拉刷新清空 cursor 和 snapshot，重新请求首屏。
- 刷新时保留旧列表直到新数据回来，避免闪白。
- 卡片 hover 上浮 4px 到 6px，触控设备不启用 hover 残留状态。
- 搜索框聚焦打开搜索建议浮层，不离开首页。
- 通知和私信点击打开 `glass-popover`，不覆盖主 Feed 阅读上下文。

States:

- loading: 5 张 FeedCard 骨架。
- empty: “暂时没有内容”，显示“刷新”按钮。
- error: “推荐流加载失败”，显示“重试”按钮。

### Web Follow Feed `[~] 部分完成`

Route: `/following`

API:

- `POST /v1/feed/follow`

Interaction:

- 游客可进入，但显示登录引导空态。
- 已登录用户请求关注流。
- 若返回空列表，显示“关注一些作者后，这里会出现他们的新内容”，提供“去搜索”按钮。
- 下拉刷新和分页与推荐流一致。

### Web Feed Card Actions `[x] 已完成`

Like:

- 已登录：调用 `POST /v1/interaction/like` 或 `POST /v1/interaction/unlike`。
- 未登录：弹 AuthSheet。
- 参数 `scene` 使用内容场景常量，前端统一封装。

Favorite:

- 已登录：调用 `POST /v1/interaction/favorite` 或 `DELETE /v1/interaction/favorite`。
- Feed 列表中如接口返回缺少 `is_favorited`，卡片默认不展示收藏状态，只在详情页展示真实状态。

Comment:

- 点击进入详情并打开评论区域。
- 移动端详情页打开评论 Sheet。

### Web Content Detail `[~] 部分完成`

Route: `/content/:contentId`

API:

- `POST /v1/content/detail`
- `POST /v1/interaction/comment/list`
- `POST /v1/interaction/comment/reply/list`

Layout:

- 桌面端主栏显示内容，右栏 sticky 显示作者和操作。
- 移动端顶部显示封面或视频，底部 sticky glass action bar。

Article:

- 封面可选。
- 标题。
- 作者行。
- 描述。
- 正文。
- 评论入口。

Video:

- 视频播放器优先占据首屏上半区。
- 标题和作者信息在播放器下方。
- 视频时长使用 tabular numbers。

Interaction:

- 点击返回回到上一页，若无 history 回首页。
- 作者按钮进入 `/user/:authorId`。
- 关注按钮调用 `POST /v1/interaction/followings` 或 `DELETE /v1/interaction/followings`。
- 删除内容仅 owner 可见，点击后弹确认框，调用 `DELETE /v1/content/:content_id`。
- 编辑内容仅 owner 可见，进入编辑页。

States:

- 内容不存在或已删除：显示 not found content state。
- 网络失败：保留顶部导航，主区显示重试。
- 评论失败不影响正文显示。

### Web Comment Panel `[~] 部分完成`

Desktop:

- 在详情页下方直接显示评论列表。
- 右侧 rail 可显示评论输入快捷入口。

Mobile:

- 点击评论打开底部 Sheet。
- Sheet 高度默认 72dvh，可上拉到 92dvh。
- 输入框固定在 Sheet 底部，键盘弹出时跟随安全区。

Interaction:

- 一级评论分页 `cursor=0` 开始。
- 回复评论时带 `parent_id`、`root_id`、`reply_to_user_id`。
- 删除评论后该项淡出，失败则恢复。
- 回复数大于 0 时显示“查看回复”。

### Web Search Home `[~] 部分完成`

Route: `/search`

Layout:

- 桌面端主栏顶部是 56px `glass-panel` 搜索框。
- 下方是最近搜索、搜索建议和空态内容。
- 移动端搜索框固定顶部，自动 focus。

Interaction:

- 输入 300ms debounce。
- query 长度 1 到 50。
- 按 Enter 或点击搜索进入结果状态。
- 清除按钮清空 query 和结果。

### Web Search Results `[~] 部分完成`

APIs:

- `POST /v1/search/contents`
- `POST /v1/search/users`

Segments:

- 内容。
- 用户。

Sort mode:

- latest。
- relevance。
- hybrid。

Interaction:

- 切换 Segment 保留 query。
- 切换 mode 重置分页。
- `latest` 可使用 `cursor`。
- `relevance` 和 `hybrid` 使用 `page_token` 和 `snapshot_id`。
- 用户结果点击进入用户主页。
- 内容结果点击进入详情。
- 搜索为空不请求接口。

States:

- empty: “没有找到相关结果”，显示更换关键词建议。
- error: “搜索失败，请重试”。
- loading: 搜索结果骨架，用户和内容形状不同。

### Web Compose Entry `[~] 部分完成`

Route: `/compose`

Auth:

- Required。
- 未登录打开 AuthSheet，成功后继续进入发布页。

Layout:

- 桌面端主编辑区 760px，右侧发布状态 rail。
- 移动端全屏编辑器，底部操作栏。

Tabs:

- 文章。
- 视频。

### Web Publish Article `[~] 部分完成`

API:

- `POST /v1/content/upload-credentials`
- `POST /v1/content/article/publish`

Fields:

- 标题，必填，1 到 100 字。
- 简介，1 到 255 字。
- 封面，可选，URL。
- 正文，必填，1 到 1000000 字。
- 可见性，必填。

Interaction:

- 标题和正文为空时禁止发布。
- 简介为空允许保存，但提交时不传空字符串。
- 封面上传先拿凭证，再直传对象存储。
- 上传进行中可继续编辑正文。
- 发布中禁用发布按钮。
- 发布成功进入内容详情页。
- 发布失败保留全部输入。
- 会话过期弹登录 Sheet，登录后继续发布。

Draft:

- Web 本地保存草稿到浏览器存储。
- 草稿 key 包含 userId 和 content type。
- 发布成功清理草稿。

### Web Publish Video `[ ] 未完成`

API:

- `POST /v1/content/upload-credentials`
- `POST /v1/content/video/publish`

Fields:

- 标题，必填，1 到 100 字。
- 简介，1 到 500 字。
- 视频 URL，必填。
- 封面 URL，必填。
- 时长，1 到 7200 秒。
- 可见性，必填。

Interaction:

- 视频和封面都完成后才允许发布。
- 上传队列显示文件名、进度、剩余时间、取消按钮。
- 时长可由上传后解析得到，也允许用户手动修正。
- 视频 URL 校验失败时字段下方提示。
- 发布成功进入视频详情。

### Web Edit Content `[x] 已完成`

Routes:

- `/content/:contentId/edit`

APIs:

- `PUT /v1/content/article/:content_id`
- `PUT /v1/content/video/:content_id`

Interaction:

- 进入编辑页先加载详情。
- 根据 `content_type` 自动进入文章或视频模式。
- 只提交变化字段。
- 保存成功回到详情。
- 保存失败保留输入。
- 删除入口在更多菜单中，不放在主按钮旁。

### Web User Profile `[~] 部分完成`

Route:

- `/user/:userId`

APIs:

- `GET /v1/user/profile/:userId`
- `POST /v1/feed/user/publish`
- `POST /v1/feed/user/favorite`
- `POST /v1/user/followers`

Layout:

- 顶部 ProfileHeader 使用轻玻璃叠层。
- 头像、昵称、简介、关注按钮。
- 统计：关注数、粉丝数、获赞、获收藏、内容数。
- Tabs：发布、收藏、粉丝。

Interaction:

- 非本人显示关注按钮。
- 本人显示编辑资料按钮。
- 发布 Tab 请求用户发布流。
- 收藏 Tab 请求用户收藏流。
- 粉丝 Tab 请求粉丝列表。
- 关注数仅展示，不进入列表，因为当前接口未暴露关注列表。

### Web Me `[~] 部分完成`

Route:

- `/me`

API:

- `GET /v1/users/me`

Interaction:

- 未登录进入登录页或 AuthSheet。
- 展示当前用户资料和统计。
- 进入编辑资料。
- 进入我的发布、我的收藏、我的粉丝。
- 退出登录调用 `POST /v1/logout`。

### Web Edit Profile `[~] 部分完成`

Route:

- `/me/edit`

APIs:

- `POST /v1/users/avatar/upload`
- `PUT /v1/users/me/profile`

Fields:

- 头像。
- 昵称。
- 简介。
- 性别。
- 邮箱。
- 生日。

Interaction:

- 头像上传失败不影响其他字段保存。
- 昵称最多 64 字。
- 简介最多 255 字。
- 邮箱格式错误就近提示。
- 保存成功回到 `/me`。
- 退出前如有未保存修改，显示确认。

### Web Followers List `[ ] 未完成`

Route:

- `/user/:userId/followers`

API:

- `POST /v1/user/followers`

Layout:

- 用户列表行，头像、昵称、简介、关注按钮。

Interaction:

- 游客可看粉丝列表，但关注按钮需要登录。
- `page_size` 20。
- 点击用户行进入主页。
- 关注按钮乐观更新。

### Web Settings `[~] 部分完成`

Route:

- `/settings`

Content:

- 账号信息。
- 当前登录状态。
- 退出登录。
- 主题模式：跟随系统、浅色、深色。
- 减少动效说明，实际跟随系统设置。

Interaction:

- 退出登录二次确认。
- 退出成功清空 token，回到游客态首页。

### Web Not Found `[x] 已完成`

Route:

- `*`

Layout:

- 简洁 `glass-panel`。
- 文案说明页面不存在。
- 操作：回首页、去搜索。

### Web Offline `[ ] 未完成`

Behavior:

- 顶部显示离线提示。
- Feed 保留已有缓存。
- 写操作禁用并提示“网络恢复后再试”。
- 恢复网络后允许用户手动刷新。

## App End Design

### App Architecture `[ ] 未完成`

App 端推荐采用原生体验实现，同步复用 `front-api`：

- iOS: SwiftUI 或 UIKit。
- Android: Jetpack Compose。
- 跨端可选 React Native 或 Flutter，但必须映射本文档 token，不直接复用 Web CSS。

App 端与 Web 端共享：

- API client。
- token 存储模型。
- Feed pagination model。
- optimistic update model。
- 文案和错误分类。
- 设计 token。

App 端差异：

- 使用原生导航栈。
- 使用系统返回手势。
- 使用系统分享。
- 使用原生 haptic feedback。
- 使用安全区和键盘避让。
- 使用平台原生模糊或实体降级。

### App Navigation `[ ] 未完成`

移动端首屏同样是产品界面，不是 Web 首屏的缩小截图。App 使用简化顶部导航、底部 Tab Bar 和单列 Feed，把趋势、搜索和发布入口折叠进更适合触控的层级。

顶部导航：

- 高度跟随平台安全区，内容区最小 56px。
- 左侧显示 zfeed wordmark 或当前频道标题。
- 中间可显示玻璃质感搜索入口，宽度不足时只保留搜索图标。
- 右侧显示通知按钮和头像入口，通知有未读角标。
- 滚动到顶部时透明度更轻，向下滚动后变为更实的 glass material。

底部 Tab:

1. 首页。
2. 关注。
3. 发布。
4. 搜索。
5. 我的。

规则：

- 发布 Tab 位于中间，使用 accent 胶囊按钮。
- 当前 Tab 使用 accent tint、细腻内高光和轻 haptic，不使用高饱和霓虹效果。
- 未登录点击发布，先打开 Auth Sheet。
- 内容详情从任意 Tab push 进入导航栈。
- 评论使用 bottom sheet，不作为独立 Tab。
- 搜索结果在 Search stack 内 push。
- 底部 Tab 使用原生模糊或半透明实体降级，始终避开系统手势安全区。

### App Launch `[ ] 未完成`

Flow:

1. 读取安全存储 token。
2. 有 token 时请求 `GET /v1/users/me`。
3. 成功进入 Home。
4. 失败清 token，进入游客 Home。

Visual:

- 启动屏使用 zfeed wordmark。
- 主界面挂载后显示 Feed 骨架。
- 不在启动屏请求所有数据。

### App Auth `[ ] 未完成`

Screens:

- Login。
- Register。

Presentation:

- 从写操作触发时用 bottom sheet。
- 从 Me tab 触发时可用 full screen auth stack。

Interaction:

- 键盘弹出时按钮保持可见。
- 登录成功后关闭 Sheet 并恢复原动作。
- 注册成功后进入 Me 或原目标页面。
- 错误提示位于字段下方，不使用只在顶部出现的 Toast。

### App Home Feed `[ ] 未完成`

Screen:

- Top area: 简化 glass nav，包含 zfeed、搜索入口、通知角标。
- Channel chips: 推荐、关注、热门、长文、视频，横向滚动，当前频道高亮。
- Feed list: 原生单列信息流，混排图文、长文摘要、图片组、视频预览和链接分享。
- Trend snapshot: 前 6 张 Feed 后插入一张轻量趋势快照，展示 3 个话题和 3 条实时热榜。
- Bottom Tab: 首页、关注、发布、搜索、我的，发布入口突出但不遮挡阅读。

首屏内容：

- 第一张卡片建议为长文摘要，展示头像、认证标识、标题、摘要、话题标签和评论入口。
- 第二张卡片建议为图片组，展示 2 到 4 张图片和收藏选中态。
- 第三张卡片建议为视频预览，展示播放按钮、时长和点赞选中态。
- 第四张卡片建议为链接分享，展示域名、标题、摘要和缩略图。

Interaction:

- 下拉刷新。
- 滚动到底自动加载更多。
- Feed 卡片点击进入 detail。
- 横向滑动不承担主要操作，避免误触。
- 点赞触发轻 haptic。
- 收藏触发 selection haptic，失败时回滚状态并显示 inline toast。
- 搜索入口 tap 后 push Search 或打开原生搜索层。
- 频道切换保留各频道滚动位置。

State:

- 首屏骨架。
- 弱网显示缓存内容和顶部网络状态。
- 推荐为空显示刷新按钮。
- 未登录仍可浏览推荐、热门、搜索和详情，写操作触发 Auth Sheet。

### App Follow Feed `[ ] 未完成`

Screen:

- Follow Tab。

Interaction:

- 未登录显示登录引导。
- 已登录请求 `/v1/feed/follow`。
- 空态引导去 Search Tab 找作者。
- 下拉刷新和分页与 Home 相同。

### App Feed Card `[ ] 未完成`

Visual:

- 卡片使用平台 glass material 或半透明实体降级，必须保留 1px 亮边、内高光和柔和阴影。
- 正文区域使用更实的 `solid-reader`，避免毛玻璃影响阅读。
- 作者行包含头像、昵称、认证标识、发布时间和更多菜单。
- 标题最多 2 行，摘要最多 4 行。
- 预览区根据类型变化：图文、长文摘要、图片组、视频预览、链接分享。
- 图片和视频封面有轻内描边，视频预览包含播放按钮和时长。
- 话题标签使用轻玻璃 chip，最多展示 3 个。
- 操作按钮放在卡片底部，包含点赞、评论、转发、收藏、分享。
- 点赞和收藏选中态使用 accent tint 和填充图标，未选中态使用线性图标。
- 按钮目标不小于 44x44。
- 卡片之间保持 14px 到 18px 间距，适合单手滚动和长时间阅读。

Gestures:

- Tap card: push detail。
- Tap avatar/name: push user profile。
- Tap like: optimistic update。
- Tap favorite: optimistic update。
- Tap comment: push detail and focus comment entry, or open comment sheet from detail。
- Tap share: open system share sheet。
- Long press card: 打开更多菜单，可分享或举报占位。举报当前无接口，默认不显示提交入口。
- Press down: 卡片轻微 scale 0.99，释放后恢复。
- Scroll: 卡片不做持续视差动画，保证列表性能。

### App Content Detail `[ ] 未完成`

Article:

- 顶部透明导航栏随滚动变为 glass material。
- 内容区为实体 surface。
- 底部固定 action bar。

Video:

- 顶部播放器沉浸显示。
- 竖屏下播放器 16:9。
- 横屏进入系统全屏播放。
- 底部 action bar 可自动收起，点击屏幕恢复。

Interaction:

- 系统返回手势回上一页。
- 分享调用系统分享。
- 评论按钮打开 bottom sheet。
- 作者关注在 action bar 或作者区展示。

### App Comments Sheet `[ ] 未完成`

Presentation:

- bottom sheet。
- 默认高度 72%。
- 可拖到 92%。
- 下滑关闭。

Interaction:

- 输入框固定底部。
- 键盘弹出时 sheet 重新布局。
- 发送中评论显示 pending。
- 失败评论显示重试。
- 回复点击后输入框显示回复对象。
- 删除评论通过 action sheet 确认。

### App Search `[ ] 未完成`

Screen:

- Search Tab 顶部原生搜索框。
- 最近搜索。
- 输入后展示内容和用户分段。

Interaction:

- 输入 debounce 300ms。
- 提交搜索后显示结果列表。
- Segmented control 切换内容和用户。
- Mode 选择放在右上 filter sheet。
- 搜索结果点击进入 detail 或 profile。

State:

- 空 query 不请求。
- 无结果显示改写关键词建议。
- 搜索失败显示重试按钮。

### App Compose `[ ] 未完成`

Screen:

- Compose Tab。
- 顶部选择文章或视频。
- 默认打开文章发布。

Article:

- 标题输入。
- 简介输入。
- 正文编辑器。
- 封面上传。
- 可见性选择。

Video:

- 视频选择或 URL 输入。
- 封面选择。
- 标题。
- 简介。
- 时长。
- 可见性。

Interaction:

- 草稿自动保存到本地。
- 发布前统一校验。
- 上传进度固定在底部或顶部 status area。
- 上传失败可重试。
- 发布成功清草稿并 push detail。
- 关闭页面时如有草稿，询问保留或放弃。

### App Edit Content `[ ] 未完成`

Entry:

- 详情页 owner 菜单。
- 我的发布列表。

Interaction:

- 加载详情后进入对应 composer 模式。
- 保存按钮替代发布按钮。
- 只提交变化字段。
- 保存成功回详情。

### App Profile `[ ] 未完成`

Screen:

- Profile header。
- 头像、昵称、简介。
- 统计。
- Follow button or edit profile button。
- Tabs: 发布, 收藏, 粉丝。

Interaction:

- 统计区不全部可点。当前只有粉丝列表可进入。
- 关注数只是展示，直到接口补齐。
- 粉丝列表 push 新屏。
- 发布和收藏是 feed list。

### App Me `[ ] 未完成`

Screen:

- Me Tab。
- 登录态显示资料和统计。
- 未登录显示登录引导。

Actions:

- 编辑资料。
- 我的发布。
- 我的收藏。
- 我的粉丝。
- 设置。
- 退出登录。

### App Edit Profile `[ ] 未完成`

Interaction:

- 头像使用系统图片选择器。
- 上传头像调用 `/v1/users/avatar/upload`。
- 保存资料调用 `/v1/users/me/profile`。
- 表单错误 inline 显示。
- 未保存退出时二次确认。

### App Followers List `[ ] 未完成`

Interaction:

- 分页加载。
- 用户行点击进入 profile。
- 关注按钮乐观更新。
- 未登录关注弹 Auth Sheet。

### App Settings `[ ] 未完成`

Content:

- 账号信息。
- 主题模式。
- 清理本地缓存。
- 退出登录。

Interaction:

- 清理缓存不删除草稿，除非用户单独确认。
- 退出登录清 token、用户信息和接口缓存，保留主题设置。

### App Native Details `[ ] 未完成`

iOS:

- 使用 safe area 管理顶部和底部。
- Navigation bar 在滚动边缘使用透明材质，滚动后变为模糊材质。
- Bottom sheet 使用 detents。
- 轻操作使用 selection haptic，成功使用 success haptic，失败使用 warning haptic。

Android:

- Edge-to-edge 布局。
- Top app bar 和 bottom bar 使用半透明 surface。
- 可用 RenderEffect 时启用 blur，否则使用实体降级。
- Back gesture 必须关闭 Sheet 或返回上一层，不直接退出 App。

## API Mapping

接口源文件以 `app/front/doc/front.api` 为总入口，下面各节的路径分别对应：

- `base/base.api`
- `user/user.api`
- `content/content.api`
- `feed/feed.api`
- `interaction/interaction.api`
- `search/search.api`

### User

| UI Action | Endpoint | Notes | Web Status |
| --- | --- | --- | --- |
| 注册 | `POST /v1/users` | 成功返回 token | `[x] 已完成` |
| 登录 | `POST /v1/login` | 保存 token | `[x] 已完成` |
| 登出 | `POST /v1/logout` | 需要 token | `[~] 部分完成，本地退出已完成，后端登出未调用` |
| 当前用户 | `GET /v1/users/me` | 会话恢复 | `[x] 已完成` |
| 用户主页 | `GET /v1/user/profile/:userId` | Optional login | `[x] 已完成` |
| 更新资料 | `PUT /v1/users/me/profile` | 需要 token | `[x] 已完成` |
| 上传头像 | `POST /v1/users/avatar/upload` | multipart | `[ ] 未完成` |
| 粉丝列表 | `POST /v1/user/followers` | Optional login | `[ ] 未完成` |

### Content

| UI Action | Endpoint | Notes | Web Status |
| --- | --- | --- | --- |
| 上传凭证 | `POST /v1/content/upload-credentials` | 发布媒体前置 | `[ ] 未完成` |
| 发布文章 | `POST /v1/content/article/publish` | 需要 token | `[x] 已完成` |
| 发布视频 | `POST /v1/content/video/publish` | 需要 token | `[ ] 未完成` |
| 编辑文章 | `PUT /v1/content/article/:content_id` | 需要 owner | `[x] 已完成` |
| 编辑视频 | `PUT /v1/content/video/:content_id` | 需要 owner | `[x] 已完成` |
| 删除内容 | `DELETE /v1/content/:content_id` | 需要 owner | `[ ] 未完成` |
| 内容详情 | `POST /v1/content/detail` | Optional login | `[x] 已完成` |

### Feed

| UI Action | Endpoint | Notes | Web Status |
| --- | --- | --- | --- |
| 推荐流 | `POST /v1/feed/recommend` | cursor + snapshot | `[~] 部分完成，首屏和 snapshot 已接入，连续分页未完成` |
| 关注流 | `POST /v1/feed/follow` | Optional login, 未登录为空态 | `[~] 部分完成，首屏和未登录空态已接入，连续分页未完成` |
| 用户发布 | `POST /v1/feed/user/publish` | Optional login | `[~] 部分完成，个人主页内已接入，独立发布列表未完成` |
| 用户收藏 | `POST /v1/feed/user/favorite` | Optional login | `[ ] 未完成` |

### Interaction

| UI Action | Endpoint | Notes | Web Status |
| --- | --- | --- | --- |
| 点赞 | `POST /v1/interaction/like` | 需要 token | `[x] 已完成` |
| 取消点赞 | `POST /v1/interaction/unlike` | 需要 token | `[x] 已完成` |
| 点赞信息 | `POST /v1/interaction/like/info` | 可用于详情补偿 | `[ ] 未完成` |
| 批量点赞信息 | `POST /v1/interaction/like/info/batch` | 可用于 Feed 补偿 | `[ ] 未完成` |
| 收藏 | `POST /v1/interaction/favorite` | 需要 token | `[x] 已完成` |
| 取消收藏 | `DELETE /v1/interaction/favorite` | 需要 token | `[x] 已完成` |
| 收藏信息 | `POST /v1/interaction/favorite/info` | 详情补偿 | `[ ] 未完成` |
| 评论 | `POST /v1/interaction/comment` | 需要 token | `[x] 已完成` |
| 评论列表 | `POST /v1/interaction/comment/list` | cursor | `[~] 部分完成，静态渲染和写操作已接入，列表接口未接入` |
| 回复列表 | `POST /v1/interaction/comment/reply/list` | cursor | `[~] 部分完成，内联回复写操作已接入，回复列表接口未接入` |
| 删除评论 | `DELETE /v1/interaction/comment` | 需要 token | `[x] 已完成` |
| 关注用户 | `POST /v1/interaction/followings` | 需要 token | `[x] 已完成` |
| 取消关注 | `DELETE /v1/interaction/followings` | 需要 token | `[x] 已完成` |

### Search

| UI Action | Endpoint | Notes | Web Status |
| --- | --- | --- | --- |
| 搜索内容 | `POST /v1/search/contents` | latest cursor, relevance/hybrid page token | `[~] 部分完成，hybrid 首屏已接入，分页和 mode 切换未完成` |
| 搜索用户 | `POST /v1/search/users` | Optional login enrich following | `[~] 部分完成，hybrid 首屏已接入，分页和 mode 切换未完成` |

## Interaction Logic

### Auth

- token 统一放在 `Authorization` header。
- Web 存储优先使用 httpOnly cookie 方案。如暂未接入 cookie，可使用安全封装的 local storage，并在文档中标明风险。
- App 使用 Keychain 或 Android Keystore。
- Optional login 接口在 token 失效时按游客态处理，不弹错误。
- Required login 接口在 401 时打开 Auth Sheet。

### Pagination

Feed:

- 首屏 `cursor=""`。
- 请求成功后保存 `next_cursor`。
- `has_more=false` 时停止自动加载。
- 推荐流额外保存 `snapshot_id`。

Search:

- latest 使用 `cursor`。
- relevance 和 hybrid 使用 `page_token` 和 `snapshot_id`。
- 切换 query、mode、segment 时重置分页。

Comments:

- 一级评论从 `cursor=0` 开始。
- 回复列表按 commentId 独立分页。

### Optimistic Updates

点赞：

1. 本地切换 `is_liked`。
2. `like_count` 加 1 或减 1。
3. 调接口。
4. 成功保持。
5. 失败恢复原状态并提示。

收藏：

- 与点赞一致，但只在有真实收藏状态的页面展示。

关注：

- 本地切换 `is_following`。
- 粉丝数是否同步变化取决于当前页面是否展示该统计。
- 失败恢复。

评论：

- 发送后插入 pending 项。
- 成功替换 `comment_id`。
- 失败保留输入和 pending error。

删除：

- 先确认。
- 本地淡出。
- 失败恢复并提示。

### Upload

1. 用户选择文件。
2. 前端读取 file name、size、extension。
3. 调 `/v1/content/upload-credentials`。
4. 使用返回的 `url` 和 `form_data` 直传。
5. 成功后拿到可发布 URL。
6. 失败可重试。
7. 上传中允许继续编辑。

限制：

- 文件大小必须小于接口 `MaxBytes` 和业务限制。
- 上传进度不使用全屏遮罩。
- 发布按钮必须等必要媒体完成后可用。

### Error Copy

| Case | User-facing copy |
| --- | --- |
| 未登录 | 请先登录后继续操作 |
| 会话过期 | 登录状态已过期，请重新登录 |
| Feed 失败 | 内容加载失败，请重试 |
| 搜索失败 | 搜索失败，请重试 |
| 发布校验失败 | 请补全必填内容 |
| 上传失败 | 上传失败，请重试 |
| 删除失败 | 删除失败，请重试 |
| 网络离线 | 网络不可用，恢复后再试 |

错误信息不暴露 RPC、Redis、MySQL、空指针等技术细节。

## Web and App Screen Matrix

| Screen | Web Required | Web Status | App Required | App Status | Notes |
| --- | --- | --- | --- | --- | --- |
| Launch | Yes | `[x] 已完成` | Yes | `[ ] 未完成` | 会话恢复 |
| Login | Yes | `[x] 已完成` | Yes | `[ ] 未完成` | Sheet 或独立页 |
| Register | Yes | `[x] 已完成` | Yes | `[ ] 未完成` | 注册后进入原目标 |
| Home recommend | Yes | `[~] 部分完成` | Yes | `[ ] 未完成` | 核心首屏 |
| Follow feed | Yes | `[~] 部分完成` | Yes | `[ ] 未完成` | 未登录空态 |
| Feed card | Yes | `[~] 部分完成` | Yes | `[ ] 未完成` | 共享组件模型 |
| Article detail | Yes | `[~] 部分完成` | Yes | `[ ] 未完成` | 正文清晰实体面 |
| Video detail | Yes | `[~] 部分完成` | Yes | `[ ] 未完成` | App 支持横屏全屏 |
| Comments | Yes | `[~] 部分完成` | Yes | `[ ] 未完成` | Web panel, App sheet |
| Search home | Yes | `[~] 部分完成` | Yes | `[ ] 未完成` | 最近搜索 |
| Search results | Yes | `[~] 部分完成` | Yes | `[ ] 未完成` | 内容和用户 |
| Publish article | Yes | `[~] 部分完成` | Yes | `[ ] 未完成` | 草稿和上传 |
| Publish video | Yes | `[ ] 未完成` | Yes | `[ ] 未完成` | 上传队列 |
| Edit content | Yes | `[x] 已完成` | Yes | `[ ] 未完成` | owner 入口 |
| User profile | Yes | `[~] 部分完成` | Yes | `[ ] 未完成` | 发布、收藏、粉丝 |
| Me | Yes | `[~] 部分完成` | Yes | `[ ] 未完成` | 当前用户统计 |
| Edit profile | Yes | `[~] 部分完成` | Yes | `[ ] 未完成` | 头像和资料 |
| Followers | Yes | `[ ] 未完成` | Yes | `[ ] 未完成` | 当前接口支持 |
| Favorites | Yes | `[ ] 未完成` | Yes | `[ ] 未完成` | 用户收藏流 |
| Published list | Yes | `[ ] 未完成` | Yes | `[ ] 未完成` | 用户发布流 |
| Settings | Yes | `[~] 部分完成` | Yes | `[ ] 未完成` | 主题、退出 |
| Not found | Yes | `[x] 已完成` | Yes | `[ ] 未完成` | App 用错误屏 |
| Offline | Yes | `[ ] 未完成` | Yes | `[ ] 未完成` | 保留缓存内容 |

## Implementation Notes

### Web Stack Recommendation

- Framework: React Router 或 Next.js。
- Styling: Tailwind + CSS variables。
- Components: 自建组件，必要时使用 Radix primitives。
- Motion: `motion/react`。
- Icons: Lucide 线性图标为主，Material Symbols 作为系统型 glyph 补充。
- Data: TanStack Query 或 SWR。
- Forms: React Hook Form 或同等能力。

### App Stack Recommendation

Native:

- iOS: SwiftUI。
- Android: Jetpack Compose。

Cross-platform:

- React Native 或 Flutter 可用，但需要建立 token mapping。
- 不建议用纯 WebView 承载核心 App，除非目标只是演示。

### Suggested Frontend Modules

```text
frontend/
  app/
    routes/
    layouts/
  features/
    auth/
    feed/
    content/
    comments/
    compose/
    profile/
    search/
    settings/
  shared/
    api/
    components/
    design-tokens/
    state/
    utils/
```

App 端可按相同 feature 分包。

## Test Steps

### Document Checks

1. 确认文档主题明确为「液态毛玻璃 Feed 流内容系统」。
2. 确认 Web 首屏不是营销落地页、宣传 hero、后台模板或孤立卡片展示。
3. 确认 Web 首屏包含顶部固定导航、左侧导航、中央 Feed 和右侧辅助栏。
4. 确认 Feed 卡片覆盖图文、长文摘要、图片组、视频预览和链接分享。
5. 确认交互状态包含当前频道高亮、点赞选中、收藏选中、搜索聚焦、通知角标、趋势榜排名和 hover 上浮。
6. 确认 App 端有简化顶部导航、底部 Tab Bar、单列 Feed 和趋势快照说明。
7. 确认所有已设计页面都有 Web 和 App 对应说明。
8. 确认当前 `front-api` 未支持的关注列表没有被设计成已支持页面。
9. 确认接口映射路径与 `app/front/doc/*/*.api` 一致。

### Web Manual Verification

1. 在 375px、768px、1280px、1440px 宽度检查首页、详情、搜索、发布、个人页。
2. 在 1440px 首屏检查顶部固定导航、左侧导航、中央 Feed、右侧辅助栏同时可见。
3. 检查首屏至少可见 2 张完整 Feed 卡片和第 3 张开头，并混排至少 2 种内容类型。
4. 检查背景不是纯白，不是深色赛博朋克，视觉主色为冷白、冰蓝、银灰、淡青和少量紫色环境光。
5. 检查搜索框聚焦态、通知角标、当前频道高亮、点赞选中、收藏选中、趋势榜排名状态。
6. 检查 Feed 卡片 hover 上浮、active 下压、玻璃边缘高光增强，并且不造成布局跳动。
7. 检查浅色和深色主题。
8. 检查 reduced motion 下页面仍可用。
9. 检查未登录用户浏览、搜索、打开详情。
10. 检查未登录用户点赞、收藏、关注、评论、发布时触发 Auth Sheet。
11. 检查 Feed 分页不会重复加载同一页。
12. 检查发布文章和视频失败时输入不丢失。
13. 检查上传失败、会话过期、网络离线状态。

### App Manual Verification

1. iOS 和 Android 检查简化顶部导航、底部 Tab、安全区、返回手势和键盘避让。
2. 检查 Home、Follow、Search、Compose、Me 五个 Tab。
3. 检查 Home 首屏为单列 Feed，包含频道 chip、通知角标、发布入口和至少 3 种 Feed 卡片类型。
4. 检查 Trend snapshot 出现在前 6 张 Feed 之后，不抢占首屏主内容。
5. 检查点赞、收藏、关注的乐观更新和失败回滚。
6. 检查内容详情 push 和评论 bottom sheet。
7. 检查发布中切后台再返回，草稿仍存在。
8. 检查视频详情横屏播放。
9. 检查弱网下骨架屏和缓存内容。
10. 检查 haptic 只用于关键写操作。

### API Flow Verification

1. 注册并登录，确认 token 存储和 header 注入。
2. 获取推荐流，记录 `snapshot_id`，继续翻页。
3. 搜索内容，分别验证 latest 和 hybrid 翻页。
4. 发布文章，进入详情。
5. 点赞、收藏、评论、回复，再刷新详情。
6. 关注作者，进入作者主页确认状态。
7. 编辑资料并重新进入 Me。
8. 退出登录，确认 Required login 操作触发登录引导。

## Acceptance Criteria

- AC-1: 文档明确覆盖 Web 端和 App 端完整界面。`[x] 已完成`
- AC-2: Web 首屏被定义为真实可用的 Feed 产品界面，而不是营销落地页或后台模板。`[x] 已完成`
- AC-3: 文档明确液态毛玻璃的视觉语言、使用边界、降级策略和可读性规则。`[x] 已完成`
- AC-4: 文档定义了顶部导航、左侧导航、中央 Feed、右侧辅助栏、移动端顶部导航和底部 Tab。`[x] 已完成`
- AC-5: Feed 卡片明确覆盖作者信息、认证、时间、标题或摘要、媒体或链接预览、话题标签和互动按钮。`[x] 已完成`
- AC-6: 交互状态明确包含频道高亮、点赞选中、收藏选中、hover 上浮、搜索聚焦、通知角标和趋势榜排名。`[x] 已完成`
- AC-7: 文档没有把当前缺失接口包装成已支持能力。`[x] 已完成`
- AC-8: 每个核心页面都包含布局、交互、状态和接口说明。`[x] 已完成`
- AC-9: Web 和 App 的平台差异被明确说明。`[x] 已完成`
- AC-10: 设计系统包含颜色、字体、圆角、玻璃层级、动效和状态组件。`[x] 已完成`
- AC-11: 验收步骤可直接用于后续实现后的 QA。`[x] 已完成`

## Change Log

| Date | Version | Description | Author |
| --- | --- | --- | --- |
| 2026-06-11 | 1.1.0 | Rewrite around high fidelity liquid glass feed system for Web and App | Codex |
| 2026-06-11 | 1.0.0 | Initial frontend experience specification for Web and App | Codex |
| 2026-06-14 | 1.1.1 | Align visual language and interface sources with the current frontend implementation | Codex |
| 2026-06-15 | 1.1.2 | Mark completed, partial, and pending implementation items against current Web project state | Codex |
