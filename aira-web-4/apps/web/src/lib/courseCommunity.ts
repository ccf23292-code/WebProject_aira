import type {
  CourseComment,
  TeacherComment,
  GradingStandard,
} from '@aira/shared';
import { api } from './api';

export interface TeacherDirectoryEntry {
  id: string;
  name: string;
  title?: string;
}

interface AddCourseCommentInput {
  comment: string;
  userName?: string;
}

interface AddTeacherCommentInput extends AddCourseCommentInput {
  teacherName?: string;
}

interface AddGradingStandardInput {
  description?: string;
  standard?: string;
  standard_img?: string;
  teacherName?: string;
}

const DEFAULT_TEACHERS: Record<string, TeacherDirectoryEntry[]> = {
  'course-101': [
    { id: 't-zhang', name: '张老师', title: '微积分' },
    { id: 't-li', name: '李老师', title: '习题课' },
  ],
  'course-201': [
    { id: 't-wang', name: '王老师', title: '数据结构' },
  ],
};

const DEFAULT_COURSE_COMMENTS: Record<string, CourseComment[]> = {
  'course-101': [
    {
      id: 'seed-course-101-1',
      course_id: 'course-101',
      user_name: 'AIRA 同学',
      comment: '期末卷覆盖面比较全，先刷真题再看回忆卷会更顺。',
      created_at: '2026-03-18T09:20:00.000Z',
    },
  ],
  'course-201': [
    {
      id: 'seed-course-201-1',
      course_id: 'course-201',
      user_name: '算法小组',
      comment: '线性表、树和排序是高频区域，适合先做课程广场里的近两年卷子。',
      created_at: '2026-03-11T14:10:00.000Z',
    },
  ],
};

const DEFAULT_TEACHER_COMMENTS: Record<string, TeacherComment[]> = {
  'course-101::t-zhang': [
    {
      id: 'seed-teacher-101-1',
      course_id: 'course-101',
      teacher_id: 't-zhang',
      teacher_name: '张老师',
      user_name: '期末复习组',
      comment: '讲义很完整，作业题和考试思路接近，适合配套整理错题。',
      created_at: '2026-03-19T10:00:00.000Z',
    },
  ],
  'course-201::t-wang': [
    {
      id: 'seed-teacher-201-1',
      course_id: 'course-201',
      teacher_id: 't-wang',
      teacher_name: '王老师',
      user_name: '刷题伙伴',
      comment: '大题喜欢考树和图的综合应用，平时实验题也建议一起复盘。',
      created_at: '2026-03-15T08:30:00.000Z',
    },
  ],
};

const DEFAULT_GRADING_STANDARDS: Record<string, GradingStandard[]> = {
  'course-101::t-zhang': [
    {
      id: 'seed-standard-101-1',
      course_id: 'course-101',
      teacher_id: 't-zhang',
      teacher_name: '张老师',
      description: '平时分重作业和随堂测，期末前整理错题会比较划算。',
      standard: '平时 40%，期中 20%，期末 40%。',
      created_at: '2026-03-21T12:00:00.000Z',
    },
  ],
};

function teacherDirectoryKey(courseId: string) {
  return `aira:teachers:${courseId}`;
}

function courseCommentsKey(courseId: string) {
  return `aira:course-comments:${courseId}`;
}

function teacherCommentsKey(courseId: string, teacherId: string) {
  return `aira:teacher-comments:${courseId}:${teacherId}`;
}

function gradingStandardsKey(courseId: string, teacherId: string) {
  return `aira:grading-standards:${courseId}:${teacherId}`;
}

function isBrowser() {
  return typeof window !== 'undefined';
}

function readStorage<T>(key: string, fallback: T): T {
  if (!isBrowser()) return fallback;
  try {
    const raw = window.localStorage.getItem(key);
    return raw ? (JSON.parse(raw) as T) : fallback;
  } catch {
    return fallback;
  }
}

function writeStorage<T>(key: string, value: T) {
  if (!isBrowser()) return;
  window.localStorage.setItem(key, JSON.stringify(value));
}

function sortNewestFirst<T extends { created_at?: string; updated_at?: string }>(items: T[]) {
  return [...items].sort((left, right) => {
    const leftTime = new Date(left.updated_at ?? left.created_at ?? 0).getTime();
    const rightTime = new Date(right.updated_at ?? right.created_at ?? 0).getTime();
    return rightTime - leftTime;
  });
}

function listFromResponse<T>(value: unknown): T[] {
  if (Array.isArray(value)) return value as T[];
  if (
    value &&
    typeof value === 'object' &&
    'items' in value &&
    Array.isArray((value as { items?: unknown[] }).items)
  ) {
    return (value as { items: T[] }).items;
  }
  return [];
}

function getStringValue(value: unknown, fallback = '') {
  return typeof value === 'string' ? value : fallback;
}

function normalizeCourseComment(item: unknown, fallbackCourseId: string): CourseComment {
  const record = (item ?? {}) as Record<string, unknown>;
  return {
    id: String(record.id ?? `course-comment-${Date.now()}`),
    course_id: getStringValue(record.course_id, fallbackCourseId),
    user_id: typeof record.user_id === 'string' || typeof record.user_id === 'number'
      ? record.user_id
      : undefined,
    user_name: getStringValue(
      record.user_name ??
      record.author_name ??
      record.display_name ??
      record.nickname,
      '匿名同学',
    ),
    comment: getStringValue(record.comment ?? record.content ?? record.body),
    created_at: getStringValue(record.created_at),
    updated_at: getStringValue(record.updated_at),
  };
}

function normalizeTeacherComment(
  item: unknown,
  fallbackCourseId: string,
  fallbackTeacherId: string,
): TeacherComment {
  const record = (item ?? {}) as Record<string, unknown>;
  return {
    ...normalizeCourseComment(record, fallbackCourseId),
    teacher_id: getStringValue(record.teacher_id, fallbackTeacherId),
    teacher_name: getStringValue(record.teacher_name ?? record.teacher ?? record.teacher_label),
  };
}

function normalizeGradingStandard(
  item: unknown,
  fallbackCourseId: string,
  fallbackTeacherId: string,
): GradingStandard {
  const record = (item ?? {}) as Record<string, unknown>;
  return {
    id: String(record.id ?? `grading-standard-${Date.now()}`),
    course_id: getStringValue(record.course_id, fallbackCourseId),
    teacher_id: getStringValue(record.teacher_id, fallbackTeacherId),
    teacher_name: getStringValue(record.teacher_name ?? record.teacher ?? record.teacher_label),
    description: getStringValue(record.description),
    standard: getStringValue(record.standard),
    standard_img: getStringValue(record.standard_img ?? record.image ?? record.image_url),
    created_at: getStringValue(record.created_at),
    updated_at: getStringValue(record.updated_at),
  };
}

function getDefaultTeacherComments(courseId: string, teacherId: string) {
  return DEFAULT_TEACHER_COMMENTS[`${courseId}::${teacherId}`] ?? [];
}

function getDefaultGradingStandards(courseId: string, teacherId: string) {
  return DEFAULT_GRADING_STANDARDS[`${courseId}::${teacherId}`] ?? [];
}

function createLocalCourseComment(
  courseId: string,
  comment: string,
  userName?: string,
): CourseComment {
  const now = new Date().toISOString();
  return {
    id: `local-course-comment-${Date.now()}`,
    course_id: courseId,
    user_name: userName || '当前用户',
    comment,
    created_at: now,
    updated_at: now,
  };
}

function createLocalTeacherComment(
  courseId: string,
  teacherId: string,
  comment: string,
  userName?: string,
  teacherName?: string,
): TeacherComment {
  const now = new Date().toISOString();
  return {
    id: `local-teacher-comment-${Date.now()}`,
    course_id: courseId,
    teacher_id: teacherId,
    teacher_name: teacherName,
    user_name: userName || '当前用户',
    comment,
    created_at: now,
    updated_at: now,
  };
}

function createLocalGradingStandard(
  courseId: string,
  teacherId: string,
  input: AddGradingStandardInput,
): GradingStandard {
  const now = new Date().toISOString();
  return {
    id: `local-grading-standard-${Date.now()}`,
    course_id: courseId,
    teacher_id: teacherId,
    teacher_name: input.teacherName,
    description: input.description?.trim(),
    standard: input.standard?.trim(),
    standard_img: input.standard_img?.trim(),
    created_at: now,
    updated_at: now,
  };
}

export function getTeacherDirectory(courseId: string): TeacherDirectoryEntry[] {
  const seeded = DEFAULT_TEACHERS[courseId] ?? [];
  const stored = readStorage<TeacherDirectoryEntry[]>(teacherDirectoryKey(courseId), []);
  const merged = [...seeded, ...stored];
  const deduped = new Map<string, TeacherDirectoryEntry>();

  for (const teacher of merged) {
    const teacherId = teacher.id.trim();
    if (!teacherId) continue;
    deduped.set(teacherId, {
      id: teacherId,
      name: teacher.name?.trim() || teacherId,
      title: teacher.title?.trim() || undefined,
    });
  }

  return [...deduped.values()];
}

export function saveTeacherDirectoryEntry(
  courseId: string,
  teacher: Omit<TeacherDirectoryEntry, 'id'> & { id?: string },
) {
  const current = getTeacherDirectory(courseId);
  const teacherId = teacher.id?.trim() || `t-${Date.now()}`;
  const filtered = current.filter((item) => item.id !== teacherId);
  const next = [
    ...filtered,
    {
      id: teacherId,
      name: teacher.name.trim() || teacherId,
      title: teacher.title?.trim() || undefined,
    },
  ];
  writeStorage(teacherDirectoryKey(courseId), next);
  return next;
}

export async function getCourseComments(courseId: string) {
  try {
    const response = await api.get<unknown>(`/courses/${encodeURIComponent(courseId)}/comments`);
    return sortNewestFirst(
      listFromResponse<unknown>(response).map((item) => normalizeCourseComment(item, courseId)),
    );
  } catch {
    const stored = readStorage<CourseComment[]>(courseCommentsKey(courseId), []);
    const seeded = DEFAULT_COURSE_COMMENTS[courseId] ?? [];
    return sortNewestFirst([...stored, ...seeded]);
  }
}

export async function addCourseComment(courseId: string, input: AddCourseCommentInput) {
  try {
    const response = await api.post<unknown>(
      `/courses/${encodeURIComponent(courseId)}/comments`,
      { comment: input.comment.trim() },
    );
    return normalizeCourseComment(response, courseId);
  } catch {
    const created = createLocalCourseComment(courseId, input.comment.trim(), input.userName);
    const current = readStorage<CourseComment[]>(courseCommentsKey(courseId), []);
    writeStorage(courseCommentsKey(courseId), sortNewestFirst([created, ...current]));
    return created;
  }
}

export async function getTeacherComments(courseId: string, teacherId: string) {
  try {
    const response = await api.get<unknown>(
      `/courses/${encodeURIComponent(courseId)}/teachers/${encodeURIComponent(teacherId)}/comments`,
    );
    return sortNewestFirst(
      listFromResponse<unknown>(response).map((item) => (
        normalizeTeacherComment(item, courseId, teacherId)
      )),
    );
  } catch {
    const stored = readStorage<TeacherComment[]>(teacherCommentsKey(courseId, teacherId), []);
    const seeded = getDefaultTeacherComments(courseId, teacherId);
    return sortNewestFirst([...stored, ...seeded]);
  }
}

export async function addTeacherComment(
  courseId: string,
  teacherId: string,
  input: AddTeacherCommentInput,
) {
  try {
    const response = await api.post<unknown>(
      `/courses/${encodeURIComponent(courseId)}/teachers/${encodeURIComponent(teacherId)}/comments`,
      { comment: input.comment.trim() },
    );
    return normalizeTeacherComment(response, courseId, teacherId);
  } catch {
    const created = createLocalTeacherComment(
      courseId,
      teacherId,
      input.comment.trim(),
      input.userName,
      input.teacherName,
    );
    const current = readStorage<TeacherComment[]>(teacherCommentsKey(courseId, teacherId), []);
    writeStorage(teacherCommentsKey(courseId, teacherId), sortNewestFirst([created, ...current]));
    return created;
  }
}

export async function getGradingStandards(courseId: string, teacherId: string) {
  try {
    const response = await api.get<unknown>(
      `/courses/${encodeURIComponent(courseId)}/teachers/${encodeURIComponent(teacherId)}/grading-standards`,
    );
    return sortNewestFirst(
      listFromResponse<unknown>(response).map((item) => (
        normalizeGradingStandard(item, courseId, teacherId)
      )),
    );
  } catch {
    const stored = readStorage<GradingStandard[]>(gradingStandardsKey(courseId, teacherId), []);
    const seeded = getDefaultGradingStandards(courseId, teacherId);
    return sortNewestFirst([...stored, ...seeded]);
  }
}

export async function addGradingStandard(
  courseId: string,
  teacherId: string,
  input: AddGradingStandardInput,
) {
  try {
    const response = await api.post<unknown>(
      `/courses/${encodeURIComponent(courseId)}/teachers/${encodeURIComponent(teacherId)}/grading-standards`,
      {
        description: input.description?.trim(),
        standard: input.standard?.trim(),
        standard_img: input.standard_img?.trim(),
      },
    );
    return normalizeGradingStandard(response, courseId, teacherId);
  } catch {
    const created = createLocalGradingStandard(courseId, teacherId, input);
    const current = readStorage<GradingStandard[]>(gradingStandardsKey(courseId, teacherId), []);
    writeStorage(gradingStandardsKey(courseId, teacherId), sortNewestFirst([created, ...current]));
    return created;
  }
}
