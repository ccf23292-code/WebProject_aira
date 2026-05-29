'use client';

import Link from 'next/link';
import { useEffect, useMemo, useState, type ReactNode } from 'react';
import type {
  CourseComment,
  TeacherComment,
  GradingStandard,
  TeacherDirectoryEntry,
} from '@aira/shared';
import { EmptyState, ErrorState } from '@/components/layout/StateDisplay';
import { useAuth } from '@/lib/auth';
import { useFetch } from '@/hooks/useFetch';
import {
  addTeacherDirectoryEntry,
  addCourseComment,
  addGradingStandard,
  addTeacherComment,
  getCourseComments,
  getGradingStandards,
  getTeacherComments,
  getTeacherDirectory,
} from '@/lib/courseCommunity';

interface CourseCommunityPanelProps {
  courseId: string;
  courseName: string;
}

export function CourseCommunityPanel({
  courseId,
  courseName,
}: CourseCommunityPanelProps) {
  const [teacherDirectoryVersion, setTeacherDirectoryVersion] = useState(0);
  const [courseCommentVersion, setCourseCommentVersion] = useState(0);
  const [teacherCommentVersion, setTeacherCommentVersion] = useState(0);
  const [gradingStandardVersion, setGradingStandardVersion] = useState(0);
  const [selectedTeacherId, setSelectedTeacherId] = useState('');
  const [newTeacherName, setNewTeacherName] = useState('');
  const [newTeacherTitle, setNewTeacherTitle] = useState('');
  const [teacherFormError, setTeacherFormError] = useState('');
  const [teacherFormSuccess, setTeacherFormSuccess] = useState('');
  const [gradingSuccess, setGradingSuccess] = useState('');
  const teachersQuery = useFetch(
    () => getTeacherDirectory(courseId),
    [courseId, teacherDirectoryVersion],
  );
  const teacherDirectory = teachersQuery.data ?? [];

  useEffect(() => {
    if (!teacherDirectory.length) {
      setSelectedTeacherId('');
      return;
    }

    const currentTeacherExists = teacherDirectory.some((teacher) => teacher.id === selectedTeacherId);
    if (!currentTeacherExists) {
      setSelectedTeacherId(teacherDirectory[0]?.id ?? '');
    }
  }, [teacherDirectory, selectedTeacherId]);

  const selectedTeacher = useMemo(
    () => teacherDirectory.find((teacher) => teacher.id === selectedTeacherId) ?? null,
    [selectedTeacherId, teacherDirectory],
  );

  const courseCommentsQuery = useFetch(
    () => getCourseComments(courseId),
    [courseId, courseCommentVersion],
  );

  const teacherCommentsQuery = useFetch(
    () => (
      selectedTeacherId
        ? getTeacherComments(courseId, selectedTeacherId)
        : Promise.resolve([] as TeacherComment[])
    ),
    [courseId, selectedTeacherId, teacherCommentVersion],
  );

  const gradingStandardsQuery = useFetch(
    () => (
      selectedTeacherId
        ? getGradingStandards(courseId, selectedTeacherId)
        : Promise.resolve([] as GradingStandard[])
    ),
    [courseId, selectedTeacherId, gradingStandardVersion],
  );

  const handleTeacherDirectorySubmit = async (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    const teacherName = newTeacherName.trim();

    if (!teacherName) {
      setTeacherFormError('请先填写教师姓名，再保存到教师列表。');
      return;
    }

    try {
      await addTeacherDirectoryEntry(courseId, {
        name: teacherName,
        title: newTeacherTitle.trim() || undefined,
      });
      setTeacherFormError('');
      setTeacherFormSuccess('教师信息已提交管理员审核，审核通过后会出现在教师目录。');
      setNewTeacherName('');
      setNewTeacherTitle('');
    } catch (error) {
      setTeacherFormError(error instanceof Error ? error.message : '保存教师失败，请稍后重试。');
      setTeacherFormSuccess('');
    }
  };

  return (
    <section className="space-y-6">
      <div className="grid gap-4 md:grid-cols-3">
        <SummaryCard
          label="课程试卷"
          value={courseName}
          hint="试卷区保留原有练习入口。"
        />
        <SummaryCard
          label="课程评论"
          value={`${courseCommentsQuery.data?.length ?? 0} 条`}
          hint="对接 /courses/{course_id}/comments"
        />
        <SummaryCard
          label="教师评分标准"
          value={selectedTeacher ? `${gradingStandardsQuery.data?.length ?? 0} 条` : '待选择教师'}
          hint="支持按教师查看评论和给分规则。"
        />
      </div>

      <div className="grid gap-6 xl:grid-cols-[0.95fr,1.05fr]">
        <SectionCard
            title="课程讨论"
            subtitle="补充了课程评论的读取与发布，方便沉淀备考经验。"
          >
          <CommentComposer
            title="发布课程评论"
            placeholder="写下这门课的复习建议、试卷特点或踩坑提醒。"
            actionLabel="发布课程评论"
            onSubmit={async (comment) => {
              await addCourseComment(courseId, { comment });
              setCourseCommentVersion((value) => value + 1);
            }}
          />

          {courseCommentsQuery.error ? (
            <ErrorState
              message={courseCommentsQuery.error}
              onRetry={courseCommentsQuery.refetch}
            />
          ) : (
            <CommentList
              loading={courseCommentsQuery.loading}
              items={courseCommentsQuery.data ?? []}
              emptyTitle="这门课还没有评论"
              emptyDescription="你可以先补充这门课的难点、题型分布或复习顺序。"
            />
          )}
        </SectionCard>

        <div className="space-y-6">
          <SectionCard
            title="教师讨论区"
            subtitle="教师目录只展示已通过审核的信息。普通用户新增教师后，会先进入管理员审核。"
          >
            <div className="space-y-4">
              {teachersQuery.error ? (
                <ErrorState
                  message={teachersQuery.error}
                  onRetry={teachersQuery.refetch}
                />
              ) : null}

              <div className="flex flex-wrap gap-2">
                {teacherDirectory.length > 0 ? (
                  teacherDirectory.map((teacher) => {
                    const active = teacher.id === selectedTeacherId;
                    return (
                      <button
                        key={teacher.id}
                        type="button"
                        onClick={() => setSelectedTeacherId(teacher.id)}
                        className={`rounded-full border px-3 py-1.5 text-sm transition-colors ${
                          active
                            ? 'border-brand-500 bg-brand-50 text-brand-700'
                            : 'border-gray-200 bg-white text-gray-600 hover:border-brand-200'
                        }`}
                      >
                        {teacher.name}
                      </button>
                    );
                  })
                ) : (
                  <p className="text-sm text-gray-500">当前课程还没有教师目录，先添加一个即可开始记录。</p>
                )}
              </div>

              <form className="grid gap-3 rounded-2xl border border-dashed border-gray-200 bg-gray-50 p-4 md:grid-cols-2" onSubmit={handleTeacherDirectorySubmit}>
                <input
                  value={newTeacherName}
                  onChange={(event) => setNewTeacherName(event.target.value)}
                  placeholder="教师姓名"
                  className="rounded-xl border border-gray-200 bg-white px-3 py-2 text-sm outline-none focus:border-brand-500 focus:ring-1 focus:ring-brand-500"
                />
                <div className="flex gap-2">
                  <input
                    value={newTeacherTitle}
                    onChange={(event) => setNewTeacherTitle(event.target.value)}
                    placeholder="方向/备注"
                    className="min-w-0 flex-1 rounded-xl border border-gray-200 bg-white px-3 py-2 text-sm outline-none focus:border-brand-500 focus:ring-1 focus:ring-brand-500"
                  />
                  <button
                    type="submit"
                    className="rounded-xl bg-gray-900 px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-gray-800"
                  >
                    保存教师
                  </button>
                </div>
              </form>

              {teacherFormError ? <p className="text-sm text-red-600">{teacherFormError}</p> : null}
              {teacherFormSuccess ? <p className="text-sm text-emerald-700">{teacherFormSuccess}</p> : null}

              {selectedTeacher ? (
                <div className="rounded-2xl border border-gray-200 bg-white p-4">
                  <div className="text-sm font-semibold text-gray-900">{selectedTeacher.name}</div>
                  <p className="mt-1 text-sm text-gray-500">
                    教师 ID: {selectedTeacher.id}
                    {selectedTeacher.title ? ` · ${selectedTeacher.title}` : ''}
                  </p>
                </div>
              ) : (
                <EmptyState
                  title="先选择一位教师"
                  description="选中教师后就可以查看评论和评分标准。"
                />
              )}
            </div>
          </SectionCard>

          <SectionCard
            title="教师评论"
            subtitle={selectedTeacher ? `当前查看 ${selectedTeacher.name} 的评论。` : '选择教师后会在这里显示评论。'}
          >
            {selectedTeacher ? (
              <>
                <CommentComposer
                  title="发布教师评论"
                  placeholder="补充这位老师的授课节奏、考试风格或复习建议。"
                  actionLabel="发布教师评论"
                  onSubmit={async (comment) => {
                    await addTeacherComment(courseId, selectedTeacher.id, { comment });
                    setTeacherCommentVersion((value) => value + 1);
                  }}
                />

                {teacherCommentsQuery.error ? (
                  <ErrorState
                    message={teacherCommentsQuery.error}
                    onRetry={teacherCommentsQuery.refetch}
                  />
                ) : (
                  <CommentList
                    loading={teacherCommentsQuery.loading}
                    items={teacherCommentsQuery.data ?? []}
                    emptyTitle="这位教师还没有评论"
                    emptyDescription="可以先记录考试风格、平时作业强度或推荐资料。"
                  />
                )}
              </>
            ) : (
              <EmptyState
                title="尚未选择教师"
                description="先从上面的教师目录中选择一位教师。"
              />
            )}
          </SectionCard>

          <SectionCard
            title="评分标准"
            subtitle={selectedTeacher ? `当前查看 ${selectedTeacher.name} 的给分信息。新提交内容需要管理员审核后才会公开。` : '选择教师后会在这里显示评分标准。'}
          >
            {selectedTeacher ? (
              <>
                <GradingStandardComposer
                  teacherName={selectedTeacher.name}
                  onSubmit={async (payload) => {
                    await addGradingStandard(courseId, selectedTeacher.id, payload);
                    setGradingSuccess('评分标准已提交管理员审核，审核通过后会公开展示。');
                  }}
                />
                {gradingSuccess ? (
                  <p className="text-sm text-emerald-700">{gradingSuccess}</p>
                ) : null}

                {gradingStandardsQuery.error ? (
                  <ErrorState
                    message={gradingStandardsQuery.error}
                    onRetry={gradingStandardsQuery.refetch}
                  />
                ) : (
                  <GradingStandardList
                    loading={gradingStandardsQuery.loading}
                    items={gradingStandardsQuery.data ?? []}
                  />
                )}
              </>
            ) : (
              <EmptyState
                title="尚未选择教师"
                description="选择教师后可以查看这位老师的平时分、期中期末占比或图片版评分规则。"
              />
            )}
          </SectionCard>
        </div>
      </div>
    </section>
  );
}

function SummaryCard({
  label,
  value,
  hint,
}: {
  label: string;
  value: string;
  hint: string;
}) {
  return (
    <div className="rounded-2xl border border-gray-200 bg-white p-5">
      <div className="text-sm font-medium text-gray-500">{label}</div>
      <div className="mt-2 text-lg font-semibold text-gray-900">{value}</div>
      <p className="mt-2 text-sm leading-6 text-gray-500">{hint}</p>
    </div>
  );
}

function SectionCard({
  title,
  subtitle,
  children,
}: {
  title: string;
  subtitle: string;
  children: ReactNode;
}) {
  return (
    <section className="rounded-3xl border border-gray-200 bg-white p-6">
      <div className="mb-5">
        <h2 className="text-lg font-semibold text-gray-900">{title}</h2>
        <p className="mt-1 text-sm leading-6 text-gray-500">{subtitle}</p>
      </div>
      <div className="space-y-5">{children}</div>
    </section>
  );
}

function CommentComposer({
  title,
  placeholder,
  actionLabel,
  onSubmit,
}: {
  title: string;
  placeholder: string;
  actionLabel: string;
  onSubmit: (comment: string) => Promise<void>;
}) {
  const [value, setValue] = useState('');
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState('');

  const handleSubmit = async (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    const trimmed = value.trim();

    if (!trimmed) {
      setError('内容不能为空。');
      return;
    }

    setSubmitting(true);
    setError('');

    try {
      await onSubmit(trimmed);
      setValue('');
    } catch (submissionError) {
      setError(submissionError instanceof Error ? submissionError.message : '提交失败，请稍后重试。');
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <form className="space-y-3 rounded-2xl border border-gray-200 bg-gray-50 p-4" onSubmit={handleSubmit}>
      <div className="text-sm font-medium text-gray-800">{title}</div>
      <textarea
        value={value}
        onChange={(event) => setValue(event.target.value)}
        placeholder={placeholder}
        rows={4}
        className="w-full rounded-2xl border border-gray-200 bg-white px-4 py-3 text-sm leading-6 outline-none focus:border-brand-500 focus:ring-1 focus:ring-brand-500"
      />
      {error ? <p className="text-sm text-red-600">{error}</p> : null}
      <div className="flex justify-end">
        <button
          type="submit"
          disabled={submitting}
          className="rounded-xl bg-brand-600 px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-brand-700 disabled:cursor-not-allowed disabled:bg-brand-300"
        >
          {submitting ? '提交中...' : actionLabel}
        </button>
      </div>
    </form>
  );
}

function CommentList({
  loading,
  items,
  emptyTitle,
  emptyDescription,
}: {
  loading: boolean;
  items: Array<CourseComment | TeacherComment>;
  emptyTitle: string;
  emptyDescription: string;
}) {
  const { user, isLoggedIn } = useAuth();

  if (loading) {
    return (
      <div className="space-y-3">
        {Array.from({ length: 2 }).map((_, index) => (
          <div key={index} className="rounded-2xl border border-gray-200 p-4">
            <div className="h-4 w-32 animate-pulse rounded bg-gray-200" />
            <div className="mt-3 h-4 w-full animate-pulse rounded bg-gray-100" />
            <div className="mt-2 h-4 w-4/5 animate-pulse rounded bg-gray-100" />
          </div>
        ))}
      </div>
    );
  }

  if (!items.length) {
    return <EmptyState title={emptyTitle} description={emptyDescription} />;
  }

  return (
    <div className="space-y-3">
      {items.map((item) => {
        // 仅对「他人」的评论显示「私信 TA」（自己的评论不显示）
        const canMessage =
          isLoggedIn &&
          item.user_id != null &&
          String(item.user_id) !== String(user?.userId ?? '');
        return (
          <article key={String(item.id)} className="rounded-2xl border border-gray-200 p-4">
            <div className="flex flex-wrap items-center justify-between gap-2">
              <div className="flex items-center gap-2">
                {item.avatar_url ? (
                  // eslint-disable-next-line @next/next/no-img-element
                  <img
                    src={item.avatar_url}
                    alt={item.user_name ?? 'avatar'}
                    className="h-8 w-8 rounded-full object-cover"
                  />
                ) : (
                  <div className="flex h-8 w-8 items-center justify-center rounded-full bg-brand-600 text-xs font-semibold text-white">
                    {item.user_name?.charAt(0)?.toUpperCase() ?? 'U'}
                  </div>
                )}
                <div className="text-sm font-semibold text-gray-900">
                  {item.user_name || '匿名同学'}
                </div>
              </div>
              <div className="flex items-center gap-3">
                {canMessage ? (
                  <Link
                    href={{
                      pathname: '/messages',
                      query: {
                        to: String(item.user_id),
                        name: item.user_name || '',
                        ...(item.avatar_url ? { avatar: item.avatar_url } : {}),
                      },
                    }}
                    className="rounded-full border border-brand-200 px-2.5 py-1 text-xs font-medium text-brand-700 transition-colors hover:bg-brand-50"
                  >
                    私信 TA
                  </Link>
                ) : null}
                <div className="text-xs text-gray-400">{formatDate(item.updated_at ?? item.created_at)}</div>
              </div>
            </div>
            <p className="mt-3 whitespace-pre-wrap text-sm leading-7 text-gray-600">{item.comment}</p>
          </article>
        );
      })}
    </div>
  );
}

function GradingStandardComposer({
  teacherName,
  onSubmit,
}: {
  teacherName: string;
  onSubmit: (payload: {
    description?: string;
    standard?: string;
    standard_img?: string;
  }) => Promise<void>;
}) {
  const [description, setDescription] = useState('');
  const [standard, setStandard] = useState('');
  const [standardImage, setStandardImage] = useState('');
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState('');

  const handleSubmit = async (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    const payload = {
      description: description.trim() || undefined,
      standard: standard.trim() || undefined,
      standard_img: standardImage.trim() || undefined,
    };

    if (!payload.description && !payload.standard && !payload.standard_img) {
      setError('评分标准、说明和图片至少填写一项。');
      return;
    }

    setSubmitting(true);
    setError('');

    try {
      await onSubmit(payload);
      setDescription('');
      setStandard('');
      setStandardImage('');
    } catch (submissionError) {
      setError(submissionError instanceof Error ? submissionError.message : '提交失败，请稍后重试。');
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <form className="space-y-3 rounded-2xl border border-gray-200 bg-gray-50 p-4" onSubmit={handleSubmit}>
      <div className="text-sm font-medium text-gray-800">补充 {teacherName} 的评分规则</div>
      <textarea
        value={description}
        onChange={(event) => setDescription(event.target.value)}
        placeholder="补充整体说明，例如课堂参与、作业频率或补分规则。"
        rows={3}
        className="w-full rounded-2xl border border-gray-200 bg-white px-4 py-3 text-sm leading-6 outline-none focus:border-brand-500 focus:ring-1 focus:ring-brand-500"
      />
      <textarea
        value={standard}
        onChange={(event) => setStandard(event.target.value)}
        placeholder="填写结构化评分标准，例如 平时 40%，期中 20%，期末 40%。"
        rows={3}
        className="w-full rounded-2xl border border-gray-200 bg-white px-4 py-3 text-sm leading-6 outline-none focus:border-brand-500 focus:ring-1 focus:ring-brand-500"
      />
      <input
        value={standardImage}
        onChange={(event) => setStandardImage(event.target.value)}
        placeholder="评分标准图片 URL，可选"
        className="w-full rounded-2xl border border-gray-200 bg-white px-4 py-3 text-sm outline-none focus:border-brand-500 focus:ring-1 focus:ring-brand-500"
      />
      {error ? <p className="text-sm text-red-600">{error}</p> : null}
      <div className="flex justify-end">
        <button
          type="submit"
          disabled={submitting}
          className="rounded-xl bg-brand-600 px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-brand-700 disabled:cursor-not-allowed disabled:bg-brand-300"
        >
          {submitting ? '提交中...' : '发布评分标准'}
        </button>
      </div>
    </form>
  );
}

function GradingStandardList({
  loading,
  items,
}: {
  loading: boolean;
  items: GradingStandard[];
}) {
  if (loading) {
    return (
      <div className="space-y-3">
        {Array.from({ length: 2 }).map((_, index) => (
          <div key={index} className="rounded-2xl border border-gray-200 p-4">
            <div className="h-4 w-40 animate-pulse rounded bg-gray-200" />
            <div className="mt-3 h-4 w-full animate-pulse rounded bg-gray-100" />
            <div className="mt-2 h-20 w-full animate-pulse rounded bg-gray-100" />
          </div>
        ))}
      </div>
    );
  }

  if (!items.length) {
    return (
      <EmptyState
        title="还没有评分标准"
        description="可以先补充平时分、考试占比或截图版评分规则。"
      />
    );
  }

  return (
    <div className="space-y-3">
      {items.map((item) => (
        <article key={String(item.id)} className="rounded-2xl border border-gray-200 p-4">
          <div className="flex flex-wrap items-center justify-between gap-2">
            <div className="text-sm font-semibold text-gray-900">
              {item.teacher_name || item.teacher_id}
            </div>
            <div className="text-xs text-gray-400">{formatDate(item.updated_at ?? item.created_at)}</div>
          </div>
          {item.description ? (
            <p className="mt-3 whitespace-pre-wrap text-sm leading-7 text-gray-600">{item.description}</p>
          ) : null}
          {item.standard ? (
            <div className="mt-3 rounded-2xl bg-gray-50 p-4 text-sm leading-7 text-gray-700">
              {item.standard}
            </div>
          ) : null}
          {item.standard_img ? (
            <div className="mt-4 space-y-3">
              <a
                href={item.standard_img}
                target="_blank"
                rel="noreferrer"
                className="text-sm font-medium text-brand-600 hover:text-brand-700"
              >
                打开评分标准图片
              </a>
              <img
                src={item.standard_img}
                alt="评分标准图片"
                className="max-h-72 w-full rounded-2xl border border-gray-200 object-cover"
              />
            </div>
          ) : null}
        </article>
      ))}
    </div>
  );
}

function formatDate(value?: string) {
  if (!value) return '刚刚';

  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return '刚刚';

  return new Intl.DateTimeFormat('zh-CN', {
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  }).format(date);
}
