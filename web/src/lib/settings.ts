// 練習の調整項目。localStorage に保存し、URLクエリで一時的に上書きできる。

export interface Settings {
  targetKPM: number; // 昇格に必要な打鍵速度
  maxMissRate: number; // 昇格できるミス率の上限 (割合)
  reactionBudgetMs: number; // 表示から打ち始めるまでの猶予
  windowSize: number; // 判定に使う直近の単語数
  minNewKanaWords: number; // 昇格までに打つ、新出かなを含む語の最低数
  intervalMs: number; // 単語と単語の間の入力を受け付けない時間
}

export const DEFAULT_SETTINGS: Settings = {
  targetKPM: 120,
  maxMissRate: 0.05,
  reactionBudgetMs: 500,
  windowSize: 100,
  minNewKanaWords: 50,
  intervalMs: 500,
};

const STORAGE_KEY = "shihandai/settings/v1";

function clamp(v: number, min: number, max: number, fallback: number): number {
  if (!Number.isFinite(v)) return fallback;
  return Math.min(Math.max(v, min), max);
}

// 範囲外や欠けた値を既定に丸める。
export function sanitize(partial: Partial<Settings>): Settings {
  const d = DEFAULT_SETTINGS;
  return {
    targetKPM: clamp(partial.targetKPM ?? d.targetKPM, 30, 400, d.targetKPM),
    maxMissRate: clamp(partial.maxMissRate ?? d.maxMissRate, 0.001, 0.5, d.maxMissRate),
    reactionBudgetMs: clamp(partial.reactionBudgetMs ?? d.reactionBudgetMs, 0, 3000, d.reactionBudgetMs),
    windowSize: Math.round(clamp(partial.windowSize ?? d.windowSize, 10, 500, d.windowSize)),
    minNewKanaWords: Math.round(clamp(partial.minNewKanaWords ?? d.minNewKanaWords, 0, 300, d.minNewKanaWords)),
    intervalMs: clamp(partial.intervalMs ?? d.intervalMs, 0, 3000, d.intervalMs),
  };
}

export function loadSettings(): Settings {
  let stored: Partial<Settings> = {};
  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    if (raw) stored = JSON.parse(raw) as Partial<Settings>;
  } catch {
    // 壊れた保存値は無視する
  }

  // URLクエリは保存値より優先する (保存はしない)
  const params = new URLSearchParams(location.search);
  const num = (name: string) => {
    const v = Number(params.get(name));
    return params.has(name) && Number.isFinite(v) ? v : undefined;
  };
  return sanitize({
    ...stored,
    targetKPM: num("kpm") ?? stored.targetKPM,
    maxMissRate: num("missrate") ?? stored.maxMissRate,
    reactionBudgetMs: num("react") ?? stored.reactionBudgetMs,
    windowSize: num("words") ?? stored.windowSize,
    minNewKanaWords: num("newwords") ?? stored.minNewKanaWords,
    intervalMs: num("interval") ?? stored.intervalMs,
  });
}

export function saveSettings(s: Settings): void {
  try {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(s));
  } catch {
    // 保存できない環境では黙って進む
  }
}
