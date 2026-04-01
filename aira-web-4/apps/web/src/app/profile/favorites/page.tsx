/**
 * app/favorites/page.tsx
 * 收藏列表 — 用户收藏的题目
 *
 * 对接:
 *   GET    /api/favorites              → 分页获取收藏列表
 *   DELETE /api/favorites/{problem_id} → 取消收藏
 */

'use client';

import { useState, useCallback } from 'react';
import Link from 'next/link';
import type { FavoriteCourseGroup, FavoriteItem, FavoriteListData } from '@aira/shared';
import { TableSkeleton } from '@/components/layout/Skeleton';
import { ErrorState, EmptyState } from '@/components/layout/StateDisplay';
import { useFetch } from '@/hooks/useFetch';
import { api } from '@/lib/api';

export default function FavoritesPage() {
  const { data: favData, loading, error, refetch } = useFetch(
    () => api.get<FavoriteListData>('/favorites'),
    [],
  );

  const groups = favData?.groups ?? [];

  // 本地乐观移除（删除请求发出后立即从 UI 隐藏）
  const [removed, setRemoved] = useState<Set<number>>(new Set());

  /** 取消收藏 — DELETE /api/favorites/{problem_id} */
  const handleRemove = useCallback(async (problemId: number) => {
    setRemoved((prev) => new Set(prev).add(problemId));
    try {
      await api.delete(`/favorites/${problemId}`);
    } catch {
      // 失败时回滚
      setRemoved((prev) => {
        const next = new Set(prev);
        next.delete(problemId);
        return next;
      });
    }
  }, []);

  const visibleGroups = groups
    .map((group) => ({
      ...group,
      items: group.items.filter((item) => !removed.has(item.problem_id)),
    }))
    .filter((group) => group.items.length > 0);

  const visibleCount = visibleGroups.reduce((sum, group) => sum + group.items.length, 0);

  return (
    <div>
      <div className="mb-6">
        <h1 className="text-xl font-semibold text-gray-900">我的收藏</h1>
        <p className="mt-1 text-sm text-gray-500">
          {visibleCount > 0 ? `共 ${visibleCount} 道题目` : ''}
        </p>
      </div>

      {loading ? (
        <TableSkeleton rows={3} />
      ) : error ? (
        <ErrorState message={error} onRetry={refetch} />
      ) : visibleGroups.length === 0 ? (
        <EmptyState title="暂无收藏" description="在做题页面点击 ☆ 收藏感兴趣的题目" />
      ) : (
        <div className="space-y-6">
          {visibleGroups.map((group) => (
            <FavoriteCourseSection key={group.course_id || group.course_name} group={group} onRemove={handleRemove} />
          ))}
        </div>
      )}
    </div>
  );
}

function FavoriteCourseSection({
  group,
  onRemove,
}: {
  group: FavoriteCourseGroup;
  onRemove: (id: number) => void;
}) {
  return (
    <section className="rounded-xl border border-gray-200 bg-white">
      <div className="flex items-center justify-between border-b border-gray-100 px-5 py-3">
        <div>
          <h2 className="text-sm font-semibold text-gray-900">{group.course_name || group.course_id}</h2>
          <p className="mt-1 text-xs text-gray-400">{group.items.length} 道收藏题目</p>
        </div>
        {group.course_id && (
          <Link
            href={`/courses/${encodeURIComponent(group.course_id)}`}
            className="text-xs font-medium text-brand-600 hover:underline"
          >
            进入课程
          </Link>
        )}
      </div>
      <div className="divide-y divide-gray-100">
        {group.items.map((item) => (
          <FavoriteRow key={item.favorite_id} item={item} onRemove={onRemove} />
        ))}
      </div>
    </section>
  );
}

function FavoriteRow({
  item,
  onRemove,
}: {
  item: FavoriteItem;
  onRemove: (id: number) => void;
}) {
  const date = new Date(item.added_at).toLocaleDateString('zh-CN', {
    month: 'short', day: 'numeric',
  });

  return (
    <div className="flex items-start justify-between px-5 py-4">
      <div className="flex-1">
        <p className="text-sm text-gray-800 leading-relaxed">{item.problem_details.test}</p>
        <div className="mt-2 flex items-center gap-3 text-xs text-gray-400">
          <span>{item.problem_details.testpaper_name}</span>
          <span>第 {item.problem_details.order} 题</span>
          <span>收藏于 {date}</span>
        </div>
      </div>
      <button onClick={() => onRemove(item.problem_id)}
        className="ml-4 shrink-0 rounded-md px-2 py-1 text-xs text-red-500 transition-colors
                   hover:bg-red-50">
        取消收藏
      </button>
    </div>
  );
}
