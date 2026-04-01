/**
 * app/papers/[paperId]/page.tsx
 * 试卷做题页 — MVP 核心交互页
 *
 * 对接:
 *   GET    /api/papers/{paper_id}/problems  → 获取题目列表
 *   POST   /api/favorites                   → 收藏题目
 *   DELETE /api/favorites/{problem_id}      → 取消收藏
 */

'use client';

import { useState, useCallback, useEffect } from 'react';
import { useParams } from 'next/navigation';
import Link from 'next/link';
import type { Problem, ProblemOption, FavoriteIdList } from '@aira/shared';
import { DetailSkeleton } from '@/components/layout/Skeleton';
import { ErrorState, EmptyState } from '@/components/layout/StateDisplay';
import { useFetch } from '@/hooks/useFetch';
import { api } from '@/lib/api';
import { useAuth } from '@/lib/auth';
import { MarkdownBlock, MarkdownInline } from '@/components/Markdown';

export default function PaperDetailPage() {
  const { paperId } = useParams<{ paperId: string }>();
  const { isLoggedIn } = useAuth();

  const { data, loading, error, refetch } = useFetch(
    () => api.get<Problem[]>(`/papers/${paperId}/problems`),
    [paperId],
  );

  // 用户选择状态：{ problemId → 选中的 option }
  const [answers, setAnswers] = useState<Record<number, string>>({});
  // 已揭晓答案的题目
  const [revealed, setRevealed] = useState<Set<number>>(new Set());
  // 已收藏的题目
  const [favorites, setFavorites] = useState<Set<number>>(new Set());

  useEffect(() => {
    if (!isLoggedIn) return;
    api.get<FavoriteIdList>('/favorites/ids')
      .then((ids) => setFavorites(new Set(ids)))
      .catch(() => {});
  }, [isLoggedIn]);

  const selectAnswer = useCallback((problemId: number, option: string) => {
    setAnswers((prev) => ({ ...prev, [problemId]: option }));
  }, []);

  const revealAnswer = useCallback((problemId: number) => {
    setRevealed((prev) => new Set(prev).add(problemId));
    if (!isLoggedIn || !data) return;
    const selected = answers[problemId];
    const problem = data.find((p) => p.id === problemId);
    if (!problem || !selected) return;
    api.post('/answers', {
      paper_id: Number(paperId),
      problem_id: problemId,
      selected_option: selected,
      is_correct: selected === problem.answer,
      mode: 'practice',
    }).catch(() => {});
  }, [answers, data, isLoggedIn, paperId]);

  /**
   * ★ 收藏/取消收藏 — 对接 POST/DELETE /api/favorites
   */
  const toggleFavorite = useCallback(async (problemId: number) => {
    const isFav = favorites.has(problemId);
    // 乐观更新 UI
    setFavorites((prev) => {
      const next = new Set(prev);
      if (isFav) next.delete(problemId);
      else next.add(problemId);
      return next;
    });
    try {
      if (isFav) {
        await api.delete(`/favorites/${problemId}`);
      } else {
        await api.post('/favorites', { problem_id: problemId });
      }
    } catch {
      // 请求失败时回滚
      setFavorites((prev) => {
        const next = new Set(prev);
        if (isFav) next.add(problemId);
        else next.delete(problemId);
        return next;
      });
    }
  }, [favorites]);

  // 统计
  const totalAnswered = Object.keys(answers).length;
  const totalCorrect = data
    ? data.filter((p) => revealed.has(p.id) && answers[p.id] === p.answer).length
    : 0;

  if (loading) return <DetailSkeleton />;
  if (error) return <ErrorState message={error} onRetry={refetch} />;
  if (!data || data.length === 0) return <EmptyState title="该试卷暂无题目" />;

  return (
    <div>
      {/* 面包屑 */}
      <nav className="mb-4 text-sm text-gray-500">
        <Link href="/courses" className="transition-colors hover:text-brand-600">课程</Link>
        <span className="mx-2">›</span>
        <span className="font-medium text-gray-900">试卷 #{paperId}</span>
      </nav>

      {/* 进度条 */}
      <div className="mb-6 flex items-center gap-4 rounded-lg border border-gray-200 bg-white px-5 py-3">
        <span className="text-sm text-gray-600">
          已答 <span className="font-semibold text-gray-900">{totalAnswered}</span> / {data.length} 题
        </span>
        {revealed.size > 0 && (
          <span className="text-sm text-gray-600">
            正确 <span className="font-semibold text-green-600">{totalCorrect}</span> 题
          </span>
        )}
        <div className="ml-auto">
          <button onClick={() => data.forEach((p) => revealAnswer(p.id))}
            className="rounded-md bg-brand-600 px-3 py-1.5 text-xs font-medium text-white
                       transition-colors hover:bg-brand-700">
            全部揭晓
          </button>
        </div>
      </div>

      {/* 题目列表 */}
      <div className="space-y-5">
        {data.map((problem) => (
          <ProblemCard
            key={problem.id}
            problem={problem}
            selected={answers[problem.id] ?? null}
            isRevealed={revealed.has(problem.id)}
            isFavorite={favorites.has(problem.id)}
            onSelect={(opt) => selectAnswer(problem.id, opt)}
            onReveal={() => revealAnswer(problem.id)}
            onToggleFavorite={() => toggleFavorite(problem.id)}
          />
        ))}
      </div>
    </div>
  );
}

/* ══════════ ProblemCard 组件 ══════════ */

interface ProblemCardProps {
  problem: Problem;
  selected: string | null;
  isRevealed: boolean;
  isFavorite: boolean;
  onSelect: (option: string) => void;
  onReveal: () => void;
  onToggleFavorite: () => void;
}

function ProblemCard({
  problem, selected, isRevealed, isFavorite,
  onSelect, onReveal, onToggleFavorite,
}: ProblemCardProps) {
  const questionType = problem.question_type ?? 'singleChoice';
  const normalizedAnswer = normalizeAnswer(problem.answer, questionType);
  const isCorrect = questionType === 'fillBlanks'
    ? selected?.trim().toLowerCase() === normalizedAnswer.toLowerCase()
    : selected === normalizedAnswer;

  return (
    <div className="rounded-xl border border-gray-200 bg-white overflow-hidden">
      {/* 题头 */}
      <div className="flex items-center justify-between border-b border-gray-100 bg-gray-50 px-5 py-3">
        <span className="text-sm font-medium text-gray-700">
          第 {problem.order} 题
        </span>
        <button onClick={onToggleFavorite}
          className={`rounded-md px-2 py-1 text-xs transition-colors ${
            isFavorite
              ? 'bg-yellow-50 text-yellow-600 font-medium'
              : 'text-gray-400 hover:text-yellow-500'
          }`}>
          {isFavorite ? '★ 已收藏' : '☆ 收藏'}
        </button>
      </div>

      <div className="px-5 py-4">
        {/* 题干 */}
        <MarkdownBlock content={problem.test} className="prose prose-sm max-w-none text-gray-800 mb-4" />

        {/* 选项 / 作答区 */}
        {questionType === 'trueOrFalse' ? (
          <div className="space-y-2 mb-4">
            {['T', 'F'].map((val) => (
              <OptionButton
                key={val}
                opt={{ option: val, text: val === 'T' ? 'True' : 'False' }}
                isSelected={selected === val}
                isAnswer={normalizedAnswer === val}
                isRevealed={isRevealed}
                onClick={() => { if (!isRevealed) onSelect(val); }}
              />
            ))}
          </div>
        ) : questionType === 'fillBlanks' ? (
          <div className="mb-4">
            <textarea
              value={selected ?? ''}
              onChange={(e) => onSelect(e.target.value)}
              placeholder="填写你的答案"
              disabled={isRevealed}
              className="w-full rounded-md border border-gray-200 px-3 py-2 text-sm text-gray-700"
              rows={3}
            />
          </div>
        ) : (
          <div className="space-y-2 mb-4">
            {problem.options.map((opt) => (
              <OptionButton
                key={opt.option}
                opt={opt}
                isSelected={selected === opt.option}
                isAnswer={normalizedAnswer === opt.option}
                isRevealed={isRevealed}
                onClick={() => { if (!isRevealed) onSelect(opt.option); }}
              />
            ))}
          </div>
        )}

        {/* 操作栏 */}
        <div className="flex items-center gap-3">
          {!isRevealed ? (
            <button onClick={onReveal} disabled={!selected}
              className="rounded-md border border-gray-200 px-3 py-1.5 text-xs text-gray-600
                         transition-colors hover:bg-gray-50
                         disabled:cursor-not-allowed disabled:opacity-40">
              查看答案
            </button>
          ) : (
            <div className="flex flex-col gap-2">
              {questionType === 'fillBlanks' ? (
                <span className="rounded-md bg-blue-50 px-3 py-1.5 text-xs font-medium text-blue-600">
                  参考答案已显示
                </span>
              ) : (
                <span className={`rounded-md px-3 py-1.5 text-xs font-medium ${
                  isCorrect
                    ? 'bg-green-50 text-green-600'
                    : 'bg-red-50 text-red-600'
                }`}>
                  {isCorrect ? '✓ 回答正确' : '✗ 回答错误'}
                </span>
              )}
              <div className="rounded-md border border-gray-100 bg-gray-50 px-3 py-2 text-xs text-gray-600">
                <span className="font-medium">正确答案：</span>
                <MarkdownInline content={normalizedAnswer} />
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}

/* ══════════ OptionButton 组件 ══════════ */

interface OptionButtonProps {
  opt: ProblemOption;
  isSelected: boolean;
  isAnswer: boolean;
  isRevealed: boolean;
  onClick: () => void;
}

function OptionButton({ opt, isSelected, isAnswer, isRevealed, onClick }: OptionButtonProps) {
  let className = 'w-full text-left rounded-lg border px-4 py-2.5 text-sm transition-all ';

  if (isRevealed && isAnswer) {
    className += 'border-green-300 bg-green-50 text-green-800';
  } else if (isRevealed && isSelected && !isAnswer) {
    className += 'border-red-300 bg-red-50 text-red-800';
  } else if (isSelected) {
    className += 'border-brand-300 bg-brand-50 text-brand-800';
  } else {
    className += 'border-gray-100 text-gray-700 hover:border-gray-200 hover:bg-gray-50';
  }

  return (
    <button onClick={onClick} className={className} disabled={isRevealed}>
      <span className="mr-2 font-medium">{opt.option}.</span>
      <MarkdownInline content={opt.text} />
      {isRevealed && isAnswer && <span className="ml-2">✓</span>}
      {isRevealed && isSelected && !isAnswer && <span className="ml-2">✗</span>}
    </button>
  );
}

function normalizeAnswer(answer: string, questionType: string) {
  const raw = answer.trim();
  if (questionType === 'trueOrFalse') {
    const lowered = raw.toLowerCase();
    if (lowered === 'true' || lowered === 't') return 'T';
    if (lowered === 'false' || lowered === 'f') return 'F';
  }
  return raw;
}
