/**
 * lib/auth.tsx
 * 认证上下文 — 管理登录状态 + Token 持久化
 *
 * 对齐后端接口：
 *   POST /api/auth/login    → 登录，存 token
 *   POST /api/auth/register → 注册，存 token
 *   POST /api/auth/logout   → 登出，清 token
 *
 * 用法：
 *   const { user, login, logout, isLoggedIn, updateAvatar } = useAuth();
 */

'use client';

import {
  createContext,
  useContext,
  useState,
  useCallback,
  useEffect,
  type ReactNode,
} from 'react';
import type { LoginData, RegisterData, RegisterDto } from '@aira/shared';
import { api } from './api';

interface AuthUser {
  userId: string;
  displayName: string;
  roles: string[];
  avatarUrl?: string;
}

interface AuthContextValue {
  user: AuthUser | null;
  isLoggedIn: boolean;
  loading: boolean;
  login: (username: string, password: string) => Promise<void>;
  register: (payload: RegisterDto) => Promise<void>;
  logout: () => Promise<void>;
  updateAvatar: (url: string) => void;
}

const AuthContext = createContext<AuthContextValue | null>(null);

function saveTokens(data: { accessToken: string; refreshToken: string }) {
  localStorage.setItem('accessToken', data.accessToken);
  localStorage.setItem('refreshToken', data.refreshToken);
}

function clearTokens() {
  localStorage.removeItem('accessToken');
  localStorage.removeItem('refreshToken');
  localStorage.removeItem('authUser');
}

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<AuthUser | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    try {
      const saved = localStorage.getItem('authUser');
      if (saved) setUser(JSON.parse(saved));
    } catch { /* ignore */ }
    setLoading(false);
  }, []);

  const login = useCallback(async (username: string, password: string) => {
    console.log('[Auth] login request:', { username, url: '/auth/login' });
    const data = await api.post<LoginData>('/auth/login', { username, password }, true);
    console.log('[Auth] login response:', data);
    saveTokens(data);
    const authUser: AuthUser = {
      userId: data.userId,
      displayName: data.displayName,
      roles: data.roles,
    };
    localStorage.setItem('authUser', JSON.stringify(authUser));
    setUser(authUser);
  }, []);

  const register = useCallback(async (payload: RegisterDto) => {
    console.log('[Auth] register request:', { username: payload.username, url: '/auth/register' });
    const data = await api.post<RegisterData>('/auth/register', payload, true);
    console.log('[Auth] register response:', data);
    saveTokens(data);
    const authUser: AuthUser = {
      userId: data.userId,
      displayName: data.displayName,
      roles: data.roles,
    };
    localStorage.setItem('authUser', JSON.stringify(authUser));
    setUser(authUser);
  }, []);

  /** 更新头像 URL — 上传成功后调用，同步到 localStorage 和 Navbar */
  const updateAvatar = useCallback((url: string) => {
    setUser((prev) => {
      if (!prev) return prev;
      const updated = { ...prev, avatarUrl: url };
      localStorage.setItem('authUser', JSON.stringify(updated));
      return updated;
    });
  }, []);

  const logout = useCallback(async () => {
    try {
      await api.post('/auth/logout', {
        refreshToken: localStorage.getItem('refreshToken') ?? '',
      });
    } catch { /* 即使请求失败也清除本地状态 */ }
    clearTokens();
    setUser(null);
  }, []);

  return (
    <AuthContext.Provider
      value={{ user, isLoggedIn: !!user, loading, login, register, logout, updateAvatar }}
    >
      {children}
    </AuthContext.Provider>
  );
}

export function useAuth(): AuthContextValue {
  const ctx = useContext(AuthContext);
  if (!ctx) throw new Error('useAuth must be used within AuthProvider');
  return ctx;
}
