import { createElement, useEffect, useState } from "react";
import { readAuthSession, getAdminRole } from "../../runtime/authStore";
import { adminGetSettings, adminUpdateSettings } from "../../runtime/apiClient";
import { showToast } from "../../runtime/toast";
import { AdminLayout } from "./AdminLayout";

type SettingsData = {
    settings: Record<string, string>;
};

type PageState =
    | { status: "loading" }
    | { status: "ready"; config: { enableRecommend: boolean; enableSearchIndex: boolean; contentReviewMode: "post" | "pre" } }
    | { status: "error"; message: string };

export function AdminSettingsPage() {
    const session = readAuthSession();
    const role = getAdminRole();
    const [state, setState] = useState<PageState>({ status: "loading" });
    const [saving, setSaving] = useState(false);

    useEffect(() => {
        adminGetSettings<SettingsData>()
            .then((data) => {
                const s = data.settings ?? {};
                setState({
                    status: "ready",
                    config: {
                        enableRecommend: s.enable_recommend !== "false",
                        enableSearchIndex: s.enable_search_index !== "false",
                        contentReviewMode: (s.content_review_mode === "pre" ? "pre" : "post"),
                    },
                });
            })
            .catch((err) => {
                setState({ status: "error", message: err instanceof Error ? err.message : "加载设置失败" });
            });
    }, []);

    const updateConfig = (patch: Partial<{ enableRecommend: boolean; enableSearchIndex: boolean; contentReviewMode: "post" | "pre" }>) => {
        if (state.status !== "ready") return;
        const newConfig = { ...state.config, ...patch };
        setState({ ...state, config: newConfig });

        const settingsPayload: Record<string, string> = {
            enable_recommend: String(newConfig.enableRecommend),
            enable_search_index: String(newConfig.enableSearchIndex),
            content_review_mode: newConfig.contentReviewMode,
        };

        setSaving(true);
        adminUpdateSettings<{ updated: number }>({ settings: settingsPayload })
            .then(() => {
                showToast("设置已保存");
            })
            .catch((err) => {
                showToast(err instanceof Error ? err.message : "保存失败");
                // Revert on error
                adminGetSettings<SettingsData>().then((data) => {
                    const s = data.settings ?? {};
                    setState({
                        status: "ready",
                        config: {
                            enableRecommend: s.enable_recommend !== "false",
                            enableSearchIndex: s.enable_search_index !== "false",
                            contentReviewMode: (s.content_review_mode === "pre" ? "pre" : "post"),
                        },
                    });
                });
            })
            .finally(() => setSaving(false));
    };

    const config = state.status === "ready" ? state.config : { enableRecommend: true, enableSearchIndex: true, contentReviewMode: "post" as const };

    return createElement(AdminLayout, { currentPath: "/admin/settings", adminName: session?.user?.nickname },
        createElement("div", { className: "space-y-5" },
            createElement("h2", { className: "text-xl font-bold text-slate-800" }, "⚙️ 系统设置"),

            // Admin info card
            createElement("div", { className: "bg-white rounded-xl shadow-sm border border-slate-200 p-5" },
                createElement("h3", { className: "text-sm font-semibold text-slate-700 mb-3" }, "🔐 管理员信息"),
                createElement("div", { className: "grid grid-cols-2 gap-3 text-sm" },
                    createElement("div", { className: "flex justify-between py-1.5 border-b border-slate-100" },
                        createElement("span", { className: "text-slate-500" }, "昵称"),
                        createElement("span", { className: "font-medium text-slate-700" }, session?.user?.nickname || "-"),
                    ),
                    createElement("div", { className: "flex justify-between py-1.5 border-b border-slate-100" },
                        createElement("span", { className: "text-slate-500" }, "角色"),
                        createElement("span", { className: `font-medium ${role >= 2 ? "text-purple-600" : "text-blue-600"}` },
                            role >= 2 ? "超级管理员" : role >= 1 ? "管理员" : "未知",
                        ),
                    ),
                    createElement("div", { className: "flex justify-between py-1.5 border-b border-slate-100" },
                        createElement("span", { className: "text-slate-500" }, "Token 有效期"),
                        createElement("span", { className: "font-medium text-slate-700" },
                            session?.expiredAt ? new Date(session.expiredAt * 1000).toLocaleString("zh-CN") : "-",
                        ),
                    ),
                    createElement("div", { className: "flex justify-between py-1.5" },
                        createElement("span", { className: "text-slate-500" }, "API 模式"),
                        createElement("span", { className: `font-medium text-xs ${state.status === "ready" ? "text-green-600" : state.status === "loading" ? "text-slate-400" : "text-red-600"}` },
                            state.status === "ready" ? "✅ 已对接" : state.status === "loading" ? "⏳ 加载中..." : "❌ 连接失败",
                        ),
                    ),
                ),
            ),

            state.status === "error" &&
                createElement("div", { className: "bg-red-50 border border-red-200 rounded-xl p-4 text-sm text-red-600" }, state.message),

            // Basic settings card
            state.status === "loading"
                ? createElement("div", { className: "bg-white rounded-xl shadow-sm border border-slate-200 p-8 text-center text-sm text-slate-400" }, "加载设置中...")
                : createElement("div", {
                    className: "bg-white rounded-xl shadow-sm border border-slate-200 divide-y divide-slate-100",
                },
                    // Toggle: enable recommend
                    createElement("div", { className: "px-6 py-4 flex items-center justify-between gap-4" },
                        createElement("div", { className: "min-w-0" },
                            createElement("p", { className: "text-sm font-medium text-slate-700" }, "个性化推荐"),
                            createElement("p", { className: "text-xs text-slate-500 mt-0.5" }, "开启多路召回 + 兴趣画像 + 排序重排"),
                        ),
                        createElement("button", {
                            onClick: () => updateConfig({ enableRecommend: !config.enableRecommend }),
                            disabled: saving,
                            className: `relative inline-flex items-center shrink-0 w-11 h-6 rounded-full transition-colors ${saving ? "opacity-50" : ""} ${
                                config.enableRecommend ? "bg-blue-600" : "bg-slate-300"
                            }`,
                        },
                            createElement("span", {
                                className: `inline-block w-5 h-5 rounded-full bg-white shadow transition-transform ${
                                    config.enableRecommend ? "translate-x-[22px]" : "translate-x-[2px]"
                                }`,
                            }),
                        ),
                    ),

                    // Toggle: search index
                    createElement("div", { className: "px-6 py-4 flex items-center justify-between gap-4" },
                        createElement("div", { className: "min-w-0" },
                            createElement("p", { className: "text-sm font-medium text-slate-700" }, "搜索引擎索引"),
                            createElement("p", { className: "text-xs text-slate-500 mt-0.5" }, "控制 OpenSearch 搜索索引的实时更新"),
                        ),
                        createElement("button", {
                            onClick: () => updateConfig({ enableSearchIndex: !config.enableSearchIndex }),
                            disabled: saving,
                            className: `relative inline-flex items-center shrink-0 w-11 h-6 rounded-full transition-colors ${saving ? "opacity-50" : ""} ${
                                config.enableSearchIndex ? "bg-blue-600" : "bg-slate-300"
                            }`,
                        },
                            createElement("span", {
                                className: `inline-block w-5 h-5 rounded-full bg-white shadow transition-transform ${
                                    config.enableSearchIndex ? "translate-x-[22px]" : "translate-x-[2px]"
                                }`,
                            }),
                        ),
                    ),

                    // Radio: content review mode
                    createElement("div", { className: "px-6 py-4" },
                        createElement("p", { className: "text-sm font-medium text-slate-700 mb-3" }, "内容审核模式"),
                        createElement("div", { className: "space-y-2" },
                            ([
                                { value: "post" as const, label: "先发后审", desc: "内容发布后进入审核队列，违规后处理" },
                                { value: "pre" as const, label: "先审后发", desc: "内容需审核通过后才能公开展示" },
                            ]).map((opt) =>
                                createElement("label", {
                                    key: opt.value,
                                    className: `flex items-start gap-3 p-3 rounded-lg border cursor-pointer transition-colors ${
                                        config.contentReviewMode === opt.value
                                            ? "border-blue-300 bg-blue-50"
                                            : "border-slate-200 hover:border-slate-300"
                                    }`,
                                },
                                    createElement("input", {
                                        type: "radio",
                                        name: "reviewMode",
                                        checked: config.contentReviewMode === opt.value,
                                        onChange: () => updateConfig({ contentReviewMode: opt.value }),
                                        disabled: saving,
                                        className: "mt-0.5",
                                    }),
                                    createElement("div", null,
                                        createElement("p", { className: "text-sm font-medium text-slate-700" }, opt.label),
                                        createElement("p", { className: "text-xs text-slate-500 mt-0.5" }, opt.desc),
                                    ),
                                ),
                            ),
                        ),
                    ),
                ),

            // Status indicator
            createElement("p", { className: "text-xs text-slate-400 mt-4 flex items-center gap-2" },
                saving
                    ? createElement("span", null, "⏳ 保存中...")
                    : createElement("span", { className: "text-green-600" }, "✅ 设置已同步至后端"),
                createElement("span", null, "· 修改开关或模式后自动保存"),
            ),
        ),
    );
}
