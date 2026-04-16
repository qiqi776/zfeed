import { useCallback, useEffect, useRef } from "react";
import { useBlocker } from "react-router-dom";

type DraftEnvelope<T> = {
  value: T;
};

type DraftBootstrap<T> = {
  restored: boolean;
  value: T;
};

function canUseStorage() {
  return typeof window !== "undefined" && typeof window.localStorage !== "undefined";
}

export function readPublishDraft<T>(storageKey: string, fallback: T): DraftBootstrap<T> {
  if (!canUseStorage()) {
    return { restored: false, value: fallback };
  }

  try {
    const raw = window.localStorage.getItem(storageKey);
    if (!raw) {
      return { restored: false, value: fallback };
    }

    const parsed = JSON.parse(raw) as DraftEnvelope<T>;
    if (!parsed || typeof parsed !== "object" || !("value" in parsed)) {
      return { restored: false, value: fallback };
    }

    return { restored: true, value: parsed.value };
  } catch {
    return { restored: false, value: fallback };
  }
}

export function savePublishDraft<T>(storageKey: string, value: T) {
  if (!canUseStorage()) {
    return;
  }

  window.localStorage.setItem(storageKey, JSON.stringify({ value } satisfies DraftEnvelope<T>));
}

export function clearPublishDraft(storageKey: string) {
  if (!canUseStorage()) {
    return;
  }

  window.localStorage.removeItem(storageKey);
}

export function isValidPublicUrl(value: string) {
  try {
    const url = new URL(value);
    return url.protocol === "http:" || url.protocol === "https:";
  } catch {
    return false;
  }
}

export function useBeforeUnloadGuard(enabled: boolean) {
  useEffect(() => {
    if (!enabled || typeof window === "undefined") {
      return;
    }

    function handleBeforeUnload(event: BeforeUnloadEvent) {
      event.preventDefault();
      event.returnValue = "";
    }

    window.addEventListener("beforeunload", handleBeforeUnload);

    return () => {
      window.removeEventListener("beforeunload", handleBeforeUnload);
    };
  }, [enabled]);
}

export function useUnsavedChangesGuard(
  enabled: boolean,
  message = "你有未保存的修改，确认现在离开当前页面吗？",
) {
  const bypassRef = useRef(false);
  const blocker = useBlocker(() => enabled && !bypassRef.current);

  useBeforeUnloadGuard(enabled && !bypassRef.current);

  useEffect(() => {
    if (blocker.state !== "blocked") {
      return;
    }

    const shouldLeave = typeof window === "undefined" ? true : window.confirm(message);
    if (shouldLeave) {
      blocker.proceed();
      return;
    }

    blocker.reset();
  }, [blocker, message]);

  return useCallback(<T,>(callback: () => T) => {
    bypassRef.current = true;

    try {
      return callback();
    } finally {
      window.setTimeout(() => {
        bypassRef.current = false;
      }, 0);
    }
  }, []);
}
