import Link from 'next/link';
import { ReactNode } from 'react';

const tabs = [
  { href: '/profile', label: '资料' },
  { href: '/profile/favorites', label: '收藏' },
  { href: '/profile/wrongbook', label: '错题本' },
  { href: '/profile/records', label: '做题记录' },
];

export default function ProfileLayout({ children }: { children: ReactNode }) {
  return (
    <div className="grid gap-6 md:grid-cols-[220px_1fr]">
      <aside className="rounded-xl border border-gray-200 bg-white p-4">
        <h2 className="mb-3 text-sm font-semibold text-gray-800">个人中心</h2>
        <nav className="space-y-1">
          {tabs.map((tab) => (
            <Link
              key={tab.href}
              href={tab.href}
              className="block rounded-md px-3 py-2 text-sm text-gray-600 hover:bg-gray-50"
            >
              {tab.label}
            </Link>
          ))}
        </nav>
      </aside>
      <div>{children}</div>
    </div>
  );
}
