'use client';

import type { ReactNode } from 'react';
import { useState } from 'react';
import type {
  CourseDescriptionSubmission,
  GradingStandardSubmission,
  TeacherSubmission,
} from '@aira/shared';
import { ErrorState, EmptyState } from '@/components/layout/StateDisplay';
import { useFetch } from '@/hooks/useFetch';
import { useAuth } from '@/lib/auth';
import {
  getCourseDescriptionSubmissions,
  getGradingStandardSubmissions,
  getTeacherSubmissions,
  reviewCourseDescriptionSubmission,
  reviewGradingStandardSubmission,
  reviewTeacherSubmission,
} from '@/lib/adminReview';

export default function AdminReviewsPage() {
  const { user, isLoggedIn } = useAuth();
  const isAdmin = !!user?.roles?.includes('admin');

  const descriptionsQuery = useFetch(
    () => (isAdmin ? getCourseDescriptionSubmissions('pending') : Promise.resolve([])),
    [isAdmin],
  );
  const teachersQuery = useFetch(
    () => (isAdmin ? getTeacherSubmissions('pending') : Promise.resolve([])),
    [isAdmin],
  );
  const gradingQuery = useFetch(
    () => (isAdmin ? getGradingStandardSubmissions('pending') : Promise.resolve([])),
    [isAdmin],
  );

  if (!isLoggedIn) {
    return (
      <EmptyState
        title="请先登录"
        description="管理员审核中心需要先完成登录。"
      />
    );
  }

  if (!isAdmin) {
    return (
      <ErrorState message="你当前不是管理员，无法访问审核中心。" />
    );
  }

  return (
    <div className="space-y-6">
      <section className="rounded-3xl border border-gray-200 bg-[linear-gradient(135deg,_#ffffff,_#f8fafc_65%,_#eef4ff)] p-6 shadow-sm">
        <div className="max-w-3xl">
          <div className="inline-flex rounded-full border border-brand-200 bg-white/80 px-3 py-1 text-xs font-medium text-brand-700">
            Admin Review Center
          </div>
          <h1 className="mt-4 text-3xl font-semibold tracking-tight text-gray-900">管理员审核中心</h1>
          <p className="mt-3 text-sm leading-7 text-gray-500">
            当前集中审核三类内容：课程简介、教师信息、评分标准。所有列表默认只显示待审核项。
          </p>
        </div>
      </section>

      <ReviewSection<CourseDescriptionSubmission>
        title="课程简介审核"
        description="审核通过后，会直接覆盖课程广场与课程详情中的公开简介。"
        query={descriptionsQuery}
        renderBody={(item) => (
          <>
            <MetaRow label="课程" value={item.course_id} />
            <p className="mt-3 whitespace-pre-wrap text-sm leading-7 text-gray-700">{item.content}</p>
          </>
        )}
        onApprove={async (item) => {
          await reviewCourseDescriptionSubmission(item.id, { action: 'approve' });
          descriptionsQuery.refetch();
        }}
        onReject={async (item) => {
          await reviewCourseDescriptionSubmission(item.id, { action: 'reject' });
          descriptionsQuery.refetch();
        }}
      />

      <ReviewSection<TeacherSubmission>
        title="教师信息审核"
        description="审核通过后，教师会进入课程教师目录，供课程评论和评分标准使用。"
        query={teachersQuery}
        renderBody={(item) => (
          <>
            <MetaRow label="课程" value={item.course_id} />
            <MetaRow label="教师姓名" value={item.name} />
            {item.title ? <MetaRow label="备注" value={item.title} /> : null}
          </>
        )}
        onApprove={async (item) => {
          await reviewTeacherSubmission(item.id, { action: 'approve' });
          teachersQuery.refetch();
        }}
        onReject={async (item) => {
          await reviewTeacherSubmission(item.id, { action: 'reject' });
          teachersQuery.refetch();
        }}
      />

      <ReviewSection<GradingStandardSubmission>
        title="评分标准审核"
        description="审核通过后，评分标准会挂到对应教师下公开展示。"
        query={gradingQuery}
        renderBody={(item) => (
          <>
            <MetaRow label="课程" value={item.course_id} />
            <MetaRow label="教师" value={item.teacher_id} />
            {item.description ? <p className="mt-3 text-sm leading-7 text-gray-700">{item.description}</p> : null}
            {item.standard ? <p className="mt-3 whitespace-pre-wrap text-sm leading-7 text-gray-700">{item.standard}</p> : null}
          </>
        )}
        onApprove={async (item) => {
          await reviewGradingStandardSubmission(item.id, { action: 'approve' });
          gradingQuery.refetch();
        }}
        onReject={async (item) => {
          await reviewGradingStandardSubmission(item.id, { action: 'reject' });
          gradingQuery.refetch();
        }}
      />
    </div>
  );
}

function ReviewSection<T extends { id: number; created_at: string }>(props: {
  title: string;
  description: string;
  query: {
    data: T[] | null;
    loading: boolean;
    error: string | null;
    refetch: () => void;
  };
  renderBody: (item: T) => ReactNode;
  onApprove: (item: T) => Promise<void>;
  onReject: (item: T) => Promise<void>;
}) {
  const { title, description, query, renderBody, onApprove, onReject } = props;

  return (
    <section className="rounded-3xl border border-gray-200 bg-white p-6 shadow-sm">
      <div className="mb-5">
        <h2 className="text-xl font-semibold text-gray-900">{title}</h2>
        <p className="mt-1 text-sm leading-6 text-gray-500">{description}</p>
      </div>

      {query.error ? (
        <ErrorState message={query.error} onRetry={query.refetch} />
      ) : query.loading ? (
        <div className="text-sm text-gray-500">加载中...</div>
      ) : query.data && query.data.length > 0 ? (
        <div className="space-y-4">
          {query.data.map((item) => (
            <ReviewCard
              key={item.id}
              createdAt={item.created_at}
              onApprove={() => onApprove(item)}
              onReject={() => onReject(item)}
            >
              {renderBody(item)}
            </ReviewCard>
          ))}
        </div>
      ) : (
        <EmptyState title="当前没有待审核项" description="这一类提交已经处理完了。" />
      )}
    </section>
  );
}

function ReviewCard(props: {
  createdAt: string;
  onApprove: () => Promise<void>;
  onReject: () => Promise<void>;
  children: ReactNode;
}) {
  const { createdAt, onApprove, onReject, children } = props;
  const [loading, setLoading] = useState<'approve' | 'reject' | ''>('');
  const [error, setError] = useState('');

  const handleAction = async (action: 'approve' | 'reject') => {
    setLoading(action);
    setError('');
    try {
      if (action === 'approve') {
        await onApprove();
      } else {
        await onReject();
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : '操作失败，请稍后重试。');
    } finally {
      setLoading('');
    }
  };

  return (
    <div className="rounded-2xl border border-gray-200 bg-gray-50 p-4">
      <div className="flex items-center justify-between gap-3">
        <div className="text-xs text-gray-500">
          提交时间：{new Intl.DateTimeFormat('zh-CN', { dateStyle: 'medium', timeStyle: 'short' }).format(new Date(createdAt))}
        </div>
        <div className="flex gap-2">
          <button
            type="button"
            onClick={() => handleAction('reject')}
            disabled={loading !== ''}
            className="rounded-xl border border-rose-200 bg-white px-3 py-1.5 text-sm text-rose-700 transition-colors hover:bg-rose-50 disabled:cursor-not-allowed disabled:opacity-60"
          >
            {loading === 'reject' ? '驳回中...' : '驳回'}
          </button>
          <button
            type="button"
            onClick={() => handleAction('approve')}
            disabled={loading !== ''}
            className="rounded-xl bg-gray-900 px-3 py-1.5 text-sm text-white transition-colors hover:bg-gray-800 disabled:cursor-not-allowed disabled:opacity-60"
          >
            {loading === 'approve' ? '通过中...' : '通过'}
          </button>
        </div>
      </div>
      <div className="mt-3">{children}</div>
      {error ? <p className="mt-3 text-sm text-red-600">{error}</p> : null}
    </div>
  );
}

function MetaRow({ label, value }: { label: string; value: string }) {
  return (
    <div className="mt-2 text-sm text-gray-600">
      <span className="font-medium text-gray-800">{label}：</span>
      {value}
    </div>
  );
}
