/**
 * components/DailyFortune.tsx
 * 每日运势签到卡片（接入后端版）
 *
 * 状态来源：
 *  - 签到状态、连签天数完全由后端 user_checkins 表持久化
 *    GET  /api/checkin/today  → 拉取
 *    POST /api/checkin        → 提交
 *  - 运势内容（等级 + 宜忌 + 段子）仍在前端用伪随机生成
 *    种子 = hash(today + userId)，同用户同日刷新结果保持一致
 *
 * 业务约束：
 *  - 未登录用户：签到按钮置灰，文案"登录后解锁运势打卡"，不调用任何接口
 *  - 重复签到（HTTP 409）：静默把按钮切到"今日已签到"灰态，不显示错误红框
 *  - 其它错误（5xx / 网络异常）：显示行内提示，可重试
 */

'use client';

import { useEffect, useMemo, useState, type ReactNode } from 'react';
import { useAuth } from '@/lib/auth';
import {
  CheckinApiError,
  getCheckinStatus,
  submitCheckin,
  type CheckinStatus,
} from '@/lib/checkin';

/* ════════════════════ 语料库 ════════════════════ */

type FortuneTone =
  | 'great'
  | 'good'
  | 'small-good'
  | 'neutral'
  | 'small-bad'
  | 'bad';

interface FortuneLevel {
  label: string;
  weight: number;
  tone: FortuneTone;
}

const LEVELS: FortuneLevel[] = [
  { label: '大吉', weight: 1, tone: 'great' },
  { label: '吉',   weight: 3, tone: 'good' },
  { label: '小吉', weight: 5, tone: 'small-good' },
  { label: '中平', weight: 6, tone: 'neutral' },
  { label: '小凶', weight: 3, tone: 'small-bad' },
  { label: '凶',   weight: 1, tone: 'bad' },
];

const GOOD_THINGS: string[] = [
  '刷题，做一道会一道',
  '背书，看一遍就记下了',
  '早八不迟到',
  '组队大作业',
  '给学弟学妹答疑',
  '去图书馆占座',
  '主动找助教问问题',
  '提交一个 Pull Request',
  '吃食堂二楼的卤肉饭',
  '早睡早起',
  '复习上周的重点章节',
  '清空 ddl 列表里最小的一项',
  '把错题本里加星的题再过一遍',
  '在论坛回答一个问题',
  '给同学安利好用的工具',
  '试卷模拟考一遍',
  '认真做笔记',
  '提前一周开始期末复习',
  '约同学一起自习',
  '把代码 commit 推到远端',
];

const BAD_THINGS: string[] = [
  '熬夜内卷',
  '空腹做实验',
  '装弱',
  '反复刷 GPA',
  '蹲点抢热门选修课',
  '通宵赶 ddl',
  '答辩前一晚才开始改 PPT',
  '在群里复读',
  '硬刚助教',
  '强行翘早八',
  '一次性囤十本教材',
  'ddl 前一小时才动笔',
  '深夜下单麻辣烫',
  '在自习室外放视频',
  '点开成绩公布的评论区',
  '相信"明天再说"',
  '在朋友圈晒分数',
  '在自习室拆辣条',
  '盲选水课',
  '把代码改完不 commit 就关机',
];

const QUOTES: Record<FortuneTone, string[]> = {
  'great': [
    '看一遍就背下来了',
    '今天的脑子是租来的，记得还',
    '助教都来主动给你答疑',
    '随手一抽，必修拉满',
  ],
  'good': [
    '手感不错，不要浪费',
    '抓住这股劲，冲一波',
    '今天写代码 bug 都温柔',
  ],
  'small-good': [
    '平稳推进就好',
    '小步前进胜过原地踏步',
    '稳，但不必上头',
  ],
  'neutral': [
    '平平淡淡才是真',
    '该干嘛干嘛，别多想',
    '今日宜苟，明日再战',
  ],
  'small-bad': [
    '今天不宜上强度',
    '保住底线为先',
    '少做选择题，多做必做题',
  ],
  'bad': [
    '宜在被窝里苟一天',
    '今天写代码记得三思而后行',
    '建议关闭朋友圈静养',
  ],
};

/* ════════════════════ 伪随机算法 ════════════════════ */

function hashString(s: string): number {
  let h = 2166136261 >>> 0;
  for (let i = 0; i < s.length; i++) {
    h ^= s.charCodeAt(i);
    h = Math.imul(h, 16777619);
  }
  return h >>> 0;
}

function mulberry32(seed: number): () => number {
  let state = seed >>> 0;
  return function rand(): number {
    state = (state + 0x6d2b79f5) >>> 0;
    let t = state;
    t = Math.imul(t ^ (t >>> 15), t | 1);
    t ^= t + Math.imul(t ^ (t >>> 7), t | 61);
    return ((t ^ (t >>> 14)) >>> 0) / 4294967296;
  };
}

function pickWeighted<T extends { weight: number }>(items: T[], r: number): T {
  const total = items.reduce((s, it) => s + it.weight, 0);
  let acc = r * total;
  for (const it of items) {
    acc -= it.weight;
    if (acc <= 0) return it;
  }
  return items[items.length - 1];
}

function pickN<T>(items: readonly T[], n: number, rand: () => number): T[] {
  const pool = items.slice();
  const out: T[] = [];
  for (let i = 0; i < n && pool.length > 0; i++) {
    const idx = Math.floor(rand() * pool.length);
    out.push(pool[idx]);
    pool.splice(idx, 1);
  }
  return out;
}

/* ════════════════════ 日期工具 ════════════════════ */

function todayString(): string {
  const d = new Date();
  const yyyy = d.getFullYear();
  const mm = String(d.getMonth() + 1).padStart(2, '0');
  const dd = String(d.getDate()).padStart(2, '0');
  return `${yyyy}-${mm}-${dd}`;
}

/* ════════════════════ 运势计算 ════════════════════ */

interface FortuneResult {
  level: FortuneLevel;
  goods: string[];
  bads: string[];
  quote: string;
}

function generateFortune(dateStr: string, userKey: string): FortuneResult {
  const seed = hashString(`${dateStr}#${userKey}`);
  const rand = mulberry32(seed);

  const level = pickWeighted(LEVELS, rand());
  const goods = pickN(GOOD_THINGS, 2, rand);
  const bads = pickN(BAD_THINGS, 2, rand);
  const quotes = QUOTES[level.tone];
  const quote = quotes[Math.floor(rand() * quotes.length)];

  return { level, goods, bads, quote };
}

/* ════════════════════ 颜色映射 ════════════════════ */

const TONE_STYLES: Record<FortuneTone, { badge: string; ring: string; accent: string }> = {
  'great':      { badge: 'bg-rose-50 text-rose-700 border-rose-200',          ring: 'ring-rose-200/60',    accent: 'text-rose-600' },
  'good':       { badge: 'bg-amber-50 text-amber-700 border-amber-200',       ring: 'ring-amber-200/60',   accent: 'text-amber-600' },
  'small-good': { badge: 'bg-emerald-50 text-emerald-700 border-emerald-200', ring: 'ring-emerald-200/60', accent: 'text-emerald-600' },
  'neutral':    { badge: 'bg-gray-50 text-gray-700 border-gray-200',          ring: 'ring-gray-200/60',    accent: 'text-gray-500' },
  'small-bad':  { badge: 'bg-sky-50 text-sky-700 border-sky-200',             ring: 'ring-sky-200/60',     accent: 'text-sky-600' },
  'bad':        { badge: 'bg-slate-100 text-slate-700 border-slate-300',      ring: 'ring-slate-300/60',   accent: 'text-slate-600' },
};

/* ════════════════════ 卡片外壳 ════════════════════ */

function FortuneFrame({ today, children }: { today: string; children: ReactNode }) {
  return (
    <div className="rounded-2xl border border-gray-200 bg-white p-5 shadow-sm">
      <div className="flex items-center justify-between">
        <div className="text-xs font-medium uppercase tracking-wider text-gray-400">
          今日运势 · DAILY
        </div>
        <div className="font-mono text-xs text-gray-400">{today}</div>
      </div>
      {children}
    </div>
  );
}

/* ════════════════════ 主组件 ════════════════════ */

export function DailyFortune() {
  const { user, isLoggedIn } = useAuth();

  const today = todayString();

  const [status, setStatus] = useState<CheckinStatus | null>(null);
  const [loading, setLoading] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string>('');

  // 拉取签到状态：仅在登录态下发起请求
  useEffect(() => {
    if (!isLoggedIn || !user) {
      setStatus(null);
      setError('');
      return;
    }

    let cancelled = false;
    setLoading(true);
    setError('');

    getCheckinStatus()
      .then((data) => {
        if (!cancelled) setStatus(data);
      })
      .catch((err: unknown) => {
        if (cancelled) return;
        const message = err instanceof Error ? err.message : '加载签到状态失败';
        setError(message);
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });

    return () => {
      cancelled = true;
    };
  }, [isLoggedIn, user]);

  const fortune = useMemo<FortuneResult>(
    () => generateFortune(today, user?.userId ?? 'anonymous'),
    [today, user?.userId],
  );
  const tone = TONE_STYLES[fortune.level.tone];

  const handleCheckIn = async () => {
    if (!isLoggedIn || submitting) return;
    setSubmitting(true);
    setError('');
    try {
      const next = await submitCheckin();
      setStatus(next);
    } catch (err: unknown) {
      // 409 already_checked：优雅降级，仅刷新状态
      if (err instanceof CheckinApiError && err.status === 409) {
        try {
          const refreshed = await getCheckinStatus();
          setStatus(refreshed);
        } catch {
          setStatus({
            checked_today: true,
            last_checkin_date: today,
            continuous_days: status?.continuous_days ?? 1,
            max_continuous: status?.max_continuous ?? 1,
            total_days: status?.total_days ?? 1,
          });
        }
      } else {
        const message = err instanceof Error ? err.message : '签到失败，请稍后重试';
        setError(message);
      }
    } finally {
      setSubmitting(false);
    }
  };

  const handleReload = () => {
    setError('');
    setLoading(true);
    getCheckinStatus()
      .then((data) => setStatus(data))
      .catch((err: unknown) => {
        const message = err instanceof Error ? err.message : '加载签到状态失败';
        setError(message);
      })
      .finally(() => setLoading(false));
  };

  /* ─── 渲染分支 ─── */

  // 1. 未登录态
  if (!isLoggedIn) {
    return (
      <FortuneFrame today={today}>
        <div className="mt-5 rounded-xl border border-dashed border-gray-200 bg-gray-50/60 px-4 py-6 text-center">
          <div className="text-sm text-gray-500">签到后揭晓今日运势 · 宜 · 忌</div>
          <div className="mt-1 text-xs text-gray-400">登录后连签天数自动云端同步</div>
        </div>
        <div className="mt-4 flex items-center justify-end border-t border-dashed border-gray-200 pt-3">
          <button
            type="button"
            disabled
            className="cursor-not-allowed rounded-lg bg-gray-100 px-3 py-1.5 text-xs font-medium text-gray-500"
          >
            登录后解锁运势打卡
          </button>
        </div>
      </FortuneFrame>
    );
  }

  // 2. 加载态
  if (loading && !status) {
    return (
      <FortuneFrame today={today}>
        <div className="mt-4 h-32 animate-pulse rounded-xl bg-gray-100" />
      </FortuneFrame>
    );
  }

  // 3. 拉取失败
  if (error && !status) {
    return (
      <FortuneFrame today={today}>
        <div className="mt-5 rounded-xl border border-dashed border-rose-200 bg-rose-50/60 px-4 py-5 text-center">
          <div className="text-sm text-rose-700">{error}</div>
        </div>
        <div className="mt-4 flex items-center justify-end border-t border-dashed border-gray-200 pt-3">
          <button
            type="button"
            onClick={handleReload}
            className="rounded-lg border border-gray-200 bg-white px-3 py-1.5 text-xs text-gray-600 transition-colors hover:bg-gray-50"
          >
            重新加载
          </button>
        </div>
      </FortuneFrame>
    );
  }

  const checkedToday = status?.checked_today ?? false;
  const streak = status?.continuous_days ?? 0;

  // 4. 已签到态
  if (checkedToday) {
    return (
      <FortuneFrame today={today}>
        <div className="mt-4 flex flex-col items-center text-center">
          <div
            className={`inline-flex items-center gap-2 rounded-full border px-4 py-1.5 text-lg font-semibold ring-4 ${tone.badge} ${tone.ring}`}
          >
            <span className="opacity-60">§</span>
            <span>{fortune.level.label}</span>
            <span className="opacity-60">§</span>
          </div>
          <p className={`mt-2 text-sm ${tone.accent}`}>{fortune.quote}</p>
        </div>

        <div className="mt-4 space-y-2 rounded-xl bg-gray-50 p-3 text-sm">
          {fortune.goods.map((g, i) => (
            <div key={`good-${i}`} className="flex items-start gap-2">
              <span className="mt-0.5 shrink-0 rounded-md bg-emerald-100 px-1.5 py-0.5 text-xs font-medium text-emerald-700">
                宜
              </span>
              <span className="flex-1 text-gray-700">{g}</span>
            </div>
          ))}
          {fortune.bads.map((b, i) => (
            <div key={`bad-${i}`} className="flex items-start gap-2">
              <span className="mt-0.5 shrink-0 rounded-md bg-rose-100 px-1.5 py-0.5 text-xs font-medium text-rose-700">
                忌
              </span>
              <span className="flex-1 text-gray-700">{b}</span>
            </div>
          ))}
        </div>

        <div className="mt-4 flex items-center justify-between border-t border-dashed border-gray-200 pt-3">
          <div className="text-xs text-gray-500">
            已连续签到 <span className="font-mono font-semibold text-brand-600">{streak}</span> 天
          </div>
          <button
            type="button"
            disabled
            className="cursor-not-allowed rounded-lg bg-gray-100 px-3 py-1.5 text-xs font-medium text-gray-500"
          >
            今日已签到
          </button>
        </div>
      </FortuneFrame>
    );
  }

  // 5. 未签到态
  return (
    <FortuneFrame today={today}>
      <div className="mt-5 rounded-xl border border-dashed border-gray-200 bg-gray-50/60 px-4 py-6 text-center">
        <div className="text-sm text-gray-500">点签到，看看今天宜做什么、忌做什么。</div>
        <div className="mt-1 text-xs text-gray-400">连签记录会同步到云端。</div>
      </div>

      {error ? <p className="mt-3 text-xs text-rose-600">{error}</p> : null}

      <div className="mt-4 flex items-center justify-between border-t border-dashed border-gray-200 pt-3">
        <div className="text-xs text-gray-500">
          {streak > 0 ? (
            <>
              上一次连签 <span className="font-mono font-semibold text-brand-600">{streak}</span> 天
            </>
          ) : (
            <>第一次来？签到开始记录连签</>
          )}
        </div>
        <button
          type="button"
          onClick={handleCheckIn}
          disabled={submitting}
          className="rounded-lg bg-brand-600 px-4 py-1.5 text-sm font-medium text-white shadow-sm transition-colors hover:bg-brand-700 disabled:cursor-not-allowed disabled:bg-gray-400"
        >
          {submitting ? '签到中…' : '签到'}
        </button>
      </div>
    </FortuneFrame>
  );
}

export default DailyFortune;
