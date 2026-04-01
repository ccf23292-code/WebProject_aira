/**
 * app/profile/page.tsx
 * 用户资料
 */

'use client';

import { useState } from 'react';
import type { UserProfile } from '@aira/shared';
import { TableSkeleton } from '@/components/layout/Skeleton';
import { ErrorState } from '@/components/layout/StateDisplay';
import { useFetch } from '@/hooks/useFetch';
import { api } from '@/lib/api';

export default function ProfilePage() {
  const { data, loading, error, refetch } = useFetch(
    () => api.get<UserProfile>('/profile'),
    [],
  );

  const [nickname, setNickname] = useState('');
  const [saving, setSaving] = useState(false);
  const [uploading, setUploading] = useState(false);

  const profile = data;

  const handleSave = async () => {
    setSaving(true);
    try {
      await api.put('/profile', { nickname: nickname || profile?.nickname || '' });
      refetch();
    } finally {
      setSaving(false);
    }
  };

  const handleUpload = async (file: File) => {
    const form = new FormData();
    form.append('avatar', file);
    setUploading(true);
    try {
      const res = await fetch(`${process.env.NEXT_PUBLIC_API_URL ?? 'http://localhost:3001/api'}/profile/avatar`, {
        method: 'POST',
        body: form,
        headers: {
          Authorization: `Bearer ${localStorage.getItem('accessToken') ?? ''}`,
        },
      });
      const body = await res.json();
      if (!res.ok || body.code >= 400) throw new Error(body.message || '上传失败');
      refetch();
    } finally {
      setUploading(false);
    }
  };

  return (
    <div>
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
          <div className="rounded-xl border border-gray-200 bg-white p-5">
            <div className="grid gap-4 md:grid-cols-3">
              <InfoItem label="用户名" value={profile.username || '-'} />
              <InfoItem label="邮箱" value={profile.email || '-'} />
              <InfoItem label="用户等级" value={`Lv.${profile.level ?? 1}`} />
            </div>
          </div>

          <div className="rounded-xl border border-gray-200 bg-white p-5">
            <div className="flex items-center gap-4">
              <div className="h-16 w-16 overflow-hidden rounded-full border border-gray-200 bg-gray-50">
                {profile.avatar_url ? (
                  // eslint-disable-next-line @next/next/no-img-element
                  <img src={profile.avatar_url} alt="avatar" className="h-full w-full object-cover" />
                ) : (
                  <div className="flex h-full w-full items-center justify-center text-sm text-gray-400">头像</div>
                )}
              </div>
              <div>
                <label className="block text-sm text-gray-600">上传新头像（不超过 5MB）</label>
                <input
                  type="file"
                  accept="image/*"
                  disabled={uploading}
                  onChange={(e) => {
                    const file = e.target.files?.[0];
                    if (file) handleUpload(file);
                  }}
                  className="mt-2 text-sm"
                />
              </div>
            </div>
          </div>

          <div className="rounded-xl border border-gray-200 bg-white p-5">
            <label className="mb-2 block text-sm text-gray-600">昵称</label>
            <input
              type="text"
              value={nickname}
              onChange={(e) => setNickname(e.target.value)}
              placeholder={profile.nickname || '填写你的昵称'}
              className="w-full rounded-md border border-gray-200 px-3 py-2 text-sm"
            />
            <button
              onClick={handleSave}
              disabled={saving}
              className="mt-3 rounded-md bg-brand-600 px-4 py-2 text-sm text-white disabled:opacity-50"
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
