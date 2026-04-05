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
  email: string;
  password: string;
  confirmPassword: string;
  verificationCode: string;
  agreeToPolicy: boolean;
}

/** POST /api/auth/register — 响应 data */
export interface RegisterData extends LoginData {
  onboardingTasks: string[];
}

/** POST /api/auth/verification-code — 请求 */
export interface VerificationCodeDto {
  email: string;
}

/** POST /api/auth/verification-code — 响应 data */
export interface VerificationCodeData {
  sent: boolean;
  code?: string;
  expiresIn: number;
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
  code: string;
  name: string;
  college: string;
  category: string;
  credits: number;
  description: string;
}

export interface CourseComment {
  id: string | number;
  course_id?: string;
  user_id?: string | number;
  user_name?: string;
  comment: string;
  created_at?: string;
  updated_at?: string;
}

export interface TeacherComment extends CourseComment {
  teacher_id: string;
  teacher_name?: string;
}

export interface GradingStandard {
  id: string | number;
  course_id?: string;
  teacher_id: string;
  teacher_name?: string;
  description?: string;
  standard?: string;
  standard_img?: string;
  created_at?: string;
  updated_at?: string;
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
  source_id?: string;
  order: number;
  sequence_id?: number;
  question_type?: string;
  category?: string;
  source_url?: string;
  test: string;             // 题干
  options: ProblemOption[];
  answer: string;           // 正确答案，如 "B"
  explanation?: string;
  score?: number;
}

export interface ProblemExplanationItem {
  id: number;
  problem_id: number;
  author_id: number;
  author_name: string;
  content_md: string;
  up_votes: number;
  down_votes: number;
  my_vote: number;
  can_edit: boolean;
  created_at: string;
  updated_at: string;
}

export interface ProblemExplanationListData {
  official_explanation: string;
  items: ProblemExplanationItem[];
  my_item?: ProblemExplanationItem | null;
}

export interface UpsertProblemExplanationDto {
  content_md: string;
}

export interface VoteProblemExplanationDto {
  value: -1 | 0 | 1;
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
  course_id: string;
  course_name: string;
  added_at: string;
  problem_details: FavoriteProblemDetails;
}

export interface FavoriteCourseGroup {
  course_id: string;
  course_name: string;
  items: FavoriteItem[];
}

/** GET /api/favorites — 分页响应 data */
export interface FavoriteListData {
  total: number;
  page: number;
  size: number;
  items: FavoriteItem[];
  groups: FavoriteCourseGroup[];
}

export type FavoriteIdList = number[];

/** POST /api/favorites — 请求 */
export interface AddFavoriteDto {
  problem_id: number;
}

/* ══════════ answers / wrongbook / profile ══════════ */

export interface AnswerRecord {
  id: number;
  user_id: number;
  course_id: string;
  paper_id: number;
  problem_id: number;
  selected_option: string;
  is_correct: boolean;
  mode: string;
  answered_at: string;
}

export interface AnswerRecordListData {
  total: number;
  page: number;
  size: number;
  items: AnswerRecord[];
}

export interface WrongBookItem {
  problem_id: number;
  paper_id: number;
  order: number;
  test: string;
  status: string;
  note: string;
  wrong_count: number;
  last_wrong_at: string;
}

export interface WrongBookCourseGroup {
  course_id: string;
  course_name: string;
  last_practice_at?: string;
  items: WrongBookItem[];
}

export interface WrongBookData {
  courses: WrongBookCourseGroup[];
}

export interface UserProfile {
  id: number;
  user_id: number;
  username: string;
  email: string;
  nickname: string;
  avatar_url: string;
  level: number;
  created_at: string;
  updated_at: string;
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
