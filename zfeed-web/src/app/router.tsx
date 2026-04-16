import { type LazyExoticComponent, type ReactNode, Suspense, lazy } from "react";
import { createBrowserRouter } from "react-router-dom";

import { ProtectedRoute } from "@/shared/ui/ProtectedRoute";
import { PublicOnlyRoute } from "@/shared/ui/PublicOnlyRoute";

const AppLayout = lazy(async () => ({
  default: (await import("@/app/layout/AppLayout")).AppLayout,
}));
const LoginPage = lazy(async () => ({
  default: (await import("@/features/auth/pages/LoginPage")).LoginPage,
}));
const RegisterPage = lazy(async () => ({
  default: (await import("@/features/auth/pages/RegisterPage")).RegisterPage,
}));
const RecommendPage = lazy(async () => ({
  default: (await import("@/features/feed/pages/RecommendPage")).RecommendPage,
}));
const FollowPage = lazy(async () => ({
  default: (await import("@/features/feed/pages/FollowPage")).FollowPage,
}));
const FavoritesPage = lazy(async () => ({
  default: (await import("@/features/favorites/pages/FavoritesPage")).FavoritesPage,
}));
const MePage = lazy(async () => ({
  default: (await import("@/features/me/pages/MePage")).MePage,
}));
const SettingsPage = lazy(async () => ({
  default: (await import("@/features/user/pages/SettingsPage")).SettingsPage,
}));
const PublishPage = lazy(async () => ({
  default: (await import("@/features/publish/pages/PublishPage")).PublishPage,
}));
const SearchPage = lazy(async () => ({
  default: (await import("@/features/search/pages/SearchPage")).SearchPage,
}));
const PublishArticlePage = lazy(async () => ({
  default: (await import("@/features/publish/pages/PublishArticlePage")).PublishArticlePage,
}));
const PublishVideoPage = lazy(async () => ({
  default: (await import("@/features/publish/pages/PublishVideoPage")).PublishVideoPage,
}));
const StudioPage = lazy(async () => ({
  default: (await import("@/features/studio/pages/StudioPage")).StudioPage,
}));
const EditArticlePage = lazy(async () => ({
  default: (await import("@/features/content/pages/EditArticlePage")).EditArticlePage,
}));
const EditVideoPage = lazy(async () => ({
  default: (await import("@/features/content/pages/EditVideoPage")).EditVideoPage,
}));
const UserProfilePage = lazy(async () => ({
  default: (await import("@/features/user/pages/UserProfilePage")).UserProfilePage,
}));
const FollowersPage = lazy(async () => ({
  default: (await import("@/features/user/pages/FollowersPage")).FollowersPage,
}));
const ContentDetailPage = lazy(async () => ({
  default: (await import("@/features/content/pages/ContentDetailPage")).ContentDetailPage,
}));
const NotFoundPage = lazy(async () => ({
  default: (await import("@/pages/not-found/NotFoundPage")).NotFoundPage,
}));

function renderRouteLoadingFallback() {
  return (
    <section className="space-y-5">
      <div className="h-12 w-44 rounded-full bg-white shadow-card" />
      <div className="grid gap-4 md:grid-cols-2">
        {Array.from({ length: 4 }).map((_, index) => (
          <div key={index} className="h-40 rounded-[28px] bg-white shadow-card" />
        ))}
      </div>
    </section>
  );
}

function lazyElement(
  Component: LazyExoticComponent<() => ReactNode>,
  fallback: ReactNode = renderRouteLoadingFallback(),
) {
  return (
    <Suspense fallback={fallback}>
      <Component />
    </Suspense>
  );
}

export const router = createBrowserRouter([
  {
    path: "/login",
    element: (
      <PublicOnlyRoute>
        {lazyElement(LoginPage)}
      </PublicOnlyRoute>
    ),
  },
  {
    path: "/register",
    element: (
      <PublicOnlyRoute>
        {lazyElement(RegisterPage)}
      </PublicOnlyRoute>
    ),
  },
  {
    path: "/",
    element: (
      <ProtectedRoute>
        {lazyElement(AppLayout)}
      </ProtectedRoute>
    ),
    children: [
      {
        index: true,
        element: lazyElement(RecommendPage),
      },
      {
        path: "me",
        element: lazyElement(MePage),
      },
      {
        path: "me/settings",
        element: lazyElement(SettingsPage),
      },
      {
        path: "following",
        element: lazyElement(FollowPage),
      },
      {
        path: "favorites",
        element: lazyElement(FavoritesPage),
      },
      {
        path: "publish",
        element: lazyElement(PublishPage),
      },
      {
        path: "search",
        element: lazyElement(SearchPage),
      },
      {
        path: "publish/article",
        element: lazyElement(PublishArticlePage),
      },
      {
        path: "publish/video",
        element: lazyElement(PublishVideoPage),
      },
      {
        path: "studio",
        element: lazyElement(StudioPage),
      },
      {
        path: "studio/article/:contentId/edit",
        element: lazyElement(EditArticlePage),
      },
      {
        path: "studio/video/:contentId/edit",
        element: lazyElement(EditVideoPage),
      },
      {
        path: "users/:userId",
        element: lazyElement(UserProfilePage),
      },
      {
        path: "users/:userId/followers",
        element: lazyElement(FollowersPage),
      },
      {
        path: "content/:contentId",
        element: lazyElement(ContentDetailPage),
      },
    ],
  },
  {
    path: "*",
    element: lazyElement(NotFoundPage),
  },
]);
