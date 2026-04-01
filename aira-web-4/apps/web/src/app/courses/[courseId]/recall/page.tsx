/**
 * app/courses/[courseId]/recall/page.tsx
 * 回忆卷列表 — 某课程下的全部回忆卷
 *
 * 对接:
 *   GET  /api/recall/courses/:course_id/papers → 回忆卷列表
 *   POST /api/recall/courses/:course_id/papers → 新建回忆卷
 */

'use client';

import { useState, useCallback } from 'react';
import { useParams, useRouter } from 'next/navigation';
import Link from 'next/link';
import type { RecallPaper, Course } from '@aira/shared';
import { TableSkeleton } from '@/components/layout/Skeleton';
import { ErrorState, EmptyState } from '@/components/layout/StateDisplay';
import { useFetch } from '@/hooks/useFetch';
import { api } from '@/lib/api';
import { useAuth } from '@/lib/auth';

export default function RecallListPage() {
  const { courseId } = useParams<{ courseId: string }>();
  const router = useRouter();
  const { isLoggedIn } = useAuth();

  // 课程名
  const { data: courses } = useFetch(() => api.get<Course[]>('/courses'), []);
  const courseName = courses?.find((c) => c.id === courseId)?.name ?? courseId;

  // 回忆卷列表
  const { data: papers, loading, error, refetch } = useFetch(
    () => api.get<RecallPaper[]>(`/recall/courses/${courseId}/papers`),
    [courseId],
  );

  // 新建回忆卷
  const [showCreate, setShowCreate] = useState(false);
  const [title, setTitle] = useState('');
  const [creating, setCreating] = useState(false);

  const handleCreate = useCallback(async () => {
    if (!title.trim()) return;
    setCreating(true);
    try {
      const paper = await api.post<RecallPaper>(
        `/recall/courses/${courseId}/papers`,
        { title: title.trim() },
      );
      setTitle('');
      setShowCreate(false);
      // 跳转到新建的回忆卷
      router.push(`/recall/${paper.id}`);
    } catch (err) {
      alert(err instanceof Error ? err.message : '创建失败');
    } finally {
      setCreating(false);
    }
  }, [title, courseId, router]);

  return (
    <div>
      {/* 面包屑 */}
      <nav className="mb-4 text-sm text-gray-500">
        <Link href="/courses" className="transition-colors hover:text-brand-600">课程</Link>
        <span className="mx-2">›</span>
        <Link href={`/courses/${courseId}`} className="transition-colors hover:text-brand-600">{courseName}</Link>
        <span className="mx-2">›</span>
        <span className="font-medium text-gray-900">回忆卷</span>
      </nav>

      <div className="mb-6 flex items-center justify-between">
        <div>
          <h1 className="text-xl font-semibold text-gray-900">回忆卷</h1>
          <p className="mt-1 text-sm text-gray-500">根据考试记忆协作还原试卷</p>
        </div>
        {isLoggedIn && (
          <button onClick={() => setShowCreate(true)}
            className="rounded-md bg-brand-600 px-4 py-2 text-sm font-medium text-white hover:bg-brand-700">
            + 新建回忆卷
          </button>
        )}
      </div>

      {/* 新建弹窗 */}
      {showCreate && (
        <div className="mb-6 rounded-lg border border-brand-200 bg-brand-50 p-4">
          <label className="mb-2 block text-sm font-medium text-gray-700">回忆卷标题</label>
          <div className="flex gap-2">
            <input type="text" value={title} onChange={(e) => setTitle(e.target.value)}
              placeholder="如：2026春夏学期期末回忆卷"
              onKeyDown={(e) => { if (e.key === 'Enter') handleCreate(); }}
              className="flex-1 rounded-md border border-gray-200 px-3 py-2 text-sm outline-none
                         focus:border-brand-500 focus:ring-1 focus:ring-brand-500" />
            <button onClick={handleCreate} disabled={creating || !title.trim()}
              className="rounded-md bg-brand-600 px-4 py-2 text-sm font-medium text-white
                         hover:bg-brand-700 disabled:opacity-50">
              {creating ? '创建中...' : '创建'}
            </button>
            <button onClick={() => { setShowCreate(false); setTitle(''); }}
              className="rounded-md border border-gray-200 px-3 py-2 text-sm text-gray-500 hover:bg-gray-50">
              取消
            </button>
          </div>
        </div>
      )}

      {/* 列表 */}
      {loading ? (
        <TableSkeleton rows={3} />
      ) : error ? (
        <ErrorState message={error} onRetry={refetch} />
      ) : papers && papers.length > 0 ? (
        <div className="space-y-3">
          {papers.map((p) => (
            <Link key={p.id} href={`/recall/${p.id}`}
              className="flex items-center justify-between rounded-lg border border-gray-200 bg-white
                         px-5 py-4 transition-all hover:border-brand-300 hover:shadow-sm">
              <div>
                <h3 className="text-sm font-medium text-gray-900">{p.title}</h3>
                <p className="mt-1 text-xs text-gray-400">
                  创建于 {new Date(p.created_at).toLocaleDateString('zh-CN')}
                </p>
              </div>
              <span className="text-xs font-medium text-brand-600">进入编辑 →</span>
            </Link>
          ))}
        </div>
      ) : (
        <EmptyState title="暂无回忆卷" description="登录后可以新建一份回忆卷，协作还原试题" />
      )}
    </div>
  );
}
