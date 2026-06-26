import type { ContentScene } from "../runtime/apiClient";

export function getContentScene(contentType?: number | string): ContentScene {
    const normalizedType = typeof contentType === "string" ? Number(contentType) : contentType;
    return normalizedType === 2 || normalizedType === 20 ? "VIDEO" : "ARTICLE";
}
