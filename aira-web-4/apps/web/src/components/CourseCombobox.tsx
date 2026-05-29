/**
 * components/CourseCombobox.tsx
 * 带搜索的课程选择器 — 替代原生 <select>。
 *
 * 场景：courses 表导入 zju 全量后有 2000+ 门课，原生下拉没法用。
 *
 * 功能：
 *   - 输入框 + 实时过滤（按 name / code / id 模糊匹配，不区分大小写）
 *   - 点选课程触发 onChange
 *   - 可选 allowNew：在列表底部显示"+ 新增课程"特殊项
 *   - 点击外部关闭面板；ESC 关闭
 *   - 滚动列表，max-height 限制避免占满屏幕
 */

'use client';

import { useEffect, useMemo, useRef, useState } from 'react';
import type { Course } from '@aira/shared';

interface CourseComboboxProps {
  /** 当前选中的 course_id；空串表示未选 */
  value: string;
  /** 点选时回调 */
  onChange: (id: string) => void;
  /** 全量课程数组，由父组件加载并传入 */
  courses: Course[];
  /** 是否在列表底部展示"+ 新增课程"项 */
  allowNew?: boolean;
  /** allowNew=true 时被选中后传给 onChange 的特殊值 */
  newValueSentinel?: string;
  /** 占位文案 */
  placeholder?: string;
  disabled?: boolean;
  /** 主题色：brand（蓝）/ amber（橙），用于聚焦边框区分 */
  tone?: 'brand' | 'amber';
}

const MAX_VISIBLE = 50; // 单次最多渲染 50 项，避免长列表卡顿

export function CourseCombobox({
  value,
  onChange,
  courses,
  allowNew = false,
  newValueSentinel = '__new__',
  placeholder = '选择课程',
  disabled = false,
  tone = 'brand',
}: CourseComboboxProps) {
  const [query, setQuery] = useState('');
  const [open, setOpen] = useState(false);
  const containerRef = useRef<HTMLDivElement | null>(null);
  const inputRef = useRef<HTMLInputElement | null>(null);

  // 用 id → Course 索引，方便显示当前选中的课程名
  const byId = useMemo(() => {
    const m = new Map<string, Course>();
    for (const c of courses) m.set(c.id, c);
    return m;
  }, [courses]);

  const selectedCourse = byId.get(value);
  const isNewSentinel = allowNew && value === newValueSentinel;

  // 过滤
  const filtered = useMemo(() => {
    const q = query.trim().toLowerCase();
    if (!q) return courses.slice(0, MAX_VISIBLE);
    const out: Course[] = [];
    for (const c of courses) {
      if (
        c.name.toLowerCase().includes(q) ||
        (c.code ?? '').toLowerCase().includes(q) ||
        c.id.toLowerCase().includes(q)
      ) {
        out.push(c);
        if (out.length >= MAX_VISIBLE) break;
      }
    }
    return out;
  }, [courses, query]);

  // 点外部关闭
  useEffect(() => {
    if (!open) return;
    function onDown(e: MouseEvent) {
      if (containerRef.current && !containerRef.current.contains(e.target as Node)) {
        setOpen(false);
      }
    }
    function onKey(e: KeyboardEvent) {
      if (e.key === 'Escape') setOpen(false);
    }
    document.addEventListener('mousedown', onDown);
    document.addEventListener('keydown', onKey);
    return () => {
      document.removeEventListener('mousedown', onDown);
      document.removeEventListener('keydown', onKey);
    };
  }, [open]);

  // 打开时聚焦搜索框
  useEffect(() => {
    if (open) inputRef.current?.focus();
  }, [open]);

  const ring = tone === 'amber' ? 'focus:border-amber-500' : 'focus:border-brand-500';

  function pick(id: string) {
    onChange(id);
    setOpen(false);
    setQuery('');
  }

  return (
    <div ref={containerRef} className="relative">
      {/* 触发器 — 显示当前选中 */}
      <button
        type="button"
        disabled={disabled}
        onClick={() => !disabled && setOpen((v) => !v)}
        className={`flex w-full items-center justify-between rounded-xl border border-gray-300 bg-white px-3 py-2 text-left text-sm outline-none disabled:bg-gray-100 ${ring}`}
      >
        <span className={`truncate ${selectedCourse || isNewSentinel ? 'text-gray-900' : 'text-gray-400'}`}>
          {isNewSentinel
            ? '+ 新增课程（需管理员审核）'
            : selectedCourse
              ? <>{selectedCourse.name}{selectedCourse.code ? <span className="text-gray-400"> · {selectedCourse.code}</span> : null}</>
              : placeholder}
        </span>
        <span className="ml-2 shrink-0 text-gray-400">▾</span>
      </button>

      {/* 面板 */}
      {open && (
        <div className="absolute left-0 right-0 top-full z-40 mt-1 max-h-80 overflow-hidden rounded-xl border border-gray-200 bg-white shadow-lg">
          <div className="border-b border-gray-100 p-2">
            <input
              ref={inputRef}
              type="text"
              value={query}
              onChange={(e) => setQuery(e.target.value)}
              placeholder="输入课程名 / 代码搜索..."
              className={`w-full rounded-lg border border-gray-200 px-2.5 py-1.5 text-sm outline-none ${ring}`}
            />
          </div>
          <ul className="max-h-60 overflow-y-auto py-1">
            {filtered.length === 0 ? (
              <li className="px-3 py-2 text-xs text-gray-500">没有匹配的课程</li>
            ) : (
              filtered.map((c) => (
                <li key={c.id}>
                  <button
                    type="button"
                    onClick={() => pick(c.id)}
                    className={`block w-full px-3 py-1.5 text-left text-sm hover:bg-brand-50 ${
                      c.id === value ? 'bg-brand-50/60 font-medium text-brand-700' : 'text-gray-800'
                    }`}
                  >
                    <div className="truncate">{c.name}</div>
                    <div className="truncate text-xs text-gray-500">
                      {c.code || c.id}{c.college ? ` · ${c.college}` : ''}
                    </div>
                  </button>
                </li>
              ))
            )}
            {/* 列表上限提示 */}
            {!query && courses.length > MAX_VISIBLE && (
              <li className="px-3 py-1 text-[10px] text-gray-400">
                共 {courses.length} 门课，未列完 — 用搜索精确查找
              </li>
            )}
          </ul>
          {allowNew && (
            <div className="border-t border-gray-100 p-1">
              <button
                type="button"
                onClick={() => pick(newValueSentinel)}
                className={`block w-full rounded-lg px-3 py-1.5 text-left text-sm hover:bg-amber-50 ${
                  isNewSentinel ? 'bg-amber-50 font-medium text-amber-700' : 'text-amber-700'
                }`}
              >
                + 新增课程（需管理员审核）
              </button>
            </div>
          )}
        </div>
      )}
    </div>
  );
}
