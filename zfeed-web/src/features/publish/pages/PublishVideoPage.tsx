import { type FormEvent, useEffect, useState } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { Link, useNavigate } from "react-router-dom";

import { useSessionStore } from "@/entities/session/model/session.store";
import { publishVideo } from "@/features/content/api/content.api";
import {
  clearPublishDraft,
  isValidPublicUrl,
  readPublishDraft,
  savePublishDraft,
  useBeforeUnloadGuard,
} from "@/features/publish/lib/publishDraft";
import { invalidatePublishSurfaces } from "@/shared/lib/query/cacheSync";
import { userKeys } from "@/shared/lib/query/queryKeys";
import { ImageFallback } from "@/shared/ui/ImageFallback";
import { InlineNotice } from "@/shared/ui/InlineNotice";
import { PageHeader } from "@/shared/ui/PageHeader";
import { useToast } from "@/shared/ui/toast/toast.store";

const defaultVisibility = "10";
const VIDEO_DRAFT_STORAGE_KEY = "zfeed-web-publish-video-draft";

type VideoFormState = {
  title: string;
  description: string;
  coverUrl: string;
  videoUrl: string;
  duration: string;
  visibility: string;
};

function createEmptyVideoForm(): VideoFormState {
  return {
    title: "",
    description: "",
    coverUrl: "",
    videoUrl: "",
    duration: "",
    visibility: defaultVisibility,
  };
}

function VideoPreviewCard({
  title,
  description,
  coverUrl,
  videoUrl,
  duration,
}: {
  title: string;
  description: string;
  coverUrl?: string;
  videoUrl?: string;
  duration: string;
}) {
  return (
    <section className="overflow-hidden rounded-[32px] border border-slate-200 bg-white shadow-card">
      {videoUrl ? (
        <video
          controls
          playsInline
          poster={coverUrl}
          className="aspect-[16/10] w-full bg-black object-cover"
          src={videoUrl}
        />
      ) : (
        <ImageFallback
          src={coverUrl}
          alt={title || "视频封面"}
          containerClassName="aspect-[16/10] bg-slate-100"
          imageClassName="h-full w-full object-cover"
        />
      )}

      <div className="space-y-4 p-5">
        <div className="flex items-center justify-between gap-3">
          <span className="rounded-full bg-[#eef7fb] px-3 py-1 text-xs font-medium text-slate-600">
            公开视频
          </span>
          <span className="text-xs text-slate-400">
            {duration.trim() ? `${duration.trim()} 秒` : "时长待填写"}
          </span>
        </div>

        <div>
          <h2 className="line-clamp-2 font-display text-2xl font-semibold text-slate-900">
            {title || "你的标题会显示在这里"}
          </h2>
          <p className="mt-2 text-sm leading-7 text-slate-600">
            {description || "补一段简短描述，让浏览者更快判断这段视频要讲什么。"}
          </p>
        </div>

        <div className="rounded-[24px] bg-slate-50 p-4">
          <p className="text-xs uppercase tracking-[0.22em] text-slate-500">Preview Notes</p>
          <p className="mt-3 text-sm leading-7 text-slate-700">
            {videoUrl
              ? "如果地址可访问，这里会直接预览视频播放效果。你可以先检查封面、标题和时长是否匹配。"
              : "填写视频 URL 后，这里会优先显示可播放预览；未填写时，则先用封面预览来检查作品气质。"}
          </p>
        </div>
      </div>
    </section>
  );
}

export function PublishVideoPage() {
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const currentUserId = useSessionStore((state) => state.user?.userId ?? 0);
  const { showToast } = useToast();

  const [draftBootstrap] = useState(() =>
    readPublishDraft<VideoFormState>(VIDEO_DRAFT_STORAGE_KEY, createEmptyVideoForm()),
  );
  const [form, setForm] = useState<VideoFormState>(draftBootstrap.value);
  const [restoredDraft, setRestoredDraft] = useState(draftBootstrap.restored);

  const trimmedTitle = form.title.trim();
  const trimmedDescription = form.description.trim();
  const trimmedCoverUrl = form.coverUrl.trim();
  const trimmedVideoUrl = form.videoUrl.trim();
  const trimmedDuration = form.duration.trim();
  const hasValidCoverUrl = !trimmedCoverUrl || isValidPublicUrl(trimmedCoverUrl);
  const hasValidVideoUrl = !trimmedVideoUrl || isValidPublicUrl(trimmedVideoUrl);
  const hasDraftContent = Boolean(
    trimmedTitle ||
      trimmedDescription ||
      trimmedCoverUrl ||
      trimmedVideoUrl ||
      trimmedDuration,
  );

  useEffect(() => {
    if (!hasDraftContent) {
      clearPublishDraft(VIDEO_DRAFT_STORAGE_KEY);
      return;
    }

    savePublishDraft(VIDEO_DRAFT_STORAGE_KEY, form);
  }, [form, hasDraftContent]);

  const mutation = useMutation({
    mutationFn: publishVideo,
    onSuccess: async (res) => {
      clearPublishDraft(VIDEO_DRAFT_STORAGE_KEY);
      setRestoredDraft(false);
      await invalidatePublishSurfaces(queryClient, currentUserId);
      await queryClient.invalidateQueries({ queryKey: userKeys.mePrefix() });
      showToast({
        tone: "success",
        title: "视频已发布",
        description: "正在跳转到内容详情。",
      });
      navigate(`/content/${res.content_id}`);
    },
    onError: (nextError: Error) => {
      showToast({
        tone: "error",
        title: "视频发布失败",
        description:
          nextError.message && nextError.message !== "视频发布失败"
            ? nextError.message
            : undefined,
      });
    },
  });

  useBeforeUnloadGuard(hasDraftContent && !mutation.isPending);

  function updateField<Key extends keyof VideoFormState>(key: Key, value: VideoFormState[Key]) {
    setForm((current) => ({
      ...current,
      [key]: value,
    }));
  }

  function handleClearDraft() {
    setForm(createEmptyVideoForm());
    setRestoredDraft(false);
    clearPublishDraft(VIDEO_DRAFT_STORAGE_KEY);
    showToast({
      tone: "info",
      title: "视频草稿已清空",
      description: "当前浏览器中的本地草稿已经移除。",
    });
  }

  function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();

    if (!trimmedTitle) {
      showToast({
        tone: "error",
        title: "标题还没有填写",
        description: "先给这个视频一个明确标题。",
      });
      return;
    }

    if (!trimmedCoverUrl || !hasValidCoverUrl) {
      showToast({
        tone: "error",
        title: "封面 URL 无效",
        description: "请输入可访问的 http 或 https 封面地址。",
      });
      return;
    }

    if (!trimmedVideoUrl || !hasValidVideoUrl) {
      showToast({
        tone: "error",
        title: "视频 URL 无效",
        description: "请输入可访问的 http 或 https 视频地址。",
      });
      return;
    }

    const parsedDuration = trimmedDuration ? Number.parseInt(trimmedDuration, 10) : undefined;
    const hasValidDuration =
      parsedDuration !== undefined && Number.isFinite(parsedDuration) && parsedDuration > 0;

    if (trimmedDuration && !hasValidDuration) {
      showToast({
        tone: "error",
        title: "视频时长无效",
        description: "请输入有效的秒数。",
      });
      return;
    }

    mutation.mutate({
      title: trimmedTitle,
      description: trimmedDescription || undefined,
      video_url: trimmedVideoUrl,
      cover_url: trimmedCoverUrl,
      duration: parsedDuration,
      visibility: Number(form.visibility),
    });
  }

  return (
    <section className="space-y-6">
      <PageHeader
        eyebrow="Video"
        title="发布视频"
        description="当前先支持填写公开视频地址和封面地址，上传直传与私密内容读链待后端补齐。"
        aside={
          <Link
            to="/publish"
            className="inline-flex rounded-full border border-slate-200 bg-white px-4 py-2 text-sm text-slate-600 transition hover:border-accent hover:text-accent"
          >
            返回发布入口
          </Link>
        }
      />

      <form className="space-y-6" onSubmit={handleSubmit}>
        <InlineNotice
          title="当前只支持公开视频发布"
          description="私密内容虽然有后端字段，但详情页和“我的发布”暂时都不会把它稳定展示出来；同时上传凭证链路仍未补齐，所以这里明确采用“公开 + URL”方案。"
          tone="soft"
        />

        {restoredDraft ? (
          <InlineNotice
            title="已恢复上次未发布的视频草稿"
            description="草稿只保存在当前浏览器。发布成功或手动清空后，这份本地草稿会自动移除。"
            tone="soft"
            action={
              <button
                type="button"
                onClick={() => setRestoredDraft(false)}
                className="rounded-full border border-slate-200 bg-white px-4 py-2 text-sm text-slate-600 transition hover:border-accent hover:text-accent"
              >
                知道了
              </button>
            }
          />
        ) : null}

        <div className="grid gap-6 xl:grid-cols-[1.12fr_0.88fr]">
          <div className="space-y-6">
            <section className="rounded-[32px] border border-slate-200 bg-white p-6 shadow-card">
              <div className="grid gap-5 lg:grid-cols-2">
                <label className="block text-sm text-slate-700">
                  标题
                  <input
                    value={form.title}
                    onChange={(event) => updateField("title", event.target.value)}
                    maxLength={100}
                    placeholder="给这个视频一个明确标题"
                    className="mt-1 w-full rounded-2xl border border-slate-200 px-4 py-3 outline-none ring-accent transition focus:ring"
                  />
                </label>

                <label className="block text-sm text-slate-700">
                  封面 URL
                  <input
                    value={form.coverUrl}
                    onChange={(event) => updateField("coverUrl", event.target.value)}
                    placeholder="https://..."
                    className="mt-1 w-full rounded-2xl border border-slate-200 px-4 py-3 outline-none ring-accent transition focus:ring"
                  />
                  <p
                    className={[
                      "mt-2 text-xs",
                      trimmedCoverUrl && !hasValidCoverUrl ? "text-ember" : "text-slate-500",
                    ].join(" ")}
                  >
                    {trimmedCoverUrl && !hasValidCoverUrl
                      ? "请输入可访问的 http 或 https 地址。"
                      : "封面会出现在推荐流、详情首屏和公开视频列表中。"}
                  </p>
                </label>
              </div>

              <div className="mt-5 grid gap-5 lg:grid-cols-2">
                <label className="block text-sm text-slate-700">
                  视频 URL
                  <input
                    value={form.videoUrl}
                    onChange={(event) => updateField("videoUrl", event.target.value)}
                    placeholder="https://..."
                    className="mt-1 w-full rounded-2xl border border-slate-200 px-4 py-3 outline-none ring-accent transition focus:ring"
                  />
                  <p
                    className={[
                      "mt-2 text-xs",
                      trimmedVideoUrl && !hasValidVideoUrl ? "text-ember" : "text-slate-500",
                    ].join(" ")}
                  >
                    {trimmedVideoUrl && !hasValidVideoUrl
                      ? "请输入可访问的 http 或 https 地址。"
                      : "如果地址可访问，右侧会优先显示视频预览。"}
                  </p>
                </label>

                <label className="block text-sm text-slate-700">
                  时长（秒）
                  <input
                    value={form.duration}
                    onChange={(event) => updateField("duration", event.target.value)}
                    inputMode="numeric"
                    placeholder="例如 120"
                    className="mt-1 w-full rounded-2xl border border-slate-200 px-4 py-3 outline-none ring-accent transition focus:ring"
                  />
                  <p className="mt-2 text-xs text-slate-500">
                    可留空；如果填写，必须是大于 0 的整数秒数。
                  </p>
                </label>
              </div>

              <div className="mt-5 grid gap-5 lg:grid-cols-[1fr_220px]">
                <label className="block text-sm text-slate-700">
                  描述
                  <textarea
                    value={form.description}
                    onChange={(event) => updateField("description", event.target.value)}
                    rows={4}
                    maxLength={500}
                    placeholder="用简短描述补充视频主题"
                    className="mt-1 w-full rounded-2xl border border-slate-200 px-4 py-3 outline-none ring-accent transition focus:ring"
                  />
                </label>

                <label className="block text-sm text-slate-700">
                  可见性
                  <select
                    value={form.visibility}
                    onChange={(event) => updateField("visibility", event.target.value)}
                    className="mt-1 w-full rounded-2xl border border-slate-200 px-4 py-3 outline-none ring-accent transition focus:ring"
                  >
                    <option value="10">公开</option>
                  </select>
                </label>
              </div>
            </section>

            <section className="rounded-[32px] border border-slate-200 bg-white p-6 shadow-card">
              <p className="text-xs uppercase tracking-[0.24em] text-slate-500">Tips</p>
              <h2 className="mt-2 font-display text-2xl font-semibold text-slate-900">
                当前输入要求
              </h2>
              <ul className="mt-4 space-y-2 text-sm leading-7 text-slate-600">
                <li>封面和视频都需要可访问的公网 URL。</li>
                <li>如果时长留空，后端会按默认值处理。</li>
                <li>后续接入上传后，会把这部分替换成文件选择和进度反馈。</li>
              </ul>
            </section>
          </div>

          <aside className="space-y-6">
            <VideoPreviewCard
              title={trimmedTitle}
              description={trimmedDescription}
              coverUrl={hasValidCoverUrl ? trimmedCoverUrl : undefined}
              videoUrl={hasValidVideoUrl ? trimmedVideoUrl : undefined}
              duration={form.duration}
            />

            <InlineNotice
              title={hasDraftContent ? "草稿会自动保存在当前浏览器" : "开始输入后会自动保存草稿"}
              description={
                hasDraftContent
                  ? "如果你中途离开页面，当前内容会保留在本地。发布成功或手动清空后，草稿会自动移除。"
                  : "本地草稿只保存在当前浏览器，不会上传到服务端。"
              }
              tone={hasDraftContent ? "soft" : "neutral"}
            />
          </aside>
        </div>

        <div className="flex flex-wrap gap-3">
          <button
            type="submit"
            disabled={mutation.isPending}
            className="rounded-full bg-ink px-5 py-2.5 text-sm font-medium text-white transition hover:bg-slate-800 disabled:opacity-60"
          >
            {mutation.isPending ? "发布中..." : "发布视频"}
          </button>
          <button
            type="button"
            onClick={handleClearDraft}
            disabled={!hasDraftContent}
            className="rounded-full border border-slate-200 bg-white px-5 py-2.5 text-sm text-slate-600 transition hover:border-accent hover:text-accent disabled:opacity-60"
          >
            清空草稿
          </button>
          <Link
            to="/studio"
            className="rounded-full border border-slate-200 bg-white px-5 py-2.5 text-sm text-slate-600 transition hover:border-accent hover:text-accent"
          >
            去我的发布
          </Link>
        </div>
      </form>
    </section>
  );
}
