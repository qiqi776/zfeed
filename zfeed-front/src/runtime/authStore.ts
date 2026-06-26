export type AuthUser = {
    userId: number;
    nickname?: string;
    avatar?: string;
    role?: number;
};

export type AuthSession = {
    token: string;
    expiredAt: number;
    user?: AuthUser;
};

export const authSessionStorageKey = "zfeed.auth.session";

export function isAdmin(): boolean {
    const session = readAuthSession();
    return (session?.user?.role ?? 0) >= 1;
}

export function getAdminRole(): number {
    const session = readAuthSession();
    return session?.user?.role ?? 0;
}

export function readAuthSession(): AuthSession | null {
    const rawSession = window.localStorage.getItem(authSessionStorageKey);
    if (!rawSession) {
        return null;
    }

    try {
        const parsed = JSON.parse(rawSession) as Partial<AuthSession>;
        if (!isValidSession(parsed)) {
            clearAuthSession();
            return null;
        }

        return {
            token: parsed.token,
            expiredAt: parsed.expiredAt,
            user: parsed.user
        };
    } catch {
        clearAuthSession();
        return null;
    }
}

export function saveAuthSession(session: AuthSession) {
    if (!isValidSession(session)) {
        clearAuthSession();
        throw new Error("Invalid auth session");
    }

    window.localStorage.setItem(authSessionStorageKey, JSON.stringify(session));
}

export function clearAuthSession() {
    window.localStorage.removeItem(authSessionStorageKey);
}

function isExpired(expiredAt: number) {
    return expiredAt <= Math.floor(Date.now() / 1000);
}

function isValidToken(token: unknown): token is string {
    return typeof token === "string" && token.trim().length > 0;
}

function isValidExpiry(expiredAt: unknown): expiredAt is number {
    return typeof expiredAt === "number" && Number.isFinite(expiredAt);
}

function isValidSession(session: Partial<AuthSession>): session is AuthSession {
    return Boolean(
        isValidToken(session.token) &&
        isValidExpiry(session.expiredAt) &&
        !isExpired(session.expiredAt) &&
        isValidOptionalUser(session.user)
    );
}

function isValidOptionalUser(user: unknown): user is AuthUser | undefined {
    if (user === undefined) {
        return true;
    }

    if (!user || typeof user !== "object") {
        return false;
    }

    const candidate = user as Partial<AuthUser>;
    return (
        typeof candidate.userId === "number" &&
        Number.isFinite(candidate.userId) &&
        (candidate.nickname === undefined || typeof candidate.nickname === "string") &&
        (candidate.avatar === undefined || typeof candidate.avatar === "string")
    );
}
