import type { ReactNode } from "react";
import { useEffect } from "react";
import {
    type ContentScene,
    commentContent,
    deleteComment,
    editArticle,
    editVideo,
    favoriteContent,
    followUser,
    likeContent,
    publishArticle,
    unfavoriteContent,
    unfollowUser,
    unlikeContent,
    updateProfile
} from "./apiClient";
import { readAuthSession } from "./authStore";
import { restoreContentInteractionOverride, setContentReactionOverride, type ContentReactionKind } from "./contentInteractionStore";
import { navigateTo } from "./navigation";
import { showToast } from "./toast";

type PageShellProps = {
    title: string;
    htmlClass: string;
    bodyClass: string;
    styles: string;
    children?: ReactNode;
};

type ButtonStateSnapshot = {
    ariaLabel: string | null;
    className: string;
    text: string;
    valueText: string | null;
};

const maxCommentLength = 255;
const visibilityPublic = 10;

export function PageShell({ title, htmlClass, bodyClass, styles, children }: PageShellProps) {
    useEffect(() => {
        document.title = title;
        document.documentElement.lang = "zh-CN";
        document.documentElement.className = htmlClass || "light";
        document.body.className = bodyClass;

        const styleElement = document.createElement("style");
        styleElement.dataset.pageStyle = "true";
        styleElement.textContent = styles;
        document.head.append(styleElement);

        ensureLegacyBrowserApis();

        return () => {
            styleElement.remove();
        };
    }, [bodyClass, htmlClass, styles, title]);

    useEffect(() => {
        const frame = requestAnimationFrame(() => {
            document.querySelector(".feed-transition")?.classList.add("feed-ready");
        });

        const onPageShow = () => {
            document.querySelector(".feed-transition")?.classList.remove("feed-leaving");
            document.querySelector(".feed-transition")?.classList.add("feed-ready");
        };

        const onClick = (event: MouseEvent) => {
            handleSelectableClick(event);
            handleComposeClick(event);
            handleFollowClick(event);
            handleReplyClick(event);
            handleReplySubmitClick(event);
            handleDeleteCommentClick(event);
            handleCommentSubmitClick(event);
            handleContentSaveClick(event);
            handleProfileSaveClick(event);
            handlePublishClick(event);
            handleActionClick(event);
        };

        const onKeyDown = (event: KeyboardEvent) => {
            handleSearchSubmit(event);
        };

        const onFocusIn = (event: FocusEvent) => {
            if (event.target instanceof Element && event.target.matches(".search-box input")) {
                event.target.closest(".search-wrap")?.classList.add("focused");
            }
        };

        const onFocusOut = (event: FocusEvent) => {
            if (!(event.target instanceof Element) || !event.target.matches(".search-box input")) {
                return;
            }

            const wrapper = event.target.closest(".search-wrap");
            window.setTimeout(() => wrapper?.classList.remove("focused"), 160);
        };

        window.addEventListener("pageshow", onPageShow);
        document.addEventListener("click", onClick);
        document.addEventListener("keydown", onKeyDown);
        document.addEventListener("focusin", onFocusIn);
        document.addEventListener("focusout", onFocusOut);

        return () => {
            cancelAnimationFrame(frame);
            window.removeEventListener("pageshow", onPageShow);
            document.removeEventListener("click", onClick);
            document.removeEventListener("keydown", onKeyDown);
            document.removeEventListener("focusin", onFocusIn);
            document.removeEventListener("focusout", onFocusOut);
        };
    }, []);

    return children;
}

function handleSelectableClick(event: MouseEvent) {
    const target = event.target instanceof Element ? event.target : null;
    const button = target?.closest<HTMLElement>(".channel, .segment, .tab, .top-channel");
    if (!button) {
        return;
    }

    const link = button.closest<HTMLAnchorElement>("a[href]");
    const href = link?.getAttribute("href");
    if (href && href !== "#") {
        return;
    }

    event.preventDefault();
    const group = button.parentElement?.querySelectorAll<HTMLElement>(".active");
    group?.forEach((item) => item.classList.remove("active"));
    button.classList.add("active");
}

function handleActionClick(event: MouseEvent) {
    const target = event.target instanceof Element ? event.target : null;
    const button = target?.closest<HTMLButtonElement>("button");
    if (!button) {
        return;
    }

    const action = resolveContentAction(button);
    if (!action) {
        return;
    }

    event.preventDefault();

    if (button.dataset.pending === "true") {
        return;
    }

    const contentId = resolveContentId(button);
    if (!contentId) {
        return;
    }
    const contentUserId = resolveContentUserId(button);
    const scene = resolveContentScene(button);

    if (!readAuthSession()) {
        navigateToLogin();
        return;
    }

    const snapshot = captureButtonState(button);
    const matchingButtons = findMatchingContentActionButtons(contentId, action.kind);
    if (!matchingButtons.includes(button)) {
        matchingButtons.push(button);
    }
    const matchingSnapshots = matchingButtons.map((matchingButton) => ({
        button: matchingButton,
        snapshot: captureButtonState(matchingButton)
    }));
    const nextCount = Math.max(0, getActionCount(button) + (action.nextActive ? 1 : -1));
    const previousOverride = setContentReactionOverride(contentId, action.kind, action.nextActive, nextCount);
    button.dataset.pending = "true";
    button.disabled = true;
    applyOptimisticStateToButtons(matchingButtons, action, nextCount);

    const request = action.kind === "like"
        ? action.nextActive ? likeContent({ contentId, contentUserId, scene }) : unlikeContent({ contentId, scene })
        : action.nextActive ? favoriteContent({ contentId, scene }) : unfavoriteContent({ contentId, scene });

    request.catch((error: unknown) => {
        restoreButtonStates(matchingSnapshots);
        restoreButtonState(button, snapshot);
        restoreContentInteractionOverride(contentId, previousOverride);
        showToast(getContentActionFailureMessage(action));
        if (isAuthError(error)) {
            navigateToLogin();
        }
    }).finally(() => {
        button.disabled = false;
        delete button.dataset.pending;
    });
}

function handleFollowClick(event: MouseEvent) {
    const target = event.target instanceof Element ? event.target : null;
    const button = target?.closest<HTMLButtonElement>("button");
    if (!button || button.closest("form")) {
        return;
    }

    const label = button.textContent?.trim().replace(/\s+/g, "");
    if (label !== "关注" && label !== "关注作者" && label !== "已关注") {
        return;
    }

    const targetUserId = resolveTargetUserId(button);
    if (!targetUserId) {
        return;
    }

    event.preventDefault();

    if (button.dataset.pending === "true") {
        return;
    }

    if (!readAuthSession()) {
        navigateToLogin();
        return;
    }

    const snapshot = captureButtonState(button);
    const nextFollowed = label !== "已关注";
    button.dataset.pending = "true";
    button.disabled = true;
    button.textContent = nextFollowed ? "已关注" : "关注";

    const request = nextFollowed ? followUser({ target_user_id: targetUserId }) : unfollowUser({ target_user_id: targetUserId });
    request.catch((error: unknown) => {
        restoreButtonState(button, snapshot);
        if (isAuthError(error)) {
            navigateToLogin();
        }
    }).finally(() => {
        button.disabled = false;
        delete button.dataset.pending;
    });
}

function handleCommentSubmitClick(event: MouseEvent) {
    const target = event.target instanceof Element ? event.target : null;
    const button = target?.closest<HTMLButtonElement>("button");
    if (!button || button.textContent?.trim() !== "发送") {
        return;
    }

    const composer = button.closest(".comment-row")?.querySelector<HTMLInputElement>('input[placeholder="写下你的观点，补充或提问..."]');
    if (!composer) {
        return;
    }

    event.preventDefault();

    if (button.dataset.pending === "true") {
        return;
    }

    const comment = composer.value.trim();
    if (!comment) {
        showInlineStatus(button, "请输入评论内容", "error");
        return;
    }

    if (comment.length > maxCommentLength) {
        showInlineStatus(button, "评论最多 255 字", "error");
        return;
    }

    if (!readAuthSession()) {
        navigateToLogin();
        return;
    }

    const contentId = resolveNumericContentIdFromPath();
    const contentUserId = button.closest<HTMLElement>("[data-content-user-id]")?.dataset.contentUserId ?? resolveContentUserIdFromPath();
    const scene = resolveContentScene(button);
    if (!contentId || !contentUserId) {
        showInlineStatus(button, "评论失败，请重试", "error");
        return;
    }

    button.dataset.pending = "true";
    button.disabled = true;
    const previousText = button.textContent ?? "发送";
    button.textContent = "发送中";
    const pendingComment = insertPendingComment(button, comment);

    commentContent({
        content_id: contentId,
        scene,
        comment,
        content_user_id: contentUserId
    }).then(() => {
        composer.value = "";
        pendingComment?.setAttribute("data-comment-state", "sent");
        updateCommentCounters(button, 1);
        showInlineStatus(button, "评论已发送", "success");
    }).catch((error: unknown) => {
        pendingComment?.remove();
        restoreCommentEmptyState(button);
        showInlineStatus(button, "评论失败，请重试", "error");
        if (isAuthError(error)) {
            navigateToLogin();
        }
    }).finally(() => {
        button.textContent = previousText;
        button.disabled = false;
        delete button.dataset.pending;
    });
}

function handleReplyClick(event: MouseEvent) {
    const target = event.target instanceof Element ? event.target : null;
    const button = target?.closest<HTMLButtonElement>("button");
    if (!button || button.closest("[data-reply-composer]")) {
        return;
    }

    const isReplyButton = button.getAttribute("aria-label") === "回复" || Array.from(button.querySelectorAll(".material-symbols-outlined"))
        .some((icon) => icon.textContent?.trim() === "reply");
    if (!isReplyButton) {
        return;
    }

    const row = button.closest<HTMLElement>(".comment-row");
    const metadata = row ? resolveCommentMetadata(row) : null;
    if (!row || !metadata) {
        return;
    }

    event.preventDefault();

    if (!readAuthSession()) {
        navigateToLogin();
        return;
    }

    row.querySelector("[data-reply-composer]")?.remove();
    row.append(createReplyComposer(metadata.authorName));
}

function handleReplySubmitClick(event: MouseEvent) {
    const target = event.target instanceof Element ? event.target : null;
    const button = target?.closest<HTMLButtonElement>("button");
    const composer = button?.closest<HTMLElement>("[data-reply-composer]");
    if (!button || !composer || button.textContent?.trim() !== "发送") {
        return;
    }

    event.preventDefault();

    if (button.dataset.pending === "true") {
        return;
    }

    const row = composer.closest<HTMLElement>(".comment-row");
    const metadata = row ? resolveCommentMetadata(row) : null;
    const input = composer.querySelector<HTMLInputElement>("input");
    const comment = input?.value.trim() ?? "";
    if (!row || !metadata || !input) {
        return;
    }

    if (!comment) {
        showInlineStatus(button, "请输入回复内容", "error");
        return;
    }

    if (comment.length > maxCommentLength) {
        showInlineStatus(button, "回复最多 255 字", "error");
        return;
    }

    if (!readAuthSession()) {
        navigateToLogin();
        return;
    }

    const contentId = resolveNumericContentIdFromPath();
    const contentUserId = row.closest<HTMLElement>("[data-content-user-id]")?.dataset.contentUserId ?? resolveContentUserIdFromPath();
    const scene = resolveContentScene(row);
    if (!contentId || !contentUserId) {
        showInlineStatus(button, "回复失败，请重试", "error");
        return;
    }

    button.dataset.pending = "true";
    button.disabled = true;
    const previousText = button.textContent ?? "发送";
    button.textContent = "发送中";
    const pendingReply = insertPendingReply(composer, comment);

    commentContent({
        content_id: contentId,
        scene,
        comment,
        parent_id: metadata.commentId,
        root_id: metadata.rootId,
        reply_to_user_id: metadata.userId,
        content_user_id: contentUserId
    }).then(() => {
        input.value = "";
        pendingReply?.setAttribute("data-comment-state", "sent");
        updateCommentCounters(button, 1);
        showInlineStatus(button, "回复已发送", "success");
    }).catch((error: unknown) => {
        pendingReply?.remove();
        showInlineStatus(button, "回复失败，请重试", "error");
        if (isAuthError(error)) {
            navigateToLogin();
        }
    }).finally(() => {
        button.textContent = previousText;
        button.disabled = false;
        delete button.dataset.pending;
    });
}

function handleDeleteCommentClick(event: MouseEvent) {
    const target = event.target instanceof Element ? event.target : null;
    const button = target?.closest<HTMLButtonElement>('button[aria-label="删除评论"]');
    if (!button) {
        return;
    }

    const row = button.closest<HTMLElement>(".comment-row");
    const metadata = row ? resolveCommentMetadata(row) : null;
    if (!row || !metadata) {
        return;
    }

    event.preventDefault();

    if (button.dataset.pending === "true") {
        return;
    }

    if (!readAuthSession()) {
        navigateToLogin();
        return;
    }

    const contentId = resolveNumericContentIdFromPath();
    if (!contentId) {
        showInlineStatus(button, "删除失败，请重试", "error");
        return;
    }

    button.dataset.pending = "true";
    const parent = row.parentElement;
    const nextSibling = row.nextSibling;
    const counterRoot = row.closest<HTMLElement>("main") ?? document.body;
    row.remove();
    updateCommentCountersInRoot(counterRoot, -1);

    deleteComment({
        comment_id: metadata.commentId,
        content_id: contentId,
        scene: resolveContentScene(row),
        root_id: metadata.rootId
    }).catch((error: unknown) => {
        if (parent) {
            parent.insertBefore(row, nextSibling);
        }
        updateCommentCountersInRoot(counterRoot, 1);
        showInlineStatus(button, "删除失败，请重试", "error");
        if (isAuthError(error)) {
            navigateToLogin();
        }
    }).finally(() => {
        delete button.dataset.pending;
    });
}

function handleProfileSaveClick(event: MouseEvent) {
    const target = event.target instanceof Element ? event.target : null;
    const button = target?.closest<HTMLButtonElement>("button");
    if (!button || !button.textContent?.includes("保存")) {
        return;
    }

    const form = document.querySelector<HTMLFormElement>('form[data-profile-edit="true"]');
    if (!form) {
        return;
    }

    event.preventDefault();

    if (button.dataset.pending === "true") {
        return;
    }

    if (!readAuthSession()) {
        navigateToLogin();
        return;
    }

    const nickname = getNamedFieldValue(form, "nickname");
    const avatar = getNamedFieldValue(form, "avatar");
    const bio = getNamedFieldValue(form, "bio");
    const email = getNamedFieldValue(form, "email");
    const genderRaw = getNamedFieldValue(form, "gender");
    const birthdayRaw = getNamedFieldValue(form, "birthday");
    const errors = validateProfileFields({ nickname, bio, email });
    if (errors.length > 0) {
        showFormStatus(form, errors[0], "error");
        return;
    }

    button.dataset.pending = "true";
    button.disabled = true;
    const previousText = button.textContent ?? "保存";
    button.textContent = "保存中";

    updateProfile({
        nickname: optionalValue(nickname),
        avatar: optionalValue(avatar),
        bio: optionalValue(bio),
        email: optionalValue(email),
        gender: optionalNumber(genderRaw),
        birthday: optionalNumber(birthdayRaw)
    }).then(() => {
        showFormStatus(form, "资料已保存", "success");
    }).catch((error: unknown) => {
        showFormStatus(form, "保存失败，请重试", "error");
        if (isAuthError(error)) {
            navigateToLogin();
        }
    }).finally(() => {
        button.textContent = previousText;
        button.disabled = false;
        delete button.dataset.pending;
    });
}

function handleContentSaveClick(event: MouseEvent) {
    const target = event.target instanceof Element ? event.target : null;
    const button = target?.closest<HTMLButtonElement>("button");
    if (!button || button.textContent?.trim() !== "保存修改") {
        return;
    }

    const form = button.closest<HTMLElement>('[data-content-edit="true"]');
    if (!form) {
        return;
    }

    event.preventDefault();

    if (button.dataset.pending === "true") {
        return;
    }

    if (!readAuthSession()) {
        navigateToLogin();
        return;
    }

    const contentId = form.dataset.contentId ?? "";
    const contentType = form.dataset.contentType ?? "article";
    const title = getNamedFieldValue(form, "title");
    const description = getNamedFieldValue(form, "description");
    const content = getNamedFieldValue(form, "content");
    const cover = getNamedFieldValue(form, "cover");
    const videoUrl = getNamedFieldValue(form, "videoUrl") || content;
    const duration = Number(getNamedFieldValue(form, "duration"));
    const errors = validateEditContentFields({ title, description, content, contentType });
    if (!contentId) {
        showFormStatus(form, "保存失败，请重试", "error");
        return;
    }

    if (errors.length > 0) {
        showFormStatus(form, errors[0], "error");
        return;
    }

    button.dataset.pending = "true";
    button.disabled = true;
    const previousText = button.textContent ?? "保存修改";
    button.textContent = "保存中";

    const request = contentType === "video"
        ? editVideo(contentId, {
            title,
            description: optionalValue(description),
            video_url: optionalValue(videoUrl),
            cover_url: optionalValue(cover),
            duration: Number.isFinite(duration) && duration > 0 ? duration : undefined
        })
        : editArticle(contentId, {
            title,
            description: optionalValue(description),
            cover: optionalValue(cover),
            content
        });

    request.then(() => {
        window.setTimeout(() => navigateTo(`/content/${contentId}`), 0);
    }).catch((error: unknown) => {
        showFormStatus(form, "保存失败，请重试", "error");
        button.textContent = previousText;
        button.disabled = false;
        delete button.dataset.pending;
        if (isAuthError(error)) {
            navigateToLogin();
        }
    });
}

function handlePublishClick(event: MouseEvent) {
    const target = event.target instanceof Element ? event.target : null;
    const button = target?.closest<HTMLButtonElement>("button");
    if (!button || button.textContent?.trim() !== "发布") {
        return;
    }

    const form = button.closest<HTMLElement>('[data-compose-form="true"]');
    if (!form) {
        return;
    }

    event.preventDefault();

    if (button.dataset.pending === "true") {
        return;
    }

    if (!readAuthSession()) {
        navigateToLogin();
        return;
    }

    const title = getNamedFieldValue(form, "title");
    const description = getNamedFieldValue(form, "description");
    const cover = getNamedFieldValue(form, "cover");
    const content = getNamedFieldValue(form, "content");
    const errors = validatePublishFields({ title, description, content });
    if (errors.length > 0) {
        showFormStatus(form, errors[0], "error");
        return;
    }

    button.dataset.pending = "true";
    button.disabled = true;
    const previousText = button.textContent ?? "发布";
    button.textContent = "发布中";

    publishArticle<{ content_id?: number | string; contentId?: number | string }>({
        title,
        description: optionalValue(description),
        cover: optionalValue(cover),
        content,
        visibility: visibilityPublic
    }).then((result) => {
        const contentId = result.content_id ?? result.contentId;
        if (!contentId) {
            showFormStatus(form, "发布失败，请重试", "error");
            button.textContent = previousText;
            button.disabled = false;
            delete button.dataset.pending;
            return;
        }

        window.setTimeout(() => navigateTo(`/content/${contentId}`), 0);
    }).catch((error: unknown) => {
        showFormStatus(form, "发布失败，请重试", "error");
        button.textContent = previousText;
        button.disabled = false;
        delete button.dataset.pending;
        if (isAuthError(error)) {
            navigateToLogin();
        }
    });
}

function handleComposeClick(event: MouseEvent) {
    const target = event.target instanceof Element ? event.target : null;
    const button = target?.closest<HTMLButtonElement>("button");
    if (!button || button.closest("form")) {
        return;
    }

    const icon = button.querySelector(".material-symbols-outlined")?.textContent?.trim();
    if (icon !== "add" || !button.textContent?.includes("发布")) {
        return;
    }

    event.preventDefault();
    navigateTo("/compose");
}

function handleSearchSubmit(event: KeyboardEvent) {
    if (event.key !== "Enter" || !(event.target instanceof HTMLInputElement)) {
        return;
    }

    if (isHomeComposerInput(event.target)) {
        navigateTo("/compose");
        return;
    }

    if (event.target.type !== "search" && !event.target.closest(".search-shell, .search-box")) {
        return;
    }

    const query = event.target.value.trim();
    const searchTarget = getSearchTarget(event.target);
    if (!query) {
        navigateTo(getSearchPath("", searchTarget));
        return;
    }

    navigateTo(getSearchPath(query, searchTarget));
}

function getSearchTarget(input: HTMLInputElement) {
    const target = input.dataset.searchTarget;
    return target === "content" || target === "author" ? target : "";
}

function getSearchPath(query: string, target: string) {
    const params = new URLSearchParams();
    if (query) {
        params.set("q", query);
    }
    if (target && target !== "content") {
        params.set("type", target);
    }

    const search = params.toString();
    return search ? `/search?${search}` : "/search";
}

function isHomeComposerInput(input: HTMLInputElement) {
    return Boolean(input.closest(".composer-shell") && input.placeholder.includes("分享"));
}

function resolveContentAction(button: HTMLElement) {
    const ariaLabel = button.getAttribute("aria-label")?.trim();
    if (ariaLabel === "点赞" || ariaLabel === "取消点赞") {
        return { kind: "like" as const, nextActive: ariaLabel === "点赞" };
    }

    if (ariaLabel === "收藏" || ariaLabel === "取消收藏") {
        return { kind: "favorite" as const, nextActive: ariaLabel === "收藏" };
    }

    const iconNames = Array.from(button.querySelectorAll(".material-symbols-outlined, [data-lucide]"))
        .map((icon) => icon.getAttribute("data-lucide") ?? icon.textContent?.trim() ?? "");

    if (iconNames.includes("favorite") || iconNames.includes("heart")) {
        return { kind: "like" as const, nextActive: !isLikeActive(button) };
    }

    if (iconNames.includes("bookmark")) {
        return { kind: "favorite" as const, nextActive: !isFavoriteActive(button) };
    }

    return null;
}

function resolveContentId(button: HTMLElement) {
    const explicitId = button.dataset.contentId;
    if (explicitId) {
        return explicitId;
    }

    if (window.location.pathname.startsWith("/content/")) {
        return normalizeRouteId(window.location.pathname.split("/")[2] ?? "");
    }

    const container = button.closest("article, section, main") ?? document;
    const contentLink = container.querySelector<HTMLAnchorElement>('a[href^="/content/"]');
    return normalizeRouteId(contentLink?.getAttribute("href")?.split("/")[2] ?? "");
}

function resolveContentUserId(button: HTMLElement) {
    const explicitId = button.dataset.contentUserId;
    if (explicitId) {
        return explicitId;
    }

    const container = button.closest("article, section, main") ?? document;
    const authorLink = container.querySelector<HTMLAnchorElement>('a[href^="/user/"]');
    const routeId = authorLink?.getAttribute("href")?.split("/")[2] ?? "";
    return normalizeRouteId(routeId) || resolveContentUserIdFromPath();
}

function resolveTargetUserId(button: HTMLElement) {
    const explicitId = button.dataset.userId;
    if (explicitId) {
        return explicitId;
    }

    const pathname = window.location.pathname;
    if (pathname.startsWith("/user/")) {
        return normalizeRouteId(pathname.split("/")[2] ?? "");
    }

    const link = button.closest("section, article, aside, main")?.querySelector<HTMLAnchorElement>('a[href^="/user/"]');
    const routeId = link?.getAttribute("href")?.split("/")[2] ?? "";
    return normalizeRouteId(routeId);
}

function resolveNumericContentIdFromPath() {
    if (!window.location.pathname.startsWith("/content/")) {
        return "";
    }

    return normalizeRouteId(window.location.pathname.split("/")[2] ?? "");
}

function resolveContentUserIdFromPath() {
    return "";
}

function resolveContentScene(element: HTMLElement): ContentScene {
    const explicitScene = element.dataset.contentScene
        ?? element.closest<HTMLElement>("[data-content-scene]")?.dataset.contentScene;
    return normalizeContentScene(explicitScene);
}

function normalizeContentScene(scene?: string): ContentScene {
    const normalized = scene?.trim().toUpperCase();
    return normalized === "VIDEO" || normalized === "COMMENT" ? normalized : "ARTICLE";
}

function normalizeRouteId(routeId: string) {
    try {
        return decodeURIComponent(routeId).trim();
    } catch {
        return routeId.trim();
    }
}

function insertPendingComment(button: HTMLElement, comment: string) {
    const comments = button.closest("section")?.querySelector(".flex.flex-col.gap-3");
    if (!comments) {
        return null;
    }

    const item = document.createElement("div");
    item.className = "comment-row rounded-2xl p-4";
    item.dataset.commentState = "pending";
    const session = readAuthSession();

    const row = document.createElement("div");
    row.className = "flex gap-3";

    const avatar = document.createElement("div");
    avatar.className = "w-10 h-10 rounded-full border border-white bg-white/55 shadow-sm";

    const body = document.createElement("div");
    body.className = "flex-1 min-w-0";

    const author = document.createElement("div");
    author.className = "font-headline-md text-[15px] text-on-surface";
    author.textContent = session?.user?.nickname ?? "我";

    const meta = document.createElement("div");
    meta.className = "font-meta-xs text-on-surface-variant mt-1";
    meta.textContent = "刚刚 · 发送中";

    const content = document.createElement("p");
    content.className = "font-body-md text-on-surface-variant mt-3";
    content.textContent = comment;

    body.append(author, meta, content);
    row.append(avatar, body);
    item.append(row);
    comments.querySelector<HTMLElement>('[data-page-state="empty"]')?.setAttribute("hidden", "true");
    comments.prepend(item);
    return item;
}

function restoreCommentEmptyState(anchor: HTMLElement) {
    const comments = anchor.closest("section")?.querySelector(".flex.flex-col.gap-3");
    if (!comments) {
        return;
    }

    const hasCommentRows = Array.from(comments.children).some((item) => {
        return item instanceof HTMLElement && item.classList.contains("comment-row");
    });
    if (hasCommentRows) {
        return;
    }

    comments.querySelector<HTMLElement>('[data-page-state="empty"]')?.removeAttribute("hidden");
}

function updateCommentCounters(anchor: HTMLElement, delta: number) {
    const root = anchor.closest("main") ?? document;
    updateCommentCountersInRoot(root, delta);
}

function updateCommentCountersInRoot(root: ParentNode, delta: number) {
    const actionCount = root.querySelector<HTMLElement>("[data-content-comment-count]");
    updateNumericText(actionCount, delta);

    const discussionCount = root.querySelector<HTMLElement>("[data-comment-discussion-count]");
    if (!discussionCount) {
        return;
    }

    const currentCount = readLeadingNumber(discussionCount.textContent ?? "");
    if (currentCount === null) {
        return;
    }

    discussionCount.textContent = `${Math.max(0, currentCount + delta)} 条讨论`;
}

function createReplyComposer(authorName: string) {
    const composer = document.createElement("div");
    composer.dataset.replyComposer = "true";
    composer.className = "mt-4 rounded-2xl bg-white/35 border border-white/40 px-4 py-3";

    const shell = document.createElement("div");
    shell.className = "composer-shell";

    const input = document.createElement("input");
    input.className = "w-full bg-transparent border-none text-body-md focus:ring-0 placeholder:text-on-surface-variant/60 transition-all duration-300";
    input.placeholder = `回复 ${authorName}...`;
    input.type = "text";
    input.maxLength = maxCommentLength;

    const actions = document.createElement("div");
    actions.className = "flex items-center justify-end gap-2 mt-3";

    const cancel = document.createElement("button");
    cancel.className = "glass-button-ghost rounded-full px-4 py-2 text-primary font-label-sm active:scale-95";
    cancel.type = "button";
    cancel.textContent = "取消";
    cancel.addEventListener("click", () => composer.remove());

    const send = document.createElement("button");
    send.className = "glass-button-primary text-white font-label-sm px-4 py-2 rounded-full active:scale-95 transition-all duration-300";
    send.type = "button";
    send.textContent = "发送";

    shell.append(input);
    actions.append(cancel, send);
    composer.append(shell, actions);
    return composer;
}

function insertPendingReply(composer: HTMLElement, comment: string) {
    const item = document.createElement("div");
    item.dataset.commentState = "pending";
    item.className = "mt-3 rounded-2xl bg-white/35 border border-white/40 px-4 py-3 font-body-md text-on-surface-variant text-[13px]";
    item.textContent = comment;
    composer.append(item);
    return item;
}

function resolveCommentMetadata(row: HTMLElement) {
    const fallbackAuthorName = Array.from(row.querySelectorAll<HTMLElement>(".font-headline-md"))
        .map((item) => item.textContent?.trim() ?? "")
        .find(Boolean) ?? "";
    const authorName = row.dataset.commentAuthorName || fallbackAuthorName;
    const commentId = row.dataset.commentId ?? "";
    const rootId = row.dataset.rootId || commentId;
    const userId = row.dataset.commentUserId ?? "";

    if (!authorName || !commentId || !rootId || !userId) {
        return null;
    }

    return { authorName, commentId, rootId, userId };
}

function showInlineStatus(anchor: HTMLElement, message: string, type: "success" | "error") {
    const parent = anchor.parentElement;
    if (!parent) {
        return;
    }

    let status = parent.querySelector<HTMLElement>("[data-inline-status]");
    if (!status) {
        status = document.createElement("span");
        status.dataset.inlineStatus = "true";
        status.className = "ml-3 text-[12px] font-semibold";
        parent.append(status);
    }

    status.textContent = message;
    status.classList.toggle("text-primary", type === "success");
    status.classList.toggle("text-red-600", type === "error");
}

function showFormStatus(form: HTMLElement, message: string, type: "success" | "error") {
    let status = form.querySelector<HTMLElement>("[data-form-status]");
    if (!status) {
        status = document.createElement("div");
        status.dataset.formStatus = "true";
        status.className = "rounded-2xl border px-4 py-3 text-[13px] font-semibold";
        form.prepend(status);
    }

    status.textContent = message;
    status.classList.toggle("border-blue-200", type === "success");
    status.classList.toggle("bg-blue-50", type === "success");
    status.classList.toggle("text-primary", type === "success");
    status.classList.toggle("border-red-200", type === "error");
    status.classList.toggle("bg-red-50", type === "error");
    status.classList.toggle("text-red-700", type === "error");
}

function getNamedFieldValue(form: HTMLElement, name: string) {
    const field = form instanceof HTMLFormElement
        ? form.elements.namedItem(name)
        : form.querySelector(`[name="${name}"]`);
    if (field instanceof HTMLInputElement || field instanceof HTMLTextAreaElement) {
        return field.value.trim();
    }

    return "";
}

function validateProfileFields(fields: { nickname: string; bio: string; email: string }) {
    const errors: string[] = [];
    if (fields.nickname.length > 64) {
        errors.push("昵称最多 64 字");
    }

    if (fields.bio.length > 255) {
        errors.push("简介最多 255 字");
    }

    if (fields.email && !/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(fields.email)) {
        errors.push("请输入有效邮箱");
    }

    return errors;
}

function validatePublishFields(fields: { title: string; description: string; content: string }) {
    const errors: string[] = [];
    if (!fields.title) {
        errors.push("请输入标题");
    }

    if (!fields.content) {
        errors.push("请输入正文");
    }

    if (fields.title.length > 80) {
        errors.push("标题最多 80 字");
    }

    if (fields.description.length > 255) {
        errors.push("简介最多 255 字");
    }

    if (fields.content.length > 5000) {
        errors.push("正文最多 5000 字");
    }

    return errors;
}

function validateEditContentFields(fields: { title: string; description: string; content: string; contentType: string }) {
    const errors: string[] = [];
    if (!fields.title) {
        errors.push("请输入标题");
    }

    if (!fields.content) {
        errors.push(fields.contentType === "video" ? "请输入视频链接" : "请输入正文");
    }

    if (fields.title.length > 100) {
        errors.push("标题最多 100 字");
    }

    if (fields.description.length > (fields.contentType === "video" ? 500 : 255)) {
        errors.push(fields.contentType === "video" ? "摘要最多 500 字" : "摘要最多 255 字");
    }

    return errors;
}

function optionalValue(value: string) {
    return value || undefined;
}

function optionalNumber(value: string) {
    if (!value) {
        return undefined;
    }

    const numberValue = Number(value);
    return Number.isFinite(numberValue) ? numberValue : undefined;
}

function captureButtonState(button: HTMLElement): ButtonStateSnapshot {
    return {
        ariaLabel: button.getAttribute("aria-label"),
        className: button.className,
        text: button.textContent ?? "",
        valueText: findActionValue(button)?.textContent ?? null
    };
}

function restoreButtonState(button: HTMLElement, snapshot: ButtonStateSnapshot) {
    button.className = snapshot.className;
    if (snapshot.ariaLabel) {
        button.setAttribute("aria-label", snapshot.ariaLabel);
    } else {
        button.removeAttribute("aria-label");
    }

    if (isFollowButtonText(snapshot.text)) {
        button.textContent = snapshot.text;
        return;
    }

    const value = findActionValue(button);
    if (value && snapshot.valueText !== null) {
        value.textContent = snapshot.valueText;
    }
}

function restoreButtonStates(snapshots: Array<{ button: HTMLElement; snapshot: ButtonStateSnapshot }>) {
    snapshots.forEach(({ button, snapshot }) => restoreButtonState(button, snapshot));
}

function isFollowButtonText(text: string) {
    const normalizedText = text.trim().replace(/\s+/g, "");
    return normalizedText === "关注" || normalizedText === "关注作者" || normalizedText === "已关注";
}

function applyOptimisticStateToButtons(buttons: HTMLElement[], action: { kind: ContentReactionKind; nextActive: boolean }, nextCount: number) {
    buttons.forEach((button) => applyOptimisticState(button, action, nextCount));
}

function applyOptimisticState(button: HTMLElement, action: { kind: ContentReactionKind; nextActive: boolean }, nextCount: number) {
    if (action.kind === "like") {
        setClassToken(button, "text-error", action.nextActive);
        setClassToken(button, "text-on-surface-variant", !action.nextActive);
        button.setAttribute("aria-label", action.nextActive ? "取消点赞" : "点赞");
        setActionCount(button, nextCount);
        return;
    }

    setClassToken(button, "text-primary", action.nextActive);
    setClassToken(button, "text-on-surface-variant", !action.nextActive);
    button.setAttribute("aria-label", action.nextActive ? "取消收藏" : "收藏");
    setActionCount(button, nextCount);
}

function getContentActionFailureMessage(action: { kind: "like" | "favorite"; nextActive: boolean }) {
    if (action.kind === "like") {
        return action.nextActive ? "点赞失败，请重试" : "取消点赞失败，请重试";
    }

    return action.nextActive ? "收藏失败，请重试" : "取消收藏失败，请重试";
}

function setClassToken(element: HTMLElement, token: string, enabled: boolean) {
    if (enabled) {
        element.classList.add(token);
    } else {
        element.classList.remove(token);
    }
}

function isLikeActive(button: HTMLElement) {
    return button.classList.contains("text-error") || button.classList.contains("selected");
}

function isFavoriteActive(button: HTMLElement) {
    return button.classList.contains("text-primary") || button.classList.contains("bookmarked") || (button.textContent ?? "").includes("已保存");
}

function findActionValue(button: HTMLElement) {
    const spans = Array.from(button.querySelectorAll("span"));
    return spans.find((item) => !item.classList.contains("material-symbols-outlined")) ?? null;
}

function findMatchingContentActionButtons(contentId: string, kind: ContentReactionKind) {
    return Array.from(document.querySelectorAll<HTMLButtonElement>("button[data-content-id]"))
        .filter((button) => button.dataset.contentId === contentId && resolveContentAction(button)?.kind === kind);
}

function getActionCount(button: HTMLElement) {
    return readLeadingNumber(findActionValue(button)?.textContent ?? "") ?? 0;
}

function setActionCount(button: HTMLElement, count: number) {
    const value = findActionValue(button);
    if (value) {
        value.textContent = String(count);
    }
}

function updateNumericText(element: HTMLElement | null, delta: number) {
    if (!element) {
        return;
    }

    const currentCount = readLeadingNumber(element.textContent ?? "");
    if (currentCount === null) {
        return;
    }

    element.textContent = String(Math.max(0, currentCount + delta));
}

function readLeadingNumber(text: string) {
    const match = text.trim().match(/^\d+/);
    if (!match) {
        return null;
    }

    return Number(match[0]);
}

function navigateToLogin() {
    const next = `${window.location.pathname}${window.location.search}`;
    navigateTo(`/login?next=${encodeURIComponent(next)}`);
}

function isAuthError(error: unknown) {
    return typeof error === "object" && error !== null && "status" in error && (error.status === 401 || error.status === 403);
}

function ensureLegacyBrowserApis() {
    if (typeof window.matchMedia !== "function") {
        Object.defineProperty(window, "matchMedia", {
            writable: true,
            value: (query: string) => ({
                matches: false,
                media: query,
                onchange: null,
                addListener: () => undefined,
                removeListener: () => undefined,
                addEventListener: () => undefined,
                removeEventListener: () => undefined,
                dispatchEvent: () => false
            })
        });
    }
}
