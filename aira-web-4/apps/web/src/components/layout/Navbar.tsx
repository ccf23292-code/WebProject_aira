/**
 * components/layout/Navbar.tsx
 * 顶部导航 — 首页 / 课程 / 个人中心 / 登录状态
 */

'use client';

import Link from 'next/link';
import { usePathname, useRouter } from 'next/navigation';
import { useAuth } from '@/lib/auth';
import { getUnreadCount } from '@/lib/messages';
import { useEffect, useRef, useState } from 'react';

const UNREAD_POLL_MS = 20000;

export function Navbar() {
  const pathname = usePathname();
  const router = useRouter();
  const { user, isLoggedIn, logout } = useAuth();
  const [menuOpen, setMenuOpen] = useState(false);
  const [unread, setUnread] = useState(0);
  const menuRef = useRef<HTMLDivElement | null>(null);

  const handleLogout = async () => {
    await logout();
    router.push('/login');
  };

  // 私信未读总数：登录后轮询（"延迟更新"），未登录清零
  useEffect(() => {
    if (!isLoggedIn) {
      setUnread(0);
      return;
    }
    let cancelled = false;
    const tick = async () => {
      try {
        const n = await getUnreadCount();
        if (!cancelled) setUnread(n);
      } catch {
        /* 静默：轮询失败不打扰 */
      }
    };
    void tick();
    const timer = setInterval(tick, UNREAD_POLL_MS);
    return () => {
      cancelled = true;
      clearInterval(timer);
    };
  }, [isLoggedIn, pathname]);

  const links = [
    { href: '/', label: '首页' },
    { href: '/courses', label: '课程' },
    ...(isLoggedIn ? [{ href: '/upload', label: '上传题库' }] : []),
    ...(isLoggedIn ? [{ href: '/profile', label: '个人中心' }] : []),
    ...(user?.roles?.includes('admin') ? [{ href: '/admin/reviews', label: '管理审核' }] : []),
    ...(user?.roles?.includes('admin') ? [{ href: '/admin/ingest', label: '上传审核' }] : []),
  ];

  useEffect(() => {
    const handler = (event: MouseEvent) => {
      if (!menuOpen) return;
      const target = event.target as Node;
      if (menuRef.current && !menuRef.current.contains(target)) {
        setMenuOpen(false);
      }
    };
    document.addEventListener('click', handler);
    return () => document.removeEventListener('click', handler);
  }, [menuOpen]);

  return (
    <header className="sticky top-0 z-40 border-b border-brand-100 bg-[#fffdf9]/95 shadow-[0_1px_0_rgba(180,120,72,0.08)] backdrop-blur-sm">
      <nav className="mx-auto flex h-14 max-w-5xl items-center justify-between px-4">
        {/* Logo */}
        <Link href="/" className="flex items-center gap-2 font-semibold text-brand-900">
          <span className="flex h-7 w-7 items-center justify-center rounded bg-brand-700 text-xs font-bold text-white shadow-sm shadow-brand-200">
            A
          </span>
          <span className="hidden sm:inline">AIRAWeb</span>
        </Link>

        {/* 导航链接 */}
        <div className="flex items-center gap-1">
          {links.map((link) => {
            const active = link.href === '/'
              ? pathname === '/'
              : pathname === link.href || pathname.startsWith(`${link.href}/`);
            return (
              <Link key={link.href} href={link.href}
                className={`rounded-md px-3 py-1.5 text-sm transition-colors ${
                  active
                    ? 'bg-brand-50 font-medium text-brand-800 ring-1 ring-brand-100 shadow-[inset_0_-2px_0_rgba(143,78,39,0.32)]'
                    : 'text-stone-600 hover:bg-brand-50 hover:text-brand-800'
                }`}>
                {link.label}
              </Link>
            );
          })}

          {/* 私信入口（带未读小红点） */}
          {isLoggedIn ? (
            <Link
              href="/messages"
              className={`relative rounded-md px-3 py-1.5 text-sm transition-colors ${
                pathname === '/messages' || pathname.startsWith('/messages/')
                  ? 'bg-brand-50 font-medium text-brand-800 ring-1 ring-brand-100 shadow-[inset_0_-2px_0_rgba(143,78,39,0.32)]'
                  : 'text-stone-600 hover:bg-brand-50 hover:text-brand-800'
              }`}
            >
              私信
              {unread > 0 ? (
                <span className="absolute -right-0.5 -top-0.5 inline-flex h-4 min-w-4 items-center justify-center rounded-full bg-rose-500 px-1 text-[10px] font-semibold text-white">
                  {unread > 99 ? '99+' : unread}
                </span>
              ) : null}
            </Link>
          ) : null}

          {/* 登录/用户 */}
          {isLoggedIn ? (
            <div className="relative ml-2" ref={menuRef}>
              <button onClick={() => setMenuOpen((prev) => !prev)}
                className="flex items-center gap-2 rounded-md px-3 py-1.5 text-sm text-stone-600 transition-colors hover:bg-brand-50 hover:text-brand-800">
                {user?.avatarUrl ? (
                  // eslint-disable-next-line @next/next/no-img-element
                  <img
                    src={user.avatarUrl}
                    alt={user.displayName}
                    className="h-6 w-6 rounded-full border border-brand-100 object-cover"
                  />
                ) : (
                  <span className="flex h-6 w-6 items-center justify-center rounded-full bg-brand-600 text-xs font-bold text-white">
                    {user?.displayName?.charAt(0)?.toUpperCase() ?? 'U'}
                  </span>
                )}
                <span className="hidden sm:inline">{user?.displayName}</span>
              </button>
              {menuOpen && (
                <div
                  className="absolute right-0 top-full mt-2 w-40 rounded-lg border border-brand-100 bg-[#fffdf9] py-1 shadow-lg shadow-stone-200/60"
                  onMouseLeave={() => setMenuOpen(false)}
                >
                  <div className="border-b border-brand-100 px-3 py-2 text-xs text-stone-500">
                    {user?.displayName}
                  </div>
                  <button onClick={handleLogout}
                    className="w-full px-3 py-2 text-left text-sm text-stone-600 hover:bg-brand-50 hover:text-brand-800">
                    退出登录
                  </button>
                </div>
              )}
            </div>
          ) : (
            <Link href="/login"
              className="ml-2 rounded-md bg-brand-700 px-3 py-1.5 text-sm font-medium text-white shadow-sm shadow-brand-200 transition-colors hover:bg-brand-800">
              登录
            </Link>
          )}
        </div>
      </nav>
    </header>
  );
}
