import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { type ChangeEvent, type FormEvent, useEffect, useRef, useState } from "react";
import { Link, useNavigate, useParams } from "react-router-dom";

import {
  editArticle,
  getContentDetail,
} from "@/features/content/api/content.api";
import { uploadContentAsset } from "@/features/content/lib/upload";
import { useUnsavedChangesGuard } from "@/features/publish/lib/publishDraft";
import { isValidHttpUrl } from "@/shared/lib/form/valueValidation";
import {
  contentImageUploadRule,
  describeFileValidationRule,
} from "@/shared/lib/media/fileValidation";
import { contentKeys, feedKeys, userKeys } from "@/shared/lib/query/queryKeys";
import { ImageFallback } from "@/shared/ui/ImageFallback";
import { InlineNotice } from "@/shared/ui/InlineNotice";
import { PageHeader } from "@/shared/ui/PageHeader";
import { StatePanel } from "@/shared/ui/StatePanel";
import { useToast } from "@/shared/ui/toast/toast.store";
import { useSessionStore } from "@/entities/session/model/session.store";

type ArticleFormState = {
  title: string;
  description: string;
  cover: string;
  content: string;
};

function buildInitialForm(data: Awaited<ReturnType<typeof getContentDetail>>["detail"]): ArticleFormState {
  return {
    title: data.title || "",
    description: data.description || "",
    cover: data.cover_url || "",
    content: data.article_content || "",
  };
}

function hasArticleFormChanges(current: ArticleFormState, initial: ArticleFormState | null) {
  if (!initial) {
    return false;
  }

  return (
    current.title.trim() !== initial.title.trim() ||
    current.description.trim() !== initial.description.trim() ||
    current.cover.trim() !== initial.cover.trim() ||
    current.content.trim() !== initial.content.trim()
  );
}

export function EditArticlePage() {
  const params = useParams();
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const currentUserId = useSessionStore((state) => state.user?.userId ?? 0);
  const { showToast } = useToast();
  const coverFileInputRef = useRef<HTMLInputElement | null>(null);

  const contentId = Number(params.contentId);
  const isValidContentId = Number.isInteger(contentId) && contentId > 0;

  const query = useQuery({
    queryKey: contentKeys.detail(contentId, currentUserId),
    enabled: isValidContentId,
    queryFn: () => getContentDetail({ content_id: contentId }),
  });

  const [form, setForm] = useState<ArticleFormState>({
    title: "",
    description: "",
    cover: "",
    content: "",
  });
  const [initialForm, setInitialForm] = useState<ArticleFormState | null>(null);
  const hasUnsavedChanges = hasArticleFormChanges(form, initialForm);
  const trimmedTitle = form.title.trim();
  const trimmedDescription = form.description.trim();
  const trimmedCover = form.cover.trim();
  const trimmedContent = form.content.trim();
  const hasValidCoverUrl = !trimmedCover || isValidHttpUrl(trimmedCover);

  useEffect(() => {
    if (!query.data?.detail) {
      return;
    }
    const next = buildInitialForm(query.data.detail);
    setForm(next);
    setInitialForm(next);
  }, [query.data]);

  const coverUploadMutation = useMutation({
    mutationFn: (file: File) => uploadContentAsset(file, "article-cover"),
    onSuccess: (res) => {
      setForm((current) => ({ ...current, cover: res.url }));
      showToast({
        tone: "success",
        title: "封面已上传",
        description: "上传完成，新的封面地址已经回填。",
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

  const mutation = useMutation({
    mutationFn: (payload: Parameters<typeof editArticle>[1]) => editArticle(contentId, payload),
    onSuccess: async () => {
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: contentKeys.detail(contentId, currentUserId) }),
        queryClient.invalidateQueries({ queryKey: feedKeys.userPublishPrefix(currentUserId) }),
        queryClient.invalidateQueries({ queryKey: feedKeys.studioPublishPrefix(currentUserId) }),
        queryClient.invalidateQueries({ queryKey: userKeys.profilePrefix(currentUserId) }),
      ]);
      showToast({
        tone: "success",
        title: "文章已更新",
        description: "已同步刷新详情页和发布列表。",
      });
      allowExit(() => navigate(`/content/${contentId}`, { replace: true }));
    },
    onError: (error: Error) => {
      showToast({
        tone: "error",
        title: "文章更新失败",
        description: error.message || "请稍后重试。",
      });
    },
  });

  const allowExit = useUnsavedChangesGuard(hasUnsavedChanges && !mutation.isPending);

  function updateField<Key extends keyof ArticleFormState>(key: Key, value: ArticleFormState[Key]) {
    setForm((current) => ({ ...current, [key]: value }));
  }

  function handleCoverFileChange(event: ChangeEvent<HTMLInputElement>) {
    const file = event.target.files?.[0];
    event.target.value = "";
    if (!file) {
      return;
    }
    coverUploadMutation.mutate(file);
  }

  function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!initialForm) {
      return;
    }

    const payload: Parameters<typeof editArticle>[1] = {};
    if (!trimmedTitle) {
      showToast({
        tone: "error",
        title: "标题不能为空",
        description: "请至少保留一个明确标题，再保存文章。",
      });
      return;
    }

    if (!trimmedCover || !hasValidCoverUrl) {
      showToast({
        tone: "error",
        title: "封面 URL 无效",
        description: "请输入可访问的 http / https 封面地址。",
      });
      return;
    }

    if (!trimmedContent) {
      showToast({
        tone: "error",
        title: "正文不能为空",
        description: "文章至少需要保留一段正文内容。",
      });
      return;
    }

    if (trimmedTitle !== initialForm.title.trim()) {
      payload.title = trimmedTitle;
    }
    if (trimmedDescription !== initialForm.description.trim()) {
      payload.description = trimmedDescription;
    }
    if (trimmedCover !== initialForm.cover.trim()) {
      payload.cover = trimmedCover;
    }
    if (trimmedContent !== initialForm.content.trim()) {
      payload.content = trimmedContent;
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
    detail.content_type !== 10 ||
    detail.author_id !== currentUserId
  ) {
    return (
      <StatePanel
        title="当前文章不可编辑"
        description="请确认这是一篇属于你自己的文章。"
        tone="error"
      />
    );
  }

  return (
    <section className="space-y-6">
      <PageHeader
        eyebrow="Edit Article"
        title="编辑文章"
        description="修改已经公开发布的文章内容，封面支持直传，正文支持直接更新。"
        aside={
          <Link
            to={`/content/${contentId}`}
            className="inline-flex rounded-full border border-slate-200 bg-white px-4 py-2 text-sm text-slate-600 transition hover:border-accent hover:text-accent"
          >
            返回详情
          </Link>
        }
      />

      <div className="grid gap-6 xl:grid-cols-[1.12fr_0.88fr]">
        <form className="space-y-5 rounded-[32px] border border-slate-200 bg-white p-6 shadow-card" onSubmit={handleSubmit}>
          <InlineNotice
            title="封面支持走上传签名链"
            description="如果 OSS 配置可用，这里可以直接上传新封面；也可以继续手动填写 URL。"
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
              maxLength={255}
              className="mt-1 w-full resize-none rounded-[24px] border border-slate-200 px-4 py-3 outline-none ring-accent transition focus:ring"
            />
            <span className="mt-2 block text-xs text-slate-400">{trimmedDescription.length}/255</span>
          </label>

          <div className="space-y-3">
            <label className="block text-sm text-slate-700">
              封面 URL
              <input
                value={form.cover}
                onChange={(event) => updateField("cover", event.target.value)}
                className="mt-1 w-full rounded-2xl border border-slate-200 px-4 py-3 outline-none ring-accent transition focus:ring"
              />
              <span
                className={[
                  "mt-2 block text-xs",
                  trimmedCover && !hasValidCoverUrl ? "text-ember" : "text-slate-400",
                ].join(" ")}
              >
                {trimmedCover && !hasValidCoverUrl
                  ? "请输入可访问的 http / https 封面地址。"
                  : "封面会继续出现在推荐流、详情首屏和公开发布列表中。"}
              </span>
            </label>
            <div className="flex flex-wrap gap-3">
              <button
                type="button"
                onClick={() => coverFileInputRef.current?.click()}
                aria-describedby="edit-article-cover-upload-hint"
                className="inline-flex rounded-full border border-slate-200 bg-white px-4 py-2 text-sm text-slate-600 transition hover:border-accent hover:text-accent"
              >
                {coverUploadMutation.isPending ? "上传中..." : "上传新封面"}
              </button>
              <input
                ref={coverFileInputRef}
                type="file"
                accept="image/png,image/jpeg,image/webp"
                tabIndex={-1}
                className="hidden"
                onChange={handleCoverFileChange}
              />
              <button
                type="button"
                onClick={() => updateField("cover", "")}
                disabled={!form.cover.trim()}
                className="rounded-full border border-slate-200 bg-white px-4 py-2 text-sm text-slate-600 transition hover:border-accent hover:text-accent disabled:opacity-60"
              >
                清空封面
              </button>
            </div>
            <p id="edit-article-cover-upload-hint" className="text-xs text-slate-500">
              上传文件支持 {describeFileValidationRule(contentImageUploadRule)}。
            </p>
          </div>

          <label className="block text-sm text-slate-700">
            正文
            <textarea
              value={form.content}
              onChange={(event) => updateField("content", event.target.value)}
              rows={16}
              className="mt-1 w-full resize-y rounded-[24px] border border-slate-200 px-4 py-3 outline-none ring-accent transition focus:ring"
            />
            <span className="mt-2 block text-xs text-slate-400">{trimmedContent.length} 字</span>
          </label>

          <button
            type="submit"
            disabled={mutation.isPending}
            className="rounded-full bg-ink px-5 py-2.5 text-sm font-medium text-white transition hover:bg-slate-800 disabled:opacity-60"
          >
            {mutation.isPending ? "保存中..." : "保存文章"}
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
          <ImageFallback
            src={hasValidCoverUrl ? form.cover : undefined}
            alt={form.title || "文章封面"}
            containerClassName="aspect-[16/10] bg-slate-100"
            imageClassName="h-full w-full object-cover"
          />
          <div className="space-y-4 p-5">
            <p className="text-xs uppercase tracking-[0.24em] text-slate-400">Preview</p>
            <h2 className="font-display text-3xl font-semibold text-slate-900">
              {form.title.trim() || "你的文章标题会显示在这里"}
            </h2>
            <p className="text-sm leading-7 text-slate-600">
              {form.description.trim() || "描述会帮助浏览者更快判断这篇文章的内容方向。"}
            </p>
            <div className="rounded-[24px] bg-slate-50 p-4 text-sm leading-7 text-slate-700">
              {form.content.trim().slice(0, 260) || "正文预览会展示在这里。"}
            </div>
          </div>
          </section>
        </div>
      </div>
    </section>
  );
}
