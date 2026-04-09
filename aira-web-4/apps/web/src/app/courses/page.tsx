'use client';

import Link from 'next/link';
import { useDeferredValue, useEffect, useMemo, useState } from 'react';
import { useRouter } from 'next/navigation';
import type { Course } from '@aira/shared';
import { TableSkeleton } from '@/components/layout/Skeleton';
import { ErrorState } from '@/components/layout/StateDisplay';
import { useFetch } from '@/hooks/useFetch';
import { api } from '@/lib/api';

export default function CoursesPage() {
  const router = useRouter();
  const [query, setQuery] = useState('');
  const [queryReady, setQueryReady] = useState(false);

  useEffect(() => {
    const params = new URLSearchParams(window.location.search);
    setQuery(params.get('query') ?? params.get('q') ?? '');
    setQueryReady(true);
  }, []);

  const deferredQuery = useDeferredValue(query.trim());

  useEffect(() => {
    if (!queryReady) return;

    const params = new URLSearchParams(window.location.search);
    if (deferredQuery) {
      params.set('query', deferredQuery);
    } else {
      params.delete('query');
    }
    params.delete('q');

    const nextUrl = params.toString() ? `/courses?${params.toString()}` : '/courses';
    const currentUrl = `${window.location.pathname}${window.location.search}`;

    if (nextUrl !== currentUrl) {
      router.replace(nextUrl, { scroll: false });
    }
  }, [deferredQuery, queryReady, router]);

  const coursesQuery = useFetch(
    () => api.get<Course[]>(
      deferredQuery
        ? `/courses?query=${encodeURIComponent(deferredQuery)}`
        : '/courses',
    ),
    [deferredQuery],
  );

  const summary = useMemo(() => {
    if (!coursesQuery.data) return '正在加载课程列表...';
    if (!coursesQuery.data.length) return '暂时没有匹配课程。';
    return `共找到 ${coursesQuery.data.length} 门课程。`;
  }, [coursesQuery.data]);

  return (
    <div className="space-y-6">
      <section className="rounded-3xl border border-gray-200 bg-[linear-gradient(135deg,_#ffffff,_#f8fafc_65%,_#eef4ff)] p-6 shadow-sm">
        <div className="max-w-3xl">
          <div className="inline-flex rounded-full border border-brand-200 bg-white/80 px-3 py-1 text-xs font-medium text-brand-700">
            Courses API Workspace
          </div>
          <h1 className="mt-4 text-3xl font-semibold tracking-tight text-gray-900">
            课程广场
          </h1>
        </div>

        <div className="mt-6 rounded-2xl border border-gray-200 bg-white p-3 shadow-sm">
          <div className="flex flex-col gap-3 md:flex-row">
            <input
              type="text"
              value={query}
              onChange={(event) => setQuery(event.target.value)}
              placeholder="搜索课程名称或课程代码"
              className="min-w-0 flex-1 rounded-xl border border-gray-200 px-4 py-3 text-sm outline-none focus:border-brand-500 focus:ring-1 focus:ring-brand-500"
            />
            <button
              type="button"
              onClick={() => setQuery(query.trim())}
              className="rounded-xl bg-brand-600 px-5 py-3 text-sm font-medium text-white transition-colors hover:bg-brand-700"
            >
              立即搜索
            </button>
          </div>
          <p className="mt-3 text-sm text-gray-500">{summary}</p>
        </div>
      </section>

      {coursesQuery.loading ? (
        <TableSkeleton rows={4} />
      ) : coursesQuery.error ? (
        <ErrorState message={coursesQuery.error} onRetry={coursesQuery.refetch} />
      ) : coursesQuery.data && coursesQuery.data.length > 0 ? (
        <section className="grid gap-4 sm:grid-cols-2 xl:grid-cols-3">
          {coursesQuery.data.map((course) => (
            <CourseCard key={course.id} course={course} />
          ))}
        </section>
      ) : (
        <div className="rounded-3xl border border-dashed border-gray-200 bg-white px-6 py-16 text-center text-sm text-gray-500">
          没有找到匹配课程，试试课程代码或更短的关键词。
        </div>
      )}
    </div>
  );
}

function CourseCard({ course }: { course: Course }) {
  const meta = [
    course.code || course.id,
    Number.isFinite(course.credits) ? `${course.credits.toFixed(1)} 学分` : null,
    course.college || null,
  ].filter(Boolean) as string[];

  const capabilities = ['试卷练习', '课程评论', '教师评分标准'];

  return (
    <Link
      href={`/courses/${encodeURIComponent(course.id)}`}
      className="group overflow-hidden rounded-3xl border border-gray-200 bg-white p-5 shadow-sm transition-all hover:-translate-y-0.5 hover:border-brand-300 hover:shadow-md"
    >
      <div className="flex items-start justify-between gap-4">
        <div>
          <h2 className="text-lg font-semibold text-gray-900 transition-colors group-hover:text-brand-700">
            {course.name}
          </h2>
          <div className="mt-3 flex flex-wrap gap-2">
            {meta.map((item) => (
              <span
                key={item}
                className="rounded-full bg-gray-100 px-3 py-1 text-xs font-medium text-gray-600"
              >
                {item}
              </span>
            ))}
          </div>
        </div>
        <div className="rounded-2xl bg-brand-50 px-3 py-2 text-xs font-semibold text-brand-700">
          进入课程
        </div>
      </div>

      <p className="mt-4 text-sm leading-7 text-gray-600">
        {course.description || '进入课程页后可以查看试卷、课程评论、教师评论和评分标准。'}
      </p>

      <div className="mt-5 flex flex-wrap gap-2">
        {capabilities.map((item) => (
          <span
            key={item}
            className="rounded-full border border-gray-200 px-3 py-1 text-xs text-gray-500"
          >
            {item}
          </span>
        ))}
      </div>
    </Link>
  );
}
