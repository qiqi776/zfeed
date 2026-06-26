import { createElement, useEffect, useState } from "react";
import { ApiError, getMe, getUserFavoriteFeed, getUserFollowers, getUserProfile, getUserPublishedFeed } from "../runtime/apiClient";
import { readAuthSession } from "../runtime/authStore";
import { mergeContentInteractionState, useContentInteractionVersion } from "../runtime/contentInteractionStore";
import { PageShell } from "../runtime/PageShell";
import { renderUserAvatar } from "./avatar";
import { getContentScene } from "./contentScene";
import { PageState } from "./PageState";

type ProfileUserInfo = {
    user_id?: number | string;
    nickname?: string;
    avatar?: string;
    bio?: string;
    mobile?: string;
    email?: string;
    gender?: number;
    status?: number;
    birthday?: number;
};

type ProfileCounts = {
    followee_count?: number;
    follower_count?: number;
    like_received_count?: number;
    favorite_received_count?: number;
    content_count?: number;
};

type MeProfileResponse = {
    user_info?: ProfileUserInfo;
    user_profile?: ProfileUserInfo;
    counts?: ProfileCounts;
    followee_count?: number;
    follower_count?: number;
    like_received_count?: number;
    favorite_received_count?: number;
    content_count?: number;
};

type MeProfile = {
    userId: string;
    nickname: string;
    avatar?: string;
    bio: string;
    handle: string;
    contentCount: number;
    followeeCount: number;
    followerCount: number;
    likeReceivedCount: number;
    favoriteReceivedCount: number;
};

type MeProfileState =
    | { status: "auth-required" }
    | { status: "loading" }
    | { status: "ready"; profile: MeProfile }
    | { status: "error" };

type UserProfileResponse = MeProfileResponse & {
    viewer?: {
        is_following?: boolean;
    };
    is_following?: boolean;
};

type UserProfile = MeProfile & {
    userId: string;
    isFollowing: boolean;
};

type UserProfileState =
    | { status: "loading" }
    | { status: "ready"; profile: UserProfile }
    | { status: "not-found" }
    | { status: "error" };

type UserPublishedFeedItem = {
    content_id: string | number;
    content_type?: number | string;
    author_id: string | number;
    author_name: string;
    author_avatar?: string;
    title: string;
    description?: string;
    cover_url?: string;
    published_at?: number;
    like_count?: number;
    favorite_count?: number;
    comment_count?: number;
    is_liked?: boolean;
    is_favorited?: boolean;
};

type UserPublishedFeedResponse = {
    items?: UserPublishedFeedItem[];
};

type UserPublishedFeedState =
    | { status: "loading" }
    | { status: "ready"; items: UserPublishedFeedItem[] }
    | { status: "empty" }
    | { status: "error" };

type ProfileContentListKind = "published" | "favorites";

type ProfileRelationItem = {
    user_id: string | number;
    nickname: string;
    avatar?: string;
    bio?: string;
    is_following?: boolean;
};

type ProfileRelationResponse = {
    items?: ProfileRelationItem[];
};

type ProfileRelationState =
    | { status: "loading" }
    | { status: "ready"; items: ProfileRelationItem[] }
    | { status: "empty" }
    | { status: "error" };

type ProfileTab = "published" | "favorites" | "followers";

type ProfileTabsState = {
    activeTab: ProfileTab;
    setActiveTab: (tab: ProfileTab) => void;
    publishedFeed: UserPublishedFeedState;
    favoriteFeed: UserPublishedFeedState;
    followers: ProfileRelationState;
};

const styles = "/* Liquid Glass Utility Classes */\r\n        .glass-panel {\r\n            background: rgba(255, 255, 255, 0.4);\r\n            backdrop-filter: blur(40px);\r\n            -webkit-backdrop-filter: blur(40px);\r\n            border: 1px solid rgba(255, 255, 255, 0.6);\r\n            border-top: 1px solid rgba(255, 255, 255, 0.8); /* Specular highlight */\r\n            box-shadow: inset 0 1px 2px rgba(255, 255, 255, 0.8), \r\n                        0 4px 20px rgba(0, 0, 0, 0.05);\r\n        }\r\n        \r\n        .glass-input {\r\n            background: rgba(255, 255, 255, 0.5);\r\n            backdrop-filter: blur(20px);\r\n            border: 1px solid rgba(255, 255, 255, 0.7);\r\n            box-shadow: inset 0 1px 3px rgba(0,0,0,0.02);\r\n            transition: all 0.3s ease-out;\r\n        }\r\n        \r\n        .glass-input:focus {\n            outline: none;\n            border-color: var(--color-primary, #1f53c9);\n            box-shadow: 0 0 0 2px rgba(31, 83, 201, 0.2), inset 0 1px 3px rgba(0,0,0,0.02);\n        }\n\n        .search-shell {\n            border-radius: 9999px;\n            transition: transform 0.3s cubic-bezier(0.4, 0, 0.2, 1), filter 0.3s ease-out;\n        }\n\n        .search-shell:hover,\n        .search-shell:focus-within {\n            transform: translateY(-2px);\n            filter: drop-shadow(0 10px 22px rgba(31, 83, 201, 0.14));\n        }\n\n        .search-shell .glass-input {\n            box-shadow: inset 0 1px 2px rgba(255, 255, 255, 0.75), 0 4px 16px rgba(0, 0, 0, 0.04);\n        }\n\n        .search-shell:hover .glass-input {\n            background: rgba(255, 255, 255, 0.62);\n            border-color: rgba(255, 255, 255, 0.9);\n            box-shadow: inset 0 1px 2px rgba(255, 255, 255, 0.9), 0 8px 24px rgba(31, 83, 201, 0.10);\n        }\n\n        .search-shell:focus-within .glass-input {\n            background: rgba(255, 255, 255, 0.72);\n            border-color: rgba(31, 83, 201, 0.55);\n            box-shadow: 0 0 0 3px rgba(31, 83, 201, 0.16), inset 0 1px 2px rgba(255, 255, 255, 0.95), 0 10px 28px rgba(31, 83, 201, 0.16);\n        }\n\n        .search-shell::after {\n            content: '';\n            position: absolute;\n            top: 0;\n            left: -140%;\n            width: 42%;\n            height: 100%;\n            background: linear-gradient(to right, rgba(255,255,255,0) 0%, rgba(255,255,255,0.45) 50%, rgba(255,255,255,0) 100%);\n            transform: skewX(-25deg);\n            transition: left 0.65s ease;\n            pointer-events: none;\n            z-index: 10;\n        }\n\n        .search-shell:hover::after,\n        .search-shell:focus-within::after {\n            left: 160%;\n        }\n\n        .composer-shell {\n            position: relative;\n            flex: 1;\n            overflow: hidden;\n            border-radius: 1.25rem;\n            background: rgba(255, 255, 255, 0.34);\n            backdrop-filter: blur(18px);\n            -webkit-backdrop-filter: blur(18px);\n            border: 1px solid rgba(255, 255, 255, 0.62);\n            box-shadow: inset 0 1px 2px rgba(255, 255, 255, 0.72), 0 4px 16px rgba(0, 0, 0, 0.035);\n            transition: transform 0.3s cubic-bezier(0.4, 0, 0.2, 1), background 0.3s ease-out, border-color 0.3s ease-out, box-shadow 0.3s ease-out;\n        }\n\n        .composer-shell:hover,\n        .composer-shell:focus-within {\n            transform: translateY(-2px);\n            background: rgba(255, 255, 255, 0.58);\n            border-color: rgba(255, 255, 255, 0.9);\n            box-shadow: inset 0 1px 2px rgba(255, 255, 255, 0.9), 0 10px 28px rgba(31, 83, 201, 0.12);\n        }\n\n        .composer-shell:focus-within {\n            border-color: rgba(31, 83, 201, 0.50);\n            box-shadow: 0 0 0 3px rgba(31, 83, 201, 0.14), inset 0 1px 2px rgba(255, 255, 255, 0.95), 0 12px 30px rgba(31, 83, 201, 0.16);\n        }\n\n        .composer-shell::after {\n            content: '';\n            position: absolute;\n            top: 0;\n            left: -140%;\n            width: 42%;\n            height: 100%;\n            background: linear-gradient(to right, rgba(255,255,255,0) 0%, rgba(255,255,255,0.42) 50%, rgba(255,255,255,0) 100%);\n            transform: skewX(-25deg);\n            transition: left 0.65s ease;\n            pointer-events: none;\n            z-index: 10;\n        }\n\n        .composer-shell:hover::after,\n        .composer-shell:focus-within::after {\n            left: 160%;\n        }\n\n        .composer-shell input {\n            position: relative;\n            z-index: 20;\n            min-height: 44px;\n            padding: 0.65rem 0.95rem;\n        }\n\n        .composer-shell input:focus {\n            outline: none;\n            box-shadow: none;\n        }\n\n        .top-channel {\n            position: relative;\n            display: inline-flex;\n            align-items: center;\n            justify-content: center;\n            overflow: hidden;\n            min-height: 34px;\n            padding: 0 12px;\n            border-radius: 9999px;\n            color: var(--color-on-surface-variant, #434654);\n            font-weight: 600;\n            transition: transform 0.25s cubic-bezier(0.4, 0, 0.2, 1), color 0.25s ease-out, background 0.25s ease-out, box-shadow 0.25s ease-out;\n        }\n\n        .top-channel::after {\n            content: '';\n            position: absolute;\n            top: 0;\n            left: -140%;\n            width: 42%;\n            height: 100%;\n            background: linear-gradient(to right, rgba(255,255,255,0) 0%, rgba(255,255,255,0.45) 50%, rgba(255,255,255,0) 100%);\n            transform: skewX(-25deg);\n            transition: left 0.65s ease;\n            pointer-events: none;\n        }\n\n        .top-channel:hover {\n            transform: translateY(-2px);\n            color: var(--color-primary, #1f53c9);\n            background: rgba(255, 255, 255, 0.62);\n            box-shadow: inset 0 1px 2px rgba(255, 255, 255, 0.9), 0 8px 24px rgba(31, 83, 201, 0.10);\n        }\n\n        .top-channel.active {\n            color: var(--color-primary, #1f53c9);\n            background: linear-gradient(135deg, rgba(255, 255, 255, 0.78), rgba(236, 244, 255, 0.62));\n            backdrop-filter: blur(20px) saturate(1.25);\n            -webkit-backdrop-filter: blur(20px) saturate(1.25);\n            box-shadow: inset 0 1px 2px rgba(255, 255, 255, 0.95), inset 0 -1px 0 rgba(31, 83, 201, 0.12), 0 10px 24px rgba(31, 83, 201, 0.10);\n        }\n\n        .top-channel:hover::after,\n        .top-channel.active::after {\n            left: 160%;\n        }\n\n        .top-channel:active {\n            transform: translateY(-1px) scale(0.97);\n        }\n\n        .top-icon-btn {\n            width: 40px;\n            height: 40px;\n            display: inline-flex;\n            align-items: center;\n            justify-content: center;\n            transform: translateY(2px);\n            background: rgba(255, 255, 255, 0.16);\n            border: 1px solid rgba(255, 255, 255, 0.22);\n            box-shadow: inset 0 1px 1px rgba(255,255,255,0.45);\n        }\n\n        .top-icon-btn:hover {\n            transform: translateY(0);\n            background: rgba(255, 255, 255, 0.42);\n            box-shadow: inset 0 1px 1px rgba(255,255,255,0.70), 0 8px 18px rgba(31, 83, 201, 0.10);\n        }\n\n        .glass-button-primary {\n            background: linear-gradient(135deg, rgba(64, 109, 228, 0.9) 0%, rgba(31, 83, 201, 0.9) 100%);\r\n            backdrop-filter: blur(10px);\r\n            box-shadow: 0 4px 15px rgba(31, 83, 201, 0.3), inset 0 1px 1px rgba(255, 255, 255, 0.4);\r\n            border: 1px solid rgba(255, 255, 255, 0.2);\r\n            transition: all 0.3s ease-out;\r\n        }\r\n        \r\n        .glass-button-primary:hover {\r\n            transform: translateY(-1px);\r\n            box-shadow: 0 6px 20px rgba(31, 83, 201, 0.4), inset 0 1px 1px rgba(255, 255, 255, 0.5);\r\n        }\r\n\r\n        .glass-button-ghost {\r\n            background: rgba(255, 255, 255, 0.3);\r\n            backdrop-filter: blur(10px);\r\n            border: 1px solid rgba(255, 255, 255, 0.6);\r\n            transition: all 0.3s ease-out;\r\n        }\r\n\r\n        .glass-button-ghost:hover {\r\n            background: rgba(255, 255, 255, 0.5);\r\n            border-color: rgba(255, 255, 255, 0.8);\r\n        }\r\n\r\n        /* Hover Lift and Shine Effects */\r\n        .hover-lift {\r\n            transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);\r\n        }\r\n        .hover-lift:hover {\r\n            transform: translateY(-4px);\r\n            box-shadow: inset 0 1px 2px rgba(255, 255, 255, 0.9), 0 12px 30px rgba(0, 0, 0, 0.08);\r\n            border-color: rgba(255, 255, 255, 0.9);\r\n        }\r\n\r\n        .shine-effect {\r\n            position: relative;\r\n            overflow: hidden;\r\n        }\r\n        .shine-effect::after {\r\n            content: '';\r\n            position: absolute;\r\n            top: 0;\r\n            left: -150%;\r\n            width: 50%;\r\n            height: 100%;\r\n            background: linear-gradient(to right, rgba(255,255,255,0) 0%, rgba(255,255,255,0.4) 50%, rgba(255,255,255,0) 100%);\r\n            transform: skewX(-25deg);\r\n            transition: left 0.7s ease;\r\n            z-index: 10;\r\n            pointer-events: none;\r\n        }\r\n        .shine-effect:hover::after {\r\n            left: 200%;\r\n        }\r\n\r\n        /* Ambient Background */\r\n        body {\n            background-color: #eef2f6;\n            background-image: \n                radial-gradient(circle at 15% 50%, rgba(180, 197, 255, 0.4) 0%, transparent 50%),\n                radial-gradient(circle at 85% 30%, rgba(219, 226, 250, 0.5) 0%, transparent 50%);\n            background-attachment: fixed;\n            min-height: 100vh;\n        }\n\n        .feed-transition {\n            opacity: 0;\n            transform: translateY(8px) scale(0.995);\n            filter: blur(8px);\n            transition:\n                opacity 0.28s ease,\n                transform 0.34s cubic-bezier(0.22, 1, 0.36, 1),\n                filter 0.34s ease;\n        }\n\n        .feed-transition.feed-ready {\n            opacity: 1;\n            transform: none;\n            filter: none;\n        }\n\n        .feed-transition.feed-leaving {\n            opacity: 0;\n            transform: translateY(6px) scale(0.996);\n            filter: blur(7px);\n            pointer-events: none;\n        }\n\n        @media (prefers-reduced-motion: reduce) {\n            .feed-transition,\n            .feed-transition.feed-ready,\n            .feed-transition.feed-leaving {\n                opacity: 1;\n                transform: none;\n                filter: none;\n                transition: none;\n            }\n        }\n\n        .chart-glow-line {\n            filter: drop-shadow(0 4px 6px rgba(31, 83, 201, 0.2));\n        }\n\r\n        .nav-link-hover {\r\n            transition: all 0.2s ease-out;\r\n        }\r\n        .nav-link-hover:hover {\n            background: rgba(255, 255, 255, 0.4);\n            border-radius: 0.75rem;\n        }\n\n        .profile-tab {\n            position: relative;\n            border: 1px solid transparent;\n            background: transparent;\n            color: rgba(67, 70, 84, 0.92);\n            font-weight: 700;\n            transition: color 0.16s ease, background-color 0.16s ease, border-color 0.16s ease;\n        }\n\n        .profile-tab::after {\n            content: \"\";\n            position: absolute;\n            left: 1rem;\n            right: 1rem;\n            bottom: 0;\n            height: 2px;\n            border-radius: 999px;\n            background: transparent;\n            transition: all 0.28s ease;\n        }\n\n        .profile-tab.active,\n        .profile-tab:hover {\n            color: var(--color-primary, #1f53c9);\n            background: rgba(255, 255, 255, 0.54);\n            border: 1px solid rgba(255, 255, 255, 0.38);\n            box-shadow: inset 0 0 0 1px rgba(255,255,255,0.28);\n            transform: none;\n        }\n\n        .profile-tab.active::after,\n        .profile-tab:hover::after {\n            background: transparent;\n        }\n\n        .profile-stat-line {\n            color: rgba(67, 70, 84, 0.92);\n        }\n\n        .profile-stat-line strong {\n            color: #191c1e;\n            font-family: \"Hanken Grotesk\", sans-serif;\n            font-size: 17px;\n            font-weight: 700;\n        }";

const profileHtmlClass = "light";
const profileBodyClass = "text-on-surface font-body-md antialiased overflow-x-hidden selection:bg-primary-container selection:text-on-primary-container";

export function ProfilePage() {
    const profileKey = parseProfileKey(window.location.pathname);

    if (profileKey === "me") {
        return createElement(MeProfilePage);
    }

    return createElement(UserProfilePage, { key: profileKey, routeUserId: profileKey });
}

function MeProfilePage() {
    const [state, setState] = useState<MeProfileState>(() => readAuthSession() ? { status: "loading" } : { status: "auth-required" });
    const [publishedFeed, setPublishedFeed] = useState<UserPublishedFeedState>({ status: "loading" });
    const [favoriteFeed, setFavoriteFeed] = useState<UserPublishedFeedState>({ status: "loading" });
    const [followers, setFollowers] = useState<ProfileRelationState>({ status: "loading" });
    const [activeTab, setActiveTab] = useState<ProfileTab>("published");
    useContentInteractionVersion();

    useEffect(() => {
        if (!readAuthSession()) {
            return;
        }

        let isCurrent = true;

        getMe<MeProfileResponse>()
            .then((response) => {
                if (!isCurrent) {
                    return;
                }

                const profile = normalizeMeProfile(response);
                setState({ status: "ready", profile });

                if (!profile.userId) {
                    setPublishedFeed({ status: "error" });
                    setFavoriteFeed({ status: "error" });
                    setFollowers({ status: "error" });
                    return;
                }

                loadProfileLists(profile.userId, () => isCurrent, {
                    setPublishedFeed,
                    setFavoriteFeed,
                    setFollowers
                });
            })
            .catch(() => {
                if (!isCurrent) {
                    return;
                }

                setState(readAuthSession() ? { status: "error" } : { status: "auth-required" });
            });

        return () => {
            isCurrent = false;
        };
    }, []);

    if (state.status !== "ready") {
        return renderMeStatePage(state);
    }

    return renderMeReadyPage(state.profile, {
        activeTab,
        setActiveTab,
        publishedFeed,
        favoriteFeed,
        followers
    });
}

function UserProfilePage({ routeUserId }: { routeUserId: string }) {
    const [state, setState] = useState<UserProfileState>({ status: "loading" });
    const [publishedFeed, setPublishedFeed] = useState<UserPublishedFeedState>({ status: "loading" });
    const [favoriteFeed, setFavoriteFeed] = useState<UserPublishedFeedState>({ status: "loading" });
    const [followers, setFollowers] = useState<ProfileRelationState>({ status: "loading" });
    const [activeTab, setActiveTab] = useState<ProfileTab>("published");
    useContentInteractionVersion();

    useEffect(() => {
        let isCurrent = true;

        getUserProfile<UserProfileResponse>(routeUserId)
            .then((response) => {
                if (!isCurrent) {
                    return;
                }

                const profile = normalizeUserProfile(response, routeUserId);
                setState({ status: "ready", profile });

                loadProfileLists(profile.userId, () => isCurrent, {
                    setPublishedFeed,
                    setFavoriteFeed,
                    setFollowers
                });
            })
            .catch((error: unknown) => {
                if (!isCurrent) {
                    return;
                }

                setState(error instanceof ApiError && error.status === 404 ? { status: "not-found" } : { status: "error" });
            });

        return () => {
            isCurrent = false;
        };
    }, [routeUserId]);

    if (state.status !== "ready") {
        return renderUserStatePage(state);
    }

    return renderUserReadyPage(state.profile, {
        activeTab,
        setActiveTab,
        publishedFeed,
        favoriteFeed,
        followers
    });
}

function loadProfileLists(
    userId: string,
    isCurrent: () => boolean,
    setters: {
        setPublishedFeed: (state: UserPublishedFeedState) => void;
        setFavoriteFeed: (state: UserPublishedFeedState) => void;
        setFollowers: (state: ProfileRelationState) => void;
    }
) {
    getUserPublishedFeed<UserPublishedFeedResponse>({ user_id: userId, cursor: "", page_size: 20 })
        .then((feedResponse) => {
            if (!isCurrent()) {
                return;
            }
            setters.setPublishedFeed(toContentListState(feedResponse));
        })
        .catch(() => {
            if (isCurrent()) {
                setters.setPublishedFeed({ status: "error" });
            }
        });

    getUserFavoriteFeed<UserPublishedFeedResponse>({ user_id: userId, cursor: "", page_size: 20 })
        .then((feedResponse) => {
            if (!isCurrent()) {
                return;
            }
            setters.setFavoriteFeed(toContentListState(feedResponse));
        })
        .catch(() => {
            if (isCurrent()) {
                setters.setFavoriteFeed({ status: "error" });
            }
        });

    getUserFollowers<ProfileRelationResponse>({ user_id: userId, cursor: 0, page_size: 20 })
        .then((response) => {
            if (!isCurrent()) {
                return;
            }
            setters.setFollowers(toRelationListState(response));
        })
        .catch(() => {
            if (isCurrent()) {
                setters.setFollowers({ status: "error" });
            }
        });
}

function toContentListState(response: UserPublishedFeedResponse): UserPublishedFeedState {
    const items = response.items ?? [];
    return items.length > 0 ? { status: "ready", items } : { status: "empty" };
}

function toRelationListState(response: ProfileRelationResponse): ProfileRelationState {
    const items = response.items ?? [];
    return items.length > 0 ? { status: "ready", items } : { status: "empty" };
}

function renderUserStatePage(state: Exclude<UserProfileState, { status: "ready" }>) {
    const title = state.status === "not-found" ? "用户不存在" : "用户主页";

    return createElement(
        PageShell,
        { title: `zfeed - ${title}`, htmlClass: profileHtmlClass, bodyClass: profileBodyClass, styles },
        createElement("div", { className: "page-root" },
            createElement("main", { className: "min-h-screen px-4 py-6 md:px-6 md:py-10" },
                createElement("section", { className: "glass-panel feed-transition feed-ready mx-auto w-full max-w-3xl rounded-[28px] p-6 md:p-8" },
                    createElement("a", { className: "text-label-sm text-primary", href: "/home" }, "返回首页"),
                    createElement(PageState, getUserStateView(state.status))
                )
            )
        )
    );
}

function renderUserReadyPage(profile: UserProfile, tabs: ProfileTabsState) {
    return createElement(
        PageShell,
        { title: `zfeed - ${profile.nickname}`, htmlClass: profileHtmlClass, bodyClass: profileBodyClass, styles },
        createElement("div", { className: "page-root" },
            renderUserHeader(profile),
            createElement("div", { className: "pt-24 px-4 md:px-6 max-w-[1600px] mx-auto pb-safe" },
                createElement("div", { className: "grid grid-cols-4 md:grid-cols-8 lg:grid-cols-12 gap-5" },
                    renderMeLeftRail(),
                    createElement("main", { className: "feed-transition col-span-4 md:col-span-8 lg:col-span-7 flex flex-col gap-4 pb-24 feed-ready" },
                        createElement("section", { className: "overflow-hidden" },
                            createElement("div", { className: "px-0 pb-2" },
                                createElement("div", { className: "relative z-20 flex flex-col md:flex-row md:items-end justify-between gap-5" },
                                    createElement("div", { className: "flex min-w-0 items-end gap-4" },
                                        renderMeAvatar(profile, "w-24 h-24 rounded-full border-4 border-white shadow-lg object-cover shrink-0"),
                                        createElement("div", { className: "min-w-0 pb-1" },
                                            createElement("h1", { className: "break-words font-display text-[34px] leading-tight text-on-surface" }, profile.nickname),
                                            createElement("div", { className: "flex flex-wrap items-center gap-3 text-on-surface-variant mt-1" },
                                                createElement("span", { className: "font-label-sm" }, profile.handle),
                                                createElement("span", { className: "w-1 h-1 rounded-full bg-outline-variant" }),
                                                createElement("span", { className: "font-label-sm" }, "创作者主页")
                                            )
                                        )
                                    ),
                                    createElement("div", { className: "flex flex-wrap gap-2 md:mb-2 self-start md:self-auto" },
                                        createElement("button", {
                                            className: "glass-button-primary text-white font-label-sm px-5 py-2.5 rounded-full active:scale-95 transition-all duration-300",
                                            "data-user-id": profile.userId,
                                            type: "button"
                                        }, profile.isFollowing ? "已关注" : "关注")
                                    )
                                ),
                                createElement("p", { className: "break-words font-body-md text-on-surface-variant mt-2" }, profile.bio),
                                renderMeStats(profile, tabs)
                            )
                        ),
                        renderProfileTabs(tabs, false)
                    ),
                    renderMeRightRail(profile)
                )
            )
        )
    );
}

function renderUserPublishedCard(item: UserPublishedFeedItem) {
    const syncedItem = mergeContentInteractionState(item);
    const contentId = String(syncedItem.content_id);
    const authorId = String(syncedItem.author_id);
    const scene = getContentScene(syncedItem.content_type);

    return createElement("article", {
        className: "glass-panel hover-lift shine-effect rounded-3xl p-5 md:p-6",
        "data-content-scene": scene,
        key: contentId
    },
        createElement("div", { className: "relative z-20 flex min-w-0 items-start gap-3" },
            renderUserAvatar(
                { avatar: syncedItem.author_avatar, nickname: syncedItem.author_name, userId: syncedItem.author_id },
                "h-11 w-11 shrink-0 rounded-full border border-white object-cover shadow-sm",
                { alt: syncedItem.author_name, textClassName: "text-[14px]" }
            ),
            createElement("div", { className: "min-w-0 flex-1" },
                createElement("div", { className: "flex min-w-0 flex-wrap items-center gap-2" },
                    createElement("a", { className: "truncate font-headline-md text-[15px] text-on-surface hover:text-primary", href: `/user/${authorId}` }, syncedItem.author_name),
                    createElement("span", { className: "font-meta-xs text-on-surface-variant" }, "·"),
                    createElement("span", { className: "font-meta-xs text-on-surface-variant" }, formatPublishedAt(syncedItem.published_at))
                ),
                createElement("a", { className: "mt-3 block", href: `/content/${contentId}` },
                    createElement("h2", { className: "break-words font-display text-[22px] font-bold leading-tight text-on-surface md:text-[26px]" }, syncedItem.title),
                    syncedItem.description
                        ? createElement("p", { className: "mt-3 line-clamp-3 break-words text-[15px] leading-7 text-on-surface-variant" }, syncedItem.description)
                        : null
                ),
                syncedItem.cover_url
                    ? createElement("a", { href: `/content/${contentId}` },
                        createElement("img", {
                            alt: "",
                            className: "mt-4 aspect-[16/9] w-full rounded-2xl border border-white/45 object-cover shadow-sm",
                            src: syncedItem.cover_url
                        })
                    )
                    : null,
                createElement("div", { className: "mt-5 flex flex-wrap items-center gap-3" },
                    createElement("button", {
                        "aria-label": syncedItem.is_liked ? "取消点赞" : "点赞",
                        className: `glass-button-ghost rounded-full px-4 py-2 flex items-center gap-2 font-label-sm active:scale-95 transition-all duration-300 ${syncedItem.is_liked ? "text-error" : "text-on-surface-variant"}`,
                        "data-content-id": contentId,
                        "data-content-scene": scene,
                        "data-content-user-id": authorId,
                        type: "button"
                    },
                        createElement("span", { className: "material-symbols-outlined text-[18px]" }, "favorite"),
                        createElement("span", null, formatPublishedCount(syncedItem.like_count))
                    ),
                    createElement("button", {
                        "aria-label": syncedItem.is_favorited ? "取消收藏" : "收藏",
                        className: `glass-button-ghost rounded-full px-4 py-2 flex items-center gap-2 font-label-sm active:scale-95 transition-all duration-300 ${syncedItem.is_favorited ? "text-primary" : "text-on-surface-variant"}`,
                        "data-content-id": contentId,
                        "data-content-scene": scene,
                        type: "button"
                    },
                        createElement("span", { className: "material-symbols-outlined text-[18px]" }, "bookmark"),
                        createElement("span", null, formatPublishedCount(syncedItem.favorite_count))
                    ),
                    createElement("a", {
                        className: "glass-button-ghost rounded-full px-4 py-2 flex items-center gap-2 font-label-sm text-on-surface-variant active:scale-95 transition-all duration-300",
                        href: `/content/${contentId}`
                    },
                        createElement("span", { className: "material-symbols-outlined text-[18px]" }, "chat_bubble"),
                        createElement("span", null, formatPublishedCount(syncedItem.comment_count))
                    )
                )
            )
        )
    );
}

function renderProfileTabs(tabs: ProfileTabsState, isMe: boolean) {
    return renderProfileTabPanel(tabs, isMe);
}

function renderProfileTabPanel(tabs: ProfileTabsState, isMe: boolean) {
    if (tabs.activeTab === "favorites") {
        return renderProfileContentList(tabs.favoriteFeed, "favorites", isMe);
    }
    if (tabs.activeTab === "followers") {
        return renderProfileRelationList(tabs.followers);
    }
    return renderProfileContentList(tabs.publishedFeed, "published", isMe);
}

function renderProfileContentList(state: UserPublishedFeedState, kind: ProfileContentListKind, isMe: boolean) {
    if (state.status !== "ready") {
        return createElement("section", { className: "glass-panel rounded-3xl p-5 hover-lift shine-effect", role: "tabpanel" },
            createElement("div", { className: "flex items-center justify-between gap-4" },
                createElement("div", null,
                    createElement("h2", { className: "font-headline-md text-on-surface" }, getProfileContentTitle(kind, isMe)),
                    createElement("p", { className: "mt-1 text-[14px] leading-6 text-on-surface-variant" }, getProfileContentDescription(kind, isMe))
                ),
                renderProfileContentAction(state, kind, isMe)
            ),
            createElement(PageState, getProfileContentStateView(state.status, kind, isMe))
        );
    }

    return createElement("section", { className: "flex flex-col gap-4", role: "tabpanel" },
        createElement("div", { className: "glass-panel rounded-3xl p-5 hover-lift shine-effect" },
            createElement("div", { className: "flex items-center justify-between gap-4" },
                createElement("div", null,
                    createElement("h2", { className: "font-headline-md text-on-surface" }, getProfileContentTitle(kind, isMe)),
                    createElement("p", { className: "mt-1 text-[14px] leading-6 text-on-surface-variant" }, getProfileContentDescription(kind, isMe))
                ),
                renderProfileContentAction(state, kind, isMe)
            )
        ),
        ...state.items.map(renderUserPublishedCard)
    );
}

function renderProfileContentAction(state: UserPublishedFeedState, kind: ProfileContentListKind, isMe: boolean) {
    if (kind === "published" && isMe) {
        return createElement("a", { className: "glass-button-ghost rounded-full px-4 py-2 text-primary font-label-sm active:scale-95", href: "/compose" }, "去发布");
    }
    if (state.status === "empty") {
        return createElement("a", { className: "glass-button-ghost rounded-full px-4 py-2 text-primary font-label-sm active:scale-95", href: "/search" }, "继续探索");
    }
    return null;
}

function renderProfileRelationList(state: ProfileRelationState) {
    if (state.status !== "ready") {
        return createElement("section", { className: "glass-panel rounded-3xl p-5 hover-lift shine-effect", role: "tabpanel" },
            createElement("h2", { className: "font-headline-md text-on-surface" }, "粉丝"),
            createElement(PageState, getProfileRelationStateView(state.status))
        );
    }

    return createElement("section", { className: "flex flex-col gap-3", role: "tabpanel" },
        createElement("div", { className: "glass-panel rounded-3xl p-5 hover-lift shine-effect" },
            createElement("h2", { className: "font-headline-md text-on-surface" }, "粉丝"),
            createElement("p", { className: "mt-1 text-[14px] leading-6 text-on-surface-variant" }, "关注这个用户的人。")
        ),
        ...state.items.map(renderProfileRelationCard)
    );
}

function renderProfileRelationCard(item: ProfileRelationItem) {
    const userId = String(item.user_id);
    const nickname = item.nickname || `用户 ${userId}`;

    return createElement("article", { className: "glass-panel hover-lift shine-effect rounded-3xl p-5", key: userId },
        createElement("div", { className: "relative z-20 flex min-w-0 items-center justify-between gap-4" },
            createElement("a", { className: "flex min-w-0 items-center gap-3", href: `/user/${userId}` },
                renderUserAvatar(
                    { avatar: item.avatar, nickname, userId },
                    "h-12 w-12 shrink-0 rounded-full border border-white object-cover shadow-sm",
                    { alt: nickname, textClassName: "text-[15px]" }
                ),
                createElement("span", { className: "min-w-0" },
                    createElement("span", { className: "block truncate font-headline-md text-[15px] text-on-surface" }, nickname),
                    createElement("span", { className: "mt-1 block line-clamp-2 break-words text-[13px] leading-5 text-on-surface-variant" }, item.bio || "这个用户还没有填写简介。")
                )
            ),
            createElement("button", {
                className: `glass-button-ghost rounded-full px-4 py-2 font-label-sm active:scale-95 transition-all duration-300 ${item.is_following ? "text-on-surface-variant" : "text-primary"}`,
                "data-user-id": userId,
                type: "button"
            }, item.is_following ? "已关注" : "关注")
        )
    );
}

function getProfileContentTitle(kind: ProfileContentListKind, isMe: boolean) {
    if (kind === "favorites") {
        return isMe ? "我的收藏" : "收藏内容";
    }
    return isMe ? "最近动态" : "发布内容";
}

function getProfileContentDescription(kind: ProfileContentListKind, isMe: boolean) {
    if (kind === "favorites") {
        return isMe ? "你收藏的公开内容。" : "这个用户收藏的公开内容。";
    }
    return isMe ? "你的公开发布内容。" : "创作者公开发布的内容。";
}

function renderUserHeader(profile: UserProfile) {
    return createElement("header", { className: "fixed top-0 w-full z-50 flex items-center justify-between px-6 py-3 bg-white/40 dark:bg-black/40 backdrop-blur-[50px] border-b border-white/20 saturate-[180%] shadow-sm border-white/30 shadow-md" },
        createElement("div", { className: "flex items-center gap-4" },
            createElement("a", { className: "flex items-center gap-3 hover:opacity-80 transition-opacity duration-300", href: "/home" },
                createElement("div", { className: "w-8 h-8 rounded-full bg-gradient-to-br from-primary-container to-primary flex items-center justify-center shadow-md" },
                    createElement("div", { className: "w-3 h-3 bg-white rounded-full" })
                ),
                createElement("span", { className: "font-display text-[24px] tracking-tight font-bold text-primary dark:text-primary-fixed" }, "zfeed")
            ),
            createElement("div", { className: "hidden md:flex ml-8 relative w-96 group md:ml-16 search-shell overflow-hidden" },
                createElement("span", { className: "material-symbols-outlined absolute left-3 top-1/2 -translate-y-1/2 text-on-surface-variant text-[20px] group-focus-within:text-primary transition-colors duration-300" }, "search"),
                createElement("input", { className: "glass-input w-full pl-10 pr-4 rounded-full font-body-md text-on-surface placeholder:text-on-surface-variant/70 border-white/40 bg-white/30 backdrop-blur-md focus:bg-white/60 transition-colors py-2.5", placeholder: "搜索内容、创作者或话题", type: "text" })
            )
        ),
        createElement("nav", { className: "hidden md:flex items-center absolute left-1/2 -translate-x-1/2 gap-5" },
            createElement("a", { className: "top-channel text-label-sm tracking-wide", href: "/home" }, "推荐"),
            createElement("a", { className: "top-channel text-label-sm tracking-wide", href: "/following" }, "关注"),
            createElement("a", { className: "top-channel text-label-sm tracking-wide", href: "/search" }, "搜索")
        ),
        createElement("div", { className: "flex items-center gap-5 pr-2 md:pr-12" },
            createElement("button", { className: "hidden sm:flex glass-button-primary text-white font-label-sm py-2 rounded-full items-center gap-2 active:scale-95 px-6", type: "button" },
                createElement("span", { className: "material-symbols-outlined text-[18px]" }, "add"),
                "发布"
            ),
            renderMeAvatar(profile, "w-8 h-8 rounded-full border-2 border-white shadow-sm object-cover")
        )
    );
}

function renderMeStatePage(state: Exclude<MeProfileState, { status: "ready" }>) {
    return createElement(
        PageShell,
        { title: "zfeed - 我的主页", htmlClass: profileHtmlClass, bodyClass: profileBodyClass, styles },
        createElement("div", { className: "page-root" },
            createElement("main", { className: "min-h-screen px-4 py-6 md:px-6 md:py-10" },
                createElement("section", { className: "glass-panel feed-transition feed-ready mx-auto w-full max-w-3xl rounded-[28px] p-6 md:p-8" },
                    createElement("a", { className: "text-label-sm text-primary", href: "/home" }, "返回首页"),
                    createElement("h1", { className: "mt-5 font-display text-[34px] leading-tight text-on-surface" }, "我的主页"),
                    createElement(PageState, getMeStateView(state.status))
                )
            )
        )
    );
}

function renderMeReadyPage(profile: MeProfile, tabs: ProfileTabsState) {
    return createElement(
        PageShell,
        { title: `zfeed - ${profile.nickname}`, htmlClass: profileHtmlClass, bodyClass: profileBodyClass, styles },
        createElement("div", { className: "page-root" },
            renderMeHeader(profile),
            createElement("div", { className: "pt-24 px-4 md:px-6 max-w-[1600px] mx-auto pb-safe" },
                createElement("div", { className: "grid grid-cols-4 md:grid-cols-8 lg:grid-cols-12 gap-5" },
                    renderMeLeftRail(),
                    createElement("main", { className: "feed-transition col-span-4 md:col-span-8 lg:col-span-7 flex flex-col gap-4 pb-24 feed-ready" },
                        createElement("section", { className: "overflow-hidden" },
                            createElement("div", { className: "px-0 pb-2" },
                                createElement("div", { className: "relative z-20 flex flex-col md:flex-row md:items-end justify-between gap-5" },
                                    createElement("div", { className: "flex min-w-0 items-end gap-4" },
                                        renderMeAvatar(profile, "w-24 h-24 rounded-full border-4 border-white shadow-lg object-cover shrink-0"),
                                        createElement("div", { className: "min-w-0 pb-1" },
                                            createElement("div", { className: "flex min-w-0 items-center gap-2" },
                                                createElement("h1", { className: "break-words font-display text-[34px] leading-tight text-on-surface" }, profile.nickname),
                                                createElement("span", { className: "material-symbols-outlined text-primary text-[20px]", style: { fontVariationSettings: "'FILL' 1" } }, "verified")
                                            ),
                                            createElement("div", { className: "flex flex-wrap items-center gap-3 text-on-surface-variant mt-1" },
                                                createElement("span", { className: "font-label-sm" }, profile.handle),
                                                createElement("span", { className: "w-1 h-1 rounded-full bg-outline-variant" }),
                                                createElement("span", { className: "font-label-sm" }, "当前登录用户")
                                            )
                                        )
                                    ),
                                    createElement("div", { className: "flex flex-wrap gap-2 md:mb-2 self-start md:self-auto" },
                                        createElement("button", { className: "glass-button-primary text-white font-label-sm px-5 py-2.5 rounded-full active:scale-95 transition-all duration-300 flex items-center gap-2", type: "button" },
                                            createElement("span", { className: "material-symbols-outlined text-[18px]" }, "add"),
                                            "发布"
                                        ),
                                        createElement("a", { className: "glass-button-ghost text-primary border-primary/20 font-label-sm px-5 py-2.5 rounded-full active:scale-95 transition-all duration-300", href: "/me/edit" }, "编辑资料")
                                    )
                                ),
                                createElement("p", { className: "break-words font-body-md text-on-surface-variant mt-2" }, profile.bio),
                                renderMeStats(profile, tabs)
                            )
                        ),
                        renderProfileTabs(tabs, true)
                    ),
                    renderMeRightRail(profile)
                )
            )
        )
    );
}

function renderMeHeader(profile: MeProfile) {
    return createElement("header", { className: "fixed top-0 w-full z-50 flex items-center justify-between px-6 py-3 bg-white/40 dark:bg-black/40 backdrop-blur-[50px] border-b border-white/20 saturate-[180%] shadow-sm border-white/30 shadow-md" },
        createElement("div", { className: "flex items-center gap-4" },
            createElement("a", { className: "flex items-center gap-3 hover:opacity-80 transition-opacity duration-300", href: "/home" },
                createElement("div", { className: "w-8 h-8 rounded-full bg-gradient-to-br from-primary-container to-primary flex items-center justify-center shadow-md" },
                    createElement("div", { className: "w-3 h-3 bg-white rounded-full" })
                ),
                createElement("span", { className: "font-display text-[24px] tracking-tight font-bold text-primary dark:text-primary-fixed" }, "zfeed")
            ),
            createElement("div", { className: "hidden md:flex ml-8 relative w-96 group md:ml-16 search-shell overflow-hidden" },
                createElement("span", { className: "material-symbols-outlined absolute left-3 top-1/2 -translate-y-1/2 text-on-surface-variant text-[20px] group-focus-within:text-primary transition-colors duration-300" }, "search"),
                createElement("input", { className: "glass-input w-full pl-10 pr-4 rounded-full font-body-md text-on-surface placeholder:text-on-surface-variant/70 border-white/40 bg-white/30 backdrop-blur-md focus:bg-white/60 transition-colors py-2.5", placeholder: "搜索内容、创作者或话题", type: "text" })
            )
        ),
        createElement("nav", { className: "hidden md:flex items-center absolute left-1/2 -translate-x-1/2 gap-5" },
            createElement("a", { className: "top-channel text-label-sm tracking-wide", href: "/home" }, "推荐"),
            createElement("a", { className: "top-channel text-label-sm tracking-wide", href: "/following" }, "关注"),
            createElement("a", { className: "top-channel text-label-sm tracking-wide", href: "/search" }, "搜索")
        ),
        createElement("div", { className: "flex items-center gap-5 pr-2 md:pr-12" },
            createElement("button", { className: "hidden sm:flex glass-button-primary text-white font-label-sm py-2 rounded-full items-center gap-2 active:scale-95 px-6", type: "button" },
                createElement("span", { className: "material-symbols-outlined text-[18px]" }, "add"),
                "发布"
            ),
            renderMeAvatar(profile, "w-8 h-8 rounded-full border-2 border-white shadow-sm object-cover")
        )
    );
}

function renderMeLeftRail() {
    return createElement("aside", { className: "hidden lg:block lg:col-span-2 relative" },
        createElement("div", { className: "sticky top-24 w-full flex flex-col gap-8" },
            createElement("nav", { className: "flex flex-col gap-1" },
                renderMeRailLink("/home", "home", "推荐"),
                renderMeRailLink("/following", "person_add", "关注"),
                renderMeRailLink("/search", "search", "搜索"),
                renderMeRailLink("/settings", "settings", "设置")
            )
        )
    );
}

function renderMeRailLink(href: string, icon: string, label: string) {
    return createElement("a", { className: "flex items-center gap-3 text-on-surface-variant p-2 nav-link-hover rounded-lg active:scale-95", href },
        createElement("span", { className: "material-symbols-outlined" }, icon),
        createElement("span", { className: "font-label-sm text-[13px]" }, label)
    );
}

function renderMeStats(profile: MeProfile, tabs: ProfileTabsState) {
    return createElement("div", { className: "profile-stat-line flex flex-wrap items-center gap-x-3 gap-y-1 mt-2 font-label-sm" },
        renderMeStatButton(profile.contentCount, "动态", "published", tabs),
        createElement("span", { className: "text-outline/70" }, "·"),
        renderMeStat(profile.followeeCount, "关注"),
        createElement("span", { className: "text-outline/70" }, "·"),
        renderMeStatButton(profile.followerCount, "粉丝", "followers", tabs),
        createElement("span", { className: "text-outline/70" }, "·"),
        renderMeStat(profile.likeReceivedCount, "获赞"),
        createElement("span", { className: "text-outline/70" }, "·"),
        renderMeStatButton(profile.favoriteReceivedCount, "收藏", "favorites", tabs)
    );
}

function renderMeStatButton(value: number, label: string, tab: ProfileTab, tabs: ProfileTabsState) {
    const isActive = tabs.activeTab === tab;
    return createElement("button", {
        "aria-pressed": isActive,
        className: `profile-tab rounded-full px-2 py-1 -mx-2 -my-1 active:scale-95 ${isActive ? "active" : ""}`,
        onClick: () => tabs.setActiveTab(tab),
        type: "button"
    },
        createElement("strong", null, formatProfileCount(value)),
        ` ${label}`
    );
}

function renderMeStat(value: number, label: string) {
    return createElement("span", null,
        createElement("strong", null, formatProfileCount(value)),
        ` ${label}`
    );
}

function renderMeRightRail(profile: MeProfile) {
    return createElement("aside", { className: "hidden lg:block lg:col-span-3 relative" },
        createElement("div", { className: "sticky top-24 flex flex-col gap-5" },
            createElement("div", { className: "glass-panel rounded-3xl p-5 hover-lift shine-effect" },
                createElement("h3", { className: "font-headline-md text-[16px] text-on-surface mb-3" }, "创作数据"),
                createElement("div", { className: "grid grid-cols-2 gap-3 text-center" },
                    renderMeMetric("内容", profile.contentCount),
                    renderMeMetric("粉丝", profile.followerCount),
                    renderMeMetric("获赞", profile.likeReceivedCount),
                    renderMeMetric("收藏", profile.favoriteReceivedCount)
                )
            ),
            createElement("div", { className: "glass-panel rounded-3xl p-5 hover-lift shine-effect" },
                createElement("h3", { className: "font-headline-md text-[16px] text-on-surface mb-3" }, "资料完整度"),
                createElement("p", { className: "text-[14px] leading-6 text-on-surface-variant" }, "昵称、头像和简介会展示在主页、评论和 Feed 中。")
            )
        )
    );
}

function renderMeMetric(label: string, value: number) {
    return createElement("div", { className: "rounded-2xl bg-white/35 px-3 py-3" },
        createElement("div", { className: "font-display text-[20px] font-bold text-on-surface" }, formatProfileCount(value)),
        createElement("div", { className: "font-meta-xs text-on-surface-variant" }, label)
    );
}

function renderMeAvatar(profile: MeProfile, className: string) {
    return renderUserAvatar(
        { avatar: profile.avatar, nickname: profile.nickname, userId: profile.userId },
        className,
        { alt: profile.nickname, textClassName: "text-[20px]" }
    );
}

function getMeStateView(status: Exclude<MeProfileState["status"], "ready">) {
    if (status === "auth-required") {
        return {
            state: "auth-required" as const,
            description: "登录后才能查看我的主页。",
            actionHref: `/login?next=${encodeURIComponent("/me")}`,
            actionLabel: "去登录"
        };
    }

    if (status === "loading") {
        return {
            state: "loading" as const,
            title: "正在加载我的主页",
            description: "正在获取你的个人资料和统计。"
        };
    }

    return {
        state: "error" as const,
        title: "我的主页加载失败",
        description: "请稍后重试。"
    };
}

function getUserStateView(status: Exclude<UserProfileState["status"], "ready">) {
    if (status === "loading") {
        return {
            state: "loading" as const,
            title: "正在加载用户主页",
            description: "正在获取创作者资料和统计。"
        };
    }

    if (status === "not-found") {
        return {
            state: "error" as const,
            title: "用户不存在",
            description: "这个用户不存在或已不可访问。",
            actionHref: "/home",
            actionLabel: "返回首页"
        };
    }

    return {
        state: "error" as const,
        title: "用户主页加载失败",
        description: "请稍后重试。"
    };
}

function getProfileContentStateView(status: Exclude<UserPublishedFeedState["status"], "ready">, kind: ProfileContentListKind, isMe: boolean) {
    if (status === "loading") {
        return {
            state: "loading" as const,
            title: kind === "favorites" ? "正在加载收藏内容" : "正在加载发布内容",
            description: kind === "favorites" ? "正在获取收藏的公开内容。" : "正在获取创作者公开发布的内容。"
        };
    }

    if (status === "empty") {
        return {
            state: "empty" as const,
            title: kind === "favorites" ? "还没有收藏内容" : "还没有公开发布内容",
            description: kind === "favorites" ? "收藏内容后会显示在这里。" : isMe ? "发布第一条动态后会显示在这里。" : "可以继续探索其他创作者和话题。",
            actionHref: kind === "published" && isMe ? "/compose" : "/search",
            actionLabel: kind === "published" && isMe ? "去发布" : "去搜索"
        };
    }

    return {
        state: "error" as const,
        title: kind === "favorites" ? "收藏内容加载失败" : "发布内容加载失败",
        description: "请稍后重试。"
    };
}

function getProfileRelationStateView(status: Exclude<ProfileRelationState["status"], "ready">) {
    if (status === "loading") {
        return {
            state: "loading" as const,
            title: "正在加载粉丝列表",
            description: "正在获取用户关系数据。"
        };
    }

    if (status === "empty") {
        return {
            state: "empty" as const,
            title: "还没有粉丝",
            description: "有人关注后会显示在这里。",
            actionHref: "/search",
            actionLabel: "去搜索"
        };
    }

    return {
        state: "error" as const,
        title: "粉丝列表加载失败",
        description: "请稍后重试。"
    };
}

function normalizeMeProfile(response: MeProfileResponse): MeProfile {
    const user = resolveProfileUser(response);
    const counts = resolveProfileCounts(response);
    const nickname = cleanProfileText(user.nickname) || "zfeed 用户";
    return {
        userId: cleanProfileText(user.user_id),
        nickname,
        avatar: cleanProfileText(user.avatar),
        bio: cleanProfileText(user.bio) || "这个用户还没有填写简介。",
        handle: getProfileHandle(user),
        contentCount: counts.contentCount,
        followeeCount: counts.followeeCount,
        followerCount: counts.followerCount,
        likeReceivedCount: counts.likeReceivedCount,
        favoriteReceivedCount: counts.favoriteReceivedCount
    };
}

function normalizeUserProfile(response: UserProfileResponse, routeUserId: string): UserProfile {
    const user = resolveProfileUser(response);
    const profile = normalizeMeProfile(response);
    return {
        ...profile,
        userId: cleanProfileText(user.user_id) || routeUserId,
        isFollowing: response.viewer?.is_following === true || response.is_following === true
    };
}

function resolveProfileUser(response: MeProfileResponse) {
    return response.user_info ?? response.user_profile ?? {};
}

function resolveProfileCounts(response: MeProfileResponse) {
    const counts = response.counts ?? {};
    return {
        contentCount: normalizeProfileCount(counts.content_count ?? response.content_count),
        followeeCount: normalizeProfileCount(counts.followee_count ?? response.followee_count),
        followerCount: normalizeProfileCount(counts.follower_count ?? response.follower_count),
        likeReceivedCount: normalizeProfileCount(counts.like_received_count ?? response.like_received_count),
        favoriteReceivedCount: normalizeProfileCount(counts.favorite_received_count ?? response.favorite_received_count)
    };
}

function getProfileHandle(user: ProfileUserInfo) {
    const raw = cleanProfileText(user.mobile) || cleanProfileText(user.email) || cleanProfileText(user.user_id);
    return raw ? `@${raw}` : "@me";
}

function cleanProfileText(value: unknown) {
    return typeof value === "string" || typeof value === "number" ? String(value).trim() : "";
}

function normalizeProfileCount(value: unknown) {
    return typeof value === "number" && Number.isFinite(value) ? Math.max(0, value) : 0;
}

function formatPublishedAt(publishedAt?: number) {
    if (!publishedAt) {
        return "刚刚";
    }

    return new Intl.DateTimeFormat("zh-CN", {
        year: "numeric",
        month: "2-digit",
        day: "2-digit"
    }).format(new Date(publishedAt * 1000));
}

function formatPublishedCount(count?: number) {
    return String(count ?? 0);
}

function formatProfileCount(value: number) {
    return String(value);
}

function parseProfileKey(pathname: string) {
    const fromPath = pathname.startsWith("/user/") ? pathname.split("/")[2] : null;
    if (fromPath) {
        try {
            return decodeURIComponent(fromPath);
        } catch {
            return fromPath;
        }
    }

    return "me";
}
