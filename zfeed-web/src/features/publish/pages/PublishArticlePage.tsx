import { type ChangeEvent, type FormEvent, useEffect, useRef, useState } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { Link, useNavigate } from "react-router-dom";

import { useSessionStore } from "@/entities/session/model/session.store";
import { publishArticle } from "@/features/content/api/content.api";
import { uploadContentAsset } from "@/features/content/lib/upload";
import {
  clearPublishDraft,
  isValidPublicUrl,
  readPublishDraft,
  savePublishDraft,
  useUnsavedChangesGuard,
} from "@/features/publish/lib/publishDraft";
import { invalidatePublishSurfaces } from "@/shared/lib/query/cacheSync";
import {
  contentImageUploadRule,
  describeFileValidationRule,
} from "@/shared/lib/media/fileValidation";
import { userKeys } from "@/shared/lib/query/queryKeys";
import { ImageFallback } from "@/shared/ui/ImageFallback";
import { InlineNotice } from "@/shared/ui/InlineNotice";
import { PageHeader } from "@/shared/ui/PageHeader";
import { useToast } from "@/shared/ui/toast/toast.store";

const defaultVisibility = "10";
const ARTICLE_DRAFT_STORAGE_KEY = "zfeed-web-publish-article-draft";

type ArticleFormState = {
  title: string;
  description: string;
  cover: string;
  content: string;
  visibility: string;
};

function createEmptyArticleForm(): ArticleFormState {
  return {
    title: "",
    description: "",
    cover: "",
    content: "",
    visibility: defaultVisibility,
  };
}

function ArticlePreviewCard({
  title,
  description,
  cover,
  content,
}: {
  title: string;
  description: string;
  cover?: string;
  content: string;
}) {
  const excerpt = content.trim().slice(0, 180);

  return (
    <section className="overflow-hidden rounded-[32px] border border-slate-200 bg-white shadow-card">
      <ImageFallback
        src={cover}
        alt={title || "文章封面"}
        containerClassName="aspect-[16/10] bg-slate-100"
        imageClassName="h-full w-full object-cover"
      />

      <div className="space-y-4 p-5">
        <div className="flex items-center justify-between gap-3">
          <span className="rounded-full bg-[#eef7fb] px-3 py-1 text-xs font-medium text-slate-600">
            公开文章
          </span>
          <span className="text-xs text-slate-400">{content.trim().length} 字</span>
        </div>

        <div>
          <h2 className="line-clamp-2 font-display text-2xl font-semibold text-slate-900">
            {title || "你的文章标题会显示在这里"}
          </h2>
          <p className="mt-2 text-sm leading-7 text-slate-600">
            {description || "补一段描述，让浏览者更快判断这篇内容是否值得点开。"}
          </p>
        </div>

        <div className="rounded-[24px] bg-slate-50 p-4">
          <p className="text-xs uppercase tracking-[0.22em] text-slate-500">Excerpt</p>
          <p className="mt-3 whitespace-pre-wrap text-sm leading-7 text-slate-700">
            {excerpt || "正文前几段会在这里形成预览。开始写作后，你可以更直观地检查封面、标题和内容气质是否统一。"}
          </p>
        </div>
      </div>
    </section>
  );
}

export function PublishArticlePage() {
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const currentUserId = useSessionStore((state) => state.user?.userId ?? 0);
  const { showToast } = useToast();
  const coverFileInputRef = useRef<HTMLInputElement | null>(null);

  const [draftBootstrap] = useState(() =>
    readPublishDraft<ArticleFormState>(ARTICLE_DRAFT_STORAGE_KEY, createEmptyArticleForm()),
  );
  const [form, setForm] = useState<ArticleFormState>(draftBootstrap.value);
  const [restoredDraft, setRestoredDraft] = useState(draftBootstrap.restored);

  const trimmedTitle = form.title.trim();
  const trimmedDescription = form.description.trim();
  const trimmedCover = form.cover.trim();
  const trimmedContent = form.content.trim();
  const hasValidCoverUrl = !trimmedCover || isValidPublicUrl(trimmedCover);
  const hasDraftContent = Boolean(
    trimmedTitle || trimmedDescription || trimmedCover || trimmedContent,
  );

  useEffect(() => {
    if (!hasDraftContent) {
      clearPublishDraft(ARTICLE_DRAFT_STORAGE_KEY);
      return;
    }

    savePublishDraft(ARTICLE_DRAFT_STORAGE_KEY, form);
  }, [form, hasDraftContent]);

  const mutation = useMutation({
    mutationFn: publishArticle,
    onSuccess: async (res) => {
      clearPublishDraft(ARTICLE_DRAFT_STORAGE_KEY);
      setRestoredDraft(false);
      await invalidatePublishSurfaces(queryClient, currentUserId);
      await queryClient.invalidateQueries({ queryKey: userKeys.mePrefix() });
      showToast({
        tone: "success",
        title: "文章已发布",
        description: "正在跳转到内容详情。",
      });
      allowExit(() => navigate(`/content/${res.content_id}`));
    },
    onError: (nextError: Error) => {
      showToast({
        tone: "error",
        title: "文章发布失败",
        description:
          nextError.message && nextError.message !== "文章发布失败"
            ? nextError.message
            : undefined,
      });
    },
  });

  const coverUploadMutation = useMutation({
    mutationFn: (file: File) => uploadContentAsset(file, "article-cover"),
    onSuccess: (res) => {
      updateField("cover", res.url);
      showToast({
        tone: "success",
        title: "封面已上传",
        description: "新的封面地址已经回填到表单。",
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

  const allowExit = useUnsavedChangesGuard(hasDraftContent && !mutation.isPending);

  function updateField<Key extends keyof ArticleFormState>(
    key: Key,
    value: ArticleFormState[Key],
  ) {
    setForm((current) => ({
      ...current,
      [key]: value,
    }));
  }

  function handleClearDraft() {
    setForm(createEmptyArticleForm());
    setRestoredDraft(false);
    clearPublishDraft(ARTICLE_DRAFT_STORAGE_KEY);
    showToast({
      tone: "info",
      title: "文章草稿已清空",
      description: "当前浏览器中的本地草稿已经移除。",
    });
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

    if (!trimmedTitle) {
      showToast({
        tone: "error",
        title: "标题还没有填写",
        description: "先给这篇文章一个明确标题。",
      });
      return;
    }

    if (!trimmedCover || !hasValidCoverUrl) {
      showToast({
        tone: "error",
        title: "封面 URL 无效",
        description: "请输入可访问的 http 或 https 封面地址。",
      });
      return;
    }

    if (!trimmedContent) {
      showToast({
        tone: "error",
        title: "正文还没有内容",
        description: "至少写一点正文，再进行发布。",
      });
      return;
    }

    mutation.mutate({
      title: trimmedTitle,
      description: trimmedDescription || undefined,
      cover: trimmedCover,
      content: trimmedContent,
      visibility: Number(form.visibility),
    });
  }

  return (
    <section className="space-y-6">
      <PageHeader
        eyebrow="Article"
        title="发布文章"
        description="支持封面直传或手填 URL，当前继续聚焦公开文章发布链路。"
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
          title="公开文章链路已经可用"
          description="你可以直接上传文章封面，也可以继续手动填写 URL。当前仍优先聚焦公开文章发布。"
          tone="soft"
        />

        {restoredDraft ? (
          <InlineNotice
            title="已恢复上次未发布的文章草稿"
            description="草稿保存在当前浏览器中。发布成功或手动清空后，这份本地草稿会自动移除。"
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
                    placeholder="给这篇文章一个明确标题"
                    className="mt-1 w-full rounded-2xl border border-slate-200 px-4 py-3 outline-none ring-accent transition focus:ring"
                  />
                </label>

                <label className="block text-sm text-slate-700">
                  封面 URL
                  <input
                    value={form.cover}
                    onChange={(event) => updateField("cover", event.target.value)}
                    placeholder="https://..."
                    className="mt-1 w-full rounded-2xl border border-slate-200 px-4 py-3 outline-none ring-accent transition focus:ring"
                  />
                  <p
                    className={[
                      "mt-2 text-xs",
                      trimmedCover && !hasValidCoverUrl ? "text-ember" : "text-slate-500",
                    ].join(" ")}
                  >
                    {trimmedCover && !hasValidCoverUrl
                      ? "请输入可访问的 http 或 https 地址。"
                      : "封面会出现在推荐流、详情首屏和公开发布列表中。"}
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
                    maxLength={255}
                    placeholder="用 1 到 2 句话交代文章内容"
                    className="mt-1 w-full rounded-2xl border border-slate-200 px-4 py-3 outline-none ring-accent transition focus:ring"
                  />
                </label>

                <div className="flex flex-wrap gap-3">
                  <button
                    type="button"
                    onClick={() => coverFileInputRef.current?.click()}
                    aria-describedby="article-cover-upload-hint"
                    className="inline-flex rounded-full border border-slate-200 bg-white px-4 py-2 text-sm text-slate-600 transition hover:border-accent hover:text-accent"
                  >
                    {coverUploadMutation.isPending ? "上传中..." : "上传文章封面"}
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
                    disabled={!trimmedCover}
                    className="rounded-full border border-slate-200 bg-white px-4 py-2 text-sm text-slate-600 transition hover:border-accent hover:text-accent disabled:opacity-60"
                  >
                    清空封面
                  </button>
                </div>
                <p id="article-cover-upload-hint" className="text-xs text-slate-500">
                  上传文件支持 {describeFileValidationRule(contentImageUploadRule)}。
                </p>

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
              <div className="flex items-center justify-between gap-3">
                <div>
                  <p className="text-xs uppercase tracking-[0.24em] text-slate-500">Content</p>
                  <h2 className="mt-2 font-display text-2xl font-semibold text-slate-900">
                    正文内容
                  </h2>
                </div>
                <span className="rounded-full bg-[#eef7fb] px-4 py-2 text-sm text-slate-600">
                  {trimmedContent.length} 字
                </span>
              </div>

              <label className="mt-5 block text-sm text-slate-700">
                正文
                <textarea
                  value={form.content}
                  onChange={(event) => updateField("content", event.target.value)}
                  rows={16}
                  maxLength={1_000_000}
                  placeholder="写下正文，支持多段文本。"
                  className="mt-1 w-full rounded-[28px] border border-slate-200 px-4 py-4 text-sm leading-7 outline-none ring-accent transition focus:ring"
                />
              </label>
            </section>
          </div>

          <aside className="space-y-6">
            <ArticlePreviewCard
              title={trimmedTitle}
              description={trimmedDescription}
              cover={hasValidCoverUrl ? trimmedCover : undefined}
              content={trimmedContent}
            />

            <InlineNotice
              title={hasDraftContent ? "草稿会自动保存在当前浏览器" : "开始输入后会自动保存草稿"}
              description={
                hasDraftContent
                  ? "如果你刷新、关闭标签页或切换站内页面，都会先提醒你；当前内容也会继续保留在本地。"
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
            {mutation.isPending ? "发布中..." : "发布文章"}
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
