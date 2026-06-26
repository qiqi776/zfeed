import { useSyncExternalStore } from "react";
import { readAuthSession } from "./authStore";

export type ContentInteractionFields = {
    content_id: string | number;
    like_count?: number;
    favorite_count?: number;
    comment_count?: number;
    is_liked?: boolean;
    is_favorited?: boolean;
};

export type ContentReactionKind = "like" | "favorite";

type ContentInteractionOverride = {
    like_count?: number;
    favorite_count?: number;
    comment_count?: number;
    is_liked?: boolean;
    is_favorited?: boolean;
};

const overrides = new Map<string, ContentInteractionOverride>();
const listeners = new Set<() => void>();
const storageKeyPrefix = "zfeed.content-interaction-overrides";
const storedOverrideTtlMs = 7 * 24 * 60 * 60 * 1000;
const maxStoredOverrides = 200;
let version = 0;
let activeStorageKey = "";

export function useContentInteractionVersion() {
    return useSyncExternalStore(subscribeContentInteractions, getContentInteractionVersion, getContentInteractionVersion);
}

export function mergeContentInteractionState<T extends ContentInteractionFields>(item: T): T {
    ensureStoredOverridesLoaded();
    const contentId = normalizeContentId(item.content_id);
    const override = contentId ? overrides.get(contentId) : undefined;
    if (!override) {
        return item;
    }

    return {
        ...item,
        ...definedValues(override)
    };
}

export function setContentReactionOverride(contentId: string, kind: ContentReactionKind, nextActive: boolean, nextCount: number) {
    ensureStoredOverridesLoaded();
    const normalizedId = normalizeContentId(contentId);
    const previous = cloneOverride(overrides.get(normalizedId));
    const next = {
        ...previous,
        ...(kind === "like"
            ? { is_liked: nextActive, like_count: nextCount }
            : { is_favorited: nextActive, favorite_count: nextCount })
    };

    overrides.set(normalizedId, next);
    persistStoredOverrides();
    notifyContentInteractionSubscribers();
    return previous;
}

export function restoreContentInteractionOverride(contentId: string, previous: ContentInteractionOverride | undefined) {
    ensureStoredOverridesLoaded();
    const normalizedId = normalizeContentId(contentId);
    if (previous) {
        overrides.set(normalizedId, previous);
    } else {
        overrides.delete(normalizedId);
    }

    persistStoredOverrides();
    notifyContentInteractionSubscribers();
}

export function resetContentInteractionState() {
    ensureStoredOverridesLoaded();
    overrides.clear();
    clearStoredOverrides();
    notifyContentInteractionSubscribers();
}

function subscribeContentInteractions(listener: () => void) {
    listeners.add(listener);
    return () => {
        listeners.delete(listener);
    };
}

function getContentInteractionVersion() {
    return version;
}

function notifyContentInteractionSubscribers() {
    version += 1;
    listeners.forEach((listener) => listener());
}

function cloneOverride(override: ContentInteractionOverride | undefined) {
    return override ? { ...override } : undefined;
}

function definedValues(override: ContentInteractionOverride) {
    return Object.fromEntries(Object.entries(override).filter(([, value]) => value !== undefined));
}

function normalizeContentId(contentId: string | number) {
    return String(contentId).trim();
}

function ensureStoredOverridesLoaded() {
    const storageKey = getStorageKey();
    if (storageKey === activeStorageKey) {
        return;
    }

    activeStorageKey = storageKey;
    overrides.clear();
    loadStoredOverrides(storageKey);
}

function getStorageKey() {
    const userId = readAuthSession()?.user?.userId;
    return `${storageKeyPrefix}:${userId ?? "guest"}`;
}

function loadStoredOverrides(storageKey: string) {
    const raw = safeGetStorageItem(storageKey);
    if (!raw) {
        return;
    }

    try {
        const parsed = JSON.parse(raw) as { entries?: unknown };
        if (!Array.isArray(parsed.entries)) {
            return;
        }

        const now = Date.now();
        parsed.entries.forEach((entry) => {
            if (!Array.isArray(entry) || entry.length !== 2 || typeof entry[0] !== "string" || !isRecord(entry[1])) {
                return;
            }

            const contentId = normalizeContentId(entry[0]);
            const override = parseStoredOverride(entry[1], now);
            if (contentId && override) {
                overrides.set(contentId, override);
            }
        });
    } catch {
        safeRemoveStorageItem(storageKey);
    }
}

function parseStoredOverride(rawOverride: Record<string, unknown>, now: number): ContentInteractionOverride | null {
    const updatedAt = typeof rawOverride.updatedAt === "number" ? rawOverride.updatedAt : 0;
    if (!Number.isFinite(updatedAt) || now - updatedAt > storedOverrideTtlMs) {
        return null;
    }

    const override: ContentInteractionOverride = {};
    copyNumberField(rawOverride, override, "like_count");
    copyNumberField(rawOverride, override, "favorite_count");
    copyNumberField(rawOverride, override, "comment_count");
    copyBooleanField(rawOverride, override, "is_liked");
    copyBooleanField(rawOverride, override, "is_favorited");

    return Object.keys(override).length > 0 ? override : null;
}

function copyNumberField(
    source: Record<string, unknown>,
    target: ContentInteractionOverride,
    field: "like_count" | "favorite_count" | "comment_count"
) {
    const value = source[field];
    if (typeof value === "number" && Number.isFinite(value) && value >= 0) {
        target[field] = value;
    }
}

function copyBooleanField(
    source: Record<string, unknown>,
    target: ContentInteractionOverride,
    field: "is_liked" | "is_favorited"
) {
    const value = source[field];
    if (typeof value === "boolean") {
        target[field] = value;
    }
}

function persistStoredOverrides() {
    const storageKey = activeStorageKey || getStorageKey();
    const now = Date.now();
    const entries = Array.from(overrides.entries())
        .slice(-maxStoredOverrides)
        .map(([contentId, override]) => [contentId, { ...definedValues(override), updatedAt: now }]);

    safeSetStorageItem(storageKey, JSON.stringify({ entries }));
}

function clearStoredOverrides() {
    if (!hasLocalStorage()) {
        return;
    }

    for (let index = window.localStorage.length - 1; index >= 0; index -= 1) {
        const key = window.localStorage.key(index);
        if (key?.startsWith(storageKeyPrefix)) {
            safeRemoveStorageItem(key);
        }
    }
}

function isRecord(value: unknown): value is Record<string, unknown> {
    return Boolean(value && typeof value === "object" && !Array.isArray(value));
}

function safeGetStorageItem(key: string) {
    if (!hasLocalStorage()) {
        return null;
    }

    try {
        return window.localStorage.getItem(key);
    } catch {
        return null;
    }
}

function safeSetStorageItem(key: string, value: string) {
    if (!hasLocalStorage()) {
        return;
    }

    try {
        window.localStorage.setItem(key, value);
    } catch {
        // Storage can be unavailable in private browsing or quota-exceeded states.
    }
}

function safeRemoveStorageItem(key: string) {
    if (!hasLocalStorage()) {
        return;
    }

    try {
        window.localStorage.removeItem(key);
    } catch {
        // Ignore storage cleanup failures; in-memory state is still authoritative.
    }
}

function hasLocalStorage() {
    try {
        return typeof window !== "undefined" && Boolean(window.localStorage);
    } catch {
        return false;
    }
}
