/**
 * app/recall/[paperId]/page.tsx
 * page1 — 回忆卷编辑主页
 *
 * 布局：
 *   按题型分区（选择题 / 填空题 / 简答题 ...）
 *   每个题型下展示各题号支持度最高的题目
 *   点击题目 → 进入 page2 查看所有版本
 *   每个题型有「添加题目」入口
 *
 * 对接:
 *   GET /api/recall/papers/:paper_id/question-types → 题型列表
 *   GET /api/recall/papers/:paper_id/questions/top  → 各题号支持度最高题目
 */

'use client';

import { useState } from 'react';
import { useParams } from 'next/navigation';
import Link from 'next/link';
import type { QuestionTypeInfo, RecallQuestion } from '@aira/shared';
import { DetailSkeleton } from '@/components/layout/Skeleton';
import { ErrorState, EmptyState } from '@/components/layout/StateDisplay';
import { useFetch } from '@/hooks/useFetch';
import { api } from '@/lib/api';

/** 题型中文映射 */
const TYPE_LABELS: Record<string, string> = {
  singleChoice: '单选题',
  multiChoice: '多选题',
  fillBlank: '填空题',
  shortAnswer: '简答题',
  calculation: '计算题',
  proof: '证明题',
  other: '其他',
};

function typeLabel(t: string): string {
  return TYPE_LABELS[t] ?? t;
}

export default function RecallPaperPage() {
  const { paperId } = useParams<{ paperId: string }>();

  // 获取题型列表
  const { data: types, loading: typesLoading, error: typesError, refetch: refetchTypes } = useFetch(
    () => api.get<QuestionTypeInfo[]>(`/recall/papers/${paperId}/question-types`),
    [paperId],
  );

  // 获取所有题号的支持度最高题目
  const { data: topQuestions, loading: topLoading, refetch: refetchTop } = useFetch(
    () => api.get<RecallQuestion[]>(`/recall/papers/${paperId}/questions/top`),
    [paperId],
  );

  // 当前选中的题型 tab
  const [activeType, setActiveType] = useState<string | null>(null);

  // 确定展示的题型
  const currentType = activeType ?? types?.[0]?.question_type ?? null;

  // 按题型分组 top 题目
  const groupedByType: Record<string, RecallQuestion[]> = {};
  if (topQuestions) {
    for (const q of topQuestions) {
      if (!groupedByType[q.question_type]) groupedByType[q.question_type] = [];
      groupedByType[q.question_type].push(q);
    }
    // 按 sequence 排序
    for (const key of Object.keys(groupedByType)) {
      groupedByType[key].sort((a, b) => a.sequence - b.sequence);
    }
  }

  const loading = typesLoading || topLoading;

  if (loading) return <DetailSkeleton />;
  if (typesError) return <ErrorState message={typesError} onRetry={refetchTypes} />;

  return (
    <div>
      {/* 面包屑 */}
      <nav className="mb-4 text-sm text-gray-500">
        <Link href="/courses" className="transition-colors hover:text-brand-600">课程</Link>
        <span className="mx-2">›</span>
        <span className="font-medium text-gray-900">回忆卷 #{paperId}</span>
      </nav>

      <div className="mb-6 flex items-center justify-between">
        <h1 className="text-xl font-semibold text-gray-900">回忆卷编辑</h1>
        <Link href={`/recall/${paperId}/new`}
          className="rounded-md bg-brand-600 px-4 py-2 text-sm font-medium text-white hover:bg-brand-700">
          + 添加题目
        </Link>
      </div>

      {/* 题型 Tab 栏 */}
      {types && types.length > 0 ? (
        <>
          <div className="mb-4 flex gap-2 overflow-x-auto pb-1">
            {types.map((t) => (
              <button key={t.question_type}
                onClick={() => setActiveType(t.question_type)}
                className={`shrink-0 rounded-md px-4 py-2 text-sm transition-colors ${
                  currentType === t.question_type
                    ? 'bg-brand-600 text-white'
                    : 'border border-gray-200 bg-white text-gray-600 hover:border-gray-300'
                }`}>
                {typeLabel(t.question_type)}
                <span className="ml-1.5 text-xs opacity-70">({t.max_sequence}题)</span>
              </button>
            ))}
          </div>

          {/* 当前题型下的题目列表 */}
          {currentType && (
            <QuestionTypeSection
              paperId={paperId}
              questionType={currentType}
              questions={groupedByType[currentType] ?? []}
              maxSeq={types.find((t) => t.question_type === currentType)?.max_sequence ?? 0}
            />
          )}
        </>
      ) : (
        <EmptyState
          title="暂无题目"
          description="点击「添加题目」开始协作回忆试题"
        />
      )}
    </div>
  );
}

/**
 * 某个题型下的题目列表区域
 * 展示各题号的支持度最高版本
 */
function QuestionTypeSection({
  paperId,
  questionType,
  questions,
  maxSeq,
}: {
  paperId: string;
  questionType: string;
  questions: RecallQuestion[];
  maxSeq: number;
}) {
  // 用 map 快速查找
  const qMap = new Map(questions.map((q) => [q.sequence, q]));

  return (
    <div className="space-y-3">
      {/* 已有题目 */}
      {questions.length > 0 ? (
        questions.map((q) => (
          <Link key={q.id}
            href={`/recall/${paperId}/q/${questionType}/${q.sequence}`}
            className="block rounded-lg border border-gray-200 bg-white px-5 py-4
                       transition-all hover:border-brand-300 hover:shadow-sm">
            <div className="mb-2 flex items-center justify-between">
              <div className="flex items-center gap-3">
                <span className="flex h-7 w-7 items-center justify-center rounded-full bg-brand-50
                                 text-xs font-semibold text-brand-700">
                  {q.sequence}
                </span>
                <span className="text-xs text-gray-400">
                  {typeLabel(questionType)}
                </span>
              </div>
              <div className="flex items-center gap-2">
                <span className="rounded-full bg-green-50 px-2 py-0.5 text-xs font-medium text-green-600">
                  👍 {q.support_count}
                </span>
                <span className="text-xs text-gray-400">›</span>
              </div>
            </div>
            {/* 题干预览（截断） */}
            <p className="text-sm text-gray-700 line-clamp-2 leading-relaxed">
              {q.content.length > 120 ? q.content.slice(0, 120) + '...' : q.content}
            </p>
          </Link>
        ))
      ) : (
        <div className="rounded-lg border border-dashed border-gray-300 py-8 text-center text-sm text-gray-400">
          该题型暂无题目
        </div>
      )}

      {/* 缺失题号提示 */}
      {maxSeq > 0 && questions.length < maxSeq && (
        <div className="rounded-md bg-yellow-50 px-4 py-2 text-xs text-yellow-700">
          当前仅有 {questions.length}/{maxSeq} 题有内容，欢迎补充缺失题号
        </div>
      )}
    </div>
  );
}
