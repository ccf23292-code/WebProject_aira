/**
 * components/problem/AIExplanationPanel.tsx
 * 题目 AI 解析面板（SSE 流式）
 *
 * 状态机：
 *   idle           初始 / 已生成完毕（可重新生成）
 *   streaming      正在接收 token
 *   error          流式过程中失败
 * 与服务端 routers/llm_controller.go 的 SSE 事件协议一一对齐。
 */

'use client';

import { useEffect, useRef, useState } from 'react';
import { streamExplain } from '@/lib/llm';
import { MarkdownBlock } from '@/components/Markdown';

type Status = 'idle' | 'streaming' | 'error';

interface Props {
  problemId: number;
}

export function AIExplanationPanel({ problemId }: Props) {
  const [content, setContent] = useState<string>('');
  const [status, setStatus] = useState<Status>('idle');
  const [errorMsg, setErrorMsg] = useState<string>('');
  const [hasGenerated, setHasGenerated] = useState(false);
  const controllerRef = useRef<AbortController | null>(null);

  // 卸载时取消未完成的流，防止内存泄漏 / setState on unmounted
  useEffect(() => {
    return () => {
      controllerRef.current?.abort();
    };
  }, []);

  const start = () => {
    // 上一轮还没结束就 abort
    controllerRef.current?.abort();
    setContent('');
    setErrorMsg('');
    setStatus('streaming');

    controllerRef.current = streamExplain(problemId, {
      onToken: (text) => {
        // 逐 token 追加：天然形成打字机效果，SSE 推一个 token 触发一次 setState
        setContent((prev) => prev + text);
      },
      onDone: () => {
        setStatus('idle');
        setHasGenerated(true);
        controllerRef.current = null;
      },
      onError: (message) => {
        setStatus('error');
        setErrorMsg(message);
        setHasGenerated(true);
        controllerRef.current = null;
      },
    });
  };

  const stop = () => {
    controllerRef.current?.abort();
    controllerRef.current = null;
    setStatus('idle');
    setHasGenerated(true);
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
    return (
      <button
        type="button"
        onClick={start}
        className="inline-flex items-center gap-1 rounded-lg bg-brand-600 px-3 py-1.5 text-xs font-medium text-white shadow-sm transition-colors hover:bg-brand-700"
      >
        {hasGenerated ? '重新生成' : '生成 AI 解析'}
      </button>
    );
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
            <div className="text-xs text-gray-500">实时生成参考思路，仅供学习参考</div>
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

      {hasGenerated && status === 'idle' ? (
        <p className="mt-2 text-[11px] text-gray-400">AI 生成，仅供参考。可点"重新生成"再问一次。</p>
      ) : null}
    </section>
  );
}

export default AIExplanationPanel;
