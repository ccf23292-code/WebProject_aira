import type { AddHomepageMessageDto, HomepageMessage } from '@aira/shared';
import { api } from './api';

export function getHomepageMessages() {
  return api.get<HomepageMessage[]>('/homepage/messages', true);
}

export function addHomepageMessage(payload: AddHomepageMessageDto) {
  return api.post<HomepageMessage>('/homepage/messages', {
    content: payload.content.trim(),
  });
}
