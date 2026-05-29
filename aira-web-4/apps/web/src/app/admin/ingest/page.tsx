/**
 * app/admin/ingest/page.tsx
 * 管理员审核队列 — 列出 awaiting_review 任务，点击进入审核详情
 *
 * 对接：GET /api/admin/ingest?status=awaiting_review
 */

'use client';

import Link from 'next/link';
import { useEffect, useState } from 'react';
import type { IngestJob, IngestJobListData, IngestJobStatus } from '@aira/shared';
import { api } from '@/lib/api';
import { useAuth } from '@/lib/auth';

const STATUS_TABS: { label: string; value: IngestJobStatus }[] = [
  { label: '待审核', value: 'awaiting_review' },
  { label: '已发布', value: 'published' },
  { label: '已拒绝', value: 'rejected' },
  { label: '失败', value: 'failed' },
];

export default function AdminIngestPage() {
  const { user, isLoggedIn, loading: authLoading } = useAuth();
  const isAdmin = !!user?.roles?.includes('admin');

  const [status, setStatus] = useState<IngestJobStatus>('awaiting_review');
  const [data, setData] = useState<IngestJobListData | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!isAdmin) return;
    setLoading(true);
    api
      .get<IngestJobListData>(`/admin/ingest?status=${status}&page=1&size=100`)
      .then((d) => {
        setData(d);
        setError(null);
      })
      .catch((err: unknown) =>
        setError(err instanceof Error ? err.message : '加载失败'),
      )
      .finally(() => setLoading(false));
  }, [isAdmin, status]);

  if (authLoading) return <div className="py-16 text-center text-gray-500">正在加载...</div>;
  if (!isLoggedIn) {
    return (
      <div className="rounded-2xl border border-gray-200 bg-white p-8 text-center text-gray-600">
        请先 <Link href="/login" className="text-brand-600 hover:underline">登录</Link>。
      </div>
    );
  }
  if (!isAdmin) {
    return (
      <div className="rounded-2xl border border-amber-200 bg-amber-50 p-8 text-center text-amber-800">
        该页面仅限管理员访问。
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <header>
        <h1 className="text-3xl font-semibold tracking-tight text-gray-900">题库上传审核</h1>
        <p className="mt-1 text-sm text-gray-600">
          审核用户上传并经 AI 清洗后的题目 / 题解；通过则正式入题库。
        </p>
      </header>

      <div className="inline-flex rounded-full border border-gray-200 bg-white p-1 text-sm">
        {STATUS_TABS.map((t) => (
          <button
            key={t.value}
            type="button"
            onClick={() => setStatus(t.value)}
            className={`rounded-full px-4 py-1.5 transition-colors ${
              status === t.value
                ? 'bg-brand-600 text-white shadow'
                : 'text-gray-600 hover:text-brand-700'
            }`}
          >
            {t.label}
          </button>
        ))}
      </div>

      {error && (
        <div className="rounded-xl border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-700">
          {error}
        </div>
      )}

      {loading ? (
        <div className="rounded-2xl border border-gray-200 bg-white p-8 text-center text-gray-500">
          加载中...
        </div>
      ) : data && data.items.length > 0 ? (
        <ul className="space-y-3">
          {data.items.map((job) => (
            <AdminJobRow key={job.id} job={job} />
          ))}
        </ul>
      ) : (
        <div className="rounded-2xl border border-gray-200 bg-white p-8 text-center text-gray-500">
          此状态下没有任务。
        </div>
      )}
    </div>
  );
}

function AdminJobRow({ job }: { job: IngestJob }) {
  return (
    <li className="rounded-2xl border border-gray-200 bg-white p-4 shadow-sm transition-shadow hover:shadow">
      <Link
        href={`/admin/ingest/${job.id}`}
        className="flex flex-wrap items-center justify-between gap-3"
      >
        <div className="min-w-0 flex-1">
          <div className="flex flex-wrap items-center gap-2">
            <span className="rounded-full bg-brand-50 px-2 py-0.5 text-xs text-brand-700">
              {job.kind === 'question' ? '题目' : '题解'}
            </span>
            <span className="text-xs text-gray-500">上传者 #{job.user_id}</span>
          </div>
          <div className="mt-1 truncate font-medium text-gray-900">{job.filename}</div>
          <div className="mt-1 text-xs text-gray-500">
            课程：{job.course_id || job.new_course_name || '— (待 admin 指派)'}
            {' · '}
            {job.kind === 'question'
              ? `试卷：${job.paper_name || '—'}`
              : `目标卷：${job.target_paper_id ? `#${job.target_paper_id}` : '—'}`}
          </div>
        </div>
        <div className="text-right text-xs text-gray-400">
          <div>#{job.id}</div>
          <div>{new Date(job.created_at).toLocaleString()}</div>
        </div>
      </Link>
    </li>
  );
}
