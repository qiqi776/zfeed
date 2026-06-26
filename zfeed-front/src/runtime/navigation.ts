export function navigateTo(path: string) {
    if (window.location.pathname + window.location.search + window.location.hash === path) {
        return;
    }

    window.history.pushState({}, "", path);
    window.dispatchEvent(new PopStateEvent("popstate"));
}
