/**
 * app/upload/jobs/[id]/page.tsx
 * 单条上传任务详情 — 普通用户视角
 *
 * 设计原则：
 *   - 不暴露 LLM 中间产物（raw_text、parsed_json 原始 JSON），那些只在管理员审核页看
 *   - 把 parsed_json.items 按题/题解渲染成 markdown 预览，给用户一个"我上传的东西被识别成啥样"的可读视图
 */

'use client';

import Link from 'next/link';
import { useParams } from 'next/navigation';
import { useEffect, useRef, useState } from 'react';
import type {
  IngestDedupMatch,
  IngestExplanationItem,
  IngestJob,
  IngestJobStatus,
  IngestParsedEnvelope,
  IngestQuestionItem,
} from '@aira/shared';
import { MarkdownBlock, MarkdownInline } from '@/components/Markdown';
import { api } from '@/lib/api';

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

const QUESTION_TYPE_LABEL: Record<string, string> = {
  singleChoice: '单选题',
  multipleChoice: '多选题',
  trueOrFalse: '判断题',
  fillBlank: '填空题',
  shortAnswer: '简答题',
};

export default function IngestJobDetailPage() {
  const { id } = useParams<{ id: string }>();
  const [job, setJob] = useState<IngestJob | null>(null);
  const [error, setError] = useState<string | null>(null);
  const timerRef = useRef<number | null>(null);

  async function refresh() {
    try {
      const j = await api.get<IngestJob>(`/ingest/${id}`);
      setJob(j);
      setError(null);
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : '加载失败');
    }
  }

  useEffect(() => {
    void refresh();
  }, [id]);

  useEffect(() => {
    if (!job) return;
    const running = job.status === 'pending' || job.status === 'processing';
    if (!running) {
      if (timerRef.current) window.clearInterval(timerRef.current);
      timerRef.current = null;
      return;
    }
    if (timerRef.current) return;
    timerRef.current = window.setInterval(refresh, 3000);
    return () => {
      if (timerRef.current) window.clearInterval(timerRef.current);
      timerRef.current = null;
    };
  }, [job]);

  if (error) {
    return (
      <div className="rounded-2xl border border-red-200 bg-red-50 p-6 text-sm text-red-700">
        {error}
        <div className="mt-3">
          <Link href="/upload/jobs" className="text-brand-600 hover:underline">← 返回列表</Link>
        </div>
      </div>
    );
  }
  if (!job) return <div className="py-16 text-center text-gray-500">加载中...</div>;

  const items = pickItems(job.parsed_json);

  return (
    <div className="space-y-6">
      <nav className="text-sm text-gray-500">
        <Link href="/upload/jobs" className="hover:text-brand-600">我的上传记录</Link>
        <span className="mx-2">/</span>
        <span className="font-medium text-gray-900">#{job.id}</span>
      </nav>

      <header className="rounded-3xl border border-gray-200 bg-white p-6 shadow-sm">
        <div className="flex flex-wrap items-center gap-2">
          <span className={`rounded-full px-2 py-0.5 text-xs ${STATUS_COLOR[job.status] ?? 'bg-gray-100 text-gray-700'}`}>
            {STATUS_LABEL[job.status] ?? job.status}
          </span>
          <span className="rounded-full bg-brand-50 px-2 py-0.5 text-xs text-brand-700">
            {job.kind === 'question' ? '题目' : '题解'}
          </span>
        </div>
        <h1 className="mt-3 text-2xl font-semibold text-gray-900">{job.filename}</h1>
        <dl className="mt-3 grid grid-cols-1 gap-2 text-sm text-gray-700 sm:grid-cols-2">
          <Field label="课程">{job.course_id || job.new_course_name || '—'}</Field>
          <Field label={job.kind === 'question' ? '试卷名' : '目标卷'}>
            {job.kind === 'question'
              ? (job.paper_name || '—')
              : (job.target_paper_id ? `#${job.target_paper_id}` : '—')}
          </Field>
          <Field label="创建于">{formatTime(job.created_at)}</Field>
          <Field label="最近更新">{formatTime(job.updated_at)}</Field>
        </dl>
        {job.error_message && (
          <div className="mt-4 rounded-xl border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-700">
            错误：{job.error_message}
          </div>
        )}
      </header>

      {/* 状态相关的进度提示 */}
      {(job.status === 'pending' || job.status === 'processing') && (
        <div className="rounded-2xl border border-amber-200 bg-amber-50 px-4 py-3 text-sm text-amber-800">
          {job.status === 'pending'
            ? '已在排队，等待 worker 拉起...'
            : 'AI 正在清洗你的文件，3 秒刷新一次。'}
        </div>
      )}
      {job.status === 'awaiting_review' && (
        <div className="rounded-2xl border border-purple-200 bg-purple-50 px-4 py-3 text-sm text-purple-800">
          清洗完成，已进入审核队列。下方是 AI 识别后的内容预览 —— 管理员审核通过后会正式进入题库。
        </div>
      )}
      {job.status === 'published' && (
        <div className="rounded-2xl border border-green-200 bg-green-50 px-4 py-3 text-sm text-green-800">
          🎉 已发布到正式题库。
        </div>
      )}
      {job.status === 'rejected' && (
        <div className="rounded-2xl border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700">
          管理员拒绝了本次上传。{job.error_message ? `原因：${job.error_message}` : ''}
        </div>
      )}

      {/* 查重提示 — 仅在题目流程且发现疑似重复时展示 */}
      {Array.isArray(job.dedup_warnings) && job.dedup_warnings.length > 0 && (
        <DedupWarningsBanner warnings={job.dedup_warnings} />
      )}

      {/* 渲染预览 — 仅当 parsed_json 有内容时展示 */}
      {items.length > 0 && (
        <section className="space-y-4">
          <div className="flex items-center justify-between">
            <h2 className="text-lg font-semibold text-gray-900">
              AI 识别预览（共 {items.length} {job.kind === 'question' ? '题' : '条题解'}）
            </h2>
            <span className="text-xs text-gray-500">
              如有错漏，等管理员审核时会修正。
            </span>
          </div>

          <ol className="space-y-3">
            {items.map((item, i) => {
              // 每题型独立编号 — 顺着 items 数组扫，统计当前题在所属类型里是第几个
              let perTypeIdx = 1;
              const myType = (item as Record<string, unknown>).question_type;
              for (let k = 0; k < i; k++) {
                if ((items[k] as Record<string, unknown>).question_type === myType) {
                  perTypeIdx++;
                }
              }
              return (
                <li key={i}>
                  {job.kind === 'question' ? (
                    <QuestionCard
                      item={item as unknown as IngestQuestionItem}
                      displaySeq={perTypeIdx}
                    />
                  ) : (
                    <ExplanationCard item={item as unknown as IngestExplanationItem} index={i} />
                  )}
                </li>
              );
            })}
          </ol>
        </section>
      )}
    </div>
  );
}

/* ───────────────── 子组件 ───────────────── */

function Field({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div>
      <dt className="text-xs text-gray-500">{label}</dt>
      <dd className="font-medium text-gray-900">{children}</dd>
    </div>
  );
}

function QuestionCard({ item, displaySeq }: { item: IngestQuestionItem; displaySeq: number }) {
  const typeLabel = QUESTION_TYPE_LABEL[item.question_type] ?? item.question_type ?? '未知题型';
  const hasOptions = Array.isArray(item.options) && item.options.length > 0;

  return (
    <article className="rounded-2xl border border-gray-200 bg-white p-5 shadow-sm">
      <header className="mb-3 flex flex-wrap items-center gap-2">
        <span className="rounded-full bg-brand-50 px-2 py-0.5 text-xs font-medium text-brand-700">
          {typeLabel}第 {displaySeq} 题
        </span>
        {item.difficulty && (
          <span className="rounded-full bg-amber-50 px-2 py-0.5 text-xs text-amber-700">
            难度 {item.difficulty}
          </span>
        )}
      </header>

      <div className="prose prose-sm max-w-none text-gray-900">
        <MarkdownBlock content={item.test ?? ''} />
      </div>

      {hasOptions && (
        <ul className="mt-3 space-y-1.5">
          {item.options.map((opt, i) => (
            <li key={i} className="flex gap-2 text-sm text-gray-800">
              <span className="shrink-0 font-medium text-gray-500">{opt.option}.</span>
              <span><MarkdownInline content={opt.text ?? ''} /></span>
            </li>
          ))}
        </ul>
      )}

      {item.answer && (
        <div className="mt-4 rounded-xl bg-green-50 px-3 py-2 text-sm text-green-800">
          <span className="font-medium">答案：</span>
          <MarkdownInline content={item.answer} />
        </div>
      )}

      {item.explanation && (
        <div className="mt-3 rounded-xl bg-purple-50 px-3 py-2 text-sm text-purple-900">
          <div className="mb-1 font-medium text-purple-700">解析</div>
          <div className="prose prose-sm max-w-none">
            <MarkdownBlock content={item.explanation} />
          </div>
        </div>
      )}

      {Array.isArray(item.tags) && item.tags.length > 0 && (
        <div className="mt-3 flex flex-wrap gap-1.5">
          {item.tags.map((t, i) => (
            <span key={i} className="rounded-full bg-gray-100 px-2 py-0.5 text-xs text-gray-600">
              #{t}
            </span>
          ))}
        </div>
      )}
    </article>
  );
}

function DedupWarningsBanner({ warnings }: { warnings: IngestDedupMatch[] }) {
  // 按新题序号分组，避免同一道新题多个候选挤在一起
  const bySeq = new Map<number, IngestDedupMatch[]>();
  for (const w of warnings) {
    const arr = bySeq.get(w.seq) ?? [];
    arr.push(w);
    bySeq.set(w.seq, arr);
  }

  return (
    <section className="rounded-2xl border border-amber-300 bg-amber-50 p-4 text-sm text-amber-900">
      <div className="mb-2 font-semibold">
        ⚠️ AI 检测到 {bySeq.size} 道题可能与题库已有题目重复
      </div>
      <p className="mb-3 text-xs text-amber-800">
        这是字符串相似度的粗筛，不一定准。管理员审核时会再判断。
      </p>
      <ul className="space-y-2">
        {[...bySeq.entries()].map(([seq, ms]) => (
          <li key={seq} className="rounded-xl border border-amber-200 bg-white/60 p-3">
            <div className="mb-1 text-xs font-medium text-amber-900">
              第 {seq} 题 · 命中 {ms.length} 条
            </div>
            <div className="space-y-1.5">
              {ms.map((m, i) => (
                <div key={i} className="text-xs text-gray-800">
                  <span className="font-mono text-amber-700">
                    {(m.similarity * 100).toFixed(0)}%
                  </span>{' '}
                  <span className="text-gray-600">↔ 《{m.paper_name}》#{m.problem_id}</span>
                  <div className="mt-0.5 text-gray-500">"{m.existing_snippet}"</div>
                </div>
              ))}
            </div>
          </li>
        ))}
      </ul>
    </section>
  );
}

function ExplanationCard({ item, index }: { item: IngestExplanationItem; index: number }) {
  const seq = typeof item.sequence_id === 'number' ? item.sequence_id : index + 1;
  return (
    <article className="rounded-2xl border border-gray-200 bg-white p-5 shadow-sm">
      <header className="mb-3">
        <span className="rounded-full bg-brand-50 px-2 py-0.5 text-xs font-medium text-brand-700">
          第 {seq} 题题解
        </span>
      </header>
      <div className="prose prose-sm max-w-none text-gray-900">
        <MarkdownBlock content={item.content_md ?? ''} />
      </div>
    </article>
  );
}

/* ───────────────── helpers ───────────────── */

/**
 * 从 parsed_json 里抽 items 数组，兼容两种形态：
 *   - { items: [...] } 信封
 *   - 顶层就是数组
 */
function pickItems(parsed: IngestParsedEnvelope | null): Array<Record<string, unknown>> {
  if (!parsed) return [];
  // 信封
  if (Array.isArray((parsed as IngestParsedEnvelope).items)) {
    return (parsed as IngestParsedEnvelope).items as Array<Record<string, unknown>>;
  }
  // 顶层数组兜底
  if (Array.isArray(parsed)) {
    return parsed as unknown as Array<Record<string, unknown>>;
  }
  return [];
}

function formatTime(s: string): string {
  if (!s) return '—';
  const d = new Date(s);
  if (Number.isNaN(d.getTime())) return s;
  return d.toLocaleString();
}
