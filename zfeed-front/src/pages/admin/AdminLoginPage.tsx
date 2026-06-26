import { createElement, useState } from "react";
import type { ChangeEvent } from "react";
import { adminLogin } from "../../runtime/apiClient";
import { saveAuthSession } from "../../runtime/authStore";
import { ROUTES } from "../../routes/constants";

type LoginState = "idle" | "loading" | "error";

export function AdminLoginPage() {
    const [mobile, setMobile] = useState("");
    const [password, setPassword] = useState("");
    const [state, setState] = useState<LoginState>("idle");
    const [errorMsg, setErrorMsg] = useState("");

    const handleSubmit = async (e: React.FormEvent) => {
        e.preventDefault();
        if (!mobile.trim() || !password.trim()) return;

        setState("loading");
        setErrorMsg("");

        try {
            const result = await adminLogin<{
                token: string;
                expired_at: number;
                user_id: number;
                nickname: string;
                avatar: string;
                role: number;
            }>({ mobile: mobile.trim(), password });

            if (!result.role || result.role < 1) {
                setErrorMsg("此账号无管理员权限");
                setState("error");
                return;
            }

            saveAuthSession({
                token: result.token,
                expiredAt: result.expired_at,
                user: {
                    userId: typeof result.user_id === "number" ? result.user_id : 0,
                    nickname: result.nickname,
                    avatar: result.avatar,
                    role: result.role,
                },
            });

            window.location.href = ROUTES.ADMIN;
        } catch (err) {
            setErrorMsg(err instanceof Error ? err.message : "登录失败");
            setState("error");
        }
    };

    return createElement("div", {
        className: "min-h-screen bg-slate-900 flex items-center justify-center",
        style: { fontFamily: "-apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif" }
    },
        createElement("div", { className: "w-full max-w-sm mx-4" },
            createElement("div", { className: "text-center mb-8" },
                createElement("h1", { className: "text-2xl font-bold text-white" }, "ZFeed 管理后台"),
                createElement("p", { className: "mt-2 text-sm text-slate-400" }, "请使用管理员账号登录"),
            ),

            createElement("form", {
                onSubmit: handleSubmit,
                className: "bg-white rounded-xl shadow-2xl p-6 space-y-4",
            },
                createElement("div", null,
                    createElement("label", {
                        className: "block text-sm font-medium text-slate-700 mb-1",
                    }, "手机号"),
                    createElement("input", {
                        type: "text",
                        value: mobile,
                        onChange: (e: ChangeEvent<HTMLInputElement>) => setMobile(e.target.value),
                        placeholder: "请输入手机号",
                        className: "w-full px-3 py-2 border border-slate-300 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent",
                        autoFocus: true,
                    }),
                ),

                createElement("div", null,
                    createElement("label", {
                        className: "block text-sm font-medium text-slate-700 mb-1",
                    }, "密码"),
                    createElement("input", {
                        type: "password",
                        value: password,
                        onChange: (e: ChangeEvent<HTMLInputElement>) => setPassword(e.target.value),
                        placeholder: "请输入密码",
                        className: "w-full px-3 py-2 border border-slate-300 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent",
                    }),
                ),

                errorMsg
                    ? createElement("p", {
                        className: "text-sm text-red-500 bg-red-50 rounded-lg px-3 py-2",
                    }, errorMsg)
                    : null,

                createElement("button", {
                    type: "submit",
                    disabled: state === "loading",
                    className: `w-full py-2.5 rounded-lg text-sm font-medium text-white transition-colors ${
                        state === "loading"
                            ? "bg-blue-400 cursor-not-allowed"
                            : "bg-blue-600 hover:bg-blue-700"
                    }`,
                }, state === "loading" ? "登录中..." : "登 录"),
            ),

            createElement("p", { className: "mt-4 text-center text-xs text-slate-500" },
                "此入口仅限管理员使用。",
                createElement("br"),
                createElement("a", { href: "/home", className: "text-blue-400 hover:underline" }, "返回前台 →"),
            ),
        ),
    );
}
