/**
 * components/layout/Skeleton.tsx
 * 骨架屏组件 — 数据加载中的占位 UI
 *
 * 提供三种预设：
 *   <TableSkeleton rows={6} />  — 表格行占位
 *   <DetailSkeleton />          — 详情页占位
 *   <Shimmer />                 — 基础闪烁条
 */

'use client';

/** 基础闪烁条 */
export function Shimmer({ className = '' }: { className?: string }) {
  return (
    <div
      className={`animate-pulse rounded bg-gray-200 ${className}`}
    />
  );
}

/** 表格骨架屏 — 用于题目列表 / 提交记录列表 */
export function TableSkeleton({ rows = 5 }: { rows?: number }) {
  return (
    <div className="rounded-lg border border-gray-200 bg-white overflow-hidden">
      {/* 表头 */}
      <div className="flex gap-4 border-b border-gray-100 bg-gray-50 px-4 py-3">
        <Shimmer className="h-3 w-8" />
        <Shimmer className="h-3 w-32" />
        <Shimmer className="h-3 w-12" />
        <Shimmer className="h-3 w-20 ml-auto" />
      </div>
      {/* 行 */}
      {Array.from({ length: rows }).map((_, i) => (
        <div
          key={i}
          className="flex items-center gap-4 border-b border-gray-50 px-4 py-4"
        >
          <Shimmer className="h-3 w-6" />
          <Shimmer className="h-3 w-40" />
          <Shimmer className="h-5 w-12 rounded" />
          <div className="flex gap-1 flex-1">
            <Shimmer className="h-4 w-10 rounded" />
            <Shimmer className="h-4 w-14 rounded" />
          </div>
          <Shimmer className="h-3 w-12 ml-auto" />
        </div>
      ))}
    </div>
  );
}

/** 详情页骨架屏 — 用于题目详情 / 提交详情 */
export function DetailSkeleton() {
  return (
    <div className="space-y-6 animate-pulse">
      {/* 面包屑 */}
      <div className="flex gap-2">
        <Shimmer className="h-3 w-12" />
        <Shimmer className="h-3 w-24" />
      </div>

      {/* 标题区 */}
      <div className="flex items-center gap-3">
        <Shimmer className="h-6 w-48" />
        <Shimmer className="h-5 w-12 rounded" />
      </div>

      {/* 左右两栏 */}
      <div className="grid gap-6 lg:grid-cols-2">
        {/* 左栏 */}
        <div className="space-y-4">
          <div className="rounded-lg border border-gray-200 bg-white p-5 space-y-3">
            <Shimmer className="h-4 w-20" />
            <Shimmer className="h-3 w-full" />
            <Shimmer className="h-3 w-full" />
            <Shimmer className="h-3 w-3/4" />
            <Shimmer className="h-4 w-20 mt-4" />
            <Shimmer className="h-3 w-full" />
            <Shimmer className="h-3 w-2/3" />
          </div>
          {/* 样例 */}
          <div className="rounded-lg border border-gray-200 bg-white p-4 space-y-2">
            <Shimmer className="h-3 w-16" />
            <div className="grid grid-cols-2 gap-4">
              <Shimmer className="h-12 w-full rounded" />
              <Shimmer className="h-12 w-full rounded" />
            </div>
          </div>
        </div>

        {/* 右栏 */}
        <div className="space-y-4">
          <div className="flex justify-between">
            <Shimmer className="h-8 w-24 rounded-md" />
            <Shimmer className="h-8 w-16 rounded-md" />
          </div>
          <Shimmer className="h-72 w-full rounded-lg" />
        </div>
      </div>
    </div>
  );
}
