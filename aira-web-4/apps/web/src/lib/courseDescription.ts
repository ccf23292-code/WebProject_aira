import type {
  CourseDescriptionSubmission,
  SubmitCourseDescriptionDto,
} from '@aira/shared';
import { api } from './api';

export function getMyCourseDescriptionSubmissions(courseId: string) {
  return api.get<CourseDescriptionSubmission[]>(
    `/courses/${encodeURIComponent(courseId)}/description-submissions/mine`,
  );
}

export function submitCourseDescriptionSuggestion(
  courseId: string,
  payload: SubmitCourseDescriptionDto,
) {
  return api.post<CourseDescriptionSubmission>(
    `/courses/${encodeURIComponent(courseId)}/description-submissions`,
    payload,
  );
}
