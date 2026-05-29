/**
 * lib/attempt.ts
 * 严格练习模式（attempt + submission）的 API 客户端
 *
 * 后端接口：
 *   POST /api/papers/:paper_id/attempts                          → 新建尝试，旧 in_progress 自动 abandon
 *   GET  /api/attempts/:attempt_id                               → 拉聚合 + 所有 submissions
 *   POST /api/attempts/:attempt_id/problems/:problem_id/submit  → 提交单题（重复 → 409 already_submitted）
 *
 * 使用原生 fetch（而非 lib/api.ts）是为了把 HTTP status 透出到上层，方便：
 *   - 409 already_submitted     → 重拉状态
 *   - 409 attempt_closed        → toast "进度已过期"
 */

const API_BASE = process.env.NEXT_PUBLIC_API_URL ?? 'http://localhost:3001/api';

interface ApiEnvelope<T> {
  code: number;
  message: string;
  data: T;
}

/** 聚合：与后端 models.PaperAttempt 对齐 */
export interface PaperAttempt {
  id: number;
  user_id: number;
  paper_id: number;
  course_id: string;
  status: 'in_progress' | 'completed' | 'abandoned';
  score: number;
  max_score: number;
  correct: number;
  submitted: number;
  total: number;
  started_at: string;
  updated_at: string;
  completed_at?: string | null;
}

/** 单题提交：与后端 models.ProblemSubmission 对齐 */
export interface ProblemSubmission {
  id: number;
  attempt_id: number;
  problem_id: number;
  user_id: number;
  user_answer: string;
  is_correct: boolean;
  score: number;
  submitted_at: string;
}

export interface CreateAttemptResult {
  attempt: PaperAttempt;
  total: number;
  /** true=新建（旧 in_progress 被 abandon，或本来没有）；false=复用了已有的 in_progress */
  created: boolean;
}

export interface AttemptDetail {
  attempt: PaperAttempt;
  submissions: ProblemSubmission[];
}

export interface SubmitProblemResult {
  submission: ProblemSubmission;
  correct_answer: string;
  attempt: PaperAttempt;
}

/** 带 HTTP 状态的错误，方便上层区分 409 / 403 / 5xx */
export class AttemptApiError extends Error {
  status: number;
  constructor(message: string, status: number) {
    super(message);
    this.name = 'AttemptApiError';
    this.status = status;
  }
}

function authHeader(): Record<string, string> {
  if (typeof window === 'undefined') return {};
  const token = window.localStorage.getItem('accessToken');
  return token ? { Authorization: `Bearer ${token}` } : {};
}

async function request<T>(path: string, init: RequestInit): Promise<T> {
  const res = await fetch(`${API_BASE}${path}`, {
    ...init,
    headers: {
      'Content-Type': 'application/json',
      ...authHeader(),
      ...(init.headers as Record<string, string> | undefined),
    },
  });

  let body: ApiEnvelope<T> | null = null;
  try {
    body = (await res.json()) as ApiEnvelope<T>;
  } catch {
    /* ignore */
  }

  if (!res.ok) {
    const message = body?.message ?? `请求失败: ${res.status}`;
    throw new AttemptApiError(message, res.status);
  }
  return body!.data;
}

/**
 * POST /api/papers/:paper_id/attempts
 *
 * @param paperId    试卷 ID
 * @param forceReset true=强制开启新一轮（abandon 旧 in_progress）；false=默认恢复进度
 */
export function createAttempt(
  paperId: number,
  forceReset = false,
): Promise<CreateAttemptResult> {
  return request<CreateAttemptResult>(`/papers/${paperId}/attempts`, {
    method: 'POST',
    body: JSON.stringify({ force_reset: forceReset }),
  });
}

/** GET /api/attempts/:attempt_id */
export function fetchAttempt(attemptId: number): Promise<AttemptDetail> {
  return request<AttemptDetail>(`/attempts/${attemptId}`, { method: 'GET' });
}

/** POST /api/attempts/:attempt_id/problems/:problem_id/submit */
export function submitProblem(
  attemptId: number,
  problemId: number,
  userAnswer: string,
): Promise<SubmitProblemResult> {
  return request<SubmitProblemResult>(
    `/attempts/${attemptId}/problems/${problemId}/submit`,
    {
      method: 'POST',
      body: JSON.stringify({ user_answer: userAnswer }),
    },
  );
}
