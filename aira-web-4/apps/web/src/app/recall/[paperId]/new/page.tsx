/**
 * app/recall/[paperId]/new/page.tsx
 * 新增题目 — Markdown 编辑器
 *
 * 流程：
 *   1. 选择题型
 *   2. 确定题号（展示当前最大题号供参考）
 *   3. 输入题干（Markdown）
 *   4. 选择题可添加选项 + 答案
 *   5. 提交 → POST /api/recall/papers/:paper_id/questions
 *
 * 对接:
 *   GET  /api/recall/papers/:paper_id/question-types → 获取现有题型和最大题号
 *   POST /api/recall/papers/:paper_id/questions       → 创建题目
 */

'use client';

import { useState, useCallback } from 'react';
import { useParams, useRouter } from 'next/navigation';
import Link from 'next/link';
import type { QuestionTypeInfo, CreateRecallQuestionDto, ProblemOption } from '@aira/shared';
import { useFetch } from '@/hooks/useFetch';
import { api } from '@/lib/api';

const QUESTION_TYPES = [
  { value: 'singleChoice', label: '单选题' },
  { value: 'multiChoice', label: '多选题' },
  { value: 'fillBlank', label: '填空题' },
  { value: 'shortAnswer', label: '简答题' },
  { value: 'calculation', label: '计算题' },
  { value: 'proof', label: '证明题' },
  { value: 'other', label: '其他' },
];

const isChoiceType = (t: string) => t === 'singleChoice' || t === 'multiChoice';

export default function NewQuestionPage() {
  const { paperId } = useParams<{ paperId: string }>();
  const router = useRouter();

  // 获取现有题型信息（用于题号参考）
  const { data: typeInfos } = useFetch(
    () => api.get<QuestionTypeInfo[]>(`/recall/papers/${paperId}/question-types`),
    [paperId],
  );

  // 表单状态
  const [questionType, setQuestionType] = useState('singleChoice');
  const [sequence, setSequence] = useState(1);
  const [content, setContent] = useState('');
  const [answer, setAnswer] = useState('');
  const [options, setOptions] = useState<ProblemOption[]>([
    { option: 'A', text: '' },
    { option: 'B', text: '' },
    { option: 'C', text: '' },
    { option: 'D', text: '' },
  ]);
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState('');

  // 当前题型的最大题号
  const currentMaxSeq = typeInfos?.find((t) => t.question_type === questionType)?.max_sequence ?? 0;

  // 切换题型时重置题号（题号可能重置）
  const handleTypeChange = (newType: string) => {
    setQuestionType(newType);
    const info = typeInfos?.find((t) => t.question_type === newType);
    setSequence((info?.max_sequence ?? 0) + 1);
    // 切换到非选择题时清空选项
    if (!isChoiceType(newType)) {
      setOptions([]);
      setAnswer('');
    } else if (options.length === 0) {
      setOptions([
        { option: 'A', text: '' },
        { option: 'B', text: '' },
        { option: 'C', text: '' },
        { option: 'D', text: '' },
      ]);
    }
  };

  // 更新选项文本
  const updateOption = (idx: number, text: string) => {
    setOptions((prev) => prev.map((o, i) => (i === idx ? { ...o, text } : o)));
  };

  // 添加选项
  const addOption = () => {
    const next = String.fromCharCode(65 + options.length); // E, F, G...
    setOptions((prev) => [...prev, { option: next, text: '' }]);
  };

  // 删除最后一个选项
  const removeLastOption = () => {
    if (options.length <= 2) return;
    setOptions((prev) => prev.slice(0, -1));
    // 如果答案是被删除的选项，清空答案
    const removedLabel = options[options.length - 1].option;
    if (answer === removedLabel) setAnswer('');
  };

  /** 提交 — POST /api/recall/papers/:paper_id/questions */
  const handleSubmit = useCallback(async () => {
    if (!content.trim()) { setError('请填写题干'); return; }
    if (sequence < 1) { setError('题号须 ≥ 1'); return; }

    setError('');
    setSubmitting(true);

    const body: CreateRecallQuestionDto = {
      question_type: questionType,
      sequence,
      content: content.trim(),
      answer: answer.trim() || undefined,
      options: isChoiceType(questionType) ? options.filter((o) => o.text.trim()) : undefined,
    };

    console.log('[NewQuestion] submitting:', body);

    try {
      await api.post(`/recall/papers/${paperId}/questions`, body);
      console.log('[NewQuestion] created, redirecting...');
      router.push(`/recall/${paperId}`);
    } catch (err) {
      console.error('[NewQuestion] error:', err);
      setError(err instanceof Error ? err.message : '提交失败');
    } finally {
      setSubmitting(false);
    }
  }, [questionType, sequence, content, answer, options, paperId, router]);

  return (
    <div className="mx-auto max-w-2xl">
      {/* 面包屑 */}
      <nav className="mb-4 text-sm text-gray-500">
        <Link href={`/recall/${paperId}`} className="transition-colors hover:text-brand-600">
          回忆卷 #{paperId}
        </Link>
        <span className="mx-2">›</span>
        <span className="font-medium text-gray-900">添加题目</span>
      </nav>

      <h1 className="mb-6 text-xl font-semibold text-gray-900">添加题目</h1>

      <div className="space-y-6 rounded-xl border border-gray-200 bg-white p-6">

        {error && (
          <div className="rounded-md bg-red-50 px-3 py-2 text-sm text-red-600">{error}</div>
        )}

        {/* ── 题型选择 ── */}
        <div>
          <label className="mb-2 block text-sm font-medium text-gray-700">题目类型</label>
          <div className="flex flex-wrap gap-2">
            {QUESTION_TYPES.map((t) => (
              <button key={t.value}
                onClick={() => handleTypeChange(t.value)}
                className={`rounded-md px-3 py-1.5 text-sm transition-colors ${
                  questionType === t.value
                    ? 'bg-brand-600 text-white'
                    : 'border border-gray-200 text-gray-600 hover:border-gray-300'
                }`}>
                {t.label}
              </button>
            ))}
          </div>
        </div>

        {/* ── 题号 ── */}
        <div>
          <label className="mb-2 block text-sm font-medium text-gray-700">
            题号
            {currentMaxSeq > 0 && (
              <span className="ml-2 font-normal text-gray-400">
                (该题型当前最大题号: {currentMaxSeq})
              </span>
            )}
          </label>
          <input type="number" min={1} value={sequence}
            onChange={(e) => setSequence(Math.max(1, Number(e.target.value)))}
            className="w-32 rounded-md border border-gray-200 px-3 py-2 text-sm outline-none
                       focus:border-brand-500 focus:ring-1 focus:ring-brand-500" />
          <p className="mt-1 text-xs text-gray-400">
            同一题号可有多人提交不同版本，最终保留支持度最高的
          </p>
        </div>

        {/* ── 题干 ── */}
        <div>
          <label className="mb-2 block text-sm font-medium text-gray-700">
            题干
            <span className="ml-1 font-normal text-gray-400">(支持 Markdown 语法)</span>
          </label>
          <textarea value={content} onChange={(e) => setContent(e.target.value)}
            rows={8} placeholder="输入题目内容，支持 Markdown 和图片链接 ![alt](url)"
            className="w-full rounded-md border border-gray-200 px-3 py-2 font-mono text-sm leading-relaxed
                       outline-none focus:border-brand-500 focus:ring-1 focus:ring-brand-500" />
        </div>

        {/* ── 选项（仅选择题） ── */}
        {isChoiceType(questionType) && (
          <div>
            <label className="mb-2 block text-sm font-medium text-gray-700">选项</label>
            <div className="space-y-2">
              {options.map((opt, i) => (
                <div key={opt.option} className="flex items-center gap-2">
                  <span className="flex h-7 w-7 shrink-0 items-center justify-center rounded-full
                                   bg-gray-100 text-xs font-medium text-gray-500">
                    {opt.option}
                  </span>
                  <input type="text" value={opt.text}
                    onChange={(e) => updateOption(i, e.target.value)}
                    placeholder={`选项 ${opt.option} 内容`}
                    className="flex-1 rounded-md border border-gray-200 px-3 py-2 text-sm outline-none
                               focus:border-brand-500 focus:ring-1 focus:ring-brand-500" />
                </div>
              ))}
            </div>
            <div className="mt-2 flex gap-2">
              <button onClick={addOption}
                className="text-xs text-brand-600 hover:underline">
                + 增加选项
              </button>
              {options.length > 2 && (
                <button onClick={removeLastOption}
                  className="text-xs text-red-500 hover:underline">
                  - 删除最后一个
                </button>
              )}
            </div>

            {/* 正确答案 */}
            <div className="mt-3">
              <label className="mb-1 block text-xs text-gray-500">正确答案（可不填）</label>
              <div className="flex gap-2">
                {options.map((opt) => (
                  <button key={opt.option}
                    onClick={() => setAnswer(answer === opt.option ? '' : opt.option)}
                    className={`rounded-md px-3 py-1.5 text-sm transition-colors ${
                      answer === opt.option
                        ? 'bg-green-500 text-white'
                        : 'border border-gray-200 text-gray-500 hover:border-green-300'
                    }`}>
                    {opt.option}
                  </button>
                ))}
              </div>
            </div>
          </div>
        )}

        {/* ── 答案（非选择题） ── */}
        {!isChoiceType(questionType) && (
          <div>
            <label className="mb-2 block text-sm font-medium text-gray-700">
              参考答案
              <span className="ml-1 font-normal text-gray-400">(可为空，支持 Markdown)</span>
            </label>
            <textarea value={answer} onChange={(e) => setAnswer(e.target.value)}
              rows={4} placeholder="输入参考答案（可选）"
              className="w-full rounded-md border border-gray-200 px-3 py-2 font-mono text-sm
                         outline-none focus:border-brand-500 focus:ring-1 focus:ring-brand-500" />
          </div>
        )}

        {/* ── 提交 ── */}
        <div className="flex gap-3 pt-2">
          <button type="button" onClick={handleSubmit}
            disabled={submitting || !content.trim()}
            className="rounded-md bg-brand-600 px-6 py-2 text-sm font-medium text-white
                       hover:bg-brand-700 disabled:opacity-50 disabled:cursor-not-allowed">
            {submitting ? '提交中...' : '提交题目'}
          </button>
          <Link href={`/recall/${paperId}`}
            className="rounded-md border border-gray-200 px-4 py-2 text-sm text-gray-500
                       hover:bg-gray-50">
            取消
          </Link>
        </div>
      </div>
    </div>
  );
}
