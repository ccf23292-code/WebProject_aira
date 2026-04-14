/**
 * lib/api.ts
 * API 请求客户端 — 统一封装 fetch + Bearer Token 注入
 *
 * 后端响应格式：{ code, message, data }
 * 所有方法返回解包后的 data 字段
 */

const API_BASE = process.env.NEXT_PUBLIC_API_URL ?? 'http://zjuaira:3001/api';

/** 从 localStorage 读取 token（仅客户端） */
function getToken(): string | null {
  if (typeof window === 'undefined') return null;
  return localStorage.getItem('accessToken');
}

/** 统一请求封装 */
async function request<T>(
  path: string,
  options?: RequestInit & { noAuth?: boolean },
): Promise<T> {
  const { noAuth, ...fetchOptions } = options ?? {};
  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
    ...(fetchOptions.headers as Record<string, string>),
  };

  // 非公开接口自动注入 Authorization
  if (!noAuth) {
    const token = getToken();
    if (token) headers['Authorization'] = `Bearer ${token}`;
  }

  const url = `${API_BASE}${path}`;
  console.log(`[API] ${fetchOptions.method ?? 'GET'} ${url}`);

  const res = await fetch(url, { ...fetchOptions, headers });
  const body = await res.json();

  console.log(`[API] Response ${res.status}:`, body);

  if (!res.ok || body.code >= 400) {
    throw new Error(body.message ?? `请求失败: ${res.status}`);
  }

  // 解包：返回 body.data
  return body.data as T;
}

async function uploadRequest<T>(path: string, formData: FormData): Promise<T> {
  const headers: Record<string, string> = {};
  const token = getToken();
  if (token) headers['Authorization'] = `Bearer ${token}`;

  const res = await fetch(`${API_BASE}${path}`, {
    method: 'POST',
    body: formData,
    headers,
  });
  const body = await res.json();
  if (!res.ok || body.code >= 400) {
    throw new Error(body.message ?? `Request failed: ${res.status}`);
  }
  return body.data as T;
}

export const api = {
  get: <T>(path: string, noAuth = false) =>
    request<T>(path, { method: 'GET', noAuth }),

  post: <T>(path: string, data?: unknown, noAuth = false) =>
    request<T>(path, { method: 'POST', body: data ? JSON.stringify(data) : undefined, noAuth }),

  put: <T>(path: string, data?: unknown) =>
    request<T>(path, { method: 'PUT', body: data ? JSON.stringify(data) : undefined }),

  patch: <T>(path: string, data?: unknown) =>
    request<T>(path, { method: 'PATCH', body: data ? JSON.stringify(data) : undefined }),

  delete: <T>(path: string) =>
    request<T>(path, { method: 'DELETE' }),

  upload: <T>(path: string, formData: FormData) =>
    uploadRequest<T>(path, formData),
};
