/**
 * lib/messages.ts
 * 私信（DM）API 客户端 —— 与后端 message_controller.go 对齐
 *
 * 后端路由（均需登录）：
 *   POST /api/messages                  发消息 { receiver_id, content }
 *   GET  /api/messages/conversations    会话列表
 *   GET  /api/messages/unread-count     未读总数
 *   GET  /api/messages/with/:userId     聊天记录（打开即标记已读）
 */

import { api } from './api';

/** 会话列表里的单个会话 */
export interface ConversationItem {
  peer_id: number;
  peer_name: string;
  peer_avatar?: string;
  last_message: string;
  last_at: string;
  last_mine: boolean;
  unread_count: number;
}

/** 聊天记录里的单条消息 */
export interface MessageItem {
  id: number;
  sender_id: number;
  receiver_id: number;
  content: string;
  mine: boolean; // 是否当前用户发出
  read: boolean; // 是否已读（对"我发出的"消息才有意义）
  created_at: string;
}

/** 会话对方的精简信息 */
export interface MessagePeer {
  peer_id: number;
  peer_name: string;
  peer_avatar?: string;
}

/** 聊天记录响应 */
export interface ThreadData {
  peer: MessagePeer;
  items: MessageItem[];
}

/** 拉取当前用户的会话列表 */
export async function listConversations(): Promise<ConversationItem[]> {
  const data = await api.get<{ items: ConversationItem[] }>('/messages/conversations');
  return data.items ?? [];
}

/** 拉取与某人的聊天记录（会顺带把对方发来的未读标记为已读） */
export function getThread(peerId: number, limit = 50): Promise<ThreadData> {
  return api.get<ThreadData>(`/messages/with/${peerId}?limit=${limit}`);
}

/** 发送一条私信，返回落库后的消息 */
export function sendMessage(receiverId: number, content: string): Promise<MessageItem> {
  return api.post<MessageItem>('/messages', { receiver_id: receiverId, content });
}

/** 查询未读私信总数（导航栏小红点用） */
export async function getUnreadCount(): Promise<number> {
  const data = await api.get<{ count: number }>('/messages/unread-count');
  return data.count ?? 0;
}
