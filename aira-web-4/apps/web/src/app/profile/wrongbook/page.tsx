/**
 * app/wrongbook/page.tsx
 * 错题本 — 按课程分组展示
 */

'use client';

import { useState, useCallback, useMemo } from 'react';
import Link from 'next/link';
import type { WrongBookData, WrongBookItem, Course } from '@aira/shared';
import { TableSkeleton } from '@/components/layout/Skeleton';
import { ErrorState, EmptyState } from '@/components/layout/StateDisplay';
import { useFetch } from '@/hooks/useFetch';
import { api } from '@/lib/api';

const statusOptions = [
  { value: '', label: '全部' },
  { value: 'unmastered', label: '未掌握' },
  { value: 'mastered', label: '已掌握' },
  { value: 'trash', label: '垃圾篓' },
];

export default function WrongBookPage() {
  const [status, setStatus] = useState('');
  const query = useMemo(() => (status ? `?status=${status}` : ''), [status]);

  const { data, loading, error, refetch } = useFetch(
    () => api.get<WrongBookData>(`/wrongbook${query}`),
    [query],
  );
  const { data: courses } = useFetch(() => api.get<Course[]>('/courses'), []);
  const courseMap = useMemo(() => {
    const map = new Map<string, Course>();
    (courses ?? []).forEach((c) => map.set(c.id, c));
    return map;
  }, [courses]);

  const handleUpdate = useCallback(async (problemId: number, payload: Partial<WrongBookItem>) => {
    await api.patch(`/wrongbook/${problemId}`, payload);
    refetch();
  }, [refetch]);

  const handleDelete = useCallback(async (problemId: number) => {
    await api.delete(`/wrongbook/${problemId}`);
    refetch();
  }, [refetch]);

  const handleClearTrash = useCallback(async () => {
    await api.delete('/wrongbook/trash');
    refetch();
  }, [refetch]);

  return (
    <div>
      <div className="mb-6 flex items-start justify-between">
        <div>
          <h1 className="text-xl font-semibold text-gray-900">错题本</h1>
          <p className="mt-1 text-sm text-gray-500">按课程归类你的错题，支持备注与分类</p>
        </div>
        <div className="flex items-center gap-2">
          <select
            value={status}
            onChange={(e) => setStatus(e.target.value)}
            className="rounded-md border border-gray-200 px-3 py-2 text-sm"
          >
            {statusOptions.map((opt) => (
              <option key={opt.value} value={opt.value}>{opt.label}</option>
            ))}
          </select>
          {status === 'trash' && (
            <button
              onClick={handleClearTrash}
              className="rounded-md border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-600"
            >
              清空垃圾篓
            </button>
          )}
        </div>
      </div>

      {loading ? (
        <TableSkeleton rows={3} />
      ) : error ? (
        <ErrorState message={error} onRetry={refetch} />
      ) : !data || data.courses.length === 0 ? (
        <EmptyState title="暂无错题" description="做题答错会自动加入错题本" />
      ) : (
        <div className="space-y-6">
          {data.courses.map((course) => (
            <div key={course.course_id} className="rounded-xl border border-gray-200 bg-white">
              <div className="flex items-center justify-between border-b border-gray-100 px-5 py-3">
                <div>
                  <h2 className="text-sm font-semibold text-gray-900">
                    {courseMap.get(course.course_id)?.name || course.course_name || course.course_id}
                  </h2>
                  {course.last_practice_at && (
                    <p className="mt-1 text-xs text-gray-400">
                      最近做题：{new Date(course.last_practice_at).toLocaleString('zh-CN')}
                    </p>
                  )}
                </div>
                <Link
                  href={`/courses/${encodeURIComponent(course.course_id)}`}
                  className="text-xs text-brand-600 hover:underline"
                >
                  进入课程 →
                </Link>
              </div>

              <div className="divide-y">
                {course.items.map((item) => (
                  <WrongItemRow
                    key={item.problem_id}
                    item={item}
                    onUpdate={handleUpdate}
                    onDelete={handleDelete}
                  />
                ))}
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

function WrongItemRow({
  item,
  onUpdate,
  onDelete,
}: {
  item: WrongBookItem;
  onUpdate: (problemId: number, payload: Partial<WrongBookItem>) => void;
  onDelete: (problemId: number) => void;
}) {
  const [note, setNote] = useState(item.note || '');

  return (
    <div className="px-5 py-4">
      <div className="flex items-start justify-between gap-4">
        <div className="flex-1">
          <p className="text-sm text-gray-800">第 {item.order} 题：{item.test}</p>
          <div className="mt-2 flex flex-wrap items-center gap-2 text-xs text-gray-500">
            <span>错误次数 {item.wrong_count}</span>
            <span>最后一次 {new Date(item.last_wrong_at).toLocaleString('zh-CN')}</span>
          </div>
        </div>
        <div className="flex items-center gap-2">
          <select
            value={item.status}
            onChange={(e) => onUpdate(item.problem_id, { status: e.target.value })}
            className="rounded-md border border-gray-200 px-2 py-1 text-xs"
          >
            <option value="unmastered">未掌握</option>
            <option value="mastered">已掌握</option>
            <option value="trash">垃圾篓</option>
          </select>
          <button
            onClick={() => onDelete(item.problem_id)}
            className="rounded-md px-2 py-1 text-xs text-red-500 hover:bg-red-50"
          >
            删除
          </button>
        </div>
      </div>

      <div className="mt-3">
        <textarea
          value={note}
          onChange={(e) => setNote(e.target.value)}
          onBlur={() => onUpdate(item.problem_id, { note })}
          placeholder="添加错题备注..."
          className="w-full rounded-md border border-gray-200 px-3 py-2 text-xs text-gray-700"
          rows={2}
        />
      </div>
    </div>
  );
}
