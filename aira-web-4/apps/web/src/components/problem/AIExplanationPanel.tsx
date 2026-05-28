/**
 * components/problem/AIExplanationPanel.tsx
 * 题目 AI 解析面板（SSE 流式 + 缓存预拉）
 *
 * 加载时序：
 *   mount → GET /api/llm/explain/cached  ← 自动预拉缓存
 *     ├── 命中：直接静默渲染完整文本；按钮 = "重新生成 ✨"
 *     └── 未命中：显示初始空态；按钮 = "AI 辅导 ✨"
 *
 *   点击按钮 → POST /api/llm/explain (SSE)
 *     ├── 流式 token 逐字追加；按钮 = "停止生成"
 *     └── 流结束：后端异步落库，前端不需要二次保存
 *
 * 状态机：
 *   idle           初始 / 已生成完毕（可重新生成）
 *   prefetching    挂载后查缓存中
 *   streaming      正在接收 token
 *   error          流式过程中失败
 */

'use client';

import { useEffect, useRef, useState } from 'react';
import { fetchCachedExplanation, streamExplain } from '@/lib/llm';
import { MarkdownBlock } from '@/components/Markdown';

type Status = 'idle' | 'prefetching' | 'streaming' | 'error';

interface Props {
  problemId: number;
}

export function AIExplanationPanel({ problemId }: Props) {
  const [content, setContent] = useState<string>('');
  const [status, setStatus] = useState<Status>('prefetching');
  const [errorMsg, setErrorMsg] = useState<string>('');
  /** 区分"从未生成 / 用户已经生成过 / 命中过缓存"：决定按钮文案 */
  const [hasContent, setHasContent] = useState(false);
  const [cachedAt, setCachedAt] = useState<string>(''); // 缓存命中时记录时间
  const controllerRef = useRef<AbortController | null>(null);

  // 挂载时尝试预拉缓存
  useEffect(() => {
    let cancelled = false;
    setStatus('prefetching');

    fetchCachedExplanation(problemId)
      .then((cache) => {
        if (cancelled) return;
        if (cache.found && cache.content) {
          setContent(cache.content);
          setHasContent(true);
          setCachedAt(cache.created_at ?? '');
        }
        setStatus('idle');
      })
      .catch(() => {
        if (cancelled) return;
        // 预拉失败时静默回落到"未生成"态，不打扰用户
        setStatus('idle');
      });

    return () => {
      cancelled = true;
      // 离开页面时取消流，防止后台残留
      controllerRef.current?.abort();
    };
  }, [problemId]);

  const start = () => {
    controllerRef.current?.abort();
    setContent('');
    setErrorMsg('');
    setCachedAt(''); // 重新生成会覆盖缓存语义
    setStatus('streaming');

    controllerRef.current = streamExplain(problemId, {
      onToken: (text) => {
        setContent((prev) => prev + text);
      },
      onDone: () => {
        setStatus('idle');
        setHasContent(true);
        controllerRef.current = null;
      },
      onError: (message) => {
        setStatus('error');
        setErrorMsg(message);
        setHasContent(true);
        controllerRef.current = null;
      },
    });
  };

  const stop = () => {
    controllerRef.current?.abort();
    controllerRef.current = null;
    setStatus('idle');
    setHasContent(true);
  };

  const renderButton = () => {
    if (status === 'streaming') {
      return (
        <button
          type="button"
          onClick={stop}
          className="inline-flex items-center gap-1 rounded-lg border border-rose-200 bg-white px-3 py-1.5 text-xs font-medium text-rose-600 transition-colors hover:bg-rose-50"
        >
          停止生成
        </button>
      );
    }
    if (status === 'prefetching') {
      return (
        <button
          type="button"
          disabled
          className="inline-flex items-center gap-1 rounded-lg bg-gray-100 px-3 py-1.5 text-xs font-medium text-gray-400"
        >
          加载中…
        </button>
      );
    }
    return (
      <button
        type="button"
        onClick={start}
        className="inline-flex items-center gap-1 rounded-lg bg-brand-600 px-3 py-1.5 text-xs font-medium text-white shadow-sm transition-colors hover:bg-brand-700"
      >
        {hasContent ? '重新生成 ✨' : 'AI 辅导 ✨'}
      </button>
    );
  };

  const formatCachedAt = (iso: string): string => {
    try {
      return new Intl.DateTimeFormat('zh-CN', {
        dateStyle: 'medium',
        timeStyle: 'short',
      }).format(new Date(iso));
    } catch {
      return iso;
    }
  };

  return (
    <section className="rounded-2xl border border-gray-200 bg-white p-4">
      <header className="flex items-center justify-between gap-3">
        <div className="flex items-center gap-2">
          <span className="inline-flex h-6 w-6 items-center justify-center rounded-full bg-gradient-to-br from-brand-500 to-brand-700 text-xs font-semibold text-white">
            AI
          </span>
          <div>
            <div className="text-sm font-semibold text-gray-900">AI 辅导</div>
            <div className="text-xs text-gray-500">
              {cachedAt && status === 'idle'
                ? `已加载历史解析（${formatCachedAt(cachedAt)}）`
                : '实时生成参考思路，仅供学习参考'}
            </div>
          </div>
        </div>
        {renderButton()}
      </header>

      {status === 'error' ? (
        <p className="mt-3 rounded-lg border border-dashed border-rose-200 bg-rose-50/60 px-3 py-2 text-xs text-rose-700">
          {errorMsg || 'AI 解析失败，请稍后重试。'}
        </p>
      ) : null}

      {status === 'streaming' && !content ? (
        <div className="mt-3 flex items-center gap-2 text-xs text-gray-500">
          <span className="inline-block h-2 w-2 animate-pulse rounded-full bg-brand-500" />
          AI 正在思考……
        </div>
      ) : null}

      {content ? (
        <div className="mt-3 rounded-xl bg-gray-50 px-4 py-3 text-sm leading-7 text-gray-800">
          <MarkdownBlock content={content} />
          {status === 'streaming' ? (
            <span className="ml-0.5 inline-block h-4 w-1.5 translate-y-0.5 animate-pulse bg-gray-400 align-middle" />
          ) : null}
        </div>
      ) : null}

      {hasContent && status === 'idle' ? (
        <p className="mt-2 text-[11px] text-gray-400">AI 生成，仅供参考。可点"重新生成"再问一次。</p>
      ) : null}
    </section>
  );
}

export default AIExplanationPanel;
