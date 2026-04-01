/**
 * app/courses/page.tsx
 * 课程列表 — 首页，展示所有课程卡片
 * 对接: GET /api/courses → Course[]
 */

'use client';

import { useMemo, useState } from 'react';
import Link from 'next/link';
import type { Course } from '@aira/shared';
import { TableSkeleton } from '@/components/layout/Skeleton';
import { ErrorState } from '@/components/layout/StateDisplay';
import { useFetch } from '@/hooks/useFetch';
import { api } from '@/lib/api';

export default function CoursesPage() {
  const [query, setQuery] = useState('');
  const queryParam = useMemo(() => query.trim(), [query]);

  const { data: courses, loading, error, refetch } = useFetch(
    () => api.get<Course[]>(queryParam ? `/courses?q=${encodeURIComponent(queryParam)}` : '/courses'),
    [queryParam],
  );

  return (
    <div>
      <div className="mb-6">
        <h1 className="text-xl font-semibold text-gray-900">选择课程</h1>
        <p className="mt-1 text-sm text-gray-500">选择一门课程，开始刷题</p>
        <div className="mt-4">
          <input
            type="text"
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            placeholder="搜索课程名称或课程代码"
            className="w-full rounded-md border border-gray-200 px-3 py-2 text-sm outline-none
                       focus:border-brand-500 focus:ring-1 focus:ring-brand-500"
          />
        </div>
      </div>

      {loading ? (
        <TableSkeleton rows={4} />
      ) : error ? (
        <ErrorState message={error} onRetry={refetch} />
      ) : courses && courses.length > 0 ? (
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {courses.map((c) => (
            <CourseCard key={c.id} course={c} />
          ))}
        </div>
      ) : (
        <div className="py-16 text-center text-sm text-gray-400">暂无课程</div>
      )}
    </div>
  );
}

function CourseCard({ course }: { course: Course }) {
  return (
    <Link href={`/courses/${encodeURIComponent(course.id)}`}
      className="group block rounded-xl border border-gray-200 bg-white p-5 transition-all
                 hover:border-brand-300 hover:shadow-sm">
      <h2 className="text-base font-semibold text-gray-900 group-hover:text-brand-600 transition-colors">
        {course.name}
      </h2>
      <div className="mt-2 flex flex-wrap items-center gap-2 text-xs text-gray-500">
        <span className="rounded-full bg-gray-100 px-2 py-0.5">{course.code || course.id}</span>
        <span>学分 {course.credits.toFixed(1)}</span>
      </div>
      <div className="mt-3 text-xs font-medium text-brand-600">查看试卷 →</div>
    </Link>
  );
}
