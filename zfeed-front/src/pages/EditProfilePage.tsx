import { createElement, useEffect, useState } from "react";
import { ApiError, getMe, searchUsers } from "../runtime/apiClient";
import { PageShell } from "../runtime/PageShell";
import { readAuthSession } from "../runtime/authStore";
import { renderUserAvatar } from "./avatar";
import { PageState } from "./PageState";
import { sharedGlassBodyClass, sharedGlassStyles } from "./sharedGlassStyles";

type ProfileUserInfo = {
    user_id?: string | number;
    mobile?: string;
    nickname?: string;
    avatar?: string;
    bio?: string;
    gender?: number;
    status?: number;
    email?: string;
    birthday?: number;
};

type MeProfileResponse = {
    user_info?: ProfileUserInfo;
};

type EditProfile = {
    userId: string;
    mobile: string;
    nickname: string;
    avatar: string;
    bio: string;
    email: string;
    gender: number;
    birthday: number;
};

type SuggestedUser = {
    user_id: string | number;
    nickname: string;
    avatar?: string;
    bio?: string;
    is_following?: boolean;
};

type SearchUsersResponse = {
    items?: SuggestedUser[];
};

type EditProfileState =
    | { status: "auth-required" }
    | { status: "loading" }
    | { status: "ready"; profile: EditProfile; suggestedUsers: SuggestedUser[] }
    | { status: "error" };

const profileStyles = `${sharedGlassStyles}
.search-shell {
    border-radius: 9999px;
    transition: transform 0.3s cubic-bezier(0.4, 0, 0.2, 1), filter 0.3s ease-out;
}

.search-shell:hover,
.search-shell:focus-within {
    transform: translateY(-2px);
    filter: drop-shadow(0 10px 22px rgba(31, 83, 201, 0.14));
}

.top-channel {
    position: relative;
    display: inline-flex;
    align-items: center;
    justify-content: center;
    overflow: hidden;
    min-height: 34px;
    padding: 0 12px;
    border-radius: 9999px;
    color: #434654;
    font-weight: 600;
    transition: transform 0.25s ease, color 0.25s ease, background 0.25s ease, box-shadow 0.25s ease;
}

.top-channel:hover {
    transform: translateY(-2px);
    color: #1f53c9;
    background: rgba(255, 255, 255, 0.62);
    box-shadow: inset 0 1px 2px rgba(255, 255, 255, 0.9), 0 8px 24px rgba(31, 83, 201, 0.10);
}

.top-icon-btn {
    width: 40px;
    height: 40px;
    display: inline-flex;
    align-items: center;
    justify-content: center;
    transform: translateY(2px);
    background: rgba(255, 255, 255, 0.16);
    border: 1px solid rgba(255, 255, 255, 0.22);
    box-shadow: inset 0 1px 1px rgba(255, 255, 255, 0.45);
}

.top-icon-btn:hover {
    transform: translateY(0);
    background: rgba(255, 255, 255, 0.42);
    box-shadow: inset 0 1px 1px rgba(255, 255, 255, 0.70), 0 8px 18px rgba(31, 83, 201, 0.10);
}

.nav-link-hover {
    transition: all 0.2s ease-out;
}

.nav-link-hover:hover {
    background: rgba(255, 255, 255, 0.4);
    border-radius: 0.75rem;
}

.edit-field {
    background: rgba(255, 255, 255, 0.38);
    border: 1px solid rgba(255, 255, 255, 0.56);
    box-shadow: inset 0 1px 2px rgba(255, 255, 255, 0.72);
    transition: border-color 0.18s ease, background-color 0.18s ease, box-shadow 0.18s ease;
}

.edit-field:focus {
    outline: none;
    border-color: rgba(31, 83, 201, 0.42);
    background: rgba(255, 255, 255, 0.62);
    box-shadow: inset 0 1px 2px rgba(255, 255, 255, 0.86), 0 0 0 3px rgba(31, 83, 201, 0.10);
}

.edit-label {
    color: rgba(67, 70, 84, 0.78);
    font-family: Inter, ui-sans-serif, system-ui, sans-serif;
    font-size: 12px;
    font-weight: 600;
}

.edit-chip {
    background: rgba(255, 255, 255, 0.44);
    border: 1px solid rgba(255, 255, 255, 0.54);
    box-shadow: inset 0 1px 1px rgba(255, 255, 255, 0.70);
}
`;

export function EditProfilePage() {
    const [state, setState] = useState<EditProfileState>(() => readAuthSession() ? { status: "loading" } : { status: "auth-required" });

    useEffect(() => {
        if (!readAuthSession()) {
            setState({ status: "auth-required" });
            return;
        }

        let isCurrent = true;

        getMe<MeProfileResponse>()
            .then(async (response) => {
                const profile = normalizeEditProfile(response);
                const suggestedUsers = await loadSuggestedUsers(profile.userId);
                if (isCurrent) {
                    setState({ status: "ready", profile, suggestedUsers });
                }
            })
            .catch((error: unknown) => {
                if (!isCurrent) {
                    return;
                }

                setState({ status: isAuthError(error) ? "auth-required" : "error" });
            });

        return () => {
            isCurrent = false;
        };
    }, []);

    return createElement(
        PageShell,
        {
            title: "zfeed - 编辑资料",
            htmlClass: "light",
            bodyClass: sharedGlassBodyClass,
            styles: profileStyles
        },
        createElement("div", { className: "page-root" },
            renderHeader(state.status === "ready" ? state.profile : null),
            createElement("div", { className: "pt-24 px-4 md:px-6 max-w-[1600px] mx-auto pb-safe" },
                createElement("div", { className: "grid grid-cols-4 md:grid-cols-8 lg:grid-cols-12 gap-5" },
                    renderLeftRail(),
                    createElement("main", { className: "feed-transition feed-ready col-span-4 md:col-span-8 lg:col-span-7 flex flex-col gap-4 pb-24" },
                        renderMainContent(state)
                    ),
                    renderRightRail(state.status === "ready" ? state.suggestedUsers : [])
                )
            )
        )
    );
}

function renderHeader(profile: EditProfile | null) {
    return createElement("header", { className: "fixed top-0 w-full z-50 flex items-center justify-between px-6 py-3 bg-white/40 dark:bg-black/40 backdrop-blur-[50px] border-b border-white/20 saturate-[180%] shadow-sm border-white/30 shadow-md" },
        createElement("div", { className: "flex items-center gap-4" },
            createElement("button", { className: "md:hidden p-2 text-on-surface-variant hover:bg-white/20 active:scale-95 rounded-full transition-all duration-300 ease-out", type: "button" },
                createElement("span", { className: "material-symbols-outlined" }, "menu")
            ),
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
            createElement("a", { className: "top-channel text-label-sm tracking-wide", href: "/following" }, "关注"),
            createElement("a", { className: "top-channel text-label-sm tracking-wide", href: "/home" }, "趋势"),
            createElement("a", { className: "top-channel text-label-sm tracking-wide", href: "/home" }, "最新"),
            createElement("a", { className: "top-channel text-label-sm tracking-wide", href: "/search" }, "搜索")
        ),
        createElement("div", { className: "flex items-center gap-5 pr-12" },
            createElement("button", { className: "top-icon-btn relative text-on-surface-variant active:scale-95 rounded-full transition-all duration-300 ease-out", type: "button" },
                createElement("span", { className: "material-symbols-outlined" }, "notifications"),
                createElement("span", { className: "absolute top-1 right-1 w-2 h-2 bg-error rounded-full border border-white" })
            ),
            createElement("a", { className: "hidden sm:flex glass-button-primary text-white font-label-sm py-2 rounded-full items-center gap-2 active:scale-95 px-6", href: "/compose" },
                createElement("span", { className: "material-symbols-outlined text-[18px]" }, "add"),
                "发布"
            ),
            profile
                ? renderUserAvatar(profile, "w-8 h-8 rounded-full border-2 border-white shadow-sm object-cover cursor-pointer hover:scale-105 transition-transform duration-300 ease-out", { alt: "用户头像", textClassName: "text-[12px]" })
                : createElement("div", { className: "w-8 h-8 rounded-full border-2 border-white bg-white/55 shadow-sm" })
        )
    );
}

function renderLeftRail() {
    return createElement("aside", { className: "hidden lg:block lg:col-span-2 relative" },
        createElement("div", { className: "sticky top-24 w-full flex flex-col gap-8" },
            createElement("nav", { className: "flex flex-col gap-1" },
                renderRailLink("home", "推荐", "/home"),
                renderRailLink("person_add", "关注", "/following"),
                renderRailLink("search", "搜索", "/search"),
                renderRailLink("bookmark", "收藏", "/settings")
            ),
            createElement("div", { className: "flex flex-col gap-2" },
                createElement("span", { className: "font-meta-xs text-on-surface-variant/70 uppercase tracking-wider px-3 mb-1" }, "频道"),
                renderChannel("A", "AI 与机器学习", "bg-blue-100 text-blue-600"),
                renderChannel("D", "设计与灵感", "bg-emerald-100 text-emerald-600"),
                renderChannel("P", "产品与增长", "bg-purple-100 text-purple-600")
            ),
            createElement("div", { className: "mt-auto flex flex-col gap-1 border-t border-white/40 pt-4" },
                renderRailLink("settings", "设置", "/settings")
            )
        )
    );
}

function renderMainContent(state: EditProfileState) {
    if (state.status !== "ready") {
        return createElement("section", { className: "glass-panel rounded-3xl p-6 md:p-8" },
            createElement("a", { className: "inline-flex items-center gap-2 text-on-surface-variant hover:text-primary transition-colors mb-3", href: "/me" },
                createElement("span", { className: "material-symbols-outlined text-[18px]" }, "arrow_back"),
                createElement("span", { className: "font-label-sm" }, "返回主页")
            ),
            createElement("h1", { className: "font-display text-[34px] leading-tight text-on-surface" }, "编辑资料"),
            createElement(PageState, getProfileStateView(state.status))
        );
    }

    const { profile } = state;

    return [
        createElement("section", { className: "flex items-center justify-between gap-4", key: "heading" },
            createElement("div", null,
                createElement("a", { className: "inline-flex items-center gap-2 text-on-surface-variant hover:text-primary transition-colors mb-3", href: "/me" },
                    createElement("span", { className: "material-symbols-outlined text-[18px]" }, "arrow_back"),
                    createElement("span", { className: "font-label-sm" }, "返回主页")
                ),
                createElement("h1", { className: "font-display text-[34px] leading-tight text-on-surface" }, "编辑资料"),
                createElement("p", { className: "font-body-md text-on-surface-variant mt-1" }, "更新你的公开身份、简介和展示信息。")
            ),
            createElement("div", { className: "flex gap-2" },
                createElement("a", { className: "glass-button-ghost text-primary border-primary/20 font-label-sm px-5 py-2.5 rounded-full active:scale-95 transition-all duration-300", href: "/me" }, "取消"),
                createElement("button", { className: "glass-button-primary text-white font-label-sm px-5 py-2.5 rounded-full active:scale-95 transition-all duration-300 flex items-center gap-2", type: "button" },
                    createElement("span", { className: "material-symbols-outlined text-[18px]" }, "check"),
                    "保存"
                )
            )
        ),
        createElement("section", { className: "grid grid-cols-1 xl:grid-cols-[1fr_280px] gap-5", key: "form" },
            renderProfileForm(profile),
            renderProfilePreview(profile)
        ),
        createElement("section", { className: "glass-panel rounded-3xl p-5", key: "security" },
            createElement("div", { className: "flex items-center justify-between gap-4" },
                createElement("div", null,
                    createElement("div", { className: "font-headline-md text-on-surface" }, "账号安全"),
                    createElement("div", { className: "font-body-md text-on-surface-variant mt-1" }, "手机号、邮箱和头像来自当前登录账号。")
                ),
                createElement("a", { className: "glass-button-ghost text-primary border-primary/20 font-label-sm px-4 py-2 rounded-full active:scale-95 transition-all duration-300", href: "/settings" }, "管理")
            )
        )
    ];
}

function renderProfileForm(profile: EditProfile) {
    return createElement("form", { className: "glass-panel rounded-3xl p-6 flex flex-col gap-5", "data-profile-edit": "true" },
        createElement("div", { className: "flex items-center gap-4 pb-5 border-b border-white/30" },
            createElement("div", { className: "relative" },
                renderUserAvatar(profile, "w-24 h-24 rounded-full border-4 border-white shadow-lg object-cover", { alt: profile.nickname, textClassName: "text-[26px]" }),
                createElement("button", { className: "absolute -right-1 bottom-1 w-9 h-9 rounded-full glass-button-primary text-white flex items-center justify-center active:scale-95 transition-all duration-300", type: "button" },
                    createElement("span", { className: "material-symbols-outlined text-[18px]" }, "photo_camera")
                )
            ),
            createElement("div", { className: "min-w-0" },
                createElement("div", { className: "font-headline-md text-on-surface" }, "头像与资料"),
                createElement("div", { className: "font-body-md text-on-surface-variant mt-1" }, "默认头像使用账号对应数字，展示在主页、评论和 Feed 中。")
            )
        ),
        createElement("div", { className: "grid grid-cols-1 md:grid-cols-2 gap-4" },
            renderInput("昵称", "nickname", "text", profile.nickname),
            renderInput("手机号", "mobile", "tel", profile.mobile, true),
            renderInput("头像", "avatar", "text", profile.avatar),
            renderInput("邮箱", "email", "email", profile.email)
        ),
        createElement("label", { className: "flex flex-col gap-2" },
            createElement("span", { className: "edit-label" }, "简介"),
            createElement("textarea", {
                className: "edit-field rounded-2xl px-4 py-3 font-body-md text-on-surface min-h-28 resize-none",
                defaultValue: profile.bio,
                name: "bio"
            })
        ),
        createElement("div", { className: "grid grid-cols-1 md:grid-cols-2 gap-4" },
            renderInput("性别", "gender", "number", String(profile.gender)),
            renderInput("生日时间戳", "birthday", "number", profile.birthday ? String(profile.birthday) : "")
        )
    );
}

function renderProfilePreview(profile: EditProfile) {
    return createElement("aside", { className: "glass-panel rounded-3xl p-5 h-fit" },
        createElement("div", { className: "font-headline-md text-on-surface mb-4" }, "主页预览"),
        createElement("div", { className: "flex items-center gap-3" },
            renderUserAvatar(profile, "w-14 h-14 rounded-full border-2 border-white shadow-sm object-cover", { alt: profile.nickname, textClassName: "text-[18px]" }),
            createElement("div", null,
                createElement("div", { className: "font-headline-md text-[17px] text-on-surface" }, profile.nickname),
                createElement("div", { className: "font-meta-xs text-on-surface-variant mt-1" }, profile.mobile ? `@${profile.mobile}` : `@user-${profile.userId}`)
            )
        ),
        createElement("p", { className: "font-body-md text-on-surface-variant mt-4" }, profile.bio || "这个用户还没有填写简介。"),
        createElement("div", { className: "flex flex-wrap gap-2 mt-4" },
            createElement("span", { className: "edit-chip rounded-full px-3 py-1.5 font-label-sm text-primary" }, `# ${profile.nickname}`),
            profile.avatar ? createElement("span", { className: "edit-chip rounded-full px-3 py-1.5 font-label-sm text-primary" }, `# 头像 ${profile.avatar}`) : null,
            profile.email ? createElement("span", { className: "edit-chip rounded-full px-3 py-1.5 font-label-sm text-primary" }, "# 已绑定邮箱") : null
        ),
        createElement("div", { className: "border-t border-white/30 mt-5 pt-4 flex items-center justify-between text-on-surface-variant" },
            createElement("span", { className: "font-label-sm" }, "公开可见"),
            createElement("span", { className: "material-symbols-outlined text-primary" }, "visibility")
        )
    );
}

function renderRightRail(suggestedUsers: SuggestedUser[]) {
    return createElement("aside", { className: "hidden lg:flex lg:col-span-3 flex-col gap-6" },
        createElement("div", { className: "glass-panel rounded-3xl p-5 hover-lift shine-effect" },
            createElement("div", { className: "flex items-center justify-between mb-4 relative z-20" },
                createElement("h3", { className: "font-headline-md text-[16px] text-on-surface" }, "趋势话题"),
                createElement("a", { className: "text-on-surface-variant text-[13px] hover:text-primary active:scale-95 transition-all duration-300", href: "/search" }, "查看全部")
            ),
            createElement("div", { className: "flex flex-col gap-3 relative z-20" },
                renderTrend("1", "# AI 产品趋势", "12.5k", "text-error", "arrow_upward"),
                renderTrend("2", "# 设计系统", "8.7k", "text-orange-400", "arrow_upward"),
                renderTrend("3", "# 独立开发日志", "6.3k", "text-yellow-500", "remove"),
                renderTrend("4", "# 远程工作技巧", "4.1k", "text-on-surface-variant", "arrow_upward"),
                renderTrend("5", "# 创作者经济", "3.2k", "text-on-surface-variant", "fiber_new")
            )
        ),
        createElement("div", { className: "glass-panel rounded-3xl p-5 hover-lift shine-effect" },
            createElement("div", { className: "flex items-center justify-between mb-4 relative z-20" },
                createElement("h3", { className: "font-headline-md text-[16px] text-on-surface" }, "为你推荐"),
                createElement("button", { className: "flex items-center gap-1 text-on-surface-variant text-[13px] hover:text-primary active:scale-95 transition-all duration-300 group", type: "button" },
                    createElement("span", { className: "material-symbols-outlined text-[16px] group-hover:rotate-180 transition-transform duration-500" }, "refresh"),
                    "换一批"
                )
            ),
            createElement("div", { className: "flex flex-col gap-4 relative z-20" },
                suggestedUsers.length > 0
                    ? suggestedUsers.map(renderSuggestedUser)
                    : createElement("div", { className: "rounded-2xl bg-white/35 px-4 py-4 text-[13px] text-on-surface-variant" }, "暂无推荐用户")
            )
        )
    );
}

function renderSuggestedUser(user: SuggestedUser) {
    const userId = String(user.user_id);
    return createElement("div", { className: "flex items-center justify-between group", key: userId },
        createElement("a", { className: "flex min-w-0 items-center gap-3", href: `/user/${userId}` },
            renderUserAvatar(
                { avatar: user.avatar, nickname: user.nickname, userId },
                "w-10 h-10 rounded-full border border-white object-cover group-hover:scale-105 transition-transform duration-300 shadow-sm",
                { alt: user.nickname, textClassName: "text-[13px]" }
            ),
            createElement("div", { className: "min-w-0 flex flex-col cursor-pointer" },
                createElement("span", { className: "truncate font-headline-md text-[14px] group-hover:text-primary transition-colors" }, user.nickname),
                createElement("span", { className: "truncate font-meta-xs text-on-surface-variant" }, user.bio || "这个用户还没有填写简介。")
            )
        ),
        createElement("button", {
            className: "glass-button-ghost px-4 py-1.5 rounded-full font-label-sm text-primary border-primary/20 hover:border-primary active:scale-95 transition-all duration-300",
            "data-user-id": userId,
            type: "button"
        }, user.is_following ? "已关注" : "关注")
    );
}

function renderRailLink(icon: string, label: string, href: string) {
    return createElement("a", { className: "flex items-center gap-3 text-on-surface-variant p-2 nav-link-hover rounded-lg active:scale-95", href },
        createElement("span", { className: "material-symbols-outlined" }, icon),
        createElement("span", { className: "font-label-sm text-[13px]" }, label)
    );
}

function renderChannel(letter: string, label: string, colorClassName: string) {
    return createElement("a", { className: "flex items-center gap-3 text-on-surface-variant p-2 nav-link-hover rounded-lg active:scale-95", href: "/home" },
        createElement("div", { className: `w-6 h-6 rounded-md ${colorClassName} flex items-center justify-center font-label-sm transition-colors duration-300` }, letter),
        createElement("span", { className: "font-body-md text-[13px]" }, label)
    );
}

function renderInput(label: string, name: string, type: string, value: string, readOnly = false) {
    return createElement("label", { className: "flex flex-col gap-2" },
        createElement("span", { className: "edit-label" }, label),
        createElement("input", {
            className: "edit-field rounded-2xl px-4 py-3 font-body-md text-on-surface disabled:text-on-surface-variant",
            defaultValue: value,
            name,
            readOnly,
            type
        })
    );
}

function renderTrend(rank: string, topic: string, count: string, rankClass: string, icon: string) {
    return createElement("a", { className: "flex items-center justify-between group cursor-pointer hover:bg-white/40 p-2 -mx-2 rounded-xl transition-colors duration-300", href: `/search?q=${encodeURIComponent(topic.replace("# ", ""))}` },
        createElement("div", { className: "flex items-center gap-3" },
            createElement("span", { className: `font-headline-md ${rankClass} w-4 text-center` }, rank),
            createElement("span", { className: "font-body-md text-on-surface group-hover:text-primary transition-colors" }, topic),
            createElement("span", { className: "material-symbols-outlined text-on-surface-variant text-[16px] group-hover:-translate-y-1 transition-transform duration-300" }, icon)
        ),
        createElement("span", { className: "font-meta-xs text-on-surface-variant" }, count)
    );
}

function getProfileStateView(status: Exclude<EditProfileState["status"], "ready">) {
    if (status === "auth-required") {
        return {
            state: "auth-required" as const,
            description: "登录后才能编辑资料。",
            actionHref: `/login?next=${encodeURIComponent("/me/edit")}`,
            actionLabel: "去登录"
        };
    }

    if (status === "loading") {
        return {
            state: "loading" as const,
            title: "正在加载资料",
            description: "正在获取当前账号信息。"
        };
    }

    return {
        state: "error" as const,
        title: "编辑资料加载失败",
        description: "请稍后重试。"
    };
}

function normalizeEditProfile(response: MeProfileResponse): EditProfile {
    const user = response.user_info ?? {};
    const userId = cleanProfileText(user.user_id);
    const nickname = cleanProfileText(user.nickname) || "zfeed 用户";
    return {
        userId,
        mobile: cleanProfileText(user.mobile),
        nickname,
        avatar: cleanProfileText(user.avatar),
        bio: cleanProfileText(user.bio),
        email: cleanProfileText(user.email),
        gender: Number.isFinite(user.gender) ? Number(user.gender) : 0,
        birthday: Number.isFinite(user.birthday) ? Number(user.birthday) : 0
    };
}

async function loadSuggestedUsers(currentUserId: string) {
    try {
        const response = await searchUsers<SearchUsersResponse>({ query: "测试", page_size: 4, mode: "relevance" });
        return (response.items ?? [])
            .filter((user) => String(user.user_id) !== currentUserId)
            .slice(0, 3);
    } catch {
        return [];
    }
}

function cleanProfileText(value: unknown) {
    if (typeof value === "string" || typeof value === "number") {
        return String(value).trim();
    }

    return "";
}

function isAuthError(error: unknown) {
    return error instanceof ApiError && (error.status === 401 || error.status === 403);
}
