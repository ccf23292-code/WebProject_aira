import type {
  AddTeacherDto,
  CourseComment,
  GradingStandard,
  TeacherComment,
  TeacherDirectoryEntry,
} from '@aira/shared';
import { api } from './api';

interface AddCourseCommentInput {
  comment: string;
}

interface AddTeacherCommentInput {
  comment: string;
}

interface AddGradingStandardInput {
  description?: string;
  standard?: string;
  standard_img?: string;
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

function normalizeTeacher(item: unknown, fallbackCourseId: string): TeacherDirectoryEntry {
  const record = (item ?? {}) as Record<string, unknown>;
  return {
    id: getStringValue(record.id, `teacher-${Date.now()}`),
    course_id: getStringValue(record.course_id, fallbackCourseId),
    name: getStringValue(record.name, '未命名教师'),
    title: getStringValue(record.title) || undefined,
    created_at: getStringValue(record.created_at),
    updated_at: getStringValue(record.updated_at),
  };
}

function normalizeCourseComment(item: unknown, fallbackCourseId: string): CourseComment {
  const record = (item ?? {}) as Record<string, unknown>;
  return {
    id: String(record.id ?? `course-comment-${Date.now()}`),
    course_id: getStringValue(record.course_id, fallbackCourseId),
    user_id:
      typeof record.user_id === 'string' || typeof record.user_id === 'number'
        ? record.user_id
        : undefined,
    user_name: getStringValue(
      record.user_name ?? record.author_name ?? record.display_name ?? record.nickname,
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

export async function getTeacherDirectory(courseId: string) {
  const response = await api.get<unknown>(`/courses/${encodeURIComponent(courseId)}/teachers`);
  return listFromResponse<unknown>(response).map((item) => normalizeTeacher(item, courseId));
}

export async function addTeacherDirectoryEntry(courseId: string, teacher: AddTeacherDto) {
  const response = await api.post<unknown>(`/courses/${encodeURIComponent(courseId)}/teachers`, {
    id: teacher.id?.trim() || undefined,
    name: teacher.name.trim(),
    title: teacher.title?.trim() || undefined,
  });
  return normalizeTeacher(response, courseId);
}

export async function getCourseComments(courseId: string) {
  const response = await api.get<unknown>(`/courses/${encodeURIComponent(courseId)}/comments`);
  return sortNewestFirst(
    listFromResponse<unknown>(response).map((item) => normalizeCourseComment(item, courseId)),
  );
}

export async function addCourseComment(courseId: string, input: AddCourseCommentInput) {
  const response = await api.post<unknown>(`/courses/${encodeURIComponent(courseId)}/comments`, {
    comment: input.comment.trim(),
  });
  return normalizeCourseComment(response, courseId);
}

export async function getTeacherComments(courseId: string, teacherId: string) {
  const response = await api.get<unknown>(
    `/courses/${encodeURIComponent(courseId)}/teachers/${encodeURIComponent(teacherId)}/comments`,
  );
  return sortNewestFirst(
    listFromResponse<unknown>(response).map((item) =>
      normalizeTeacherComment(item, courseId, teacherId),
    ),
  );
}

export async function addTeacherComment(
  courseId: string,
  teacherId: string,
  input: AddTeacherCommentInput,
) {
  const response = await api.post<unknown>(
    `/courses/${encodeURIComponent(courseId)}/teachers/${encodeURIComponent(teacherId)}/comments`,
    { comment: input.comment.trim() },
  );
  return normalizeTeacherComment(response, courseId, teacherId);
}

export async function getGradingStandards(courseId: string, teacherId: string) {
  const response = await api.get<unknown>(
    `/courses/${encodeURIComponent(courseId)}/teachers/${encodeURIComponent(teacherId)}/grading-standards`,
  );
  return sortNewestFirst(
    listFromResponse<unknown>(response).map((item) =>
      normalizeGradingStandard(item, courseId, teacherId),
    ),
  );
}

export async function addGradingStandard(
  courseId: string,
  teacherId: string,
  input: AddGradingStandardInput,
) {
  const response = await api.post<unknown>(
    `/courses/${encodeURIComponent(courseId)}/teachers/${encodeURIComponent(teacherId)}/grading-standards`,
    {
      description: input.description?.trim(),
      standard: input.standard?.trim(),
      standard_img: input.standard_img?.trim(),
    },
  );
  return normalizeGradingStandard(response, courseId, teacherId);
}
