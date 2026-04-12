import { useEffect, useState } from "react";

type ImageFallbackVariant = "avatar" | "cover";

type ImageFallbackProps = {
  src?: string | null;
  alt: string;
  name?: string;
  variant?: ImageFallbackVariant;
  containerClassName?: string;
  imageClassName?: string;
  fallbackClassName?: string;
  loading?: "eager" | "lazy";
};

const variantClassName: Record<ImageFallbackVariant, string> = {
  avatar:
    "bg-[radial-gradient(circle_at_top,#dff7f3,transparent_45%),linear-gradient(135deg,#eef7fb,#f8fbff)] text-slate-600",
  cover:
    "bg-[radial-gradient(circle_at_top_left,#dff7f3,transparent_32%),radial-gradient(circle_at_bottom_right,#ffe5dc,transparent_38%),linear-gradient(135deg,#f4f8fc,#edf4fa)] text-slate-500",
};

function resolveAvatarFallback(name?: string) {
  const trimmed = name?.trim();
  if (!trimmed) {
    return "Z";
  }
  return trimmed.slice(0, 1).toUpperCase();
}

export function ImageFallback({
  src,
  alt,
  name,
  variant = "cover",
  containerClassName = "",
  imageClassName = "",
  fallbackClassName = "",
  loading = "lazy",
}: ImageFallbackProps) {
  const [hasError, setHasError] = useState(false);

  useEffect(() => {
    setHasError(false);
  }, [src]);

  const showImage = Boolean(src) && !hasError;
  const fallbackLabel = variant === "avatar" ? resolveAvatarFallback(name || alt) : "暂无封面";

  return (
    <div className={containerClassName}>
      {showImage ? (
        <img
          src={src ?? undefined}
          alt={alt}
          className={imageClassName}
          loading={loading}
          onError={() => setHasError(true)}
        />
      ) : null}
      {!showImage ? (
        <div
          aria-label={`${alt} 占位图`}
          className={[
            "flex h-full w-full items-center justify-center",
            variantClassName[variant],
            variant === "avatar"
              ? "text-lg font-semibold tracking-[0.08em]"
              : "text-xs font-medium uppercase tracking-[0.22em]",
            fallbackClassName,
          ].join(" ")}
        >
          {fallbackLabel}
        </div>
      ) : null}
    </div>
  );
}
