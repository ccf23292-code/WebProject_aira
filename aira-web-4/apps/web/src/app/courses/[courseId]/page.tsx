/**
 * app/courses/[courseId]/page.tsx
 * 课程详情 → 该课程下的试卷列表
 * 对接: GET /api/courses/{course_id}/papers → Paper[]
 */

'use client';

import { useParams } from 'next/navigation';
import Link from 'next/link';
import type { Paper, Course } from '@aira/shared';
import { TableSkeleton } from '@/components/layout/Skeleton';
import { ErrorState, EmptyState } from '@/components/layout/StateDisplay';
import { useFetch } from '@/hooks/useFetch';
import { api } from '@/lib/api';

export default function CourseDetailPage() {
  const { courseId } = useParams<{ courseId: string }>();

  // 获取课程名称（从课程列表中查找）
  const { data: courses } = useFetch(
    () => api.get<Course[]>('/courses'),
    [],
  );
  const courseName = courses?.find((c) => c.id === courseId)?.name ?? courseId;

  // 获取该课程的试卷列表
  const { data, loading, error, refetch } = useFetch(
    () => api.get<Paper[]>(`/courses/${courseId}/papers`),
    [courseId],
  );

  return (
    <div>
      {/* 面包屑 */}
      <nav className="mb-4 text-sm text-gray-500">
        <Link href="/courses" className="transition-colors hover:text-brand-600">课程</Link>
        <span className="mx-2">›</span>
        <span className="font-medium text-gray-900">{courseName}</span>
      </nav>

      <div className="mb-6 flex items-center justify-between">
        <div>
          <h1 className="text-xl font-semibold text-gray-900">{courseName}</h1>
          <p className="mt-1 text-sm text-gray-500">
            {data ? `共 ${data.length} 份试卷` : '加载中...'}
          </p>
        </div>
        <Link href={`/courses/${courseId}/recall`}
          className="rounded-md border border-orange-200 bg-orange-50 px-4 py-2 text-sm
                     font-medium text-orange-700 transition-colors hover:bg-orange-100">
          📝 回忆卷
        </Link>
      </div>

      {loading ? (
        <TableSkeleton rows={3} />
      ) : error ? (
        <ErrorState message={error} onRetry={refetch} />
      ) : data && data.length > 0 ? (
        <div className="space-y-3">
          {data.map((paper) => (
            <PaperRow key={paper.id} paper={paper} />
          ))}
        </div>
      ) : (
        <EmptyState title="该课程暂无试卷" description="试卷由管理员上传后显示" />
      )}
    </div>
  );
}

function PaperRow({ paper }: { paper: Paper }) {
  const date = new Date(paper.created_at).toLocaleDateString('zh-CN', {
    year: 'numeric', month: 'long', day: 'numeric',
  });

  return (
    <Link href={`/papers/${paper.id}`}
      className="flex items-center justify-between rounded-lg border border-gray-200 bg-white
                 px-5 py-4 transition-all hover:border-brand-300 hover:shadow-sm">
      <div>
        <h3 className="text-sm font-medium text-gray-900">{paper.name}</h3>
        <p className="mt-1 text-xs text-gray-400">上传于 {date}</p>
      </div>
      <span className="text-xs font-medium text-brand-600">开始做题 →</span>
    </Link>
  );
}
