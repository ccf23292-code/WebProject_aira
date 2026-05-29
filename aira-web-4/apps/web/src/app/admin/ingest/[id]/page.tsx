/**
 * app/admin/ingest/[id]/page.tsx
 * 管理员审核详情：编辑课程 / 试卷 / parsed_json → 发布 or 拒绝
 *
 * 对接：
 *   GET   /api/admin/ingest/:id
 *   PATCH /api/admin/ingest/:id              { course_id?, new_course_name?, paper_name?, target_paper_id?, parsed_json? }
 *   POST  /api/admin/ingest/:id/publish
 *   POST  /api/admin/ingest/:id/reject       { reason }
 *   GET   /api/courses                       课程下拉
 *   GET   /api/courses/:id/papers            题解流程下：目标卷下拉
 */

'use client';

import Link from 'next/link';
import { useParams, useRouter } from 'next/navigation';
import { useEffect, useMemo, useState } from 'react';
import type {
  AdminIngestPatchDto,
  Course,
  IngestDedupMatch,
  IngestJob,
  IngestParsedEnvelope,
  Paper,
} from '@aira/shared';
import { PAPER_EXAM_TYPES, PAPER_SEMESTERS } from '@aira/shared';
import { CourseCombobox } from '@/components/CourseCombobox';
import { api } from '@/lib/api';
import { useAuth } from '@/lib/auth';

const CURRENT_YEAR = new Date().getFullYear();
const YEAR_OPTIONS = Array.from({ length: 11 }, (_, i) => CURRENT_YEAR - i);

function AdminDedupSection({ warnings }: { warnings: IngestDedupMatch[] }) {
  const bySeq = new Map<number, IngestDedupMatch[]>();
  for (const w of warnings) {
    const arr = bySeq.get(w.seq) ?? [];
    arr.push(w);
    bySeq.set(w.seq, arr);
  }
  return (
    <section className="rounded-3xl border border-amber-300 bg-amber-50/60 p-6 shadow-sm">
      <h2 className="mb-1 text-lg font-semibold text-amber-900">
        ⚠️ 查重提示 — 命中 {bySeq.size} 道题
      </h2>
      <p className="mb-4 text-xs text-amber-800">
        字符串相似度粗筛（n-gram + Jaccard ≥ 0.70）。审核时人工判断是否真的重复 —— 若是，建议拒绝本次上传或在 JSON 里手动删除重复条目。
      </p>
      <ul className="space-y-3">
        {[...bySeq.entries()].map(([seq, ms]) => (
          <li key={seq} className="rounded-2xl border border-amber-200 bg-white p-4">
            <div className="mb-2 text-sm font-medium text-amber-900">第 {seq} 题（新）</div>
            <div className="space-y-2">
              {ms.map((m, i) => (
                <div key={i} className="rounded-xl bg-amber-50/70 p-3 text-xs text-gray-800">
                  <div className="mb-1.5 flex items-center gap-2">
                    <span className="rounded-full bg-amber-200 px-2 py-0.5 font-mono text-amber-900">
                      {(m.similarity * 100).toFixed(0)}%
                    </span>
                    <span className="text-gray-600">
                      ↔ 《{m.paper_name}》#{m.problem_id}
                    </span>
                  </div>
                  <div className="grid gap-2 sm:grid-cols-2">
                    <div className="rounded-lg bg-white px-2 py-1.5">
                      <div className="text-[10px] uppercase tracking-wide text-gray-400">本次上传</div>
                      <div className="mt-0.5 text-gray-800">{m.new_snippet}</div>
                    </div>
                    <div className="rounded-lg bg-white px-2 py-1.5">
                      <div className="text-[10px] uppercase tracking-wide text-gray-400">已有题</div>
                      <div className="mt-0.5 text-gray-800">{m.existing_snippet}</div>
                    </div>
                  </div>
                </div>
              ))}
            </div>
          </li>
        ))}
      </ul>
    </section>
  );
}

export default function AdminIngestDetailPage() {
  const { id } = useParams<{ id: string }>();
  const router = useRouter();
  const { user, isLoggedIn, loading: authLoading } = useAuth();
  const isAdmin = !!user?.roles?.includes('admin');

  const [job, setJob] = useState<IngestJob | null>(null);
  const [courses, setCourses] = useState<Course[]>([]);
  const [papers, setPapers] = useState<Paper[]>([]);

  // 可编辑字段
  const [courseId, setCourseId] = useState('');
  const [newCourseName, setNewCourseName] = useState('');
  const [paperName, setPaperName] = useState('');
  const [year, setYear] = useState<string>('');
  const [semester, setSemester] = useState<string>('');
  const [examType, setExamType] = useState<string>('');
  const [targetPaperId, setTargetPaperId] = useState('');
  const [parsedText, setParsedText] = useState('');
  const [parsedError, setParsedError] = useState<string | null>(null);

  const [saving, setSaving] = useState(false);
  const [publishing, setPublishing] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [info, setInfo] = useState<string | null>(null);

  useEffect(() => {
    if (!isAdmin) return;
    void load();
    api.get<Course[]>('/courses').then(setCourses).catch(() => {});
  }, [isAdmin, id]);

  useEffect(() => {
    if (!courseId) {
      setPapers([]);
      return;
    }
    api
      .get<Paper[]>(`/courses/${encodeURIComponent(courseId)}/papers`)
      .then(setPapers)
      .catch(() => setPapers([]));
  }, [courseId]);

  async function load() {
    try {
      const j = await api.get<IngestJob>(`/admin/ingest/${id}`);
      setJob(j);
      setCourseId(j.course_id);
      setNewCourseName(j.new_course_name);
      setPaperName(j.paper_name);
      setYear(j.year ? String(j.year) : '');
      setSemester(j.semester ?? '');
      setExamType(j.exam_type ?? '');
      setTargetPaperId(j.target_paper_id ? String(j.target_paper_id) : '');
      setParsedText(j.parsed_json ? JSON.stringify(j.parsed_json, null, 2) : '');
      setError(null);
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : '加载失败');
    }
  }

  const parsedJsonValid = useMemo<{ ok: boolean; envelope?: IngestParsedEnvelope }>(() => {
    if (!parsedText.trim()) return { ok: false };
    try {
      const obj = JSON.parse(parsedText);
      if (!obj || typeof obj !== 'object' || !Array.isArray(obj.items)) {
        return { ok: false };
      }
      return { ok: true, envelope: obj as IngestParsedEnvelope };
    } catch {
      return { ok: false };
    }
  }, [parsedText]);

  async function handleSave() {
    if (!job) return;
    setSaving(true);
    setError(null);
    setInfo(null);
    setParsedError(null);
    try {
      const patch: AdminIngestPatchDto = {
        course_id: courseId,
        new_course_name: newCourseName,
        paper_name: paperName,
      };
      if (job.kind === 'question') {
        // 结构化三段也下发；后端会用它们重算 paper_name 并参与合并匹配
        if (year) patch.year = Number(year);
        if (semester) patch.semester = semester;
        if (examType) patch.exam_type = examType;
      }
      if (job.kind === 'explanation') {
        patch.target_paper_id = targetPaperId ? Number(targetPaperId) : undefined;
      }
      if (parsedText.trim()) {
        if (!parsedJsonValid.ok) {
          setParsedError('parsed_json 必须是 {"items":[...]} 形态的合法 JSON');
          return;
        }
        patch.parsed_json = parsedJsonValid.envelope;
      }
      const updated = await api.patch<IngestJob>(`/admin/ingest/${id}`, patch);
      setJob(updated);
      setInfo('已保存');
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : '保存失败');
    } finally {
      setSaving(false);
    }
  }

  async function handlePublish() {
    if (!job) return;
    if (!confirm('确认发布到正式题库？此操作会创建/更新对应的题目或题解。')) return;
    setPublishing(true);
    setError(null);
    setInfo(null);
    try {
      const updated = await api.post<IngestJob>(`/admin/ingest/${id}/publish`);
      setJob(updated);
      setInfo('已发布到题库');
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : '发布失败');
    } finally {
      setPublishing(false);
    }
  }

  async function handleReject() {
    if (!job) return;
    const reason = prompt('拒绝原因（可留空）：') ?? '';
    setError(null);
    setInfo(null);
    try {
      const updated = await api.post<IngestJob>(`/admin/ingest/${id}/reject`, { reason });
      setJob(updated);
      setInfo('已拒绝');
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : '拒绝失败');
    }
  }

  if (authLoading) return <div className="py-16 text-center text-gray-500">正在加载...</div>;
  if (!isLoggedIn) {
    return (
      <div className="rounded-2xl border border-gray-200 bg-white p-8 text-center text-gray-600">
        请先 <Link href="/login" className="text-brand-600 hover:underline">登录</Link>。
      </div>
    );
  }
  if (!isAdmin) {
    return (
      <div className="rounded-2xl border border-amber-200 bg-amber-50 p-8 text-center text-amber-800">
        该页面仅限管理员访问。
      </div>
    );
  }
  if (!job) {
    return error ? (
      <div className="rounded-xl border border-red-200 bg-red-50 p-4 text-sm text-red-700">
        {error}
        <div className="mt-2">
          <button onClick={() => router.back()} className="text-brand-600 hover:underline">
            ← 返回
          </button>
        </div>
      </div>
    ) : (
      <div className="py-16 text-center text-gray-500">加载中...</div>
    );
  }

  const editable = job.status === 'awaiting_review';

  return (
    <div className="space-y-6">
      <nav className="text-sm text-gray-500">
        <Link href="/admin/ingest" className="hover:text-brand-600">审核队列</Link>
        <span className="mx-2">/</span>
        <span className="font-medium text-gray-900">#{job.id}</span>
      </nav>

      <header className="flex flex-wrap items-center justify-between gap-3 rounded-3xl border border-gray-200 bg-white p-6 shadow-sm">
        <div>
          <div className="flex items-center gap-2">
            <span className="rounded-full bg-purple-100 px-2 py-0.5 text-xs text-purple-800">
              {job.status}
            </span>
            <span className="rounded-full bg-brand-50 px-2 py-0.5 text-xs text-brand-700">
              {job.kind === 'question' ? '题目' : '题解'}
            </span>
            <span className="text-xs text-gray-500">
              上传者 #{job.user_id} · 模型 {job.llm_model || '—'}
            </span>
          </div>
          <h1 className="mt-2 text-2xl font-semibold text-gray-900">{job.filename}</h1>
        </div>
        <div className="flex flex-wrap gap-2">
          <button
            onClick={handleReject}
            disabled={!editable}
            className="rounded-full border border-red-300 px-4 py-2 text-sm text-red-700 hover:bg-red-50 disabled:cursor-not-allowed disabled:opacity-50"
          >
            拒绝
          </button>
          <button
            onClick={handleSave}
            disabled={!editable || saving}
            className="rounded-full border border-gray-300 px-4 py-2 text-sm text-gray-700 hover:bg-gray-50 disabled:cursor-not-allowed disabled:opacity-50"
          >
            {saving ? '保存中...' : '保存编辑'}
          </button>
          <button
            onClick={handlePublish}
            disabled={!editable || publishing || !parsedJsonValid.ok}
            className="rounded-full bg-brand-600 px-4 py-2 text-sm font-medium text-white shadow hover:bg-brand-700 disabled:cursor-not-allowed disabled:bg-gray-300"
          >
            {publishing ? '发布中...' : '发布到题库'}
          </button>
        </div>
      </header>

      {(error || info || job.error_message) && (
        <div className="space-y-2">
          {error && <div className="rounded-xl border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-700">{error}</div>}
          {info && <div className="rounded-xl border border-green-200 bg-green-50 px-3 py-2 text-sm text-green-700">{info}</div>}
          {job.error_message && (
            <div className="rounded-xl border border-amber-200 bg-amber-50 px-3 py-2 text-sm text-amber-800">
              任务历史错误：{job.error_message}
            </div>
          )}
        </div>
      )}

      {/* 元信息编辑 */}
      <section className="rounded-3xl border border-gray-200 bg-white p-6 shadow-sm">
        <h2 className="mb-4 text-lg font-semibold text-gray-900">归属与试卷</h2>
        <div className="grid gap-4 sm:grid-cols-2">
          <div>
            <label className="mb-1 block text-sm font-medium text-gray-800">课程</label>
            <CourseCombobox
              value={courseId}
              onChange={setCourseId}
              courses={courses}
              placeholder="搜索课程名或代码"
              disabled={!editable}
            />
            {job.new_course_name && !courseId && (
              <p className="mt-1 text-xs text-amber-700">
                上传者建议新课程名：「{job.new_course_name}」 — 若为新课，请在下方填写后发布将自动创建。
              </p>
            )}
          </div>
          <div>
            <label className="mb-1 block text-sm font-medium text-gray-800">
              新课程名（若需新建）
            </label>
            <input
              value={newCourseName}
              onChange={(e) => setNewCourseName(e.target.value)}
              disabled={!editable}
              className="w-full rounded-xl border border-gray-300 px-3 py-2 text-sm disabled:bg-gray-100"
            />
          </div>

          {job.kind === 'question' ? (
            <>
              <div className="sm:col-span-2">
                <label className="mb-1 block text-sm font-medium text-gray-800">
                  试卷结构化标识（决定合并目标）
                </label>
                <div className="grid gap-2 sm:grid-cols-3">
                  <select
                    value={year}
                    onChange={(e) => setYear(e.target.value)}
                    disabled={!editable}
                    className="rounded-xl border border-gray-300 px-3 py-2 text-sm disabled:bg-gray-100"
                  >
                    <option value="">年份</option>
                    {YEAR_OPTIONS.map((y) => (
                      <option key={y} value={String(y)}>{y}</option>
                    ))}
                  </select>
                  <select
                    value={semester}
                    onChange={(e) => setSemester(e.target.value)}
                    disabled={!editable}
                    className="rounded-xl border border-gray-300 px-3 py-2 text-sm disabled:bg-gray-100"
                  >
                    <option value="">学期</option>
                    {PAPER_SEMESTERS.map((s) => (
                      <option key={s} value={s}>{s}</option>
                    ))}
                  </select>
                  <select
                    value={examType}
                    onChange={(e) => setExamType(e.target.value)}
                    disabled={!editable}
                    className="rounded-xl border border-gray-300 px-3 py-2 text-sm disabled:bg-gray-100"
                  >
                    <option value="">考试类型</option>
                    {PAPER_EXAM_TYPES.map((t) => (
                      <option key={t} value={t}>{t}</option>
                    ))}
                  </select>
                </div>
                <p className="mt-1 text-xs text-gray-500">
                  三段都填后，发布时若该课程下已存在 (年份/学期/考试类型) 相同的试卷，新题会合并到那份；否则新建。
                </p>
              </div>
              <div className="sm:col-span-2">
                <label className="mb-1 block text-sm font-medium text-gray-800">
                  试卷显示名（自动由上面三段合成，可手工覆写）
                </label>
                <input
                  value={paperName}
                  onChange={(e) => setPaperName(e.target.value)}
                  disabled={!editable}
                  placeholder={year && semester && examType ? `${year} ${semester}${examType}` : ''}
                  className="w-full rounded-xl border border-gray-300 px-3 py-2 text-sm disabled:bg-gray-100"
                />
              </div>
            </>
          ) : (
            <div className="sm:col-span-2">
              <label className="mb-1 block text-sm font-medium text-gray-800">目标试卷</label>
              <select
                value={targetPaperId}
                onChange={(e) => setTargetPaperId(e.target.value)}
                disabled={!editable || !courseId}
                className="w-full rounded-xl border border-gray-300 px-3 py-2 text-sm disabled:bg-gray-100"
              >
                <option value="">未选择</option>
                {papers.map((p) => (
                  <option key={p.id} value={p.id}>{p.name}</option>
                ))}
              </select>
            </div>
          )}
        </div>
      </section>

      {/* 查重提示 — admin 视角带 side-by-side 对比 */}
      {Array.isArray(job.dedup_warnings) && job.dedup_warnings.length > 0 && (
        <AdminDedupSection warnings={job.dedup_warnings} />
      )}

      {/* JSON 编辑 */}
      <section className="rounded-3xl border border-gray-200 bg-white p-6 shadow-sm">
        <div className="mb-3 flex items-center justify-between">
          <h2 className="text-lg font-semibold text-gray-900">AI 解析结果（可编辑）</h2>
          <span className={`text-xs ${parsedJsonValid.ok ? 'text-green-700' : 'text-red-700'}`}>
            {parsedJsonValid.ok ? '✓ JSON 合法' : '✗ JSON 不合法'}
          </span>
        </div>
        <textarea
          value={parsedText}
          onChange={(e) => setParsedText(e.target.value)}
          disabled={!editable}
          spellCheck={false}
          className="h-[60vh] w-full rounded-xl border border-gray-300 bg-gray-50 p-4 font-mono text-xs disabled:bg-gray-100"
        />
        {parsedError && (
          <p className="mt-2 text-xs text-red-700">{parsedError}</p>
        )}
        <p className="mt-2 text-xs text-gray-500">
          需要保持 {`{"items":[...]}`} 结构。题目要求 items 中每条带 sequence_id / question_type / test
          / options / answer / explanation / difficulty / tags；题解要求 sequence_id / content_md。
        </p>
      </section>

      {/* 原文预览 */}
      {job.raw_text && (
        <section className="rounded-3xl border border-gray-200 bg-white p-6 shadow-sm">
          <h2 className="mb-3 text-lg font-semibold text-gray-900">原文提取（只读）</h2>
          <pre className="max-h-[40vh] overflow-auto whitespace-pre-wrap rounded-xl bg-gray-50 p-4 text-xs text-gray-800">
{job.raw_text}
          </pre>
        </section>
      )}
    </div>
  );
}
