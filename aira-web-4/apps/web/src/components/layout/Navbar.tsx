/**
 * components/layout/Navbar.tsx
 * 顶部导航 — 首页 / 课程 / 个人中心 / 登录状态
 */

'use client';

import Link from 'next/link';
import { usePathname, useRouter } from 'next/navigation';
import { useAuth } from '@/lib/auth';
import { useEffect, useRef, useState } from 'react';

export function Navbar() {
  const pathname = usePathname();
  const router = useRouter();
  const { user, isLoggedIn, logout } = useAuth();
  const [menuOpen, setMenuOpen] = useState(false);
  const menuRef = useRef<HTMLDivElement | null>(null);

  const handleLogout = async () => {
    await logout();
    router.push('/login');
  };

  const links = [
    { href: '/', label: '首页' },
    { href: '/courses', label: '课程' },
    ...(isLoggedIn ? [{ href: '/profile', label: '个人中心' }] : []),
    ...(user?.roles?.includes('admin') ? [{ href: '/admin/reviews', label: '管理审核' }] : []),
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
    <header className="sticky top-0 z-40 border-b border-gray-200 bg-white/90 backdrop-blur-sm">
      <nav className="mx-auto flex h-14 max-w-5xl items-center justify-between px-4">
        {/* Logo */}
        <Link href="/" className="flex items-center gap-2 font-semibold text-brand-800">
          <span className="flex h-7 w-7 items-center justify-center rounded bg-brand-700 text-xs font-bold text-white">
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
                    ? 'bg-brand-50 font-medium text-brand-700'
                    : 'text-gray-600 hover:bg-gray-100'
                }`}>
                {link.label}
              </Link>
            );
          })}

          {/* 登录/用户 */}
          {isLoggedIn ? (
            <div className="relative ml-2" ref={menuRef}>
              <button onClick={() => setMenuOpen((prev) => !prev)}
                className="flex items-center gap-2 rounded-md px-3 py-1.5 text-sm text-gray-600 hover:bg-gray-100">
                <span className="flex h-6 w-6 items-center justify-center rounded-full bg-brand-600 text-xs font-bold text-white">
                  {user?.displayName?.charAt(0)?.toUpperCase() ?? 'U'}
                </span>
                <span className="hidden sm:inline">{user?.displayName}</span>
              </button>
              {menuOpen && (
                <div
                  className="absolute right-0 top-full mt-1 w-40 rounded-lg border border-gray-200 bg-white py-1 shadow-lg"
                  onMouseLeave={() => setMenuOpen(false)}
                >
                  <div className="border-b border-gray-100 px-3 py-2 text-xs text-gray-500">
                    {user?.displayName}
                  </div>
                  <button onClick={handleLogout}
                    className="w-full px-3 py-2 text-left text-sm text-gray-600 hover:bg-gray-50">
                    退出登录
                  </button>
                </div>
              )}
            </div>
          ) : (
            <Link href="/login"
              className="ml-2 rounded-md bg-brand-600 px-3 py-1.5 text-sm font-medium text-white hover:bg-brand-700">
              登录
            </Link>
          )}
        </div>
      </nav>
    </header>
  );
}
