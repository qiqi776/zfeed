import {
  getContentUploadCredentials,
  type ContentUploadCredentialsRes,
  type ContentUploadScene,
} from "@/features/content/api/content.api";
import {
  contentImageUploadRule,
  contentVideoUploadRule,
  validateSelectedFile,
} from "@/shared/lib/media/fileValidation";

function readFileExt(fileName: string) {
  const index = fileName.lastIndexOf(".");
  if (index < 0 || index === fileName.length - 1) {
    return "";
  }
  return fileName.slice(index).toLowerCase();
}

export function inferUploadExt(file: File) {
  const ext = readFileExt(file.name);
  if (ext) {
    return ext;
  }
  return file.type === "image/jpeg" ? ".jpg" : "";
}

export async function postSignedUpload(
  file: File,
  credentials: ContentUploadCredentialsRes,
  fetchImpl: typeof fetch = fetch,
) {
  const formData = new FormData();
  formData.append("key", credentials.form_data.key);
  formData.append("policy", credentials.form_data.policy);
  formData.append("signature", credentials.form_data.signature);
  formData.append(
    "x-oss-signature-version",
    credentials.form_data["x-oss-signature-version"],
  );
  formData.append("x-oss-credential", credentials.form_data["x-oss-credential"]);
  formData.append("x-oss-date", credentials.form_data["x-oss-date"]);
  if (credentials.form_data["x-oss-security-token"]) {
    formData.append(
      "x-oss-security-token",
      credentials.form_data["x-oss-security-token"],
    );
  }
  formData.append("file", file);

  const response = await fetchImpl(credentials.form_data.host, {
    method: "POST",
    body: formData,
  });

  if (!response.ok) {
    throw new Error("文件上传失败");
  }
}

export async function uploadContentAsset(
  file: File,
  scene: ContentUploadScene,
  fetchImpl: typeof fetch = fetch,
) {
  validateSelectedFile(
    file,
    scene === "video-source" ? contentVideoUploadRule : contentImageUploadRule,
  );

  const fileExt = inferUploadExt(file);
  if (!fileExt) {
    throw new Error("无法识别文件类型");
  }

  const credentials = await getContentUploadCredentials({
    scene,
    file_ext: fileExt,
    file_name: file.name,
    file_size: file.size,
  });

  await postSignedUpload(file, credentials, fetchImpl);
  return {
    objectKey: credentials.object_key,
    url: credentials.url,
  };
}
