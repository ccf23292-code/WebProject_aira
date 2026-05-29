/**
 * app/upload/jobs/page.tsx
 * 我的上传记录 — 列表 + 状态徽章 + 5s 轮询（仅当存在 pending/processing 时）
 *
 * 对接：GET /api/ingest/my
 */

'use client';

import Link from 'next/link';
import { useEffect, useRef, useState } from 'react';
import type { IngestJob, IngestJobListData, IngestJobStatus } from '@aira/shared';
import { api } from '@/lib/api';
import { useAuth } from '@/lib/auth';

const STATUS_LABEL: Record<IngestJobStatus, string> = {
  pending: '排队中',
  processing: 'AI 清洗中',
  awaiting_review: '待管理员审核',
  published: '已发布',
  rejected: '已拒绝',
  failed: '清洗失败',
};

const STATUS_COLOR: Record<IngestJobStatus, string> = {
  pending: 'bg-gray-100 text-gray-700',
  processing: 'bg-amber-100 text-amber-800',
  awaiting_review: 'bg-purple-100 text-purple-800',
  published: 'bg-green-100 text-green-800',
  rejected: 'bg-red-100 text-red-700',
  failed: 'bg-red-100 text-red-700',
};

export default function MyIngestJobsPage() {
  const { isLoggedIn, loading: authLoading } = useAuth();
  const [data, setData] = useState<IngestJobListData | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const timerRef = useRef<number | null>(null);

  async function refresh() {
    try {
      const d = await api.get<IngestJobListData>('/ingest/my?page=1&size=50');
      setData(d);
      setError(null);
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : '加载失败');
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    if (!isLoggedIn) return;
    void refresh();
  }, [isLoggedIn]);

  // 仅当有进行中任务时启用轮询，避免无谓打 API
  useEffect(() => {
    if (!data) return;
    const hasRunning = data.items.some(
      (j) => j.status === 'pending' || j.status === 'processing',
    );
    if (!hasRunning) {
      if (timerRef.current) window.clearInterval(timerRef.current);
      timerRef.current = null;
      return;
    }
    if (timerRef.current) return;
    timerRef.current = window.setInterval(refresh, 5000);
    return () => {
      if (timerRef.current) window.clearInterval(timerRef.current);
      timerRef.current = null;
    };
  }, [data]);

  if (authLoading) return <div className="py-16 text-center text-gray-500">正在加载...</div>;
  if (!isLoggedIn) {
    return (
      <div className="rounded-2xl border border-gray-200 bg-white p-8 text-center text-gray-600">
        请先 <Link href="/login" className="text-brand-600 hover:underline">登录</Link>。
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <header className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-semibold tracking-tight text-gray-900">我的上传记录</h1>
          <p className="mt-1 text-sm text-gray-600">追踪每一次上传的清洗 / 审核 / 发布状态。</p>
        </div>
        <Link
          href="/upload/file"
          className="rounded-full bg-brand-600 px-4 py-2 text-sm font-medium text-white shadow hover:bg-brand-700"
        >
          + 新上传
        </Link>
      </header>

      {error && (
        <div className="rounded-xl border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-700">{error}</div>
      )}

      {loading ? (
        <div className="rounded-2xl border border-gray-200 bg-white p-8 text-center text-gray-500">
          加载中...
        </div>
      ) : data && data.items.length > 0 ? (
        <ul className="space-y-3">
          {data.items.map((job) => (
            <JobRow key={job.id} job={job} />
          ))}
        </ul>
      ) : (
        <div className="rounded-2xl border border-gray-200 bg-white p-8 text-center text-gray-500">
          还没有上传记录，去 <Link href="/upload/file" className="text-brand-600 hover:underline">上传一份</Link> 吧。
        </div>
      )}
    </div>
  );
}

function JobRow({ job }: { job: IngestJob }) {
  const color = STATUS_COLOR[job.status] ?? 'bg-gray-100 text-gray-700';
  const label = STATUS_LABEL[job.status] ?? job.status;
  return (
    <li className="rounded-2xl border border-gray-200 bg-white p-4 shadow-sm transition-shadow hover:shadow">
      <Link href={`/upload/jobs/${job.id}`} className="flex flex-wrap items-center justify-between gap-3">
        <div className="min-w-0 flex-1">
          <div className="flex items-center gap-2">
            <span className={`rounded-full px-2 py-0.5 text-xs ${color}`}>{label}</span>
            <span className="rounded-full bg-brand-50 px-2 py-0.5 text-xs text-brand-700">
              {job.kind === 'question' ? '题目' : '题解'}
            </span>
          </div>
          <div className="mt-2 truncate font-medium text-gray-900">{job.filename}</div>
          <div className="mt-1 text-xs text-gray-500">
            {job.kind === 'question'
              ? `课程: ${job.course_id || job.new_course_name || '—'}  ·  试卷: ${job.paper_name || '—'}`
              : `课程: ${job.course_id || '—'}  ·  目标卷: #${job.target_paper_id ?? '—'}`}
          </div>
          {job.error_message && (
            <div className="mt-1 truncate text-xs text-red-600">错误：{job.error_message}</div>
          )}
        </div>
        <div className="text-right text-xs text-gray-400">
          <div>#{job.id}</div>
          <div>{formatDate(job.created_at)}</div>
        </div>
      </Link>
    </li>
  );
}

function formatDate(s: string): string {
  if (!s) return '';
  const d = new Date(s);
  if (Number.isNaN(d.getTime())) return s;
  return `${d.getMonth() + 1}/${d.getDate()} ${String(d.getHours()).padStart(2, '0')}:${String(d.getMinutes()).padStart(2, '0')}`;
}
