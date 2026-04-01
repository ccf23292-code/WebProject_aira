/**
 * lib/mock.ts
 * Mock 数据 — 字段严格对齐后端 API 响应结构
 * 后端就绪后删除此文件，页面中的 mockDelay() 改为 api.get()
 */

import type { Course, Paper, Problem, FavoriteItem } from '@aira/shared';

/** GET /api/courses */
export const MOCK_COURSES: Course[] = [
  { id: 'course-101', name: '高等数学', description: '2026春夏学期高等数学课程资料' },
  { id: 'course-102', name: '线性代数', description: '2026春夏学期线性代数课程资料' },
  { id: 'course-201', name: '数据结构', description: '2026春夏学期数据结构课程资料' },
  { id: 'course-202', name: '操作系统', description: '2025秋冬学期操作系统课程资料' },
  { id: 'course-301', name: '概率论与数理统计', description: '2026春夏学期概率统计课程资料' },
  { id: 'course-302', name: '离散数学', description: '2025秋冬学期离散数学课程资料' },
];

/** GET /api/courses/{course_id}/papers */
export const MOCK_PAPERS: Record<string, Paper[]> = {
  'course-101': [
    { id: 1, course_id: 'course-101', name: '2026-spring-summer 期末卷', created_at: '2026-06-20T10:00:00Z' },
    { id: 2, course_id: 'course-101', name: '2025-autumn-winter 期末卷', created_at: '2026-01-15T10:00:00Z' },
    { id: 3, course_id: 'course-101', name: '2025-spring-summer 期中卷', created_at: '2025-04-20T10:00:00Z' },
  ],
  'course-102': [
    { id: 4, course_id: 'course-102', name: '2026-spring-summer 期末卷', created_at: '2026-06-22T10:00:00Z' },
  ],
  'course-201': [
    { id: 5, course_id: 'course-201', name: '2026-spring-summer 期末卷', created_at: '2026-06-25T10:00:00Z' },
    { id: 6, course_id: 'course-201', name: '2025-autumn-winter 期末卷', created_at: '2026-01-10T10:00:00Z' },
  ],
  'course-202': [
    { id: 7, course_id: 'course-202', name: '2025-autumn-winter 期末卷', created_at: '2026-01-12T10:00:00Z' },
  ],
  'course-301': [
    { id: 8, course_id: 'course-301', name: '2026-spring-summer 期末卷', created_at: '2026-06-18T10:00:00Z' },
  ],
  'course-302': [
    { id: 9, course_id: 'course-302', name: '2025-autumn-winter 期末卷', created_at: '2026-01-08T10:00:00Z' },
  ],
};

/** GET /api/papers/{paper_id}/problems */
export const MOCK_PROBLEMS: Record<number, Problem[]> = {
  1: [
    {
      id: 1001, testpaper_id: 1, order: 1,
      test: '函数 f(x) = x² 的导数是？',
      options: [
        { option: 'A', text: 'x' },
        { option: 'B', text: '2x' },
        { option: 'C', text: 'x²' },
        { option: 'D', text: '2x²' },
      ],
      answer: 'B',
    },
    {
      id: 1002, testpaper_id: 1, order: 2,
      test: '∫ 2x dx 的结果是？',
      options: [
        { option: 'A', text: 'x² + C' },
        { option: 'B', text: '2x² + C' },
        { option: 'C', text: 'x + C' },
        { option: 'D', text: '2 + C' },
      ],
      answer: 'A',
    },
    {
      id: 1003, testpaper_id: 1, order: 3,
      test: 'lim(x→0) sin(x)/x 的值是？',
      options: [
        { option: 'A', text: '0' },
        { option: 'B', text: '1' },
        { option: 'C', text: '∞' },
        { option: 'D', text: '不存在' },
      ],
      answer: 'B',
    },
    {
      id: 1004, testpaper_id: 1, order: 4,
      test: '函数 f(x) = eˣ 的不定积分是？',
      options: [
        { option: 'A', text: 'eˣ + C' },
        { option: 'B', text: 'xeˣ + C' },
        { option: 'C', text: 'eˣ⁺¹/(x+1) + C' },
        { option: 'D', text: 'ln(x) + C' },
      ],
      answer: 'A',
    },
  ],
  5: [
    {
      id: 2001, testpaper_id: 5, order: 1,
      test: '在一个具有 n 个结点的二叉链表中，所有结点的空指针域个数为？',
      options: [
        { option: 'A', text: 'n - 1' },
        { option: 'B', text: 'n' },
        { option: 'C', text: 'n + 1' },
        { option: 'D', text: '2n' },
      ],
      answer: 'C',
    },
    {
      id: 2002, testpaper_id: 5, order: 2,
      test: '快速排序的平均时间复杂度是？',
      options: [
        { option: 'A', text: 'O(n)' },
        { option: 'B', text: 'O(n log n)' },
        { option: 'C', text: 'O(n²)' },
        { option: 'D', text: 'O(log n)' },
      ],
      answer: 'B',
    },
  ],
};

/** GET /api/favorites */
export const MOCK_FAVORITES: FavoriteItem[] = [
  {
    favorite_id: 55,
    problem_id: 1001,
    added_at: '2026-03-16T12:00:00Z',
    problem_details: {
      testpaper_name: '2026-spring-summer 期末卷',
      order: 1,
      test: '函数 f(x) = x² 的导数是？',
    },
  },
  {
    favorite_id: 56,
    problem_id: 2001,
    added_at: '2026-03-15T09:30:00Z',
    problem_details: {
      testpaper_name: '2026-spring-summer 期末卷',
      order: 1,
      test: '在一个具有 n 个结点的二叉链表中，所有结点的空指针域个数为？',
    },
  },
];
