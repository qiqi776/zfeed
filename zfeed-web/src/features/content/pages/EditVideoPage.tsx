import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { type ChangeEvent, type FormEvent, useEffect, useRef, useState } from "react";
import { Link, useNavigate, useParams } from "react-router-dom";

import { useSessionStore } from "@/entities/session/model/session.store";
import {
  editVideo,
  getContentDetail,
} from "@/features/content/api/content.api";
import { uploadContentAsset } from "@/features/content/lib/upload";
import { useUnsavedChangesGuard } from "@/features/publish/lib/publishDraft";
import { isValidHttpUrl } from "@/shared/lib/form/valueValidation";
import {
  contentImageUploadRule,
  contentVideoUploadRule,
  describeFileValidationRule,
} from "@/shared/lib/media/fileValidation";
import { contentKeys, feedKeys, userKeys } from "@/shared/lib/query/queryKeys";
import { ImageFallback } from "@/shared/ui/ImageFallback";
import { InlineNotice } from "@/shared/ui/InlineNotice";
import { PageHeader } from "@/shared/ui/PageHeader";
import { StatePanel } from "@/shared/ui/StatePanel";
import { useToast } from "@/shared/ui/toast/toast.store";

type VideoFormState = {
  title: string;
  description: string;
  coverUrl: string;
  videoUrl: string;
  duration: string;
};

function buildInitialForm(data: Awaited<ReturnType<typeof getContentDetail>>["detail"]): VideoFormState {
  return {
    title: data.title || "",
    description: data.description || "",
    coverUrl: data.cover_url || "",
    videoUrl: data.video_url || "",
    duration: data.video_duration ? String(data.video_duration) : "",
  };
}

function hasVideoFormChanges(current: VideoFormState, initial: VideoFormState | null) {
  if (!initial) {
    return false;
  }

  return (
    current.title.trim() !== initial.title.trim() ||
    current.description.trim() !== initial.description.trim() ||
    current.coverUrl.trim() !== initial.coverUrl.trim() ||
    current.videoUrl.trim() !== initial.videoUrl.trim() ||
    current.duration.trim() !== initial.duration.trim()
  );
}

export function EditVideoPage() {
  const params = useParams();
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const currentUserId = useSessionStore((state) => state.user?.userId ?? 0);
  const { showToast } = useToast();
  const coverFileInputRef = useRef<HTMLInputElement | null>(null);
  const videoFileInputRef = useRef<HTMLInputElement | null>(null);

  const contentId = Number(params.contentId);
  const isValidContentId = Number.isInteger(contentId) && contentId > 0;

  const query = useQuery({
    queryKey: contentKeys.detail(contentId, currentUserId),
    enabled: isValidContentId,
    queryFn: () => getContentDetail({ content_id: contentId }),
  });

  const [form, setForm] = useState<VideoFormState>({
    title: "",
    description: "",
    coverUrl: "",
    videoUrl: "",
    duration: "",
  });
  const [initialForm, setInitialForm] = useState<VideoFormState | null>(null);
  const hasUnsavedChanges = hasVideoFormChanges(form, initialForm);
  const trimmedTitle = form.title.trim();
  const trimmedDescription = form.description.trim();
  const trimmedCoverUrl = form.coverUrl.trim();
  const trimmedVideoUrl = form.videoUrl.trim();
  const trimmedDuration = form.duration.trim();
  const hasValidCoverUrl = !trimmedCoverUrl || isValidHttpUrl(trimmedCoverUrl);
  const hasValidVideoUrl = !trimmedVideoUrl || isValidHttpUrl(trimmedVideoUrl);

  useEffect(() => {
    if (!query.data?.detail) {
      return;
    }
    const next = buildInitialForm(query.data.detail);
    setForm(next);
    setInitialForm(next);
  }, [query.data]);

  const coverUploadMutation = useMutation({
    mutationFn: (file: File) => uploadContentAsset(file, "video-cover"),
    onSuccess: (res) => {
      setForm((current) => ({ ...current, coverUrl: res.url }));
      showToast({
        tone: "success",
        title: "视频封面已上传",
        description: "新的封面地址已经回填。",
      });
    },
    onError: (error: Error) => {
      showToast({
        tone: "error",
        title: "封面上传失败",
        description: error.message || "请稍后重试。",
      });
    },
  });

  const videoUploadMutation = useMutation({
    mutationFn: (file: File) => uploadContentAsset(file, "video-source"),
    onSuccess: (res) => {
      setForm((current) => ({ ...current, videoUrl: res.url }));
      showToast({
        tone: "success",
        title: "视频文件已上传",
        description: "新的视频地址已经回填。",
      });
    },
    onError: (error: Error) => {
      showToast({
        tone: "error",
        title: "视频上传失败",
        description: error.message || "请稍后重试。",
      });
    },
  });

  const mutation = useMutation({
    mutationFn: (payload: Parameters<typeof editVideo>[1]) => editVideo(contentId, payload),
    onSuccess: async () => {
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: contentKeys.detail(contentId, currentUserId) }),
        queryClient.invalidateQueries({ queryKey: feedKeys.userPublishPrefix(currentUserId) }),
        queryClient.invalidateQueries({ queryKey: feedKeys.studioPublishPrefix(currentUserId) }),
        queryClient.invalidateQueries({ queryKey: userKeys.profilePrefix(currentUserId) }),
      ]);
      showToast({
        tone: "success",
        title: "视频已更新",
        description: "已同步刷新详情页和发布列表。",
      });
      allowExit(() => navigate(`/content/${contentId}`, { replace: true }));
    },
    onError: (error: Error) => {
      showToast({
        tone: "error",
        title: "视频更新失败",
        description: error.message || "请稍后重试。",
      });
    },
  });

  const allowExit = useUnsavedChangesGuard(hasUnsavedChanges && !mutation.isPending);

  function updateField<Key extends keyof VideoFormState>(key: Key, value: VideoFormState[Key]) {
    setForm((current) => ({ ...current, [key]: value }));
  }

  function handleFileChange(
    event: ChangeEvent<HTMLInputElement>,
    mutate: (file: File) => void,
  ) {
    const file = event.target.files?.[0];
    event.target.value = "";
    if (!file) {
      return;
    }
    mutate(file);
  }

  function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!initialForm) {
      return;
    }

    const payload: Parameters<typeof editVideo>[1] = {};
    if (!trimmedTitle) {
      showToast({
        tone: "error",
        title: "标题不能为空",
        description: "请至少保留一个明确标题，再保存视频。",
      });
      return;
    }

    if (!trimmedCoverUrl || !hasValidCoverUrl) {
      showToast({
        tone: "error",
        title: "封面 URL 无效",
        description: "请输入可访问的 http / https 封面地址。",
      });
      return;
    }

    if (!trimmedVideoUrl || !hasValidVideoUrl) {
      showToast({
        tone: "error",
        title: "视频 URL 无效",
        description: "请输入可访问的 http / https 视频地址。",
      });
      return;
    }

    if (trimmedTitle !== initialForm.title.trim()) {
      payload.title = trimmedTitle;
    }
    if (trimmedDescription !== initialForm.description.trim()) {
      payload.description = trimmedDescription;
    }
    if (trimmedCoverUrl !== initialForm.coverUrl.trim()) {
      payload.cover_url = trimmedCoverUrl;
    }
    if (trimmedVideoUrl !== initialForm.videoUrl.trim()) {
      payload.video_url = trimmedVideoUrl;
    }
    if (trimmedDuration !== initialForm.duration.trim()) {
      const parsedDuration = Number.parseInt(trimmedDuration, 10);
      if (!Number.isFinite(parsedDuration) || parsedDuration <= 0) {
        showToast({
          tone: "error",
          title: "视频时长无效",
          description: "请输入有效秒数。",
        });
        return;
      }
      payload.duration = parsedDuration;
    }

    if (Object.keys(payload).length === 0) {
      showToast({
        tone: "info",
        title: "没有改动",
        description: "当前内容和已保存版本一致。",
      });
      return;
    }
    mutation.mutate(payload);
  }

  if (!isValidContentId) {
    return <StatePanel title="内容 ID 无效" description="请检查当前链接。" tone="error" />;
  }

  if (query.isLoading || !initialForm) {
    return (
      <section className="space-y-6">
        <div className="h-12 w-44 rounded-full bg-white shadow-card" />
        <div className="h-80 rounded-[32px] bg-white shadow-card" />
      </section>
    );
  }

  const detail = query.data?.detail;
  if (
    query.isError ||
    !detail ||
    detail.content_type !== 20 ||
    detail.author_id !== currentUserId
  ) {
    return (
      <StatePanel
        title="当前视频不可编辑"
        description="请确认这是一条属于你自己的视频。"
        tone="error"
      />
    );
  }

  return (
    <section className="space-y-6">
      <PageHeader
        eyebrow="Edit Video"
        title="编辑视频"
        description="修改标题、简介、封面、视频源文件地址和时长。"
        aside={
          <Link
            to={`/content/${contentId}`}
            className="inline-flex rounded-full border border-slate-200 bg-white px-4 py-2 text-sm text-slate-600 transition hover:border-accent hover:text-accent"
          >
            返回详情
          </Link>
        }
      />

      <div className="grid gap-6 xl:grid-cols-[1.05fr_0.95fr]">
        <form className="space-y-5 rounded-[32px] border border-slate-200 bg-white p-6 shadow-card" onSubmit={handleSubmit}>
          <InlineNotice
            title="视频源文件与封面都支持直传"
            description="如果后端 OSS 配置可用，可以直接上传；否则也可以手动填写 URL 继续编辑。"
            tone="soft"
          />

          <label className="block text-sm text-slate-700">
            标题
            <input
              value={form.title}
              onChange={(event) => updateField("title", event.target.value)}
              maxLength={100}
              className="mt-1 w-full rounded-2xl border border-slate-200 px-4 py-3 outline-none ring-accent transition focus:ring"
            />
            <span className="mt-2 block text-xs text-slate-400">{trimmedTitle.length}/100</span>
          </label>

          <label className="block text-sm text-slate-700">
            描述
            <textarea
              value={form.description}
              onChange={(event) => updateField("description", event.target.value)}
              rows={4}
              maxLength={500}
              className="mt-1 w-full resize-none rounded-[24px] border border-slate-200 px-4 py-3 outline-none ring-accent transition focus:ring"
            />
            <span className="mt-2 block text-xs text-slate-400">{trimmedDescription.length}/500</span>
          </label>

          <label className="block text-sm text-slate-700">
            封面 URL
            <input
              value={form.coverUrl}
              onChange={(event) => updateField("coverUrl", event.target.value)}
              className="mt-1 w-full rounded-2xl border border-slate-200 px-4 py-3 outline-none ring-accent transition focus:ring"
            />
            <span
              className={[
                "mt-2 block text-xs",
                trimmedCoverUrl && !hasValidCoverUrl ? "text-ember" : "text-slate-400",
              ].join(" ")}
            >
              {trimmedCoverUrl && !hasValidCoverUrl
                ? "请输入可访问的 http / https 封面地址。"
                : "封面会继续出现在推荐流、详情首屏和公开视频列表中。"}
            </span>
          </label>

          <div className="flex flex-wrap gap-3">
            <button
              type="button"
              onClick={() => coverFileInputRef.current?.click()}
              aria-describedby="edit-video-cover-upload-hint"
              className="inline-flex rounded-full border border-slate-200 bg-white px-4 py-2 text-sm text-slate-600 transition hover:border-accent hover:text-accent"
            >
              {coverUploadMutation.isPending ? "上传中..." : "上传视频封面"}
            </button>
            <input
              ref={coverFileInputRef}
              type="file"
              accept="image/png,image/jpeg,image/webp"
              tabIndex={-1}
              className="hidden"
              onChange={(event) => handleFileChange(event, coverUploadMutation.mutate)}
            />
            <button
              type="button"
              onClick={() => updateField("coverUrl", "")}
              disabled={!form.coverUrl.trim()}
              className="rounded-full border border-slate-200 bg-white px-4 py-2 text-sm text-slate-600 transition hover:border-accent hover:text-accent disabled:opacity-60"
            >
              清空封面
            </button>
          </div>
          <p id="edit-video-cover-upload-hint" className="text-xs text-slate-500">
            封面文件支持 {describeFileValidationRule(contentImageUploadRule)}。
          </p>

          <label className="block text-sm text-slate-700">
            视频 URL
            <input
              value={form.videoUrl}
              onChange={(event) => updateField("videoUrl", event.target.value)}
              className="mt-1 w-full rounded-2xl border border-slate-200 px-4 py-3 outline-none ring-accent transition focus:ring"
            />
            <span
              className={[
                "mt-2 block text-xs",
                trimmedVideoUrl && !hasValidVideoUrl ? "text-ember" : "text-slate-400",
              ].join(" ")}
            >
              {trimmedVideoUrl && !hasValidVideoUrl
                ? "请输入可访问的 http / https 视频地址。"
                : "如果地址可访问，右侧会优先显示视频预览。"}
            </span>
          </label>

          <div className="flex flex-wrap gap-3">
            <button
              type="button"
              onClick={() => videoFileInputRef.current?.click()}
              aria-describedby="edit-video-source-upload-hint"
              className="inline-flex rounded-full border border-slate-200 bg-white px-4 py-2 text-sm text-slate-600 transition hover:border-accent hover:text-accent"
            >
              {videoUploadMutation.isPending ? "上传中..." : "上传视频文件"}
            </button>
            <input
              ref={videoFileInputRef}
              type="file"
              accept="video/mp4,video/quicktime,video/webm"
              tabIndex={-1}
              className="hidden"
              onChange={(event) => handleFileChange(event, videoUploadMutation.mutate)}
            />
            <button
              type="button"
              onClick={() => updateField("videoUrl", "")}
              disabled={!form.videoUrl.trim()}
              className="rounded-full border border-slate-200 bg-white px-4 py-2 text-sm text-slate-600 transition hover:border-accent hover:text-accent disabled:opacity-60"
            >
              清空视频地址
            </button>
          </div>
          <p id="edit-video-source-upload-hint" className="text-xs text-slate-500">
            视频文件支持 {describeFileValidationRule(contentVideoUploadRule)}。
          </p>

          <label className="block text-sm text-slate-700">
            时长（秒）
            <input
              value={form.duration}
              onChange={(event) => updateField("duration", event.target.value)}
              inputMode="numeric"
              className="mt-1 w-full rounded-2xl border border-slate-200 px-4 py-3 outline-none ring-accent transition focus:ring"
            />
            <span className="mt-2 block text-xs text-slate-400">
              {trimmedDuration ? "如果变更时长，必须填写大于 0 的整数秒数。" : "保持原值则留空不动即可。"}
            </span>
          </label>

          <button
            type="submit"
            disabled={mutation.isPending}
            className="rounded-full bg-ink px-5 py-2.5 text-sm font-medium text-white transition hover:bg-slate-800 disabled:opacity-60"
          >
            {mutation.isPending ? "保存中..." : "保存视频"}
          </button>
        </form>

        <div className="space-y-6">
          <InlineNotice
            title={hasUnsavedChanges ? "离开当前页面前会提醒你" : "当前没有未保存修改"}
            description={
              hasUnsavedChanges
                ? "如果你刷新、关闭标签页或切换到站内其他页面，都会先弹出确认。"
                : "保存成功后会自动返回内容详情页。"
            }
            tone={hasUnsavedChanges ? "soft" : "neutral"}
          />

          <section className="overflow-hidden rounded-[32px] border border-slate-200 bg-white shadow-card">
          {hasValidVideoUrl && form.videoUrl ? (
            <video
              controls
              playsInline
              poster={hasValidCoverUrl ? form.coverUrl : undefined}
              className="aspect-[16/10] w-full bg-black object-cover"
              src={form.videoUrl}
            />
          ) : (
            <ImageFallback
              src={hasValidCoverUrl ? form.coverUrl : undefined}
              alt={form.title || "视频封面"}
              containerClassName="aspect-[16/10] bg-slate-100"
              imageClassName="h-full w-full object-cover"
            />
          )}
          <div className="space-y-4 p-5">
            <p className="text-xs uppercase tracking-[0.24em] text-slate-400">Preview</p>
            <h2 className="font-display text-3xl font-semibold text-slate-900">
              {form.title.trim() || "视频标题会显示在这里"}
            </h2>
            <p className="text-sm leading-7 text-slate-600">
              {form.description.trim() || "简介会帮助浏览者快速判断这段视频的主题。"}
            </p>
            <p className="text-sm text-slate-500">
              时长：{trimmedDuration ? `${trimmedDuration} 秒` : "待填写"}
            </p>
          </div>
          </section>
        </div>
      </div>
    </section>
  );
}
