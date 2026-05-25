'use client';

import Link from 'next/link';
import { useRouter } from 'next/navigation';
import { useMemo, useState } from 'react';
import type { HomepageMessage } from '@aira/shared';
import { DailyFortune } from '@/components/DailyFortune';
import { ErrorState } from '@/components/layout/StateDisplay';
import { useFetch } from '@/hooks/useFetch';
import { useAuth } from '@/lib/auth';
import {
  addHomepageMessage,
  deleteHomepageMessage,
  getHomepageMessages,
  updateHomepageMessage,
} from '@/lib/homepage';

export default function HomePage() {
  const router = useRouter();
  const { isLoggedIn, user } = useAuth();
  const [query, setQuery] = useState('');
  const [messageValue, setMessageValue] = useState('');
  const [messageError, setMessageError] = useState('');
  const [submittingMessage, setSubmittingMessage] = useState(false);
  const messagesQuery = useFetch(
    () => getHomepageMessages(),
    [],
  );

  const quickLinks = useMemo(() => (
    isLoggedIn
      ? [
        { href: '/courses', title: '课程广场', description: '搜索课程代码或课程名称，进入课程详情和试卷列表。' },
        { href: '/profile/wrongbook', title: '错题本', description: '按课程查看未掌握 / 已掌握 / 垃圾篓题目。' },
        { href: '/profile/favorites', title: '收藏题目', description: '按课程整理你打星的题目。' },
        { href: '/profile/records', title: '做题记录', description: '查看最近做题情况和练习轨迹。' },
      ]
      : [
        { href: '/courses', title: '课程广场', description: '浏览课程、搜索课程代码、进入试卷练习。' },
        { href: '/login', title: '登录', description: '登录后可以收藏题目、同步错题本和上传题解。' },
      ]
  ), [isLoggedIn]);

  const handleSearch = () => {
    const trimmed = query.trim();
    router.push(trimmed ? `/courses?query=${encodeURIComponent(trimmed)}` : '/courses');
  };

  const handleSubmitMessage = async (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    const content = messageValue.trim();
    if (!content) {
      setMessageError('留言内容不能为空。');
      return;
    }

    setSubmittingMessage(true);
    setMessageError('');
    try {
      await addHomepageMessage({ content });
      setMessageValue('');
      messagesQuery.refetch();
    } catch (error) {
      setMessageError(error instanceof Error ? error.message : '发布失败，请稍后重试。');
    } finally {
      setSubmittingMessage(false);
    }
  };

  return (
    <div className="space-y-10">
      <section className="overflow-hidden rounded-3xl border border-gray-200 bg-[radial-gradient(circle_at_top_left,_rgba(59,130,246,0.18),_transparent_35%),linear-gradient(135deg,_#ffffff,_#f8fafc_65%,_#eff6ff)] px-6 py-10 md:px-10">
        <div className="grid gap-8 lg:grid-cols-[minmax(0,1fr),320px]">
          <div>
            <div className="inline-flex items-center rounded-full border border-brand-200 bg-white/70 px-3 py-1 text-xs font-medium text-brand-700">
              AIRAWeb · 浙江大学课程题库协作平台
            </div>
            <h1 className="mt-4 text-3xl font-semibold tracking-tight text-gray-900 md:text-5xl">
              从课程、试卷、题解到错题本，围绕一门课把练习闭环做起来。
            </h1>
            <p className="mt-4 max-w-2xl text-sm leading-7 text-gray-600 md:text-base">
              课程广场负责发现资料，试卷页负责练习与模拟考，个人中心负责沉淀错题、收藏和做题记录。
              当前测试数据已接入 FDS，可直接用来验证核心流程。
            </p>

            <div className="mt-6 flex flex-col gap-3 sm:flex-row">
              <div className="flex flex-1 items-center rounded-xl border border-gray-200 bg-white shadow-sm">
                <input
                  value={query}
                  onChange={(e) => setQuery(e.target.value)}
                  onKeyDown={(e) => { if (e.key === 'Enter') handleSearch(); }}
                  placeholder="搜索课程名称或课程代码，例如：数据结构基础 / CS1018F"
                  className="min-w-0 flex-1 bg-transparent px-4 py-3 text-sm text-gray-700 outline-none"
                />
                <button
                  onClick={handleSearch}
                  className="mr-2 rounded-lg bg-brand-600 px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-brand-700"
                >
                  搜索课程
                </button>
              </div>
            </div>

            <div className="mt-6 flex flex-wrap items-center gap-3 text-sm">
              <Link href="/courses" className="rounded-md bg-gray-900 px-4 py-2 font-medium text-white transition-colors hover:bg-gray-800">
                进入课程广场
              </Link>
              {isLoggedIn ? (
                <Link href="/profile" className="rounded-md border border-gray-300 bg-white px-4 py-2 font-medium text-gray-700 transition-colors hover:bg-gray-50">
                  进入个人中心
                </Link>
              ) : (
                <Link href="/login" className="rounded-md border border-gray-300 bg-white px-4 py-2 font-medium text-gray-700 transition-colors hover:bg-gray-50">
                  登录后同步题解 / 错题本
                </Link>
              )}
            </div>
          </div>

          <div className="grid gap-3">
            <DailyFortune />
            <InfoCard title="课程广场" value="课程搜索 / 课程详情" />
            <InfoCard title="做题模式" value="刷题 / 模拟考" />
            <InfoCard title="个人沉淀" value="错题本 / 收藏 / 记录" />
          </div>
        </div>
      </section>

      <section className="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
        {quickLinks.map((item) => (
          <Link
            key={item.href}
            href={item.href}
            className="rounded-2xl border border-gray-200 bg-white px-5 py-5 transition-colors hover:border-brand-200 hover:bg-brand-50/40"
          >
            <div className="text-base font-semibold text-gray-900">{item.title}</div>
            <p className="mt-2 text-sm leading-6 text-gray-500">{item.description}</p>
          </Link>
        ))}
      </section>

      <section className="grid gap-4 lg:grid-cols-[1.2fr,0.8fr]">
        <div className="rounded-2xl border border-gray-200 bg-white px-6 py-6">
          <div className="text-lg font-semibold text-gray-900">当前测试范围</div>
          <div className="mt-4 space-y-3 text-sm leading-7 text-gray-600">
            <p>1. 已导入 FDS 课程、试卷和题目，可直接在课程页进入练习。</p>
            <p>2. 题解支持 Markdown / LaTeX、作者编辑、赞踩和 Top 3 展示。</p>
            <p>3. 做题页支持刷题模式与模拟考模式，模拟考可自定义时长和是否自动交卷。</p>
          </div>
        </div>

        <div className="rounded-2xl border border-gray-200 bg-white px-6 py-6">
          <div className="text-lg font-semibold text-gray-900">推荐验证路径</div>
          <ol className="mt-4 space-y-3 text-sm leading-7 text-gray-600">
            <li>1. 从课程广场搜索 `CS1018F` 并进入课程详情。</li>
            <li>2. 进入试卷，在刷题模式下验证题解、收藏和错题记录。</li>
            <li>3. 再切到模拟考模式，验证倒计时、目录跳转和交卷流程。</li>
          </ol>
        </div>
      </section>

      <section className="rounded-3xl border border-gray-200 bg-white p-6 shadow-sm">
        <div className="grid gap-6 lg:grid-cols-[0.95fr,1.05fr]">
          <div className="space-y-4">
            <div>
              <div className="inline-flex rounded-full border border-gray-200 bg-gray-50 px-3 py-1 text-xs font-medium text-gray-600">
                留言广场
              </div>
              <h2 className="mt-3 text-2xl font-semibold text-gray-900">留下你希望这个项目继续改进的地方</h2>
              <p className="mt-2 text-sm leading-7 text-gray-500">
                这里收集对题库、课程页、模拟考、题解协作和用户中心的改进建议。留言直接公开展示，方便持续迭代。
              </p>
            </div>

            {isLoggedIn ? (
              <form className="space-y-3 rounded-2xl border border-gray-200 bg-gray-50 p-4" onSubmit={handleSubmitMessage}>
                <textarea
                  value={messageValue}
                  onChange={(event) => setMessageValue(event.target.value)}
                  rows={6}
                  placeholder="例如：希望课程详情页补充章节导航、留言支持筛选、模拟考支持断点恢复。"
                  className="w-full rounded-2xl border border-gray-200 bg-white px-4 py-3 text-sm leading-6 outline-none focus:border-brand-500 focus:ring-1 focus:ring-brand-500"
                />
                {messageError ? <p className="text-sm text-red-600">{messageError}</p> : null}
                <div className="flex justify-end">
                  <button
                    type="submit"
                    disabled={submittingMessage}
                    className="rounded-xl bg-gray-900 px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-gray-800 disabled:cursor-not-allowed disabled:bg-gray-400"
                  >
                    {submittingMessage ? '发布中...' : '发布留言'}
                  </button>
                </div>
              </form>
            ) : (
              <div className="rounded-2xl border border-dashed border-gray-200 bg-gray-50 px-4 py-5 text-sm leading-7 text-gray-500">
                当前可先浏览其他用户的建议。登录后，你也可以直接发布对项目的改进意见。
              </div>
            )}
          </div>

          <div className="space-y-4">
            <div className="flex items-center justify-between gap-3">
              <div>
                <h3 className="text-lg font-semibold text-gray-900">最新留言</h3>
                <p className="mt-1 text-sm text-gray-500">按时间倒序展示首页反馈。</p>
              </div>
              <button
                type="button"
                onClick={messagesQuery.refetch}
                className="rounded-lg border border-gray-200 bg-white px-3 py-2 text-sm text-gray-600 transition-colors hover:bg-gray-50"
              >
                刷新
              </button>
            </div>

            {messagesQuery.error ? (
              <ErrorState message={messagesQuery.error} onRetry={messagesQuery.refetch} />
            ) : messagesQuery.loading ? (
              <div className="rounded-2xl border border-gray-200 bg-gray-50 px-4 py-5 text-sm text-gray-500">
                正在加载留言...
              </div>
            ) : messagesQuery.data && messagesQuery.data.length > 0 ? (
              <div className="space-y-3">
                {messagesQuery.data.map((item) => (
                  <HomepageMessageCard
                    key={item.id}
                    item={item}
                    currentUserId={user?.userId ?? null}
                    onChanged={messagesQuery.refetch}
                  />
                ))}
              </div>
            ) : (
              <div className="rounded-2xl border border-dashed border-gray-200 bg-gray-50 px-4 py-5 text-sm text-gray-500">
                还没有留言。你可以先发布第一条改进建议。
              </div>
            )}
          </div>
        </div>
      </section>
    </div>
  );
}

function InfoCard({ title, value }: { title: string; value: string }) {
  return (
    <div className="rounded-2xl border border-white/80 bg-white/80 px-5 py-5 shadow-sm">
      <div className="text-xs font-medium uppercase tracking-wide text-gray-400">{title}</div>
      <div className="mt-2 text-lg font-semibold text-gray-900">{value}</div>
    </div>
  );
}

function HomepageMessageCard({
  item,
  currentUserId,
  onChanged,
}: {
  item: HomepageMessage;
  currentUserId: string | null;
  onChanged: () => void;
}) {
  const isOwner = currentUserId === formatFrontendUserId(item.user_id);
  const [editing, setEditing] = useState(false);
  const [draft, setDraft] = useState(item.content);
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState('');

  const handleSave = async () => {
    const content = draft.trim();
    if (!content) {
      setError('留言内容不能为空。');
      return;
    }
    setSubmitting(true);
    setError('');
    try {
      await updateHomepageMessage(item.id, { content });
      setEditing(false);
      onChanged();
    } catch (saveError) {
      setError(saveError instanceof Error ? saveError.message : '保存失败，请稍后重试。');
    } finally {
      setSubmitting(false);
    }
  };

  const handleDelete = async () => {
    if (!window.confirm('确定删除这条留言吗？')) return;
    setSubmitting(true);
    setError('');
    try {
      await deleteHomepageMessage(item.id);
      onChanged();
    } catch (deleteError) {
      setError(deleteError instanceof Error ? deleteError.message : '删除失败，请稍后重试。');
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <article className="rounded-2xl border border-gray-200 bg-gray-50 p-4">
      <div className="flex items-start gap-3">
        {item.avatar_url ? (
          // eslint-disable-next-line @next/next/no-img-element
          <img
            src={item.avatar_url}
            alt={item.user_name}
            className="h-10 w-10 rounded-full object-cover"
          />
        ) : (
          <div className="flex h-10 w-10 items-center justify-center rounded-full bg-brand-600 text-sm font-semibold text-white">
            {item.user_name?.charAt(0)?.toUpperCase() ?? 'U'}
          </div>
        )}

        <div className="min-w-0 flex-1">
          <div className="flex flex-wrap items-start justify-between gap-3">
            <div className="flex flex-wrap items-center gap-x-3 gap-y-1">
              <div className="text-sm font-medium text-gray-900">{item.user_name || '匿名同学'}</div>
              <div className="text-xs text-gray-500">
                {new Intl.DateTimeFormat('zh-CN', { dateStyle: 'medium', timeStyle: 'short' }).format(new Date(item.created_at))}
              </div>
            </div>
            {isOwner ? (
              <div className="flex gap-2">
                <button
                  type="button"
                  onClick={() => {
                    setDraft(item.content);
                    setEditing((value) => !value);
                    setError('');
                  }}
                  disabled={submitting}
                  className="rounded-lg border border-gray-200 bg-white px-3 py-1.5 text-xs text-gray-600 transition-colors hover:bg-gray-100 disabled:cursor-not-allowed disabled:opacity-60"
                >
                  {editing ? '取消' : '编辑'}
                </button>
                <button
                  type="button"
                  onClick={handleDelete}
                  disabled={submitting}
                  className="rounded-lg border border-rose-200 bg-white px-3 py-1.5 text-xs text-rose-700 transition-colors hover:bg-rose-50 disabled:cursor-not-allowed disabled:opacity-60"
                >
                  删除
                </button>
              </div>
            ) : null}
          </div>
          {editing ? (
            <div className="mt-3 space-y-3">
              <textarea
                value={draft}
                onChange={(event) => setDraft(event.target.value)}
                rows={4}
                className="w-full rounded-2xl border border-gray-200 bg-white px-4 py-3 text-sm leading-6 outline-none focus:border-brand-500 focus:ring-1 focus:ring-brand-500"
              />
              <div className="flex justify-end">
                <button
                  type="button"
                  onClick={handleSave}
                  disabled={submitting}
                  className="rounded-lg bg-gray-900 px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-gray-800 disabled:cursor-not-allowed disabled:bg-gray-400"
                >
                  {submitting ? '保存中...' : '保存修改'}
                </button>
              </div>
            </div>
          ) : (
            <p className="mt-2 whitespace-pre-wrap text-sm leading-7 text-gray-600">{item.content}</p>
          )}
          {error ? <p className="mt-2 text-sm text-red-600">{error}</p> : null}
        </div>
      </div>
    </article>
  );
}

function formatFrontendUserId(id: number) {
  return `u-${String(id).padStart(8, '0')}`;
}
