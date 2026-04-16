type FileValidationRule = {
  label: string;
  maxBytes: number;
  mimeTypes: readonly string[];
  extensions: readonly string[];
};

const mb = 1024 * 1024;

export const avatarUploadRule: FileValidationRule = {
  label: "头像",
  maxBytes: 5 * mb,
  mimeTypes: ["image/png", "image/jpeg", "image/webp"],
  extensions: [".png", ".jpg", ".jpeg", ".webp"],
};

export const contentImageUploadRule: FileValidationRule = {
  label: "图片",
  maxBytes: 5 * mb,
  mimeTypes: ["image/png", "image/jpeg", "image/webp"],
  extensions: [".png", ".jpg", ".jpeg", ".webp"],
};

export const contentVideoUploadRule: FileValidationRule = {
  label: "视频",
  maxBytes: 200 * mb,
  mimeTypes: ["video/mp4", "video/quicktime", "video/webm"],
  extensions: [".mp4", ".mov", ".webm"],
};

function normalizeExt(fileName: string) {
  const dotIndex = fileName.lastIndexOf(".");
  if (dotIndex < 0 || dotIndex === fileName.length - 1) {
    return "";
  }

  return fileName.slice(dotIndex).toLowerCase();
}

function formatBytes(bytes: number) {
  if (bytes % mb === 0) {
    return `${bytes / mb} MB`;
  }

  return `${(bytes / mb).toFixed(1)} MB`;
}

export function describeFileValidationRule(rule: FileValidationRule) {
  return `${rule.extensions
    .map((item) => item.replace(".", "").toUpperCase())
    .join(" / ")}，最大 ${formatBytes(rule.maxBytes)}`;
}

export function validateSelectedFile(file: File, rule: FileValidationRule) {
  const ext = normalizeExt(file.name);
  const mimeValid = file.type ? rule.mimeTypes.includes(file.type) : false;
  const extValid = ext ? rule.extensions.includes(ext) : false;

  if (!mimeValid && !extValid) {
    throw new Error(`${rule.label}文件类型不支持，请选择 ${describeFileValidationRule(rule)} 的文件。`);
  }

  if (file.size <= 0) {
    throw new Error(`${rule.label}文件为空，请重新选择。`);
  }

  if (file.size > rule.maxBytes) {
    throw new Error(`${rule.label}文件过大，请选择不超过 ${formatBytes(rule.maxBytes)} 的文件。`);
  }
}
