export const sharedGlassStyles = `:root {
    color-scheme: light;
}

body {
    margin: 0;
    min-height: 100vh;
    color: #10171a;
    background-color: #eef2f6;
    background-image:
        radial-gradient(circle at 15% 50%, rgba(180, 197, 255, 0.4) 0%, transparent 50%),
        radial-gradient(circle at 85% 30%, rgba(219, 226, 250, 0.5) 0%, transparent 50%);
    background-attachment: fixed;
}

button,
input,
textarea {
    font: inherit;
}

a {
    color: inherit;
    text-decoration: none;
}

img {
    display: block;
    max-width: 100%;
}

.glass-panel {
    background: rgba(255, 255, 255, 0.4);
    backdrop-filter: blur(40px);
    -webkit-backdrop-filter: blur(40px);
    border: 1px solid rgba(255, 255, 255, 0.6);
    border-top: 1px solid rgba(255, 255, 255, 0.8);
    box-shadow: inset 0 1px 2px rgba(255, 255, 255, 0.8), 0 4px 20px rgba(0, 0, 0, 0.05);
}

.glass-input {
    background: rgba(255, 255, 255, 0.5);
    backdrop-filter: blur(20px);
    -webkit-backdrop-filter: blur(20px);
    border: 1px solid rgba(255, 255, 255, 0.7);
    box-shadow: inset 0 1px 3px rgba(0, 0, 0, 0.02);
    transition: all 0.3s ease-out;
}

.glass-input:focus {
    outline: none;
    border-color: #1f53c9;
    box-shadow: 0 0 0 2px rgba(31, 83, 201, 0.2), inset 0 1px 3px rgba(0, 0, 0, 0.02);
}

.glass-button-primary {
    background: linear-gradient(135deg, rgba(64, 109, 228, 0.9) 0%, rgba(31, 83, 201, 0.9) 100%);
    backdrop-filter: blur(10px);
    -webkit-backdrop-filter: blur(10px);
    box-shadow: 0 4px 15px rgba(31, 83, 201, 0.3), inset 0 1px 1px rgba(255, 255, 255, 0.4);
    border: 1px solid rgba(255, 255, 255, 0.2);
    transition: all 0.3s ease-out;
}

.glass-button-primary:hover {
    transform: translateY(-1px);
    box-shadow: 0 6px 20px rgba(31, 83, 201, 0.4), inset 0 1px 1px rgba(255, 255, 255, 0.5);
}

.glass-button-ghost {
    background: rgba(255, 255, 255, 0.3);
    backdrop-filter: blur(10px);
    -webkit-backdrop-filter: blur(10px);
    border: 1px solid rgba(255, 255, 255, 0.6);
    transition: all 0.3s ease-out;
}

.glass-button-ghost:hover {
    background: rgba(255, 255, 255, 0.5);
    border-color: rgba(255, 255, 255, 0.8);
}

.hover-lift {
    transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
}

.hover-lift:hover {
    transform: translateY(-4px);
    box-shadow: inset 0 1px 2px rgba(255, 255, 255, 0.9), 0 12px 30px rgba(0, 0, 0, 0.08);
    border-color: rgba(255, 255, 255, 0.9);
}

.shine-effect {
    position: relative;
    overflow: hidden;
}

.shine-effect::after {
    content: "";
    position: absolute;
    top: 0;
    left: -150%;
    width: 50%;
    height: 100%;
    background: linear-gradient(to right, rgba(255, 255, 255, 0) 0%, rgba(255, 255, 255, 0.4) 50%, rgba(255, 255, 255, 0) 100%);
    transform: skewX(-25deg);
    transition: left 0.7s ease;
    z-index: 10;
    pointer-events: none;
}

.shine-effect:hover::after {
    left: 200%;
}

.feed-transition {
    opacity: 0;
    transform: translateY(8px) scale(0.995);
    filter: blur(8px);
    transition:
        opacity 0.28s ease,
        transform 0.34s cubic-bezier(0.22, 1, 0.36, 1),
        filter 0.34s ease;
}

.feed-transition.feed-ready {
    opacity: 1;
    transform: none;
    filter: none;
}

.feed-transition.feed-leaving {
    opacity: 0;
    transform: translateY(6px) scale(0.996);
    filter: blur(7px);
    pointer-events: none;
}

.auth-field {
    background: rgba(255, 255, 255, 0.38);
    border: 1px solid rgba(255, 255, 255, 0.56);
    box-shadow: inset 0 1px 2px rgba(255, 255, 255, 0.72);
    transition: border-color 0.18s ease, background-color 0.18s ease, box-shadow 0.18s ease;
}

.auth-field:focus {
    outline: none;
    border-color: rgba(31, 83, 201, 0.42);
    background: rgba(255, 255, 255, 0.62);
    box-shadow: inset 0 1px 2px rgba(255, 255, 255, 0.86), 0 0 0 3px rgba(31, 83, 201, 0.1);
}

.auth-sheet {
    background: rgba(255, 255, 255, 0.52);
    border: 1px solid rgba(255, 255, 255, 0.72);
    box-shadow: inset 0 1px 2px rgba(255, 255, 255, 0.76), 0 24px 60px rgba(54, 83, 121, 0.12);
}

.auth-alert {
    background: rgba(255, 255, 255, 0.42);
    backdrop-filter: blur(18px);
    -webkit-backdrop-filter: blur(18px);
    border: 1px solid rgba(255, 255, 255, 0.58);
    box-shadow: inset 0 1px 2px rgba(255, 255, 255, 0.72), 0 10px 26px rgba(54, 83, 121, 0.08);
}

.auth-alert-error {
    color: #9f2a24;
    background:
        linear-gradient(135deg, rgba(255, 255, 255, 0.54), rgba(255, 245, 245, 0.62)),
        rgba(255, 255, 255, 0.38);
    border-color: rgba(200, 66, 58, 0.22);
    box-shadow: inset 0 1px 2px rgba(255, 255, 255, 0.78), 0 12px 30px rgba(200, 66, 58, 0.08);
}

.auth-gateway-root {
    position: relative;
    min-height: 100vh;
    overflow: hidden;
}

.auth-home-backdrop {
    position: absolute;
    inset: 0;
    filter: blur(22px) saturate(0.9);
    transform: none;
    opacity: 0.92;
    pointer-events: none;
    overflow: clip;
}

.auth-home-backdrop::before {
    content: "";
    position: absolute;
    inset: 0;
    background:
        radial-gradient(circle at 12% 20%, rgba(255, 255, 255, 0.7) 0, rgba(255, 255, 255, 0) 28%),
        radial-gradient(circle at 84% 18%, rgba(210, 222, 255, 0.65) 0, rgba(210, 222, 255, 0) 24%),
        radial-gradient(circle at 50% 72%, rgba(255, 255, 255, 0.42) 0, rgba(255, 255, 255, 0) 34%),
        linear-gradient(180deg, rgba(238, 242, 246, 0.72), rgba(238, 242, 246, 0.78));
}

.auth-home-shell {
    position: absolute;
    inset: 0;
    padding: 20px 24px 24px;
    overflow: hidden;
}

.auth-home-topbar {
    display: grid;
    grid-template-columns: 40px 90px minmax(140px, 1fr) 40px 40px;
    gap: 12px;
    align-items: center;
    max-width: 1440px;
    margin: 0 auto;
    height: 72px;
    padding: 0 8px;
    border-bottom: 1px solid rgba(255, 255, 255, 0.36);
}

.auth-logo-dot,
.auth-home-icon,
.auth-home-search {
    border-radius: 9999px;
    background: rgba(255, 255, 255, 0.52);
    box-shadow: inset 0 1px 2px rgba(255, 255, 255, 0.82);
}

.auth-logo-dot {
    width: 40px;
    height: 40px;
}

.auth-home-wordmark {
    color: #1f53c9;
    font-family: "Hanken Grotesk", "Inter", sans-serif;
    font-size: 22px;
    font-weight: 700;
}

.auth-home-search {
    height: 40px;
}

.auth-home-icon {
    width: 40px;
    height: 40px;
}

.auth-home-grid {
    display: grid;
    grid-template-columns: 248px minmax(0, 1fr) 336px;
    gap: 20px;
    max-width: 1440px;
    margin: 28px auto 0;
    height: calc(100% - 100px);
}

.auth-home-left,
.auth-home-right {
    display: flex;
    flex-direction: column;
    gap: 14px;
    padding-top: 12px;
}

.auth-home-feed {
    display: flex;
    flex-direction: column;
    gap: 16px;
    align-items: stretch;
}

.auth-backdrop-card {
    border-radius: 24px;
    padding: 18px;
    background: rgba(255, 255, 255, 0.42);
    border: 1px solid rgba(255, 255, 255, 0.58);
    box-shadow: inset 0 1px 2px rgba(255, 255, 255, 0.78), 0 10px 30px rgba(31, 83, 201, 0.07);
}

.font-title {
    font-family: "Hanken Grotesk", "Inter", sans-serif;
    font-weight: 650;
}

.auth-label {
    color: rgba(67, 70, 84, 0.78);
    font-family: "Inter", sans-serif;
    font-size: 12px;
    font-weight: 600;
}

@media (prefers-reduced-motion: reduce) {
    .feed-transition,
    .feed-transition.feed-ready,
    .feed-transition.feed-leaving {
        opacity: 1;
        transform: none;
        filter: none;
        transition: none;
    }
}
`;

export const sharedGlassBodyClass =
    "text-on-surface font-body-md antialiased overflow-x-hidden selection:bg-primary-container selection:text-on-primary-container";
