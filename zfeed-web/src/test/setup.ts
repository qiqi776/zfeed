import "@testing-library/jest-dom/vitest";
import { afterEach } from "vitest";

import { useSessionStore } from "@/entities/session/model/session.store";
import { queryClient } from "@/shared/lib/query/queryClient";
import { useToastStore } from "@/shared/ui/toast/toast.store";

afterEach(() => {
  queryClient.clear();
  useSessionStore.getState().clearSession();
  useToastStore.getState().clearToasts();
  window.localStorage.clear();
});
