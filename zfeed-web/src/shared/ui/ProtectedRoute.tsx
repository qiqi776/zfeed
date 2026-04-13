import type { PropsWithChildren } from "react";
import { Navigate, useLocation } from "react-router-dom";

import { useSessionStore } from "@/entities/session/model/session.store";

export function ProtectedRoute({ children }: PropsWithChildren) {
  const token = useSessionStore((state) => state.token);
  const location = useLocation();

  if (!token) {
    return <Navigate to="/login" replace state={{ from: location }} />;
  }

  return children;
}
