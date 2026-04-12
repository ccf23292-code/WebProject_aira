'use client';

import Link from 'next/link';
import { useParams } from 'next/navigation';
import type { ReactNode } from 'react';
import type { Course, Paper } from '@aira/shared';
import { CourseDescriptionPanel } from '@/components/course/CourseDescriptionPanel';
import { CourseCommunityPanel } from '@/components/course/CourseCommunityPanel';
import { DetailSkeleton } from '@/components/layout/Skeleton';
import { EmptyState, ErrorState } from '@/components/layout/StateDisplay';
import { useFetch } from '@/hooks/useFetch';
import { api } from '@/lib/api';

export default function CourseDetailPage() {
  const { courseId } = useParams<{ courseId: string }>();

  const courseQuery = useFetch(
    () => api.get<Course>(`/courses/${encodeURIComponent(courseId)}`),
    [courseId],
  );

  const papersQuery = useFetch(
    () => api.get<Paper[]>(`/courses/${encodeURIComponent(courseId)}/papers`),
    [courseId],
  );

  const course = courseQuery.data;
  const courseName = course?.name ?? courseId;

  if (courseQuery.loading && !course) {
    return <DetailSkeleton />;
  }

  return (
    <div className="space-y-8">
      <nav className="flex flex-wrap items-center gap-2 text-sm text-gray-500">
        <Link href="/courses" className="transition-colors hover:text-brand-600">
          课程广场
        </Link>
        <span>/</span>
        <span className="font-medium text-gray-900">{courseName}</span>
      </nav>

      <section className="overflow-hidden rounded-3xl border border-gray-200 bg-[linear-gradient(135deg,_#ffffff,_#f8fafc_55%,_#eef6ff)] shadow-sm">
        <div className="grid gap-6 p-6 lg:grid-cols-[1.2fr,0.8fr] lg:p-8">
          <div>
            <div className="inline-flex rounded-full border border-brand-200 bg-white/80 px-3 py-1 text-xs font-medium text-brand-700">
              Course Detail Workspace
            </div>
            <h1 className="mt-4 text-3xl font-semibold tracking-tight text-gray-900">
              {courseName}
            </h1>

            {courseQuery.error ? (
              <div className="mt-4 rounded-2xl border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700">
                课程详情加载失败：{courseQuery.error}
              </div>
            ) : null}

            {course ? (
              <div className="mt-5 flex flex-wrap gap-2">
                <MetaPill>{course.code || course.id}</MetaPill>
                <MetaPill>{course.credits.toFixed(1)} 学分</MetaPill>
                {course.college ? <MetaPill>{course.college}</MetaPill> : null}
                {course.category ? <MetaPill>{course.category}</MetaPill> : null}
              </div>
            ) : null}
          </div>

          <div className="grid gap-3 sm:grid-cols-3 lg:grid-cols-1">
            <QuickStat
              label="试卷数"
              value={papersQuery.data ? String(papersQuery.data.length) : '...'}
            />
            <QuickStat
              label="回忆卷"
              value="已接入"
            />
            <Link
              href={`/courses/${encodeURIComponent(courseId)}/recall`}
              className="flex min-h-28 flex-col justify-between rounded-3xl border border-orange-200 bg-orange-50 p-5 text-orange-800 transition-colors hover:bg-orange-100"
            >
              <div className="text-sm font-medium">协作回忆卷</div>
              <div className="text-lg font-semibold">进入回忆卷空间</div>
            </Link>
          </div>
        </div>
      </section>

      <section className="rounded-3xl border border-gray-200 bg-white p-6 shadow-sm">
        <div className="mb-5 flex items-center justify-between gap-4">
          <div>
            <h2 className="text-xl font-semibold text-gray-900">试卷列表</h2>
          </div>
        </div>

        {papersQuery.loading ? (
          <DetailSkeleton />
        ) : papersQuery.error ? (
          <ErrorState message={papersQuery.error} onRetry={papersQuery.refetch} />
        ) : papersQuery.data && papersQuery.data.length > 0 ? (
          <div className="space-y-3">
            {papersQuery.data.map((paper) => (
              <PaperRow key={paper.id} paper={paper} />
            ))}
          </div>
        ) : (
          <EmptyState
            title="当前课程还没有试卷"
            description="试卷上传后会自动出现在这里。"
          />
        )}
      </section>

      {course ? <CourseDescriptionPanel course={course} /> : null}

      <CourseCommunityPanel courseId={courseId} courseName={courseName} />
    </div>
  );
}

function PaperRow({ paper }: { paper: Paper }) {
  const date = new Intl.DateTimeFormat('zh-CN', {
    year: 'numeric',
    month: 'short',
    day: 'numeric',
  }).format(new Date(paper.created_at));

  return (
    <Link
      href={`/papers/${paper.id}`}
      className="flex items-center justify-between gap-4 rounded-2xl border border-gray-200 p-5 transition-colors hover:border-brand-300 hover:bg-brand-50/40"
    >
      <div>
        <h3 className="text-base font-semibold text-gray-900">{paper.name}</h3>
        <p className="mt-2 text-sm text-gray-500">上传时间：{date}</p>
      </div>
      <span className="rounded-full bg-brand-50 px-3 py-1 text-sm font-medium text-brand-700">
        开始练习
      </span>
    </Link>
  );
}

function MetaPill({ children }: { children: ReactNode }) {
  return (
    <span className="rounded-full bg-white px-3 py-1 text-sm text-gray-600 shadow-sm">
      {children}
    </span>
  );
}

function QuickStat({
  label,
  value,
}: {
  label: string;
  value: string;
}) {
  return (
    <div className="rounded-3xl border border-gray-200 bg-white p-5">
      <div className="text-sm font-medium text-gray-500">{label}</div>
      <div className="mt-2 text-2xl font-semibold text-gray-900">{value}</div>
    </div>
  );
}
