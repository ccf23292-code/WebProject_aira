/**
 * lib/checkin.ts
 * 每日签到模块的 API 客户端
 *
 * 对齐后端接口：
 *   GET  /api/checkin/today  → 查询当前签到状态
 *   POST /api/checkin        → 提交签到（重复签到返回 409 already_checked）
 *
 * 使用原生 fetch 而非 lib/api.ts，是因为我们需要在 catch 中区分 409 与
 * 其它错误，做"今日已签到"的优雅降级，而通用 api 客户端将所有非 2xx 抛
 * 成同一个 Error，无法读取 HTTP status。
 */

const API_BASE = process.env.NEXT_PUBLIC_API_URL ?? 'http://localhost:3001/api';

/** 后端响应包装 */
interface ApiEnvelope<T> {
  code: number;
  message: string;
  data: T;
}

/** 签到状态：GET /api/checkin/today 与 POST /api/checkin 返回结构一致 */
export interface CheckinStatus {
  checked_today: boolean;
  last_checkin_date: string;
  continuous_days: number;
  max_continuous: number;
  total_days: number;
}

/** 带 HTTP 状态的错误，方便上层区分 409 / 401 / 5xx */
export class CheckinApiError extends Error {
  status: number;
  code: string;
  constructor(message: string, status: number, code: string) {
    super(message);
    this.name = 'CheckinApiError';
    this.status = status;
    this.code = code;
  }
}

function authHeaders(): Record<string, string> {
  if (typeof window === 'undefined') return {};
  const token = window.localStorage.getItem('accessToken');
  return token ? { Authorization: `Bearer ${token}` } : {};
}

async function request<T>(path: string, init: RequestInit): Promise<T> {
  const res = await fetch(`${API_BASE}${path}`, {
    ...init,
    headers: {
      'Content-Type': 'application/json',
      ...authHeaders(),
      ...(init.headers as Record<string, string> | undefined),
    },
  });

  let body: ApiEnvelope<T> | null = null;
  try {
    body = (await res.json()) as ApiEnvelope<T>;
  } catch {
    // 服务器没有返回 JSON 的极端情况
  }

  if (!res.ok) {
    const message = body?.message ?? `请求失败: ${res.status}`;
    // 后端会在 message 里描述业务原因；此处再透出业务 code（"already_checked" 等）
    // 当前后端 utils.JSONError 没回传业务 code，只回 status，所以暂时只用 status 判别。
    throw new CheckinApiError(message, res.status, '');
  }

  return body!.data;
}

/** 查询签到状态 */
export function getCheckinStatus(): Promise<CheckinStatus> {
  return request<CheckinStatus>('/checkin/today', { method: 'GET' });
}

/** 提交签到 */
export function submitCheckin(): Promise<CheckinStatus> {
  return request<CheckinStatus>('/checkin', { method: 'POST', body: '{}' });
}
