import type { PropsWithChildren } from "react";
import { Navigate } from "react-router-dom";

import { useSessionStore } from "@/entities/session/model/session.store";

export function PublicOnlyRoute({ children }: PropsWithChildren) {
  const token = useSessionStore((state) => state.token);
  if (token) {
    return <Navigate to="/" replace />;
  }
  return children;
}
