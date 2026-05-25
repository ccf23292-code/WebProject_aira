/**
 * app/profile/page.tsx
 * 用户资料 - 头像上传 / 昵称修改
 */

'use client';

import { useRef, useState } from 'react';
import type { UserProfile } from '@aira/shared';
import { TableSkeleton } from '@/components/layout/Skeleton';
import { ErrorState } from '@/components/layout/StateDisplay';
import { useFetch } from '@/hooks/useFetch';
import { api } from '@/lib/api';
import { useAuth } from '@/lib/auth';

export default function ProfilePage() {
  const { data, loading, error, refetch } = useFetch(
    () => api.get<UserProfile>('/profile'),
    [],
  );
  const { updateAvatar } = useAuth();

  const [nickname, setNickname] = useState('');
  const [saving, setSaving] = useState(false);
  const [uploading, setUploading] = useState(false);
  const [toast, setToast] = useState<{ type: 'success' | 'error'; text: string } | null>(null);
  const fileInputRef = useRef<HTMLInputElement>(null);

  const profile = data;

  const showToast = (type: 'success' | 'error', text: string) => {
    setToast({ type, text });
    setTimeout(() => setToast(null), 3000);
  };

  const handleSave = async () => {
    setSaving(true);
    try {
      await api.put('/profile', { nickname: nickname || profile?.nickname || '' });
      refetch();
      showToast('success', '昵称已保存');
    } catch (e) {
      showToast('error', e instanceof Error ? e.message : '保存失败');
    } finally {
      setSaving(false);
    }
  };

  const handleUpload = async (file: File) => {
    if (file.size > 5 * 1024 * 1024) {
      showToast('error', '图片不能超过 5MB');
      return;
    }
    const form = new FormData();
    form.append('avatar', file);
    setUploading(true);
    try {
      const result = await api.upload<UserProfile>('/profile/avatar', form);
      if (result.avatar_url) updateAvatar(result.avatar_url);
      refetch();
      showToast('success', '头像已更新');
    } catch (e) {
      showToast('error', e instanceof Error ? e.message : '上传失败，请重试');
    } finally {
      setUploading(false);
      if (fileInputRef.current) fileInputRef.current.value = '';
    }
  };

  return (
    <div>
      {/* Toast 通知 */}
      {toast && (
        <div className={`fixed right-6 top-20 z-50 rounded-xl border px-4 py-3 text-sm font-medium shadow-lg ${toast.type === 'success' ? 'border-green-200 bg-green-50 text-green-700' : 'border-red-200 bg-red-50 text-red-700'}`}>
          {toast.type === 'success' ? '✓ ' : '✕ '}{toast.text}
        </div>
      )}

      <div className="mb-6">
        <h1 className="text-xl font-semibold text-gray-900">个人资料</h1>
        <p className="mt-1 text-sm text-gray-500">管理昵称与头像</p>
      </div>

      {loading ? (
        <TableSkeleton rows={2} />
      ) : error ? (
        <ErrorState message={error} onRetry={refetch} />
      ) : profile ? (
        <div className="space-y-6">
          {/* 基本信息 */}
          <div className="rounded-xl border border-gray-200 bg-white p-5">
            <div className="grid gap-4 md:grid-cols-3">
              <InfoItem label="用户名" value={profile.username || '-'} />
              <InfoItem label="邮箱" value={profile.email || '-'} />
              <InfoItem label="用户等级" value={`Lv.${profile.level ?? 1}`} />
            </div>
          </div>

          {/* 头像上传 */}
          <div className="rounded-xl border border-gray-200 bg-white p-5">
            <div className="mb-3 text-sm font-medium text-gray-700">头像</div>
            <div className="flex items-center gap-5">
              {/* 可点击的头像圆圈 */}
              <button
                type="button"
                onClick={() => !uploading && fileInputRef.current?.click()}
                disabled={uploading}
                className="group relative h-20 w-20 flex-shrink-0 overflow-hidden rounded-full border-2 border-gray-200 bg-gray-50 transition-colors hover:border-brand-400 disabled:cursor-not-allowed"
                title="点击更换头像"
              >
                {/* 当前头像或首字母占位 */}
                {profile.avatar_url ? (
                  // eslint-disable-next-line @next/next/no-img-element
                  <img src={profile.avatar_url} alt="头像" className="h-full w-full object-cover" />
                ) : (
                  <span className="flex h-full w-full items-center justify-center text-2xl font-bold text-gray-300">
                    {(profile.nickname || profile.username || 'U').charAt(0).toUpperCase()}
                  </span>
                )}

                {/* 上传中：旋转 loading 遮罩 */}
                {uploading && (
                  <div className="absolute inset-0 flex items-center justify-center bg-black/40">
                    <svg
                      className="h-7 w-7 animate-spin text-white"
                      viewBox="0 0 24 24"
                      fill="none"
                      xmlns="http://www.w3.org/2000/svg"
                    >
                      <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
                      <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8v4a4 4 0 00-4 4H4z" />
                    </svg>
                  </div>
                )}

                {/* hover 提示遮罩 */}
                {!uploading && (
                  <div className="absolute inset-0 flex items-center justify-center bg-black/0 transition-colors group-hover:bg-black/30">
                    <span className="text-xs font-medium text-white opacity-0 transition-opacity group-hover:opacity-100">
                      更换头像
                    </span>
                  </div>
                )}
              </button>

              {/* 隐藏的文件选择器 */}
              <input
                ref={fileInputRef}
                type="file"
                accept="image/*"
                className="hidden"
                onChange={(e) => {
                  const file = e.target.files?.[0];
                  if (file) handleUpload(file);
                }}
              />

              <div className="text-sm leading-6 text-gray-500">
                <p>点击头像选择图片进行上传</p>
                <p className="mt-1 text-xs text-gray-400">支持 JPG、PNG、GIF，不超过 5MB</p>
              </div>
            </div>
          </div>

          {/* 昵称修改 */}
          <div className="rounded-xl border border-gray-200 bg-white p-5">
            <label className="mb-2 block text-sm font-medium text-gray-700">昵称</label>
            <input
              type="text"
              value={nickname}
              onChange={(e) => setNickname(e.target.value)}
              placeholder={profile.nickname || '填写你的昵称'}
              className="w-full rounded-lg border border-gray-200 px-3 py-2 text-sm outline-none focus:border-brand-400 focus:ring-1 focus:ring-brand-400"
            />
            <button
              onClick={handleSave}
              disabled={saving}
              className="mt-3 rounded-lg bg-brand-600 px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-brand-700 disabled:opacity-50"
            >
              {saving ? '保存中...' : '保存资料'}
            </button>
          </div>
        </div>
      ) : null}
    </div>
  );
}

function InfoItem({ label, value }: { label: string; value: string }) {
  return (
    <div>
      <div className="text-xs font-medium uppercase tracking-wide text-gray-400">{label}</div>
      <div className="mt-2 text-sm font-medium text-gray-900">{value}</div>
    </div>
  );
}
