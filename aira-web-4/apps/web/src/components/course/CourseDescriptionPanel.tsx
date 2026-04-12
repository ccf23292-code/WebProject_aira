'use client';

import { useMemo, useState } from 'react';
import type { Course, CourseDescriptionSubmission } from '@aira/shared';
import { useFetch } from '@/hooks/useFetch';
import { useAuth } from '@/lib/auth';
import {
  getMyCourseDescriptionSubmissions,
  submitCourseDescriptionSuggestion,
} from '@/lib/courseDescription';
import { ErrorState } from '@/components/layout/StateDisplay';

interface CourseDescriptionPanelProps {
  course: Course;
}

const STATUS_LABEL: Record<CourseDescriptionSubmission['status'], string> = {
  pending: '待审核',
  approved: '已通过',
  rejected: '已驳回',
};

const STATUS_CLASS: Record<CourseDescriptionSubmission['status'], string> = {
  pending: 'border-amber-200 bg-amber-50 text-amber-700',
  approved: 'border-emerald-200 bg-emerald-50 text-emerald-700',
  rejected: 'border-rose-200 bg-rose-50 text-rose-700',
};

export function CourseDescriptionPanel({ course }: CourseDescriptionPanelProps) {
  const { isLoggedIn } = useAuth();
  const [value, setValue] = useState('');
  const [submitting, setSubmitting] = useState(false);
  const [submitError, setSubmitError] = useState('');
  const submissionsQuery = useFetch(
    () => (isLoggedIn ? getMyCourseDescriptionSubmissions(course.id) : Promise.resolve([])),
    [course.id, isLoggedIn],
  );

  const latestSubmission = useMemo(
    () => submissionsQuery.data?.[0] ?? null,
    [submissionsQuery.data],
  );

  const handleSubmit = async (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    const content = value.trim();
    if (!content) {
      setSubmitError('简介内容不能为空。');
      return;
    }

    setSubmitting(true);
    setSubmitError('');
    try {
      await submitCourseDescriptionSuggestion(course.id, { content });
      setValue('');
      submissionsQuery.refetch();
    } catch (error) {
      setSubmitError(error instanceof Error ? error.message : '提交失败，请稍后重试。');
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <section className="rounded-3xl border border-gray-200 bg-white p-6 shadow-sm">
      <div className="grid gap-6 lg:grid-cols-[1.05fr,0.95fr]">
        <div className="space-y-4">
          <div>
            <h2 className="text-xl font-semibold text-gray-900">课程简介</h2>
            <p className="mt-1 text-sm leading-6 text-gray-500">
              这里展示当前已经发布的课程简介。课程广场中的卡片也会同步显示这段内容。
            </p>
          </div>
          <div className="rounded-2xl border border-gray-200 bg-gray-50 p-4">
            <p className="text-sm leading-7 text-gray-700 whitespace-pre-wrap">
              {course.description?.trim() || '当前还没有正式课程简介，你可以提交一版供管理员审核。'}
            </p>
          </div>
        </div>

        <div className="space-y-4">
          <div>
            <h3 className="text-lg font-semibold text-gray-900">提交简介修改</h3>
            <p className="mt-1 text-sm leading-6 text-gray-500">
              普通用户提交后需要管理员审核，审核通过后才会替换公开简介。
            </p>
          </div>

          {isLoggedIn ? (
            <form className="space-y-3 rounded-2xl border border-gray-200 bg-gray-50 p-4" onSubmit={handleSubmit}>
              <textarea
                value={value}
                onChange={(event) => setValue(event.target.value)}
                rows={6}
                placeholder="写一版更准确的课程简介，例如重点章节、考试侧重点、适合的学习路径。"
                className="w-full rounded-2xl border border-gray-200 bg-white px-4 py-3 text-sm leading-6 outline-none focus:border-brand-500 focus:ring-1 focus:ring-brand-500"
              />
              {submitError ? <p className="text-sm text-red-600">{submitError}</p> : null}
              <div className="flex justify-end">
                <button
                  type="submit"
                  disabled={submitting}
                  className="rounded-xl bg-brand-600 px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-brand-700 disabled:cursor-not-allowed disabled:bg-brand-300"
                >
                  {submitting ? '提交中...' : '提交简介修改'}
                </button>
              </div>
            </form>
          ) : (
            <div className="rounded-2xl border border-dashed border-gray-200 bg-gray-50 px-4 py-5 text-sm text-gray-500">
              登录后才能提交课程简介修改。
            </div>
          )}

          {submissionsQuery.error ? (
            <ErrorState message={submissionsQuery.error} onRetry={submissionsQuery.refetch} />
          ) : latestSubmission ? (
            <div className="rounded-2xl border border-gray-200 bg-white p-4">
              <div className="flex items-center justify-between gap-3">
                <div className="text-sm font-medium text-gray-900">你最近一次提交</div>
                <span className={`rounded-full border px-3 py-1 text-xs font-medium ${STATUS_CLASS[latestSubmission.status]}`}>
                  {STATUS_LABEL[latestSubmission.status]}
                </span>
              </div>
              <p className="mt-3 whitespace-pre-wrap text-sm leading-7 text-gray-700">
                {latestSubmission.content}
              </p>
              {latestSubmission.review_note ? (
                <p className="mt-3 rounded-xl bg-gray-50 px-3 py-2 text-xs leading-6 text-gray-500">
                  审核备注：{latestSubmission.review_note}
                </p>
              ) : null}
            </div>
          ) : (
            <div className="rounded-2xl border border-dashed border-gray-200 bg-gray-50 px-4 py-5 text-sm text-gray-500">
              你还没有提交过课程简介修改。
            </div>
          )}
        </div>
      </div>
    </section>
  );
}
