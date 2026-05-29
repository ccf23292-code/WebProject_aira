/**
 * app/messages/page.tsx
 * 私信中心 —— 左侧会话列表 + 右侧聊天窗
 *
 * - 轮询刷新（"延迟更新"）：聊天记录每 5s、会话列表每 8s
 * - 已读回执：打开会话即把对方发来的未读标记已读；自己发出的气泡显示"已读/未读"
 * - 深链：/messages?to=<userId>&name=<昵称>&avatar=<头像> 可直接开聊（来自"私信 TA"）
 */

'use client';

import { Suspense, useCallback, useEffect, useRef, useState } from 'react';
import Link from 'next/link';
import { useSearchParams } from 'next/navigation';
import { useAuth } from '@/lib/auth';
import {
  type ConversationItem,
  type MessageItem,
  type MessagePeer,
  listConversations,
  getThread,
  sendMessage,
} from '@/lib/messages';

const THREAD_POLL_MS = 5000;
const LIST_POLL_MS = 8000;

function formatTime(value?: string): string {
  if (!value) return '';
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return '';
  return new Intl.DateTimeFormat('zh-CN', {
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  }).format(date);
}

function Avatar({ url, name, size = 'h-10 w-10' }: { url?: string; name: string; size?: string }) {
  if (url) {
    // eslint-disable-next-line @next/next/no-img-element
    return <img src={url} alt={name} className={`${size} shrink-0 rounded-full border border-brand-100 object-cover`} />;
  }
  return (
    <div className={`${size} flex shrink-0 items-center justify-center rounded-full bg-brand-600 text-sm font-semibold text-white`}>
      {name?.charAt(0)?.toUpperCase() ?? 'U'}
    </div>
  );
}

function MessagesInner() {
  const { isLoggedIn, loading } = useAuth();
  const params = useSearchParams();
  const initialTo = params.get('to');
  const initialName = params.get('name') ?? '';
  const initialAvatar = params.get('avatar') ?? '';

  const [conversations, setConversations] = useState<ConversationItem[]>([]);
  const [selectedPeer, setSelectedPeer] = useState<MessagePeer | null>(null);
  const [messages, setMessages] = useState<MessageItem[]>([]);
  const [draft, setDraft] = useState('');
  const [sending, setSending] = useState(false);
  const [error, setError] = useState('');
  const bottomRef = useRef<HTMLDivElement | null>(null);

  const selectedId = selectedPeer?.peer_id;

  // 深链：从 ?to= 初始化选中对象（哪怕还没有历史消息也能开聊）
  useEffect(() => {
    const id = Number(initialTo);
    if (initialTo && Number.isFinite(id) && id > 0) {
      setSelectedPeer({
        peer_id: id,
        peer_name: initialName || '同学',
        peer_avatar: initialAvatar || undefined,
      });
    }
  }, [initialTo, initialName, initialAvatar]);

  const loadConversations = useCallback(async () => {
    try {
      setConversations(await listConversations());
    } catch {
      /* 静默：轮询失败不打扰 */
    }
  }, []);

  const loadThread = useCallback(async (peerId: number) => {
    try {
      const data = await getThread(peerId);
      setMessages(data.items);
      // 用后端返回的对方资料补全昵称/头像
      setSelectedPeer((prev) =>
        prev && prev.peer_id === peerId
          ? {
              peer_id: peerId,
              peer_name: data.peer.peer_name || prev.peer_name,
              peer_avatar: data.peer.peer_avatar || prev.peer_avatar,
            }
          : prev,
      );
    } catch {
      /* 静默 */
    }
  }, []);

  // 会话列表：登录后加载并轮询
  useEffect(() => {
    if (!isLoggedIn) return;
    void loadConversations();
    const timer = setInterval(loadConversations, LIST_POLL_MS);
    return () => clearInterval(timer);
  }, [isLoggedIn, loadConversations]);

  // 聊天记录：选中对象后加载并轮询
  useEffect(() => {
    if (!isLoggedIn || !selectedId) return;
    void loadThread(selectedId);
    const timer = setInterval(() => loadThread(selectedId), THREAD_POLL_MS);
    return () => clearInterval(timer);
  }, [isLoggedIn, selectedId, loadThread]);

  // 新消息时滚动到底
  useEffect(() => {
    bottomRef.current?.scrollIntoView({ behavior: 'smooth' });
  }, [messages.length, selectedId]);

  const handleSelect = (item: ConversationItem) => {
    setMessages([]);
    setError('');
    setSelectedPeer({ peer_id: item.peer_id, peer_name: item.peer_name, peer_avatar: item.peer_avatar });
    // 乐观清掉未读小红点（后端打开会话时也会标记已读）
    setConversations((prev) =>
      prev.map((c) => (c.peer_id === item.peer_id ? { ...c, unread_count: 0 } : c)),
    );
  };

  const handleSend = async (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    const content = draft.trim();
    if (!content || !selectedId) return;
    setSending(true);
    setError('');
    try {
      const sent = await sendMessage(selectedId, content);
      setMessages((prev) => [...prev, sent]);
      setDraft('');
      void loadConversations();
    } catch (err) {
      setError(err instanceof Error ? err.message : '发送失败，请稍后重试。');
    } finally {
      setSending(false);
    }
  };

  if (loading) {
    return <p className="py-10 text-center text-sm text-gray-500">加载中…</p>;
  }

  if (!isLoggedIn) {
    return (
      <div className="rounded-2xl border border-gray-200 bg-white p-8 text-center">
        <p className="text-sm text-gray-600">登录后才能使用私信功能。</p>
        <Link
          href="/login"
          className="mt-4 inline-block rounded-lg bg-brand-700 px-4 py-2 text-sm font-medium text-white hover:bg-brand-800"
        >
          去登录
        </Link>
      </div>
    );
  }

  return (
    <div>
      <h1 className="mb-4 text-lg font-semibold text-gray-900">私信</h1>
      <div className="grid h-[70vh] grid-cols-1 overflow-hidden rounded-2xl border border-gray-200 bg-white md:grid-cols-[280px,1fr]">
        {/* 左：会话列表 */}
        <aside className={`flex flex-col border-r border-gray-200 ${selectedId ? 'hidden md:flex' : 'flex'}`}>
          <div className="border-b border-gray-100 px-4 py-3 text-sm font-medium text-gray-700">
            会话
          </div>
          <div className="flex-1 overflow-y-auto">
            {conversations.length === 0 ? (
              <p className="px-4 py-6 text-xs leading-6 text-gray-400">
                还没有会话。去课程讨论区，点开同学头像旁的「私信 TA」就能开始聊天。
              </p>
            ) : (
              conversations.map((item) => {
                const active = item.peer_id === selectedId;
                return (
                  <button
                    key={item.peer_id}
                    type="button"
                    onClick={() => handleSelect(item)}
                    className={`flex w-full items-center gap-3 px-4 py-3 text-left transition-colors ${
                      active ? 'bg-brand-50' : 'hover:bg-gray-50'
                    }`}
                  >
                    <Avatar url={item.peer_avatar} name={item.peer_name} />
                    <div className="min-w-0 flex-1">
                      <div className="flex items-center justify-between gap-2">
                        <span className="truncate text-sm font-medium text-gray-900">{item.peer_name}</span>
                        <span className="shrink-0 text-[11px] text-gray-400">{formatTime(item.last_at)}</span>
                      </div>
                      <div className="flex items-center justify-between gap-2">
                        <span className="truncate text-xs text-gray-500">
                          {item.last_mine ? '我: ' : ''}
                          {item.last_message}
                        </span>
                        {item.unread_count > 0 ? (
                          <span className="ml-1 inline-flex h-4 min-w-4 items-center justify-center rounded-full bg-rose-500 px-1 text-[10px] font-semibold text-white">
                            {item.unread_count}
                          </span>
                        ) : null}
                      </div>
                    </div>
                  </button>
                );
              })
            )}
          </div>
        </aside>

        {/* 右：聊天窗 */}
        <section className={`flex flex-col ${selectedId ? 'flex' : 'hidden md:flex'}`}>
          {selectedPeer ? (
            <>
              <header className="flex items-center gap-3 border-b border-gray-100 px-4 py-3">
                <button
                  type="button"
                  onClick={() => setSelectedPeer(null)}
                  className="rounded-md p-1 text-gray-400 hover:bg-gray-100 md:hidden"
                  aria-label="返回会话列表"
                >
                  ←
                </button>
                <Avatar url={selectedPeer.peer_avatar} name={selectedPeer.peer_name} size="h-8 w-8" />
                <span className="text-sm font-medium text-gray-900">{selectedPeer.peer_name}</span>
              </header>

              <div className="flex-1 space-y-3 overflow-y-auto bg-gray-50 px-4 py-4">
                {messages.length === 0 ? (
                  <p className="py-10 text-center text-xs text-gray-400">还没有消息，发一条打个招呼吧～</p>
                ) : (
                  messages.map((m) => (
                    <div key={m.id} className={`flex ${m.mine ? 'justify-end' : 'justify-start'}`}>
                      <div className={`max-w-[75%] ${m.mine ? 'items-end' : 'items-start'} flex flex-col`}>
                        <div
                          className={`whitespace-pre-wrap break-words rounded-2xl px-3 py-2 text-sm leading-6 ${
                            m.mine
                              ? 'rounded-br-sm bg-brand-600 text-white'
                              : 'rounded-bl-sm border border-gray-200 bg-white text-gray-800'
                          }`}
                        >
                          {m.content}
                        </div>
                        <div className="mt-1 flex items-center gap-2 px-1 text-[10px] text-gray-400">
                          <span>{formatTime(m.created_at)}</span>
                          {m.mine ? (
                            <span className={m.read ? 'text-brand-500' : 'text-gray-400'}>
                              {m.read ? '已读' : '未读'}
                            </span>
                          ) : null}
                        </div>
                      </div>
                    </div>
                  ))
                )}
                <div ref={bottomRef} />
              </div>

              <form onSubmit={handleSend} className="flex items-end gap-2 border-t border-gray-100 px-4 py-3">
                <textarea
                  value={draft}
                  onChange={(e) => setDraft(e.target.value)}
                  onKeyDown={(e) => {
                    if (e.key === 'Enter' && !e.shiftKey) {
                      e.preventDefault();
                      void handleSend(e as unknown as React.FormEvent<HTMLFormElement>);
                    }
                  }}
                  rows={1}
                  placeholder="输入消息，Enter 发送，Shift+Enter 换行"
                  className="max-h-32 min-h-[40px] flex-1 resize-none rounded-xl border border-gray-200 bg-white px-3 py-2 text-sm outline-none focus:border-brand-500 focus:ring-1 focus:ring-brand-500"
                />
                <button
                  type="submit"
                  disabled={sending || !draft.trim()}
                  className="shrink-0 rounded-xl bg-brand-600 px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-brand-700 disabled:cursor-not-allowed disabled:bg-brand-300"
                >
                  {sending ? '发送中…' : '发送'}
                </button>
              </form>
              {error ? <p className="px-4 pb-3 text-xs text-rose-600">{error}</p> : null}
            </>
          ) : (
            <div className="flex flex-1 items-center justify-center text-sm text-gray-400">
              选择一个会话开始聊天
            </div>
          )}
        </section>
      </div>
    </div>
  );
}

export default function MessagesPage() {
  return (
    <Suspense fallback={<p className="py-10 text-center text-sm text-gray-500">加载中…</p>}>
      <MessagesInner />
    </Suspense>
  );
}
