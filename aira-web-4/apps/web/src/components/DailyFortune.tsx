/**
 * components/DailyFortune.tsx
 * 每日运势签到卡片（纯前端版）
 *
 * 设计说明：
 *  - 运势随机种子 = hash(日期 + 用户唯一标识)，保证：
 *      * 同一用户同一天刷新页面看到的运势保持一致
 *      * 不同用户同一天看到的运势不同
 *      * 第二天自然换一份
 *  - 签到状态、连续天数使用 localStorage 持久化，key 按 userId 隔离
 *  - 未登录用户用 'anonymous' 作为 userKey，体验上同样可用
 *  - 所有"未来对接后端"的位置已用 TODO(后端对接) 注释标出
 */

'use client';

import { useEffect, useMemo, useState } from 'react';
import { useAuth } from '@/lib/auth';

/* ════════════════════ 语料库 ════════════════════ */

type FortuneTone =
  | 'great'      // 大吉
  | 'good'       // 吉
  | 'small-good' // 小吉
  | 'neutral'    // 中平
  | 'small-bad'  // 小凶
  | 'bad';       // 凶

interface FortuneLevel {
  label: string;
  weight: number;
  tone: FortuneTone;
}

/** 运势等级 + 权重（大吉/凶最稀有，中平最常见） */
const LEVELS: FortuneLevel[] = [
  { label: '大吉', weight: 1, tone: 'great' },
  { label: '吉',   weight: 3, tone: 'good' },
  { label: '小吉', weight: 5, tone: 'small-good' },
  { label: '中平', weight: 6, tone: 'neutral' },
  { label: '小凶', weight: 3, tone: 'small-bad' },
  { label: '凶',   weight: 1, tone: 'bad' },
];

/** 宜：贴近大学生日常的正面行为 */
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

/** 忌：反向的大学生 meme */
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

/** 不同运势对应的小段子，弹幕风格 */
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

/** FNV-1a 32 位字符串哈希，结果稳定，跨浏览器一致 */
function hashString(s: string): number {
  let h = 2166136261 >>> 0;
  for (let i = 0; i < s.length; i++) {
    h ^= s.charCodeAt(i);
    h = Math.imul(h, 16777619);
  }
  return h >>> 0;
}

/** Mulberry32：给一个种子生成一个确定性的 [0, 1) 序列 */
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

/** 按 weight 加权抽取一项 */
function pickWeighted<T extends { weight: number }>(items: T[], r: number): T {
  const total = items.reduce((s, it) => s + it.weight, 0);
  let acc = r * total;
  for (const it of items) {
    acc -= it.weight;
    if (acc <= 0) return it;
  }
  return items[items.length - 1];
}

/** 从数组中随机不重复抽取 n 项 */
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

function formatDate(d: Date): string {
  const yyyy = d.getFullYear();
  const mm = String(d.getMonth() + 1).padStart(2, '0');
  const dd = String(d.getDate()).padStart(2, '0');
  return `${yyyy}-${mm}-${dd}`;
}

function todayString(): string {
  return formatDate(new Date());
}

function yesterdayString(): string {
  const d = new Date();
  d.setDate(d.getDate() - 1);
  return formatDate(d);
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

/* ════════════════════ 组件 ════════════════════ */

export function DailyFortune() {
  const { user } = useAuth();

  // 未登录时也允许使用，但走匿名 key，避免 streak 在登录后被覆盖
  const userKey = user?.userId ?? 'anonymous';
  const today = todayString();

  // localStorage key 按用户隔离，防止多账户登录互相覆盖
  const storageKeyDate = `aira:fortune:lastDate:${userKey}`;
  const storageKeyStreak = `aira:fortune:streak:${userKey}`;

  const [hasCheckedIn, setHasCheckedIn] = useState(false);
  const [streak, setStreak] = useState(0);
  const [hydrated, setHydrated] = useState(false);

  // 首次水合时读取 localStorage（SSR 阶段不可用，避免 hydration mismatch）
  useEffect(() => {
    // TODO(后端对接): 调用 GET /api/checkin/today，返回
    //   { checked: boolean, last_date: string, streak: number, fortune?: FortuneResult }
    //   - 若后端 fortune 不为空，应直接使用后端结果（保证多设备一致）
    //   - 若 checked === false，再走前端的伪随机预览
    const lastDate = typeof window !== 'undefined' ? window.localStorage.getItem(storageKeyDate) : null;
    const savedStreak = typeof window !== 'undefined' ? Number(window.localStorage.getItem(storageKeyStreak) ?? '0') : 0;
    setHasCheckedIn(lastDate === today);
    setStreak(Number.isFinite(savedStreak) ? savedStreak : 0);
    setHydrated(true);
  }, [storageKeyDate, storageKeyStreak, today]);

  // 当日运势（用 useMemo 缓存；签到前作为"预览"展示也无妨，但默认遮起来更有仪式感）
  const fortune = useMemo<FortuneResult>(
    () => generateFortune(today, userKey),
    [today, userKey],
  );
  const tone = TONE_STYLES[fortune.level.tone];

  const handleCheckIn = () => {
    // TODO(后端对接): 调用 POST /api/checkin
    //   - 请求体 { date?: string }，后端以服务器时区为准
    //   - 响应 { streak: number, last_date: string, fortune: FortuneResult }
    //   - 接入后端后，下面的 localStorage 写入可保留为离线降级，也可移除
    if (typeof window === 'undefined') return;

    const last = window.localStorage.getItem(storageKeyDate);
    let nextStreak: number;
    if (last === today) {
      nextStreak = streak;            // 同日重复点击，按理走不到这里（按钮已禁用）
    } else if (last === yesterdayString()) {
      nextStreak = streak + 1;        // 昨天也签了：连签 +1
    } else {
      nextStreak = 1;                 // 断签 / 首次签到
    }

    window.localStorage.setItem(storageKeyDate, today);
    window.localStorage.setItem(storageKeyStreak, String(nextStreak));
    setStreak(nextStreak);
    setHasCheckedIn(true);
  };

  // 水合占位，防止 server / client 不一致
  if (!hydrated) {
    return (
      <div className="rounded-2xl border border-gray-200 bg-white p-5 shadow-sm">
        <div className="h-40 animate-pulse rounded-xl bg-gray-100" />
      </div>
    );
  }

  return (
    <div className="rounded-2xl border border-gray-200 bg-white p-5 shadow-sm">
      <div className="flex items-center justify-between">
        <div className="text-xs font-medium uppercase tracking-wider text-gray-400">
          今日运势 · DAILY
        </div>
        <div className="font-mono text-xs text-gray-400">{today}</div>
      </div>

      {hasCheckedIn ? (
        <>
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
        </>
      ) : (
        <>
          <div className="mt-5 rounded-xl border border-dashed border-gray-200 bg-gray-50/60 px-4 py-6 text-center">
            <div className="text-sm text-gray-500">点签到，看看今天宜做什么、忌做什么。</div>
            <div className="mt-1 text-xs text-gray-400">同一天内刷新页面运势保持一致。</div>
          </div>

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
              className="rounded-lg bg-brand-600 px-4 py-1.5 text-sm font-medium text-white shadow-sm transition-colors hover:bg-brand-700"
            >
              签到
            </button>
          </div>
        </>
      )}
    </div>
  );
}

export default DailyFortune;
