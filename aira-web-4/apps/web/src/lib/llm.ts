/**
 * lib/llm.ts
 * LLM 流式调用客户端 —— 通过 fetch + ReadableStream 手动解析 SSE
 *
 * 为什么不用 EventSource：
 *  - EventSource 不支持自定义请求头，无法附 Authorization Bearer
 *  - EventSource 只支持 GET，而我们需要 POST 传递 problem_id
 *
 * 服务端事件协议（与 routers/llm_controller.go 对齐）：
 *   event: token   data: {"text":"..."}
 *   event: done    data: {"reason":"stop"}
 *   event: error   data: {"message":"..."}
 */

const API_BASE = process.env.NEXT_PUBLIC_API_URL ?? 'http://localhost:3001/api';

export interface StreamHandlers {
  /** 每来一段 token 文本时回调；上层应做"追加到尾部" */
  onToken: (text: string) => void;
  /** 流正常结束 */
  onDone: (reason: string) => void;
  /** 流异常 / 网络错误 / 用户中断之外的失败 */
  onError: (message: string) => void;
}

function authHeader(): Record<string, string> {
  if (typeof window === 'undefined') return {};
  const token = window.localStorage.getItem('accessToken');
  return token ? { Authorization: `Bearer ${token}` } : {};
}

/**
 * 针对指定题目向后端发起流式 AI 解析请求。
 * 返回 AbortController；上层 "停止" 按钮调用 controller.abort() 即可。
 */
export function streamExplain(problemId: number, h: StreamHandlers): AbortController {
  const controller = new AbortController();

  (async () => {
    try {
      const res = await fetch(`${API_BASE}/llm/explain`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Accept: 'text/event-stream',
          ...authHeader(),
        },
        body: JSON.stringify({ problem_id: problemId }),
        signal: controller.signal,
      });

      // 非 200：服务器在 SSE 流开始前以普通 JSON 返回错误
      if (!res.ok) {
        let message = `请求失败: ${res.status}`;
        try {
          const body = await res.json();
          if (body?.message) message = body.message;
        } catch {
          /* ignore parse error */
        }
        h.onError(message);
        return;
      }

      if (!res.body) {
        h.onError('服务器未返回流');
        return;
      }

      const reader = res.body.getReader();
      const decoder = new TextDecoder('utf-8');
      let buffer = '';

      // 逐块读取并切分 SSE 事件（以 "\n\n" 分隔）
      while (true) {
        const { value, done } = await reader.read();
        if (done) break;
        buffer += decoder.decode(value, { stream: true });

        let sepIndex = buffer.indexOf('\n\n');
        while (sepIndex !== -1) {
          const rawEvent = buffer.slice(0, sepIndex);
          buffer = buffer.slice(sepIndex + 2);
          handleEvent(rawEvent, h);
          sepIndex = buffer.indexOf('\n\n');
        }
      }

      // 流被服务端正常关闭，但事件里可能没显式 done（边界情况）
      if (buffer.trim()) {
        handleEvent(buffer, h);
      }
    } catch (err: unknown) {
      // AbortError 视为用户主动停止，不当作错误上报
      if (err instanceof DOMException && err.name === 'AbortError') return;
      const message = err instanceof Error ? err.message : '网络异常';
      h.onError(message);
    }
  })();

  return controller;
}

/** 解析单个 SSE 事件块 —— 可能包含多行，按 "field: value" 拼装 */
function handleEvent(raw: string, h: StreamHandlers): void {
  const lines = raw.split('\n');
  let eventName = 'message';
  let dataPayload = '';

  for (const line of lines) {
    if (line.startsWith('event:')) {
      eventName = line.slice(6).trim();
    } else if (line.startsWith('data:')) {
      // 多行 data 字段需要换行拼接
      dataPayload = dataPayload ? dataPayload + '\n' + line.slice(5).trim() : line.slice(5).trim();
    }
    // 忽略 id: / retry: / 空行
  }

  if (!dataPayload) return;

  let payload: { text?: string; message?: string; reason?: string };
  try {
    payload = JSON.parse(dataPayload);
  } catch {
    // 非 JSON：当作 token 透传
    if (eventName === 'token') h.onToken(dataPayload);
    return;
  }

  switch (eventName) {
    case 'token':
      if (payload.text) h.onToken(payload.text);
      break;
    case 'done':
      h.onDone(payload.reason ?? 'stop');
      break;
    case 'error':
      h.onError(payload.message ?? 'LLM 调用失败');
      break;
    default:
      break;
  }
}
