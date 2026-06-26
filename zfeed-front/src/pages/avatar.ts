import { createElement } from "react";
import type { ReactNode } from "react";

type AvatarSubject = {
    avatar?: unknown;
    nickname?: unknown;
    userId?: unknown;
};

type AvatarOptions = {
    alt?: string;
    textClassName?: string;
};

export function renderUserAvatar(subject: AvatarSubject, className: string, options: AvatarOptions = {}): ReactNode {
    const avatar = cleanAvatarValue(subject.avatar);
    const nickname = cleanAvatarValue(subject.nickname);
    const userId = cleanAvatarValue(subject.userId);
    const alt = options.alt ?? `${nickname || "用户"}头像`;

    if (avatar && isImageAvatar(avatar)) {
        return createElement("img", {
            alt,
            className,
            src: avatar
        });
    }

    return createElement("div", {
        "aria-label": alt,
        className: `${className} flex items-center justify-center bg-primary text-white font-display ${options.textClassName ?? "text-[16px]"}`
    }, getAvatarLabel(avatar, nickname, userId));
}

export function cleanAvatarValue(value: unknown) {
    if (typeof value === "string" || typeof value === "number") {
        return String(value).trim();
    }

    return "";
}

function isImageAvatar(value: string) {
    return /^(https?:)?\/\//i.test(value) ||
        value.startsWith("/") ||
        value.startsWith("data:image/") ||
        value.startsWith("blob:");
}

function getAvatarLabel(avatar: string, nickname: string, userId: string) {
    if (avatar) {
        return avatar;
    }

    if (nickname) {
        return Array.from(nickname)[0]?.toUpperCase() ?? "Z";
    }

    return userId || "Z";
}
