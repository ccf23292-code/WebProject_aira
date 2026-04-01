/**
 * app/records/page.tsx
 * 做题记录
 */

'use client';

import type { AnswerRecordListData } from '@aira/shared';
import { TableSkeleton } from '@/components/layout/Skeleton';
import { ErrorState, EmptyState } from '@/components/layout/StateDisplay';
import { useFetch } from '@/hooks/useFetch';
import { api } from '@/lib/api';

export default function RecordsPage() {
  const { data, loading, error, refetch } = useFetch(
    () => api.get<AnswerRecordListData>('/answers'),
    [],
  );

  const items = data?.items ?? [];

  return (
    <div>
      <div className="mb-6">
        <h1 className="text-xl font-semibold text-gray-900">做题记录</h1>
        <p className="mt-1 text-sm text-gray-500">按时间倒序展示你的作答记录</p>
      </div>

      {loading ? (
        <TableSkeleton rows={3} />
      ) : error ? (
        <ErrorState message={error} onRetry={refetch} />
      ) : items.length === 0 ? (
        <EmptyState title="暂无记录" description="开始做题后会在这里留下记录" />
      ) : (
        <div className="space-y-3">
          {items.map((record) => (
            <div key={record.id} className="rounded-lg border border-gray-200 bg-white px-5 py-4">
              <div className="flex items-center justify-between">
                <div>
                  <p className="text-sm text-gray-800">题目 #{record.problem_id}</p>
                  <div className="mt-2 flex flex-wrap items-center gap-2 text-xs text-gray-400">
                    <span>课程 {record.course_id}</span>
                    <span>试卷 {record.paper_id}</span>
                    <span>模式 {record.mode || 'practice'}</span>
                  </div>
                </div>
                <span className={`text-xs font-medium ${record.is_correct ? 'text-green-600' : 'text-red-600'}`}>
                  {record.is_correct ? '正确' : '错误'}
                </span>
              </div>
              <p className="mt-2 text-xs text-gray-400">
                作答时间 {new Date(record.answered_at).toLocaleString('zh-CN')}
              </p>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
