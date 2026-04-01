/**
 * components/layout/StateDisplay.tsx
 * 页面状态组件 — 错误 / 空结果 / 通用提示
 */

'use client';

/** 错误状态 — 请求失败时展示 */
export function ErrorState({
  message = '加载失败，请稍后重试',
  onRetry,
}: {
  message?: string;
  onRetry?: () => void;
}) {
  return (
    <div className="flex flex-col items-center justify-center py-16 text-center">
      <div className="mb-3 text-4xl">⚠️</div>
      <p className="text-sm text-gray-500 mb-4">{message}</p>
      {onRetry && (
        <button
          onClick={onRetry}
          className="rounded-md bg-brand-600 px-4 py-1.5 text-sm text-white
                     transition-colors hover:bg-brand-700"
        >
          重试
        </button>
      )}
    </div>
  );
}

/** 空状态 — 列表无数据时展示 */
export function EmptyState({
  title = '暂无数据',
  description,
}: {
  title?: string;
  description?: string;
}) {
  return (
    <div className="flex flex-col items-center justify-center py-16 text-center">
      <div className="mb-3 text-4xl">📭</div>
      <p className="text-sm font-medium text-gray-600">{title}</p>
      {description && (
        <p className="mt-1 text-xs text-gray-400">{description}</p>
      )}
    </div>
  );
}
