/**
 * app/papers/[paperId]/page.tsx
 * 试卷做题页
 *
 * 对接:
 *   GET    /api/papers/{paper_id}/problems  → 获取题目列表
 *   POST   /api/favorites                   → 收藏题目
 *   DELETE /api/favorites/{problem_id}      → 取消收藏
 *   POST   /api/answers                     → 刷题模式记录单题
 *   POST   /api/answers/batch               → 模拟考交卷批量记录
 */

'use client';

import { useState, useCallback, useEffect, useMemo, useRef } from 'react';
import { useParams, useRouter, useSearchParams } from 'next/navigation';
import Link from 'next/link';
import type { FavoriteIdList, Problem, ProblemOption } from '@aira/shared';
import { DetailSkeleton } from '@/components/layout/Skeleton';
import { EmptyState, ErrorState } from '@/components/layout/StateDisplay';
import { MarkdownBlock, MarkdownInline } from '@/components/Markdown';
import { ExplanationSection } from '@/components/problem/ExplanationSection';
import { useFetch } from '@/hooks/useFetch';
import { api } from '@/lib/api';
import { useAuth } from '@/lib/auth';
import {
  AttemptApiError,
  createAttempt,
  fetchAttempt,
  submitProblem,
  type PaperAttempt,
  type ProblemSubmission,
} from '@/lib/attempt';

type SessionMode = 'practice' | 'exam';

const DEFAULT_DURATION_MINUTES = 120;

export default function PaperDetailPage() {
  const { paperId } = useParams<{ paperId: string }>();
  const router = useRouter();
  const searchParams = useSearchParams();
  const { isLoggedIn } = useAuth();
  const autoSubmittedRef = useRef(false);
  // 答对后自动滚动到下一题的延时句柄；新滚动到来时取消旧的，组件卸载时清理
  const autoScrollTimerRef = useRef<number | null>(null);

  const { data, loading, error, refetch } = useFetch(
    () => api.get<Problem[]>(`/papers/${paperId}/problems`),
    [paperId],
  );

  const [answers, setAnswers] = useState<Record<number, string>>({});
  const [revealed, setRevealed] = useState<Set<number>>(new Set());
  const [favorites, setFavorites] = useState<Set<number>>(new Set());
  const [mode, setMode] = useState<SessionMode>('practice');
  const [started, setStarted] = useState(false);
  const [submitted, setSubmitted] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [durationMinutes, setDurationMinutes] = useState(DEFAULT_DURATION_MINUTES);
  const [autoSubmitOnTimeout, setAutoSubmitOnTimeout] = useState(true);
  const [examStartedAt, setExamStartedAt] = useState<number | null>(null);
  const [now, setNow] = useState(Date.now());
  const [sessionError, setSessionError] = useState<string | null>(null);

  // ── 严格练习模式（仅 practice 用，exam 不动）──
  const [attempt, setAttempt] = useState<PaperAttempt | null>(null);
  const [submissions, setSubmissions] = useState<Record<number, ProblemSubmission>>({});
  const [correctAnswers, setCorrectAnswers] = useState<Record<number, string>>({});
  const [submittingIds, setSubmittingIds] = useState<Set<number>>(new Set());
  const [toast, setToast] = useState<string>('');

  const showToast = useCallback((msg: string) => {
    setToast(msg);
    window.setTimeout(() => setToast(''), 3500);
  }, []);

  // 组件卸载时清理潜在的自动滚动 timer，避免 setState on unmounted / DOM 已消失的边缘问题
  useEffect(() => {
    return () => {
      if (autoScrollTimerRef.current !== null) {
        window.clearTimeout(autoScrollTimerRef.current);
        autoScrollTimerRef.current = null;
      }
    };
  }, []);

  // URL 上有 ?attempt_id=N → 直接恢复现场（刷新 / 跨标签打开）
  useEffect(() => {
    const raw = searchParams.get('attempt_id');
    if (!raw) return;
    const aid = Number(raw);
    if (!Number.isFinite(aid) || aid <= 0) return;

    let cancelled = false;
    fetchAttempt(aid)
      .then((detail) => {
        if (cancelled) return;
        setAttempt(detail.attempt);
        const map: Record<number, ProblemSubmission> = {};
        for (const s of detail.submissions) map[s.problem_id] = s;
        setSubmissions(map);
        setMode('practice');
        setStarted(true);
      })
      .catch(() => {
        if (cancelled) return;
        // 拉失败：URL 上的 attempt_id 无效，清掉，回入口
        router.replace(`/papers/${paperId}`);
      });
    return () => {
      cancelled = true;
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [searchParams, paperId]);

  useEffect(() => {
    if (!isLoggedIn) return;
    api.get<FavoriteIdList>('/favorites/ids')
      .then((ids) => setFavorites(new Set(ids)))
      .catch(() => {});
  }, [isLoggedIn]);

  useEffect(() => {
    if (!(started && mode === 'exam' && !submitted)) return undefined;
    const timer = window.setInterval(() => setNow(Date.now()), 1000);
    return () => window.clearInterval(timer);
  }, [mode, started, submitted]);

  const remainingMs = useMemo(() => {
    if (!started || mode !== 'exam' || examStartedAt === null) return durationMinutes * 60 * 1000;
    return examStartedAt + durationMinutes * 60 * 1000 - now;
  }, [durationMinutes, examStartedAt, mode, now, started]);

  const isOvertime = started && mode === 'exam' && !submitted && remainingMs < 0;

  const questionGroups = useMemo(() => {
    if (!data) return [];
    const groups = new Map<string, Problem[]>();
    data.forEach((problem) => {
      const key = problem.question_type ?? 'singleChoice';
      const current = groups.get(key) ?? [];
      current.push(problem);
      groups.set(key, current);
    });
    return Array.from(groups.entries()).map(([key, items]) => ({
      key,
      label: questionTypeLabel(key),
      items,
    }));
  }, [data]);

  const totalAnswered = mode === 'practice' && attempt
    ? attempt.submitted
    : Object.keys(answers).length;
  const totalCorrect = mode === 'practice' && attempt
    ? attempt.correct
    : (data
      ? data.filter((problem) => answers[problem.id] !== undefined && isProblemCorrect(problem, answers[problem.id])).length
      : 0);

  const showExamActions = started && mode === 'exam' && !submitted;

  const submitExam = useCallback(async (forced = false) => {
    if (!data || submitted || submitting) return;
    setSubmitting(true);
    setSessionError(null);
    try {
      const answeredProblems = data.filter((problem) => {
        const selected = answers[problem.id];
        return typeof selected === 'string' && selected.trim() !== '';
      });

      if (isLoggedIn && answeredProblems.length > 0) {
        await api.post('/answers/batch', {
          answers: answeredProblems.map((problem) => ({
            paper_id: Number(paperId),
            problem_id: problem.id,
            selected_option: answers[problem.id],
            is_correct: isProblemCorrect(problem, answers[problem.id]),
            mode: forced ? 'exam_auto_submit' : 'exam',
          })),
        });
      }

      setRevealed(new Set(data.map((problem) => problem.id)));
      setSubmitted(true);
    } catch (err) {
      setSessionError(err instanceof Error ? err.message : '交卷失败');
    } finally {
      setSubmitting(false);
    }
  }, [answers, data, isLoggedIn, paperId, submitted, submitting]);

  useEffect(() => {
    if (!(mode === 'exam' && started && !submitted && autoSubmitOnTimeout && remainingMs <= 0)) return;
    if (autoSubmittedRef.current) return;
    autoSubmittedRef.current = true;
    void submitExam(true);
  }, [autoSubmitOnTimeout, mode, remainingMs, started, submitExam, submitted]);

  /**
   * 进入做题：
   *   forceReset=false 默认 → 后端 get-or-create，命中旧 in_progress 直接恢复
   *   forceReset=true       → 后端废弃旧 in_progress 后新建
   *
   * 复用旧 attempt 时立即拉 submissions，避免"URL 没变 → useEffect 不重跑"的死角。
   */
  const startSession = useCallback(async (forceReset = false) => {
    autoSubmittedRef.current = false;
    setSubmitted(false);
    setSessionError(null);
    setAnswers({});
    setRevealed(new Set());
    setNow(Date.now());

    if (mode === 'exam') {
      setStarted(true);
      setExamStartedAt(Date.now());
      return;
    }

    setExamStartedAt(null);
    try {
      const result = await createAttempt(Number(paperId), forceReset);
      setAttempt(result.attempt);
      setCorrectAnswers({});
      setSubmittingIds(new Set());

      if (result.created) {
        // 新建：submissions 必然为空
        setSubmissions({});
      } else {
        // 复用：立即拉一次详情把已答记录灌进来
        try {
          const detail = await fetchAttempt(result.attempt.id);
          setAttempt(detail.attempt);
          const map: Record<number, ProblemSubmission> = {};
          for (const s of detail.submissions) map[s.problem_id] = s;
          setSubmissions(map);
        } catch {
          /* 拉详情失败不阻塞进入做题，URL useEffect 还会兜底 */
        }
        showToast(
          `已恢复上次进度（${result.attempt.submitted}/${result.attempt.total} 已完成）`,
        );
      }

      setStarted(true);
      router.replace(`/papers/${paperId}?attempt_id=${result.attempt.id}`);
    } catch (err) {
      setSessionError(err instanceof Error ? err.message : '创建尝试失败');
    }
  }, [mode, paperId, router, showToast]);

  const selectAnswer = useCallback((problem: Problem, value: string) => {
    setAnswers((prev) => ({ ...prev, [problem.id]: value }));
  }, []);

  /** 严格练习单题提交：网络往返，成功后锁题 + 拉服务端正确答案 */
  const submitStrict = useCallback(async (problem: Problem, value: string) => {
    if (!attempt) return;
    if (submissions[problem.id]) return; // 已提交
    if (submittingIds.has(problem.id)) return; // 提交中防双击
    const trimmed = (value ?? '').trim();
    if (trimmed === '') return;

    setSubmittingIds((prev) => {
      const next = new Set(prev);
      next.add(problem.id);
      return next;
    });

    try {
      const result = await submitProblem(attempt.id, problem.id, value);
      setSubmissions((prev) => ({ ...prev, [problem.id]: result.submission }));
      setCorrectAnswers((prev) => ({ ...prev, [problem.id]: result.correct_answer }));
      setAttempt(result.attempt);

      // 答对 → 延时 200ms 后平滑滚到"下一道未提交的题"（同向向后扫，不 wrap）
      // 若当前题已不在视口（用户主动滚开了），则放弃自动滚动以尊重用户操作
      if (result.submission.is_correct && data) {
        const submittedSet = new Set(
          Object.keys(submissions).map((k) => Number(k)),
        );
        submittedSet.add(problem.id);

        const idx = data.findIndex((p) => p.id === problem.id);
        let next: Problem | null = null;
        if (idx >= 0) {
          for (let i = idx + 1; i < data.length; i++) {
            if (!submittedSet.has(data[i].id)) {
              next = data[i];
              break;
            }
          }
        }

        if (next && isProblemCardInViewport(problem.id)) {
          if (autoScrollTimerRef.current !== null) {
            window.clearTimeout(autoScrollTimerRef.current);
          }
          const nextId = next.id;
          autoScrollTimerRef.current = window.setTimeout(() => {
            autoScrollTimerRef.current = null;
            document
              .getElementById(`problem-card-${nextId}`)
              ?.scrollIntoView({ behavior: 'smooth', block: 'center' });
          }, 200);
        }
      }
    } catch (err) {
      if (err instanceof AttemptApiError && err.status === 409) {
        // 重复提交 / attempt 已被新尝试替代：拉一次状态修复本地
        try {
          const detail = await fetchAttempt(attempt.id);
          setAttempt(detail.attempt);
          const map: Record<number, ProblemSubmission> = {};
          for (const s of detail.submissions) map[s.problem_id] = s;
          setSubmissions(map);
        } catch {
          /* ignore */
        }
        showToast('当前练习进度已过期，请刷新');
      } else {
        showToast(err instanceof Error ? err.message : '提交失败，请稍后重试');
      }
    } finally {
      setSubmittingIds((prev) => {
        const next = new Set(prev);
        next.delete(problem.id);
        return next;
      });
    }
  }, [attempt, data, submissions, submittingIds, showToast]);

  const handleSelect = useCallback((problem: Problem, value: string) => {
    // 严格练习模式：T/F / 单选 / 多选点击即提交；填空 / 编程仅保存草稿
    if (mode === 'practice' && attempt) {
      if (submissions[problem.id] || submittingIds.has(problem.id)) return;
      if (isImmediateRevealType(problem.question_type)) {
        void submitStrict(problem, value);
        return;
      }
      // 文本题：保存草稿，等用户点"提交并查看解析"
      selectAnswer(problem, value);
      return;
    }
    // exam 模式保持原状
    selectAnswer(problem, value);
  }, [attempt, mode, selectAnswer, submissions, submittingIds, submitStrict]);

  const toggleFavorite = useCallback(async (problemId: number) => {
    const isFav = favorites.has(problemId);
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
      setFavorites((prev) => {
        const next = new Set(prev);
        if (isFav) next.add(problemId);
        else next.delete(problemId);
        return next;
      });
    }
  }, [favorites]);

  const jumpToProblem = useCallback((problemId: number) => {
    document.getElementById(`problem-${problemId}`)?.scrollIntoView({ behavior: 'smooth', block: 'start' });
  }, []);

  if (loading) return <DetailSkeleton />;
  if (error) return <ErrorState message={error} onRetry={refetch} />;
  if (!data || data.length === 0) return <EmptyState title="该试卷暂无题目" />;

  return (
    <div>
      {toast ? (
        <div className="fixed right-6 top-20 z-50 rounded-xl border border-amber-200 bg-amber-50 px-4 py-3 text-sm font-medium text-amber-800 shadow-lg">
          {toast}
        </div>
      ) : null}

      <nav className="mb-4 text-sm text-gray-500">
        <Link href="/courses" className="transition-colors hover:text-brand-600">课程</Link>
        <span className="mx-2">›</span>
        <span className="font-medium text-gray-900">试卷 #{paperId}</span>
      </nav>

      {!started ? (
        <ExamSetupPanel
          mode={mode}
          durationMinutes={durationMinutes}
          autoSubmitOnTimeout={autoSubmitOnTimeout}
          onModeChange={setMode}
          onDurationChange={setDurationMinutes}
          onAutoSubmitChange={setAutoSubmitOnTimeout}
          onStart={() => void startSession(false)}
          onReset={() => {
            if (window.confirm('重置后当前进度会被废弃，确认吗？')) {
              void startSession(true);
            }
          }}
        />
      ) : (
        <>
          <div className="mb-6 flex flex-wrap items-center gap-4 rounded-xl border border-gray-200 bg-white px-5 py-4">
            <div>
              <div className="text-xs font-medium uppercase tracking-wide text-gray-400">
                {mode === 'practice' ? '刷题模式' : '模拟考模式'}
              </div>
              <div className="mt-1 text-sm text-gray-600">
                已答 <span className="font-semibold text-gray-900">{totalAnswered}</span> / {data.length} 题
                {revealed.size > 0 && (
                  <>
                    <span className="mx-2 text-gray-300">|</span>
                    正确 <span className="font-semibold text-green-600">{totalCorrect}</span> 题
                  </>
                )}
              </div>
            </div>

            {mode === 'exam' ? (
              <div className="rounded-lg border border-gray-100 bg-gray-50 px-4 py-2">
                <div className="text-xs text-gray-400">剩余时间</div>
                <div className={`mt-1 text-lg font-semibold ${isOvertime ? 'text-red-600' : 'text-gray-900'}`}>
                  {formatDuration(remainingMs)}
                </div>
              </div>
            ) : (
              <div className="rounded-lg border border-gray-100 bg-gray-50 px-4 py-2 text-sm text-gray-600">
                选择答案后可立即查看解析，填空 / 编程题需手动提交当前题目。
              </div>
            )}

            <div className="ml-auto flex items-center gap-3">
              {mode === 'practice' ? null : showExamActions ? (
                <button
                  onClick={() => void submitExam(false)}
                  disabled={submitting}
                  className="rounded-md bg-brand-600 px-3 py-1.5 text-xs font-medium text-white transition-colors hover:bg-brand-700 disabled:cursor-not-allowed disabled:opacity-50"
                >
                  {submitting ? '交卷中...' : '提交试卷'}
                </button>
              ) : (
                <span className="rounded-md bg-green-50 px-3 py-1.5 text-xs font-medium text-green-700">
                  已交卷，答案与解析已开放
                </span>
              )}
            </div>
          </div>

          {sessionError && (
            <div className="mb-4 rounded-lg border border-red-100 bg-red-50 px-4 py-3 text-sm text-red-600">
              {sessionError}
            </div>
          )}

          <div className="grid gap-6 lg:grid-cols-[240px_minmax(0,1fr)]">
            <aside className="lg:sticky lg:top-24 lg:self-start">
              <QuestionOutline
                groups={questionGroups}
                answers={answers}
                revealed={
                  mode === 'practice' && attempt
                    ? new Set(Object.keys(submissions).map(Number))
                    : revealed
                }
                onJump={jumpToProblem}
                mode={mode}
              />
            </aside>

            <div className="space-y-5">
              {data.map((problem) => {
                const sub = submissions[problem.id];
                const isStrict = mode === 'practice' && Boolean(attempt);
                const selectedDisplay = isStrict
                  ? (sub?.user_answer ?? answers[problem.id] ?? null)
                  : (answers[problem.id] ?? null);
                const isRevealedDisplay = isStrict
                  ? Boolean(sub)
                  : revealed.has(problem.id);
                return (
                  // 外层 id 用 problem-card-* 给"答对后自动滚下一题"用；
                  // ProblemCard 内层另有 id="problem-{id}" 给目录侧栏跳题用，两者并存不冲突
                  <div key={problem.id} id={`problem-card-${problem.id}`}>
                    <ProblemCard
                      problem={problem}
                      selected={selectedDisplay}
                      isRevealed={isRevealedDisplay}
                      isFavorite={favorites.has(problem.id)}
                      mode={mode}
                      submitting={submittingIds.has(problem.id)}
                      correctAnswerOverride={correctAnswers[problem.id]}
                      submissionIsCorrect={sub?.is_correct}
                      onSelect={(value) => handleSelect(problem, value)}
                      onReveal={() => {
                        if (isStrict) {
                          const value = answers[problem.id] ?? '';
                          void submitStrict(problem, value);
                        } else {
                          // exam 模式下文本题不会走 onReveal（旧 practice 逻辑已被 isStrict 覆盖）
                        }
                      }}
                      onToggleFavorite={() => void toggleFavorite(problem.id)}
                    />
                  </div>
                );
              })}
            </div>
          </div>
        </>
      )}
    </div>
  );
}

function ExamSetupPanel({
  mode,
  durationMinutes,
  autoSubmitOnTimeout,
  onModeChange,
  onDurationChange,
  onAutoSubmitChange,
  onStart,
  onReset,
}: {
  mode: SessionMode;
  durationMinutes: number;
  autoSubmitOnTimeout: boolean;
  onModeChange: (mode: SessionMode) => void;
  onDurationChange: (minutes: number) => void;
  onAutoSubmitChange: (value: boolean) => void;
  onStart: () => void;
  /** 重置按钮回调；仅 practice 模式下渲染。未传则不显示。 */
  onReset?: () => void;
}) {
  return (
    <section className="rounded-2xl border border-gray-200 bg-white px-6 py-6">
      <div className="mb-5">
        <h1 className="text-xl font-semibold text-gray-900">开始做题</h1>
        <p className="mt-1 text-sm text-gray-500">先选择刷题模式或模拟考模式，再进入试卷。</p>
      </div>

      <div className="grid gap-4 md:grid-cols-2">
        <ModeCard
          title="刷题模式"
          description="逐题作答并查看答案 / 解析，适合日常练习。"
          active={mode === 'practice'}
          onClick={() => onModeChange('practice')}
        />
        <ModeCard
          title="模拟考模式"
          description="进入前设定时长，交卷前不显示答案，到时可自动交卷。"
          active={mode === 'exam'}
          onClick={() => onModeChange('exam')}
        />
      </div>

      {mode === 'exam' && (
        <div className="mt-5 grid gap-4 rounded-xl border border-gray-100 bg-gray-50 px-4 py-4 md:grid-cols-2">
          <label className="text-sm text-gray-700">
            <span className="mb-2 block text-xs font-medium text-gray-500">考试时长（分钟）</span>
            <input
              type="number"
              min={1}
              value={durationMinutes}
              onChange={(e) => onDurationChange(Math.max(1, Number(e.target.value) || DEFAULT_DURATION_MINUTES))}
              className="w-full rounded-md border border-gray-200 bg-white px-3 py-2"
            />
          </label>

          <label className="flex items-center gap-3 rounded-md border border-gray-200 bg-white px-3 py-2 text-sm text-gray-700">
            <input
              type="checkbox"
              checked={autoSubmitOnTimeout}
              onChange={(e) => onAutoSubmitChange(e.target.checked)}
              className="size-4"
            />
            <span>时间到后自动交卷；若关闭，则超时后继续作答并以红色显示超时计时。</span>
          </label>
        </div>
      )}

      <div className="mt-6 flex flex-col items-start gap-2">
        <button
          onClick={onStart}
          className="rounded-md bg-brand-600 px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-brand-700"
        >
          进入试卷
        </button>
        {/* 重置按钮：仅 practice 模式下显示；视觉权重低于主按钮，下方链接样式 */}
        {mode === 'practice' && onReset ? (
          <button
            type="button"
            onClick={onReset}
            className="text-xs text-rose-600 underline-offset-2 transition-colors hover:underline"
          >
            重置试卷 — 废弃当前进度，开始新一轮
          </button>
        ) : null}
      </div>
    </section>
  );
}

function ModeCard({
  title,
  description,
  active,
  onClick,
}: {
  title: string;
  description: string;
  active: boolean;
  onClick: () => void;
}) {
  return (
    <button
      onClick={onClick}
      className={`rounded-xl border px-4 py-4 text-left transition-colors ${
        active ? 'border-brand-300 bg-brand-50' : 'border-gray-200 bg-white hover:bg-gray-50'
      }`}
    >
      <div className="text-sm font-semibold text-gray-900">{title}</div>
      <div className="mt-2 text-sm leading-6 text-gray-500">{description}</div>
    </button>
  );
}

function QuestionOutline({
  groups,
  answers,
  revealed,
  onJump,
  mode,
}: {
  groups: { key: string; label: string; items: Problem[] }[];
  answers: Record<number, string>;
  revealed: Set<number>;
  onJump: (problemId: number) => void;
  mode: SessionMode;
}) {
  return (
    <div className="rounded-xl border border-gray-200 bg-white px-4 py-4">
      <div className="mb-4 text-sm font-semibold text-gray-900">题目目录</div>
      <div className="space-y-4">
        {groups.map((group) => (
          <div key={group.key}>
            <div className="mb-2 text-xs font-medium uppercase tracking-wide text-gray-400">{group.label}</div>
            <div className="flex flex-wrap gap-2">
              {group.items.map((problem) => {
                const answered = typeof answers[problem.id] === 'string' && answers[problem.id].trim() !== '';
                const done = mode === 'practice' ? revealed.has(problem.id) : answered;
                return (
                  <button
                    key={problem.id}
                    onClick={() => onJump(problem.id)}
                    className={`flex size-9 items-center justify-center rounded-md border text-xs font-medium transition-colors ${
                      done
                        ? 'border-brand-300 bg-brand-50 text-brand-700'
                        : 'border-gray-200 text-gray-500 hover:bg-gray-50'
                    }`}
                  >
                    {problem.order}
                  </button>
                );
              })}
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}

interface ProblemCardProps {
  problem: Problem;
  selected: string | null;
  isRevealed: boolean;
  isFavorite: boolean;
  mode: SessionMode;
  submitting?: boolean;
  correctAnswerOverride?: string;
  submissionIsCorrect?: boolean;
  onSelect: (value: string) => void;
  onReveal: () => void;
  onToggleFavorite: () => void;
}

function ProblemCard({
  problem,
  selected,
  isRevealed,
  isFavorite,
  mode,
  submitting = false,
  correctAnswerOverride,
  submissionIsCorrect,
  onSelect,
  onReveal,
  onToggleFavorite,
}: ProblemCardProps) {
  const questionType = problem.question_type ?? 'singleChoice';
  // 严格模式下优先用服务端返回的正确答案；否则 fallback 到题目自带的 answer 字段
  const rawCorrect = correctAnswerOverride ?? problem.answer;
  const normalizedAnswer = normalizeAnswer(rawCorrect, questionType);
  const isTextQuestion = isTextResponseType(questionType);
  // 已锁时若有服务端 is_correct 用服务端；否则用客户端推断
  const isCorrect = isRevealed
    ? (submissionIsCorrect ?? (selected ? isProblemCorrect(problem, selected) : false))
    : false;
  const answerVisible = isRevealed;

  return (
    <div id={`problem-${problem.id}`} className="overflow-hidden rounded-xl border border-gray-200 bg-white">
      <div className="flex items-center justify-between border-b border-gray-100 bg-gray-50 px-5 py-3">
        <div className="flex items-center gap-3">
          <span className="text-sm font-medium text-gray-700">第 {problem.order} 题</span>
          <span className="rounded-md bg-white px-2 py-1 text-xs text-gray-500">
            {questionTypeLabel(questionType)}
          </span>
          <span className="rounded-md bg-white px-2 py-1 text-xs text-gray-500">
            {problem.score ?? 0} 分
          </span>
        </div>
        <button
          onClick={onToggleFavorite}
          className={`rounded-md px-2 py-1 text-xs transition-colors ${
            isFavorite ? 'bg-yellow-50 font-medium text-yellow-600' : 'text-gray-400 hover:text-yellow-500'
          }`}
        >
          {isFavorite ? '★ 已收藏' : '☆ 收藏'}
        </button>
      </div>

      <div className="px-5 py-4">
        <MarkdownBlock content={problem.test} className="prose prose-sm mb-4 max-w-none text-gray-800" />

        {questionType === 'trueOrFalse' ? (
          <div className={`mb-4 space-y-2 ${submitting ? 'opacity-60' : ''}`}>
            {['T', 'F'].map((value) => (
              <OptionButton
                key={value}
                opt={{ option: value, text: value === 'T' ? 'True' : 'False' }}
                isSelected={selected === value}
                isAnswer={normalizedAnswer === value}
                isRevealed={answerVisible}
                onClick={() => { if (!answerVisible && !submitting) onSelect(value); }}
              />
            ))}
          </div>
        ) : isTextQuestion ? (
          <div className="mb-4">
            <textarea
              value={selected ?? ''}
              onChange={(e) => onSelect(e.target.value)}
              disabled={answerVisible || submitting}
              placeholder={questionType === 'programming' ? '输入你的代码思路或答案' : '填写你的答案'}
              rows={questionType === 'programming' ? 8 : 4}
              className="w-full rounded-md border border-gray-200 px-3 py-2 text-sm text-gray-700 disabled:bg-gray-50"
            />
          </div>
        ) : (
          <div className={`mb-4 space-y-2 ${submitting ? 'opacity-60' : ''}`}>
            {problem.options.map((opt) => (
              <OptionButton
                key={opt.option}
                opt={opt}
                isSelected={selected === opt.option}
                isAnswer={normalizedAnswer === opt.option}
                isRevealed={answerVisible}
                onClick={() => { if (!answerVisible && !submitting) onSelect(opt.option); }}
              />
            ))}
          </div>
        )}

        <div className="flex items-center gap-3">
          {mode === 'practice' && !answerVisible && isTextQuestion && (
            <button
              onClick={onReveal}
              disabled={!selected?.trim() || submitting}
              className="rounded-md border border-gray-200 px-3 py-1.5 text-xs text-gray-600 transition-colors hover:bg-gray-50 disabled:cursor-not-allowed disabled:opacity-40"
            >
              {submitting ? '提交中…' : '提交并查看解析'}
            </button>
          )}
          {mode === 'practice' && !answerVisible && !isTextQuestion && submitting && (
            <span className="inline-flex items-center gap-2 text-xs text-gray-500">
              <span className="inline-block h-2 w-2 animate-pulse rounded-full bg-brand-500" />
              提交中…
            </span>
          )}

          {mode === 'exam' && !answerVisible && (
            <span className="text-xs text-gray-400">模拟考中，交卷后统一显示答案与解析。</span>
          )}
        </div>

        {answerVisible && (
          <div className="mt-4 flex flex-col gap-2">
            {isTextQuestion ? (
              <span className="rounded-md bg-blue-50 px-3 py-1.5 text-xs font-medium text-blue-600">
                参考答案已显示
              </span>
            ) : (
              <span className={`w-fit rounded-md px-3 py-1.5 text-xs font-medium ${
                isCorrect ? 'bg-green-50 text-green-600' : 'bg-red-50 text-red-600'
              }`}>
                {isCorrect ? '✓ 回答正确' : '✗ 回答错误'}
              </span>
            )}

            <div className="rounded-md border border-gray-100 bg-gray-50 px-3 py-2 text-xs text-gray-600">
              <span className="font-medium">正确答案：</span>
              {isTextQuestion ? (
                <MarkdownBlock content={normalizedAnswer} className="prose prose-sm mt-2 max-w-none text-gray-700" />
              ) : (
                <MarkdownInline content={normalizedAnswer} />
              )}
            </div>
          </div>
        )}

        <ExplanationSection
          problemId={problem.id}
          enabled={answerVisible}
          officialExplanation={problem.explanation}
        />
      </div>
    </div>
  );
}

function OptionButton({
  opt,
  isSelected,
  isAnswer,
  isRevealed,
  onClick,
}: {
  opt: ProblemOption;
  isSelected: boolean;
  isAnswer: boolean;
  isRevealed: boolean;
  onClick: () => void;
}) {
  let className = 'w-full rounded-lg border px-4 py-2.5 text-left text-sm transition-all ';

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

/**
 * isProblemCardInViewport
 * 判断指定题目卡片是否还在用户当前视口里。
 * 用于"答对后自动滚到下一题"前的检查：如果用户已经主动滚走，就放弃自动滚动，尊重用户操作。
 */
function isProblemCardInViewport(problemId: number): boolean {
  if (typeof window === 'undefined' || typeof document === 'undefined') return false;
  const el = document.getElementById(`problem-card-${problemId}`);
  if (!el) return false;
  const rect = el.getBoundingClientRect();
  return rect.bottom > 0 && rect.top < window.innerHeight;
}

function questionTypeLabel(type?: string) {
  switch (type) {
    case 'singleChoice':
      return '单选题';
    case 'trueOrFalse':
      return '判断题';
    case 'fillBlanks':
      return '填空题';
    case 'programming':
      return '编程题';
    default:
      return type ?? '题目';
  }
}

function isTextResponseType(type?: string) {
  return type === 'fillBlanks' || type === 'programming';
}

function isImmediateRevealType(type?: string) {
  return type === 'singleChoice' || type === 'trueOrFalse';
}

function normalizeAnswer(answer: string, questionType?: string) {
  const raw = answer.trim();
  if (questionType === 'trueOrFalse') {
    const lowered = raw.toLowerCase();
    if (lowered === 'true' || lowered === 't') return 'T';
    if (lowered === 'false' || lowered === 'f') return 'F';
  }
  return raw;
}

function isProblemCorrect(problem: Problem, selected: string) {
  const questionType = problem.question_type ?? 'singleChoice';
  const normalizedAnswer = normalizeAnswer(problem.answer, questionType);
  if (isTextResponseType(questionType)) {
    return selected.trim().toLowerCase() === normalizedAnswer.trim().toLowerCase();
  }
  return selected === normalizedAnswer;
}

function formatDuration(ms: number) {
  const totalSeconds = Math.floor(Math.abs(ms) / 1000);
  const hours = String(Math.floor(totalSeconds / 3600)).padStart(2, '0');
  const minutes = String(Math.floor((totalSeconds % 3600) / 60)).padStart(2, '0');
  const seconds = String(totalSeconds % 60).padStart(2, '0');
  const content = `${hours}:${minutes}:${seconds}`;
  return ms >= 0 ? content : `+${content}`;
}
