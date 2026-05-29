/**
 * app/upload/page.tsx
 * 上传中心 — 用户提交题目 / 题解原文件，后端 LLM 清洗后进入审核队列
 *
 * 对接：
 *   POST /api/ingest/upload       multipart 上传，新建 IngestJob
 *   GET  /api/courses             课程下拉
 *   GET  /api/courses/:id/papers  题解 Tab 下试卷下拉
 */

'use client';

import Link from 'next/link';
import { useRouter, useSearchParams } from 'next/navigation';
import { useEffect, useMemo, useState } from 'react';
import type { Course, IngestJob, Paper } from '@aira/shared';
import { PAPER_EXAM_TYPES, PAPER_SEMESTERS } from '@aira/shared';
import { CourseCombobox } from '@/components/CourseCombobox';
import { api } from '@/lib/api';
import { useAuth } from '@/lib/auth';

type Tab = 'question' | 'explanation';

const ACCEPT = '.md,.markdown,.txt,.pdf,.docx,.jpg,.jpeg,.png';
const MAX_SIZE_MB = 20;
const NEW_COURSE_VALUE = '__new__';

const CURRENT_YEAR = new Date().getFullYear();
// 年份下拉只给最近 10 年 + 当年；够覆盖在校生关心的真题范围
const YEAR_OPTIONS = Array.from({ length: 11 }, (_, i) => CURRENT_YEAR - i);

export default function UploadPage() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const { isLoggedIn, loading: authLoading } = useAuth();
  const [tab, setTab] = useState<Tab>('question');

  // 表单字段。从 ?courseId=xxx 预填课程（例如从课程详情页"上传文件"按钮跳过来）。
  const [courseId, setCourseId] = useState<string>(() => searchParams.get('courseId') ?? '');
  const [newCourseName, setNewCourseName] = useState<string>('');
  // 结构化试卷命名三段
  const [year, setYear] = useState<string>('');
  const [semester, setSemester] = useState<string>('');
  const [examType, setExamType] = useState<string>('');
  const [targetPaperId, setTargetPaperId] = useState<string>('');
  const [file, setFile] = useState<File | null>(null);

  // 远端数据
  const [courses, setCourses] = useState<Course[]>([]);
  const [papers, setPapers] = useState<Paper[]>([]);
  const [coursesLoaded, setCoursesLoaded] = useState(false);

  // 提交状态
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!isLoggedIn) return;
    api.get<Course[]>('/courses')
      .then((list) => {
        setCourses(list);
        setCoursesLoaded(true);
      })
      .catch((err) => setError(`课程列表加载失败：${err.message}`));
  }, [isLoggedIn]);

  // 切换课程或切换到题解 Tab 时，刷新该课程的试卷下拉
  useEffect(() => {
    setPapers([]);
    setTargetPaperId('');
    if (tab !== 'explanation') return;
    if (!courseId || courseId === NEW_COURSE_VALUE) return;
    api.get<Paper[]>(`/courses/${encodeURIComponent(courseId)}/papers`)
      .then(setPapers)
      .catch(() => setPapers([]));
  }, [courseId, tab]);

  const isNewCourse = courseId === NEW_COURSE_VALUE;

  const composedPaperName = useMemo(() => {
    if (!year || !semester || !examType) return '';
    return `${year} ${semester}${examType}`;
  }, [year, semester, examType]);

  const canSubmit = useMemo(() => {
    if (!file) return false;
    if (!courseId) return false;
    if (isNewCourse && !newCourseName.trim()) return false;
    if (tab === 'question' && !(year && semester && examType)) return false;
    if (tab === 'explanation' && !targetPaperId) return false;
    return true;
  }, [file, courseId, isNewCourse, newCourseName, tab, year, semester, examType, targetPaperId]);

  function resetForm() {
    setCourseId('');
    setNewCourseName('');
    setYear('');
    setSemester('');
    setExamType('');
    setTargetPaperId('');
    setFile(null);
  }

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (!canSubmit || !file) return;

    if (file.size > MAX_SIZE_MB * 1024 * 1024) {
      setError(`文件大小不能超过 ${MAX_SIZE_MB}MB`);
      return;
    }

    setSubmitting(true);
    setError(null);
    try {
      const fd = new FormData();
      fd.append('kind', tab);
      if (isNewCourse) {
        fd.append('new_course_name', newCourseName.trim());
      } else {
        fd.append('course_id', courseId);
      }
      if (tab === 'question') {
        fd.append('year', year);
        fd.append('semester', semester);
        fd.append('exam_type', examType);
      } else {
        fd.append('target_paper_id', targetPaperId);
      }
      fd.append('file', file);

      const job = await api.upload<IngestJob>('/ingest/upload', fd);
      resetForm();
      router.push(`/upload/jobs/${job.id}`);
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : '上传失败');
    } finally {
      setSubmitting(false);
    }
  }

  if (authLoading) {
    return <div className="py-16 text-center text-gray-500">正在加载...</div>;
  }
  if (!isLoggedIn) {
    return (
      <div className="rounded-2xl border border-gray-200 bg-white p-8 text-center text-gray-600">
        请先 <Link href="/login" className="text-brand-600 hover:underline">登录</Link> 再使用上传功能。
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <nav className="flex flex-wrap items-center gap-2 text-sm text-gray-500">
        <Link href="/upload" className="transition-colors hover:text-brand-600">上传中心</Link>
        <span>/</span>
        <span className="font-medium text-gray-900">上传文件清洗</span>
      </nav>

      <header className="space-y-2">
        <h1 className="text-3xl font-semibold tracking-tight text-gray-900">上传文件清洗</h1>
        <p className="text-sm text-gray-600">
          支持 PDF / DOCX / Markdown / 图片，AI 自动结构化后进入审核队列。
          <Link href="/upload/jobs" className="ml-2 text-brand-600 hover:underline">查看我的上传记录</Link>
        </p>
      </header>

      {/* Tabs */}
      <div className="inline-flex rounded-full border border-gray-200 bg-white p-1 text-sm">
        {(['question', 'explanation'] as Tab[]).map((t) => (
          <button
            key={t}
            type="button"
            onClick={() => setTab(t)}
            className={`rounded-full px-4 py-1.5 transition-colors ${
              tab === t ? 'bg-brand-600 text-white shadow' : 'text-gray-600 hover:text-brand-700'
            }`}
          >
            {t === 'question' ? '上传题目' : '上传题解'}
          </button>
        ))}
      </div>

      <form
        onSubmit={handleSubmit}
        className="space-y-5 rounded-3xl border border-gray-200 bg-white p-6 shadow-sm"
      >
        {/* 课程选择 */}
        <div>
          <label className="mb-1 block text-sm font-medium text-gray-800">课程</label>
          <CourseCombobox
            value={courseId}
            onChange={setCourseId}
            courses={courses}
            placeholder={coursesLoaded ? '搜索课程名或代码' : '加载中...'}
            allowNew
            newValueSentinel={NEW_COURSE_VALUE}
          />
        </div>

        {isNewCourse && (
          <div>
            <label className="mb-1 block text-sm font-medium text-gray-800">新课程名</label>
            <input
              value={newCourseName}
              onChange={(e) => setNewCourseName(e.target.value)}
              placeholder="例如：数据结构与算法基础"
              className="w-full rounded-xl border border-gray-300 px-3 py-2 text-sm focus:border-brand-500 focus:outline-none"
            />
          </div>
        )}

        {/* 题目流程：结构化试卷命名（年份 + 学期 + 考试类型） */}
        {tab === 'question' && (
          <div>
            <label className="mb-1 block text-sm font-medium text-gray-800">试卷标识</label>
            <div className="grid gap-2 sm:grid-cols-3">
              <select
                value={year}
                onChange={(e) => setYear(e.target.value)}
                className="rounded-xl border border-gray-300 px-3 py-2 text-sm focus:border-brand-500 focus:outline-none"
              >
                <option value="">年份</option>
                {YEAR_OPTIONS.map((y) => (
                  <option key={y} value={String(y)}>{y}</option>
                ))}
              </select>
              <select
                value={semester}
                onChange={(e) => setSemester(e.target.value)}
                className="rounded-xl border border-gray-300 px-3 py-2 text-sm focus:border-brand-500 focus:outline-none"
              >
                <option value="">学期</option>
                {PAPER_SEMESTERS.map((s) => (
                  <option key={s} value={s}>{s}</option>
                ))}
              </select>
              <select
                value={examType}
                onChange={(e) => setExamType(e.target.value)}
                className="rounded-xl border border-gray-300 px-3 py-2 text-sm focus:border-brand-500 focus:outline-none"
              >
                <option value="">考试类型</option>
                {PAPER_EXAM_TYPES.map((t) => (
                  <option key={t} value={t}>{t}</option>
                ))}
              </select>
            </div>
            <p className="mt-1 text-xs text-gray-500">
              {composedPaperName
                ? <>试卷名将为 <span className="font-medium text-gray-800">"{composedPaperName}"</span>，相同的会自动并入同一份试卷。</>
                : <>三段决定试卷的唯一身份；填齐后相同的会自动合并，避免重复试卷。</>}
            </p>
          </div>
        )}

        {/* 题解流程：选择目标试卷 */}
        {tab === 'explanation' && (
          <div>
            <label className="mb-1 block text-sm font-medium text-gray-800">挂到哪份试卷</label>
            <select
              value={targetPaperId}
              onChange={(e) => setTargetPaperId(e.target.value)}
              disabled={!courseId || isNewCourse}
              className="w-full rounded-xl border border-gray-300 px-3 py-2 text-sm disabled:bg-gray-100 focus:border-brand-500 focus:outline-none"
            >
              <option value="">
                {!courseId || isNewCourse ? '请先选择已有课程' : papers.length ? '请选择试卷' : '该课程下暂无试卷'}
              </option>
              {papers.map((p) => (
                <option key={p.id} value={p.id}>{p.name}</option>
              ))}
            </select>
            <p className="mt-1 text-xs text-gray-500">
              题解会按"第几题"自动匹配到该试卷的题目上。
            </p>
          </div>
        )}

        {/* 文件 */}
        <div>
          <label className="mb-1 block text-sm font-medium text-gray-800">文件</label>
          <input
            type="file"
            accept={ACCEPT}
            onChange={(e) => setFile(e.target.files?.[0] ?? null)}
            className="block w-full text-sm text-gray-700 file:mr-4 file:rounded-full file:border-0 file:bg-brand-50 file:px-4 file:py-2 file:text-sm file:font-medium file:text-brand-700 hover:file:bg-brand-100"
          />
          <p className="mt-1 text-xs text-gray-500">
            支持 PDF / DOCX / Markdown / TXT / JPG / PNG，单文件 ≤ {MAX_SIZE_MB}MB。
            图片类型走 OCR 模型识别，速度较慢。
          </p>
          {file && (
            <p className="mt-1 text-xs text-brand-700">
              已选择：{file.name}（{(file.size / 1024).toFixed(1)} KB）
            </p>
          )}
        </div>

        {error && (
          <div className="rounded-xl border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-700">
            {error}
          </div>
        )}

        <div className="flex items-center justify-end gap-3">
          <Link
            href="/upload/jobs"
            className="rounded-full border border-gray-300 px-4 py-2 text-sm text-gray-700 hover:bg-gray-50"
          >
            我的上传记录
          </Link>
          <button
            type="submit"
            disabled={!canSubmit || submitting}
            className="rounded-full bg-brand-600 px-5 py-2 text-sm font-medium text-white shadow transition-colors hover:bg-brand-700 disabled:cursor-not-allowed disabled:bg-gray-300"
          >
            {submitting ? '上传中...' : '提交并开始清洗'}
          </button>
        </div>
      </form>
    </div>
  );
}
