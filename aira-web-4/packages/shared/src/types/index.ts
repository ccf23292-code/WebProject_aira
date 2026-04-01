/**
 * packages/shared/src/types/index.ts
 * 类型定义 — 严格对齐后端 API 响应 JSON 结构
 *
 * 来源：后端同学提供的接口文档
 * 模块：auth_module / browse_module / favorite_module / admin_module
 */

import { UserRole } from '../enums';

/* ══════════ 通用响应包装 ══════════ */

export interface ApiResponse<T> {
  code: number;
  message: string;
  data: T;
}

/* ══════════ auth_module ══════════ */

/** POST /api/auth/login — 请求 */
export interface LoginDto {
  username: string;   // 后端字段名待确认，先用 username
  password: string;
}

/** POST /api/auth/login — 响应 data */
export interface LoginData {
  userId: string;
  displayName: string;
  accessToken: string;
  refreshToken: string;
  roles: UserRole[];
  expiresIn: number;  // 秒
}

/** POST /api/auth/register — 请求 */
export interface RegisterDto {
  username: string;
  password: string;
}

/** POST /api/auth/register — 响应 data */
export interface RegisterData extends LoginData {
  onboardingTasks: string[];
}

/** POST /api/auth/logout — 响应 data */
export interface LogoutData {
  success: boolean;
  message: string;
}

/* ══════════ browse_module ══════════ */

/** GET /api/courses — 课程 */
export interface Course {
  id: string;
  name: string;
  description: string;
}

/** GET /api/courses/{course_id}/papers — 试卷 */
export interface Paper {
  id: number;
  course_id: string;
  name: string;
  created_at: string;  // ISO 8601
}

/** 选项 */
export interface ProblemOption {
  option: string;  // "A" / "B" / "C" / "D"
  text: string;
}

/** GET /api/papers/{paper_id}/problems — 题目 */
export interface Problem {
  id: number;
  testpaper_id: number;
  order: number;
  test: string;             // 题干
  options: ProblemOption[];
  answer: string;           // 正确答案，如 "B"
}

/* ══════════ favorite_module ══════════ */

/** 收藏项中的题目摘要 */
export interface FavoriteProblemDetails {
  testpaper_name: string;
  order: number;
  test: string;
}

/** GET /api/favorites — 收藏项 */
export interface FavoriteItem {
  favorite_id: number;
  problem_id: number;
  added_at: string;
  problem_details: FavoriteProblemDetails;
}

/** GET /api/favorites — 分页响应 data */
export interface FavoriteListData {
  total: number;
  page: number;
  size: number;
  items: FavoriteItem[];
}

/** POST /api/favorites — 请求 */
export interface AddFavoriteDto {
  problem_id: number;
}

/* ══════════ admin_module ══════════ */

/** POST /api/admin/papers — 上传试卷响应 data */
export interface UploadPaperData {
  paper_id: number;
  inserted_problems_count: number;
}

/* ══════════ recall_module (回忆卷) ══════════ */

/** 回忆卷 */
export interface RecallPaper {
  id: number;
  course_id: string;
  title: string;
  created_by: string;       // userId
  created_at: string;
  updated_at: string;
}

/** 题型 + 当前最大题号 */
export interface QuestionTypeInfo {
  question_type: string;     // "singleChoice" | "multiChoice" | "fillBlank" | "shortAnswer" | ...
  max_sequence: number;
}

/** 回忆卷题目（完整） */
export interface RecallQuestion {
  id: number;
  paper_id: number;
  question_type: string;
  sequence: number;
  content: string;           // Markdown
  answer: string;            // Markdown，可为空
  options: ProblemOption[];  // 选择题有，其他题型为 []
  source_user_id: string;
  support_count: number;
  last_editor_id: string | null;
  created_at: string;
  updated_at: string;
}

/** 新增题目请求 */
export interface CreateRecallQuestionDto {
  question_type: string;
  sequence: number;
  content: string;
  answer?: string;
  options?: ProblemOption[];
}

/** 编辑题目请求 */
export interface PatchRecallQuestionDto {
  content?: string;
  answer?: string;
  options?: ProblemOption[];
}

/** 评论 */
export interface RecallComment {
  id: number;
  question_id: number;
  user_id: string;
  display_name?: string;     // 后端 join 返回
  content: string;
  created_at: string;
  updated_at: string;
}

/** 评论分页响应 */
export interface RecallCommentListData {
  total: number;
  page: number;
  size: number;
  items: RecallComment[];
}
