import { useQuery } from "@tanstack/react-query";
import { Link } from "react-router-dom";

import { useSessionStore } from "@/entities/session/model/session.store";
import { getMe } from "@/features/auth/api/auth.api";
import { userKeys } from "@/shared/lib/query/queryKeys";
import { ImageFallback } from "@/shared/ui/ImageFallback";
import { PageHeader } from "@/shared/ui/PageHeader";
import {
  PersonalMetricGrid,
  PersonalSpaceHero,
  PersonalSpaceInfoCard,
  PersonalSpaceSection,
} from "@/shared/ui/PersonalSpace";
import { StatePanel } from "@/shared/ui/StatePanel";

export function MePage() {
  const currentUserId = useSessionStore((state) => state.user?.userId ?? 0);
  const query = useQuery({
    queryKey: userKeys.me(currentUserId),
    queryFn: getMe,
  });

  if (query.isLoading) {
    return (
      <section className="space-y-6">
        <div className="h-12 w-40 rounded-full bg-white shadow-card" />
        <div className="h-56 rounded-[32px] bg-white shadow-card" />
        <div className="grid gap-3 md:grid-cols-4">
          {Array.from({ length: 4 }).map((_, index) => (
            <div key={index} className="h-24 rounded-[24px] bg-white shadow-card" />
          ))}
        </div>
        <div className="grid gap-6 lg:grid-cols-[1.1fr_0.9fr]">
          <div className="h-72 rounded-[32px] bg-white shadow-card" />
          <div className="h-72 rounded-[32px] bg-white shadow-card" />
        </div>
      </section>
    );
  }

  if (query.isError || !query.data) {
    return (
      <StatePanel
        title="个人信息加载失败"
        description={(query.error as Error)?.message || "请稍后重试"}
        tone="error"
      />
    );
  }

  const {
    user_info: user,
    followee_count,
    follower_count,
    like_received_count,
    favorite_received_count,
  } = query.data;

  return (
    <section className="space-y-6">
      <PageHeader
        eyebrow="Me"
        title="我的主页"
        description="集中查看你的公开形象、关系数据和常用入口。"
      />

      <PersonalSpaceHero
        eyebrow="My Space"
        identity={`ID ${user.user_id}`}
        title={user.nickname}
        description={user.bio || "这个人很神秘，还没有留下简介。"}
        media={
          <ImageFallback
            src={user.avatar}
            alt={user.nickname}
            name={user.nickname}
            variant="avatar"
            containerClassName="h-20 w-20 overflow-hidden rounded-full border border-white/70 bg-white/80"
            imageClassName="h-full w-full object-cover"
          />
        }
        aside={
          <div className="flex flex-wrap gap-3 lg:max-w-sm lg:justify-end">
            <Link
              to={`/users/${user.user_id}`}
              className="rounded-full bg-ink px-4 py-2 text-sm font-medium text-white transition hover:bg-slate-800"
            >
              查看公开主页
            </Link>
            <Link
              to="/publish"
              className="rounded-full border border-slate-200 bg-white px-4 py-2 text-sm text-slate-600 transition hover:border-accent hover:text-accent"
            >
              去发布
            </Link>
          </div>
        }
      />

      <PersonalMetricGrid
        items={[
          { label: "关注", value: followee_count },
          { label: "粉丝", value: follower_count },
          { label: "获赞", value: like_received_count },
          { label: "被收藏", value: favorite_received_count },
        ]}
        columns={4}
      />

      <div className="grid gap-6 lg:grid-cols-[1.1fr_0.9fr]">
        <PersonalSpaceSection
          eyebrow="Entry"
          title="常用入口"
          description="这里聚合你最常回访的个人内容空间和创作入口。"
        >
          <div className="grid gap-3 sm:grid-cols-2">
            <EntryLinkCard
              to="/favorites"
              title="我的收藏"
              description="回看你收藏过的内容，并检查跨页同步结果。"
            />
            <EntryLinkCard
              to="/studio"
              title="我的公开发布"
              description="浏览当前已经公开可见的作品与状态。"
            />
            <EntryLinkCard
              to="/following"
              title="关注流"
              description="回到关系驱动的浏览路径，检查关注内容更新。"
            />
            <EntryLinkCard
              to="/publish"
              title="发布入口"
              description="继续发布文章或视频，进入当前能力版创作链路。"
            />
          </div>
        </PersonalSpaceSection>

        <PersonalSpaceSection
          eyebrow="Profile"
          title="公开形象"
          description="这里说明当前主页在公开展示和后端能力上的边界。"
        >
          <div className="space-y-3">
            <PersonalSpaceInfoCard
              label="公开主页"
              value="可直接预览"
              description="点击“查看公开主页”即可看到其他用户当前看到的版本。"
            />
            <PersonalSpaceInfoCard
              label="资料编辑"
              value="等待后端补齐"
              description="昵称、简介和头像更新链路尚未开放，所以当前以浏览和回访为主。"
            />
            <PersonalSpaceInfoCard
              label="个人语义"
              value="公开形象 + 内容关系"
              description="我的主页不作为设置中心，而是你的公开身份与内容入口。"
            />
          </div>
        </PersonalSpaceSection>
      </div>
    </section>
  );
}

function EntryLinkCard({
  to,
  title,
  description,
}: {
  to: string;
  title: string;
  description: string;
}) {
  return (
    <Link
      to={to}
      className="rounded-[24px] border border-slate-200 bg-[linear-gradient(180deg,#ffffff,#f8fbfd)] p-4 transition hover:border-accent hover:-translate-y-0.5"
    >
      <p className="text-lg font-semibold text-slate-900">{title}</p>
      <p className="mt-2 text-sm leading-6 text-slate-500">{description}</p>
      <p className="mt-4 text-sm font-medium text-accent">进入</p>
    </Link>
  );
}
