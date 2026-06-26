import { createElement, useEffect, useState } from "react";
import { getUserProfile, getUserPublishedFeed } from "../../runtime/apiClient";

function isUserActive(status: unknown): boolean {
    if (typeof status === "number") return status === 10;
    if (typeof status === "string") return status === "USER_STATUS_ACTIVE" || status === "10";
    return false;
}

type Props = {
    userId: number;
    open: boolean;
    onClose: () => void;
};

type UserInfo = {
    user_profile?: {
        user_id?: number;
        nickname?: string;
        avatar?: string;
        bio?: string;
        gender?: number;
        status?: number;
    };
    counts?: {
        followee_count?: number;
        follower_count?: number;
        like_received_count?: number;
        favorite_received_count?: number;
        content_count?: number;
    };
};

type FeedItem = {
    content_id: string | number;
    title: string;
    cover_url?: string;
    published_at?: number;
    like_count?: number;
    comment_count?: number;
};

export function UserDetailDrawer({ userId, open, onClose }: Props) {
    const [profile, setProfile] = useState<UserInfo | null>(null);
    const [posts, setPosts] = useState<FeedItem[]>([]);
    const [loading, setLoading] = useState(true);

    useEffect(() => {
        if (!open) return;
        setLoading(true);
        Promise.all([
            getUserProfile<UserInfo>(userId).catch(() => null),
            getUserPublishedFeed<{ items?: FeedItem[] }>({ user_id: String(userId), cursor: "0", page_size: 10 })
                .then((r) => r?.items ?? []).catch(() => []),
        ]).then(([p, f]) => {
            setProfile(p);
            setPosts(f);
            setLoading(false);
        });
    }, [userId, open]);

    if (!open) return null;

    const info = profile?.user_profile;
    const counts = profile?.counts;

    return createElement("div", {
        className: "fixed inset-0 z-50 flex justify-end bg-black/30",
        onClick: onClose,
    },
        createElement("div", {
            className: "w-full max-w-md bg-white h-full overflow-y-auto shadow-2xl",
            onClick: (e: Event) => e.stopPropagation(),
        },
            // Header
            createElement("div", { className: "flex items-center justify-between px-5 py-4 border-b border-slate-200" },
                createElement("h3", { className: "text-lg font-semibold text-slate-800" }, "用户详情"),
                createElement("button", {
                    onClick: onClose,
                    className: "text-slate-400 hover:text-slate-600 text-xl leading-none",
                }, "✕"),
            ),

            createElement("div", { className: "p-5 space-y-5" },
                loading
                    ? createElement("p", { className: "text-sm text-slate-400" }, "加载中...")
                    : createElement("div", { className: "space-y-5" },
                        // Avatar + name
                        createElement("div", { className: "flex items-center gap-4" },
                            info?.avatar
                                ? createElement("img", { src: info.avatar, className: "w-16 h-16 rounded-full object-cover" })
                                : createElement("div", { className: "w-16 h-16 rounded-full bg-slate-200 flex items-center justify-center text-slate-400 text-xl" }, "U"),
                            createElement("div", null,
                                createElement("p", { className: "text-lg font-semibold text-slate-800" }, info?.nickname || "未设置"),
                                createElement("p", { className: "text-sm text-slate-500" }, `ID: ${info?.user_id ?? userId}`),
                                createElement("span", {
                                    className: `inline-block mt-1 px-2 py-0.5 rounded-full text-xs font-medium ${isUserActive(info?.status) ? "bg-green-100 text-green-700" : "bg-red-100 text-red-700"}`,
                                }, isUserActive(info?.status) ? "正常" : "已禁用"),
                            ),
                        ),

                        // Bio
                        info?.bio &&
                            createElement("p", { className: "text-sm text-slate-600 bg-slate-50 rounded-lg p-3" }, info.bio),

                        // Stats
                        counts && createElement("div", {
                            className: "grid gap-3",
                            style: { gridTemplateColumns: "repeat(3, 1fr)" }
                        },
                            ...[
                                { label: "关注", value: counts.followee_count ?? 0 },
                                { label: "粉丝", value: counts.follower_count ?? 0 },
                                { label: "获赞", value: counts.like_received_count ?? 0 },
                                { label: "获收藏", value: counts.favorite_received_count ?? 0 },
                                { label: "内容数", value: counts.content_count ?? 0 },
                            ].map((s) =>
                                createElement("div", { key: s.label, className: "bg-slate-50 rounded-lg p-3 text-center" },
                                    createElement("p", { className: "text-lg font-bold text-slate-800" }, String(s.value)),
                                    createElement("p", { className: "text-xs text-slate-500" }, s.label),
                                ),
                            ),
                        ),

                        // Recent posts
                        createElement("div", null,
                            createElement("h4", { className: "text-sm font-semibold text-slate-700 mb-3" }, "最近发布"),
                            posts.length === 0
                                ? createElement("p", { className: "text-sm text-slate-400" }, "暂无内容")
                                : createElement("div", { className: "space-y-2" },
                                    posts.slice(0, 5).map((item) =>
                                        createElement("a", {
                                            key: String(item.content_id),
                                            href: `/content/${item.content_id}`,
                                            target: "_blank",
                                            className: "block bg-slate-50 rounded-lg p-3 hover:bg-slate-100 transition-colors",
                                        },
                                            createElement("div", { className: "flex gap-3" },
                                                item.cover_url
                                                    ? createElement("img", { src: item.cover_url, className: "w-14 h-10 rounded object-cover shrink-0" })
                                                    : createElement("div", { className: "w-14 h-10 rounded bg-slate-200 shrink-0" }),
                                                createElement("div", { className: "min-w-0 flex-1" },
                                                    createElement("p", { className: "text-sm font-medium text-slate-700 truncate" }, item.title || "无标题"),
                                                    createElement("p", { className: "text-xs text-slate-400 mt-1" },
                                                        `👍 ${item.like_count ?? 0}  💬 ${item.comment_count ?? 0}`,
                                                    ),
                                                ),
                                            ),
                                        ),
                                    ),
                                ),
                        ),

                        // Actions
                        createElement("div", { className: "flex gap-2 pt-2" },
                            createElement("a", {
                                href: `/user/${userId}`,
                                target: "_blank",
                                className: "flex-1 text-center px-4 py-2 text-sm font-medium text-blue-600 bg-blue-50 rounded-lg hover:bg-blue-100 transition-colors",
                            }, "前台主页"),
                        ),
                    ),
            ),
        ),
    );
}
