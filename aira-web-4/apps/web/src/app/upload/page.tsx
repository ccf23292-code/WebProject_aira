/**
 * app/upload/page.tsx
 * 上传题库入口选择 — hub 页
 *
 * 把两种"给题库添砖加瓦"的方式放在一起：
 *   1. 上传文件（Ingest）：有 PDF/DOCX/MD/图片 → AI 清洗 → admin 审核 → 入题库
 *   2. 凭印象回忆（Recall）：考完试当场敲题，多人投票 → admin convert → 入题库
 *
 * 实际表单分别在：
 *   - /upload/file
 *   - /courses/[courseId]/recall
 */

'use client';

import Link from 'next/link';
import { useRouter } from 'next/navigation';
import { useEffect, useState } from 'react';
import type { Course } from '@aira/shared';
import { CourseCombobox } from '@/components/CourseCombobox';
import { api } from '@/lib/api';
import { useAuth } from '@/lib/auth';

export default function UploadHubPage() {
  const router = useRouter();
  const { isLoggedIn, loading: authLoading } = useAuth();

  const [courses, setCourses] = useState<Course[]>([]);
  const [recallCourseId, setRecallCourseId] = useState('');

  useEffect(() => {
    if (!isLoggedIn) return;
    api.get<Course[]>('/courses')
      .then(setCourses)
      .catch(() => {});
  }, [isLoggedIn]);

  if (authLoading) return <div className="py-16 text-center text-gray-500">正在加载...</div>;
  if (!isLoggedIn) {
    return (
      <div className="rounded-2xl border border-gray-200 bg-white p-8 text-center text-gray-600">
        请先 <Link href="/login" className="text-brand-600 hover:underline">登录</Link>，再选择上传方式。
      </div>
    );
  }

  function handleStartRecall(e: React.FormEvent) {
    e.preventDefault();
    if (!recallCourseId) return;
    router.push(`/courses/${encodeURIComponent(recallCourseId)}/recall`);
  }

  return (
    <div className="space-y-6">
      <header className="space-y-2">
        <h1 className="text-3xl font-semibold tracking-tight text-gray-900">给题库添题</h1>
        <p className="text-sm text-gray-600">
          有两种方式 —— 看你手头有什么，选合适的那个。两边最终都会经过管理员审核才正式入题库。
          <Link href="/upload/jobs" className="ml-2 text-brand-600 hover:underline">查看我的上传记录</Link>
        </p>
      </header>

      {/* 区别说明 */}
      <section className="rounded-2xl border border-gray-200 bg-gray-50 p-5 text-sm text-gray-700">
        <div className="mb-2 font-medium text-gray-900">该选哪个？</div>
        <div className="grid gap-3 md:grid-cols-2">
          <div>
            <div className="font-medium text-brand-700">📤 上传文件</div>
            <ul className="mt-1 list-disc space-y-0.5 pl-5 text-xs">
              <li>你手里有 <b>PDF / DOCX / Markdown / 图片</b></li>
              <li>整份文件一次性上传，AI 自动拆题</li>
              <li>30 秒内拿到结构化结果</li>
              <li>单人单文件，不参与众包投票</li>
            </ul>
          </div>
          <div>
            <div className="font-medium text-amber-700">✍️ 凭印象回忆</div>
            <ul className="mt-1 list-disc space-y-0.5 pl-5 text-xs">
              <li>刚考完试 <b>手头没材料</b>，只能凭脑子</li>
              <li>逐题打字填进网页表单</li>
              <li>同一道题别人也能"+1 我也记得"，按票数定稿</li>
              <li>不用 LLM，是同学间的众包接力</li>
            </ul>
          </div>
        </div>
      </section>

      {/* 两张卡片 */}
      <section className="grid gap-4 md:grid-cols-2">
        {/* 上传文件卡 */}
        <Link
          href="/upload/file"
          className="group rounded-3xl border border-gray-200 bg-white p-6 shadow-sm transition-shadow hover:border-brand-300 hover:shadow-md"
        >
          <div className="flex items-start gap-4">
            <div className="flex h-12 w-12 shrink-0 items-center justify-center rounded-2xl bg-brand-50 text-2xl">
              📤
            </div>
            <div className="min-w-0 flex-1">
              <h2 className="text-lg font-semibold text-gray-900 group-hover:text-brand-700">
                上传文件清洗
              </h2>
              <p className="mt-1 text-sm text-gray-600">
                有 PDF、DOCX、Markdown 或截图？传上来，AI 帮你结构化整理。
              </p>
              <div className="mt-3 inline-flex items-center text-sm font-medium text-brand-700 group-hover:gap-2">
                <span>开始上传</span>
                <span className="ml-1 transition-all group-hover:translate-x-0.5">→</span>
              </div>
            </div>
          </div>
        </Link>

        {/* 回忆卷卡 */}
        <div className="rounded-3xl border border-gray-200 bg-white p-6 shadow-sm transition-shadow hover:border-amber-300 hover:shadow-md">
          <div className="flex items-start gap-4">
            <div className="flex h-12 w-12 shrink-0 items-center justify-center rounded-2xl bg-amber-50 text-2xl">
              ✍️
            </div>
            <div className="min-w-0 flex-1">
              <h2 className="text-lg font-semibold text-gray-900">凭印象回忆卷</h2>
              <p className="mt-1 text-sm text-gray-600">
                没材料？选一门课，到回忆卷里逐题敲，和同学一起拼出整张卷。
              </p>

              <form onSubmit={handleStartRecall} className="mt-3 flex flex-col gap-2 sm:flex-row">
                <div className="flex-1">
                  <CourseCombobox
                    value={recallCourseId}
                    onChange={setRecallCourseId}
                    courses={courses}
                    placeholder="搜索课程名或代码"
                    tone="amber"
                  />
                </div>
                <button
                  type="submit"
                  disabled={!recallCourseId}
                  className="rounded-xl bg-amber-600 px-4 py-2 text-sm font-medium text-white shadow transition-colors hover:bg-amber-700 disabled:cursor-not-allowed disabled:bg-gray-300"
                >
                  进入回忆卷
                </button>
              </form>
              {courses.length === 0 && (
                <p className="mt-2 text-xs text-gray-500">课程列表加载中... 或者题库里还没有课程。</p>
              )}
            </div>
          </div>
        </div>
      </section>
    </div>
  );
}
