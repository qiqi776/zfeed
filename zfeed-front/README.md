# zfeed-front

zfeed-front 是 zfeed 内容社区系统的 React 前端实现，支持用户端全功能（注册、登录、内容发布、Feed 浏览、互动、搜索、个人资料编辑）和管理后台（数据看板、用户管理、内容审核、评论管理、系统设置）。

## 项目结构

```text
src/
├── pages/              页面级 TSX 组件
│   └── admin/          管理后台页面（仪表盘、用户管理、内容审核、评论管理、设置、登录）
├── routes/             路由解析（前台路由 + 管理后台路由）
│   ├── pageRoutes.ts   前台路由映射（/home, /following, /me, /search, /compose …）
│   ├── adminRoutes.ts  管理后台路由映射（/admin, /admin/login, /admin/users …）
│   └── constants.ts    路由常量与辅助函数
├── runtime/            共享运行时行为
│   ├── apiClient.ts    HTTP 请求封装、OSS 文件上传、全部 API 函数
│   ├── authStore.ts    认证会话管理（token、user、role、admin 判断）
│   ├── navigation.ts   客户端路由导航
│   ├── PageShell.tsx   页面壳组件
│   └── toast.ts        全局 toast 通知
├── config/             Vite 代理配置测试
├── styles/             全局 Tailwind 样式
├── App.tsx             SPA 入口（路由懒加载、客户端导航拦截）
└── main.tsx            React DOM 挂载入口
```

## 快速开始

```bash
npm install
npm run dev
```

应用在 `/` 打开会话网关：已有有效 session 时通过 `GET /v1/users/me` 恢复登录态并进入 `/home`；未认证则跳转至 `/login`。登录与注册共用毛玻璃风格的首页背景。推荐 Feed 位于 `/home`。

### 前台路由

| 路由                  | 页面           | 说明                       |
| --------------------- | -------------- | -------------------------- |
| `/`                   | 会话网关       | 自动恢复登录态或跳转登录   |
| `/home`               | 推荐 Feed      | 个性化推荐信息流           |
| `/following`          | 关注 Feed      | 已关注用户的内容流         |
| `/me`                 | 个人主页       | 当前用户资料与发布列表     |
| `/me/edit`            | 编辑资料       | 修改昵称、头像、简介等     |
| `/user/:id`           | 用户主页       | 查看其他用户资料           |
| `/content/:id`        | 内容详情       | 文章/视频详情、评论、互动  |
| `/content/:id/edit`   | 编辑内容       | 编辑已发布内容             |
| `/compose`            | 内容发布       | 支持文章/视频发布（含草稿）|
| `/search`             | 搜索           | 用户与内容并行搜索         |
| `/settings`           | 设置           | 账号设置                   |
| `/login`              | 登录           | 手机号 + 密码登录          |
| `/register`           | 注册           | 新用户注册                 |

### 管理后台路由

| 路由              | 页面         | 说明             | 角色要求 |
| ----------------- | ------------ | ---------------- | -------- |
| `/admin`          | 数据看板     | 系统核心指标概览 | ≥1       |
| `/admin/login`    | 管理员登录   | 管理员身份认证   | 无       |
| `/admin/users`    | 用户管理     | 列表、状态变更   | ≥1       |
| `/admin/contents` | 内容审核     | 列表、状态变更   | ≥1       |
| `/admin/comments` | 评论管理     | 列表、删除       | ≥1       |
| `/admin/settings` | 系统设置     | 管理员信息与配置 | ≥1       |

> 管理后台使用 `AdminLayout` 统一壳布局（侧边栏导航 + 顶栏），与前台完全独立。角色判定由 `authStore.isAdmin()` / `getAdminRole()` 依据 `user.role` 字段实现。

## OSS 文件上传

前端通过 `uploadFileToOSS(file, scene, onProgress)` 上传文件到阿里云 OSS（或 fallback 为本地 Data URL）：

| 场景（scene）      | 文件类型            | 大小限制 | OSS 未配置时的行为                        |
| ------------------ | ------------------- | -------- | ----------------------------------------- |
| `avatar`           | .jpg/.jpeg/.png/.webp | ≤5 MB    | 后端返回错误，上传不可用                  |
| `article-cover`    | .jpg/.jpeg/.png/.webp | ≤10 MB   | fallback 为 Data URL                      |
| `article-image`    | .jpg/.jpeg/.png/.webp | ≤10 MB   | fallback 为 Data URL                      |
| `video-cover`      | .jpg/.jpeg/.png/.webp | ≤10 MB   | fallback 为 Data URL                      |
| `video-source`     | .mp4/.mov/.m4v/.webm  | ≤512 MB  | 不可用，提示"视频文件需要 OSS 存储支持"   |

上传流程：`getUploadCredentials` → 获取 OSS 表单凭证 → 直传到 OSS Host → 返回文件 URL。若凭证获取失败（OSS 未配置）则自动进入 `uploadFileLocal` 本地模式（仅图片，≤10MB）。

OSS 配置方法详见[后端工程文档](../zfeed-main/README.md#oss-对象存储配置)。

## 主要技术特性

- **React 18** + TypeScript，SPA 客户端路由（`history.pushState` + `popstate`）
- **代码分割**：所有页面通过 `React.lazy()` 动态导入，按路由按需加载
- **Tailwind CSS**：全局样式 + 玻璃拟态（glass morphism）设计系统 (`sharedGlassStyles.ts`)
- **认证体系**：Bearer Token 存储在 `localStorage`，`authStore` 管理 session 生命周期（含 role 字段用于管理员判断）
- **API 封装**：`apiClient.ts` 统一处理 auth/optionalAuth/guest 三种请求模式，自动清理过期 session
- **OSS 上传**：支持带进度回调的 XHR 直传，Local Data URL fallback
- **Vite**：开发代理到后端 API（`VITE_API_BASE_URL`）
- **测试**：Vitest 单元测试 + Playwright e2e 测试

## 验证

```bash
npm test         # 单元测试
npm run lint     # ESLint 检查
npm run build    # 生产构建
npm run test:e2e # Playwright e2e 测试
```

## 后端依赖

前端依赖 [zfeed 后端服务](../zfeed-main/README.md) 提供全部 API。本地开发时需先启动后端 Docker 栈：

```bash
cd ../zfeed-main && bash ./scripts/start.sh
```

启动后前端默认代理到 `http://127.0.0.1:5000`（可通过 `VITE_API_BASE_URL` 环境变量覆盖）。
