'use client';

import { useCallback, useEffect, useMemo, useState } from 'react';
import type {
  ProblemExplanationItem,
  ProblemExplanationListData,
  UpsertProblemExplanationDto,
  VoteProblemExplanationDto,
} from '@aira/shared';
import { api } from '@/lib/api';
import { useAuth } from '@/lib/auth';
import { MarkdownBlock } from '@/components/Markdown';
import { AIExplanationPanel } from '@/components/problem/AIExplanationPanel';

interface ExplanationSectionProps {
  problemId: number;
  enabled: boolean;
  officialExplanation?: string;
}

export function ExplanationSection({
  problemId,
  enabled,
  officialExplanation,
}: ExplanationSectionProps) {
  const { isLoggedIn } = useAuth();
  const [data, setData] = useState<ProblemExplanationListData | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [draft, setDraft] = useState('');
  const [saving, setSaving] = useState(false);
  const [votingId, setVotingId] = useState<number | null>(null);

  const load = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const result = await api.get<ProblemExplanationListData>(`/problems/${problemId}/explanations`);
      setData(result);
      setDraft(result.my_item?.content_md ?? result.items.find((item) => item.can_edit)?.content_md ?? '');
    } catch (err) {
      setError(err instanceof Error ? err.message : '题解加载失败');
    } finally {
      setLoading(false);
    }
  }, [problemId]);

  useEffect(() => {
    if (!enabled || data || loading) return;
    void load();
  }, [data, enabled, load, loading]);

  const displayItems = useMemo(() => data?.items ?? [], [data?.items]);
  const ownItem = useMemo(
    () => data?.my_item ?? data?.items.find((item) => item.can_edit) ?? null,
    [data?.items, data?.my_item],
  );

  const handleSave = useCallback(async () => {
    const payload: UpsertProblemExplanationDto = { content_md: draft };
    setSaving(true);
    try {
      await api.post(`/problems/${problemId}/explanations`, payload);
      await load();
    } catch (err) {
      setError(err instanceof Error ? err.message : '保存题解失败');
    } finally {
      setSaving(false);
    }
  }, [draft, load, problemId]);

  const handleVote = useCallback(async (item: ProblemExplanationItem, nextValue: -1 | 1) => {
    const payload: VoteProblemExplanationDto = {
      value: item.my_vote === nextValue ? 0 : nextValue,
    };
    setVotingId(item.id);
    try {
      await api.post(`/problems/${problemId}/explanations/${item.id}/vote`, payload);
      await load();
    } catch (err) {
      setError(err instanceof Error ? err.message : '投票失败');
    } finally {
      setVotingId(null);
    }
  }, [load, problemId]);

  const myItemInTopList = ownItem ? displayItems.some((item) => item.id === ownItem.id) : false;

  if (!enabled) return null;

  return (
    <div className="mt-4 space-y-4 rounded-lg border border-gray-100 bg-gray-50 px-4 py-4">
      <div>
        <h3 className="text-sm font-semibold text-gray-900">题目解析</h3>
        <p className="mt-1 text-xs text-gray-500">先显示参考解析，再显示评分最高的 3 条同学解析。</p>
      </div>

      {officialExplanation?.trim() && (
        <section className="rounded-lg border border-blue-100 bg-blue-50 px-4 py-3">
          <div className="text-xs font-medium text-blue-700">参考解析</div>
          <MarkdownBlock content={officialExplanation} className="prose prose-sm mt-2 max-w-none text-gray-800" />
        </section>
      )}

      {isLoggedIn ? <AIExplanationPanel problemId={problemId} /> : null}

      {loading ? (
        <div className="text-sm text-gray-500">题解加载中...</div>
      ) : error ? (
        <div className="rounded-md border border-red-100 bg-red-50 px-3 py-2 text-sm text-red-600">{error}</div>
      ) : (
        <section className="space-y-3">
          <div className="text-xs font-medium text-gray-600">同学解析</div>
          {displayItems.length === 0 ? (
            <div className="rounded-md border border-dashed border-gray-200 bg-white px-3 py-3 text-sm text-gray-500">
              暂无同学解析，欢迎补充第一份。
            </div>
          ) : (
            displayItems.map((item) => (
              <ExplanationCard
                key={item.id}
                item={item}
                canVote={isLoggedIn}
                voting={votingId === item.id}
                onVote={handleVote}
              />
            ))
          )}
        </section>
      )}

      {isLoggedIn ? (
        <section className="rounded-lg border border-gray-200 bg-white px-4 py-3">
          <div className="mb-2 flex items-center justify-between">
            <div className="text-xs font-medium text-gray-700">
              {ownItem ? '编辑我的解析' : '提交我的解析'}
            </div>
            {ownItem && !myItemInTopList && (
              <span className="text-xs text-gray-400">你的解析当前未进入 Top 3，但仍可编辑</span>
            )}
          </div>
          <textarea
            value={draft}
            onChange={(e) => setDraft(e.target.value)}
            rows={6}
            placeholder="支持 Markdown 和 LaTeX 公式，例如：$O(n \\log n)$"
            className="w-full rounded-md border border-gray-200 px-3 py-2 text-sm text-gray-700"
          />
          <div className="mt-3 flex items-center justify-between gap-3">
            <div className="text-xs text-gray-400">只会保留你在这道题上的一份解析，重复提交将更新原内容。</div>
            <button
              onClick={() => void handleSave()}
              disabled={saving || !draft.trim()}
              className="rounded-md bg-brand-600 px-3 py-1.5 text-xs font-medium text-white transition-colors hover:bg-brand-700 disabled:cursor-not-allowed disabled:opacity-50"
            >
              {saving ? '保存中...' : '保存解析'}
            </button>
          </div>
          {ownItem && !myItemInTopList && (
            <div className="mt-4 rounded-md border border-amber-100 bg-amber-50 px-3 py-3">
              <div className="text-xs font-medium text-amber-700">我的解析</div>
              <MarkdownBlock content={ownItem.content_md} className="prose prose-sm mt-2 max-w-none text-gray-800" />
            </div>
          )}
        </section>
      ) : (
        <div className="text-xs text-gray-500">登录后可提交题解、点赞或点踩。</div>
      )}
    </div>
  );
}

function ExplanationCard({
  item,
  canVote,
  voting,
  onVote,
}: {
  item: ProblemExplanationItem;
  canVote: boolean;
  voting: boolean;
  onVote: (item: ProblemExplanationItem, value: -1 | 1) => void;
}) {
  return (
    <article className="rounded-lg border border-gray-200 bg-white px-4 py-3">
      <div className="flex items-center justify-between gap-3">
        <div>
          <div className="text-sm font-medium text-gray-900">{item.author_name}</div>
          <div className="mt-1 text-xs text-gray-400">
            {new Date(item.updated_at).toLocaleString('zh-CN')}
            {item.can_edit ? ' · 你提交的解析' : ''}
          </div>
        </div>
        {!item.can_edit && canVote && (
          <div className="flex items-center gap-2">
            <button
              onClick={() => onVote(item, 1)}
              disabled={voting}
              className={`rounded-md px-2 py-1 text-xs transition-colors ${
                item.my_vote === 1 ? 'bg-green-50 text-green-700' : 'bg-gray-50 text-gray-500 hover:bg-gray-100'
              }`}
            >
              赞 {item.up_votes}
            </button>
            <button
              onClick={() => onVote(item, -1)}
              disabled={voting}
              className={`rounded-md px-2 py-1 text-xs transition-colors ${
                item.my_vote === -1 ? 'bg-red-50 text-red-700' : 'bg-gray-50 text-gray-500 hover:bg-gray-100'
              }`}
            >
              踩 {item.down_votes}
            </button>
          </div>
        )}
      </div>
      <MarkdownBlock content={item.content_md} className="prose prose-sm mt-3 max-w-none text-gray-800" />
    </article>
  );
}
