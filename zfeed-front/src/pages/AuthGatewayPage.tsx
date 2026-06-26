import { createElement, useEffect, useState } from "react";
import { PageShell } from "../runtime/PageShell";
import { getMe, ApiError } from "../runtime/apiClient";
import { authSessionStorageKey, clearAuthSession, readAuthSession, saveAuthSession } from "../runtime/authStore";
import type { AuthSession } from "../runtime/authStore";
import { navigateTo } from "../runtime/navigation";
import { AuthHomeBackdrop } from "./AuthHomeBackdrop";
import { sharedGlassBodyClass, sharedGlassStyles } from "./sharedGlassStyles";

const restoreTimeoutMs = 5_000;
export function AuthGatewayPage() {
    const [hadStoredSession] = useState(() => window.localStorage.getItem(authSessionStorageKey) !== null);
    const [sessionToRestore] = useState<AuthSession | null>(() => readAuthSession());
    const isRestoring = sessionToRestore !== null;

    useEffect(() => {
        if (!sessionToRestore) {
            navigateTo(hadStoredSession ? "/home" : "/login");
            return;
        }

        let isActive = true;
        const restoreTimer = window.setTimeout(() => {
            if (!isActive) {
                return;
            }

            isActive = false;
            clearAuthSession();
            navigateTo("/home");
        }, restoreTimeoutMs);

        getMe<{
            user_info: {
                user_id: number;
                nickname: string;
                avatar: string;
            };
        }>()
            .then((response) => {
                if (!isActive) {
                    return;
                }

                saveAuthSession({
                    ...sessionToRestore,
                    user: {
                        userId: response.user_info.user_id,
                        nickname: response.user_info.nickname,
                        avatar: response.user_info.avatar
                    }
                });
                window.clearTimeout(restoreTimer);
                isActive = false;
                navigateTo("/home");
            })
            .catch((error: unknown) => {
                if (!isActive) {
                    return;
                }

                window.clearTimeout(restoreTimer);
                isActive = false;
                if (isAuthError(error)) {
                    clearAuthSession();
                }
                navigateTo("/home");
            });

        return () => {
            isActive = false;
            window.clearTimeout(restoreTimer);
        };
    }, [hadStoredSession, sessionToRestore]);

    return createElement(
        PageShell,
        {
            title: "zfeed - 启动中",
            htmlClass: "light",
            bodyClass: sharedGlassBodyClass,
            styles: sharedGlassStyles
        },
        createElement("div", { className: "page-root auth-gateway-root" },
            createElement(AuthHomeBackdrop),
            createElement("main", {
                className:
                    "relative z-10 flex min-h-screen items-end justify-center px-4 py-4 md:items-center md:px-6 md:py-8"
            },
                createElement("section", {
                    className:
                        "auth-sheet feed-transition feed-ready w-full max-w-[460px] overflow-hidden rounded-t-[28px] rounded-b-none p-6 md:rounded-[28px] md:p-8",
                    "data-page-state": "loading"
                },
                    createElement("div", { className: "flex items-center gap-3" },
                        createElement("div", {
                            className:
                                "flex h-12 w-12 items-center justify-center rounded-2xl bg-gradient-to-br from-primary-container to-primary text-white shadow-md"
                        }, "z"),
                        createElement("div", null,
                            createElement("div", { className: "font-display text-[26px] leading-none text-on-surface" }, "zfeed"),
                            createElement("div", { className: "mt-1 font-meta-xs text-on-surface-variant" }, "继续浏览、关注和发布")
                        )
                    ),
                    createElement("div", { className: "mt-7" },
                        createElement("h1", {
                            className: "font-display text-[34px] leading-tight text-on-surface md:text-[38px]"
                        }, isRestoring ? "正在恢复 zfeed 会话" : "正在进入 zfeed"),
                        createElement("p", {
                            className: "mt-3 text-[15px] leading-7 text-on-surface-variant"
                        }, isRestoring ? "最多等待 5 秒，失效或超时后会自动进入游客首页。" : "正在打开登录入口。")
                    )
                )
            )
        )
    );
}

function isAuthError(error: unknown) {
    return error instanceof ApiError && (error.status === 401 || error.status === 403);
}
