/**
 * app/login/page.tsx
 * 登录页 — 对接 POST /api/auth/login
 */

'use client';

import { useState } from 'react';
import Link from 'next/link';
import { useRouter } from 'next/navigation';
import { useAuth } from '@/lib/auth';

export default function LoginPage() {
  const router = useRouter();
  const { login } = useAuth();
  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);

  const handleSubmit = async () => {
    console.log('[LoginPage] handleSubmit called', { username });

    if (!username.trim() || !password.trim()) {
      setError('请填写用户名和密码');
      return;
    }

    setError('');
    setLoading(true);
    try {
      await login(username, password);
      console.log('[LoginPage] login success, redirecting...');
      router.push('/courses');
    } catch (err) {
      console.error('[LoginPage] login failed:', err);
      setError(err instanceof Error ? err.message : '登录失败');
    } finally {
      setLoading(false);
    }
  };

  const canSubmit = !loading && !!username.trim() && !!password.trim();

  return (
    <div className="mx-auto mt-16 max-w-sm">
      <div className="mb-8 text-center">
        <div className="mx-auto mb-3 flex h-12 w-12 items-center justify-center rounded-xl bg-brand-700 text-lg font-bold text-white">
          A
        </div>
        <h1 className="text-xl font-semibold text-gray-900">登录 AIRAWeb</h1>
        <p className="mt-1 text-sm text-gray-500">课程试卷在线刷题平台</p>
      </div>

      <div className="rounded-xl border border-gray-200 bg-white p-6 space-y-4">
        {error && (
          <div className="rounded-md bg-red-50 px-3 py-2 text-sm text-red-600">
            {error}
          </div>
        )}

        <div>
          <label className="mb-1 block text-sm font-medium text-gray-700">用户名</label>
          <input type="text" value={username} onChange={(e) => setUsername(e.target.value)}
            placeholder="请输入用户名"
            className="w-full rounded-md border border-gray-200 px-3 py-2 text-sm outline-none
                       focus:border-brand-500 focus:ring-1 focus:ring-brand-500" />
        </div>

        <div>
          <label className="mb-1 block text-sm font-medium text-gray-700">密码</label>
          <input type="password" value={password} onChange={(e) => setPassword(e.target.value)}
            placeholder="请输入密码"
            onKeyDown={(e) => { if (e.key === 'Enter' && canSubmit) handleSubmit(); }}
            className="w-full rounded-md border border-gray-200 px-3 py-2 text-sm outline-none
                       focus:border-brand-500 focus:ring-1 focus:ring-brand-500" />
        </div>

        <button type="button" onClick={handleSubmit} disabled={!canSubmit}
          className="w-full rounded-md bg-brand-600 py-2 text-sm font-medium text-white
                     transition-colors hover:bg-brand-700 disabled:opacity-50 disabled:cursor-not-allowed">
          {loading ? '登录中...' : '登录'}
        </button>

        <p className="text-center text-sm text-gray-500">
          没有账号？
          <Link href="/register" className="ml-1 font-medium text-brand-600 hover:underline">
            注册
          </Link>
        </p>
      </div>
    </div>
  );
}
