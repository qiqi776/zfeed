import { clearAuthSession, readAuthSession } from "./authStore";

type RequestOptions = {
    method?: "GET" | "POST" | "PUT" | "DELETE";
    body?: unknown;
    auth?: boolean;
    optionalAuth?: boolean;
    fallbackToGuestOnAuthFailure?: boolean;
};

export type ContentScene = "ARTICLE" | "VIDEO" | "COMMENT";

type ContentActionBody = {
    contentId: string;
    contentUserId?: string;
    scene?: ContentScene;
};

type FollowUserBody = {
    target_user_id: string;
};

type CommentBody = {
    content_id: string;
    scene: ContentScene;
    comment: string;
    content_user_id: string;
    parent_id?: string;
    root_id?: string;
    reply_to_user_id?: string;
};

type DeleteCommentBody = {
    comment_id: string;
    content_id: string;
    scene: ContentScene;
    root_id?: string;
    parent_id?: string;
};

type UpdateProfileBody = {
    nickname?: string;
    avatar?: string;
    bio?: string;
    gender?: number;
    email?: string;
    birthday?: number;
};

type PublishArticleBody = {
    title: string;
    description?: string;
    cover?: string;
    content: string;
    visibility: number;
};

type EditArticleBody = {
    title?: string;
    description?: string;
    cover?: string;
    content?: string;
};

type EditVideoBody = {
    title?: string;
    description?: string;
    video_url?: string;
    cover_url?: string;
    duration?: number;
};

type SearchMode = "latest" | "relevance" | "hybrid";

type SearchRequestBody = {
    query: string;
    cursor?: number;
    page_size?: number;
    mode?: SearchMode;
    page_token?: string;
    snapshot_id?: string;
};

type RecommendFeedBody = {
    cursor: string;
    page_size: number;
    snapshot_id?: string;
};

type FollowFeedBody = {
    cursor: string;
    page_size: number;
};

type UserPublishedFeedBody = {
    user_id: string;
    cursor: string;
    page_size: number;
};

type UserFavoriteFeedBody = {
    user_id: string;
    cursor: string;
    page_size: number;
};

type UserFollowListBody = {
    user_id: string;
    cursor: number;
    page_size: number;
};

type ContentDetailBody = {
    content_id: string;
};

type CommentListBody = {
    content_id: string;
    scene: ContentScene;
    cursor: string | number;
    page_size: number;
};

type CommentReplyListBody = {
    comment_id: string;
    cursor: string | number;
    page_size: number;
};

type UserProfileId = string | number;

export class ApiError extends Error {
    constructor(
        public readonly status: number,
        message = "请求失败，请稍后重试"
    ) {
        super(message);
    }
}

export async function apiRequest<T>(path: string, options: RequestOptions = {}): Promise<T> {
    const request = options.auth ? authorizedFetch : rawFetch;
    const response = await request(path, options).catch((error: unknown) => {
        if (!shouldRetryAsGuest(error, options)) {
            throw error;
        }

        return rawFetch(path, {
            ...options,
            optionalAuth: false,
            fallbackToGuestOnAuthFailure: false
        });
    });

    return parseJsonResponse<T>(response);
}

function shouldRetryAsGuest(error: unknown, options: RequestOptions) {
    return Boolean(options.optionalAuth && options.fallbackToGuestOnAuthFailure && isAuthError(error));
}

async function parseJsonResponse<T>(response: Response): Promise<T> {
    if (response.status === 204) {
        return undefined as T;
    }

    const body = await response.text();
    if (!body) {
        return undefined as T;
    }

    try {
        return JSON.parse(body) as T;
    } catch {
        throw new ApiError(response.status, "响应解析失败，请稍后重试");
    }
}

export function authorizedFetch(path: string, options: RequestOptions = {}) {
    const session = readAuthSession();
    if (!session) {
        return Promise.reject(new ApiError(401, "请先登录后再操作"));
    }

    if (isAbsoluteHttpUrl(path)) {
        return Promise.reject(new ApiError(400, "认证请求只能发送到站内 API"));
    }

    return rawFetch(path, {
        ...options,
        auth: true
    });
}

async function rawFetch(path: string, options: RequestOptions = {}) {
    const headers: Record<string, string> = {
        Accept: "application/json"
    };
    const init: RequestInit = {
        method: options.method ?? "GET",
        headers
    };

    if (options.body !== undefined) {
        headers["Content-Type"] = "application/json";
        init.body = JSON.stringify(options.body);
    }

    if ((options.auth || options.optionalAuth) && isAbsoluteHttpUrl(path)) {
        throw new ApiError(400, "认证请求只能发送到站内 API");
    }

    const session = options.auth || options.optionalAuth ? readAuthSession() : null;
    if (session) {
        headers.Authorization = `Bearer ${session.token}`;
    }

    let response: Response;
    try {
        response = await fetch(resolveApiUrl(path), init);
    } catch {
        throw new ApiError(0, "网络连接失败，请稍后重试");
    }

    if (response.status === 401 || response.status === 403) {
        clearAuthSession();
        throw new ApiError(response.status, "登录状态已失效");
    }

    if (!response.ok) {
        throw new ApiError(response.status);
    }

    return response;
}

export function getApiBaseUrl() {
    return (import.meta.env.VITE_API_BASE_URL ?? "").replace(/\/$/, "");
}

function resolveApiUrl(path: string) {
    const baseUrl = getApiBaseUrl();
    if (!baseUrl || isAbsoluteHttpUrl(path)) {
        return path;
    }

    return `${baseUrl}${path.startsWith("/") ? path : `/${path}`}`;
}

function isAbsoluteHttpUrl(path: string) {
    return /^https?:\/\//i.test(path);
}

function isAuthError(error: unknown) {
    return error instanceof ApiError && (error.status === 401 || error.status === 403);
}

export function getMe<T>() {
    return apiRequest<T>("/v1/users/me", { auth: true });
}

export function login<T>(body: { mobile: string; password: string }) {
    return apiRequest<T>("/v1/login", { method: "POST", body });
}

export function register<T>(body: Record<string, string | undefined>) {
    return apiRequest<T>("/v1/users", { method: "POST", body });
}

export function getRecommendFeed<T>(body: RecommendFeedBody) {
    return apiRequest<T>("/v1/feed/recommend", { method: "POST", body, optionalAuth: true, fallbackToGuestOnAuthFailure: true });
}

export function getFollowFeed<T>(body: FollowFeedBody) {
    return apiRequest<T>("/v1/feed/follow", { method: "POST", body, optionalAuth: true });
}

export function getUserPublishedFeed<T>(body: UserPublishedFeedBody) {
    return apiRequest<T>("/v1/feed/user/publish", { method: "POST", body, optionalAuth: true, fallbackToGuestOnAuthFailure: true });
}

export function getUserFavoriteFeed<T>(body: UserFavoriteFeedBody) {
    return apiRequest<T>("/v1/feed/user/favorite", { method: "POST", body, optionalAuth: true, fallbackToGuestOnAuthFailure: true });
}

export function getUserFollowers<T>(body: UserFollowListBody) {
    return apiRequest<T>("/v1/user/followers", { method: "POST", body, optionalAuth: true, fallbackToGuestOnAuthFailure: true });
}

export function getContentDetail<T>(body: ContentDetailBody) {
    return apiRequest<T>("/v1/content/detail", { method: "POST", body, optionalAuth: true, fallbackToGuestOnAuthFailure: true });
}

export function getEditableContentDetail<T>(body: ContentDetailBody) {
    return apiRequest<T>("/v1/content/detail", { method: "POST", body, auth: true });
}

export function listComments<T>(body: CommentListBody) {
    return apiRequest<T>("/v1/interaction/comment/list", { method: "POST", body, optionalAuth: true, fallbackToGuestOnAuthFailure: true });
}

export function listCommentReplies<T>(body: CommentReplyListBody) {
    return apiRequest<T>("/v1/interaction/comment/reply/list", { method: "POST", body, optionalAuth: true, fallbackToGuestOnAuthFailure: true });
}

export function getUserProfile<T>(userId: UserProfileId) {
    return apiRequest<T>(`/v1/user/profile/${encodeURIComponent(String(userId))}`, { optionalAuth: true, fallbackToGuestOnAuthFailure: true });
}

export function searchContents<T>(body: SearchRequestBody) {
    return apiRequest<T>("/v1/search/contents", { method: "POST", body, optionalAuth: true, fallbackToGuestOnAuthFailure: true });
}

export function searchUsers<T>(body: SearchRequestBody) {
    return apiRequest<T>("/v1/search/users", { method: "POST", body, optionalAuth: true, fallbackToGuestOnAuthFailure: true });
}

export function updateProfile<T>(body: UpdateProfileBody) {
    return apiRequest<T>("/v1/users/me/profile", { method: "PUT", body, auth: true });
}

export function publishArticle<T>(body: PublishArticleBody) {
    return apiRequest<T>("/v1/content/article/publish", { method: "POST", body, auth: true });
}

export function editArticle<T>(contentId: string, body: EditArticleBody) {
    return apiRequest<T>(`/v1/content/article/${encodeURIComponent(contentId)}`, { method: "PUT", body, auth: true });
}

export function editVideo<T>(contentId: string, body: EditVideoBody) {
    return apiRequest<T>(`/v1/content/video/${encodeURIComponent(contentId)}`, { method: "PUT", body, auth: true });
}

export function likeContent<T = unknown>(body: ContentActionBody) {
    return apiRequest<T>("/v1/interaction/like", { method: "POST", body: toContentActionPayload(body, true), auth: true });
}

export function unlikeContent<T = unknown>(body: ContentActionBody) {
    return apiRequest<T>("/v1/interaction/unlike", { method: "POST", body: toContentActionPayload(body), auth: true });
}

export function favoriteContent<T = unknown>(body: ContentActionBody) {
    return apiRequest<T>("/v1/interaction/favorite", { method: "POST", body: toContentActionPayload(body), auth: true });
}

export function unfavoriteContent<T = unknown>(body: ContentActionBody) {
    return apiRequest<T>("/v1/interaction/favorite", { method: "DELETE", body: toContentActionPayload(body), auth: true });
}

export function commentContent<T = unknown>(body: CommentBody) {
    return apiRequest<T>("/v1/interaction/comment", { method: "POST", body, auth: true });
}

export function deleteComment<T = unknown>(body: DeleteCommentBody) {
    return apiRequest<T>("/v1/interaction/comment", { method: "DELETE", body, auth: true });
}

export function followUser<T = unknown>(body: FollowUserBody) {
    return apiRequest<T>("/v1/interaction/followings", { method: "POST", body, auth: true });
}

export function unfollowUser<T = unknown>(body: FollowUserBody) {
    return apiRequest<T>("/v1/interaction/followings", { method: "DELETE", body, auth: true });
}

// ---- Admin API types ----

type AdminLoginBody = {
    mobile: string;
    password: string;
};

type AdminListUsersBody = {
    query: string;
    page: number;
    page_size: number;
    status?: number;
    role?: number;
};

type AdminUpdateUserStatusBody = {
    user_id: number;
    status: number;
};

type AdminListContentsBody = {
    query: string;
    page: number;
    page_size: number;
    status?: number;
    user_id?: number;
};

type AdminUpdateContentStatusBody = {
    content_id: number;
    status: number;
};

type AdminBatchContentStatusBody = {
    content_ids: number[];
    status: number;
};

type AdminListCommentsBody = {
    query: string;
    page: number;
    page_size: number;
    status?: number;
    content_id?: number;
};

type AdminUpdateCommentStatusBody = {
    comment_id: number;
    status: number;
};

// ---- Admin API functions ----

export function adminLogin<T>(body: AdminLoginBody) {
    return apiRequest<T>("/v1/admin/login", { method: "POST", body });
}

export function adminDashboard<T>() {
    return apiRequest<T>("/v1/admin/dashboard", { method: "POST", auth: true });
}

export function adminListUsers<T>(body: AdminListUsersBody) {
    return apiRequest<T>("/v1/admin/users/list", { method: "POST", body, auth: true });
}

export function adminUpdateUserStatus<T>(body: AdminUpdateUserStatusBody) {
    return apiRequest<T>("/v1/admin/users/status", { method: "POST", body, auth: true });
}

export function adminListContents<T>(body: AdminListContentsBody) {
    return apiRequest<T>("/v1/admin/contents/list", { method: "POST", body, auth: true });
}

export function adminUpdateContentStatus<T>(body: AdminUpdateContentStatusBody) {
    return apiRequest<T>("/v1/admin/contents/status", { method: "POST", body, auth: true });
}

export function adminBatchContentStatus<T>(body: AdminBatchContentStatusBody) {
    return apiRequest<T>("/v1/admin/contents/batch-status", { method: "POST", body, auth: true });
}

export function adminListComments<T>(body: AdminListCommentsBody) {
    return apiRequest<T>("/v1/admin/comments/list", { method: "POST", body, auth: true });
}

export function adminUpdateCommentStatus<T>(body: AdminUpdateCommentStatusBody) {
    return apiRequest<T>("/v1/admin/comments/status", { method: "POST", body, auth: true });
}

export function adminGetSettings<T>() {
    return apiRequest<T>("/v1/admin/settings", { method: "POST", auth: true });
}

export function adminUpdateSettings<T>(body: { settings: Record<string, string> }) {
    return apiRequest<T>("/v1/admin/settings/update", { method: "POST", body, auth: true });
}

function toContentActionPayload(body: ContentActionBody, includeContentUser = false) {
    return {
        content_id: body.contentId,
        ...(includeContentUser && body.contentUserId ? { content_user_id: body.contentUserId } : {}),
        scene: body.scene ?? "ARTICLE"
    };
}
