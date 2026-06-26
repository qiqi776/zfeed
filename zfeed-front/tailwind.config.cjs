const forms = require("@tailwindcss/forms");
const containerQueries = require("@tailwindcss/container-queries");

module.exports = {
    darkMode: "class",
    content: ["./index.html", "./src/**/*.{ts,tsx}"],
    theme: {
        extend: {
            colors: {
                secondary: "#575e72",
                error: "#ba1a1a",
                "surface-dim": "#d8dadc",
                "on-tertiary-fixed": "#1d192a",
                "surface-container-high": "#e6e8ea",
                "secondary-fixed": "#dbe2fa",
                "on-surface": "#191c1e",
                "surface-container": "#eceef0",
                primary: "#1f53c9",
                "inverse-surface": "#2d3133",
                "surface-tint": "#2355cc",
                "surface-container-highest": "#e0e3e5",
                "surface-container-lowest": "#ffffff",
                "inverse-on-surface": "#eff1f3",
                "on-secondary-container": "#5d6478",
                "surface-container-low": "#f2f4f6",
                "secondary-fixed-dim": "#bfc6dd",
                tertiary: "#5f596d",
                "surface-bright": "#f7f9fb",
                "on-primary-container": "#fefcff",
                "secondary-container": "#dbe2fa",
                "tertiary-fixed-dim": "#cbc3da",
                "on-error-container": "#93000a",
                "on-tertiary-fixed-variant": "#494457",
                "primary-container": "#406de4",
                "on-error": "#ffffff",
                "error-container": "#ffdad6",
                "primary-fixed-dim": "#b4c5ff",
                "outline-variant": "#c3c6d6",
                "on-surface-variant": "#434654",
                "on-primary-fixed": "#00174c",
                "on-primary-fixed-variant": "#003daa",
                "surface-variant": "#e0e3e5",
                "inverse-primary": "#b4c5ff",
                surface: "#f7f9fb",
                "primary-fixed": "#dbe1ff",
                "on-secondary": "#ffffff",
                "tertiary-container": "#787186",
                background: "#f7f9fb",
                "on-primary": "#ffffff",
                "on-secondary-fixed-variant": "#3f4759",
                "tertiary-fixed": "#e7dff6",
                "on-background": "#191c1e",
                "on-tertiary-container": "#fffbff",
                outline: "#747685",
                "on-secondary-fixed": "#141b2c",
                "on-tertiary": "#ffffff"
            },
            borderRadius: {
                DEFAULT: "0.25rem",
                lg: "0.5rem",
                xl: "0.75rem",
                "2xl": "1rem",
                "3xl": "1.5rem",
                full: "9999px"
            },
            spacing: {
                "section-margin": "48px",
                "container-padding": "32px",
                "element-gap": "16px",
                "glass-padding": "24px",
                "grid-gutter": "20px"
            },
            fontFamily: {
                "body-lg": ["Hanken Grotesk"],
                "headline-lg-mobile": ["Hanken Grotesk"],
                "headline-lg": ["Hanken Grotesk"],
                display: ["Hanken Grotesk"],
                "headline-md": ["Hanken Grotesk"],
                "body-md": ["Hanken Grotesk"],
                "meta-xs": ["Inter"],
                "label-sm": ["Inter"]
            },
            fontSize: {
                "body-lg": ["16px", { lineHeight: "1.6", fontWeight: "400" }],
                "headline-lg-mobile": ["24px", { lineHeight: "1.2", fontWeight: "600" }],
                "headline-lg": ["32px", { lineHeight: "1.2", letterSpacing: "-0.01em", fontWeight: "600" }],
                display: ["48px", { lineHeight: "1.1", letterSpacing: "-0.02em", fontWeight: "700" }],
                "headline-md": ["20px", { lineHeight: "1.4", fontWeight: "600" }],
                "body-md": ["14px", { lineHeight: "1.5", fontWeight: "400" }],
                "meta-xs": ["11px", { lineHeight: "1", letterSpacing: "0.01em", fontWeight: "400" }],
                "label-sm": ["12px", { lineHeight: "1", letterSpacing: "0.02em", fontWeight: "500" }]
            }
        }
    },
    plugins: [forms, containerQueries]
};
