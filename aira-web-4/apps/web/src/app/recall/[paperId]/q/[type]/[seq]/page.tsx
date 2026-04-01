/**
 * app/recall/[paperId]/q/[type]/[seq]/page.tsx
 * page2 — 同一题号下的所有题目版本
 *
 * 功能：
 *   - 按支持度倒序展示所有版本
 *   - 每个版本可投支持票（每人每题一次）
 *   - 每个版本可二次编辑
 *   - 每个版本有评论区
 *
 * 对接:
 *   GET   /api/recall/papers/:paper_id/questions?question_type=x&sequence=y
 *   POST  /api/recall/questions/:question_id/support
 *   PATCH /api/recall/questions/:question_id
 *   GET   /api/recall/questions/:question_id/comments?page=1&size=10
 *   POST  /api/recall/questions/:question_id/comments
 */

'use client';

import { useState, useCallback } from 'react';
import { useParams } from 'next/navigation';
import Link from 'next/link';
import type { RecallQuestion, RecallComment, RecallCommentListData, PatchRecallQuestionDto } from '@aira/shared';
import { DetailSkeleton } from '@/components/layout/Skeleton';
import { ErrorState, EmptyState } from '@/components/layout/StateDisplay';
import { useFetch } from '@/hooks/useFetch';
import { api } from '@/lib/api';
import { useAuth } from '@/lib/auth';

const TYPE_LABELS: Record<string, string> = {
  singleChoice: '单选题', multiChoice: '多选题', fillBlank: '填空题',
  shortAnswer: '简答题', calculation: '计算题', proof: '证明题', other: '其他',
};

export default function QuestionVersionsPage() {
  const { paperId, type, seq } = useParams<{ paperId: string; type: string; seq: string }>();

  // 获取该题号下所有版本
  const { data: versions, loading, error, refetch } = useFetch(
    () => api.get<RecallQuestion[]>(
      `/recall/papers/${paperId}/questions?question_type=${type}&sequence=${seq}`
    ),
    [paperId, type, seq],
  );

  // 当前展开评论的题目 ID
  const [expandedCommentId, setExpandedCommentId] = useState<number | null>(null);
  // 当前正在编辑的题目 ID
  const [editingId, setEditingId] = useState<number | null>(null);

  if (loading) return <DetailSkeleton />;
  if (error) return <ErrorState message={error} onRetry={refetch} />;

  return (
    <div>
      {/* 面包屑 */}
      <nav className="mb-4 text-sm text-gray-500">
        <Link href={`/recall/${paperId}`} className="transition-colors hover:text-brand-600">
          回忆卷 #{paperId}
        </Link>
        <span className="mx-2">›</span>
        <span className="font-medium text-gray-900">
          {TYPE_LABELS[type] ?? type} 第 {seq} 题
        </span>
      </nav>

      <div className="mb-6">
        <h1 className="text-xl font-semibold text-gray-900">
          {TYPE_LABELS[type] ?? type} · 第 {seq} 题
        </h1>
        <p className="mt-1 text-sm text-gray-500">
          共 {versions?.length ?? 0} 个版本，按支持度排序
        </p>
      </div>

      {versions && versions.length > 0 ? (
        <div className="space-y-5">
          {versions.map((q, idx) => (
            <VersionCard
              key={q.id}
              question={q}
              rank={idx + 1}
              isEditing={editingId === q.id}
              onStartEdit={() => setEditingId(q.id)}
              onCancelEdit={() => setEditingId(null)}
              onEdited={() => { setEditingId(null); refetch(); }}
              showComments={expandedCommentId === q.id}
              onToggleComments={() =>
                setExpandedCommentId(expandedCommentId === q.id ? null : q.id)
              }
              onSupported={refetch}
            />
          ))}
        </div>
      ) : (
        <EmptyState title="该题号暂无内容" description="返回上一页添加题目" />
      )}
    </div>
  );
}

/* ══════════ 单个版本卡片 ══════════ */

function VersionCard({
  question: q, rank, isEditing, onStartEdit, onCancelEdit, onEdited,
  showComments, onToggleComments, onSupported,
}: {
  question: RecallQuestion;
  rank: number;
  isEditing: boolean;
  onStartEdit: () => void;
  onCancelEdit: () => void;
  onEdited: () => void;
  showComments: boolean;
  onToggleComments: () => void;
  onSupported: () => void;
}) {
  const { isLoggedIn } = useAuth();
  const [supporting, setSupporting] = useState(false);

  /** 投支持票 — POST /api/recall/questions/:id/support */
  const handleSupport = useCallback(async () => {
    setSupporting(true);
    try {
      await api.post(`/recall/questions/${q.id}/support`);
      onSupported();
    } catch (err) {
      alert(err instanceof Error ? err.message : '操作失败');
    } finally {
      setSupporting(false);
    }
  }, [q.id, onSupported]);

  return (
    <div className={`rounded-xl border bg-white overflow-hidden ${
      rank === 1 ? 'border-green-200' : 'border-gray-200'
    }`}>
      {/* 头部 */}
      <div className={`flex items-center justify-between px-5 py-3 ${
        rank === 1 ? 'bg-green-50' : 'bg-gray-50'
      }`}>
        <div className="flex items-center gap-3">
          <span className={`flex h-6 w-6 items-center justify-center rounded-full text-xs font-bold ${
            rank === 1 ? 'bg-green-500 text-white' : 'bg-gray-200 text-gray-500'
          }`}>
            {rank}
          </span>
          <span className="text-xs text-gray-400">
            上传者: {q.source_user_id}
            {q.last_editor_id && ` · 最后编辑: ${q.last_editor_id}`}
          </span>
        </div>

        <div className="flex items-center gap-2">
          {/* 支持度 */}
          {isLoggedIn && (
            <button onClick={handleSupport} disabled={supporting}
              className="rounded-full border border-green-200 bg-white px-3 py-1 text-xs
                         text-green-600 transition-colors hover:bg-green-50
                         disabled:opacity-50">
              👍 {q.support_count}
            </button>
          )}
          {!isLoggedIn && (
            <span className="rounded-full bg-green-50 px-3 py-1 text-xs text-green-600">
              👍 {q.support_count}
            </span>
          )}
        </div>
      </div>

      {/* 内容区 */}
      <div className="px-5 py-4">
        {isEditing ? (
          <EditForm question={q} onCancel={onCancelEdit} onSaved={onEdited} />
        ) : (
          <>
            {/* 题干 */}
            <div className="mb-3 whitespace-pre-wrap text-sm leading-relaxed text-gray-800">
              {q.content}
            </div>

            {/* 选项 */}
            {q.options && q.options.length > 0 && (
              <div className="mb-3 space-y-1.5">
                {q.options.map((o) => (
                  <div key={o.option}
                    className={`rounded-md border px-3 py-2 text-sm ${
                      q.answer === o.option
                        ? 'border-green-200 bg-green-50 text-green-800'
                        : 'border-gray-100 text-gray-600'
                    }`}>
                    <span className="mr-2 font-medium">{o.option}.</span>
                    {o.text}
                    {q.answer === o.option && <span className="ml-2 text-xs">✓ 答案</span>}
                  </div>
                ))}
              </div>
            )}

            {/* 答案（非选择题） */}
            {q.answer && (!q.options || q.options.length === 0) && (
              <div className="mb-3 rounded-md border border-green-200 bg-green-50 p-3">
                <div className="mb-1 text-xs font-medium text-green-700">参考答案</div>
                <div className="whitespace-pre-wrap text-sm text-green-800">{q.answer}</div>
              </div>
            )}

            {/* 操作栏 */}
            <div className="flex items-center gap-3 pt-2">
              {isLoggedIn && (
                <button onClick={onStartEdit}
                  className="rounded-md border border-gray-200 px-3 py-1.5 text-xs
                             text-gray-600 hover:bg-gray-50">
                  ✏️ 编辑
                </button>
              )}
              <button onClick={onToggleComments}
                className="rounded-md border border-gray-200 px-3 py-1.5 text-xs
                           text-gray-600 hover:bg-gray-50">
                💬 {showComments ? '收起评论' : '评论'}
              </button>
            </div>
          </>
        )}
      </div>

      {/* 评论区 */}
      {showComments && !isEditing && (
        <CommentSection questionId={q.id} />
      )}
    </div>
  );
}

/* ══════════ 编辑表单 ══════════ */

function EditForm({
  question: q, onCancel, onSaved,
}: {
  question: RecallQuestion;
  onCancel: () => void;
  onSaved: () => void;
}) {
  const [content, setContent] = useState(q.content);
  const [answer, setAnswer] = useState(q.answer);
  const [saving, setSaving] = useState(false);

  /** PATCH /api/recall/questions/:id */
  const handleSave = async () => {
    setSaving(true);
    try {
      const body: PatchRecallQuestionDto = { content, answer };
      await api.patch(`/recall/questions/${q.id}`, body);
      onSaved();
    } catch (err) {
      alert(err instanceof Error ? err.message : '保存失败');
    } finally {
      setSaving(false);
    }
  };

  return (
    <div className="space-y-3">
      <div>
        <label className="mb-1 block text-xs font-medium text-gray-500">题干 (Markdown)</label>
        <textarea value={content} onChange={(e) => setContent(e.target.value)}
          rows={6}
          className="w-full rounded-md border border-gray-200 px-3 py-2 font-mono text-sm
                     outline-none focus:border-brand-500 focus:ring-1 focus:ring-brand-500" />
      </div>
      <div>
        <label className="mb-1 block text-xs font-medium text-gray-500">答案 (Markdown，可为空)</label>
        <textarea value={answer} onChange={(e) => setAnswer(e.target.value)}
          rows={3}
          className="w-full rounded-md border border-gray-200 px-3 py-2 font-mono text-sm
                     outline-none focus:border-brand-500 focus:ring-1 focus:ring-brand-500" />
      </div>
      <div className="flex gap-2">
        <button onClick={handleSave} disabled={saving || !content.trim()}
          className="rounded-md bg-brand-600 px-4 py-1.5 text-sm font-medium text-white
                     hover:bg-brand-700 disabled:opacity-50">
          {saving ? '保存中...' : '保存'}
        </button>
        <button onClick={onCancel}
          className="rounded-md border border-gray-200 px-4 py-1.5 text-sm text-gray-500
                     hover:bg-gray-50">
          取消
        </button>
      </div>
    </div>
  );
}

/* ══════════ 评论区 ══════════ */

function CommentSection({ questionId }: { questionId: number }) {
  const { isLoggedIn } = useAuth();

  const { data: commentsData, loading, refetch } = useFetch(
    () => api.get<RecallCommentListData>(
      `/recall/questions/${questionId}/comments?page=1&size=20`
    ),
    [questionId],
  );

  const [newComment, setNewComment] = useState('');
  const [posting, setPosting] = useState(false);

  const handlePost = async () => {
    if (!newComment.trim()) return;
    setPosting(true);
    try {
      await api.post(`/recall/questions/${questionId}/comments`, {
        content: newComment.trim(),
      });
      setNewComment('');
      refetch();
    } catch (err) {
      alert(err instanceof Error ? err.message : '发表失败');
    } finally {
      setPosting(false);
    }
  };

  const comments = commentsData?.items ?? [];

  return (
    <div className="border-t border-gray-100 bg-gray-50 px-5 py-4">
      <h4 className="mb-3 text-xs font-medium text-gray-500">
        评论 ({commentsData?.total ?? 0})
      </h4>

      {/* 评论列表 */}
      {loading ? (
        <div className="py-4 text-center text-xs text-gray-400">加载中...</div>
      ) : comments.length > 0 ? (
        <div className="mb-4 space-y-3">
          {comments.map((c) => (
            <div key={c.id} className="rounded-md bg-white p-3 text-sm">
              <div className="mb-1 flex items-center gap-2 text-xs text-gray-400">
                <span className="font-medium text-gray-600">
                  {c.display_name ?? c.user_id}
                </span>
                <span>{new Date(c.created_at).toLocaleString('zh-CN')}</span>
              </div>
              <p className="text-gray-700 leading-relaxed">{c.content}</p>
            </div>
          ))}
        </div>
      ) : (
        <div className="mb-4 py-3 text-center text-xs text-gray-400">暂无评论</div>
      )}

      {/* 发表评论 */}
      {isLoggedIn && (
        <div className="flex gap-2">
          <input type="text" value={newComment} onChange={(e) => setNewComment(e.target.value)}
            placeholder="发表你的看法..."
            onKeyDown={(e) => { if (e.key === 'Enter' && !posting) handlePost(); }}
            className="flex-1 rounded-md border border-gray-200 bg-white px-3 py-2 text-sm
                       outline-none focus:border-brand-500 focus:ring-1 focus:ring-brand-500" />
          <button onClick={handlePost} disabled={posting || !newComment.trim()}
            className="rounded-md bg-brand-600 px-4 py-2 text-sm font-medium text-white
                       hover:bg-brand-700 disabled:opacity-50">
            {posting ? '...' : '发表'}
          </button>
        </div>
      )}
    </div>
  );
}
