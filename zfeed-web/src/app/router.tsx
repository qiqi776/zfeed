import { createBrowserRouter } from "react-router-dom";

import { AppLayout } from "@/app/layout/AppLayout";
import { LoginPage } from "@/features/auth/pages/LoginPage";
import { RegisterPage } from "@/features/auth/pages/RegisterPage";
import { ContentDetailPage } from "@/features/content/pages/ContentDetailPage";
import { FavoritesPage } from "@/features/favorites/pages/FavoritesPage";
import { FollowPage } from "@/features/feed/pages/FollowPage";
import { RecommendPage } from "@/features/feed/pages/RecommendPage";
import { MePage } from "@/features/me/pages/MePage";
import { PublishArticlePage } from "@/features/publish/pages/PublishArticlePage";
import { PublishPage } from "@/features/publish/pages/PublishPage";
import { PublishVideoPage } from "@/features/publish/pages/PublishVideoPage";
import { StudioPage } from "@/features/studio/pages/StudioPage";
import { UserProfilePage } from "@/features/user/pages/UserProfilePage";
import { NotFoundPage } from "@/pages/not-found/NotFoundPage";
import { ProtectedRoute } from "@/shared/ui/ProtectedRoute";
import { PublicOnlyRoute } from "@/shared/ui/PublicOnlyRoute";

export const router = createBrowserRouter([
  {
    path: "/login",
    element: (
      <PublicOnlyRoute>
        <LoginPage />
      </PublicOnlyRoute>
    ),
  },
  {
    path: "/register",
    element: (
      <PublicOnlyRoute>
        <RegisterPage />
      </PublicOnlyRoute>
    ),
  },
  {
    path: "/",
    element: (
      <ProtectedRoute>
        <AppLayout />
      </ProtectedRoute>
    ),
    children: [
      {
        index: true,
        element: <RecommendPage />,
      },
      {
        path: "me",
        element: <MePage />,
      },
      {
        path: "following",
        element: <FollowPage />,
      },
      {
        path: "favorites",
        element: <FavoritesPage />,
      },
      {
        path: "publish",
        element: <PublishPage />,
      },
      {
        path: "publish/article",
        element: <PublishArticlePage />,
      },
      {
        path: "publish/video",
        element: <PublishVideoPage />,
      },
      {
        path: "studio",
        element: <StudioPage />,
      },
      {
        path: "users/:userId",
        element: <UserProfilePage />,
      },
      {
        path: "content/:contentId",
        element: <ContentDetailPage />,
      },
    ],
  },
  {
    path: "*",
    element: <NotFoundPage />,
  },
]);
