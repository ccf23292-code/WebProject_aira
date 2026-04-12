import type {
  AddHomepageMessageDto,
  HomepageMessage,
  UpdateHomepageMessageDto,
} from '@aira/shared';
import { api } from './api';

export function getHomepageMessages() {
  return api.get<HomepageMessage[]>('/homepage/messages', true);
}

export function addHomepageMessage(payload: AddHomepageMessageDto) {
  return api.post<HomepageMessage>('/homepage/messages', {
    content: payload.content.trim(),
  });
}

export function updateHomepageMessage(id: number, payload: UpdateHomepageMessageDto) {
  return api.put<HomepageMessage>(`/homepage/messages/${id}`, {
    content: payload.content.trim(),
  });
}

export function deleteHomepageMessage(id: number) {
  return api.delete<void>(`/homepage/messages/${id}`);
}
