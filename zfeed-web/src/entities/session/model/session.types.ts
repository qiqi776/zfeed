export type SessionUser = {
  userId: number;
  nickname: string;
  avatar: string;
};

export type SessionState = {
  token: string | null;
  expiredAt: number | null;
  user: SessionUser | null;
  setSession: (payload: {
    token: string;
    expiredAt: number;
    user: SessionUser | null;
  }) => void;
  clearSession: () => void;
};
