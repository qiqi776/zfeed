import { create } from "zustand";
import { persist } from "zustand/middleware";

import type { SessionState } from "@/entities/session/model/session.types";

export const useSessionStore = create<SessionState>()(
  persist(
    (set) => ({
      token: null,
      expiredAt: null,
      user: null,
      setSession: ({ token, expiredAt, user }) => {
        set({ token, expiredAt, user });
      },
      clearSession: () => {
        set({ token: null, expiredAt: null, user: null });
      },
    }),
    {
      name: "zfeed-web-session",
      partialize: (state) => ({
        token: state.token,
        expiredAt: state.expiredAt,
        user: state.user,
      }),
    },
  ),
);
