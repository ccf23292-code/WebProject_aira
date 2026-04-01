/**
 * app/register/page.tsx
 * 注册页 — 对接 POST /api/auth/register
 */

'use client';

import { useEffect, useState } from 'react';
import Link from 'next/link';
import { useRouter } from 'next/navigation';
import { useAuth } from '@/lib/auth';
import { api } from '@/lib/api';
import type { VerificationCodeData } from '@aira/shared';

export default function RegisterPage() {
  const router = useRouter();
  const { register } = useAuth();
  const [username, setUsername] = useState('');
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [confirm, setConfirm] = useState('');
  const [verificationCode, setVerificationCode] = useState('');
  const [agreeToPolicy, setAgreeToPolicy] = useState(false);
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);
  const [sendingCode, setSendingCode] = useState(false);
  const [countdown, setCountdown] = useState(0);
  const [devCode, setDevCode] = useState<string | null>(null);

  useEffect(() => {
    if (countdown <= 0) return undefined;
    const timer = setInterval(() => setCountdown((prev) => prev - 1), 1000);
    return () => clearInterval(timer);
  }, [countdown]);

  const handleSendCode = async () => {
    if (!email.trim()) {
      setError('请先填写邮箱');
      return;
    }
    setError('');
    setSendingCode(true);
    try {
      const data = await api.post<VerificationCodeData>(
        '/auth/verification-code',
        { email },
        true,
      );
      if (data?.code) setDevCode(data.code);
      setCountdown(60);
    } catch (err) {
      setError(err instanceof Error ? err.message : '发送验证码失败');
    } finally {
      setSendingCode(false);
    }
  };

  const handleSubmit = async () => {
    console.log('[RegisterPage] handleSubmit called', { username, password: '***' });

    if (!username.trim() || !email.trim() || !password.trim() || !confirm.trim() || !verificationCode.trim()) {
      setError('请填写所有字段');
      return;
    }
    if (password !== confirm) {
      setError('两次密码不一致');
      return;
    }
    if (verificationCode.trim().length !== 6) {
      setError('验证码必须为 6 位');
      return;
    }
    if (!agreeToPolicy) {
      setError('请先同意隐私政策与用户协议');
      return;
    }

    setError('');
    setLoading(true);
    try {
      await register({
        username,
        email,
        password,
        confirmPassword: confirm,
        verificationCode,
        agreeToPolicy,
      });
      console.log('[RegisterPage] register success, redirecting...');
      router.push('/courses');
    } catch (err) {
      console.error('[RegisterPage] register failed:', err);
      setError(err instanceof Error ? err.message : '注册失败');
    } finally {
      setLoading(false);
    }
  };

  const canSubmit = !loading
    && !!username.trim()
    && !!email.trim()
    && !!password.trim()
    && !!confirm.trim()
    && !!verificationCode.trim()
    && agreeToPolicy;

  return (
    <div className="mx-auto mt-16 max-w-sm">
      <div className="mb-8 text-center">
        <div className="mx-auto mb-3 flex h-12 w-12 items-center justify-center rounded-xl bg-brand-700 text-lg font-bold text-white">
          A
        </div>
        <h1 className="text-xl font-semibold text-gray-900">注册 AIRAWeb</h1>
      </div>

      <div className="rounded-xl border border-gray-200 bg-white p-6 space-y-4">
        {error && (
          <div className="rounded-md bg-red-50 px-3 py-2 text-sm text-red-600">{error}</div>
        )}

        <div>
          <label className="mb-1 block text-sm font-medium text-gray-700">用户名</label>
          <input type="text" value={username} onChange={(e) => setUsername(e.target.value)}
            placeholder="请输入用户名"
            className="w-full rounded-md border border-gray-200 px-3 py-2 text-sm outline-none
                       focus:border-brand-500 focus:ring-1 focus:ring-brand-500" />
        </div>

        <div>
          <label className="mb-1 block text-sm font-medium text-gray-700">邮箱</label>
          <div className="flex gap-2">
            <input type="email" value={email} onChange={(e) => setEmail(e.target.value)}
              placeholder="name@example.com"
              className="w-full rounded-md border border-gray-200 px-3 py-2 text-sm outline-none
                         focus:border-brand-500 focus:ring-1 focus:ring-brand-500" />
            <button
              type="button"
              onClick={handleSendCode}
              disabled={sendingCode || countdown > 0}
              className="whitespace-nowrap rounded-md border border-gray-200 px-3 py-2 text-sm
                         text-gray-700 hover:bg-gray-50 disabled:cursor-not-allowed disabled:opacity-50"
            >
              {countdown > 0 ? `${countdown}s` : '获取验证码'}
            </button>
          </div>
          {devCode && (
            <div className="mt-2 text-xs text-amber-600">
              开发模式验证码：{devCode}
            </div>
          )}
        </div>

        <div>
          <label className="mb-1 block text-sm font-medium text-gray-700">密码</label>
          <input type="password" value={password} onChange={(e) => setPassword(e.target.value)}
            placeholder="请输入密码"
            className="w-full rounded-md border border-gray-200 px-3 py-2 text-sm outline-none
                       focus:border-brand-500 focus:ring-1 focus:ring-brand-500" />
        </div>

        <div>
          <label className="mb-1 block text-sm font-medium text-gray-700">确认密码</label>
          <input type="password" value={confirm} onChange={(e) => setConfirm(e.target.value)}
            placeholder="再次输入密码"
            onKeyDown={(e) => { if (e.key === 'Enter' && canSubmit) handleSubmit(); }}
            className="w-full rounded-md border border-gray-200 px-3 py-2 text-sm outline-none
                       focus:border-brand-500 focus:ring-1 focus:ring-brand-500" />
        </div>

        <div>
          <label className="mb-1 block text-sm font-medium text-gray-700">验证码</label>
          <input type="text" value={verificationCode} onChange={(e) => setVerificationCode(e.target.value)}
            placeholder="请输入 6 位验证码"
            className="w-full rounded-md border border-gray-200 px-3 py-2 text-sm outline-none
                       focus:border-brand-500 focus:ring-1 focus:ring-brand-500" />
        </div>

        <label className="flex items-center gap-2 text-sm text-gray-600">
          <input
            type="checkbox"
            checked={agreeToPolicy}
            onChange={(e) => setAgreeToPolicy(e.target.checked)}
            className="h-4 w-4 rounded border-gray-300 text-brand-600 focus:ring-brand-500"
          />
          我已阅读并同意隐私政策与用户协议
        </label>

        <button type="button" onClick={handleSubmit} disabled={!canSubmit}
          className="w-full rounded-md bg-brand-600 py-2 text-sm font-medium text-white
                     transition-colors hover:bg-brand-700 disabled:opacity-50 disabled:cursor-not-allowed">
          {loading ? '注册中...' : '注册'}
        </button>

        <p className="text-center text-sm text-gray-500">
          已有账号？
          <Link href="/login" className="ml-1 font-medium text-brand-600 hover:underline">
            登录
          </Link>
        </p>
      </div>
    </div>
  );
}
