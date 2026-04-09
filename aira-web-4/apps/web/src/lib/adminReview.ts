import type {
  CourseDescriptionSubmission,
  GradingStandardSubmission,
  ReviewCourseDescriptionDto,
  TeacherSubmission,
} from '@aira/shared';
import { api } from './api';

export function getCourseDescriptionSubmissions(status = 'pending') {
  return api.get<CourseDescriptionSubmission[]>(
    `/admin/course-description-submissions?status=${encodeURIComponent(status)}`,
  );
}

export function reviewCourseDescriptionSubmission(
  id: number,
  payload: ReviewCourseDescriptionDto,
) {
  return api.post<CourseDescriptionSubmission>(
    `/admin/course-description-submissions/${id}/review`,
    payload,
  );
}

export function getTeacherSubmissions(status = 'pending') {
  return api.get<TeacherSubmission[]>(
    `/admin/teacher-submissions?status=${encodeURIComponent(status)}`,
  );
}

export function reviewTeacherSubmission(
  id: number,
  payload: ReviewCourseDescriptionDto,
) {
  return api.post<TeacherSubmission>(`/admin/teacher-submissions/${id}/review`, payload);
}

export function getGradingStandardSubmissions(status = 'pending') {
  return api.get<GradingStandardSubmission[]>(
    `/admin/grading-standard-submissions?status=${encodeURIComponent(status)}`,
  );
}

export function reviewGradingStandardSubmission(
  id: number,
  payload: ReviewCourseDescriptionDto,
) {
  return api.post<GradingStandardSubmission>(
    `/admin/grading-standard-submissions/${id}/review`,
    payload,
  );
}
