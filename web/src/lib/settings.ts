// 練習の調整項目。localStorage に保存し、URLクエリで一時的に上書きできる。

import type { SoundType } from "./sound";

// 練習モードで切り替わる条件の部分。
export interface PresetSettings {
  targetKPM: number; // 昇格に必要な打鍵速度
  maxMissRate: number; // 昇格できるミス率の上限 (割合)
  upcomingWords: number; // 先に見える単語の数
  windowSize: number; // 判定に使う直近の単語数
  minNewKanaWords: number; // 昇格までに打つ、新出かなを含む語の最低数
  intervalMs: number; // 単語と単語の間の入力を受け付けない時間
  requireBackspace: boolean; // ミス時にバックスペース修正が必要か
}

export interface Settings extends PresetSettings {
  soundEnabled: boolean; // タイプ音を鳴らすか
  soundType: SoundType; // タイプ音の種類
}

export interface Preset {
  name: string;
  settings: PresetSettings;
}

// 練習モード。値をいじるとカスタム扱いになる。
export const PRESETS: Preset[] = [
  {
    name: "初心者",
    settings: {
      targetKPM: 60,
      maxMissRate: 0.01,
      upcomingWords: 5,
      windowSize: 50,
      minNewKanaWords: 30,
      intervalMs: 500,
      requireBackspace: false,
    },
  },
  {
    name: "中級者",
    settings: {
      targetKPM: 120,
      maxMissRate: 0.02,
      upcomingWords: 5,
      windowSize: 100,
      minNewKanaWords: 60,
      intervalMs: 500,
      requireBackspace: true,
    },
  },
  {
    name: "上級者",
    settings: {
      targetKPM: 180,
      maxMissRate: 0.03,
      upcomingWords: 5,
      windowSize: 150,
      minNewKanaWords: 60,
      intervalMs: 500,
      requireBackspace: true,
    },
  },
  {
    name: "師範代",
    settings: {
      targetKPM: 300,
      maxMissRate: 0.03,
      upcomingWords: 5,
      windowSize: 200,
      minNewKanaWords: 120,
      intervalMs: 500,
      requireBackspace: true,
    },
  },
];

export const DEFAULT_SETTINGS: Settings = {
  ...PRESETS[0].settings,
  soundEnabled: true,
  soundType: "mech",
};

// 練習条件が一致するプリセット名を返す。なければ null (カスタム)。
// タイプ音はモードと無関係なので比較しない。
export function matchPreset(s: Settings): string | null {
  for (const p of PRESETS) {
    if (
      p.settings.targetKPM === s.targetKPM &&
      p.settings.maxMissRate === s.maxMissRate &&
      p.settings.upcomingWords === s.upcomingWords &&
      p.settings.windowSize === s.windowSize &&
      p.settings.minNewKanaWords === s.minNewKanaWords &&
      p.settings.intervalMs === s.intervalMs &&
      p.settings.requireBackspace === s.requireBackspace
    ) {
      return p.name;
    }
  }
  return null;
}

const STORAGE_KEY = "shihandai/settings/v1";

function clamp(v: number, min: number, max: number, fallback: number): number {
  if (!Number.isFinite(v)) return fallback;
  return Math.min(Math.max(v, min), max);
}

// 範囲外や欠けた値を既定に丸める。
export function sanitize(partial: Partial<Settings>): Settings {
  const d = DEFAULT_SETTINGS;
  return {
    targetKPM: clamp(partial.targetKPM ?? d.targetKPM, 30, 600, d.targetKPM),
    maxMissRate: clamp(partial.maxMissRate ?? d.maxMissRate, 0.001, 0.5, d.maxMissRate),
    upcomingWords: Math.round(clamp(partial.upcomingWords ?? d.upcomingWords, 0, 10, d.upcomingWords)),
    windowSize: Math.round(clamp(partial.windowSize ?? d.windowSize, 10, 500, d.windowSize)),
    minNewKanaWords: Math.round(clamp(partial.minNewKanaWords ?? d.minNewKanaWords, 0, 300, d.minNewKanaWords)),
    intervalMs: clamp(partial.intervalMs ?? d.intervalMs, 0, 3000, d.intervalMs),
    requireBackspace:
      typeof partial.requireBackspace === "boolean"
        ? partial.requireBackspace
        : d.requireBackspace,
    soundEnabled:
      typeof partial.soundEnabled === "boolean"
        ? partial.soundEnabled
        : d.soundEnabled,
    soundType:
      partial.soundType === "mech" ||
      partial.soundType === "typewriter" ||
      partial.soundType === "pop"
        ? partial.soundType
        : d.soundType,
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
  const bool = (name: string) => {
    if (!params.has(name)) return undefined;
    return params.get(name) === "1" || params.get(name) === "true";
  };
  return sanitize({
    ...stored,
    targetKPM: num("kpm") ?? stored.targetKPM,
    maxMissRate: num("missrate") ?? stored.maxMissRate,
    upcomingWords: num("upcoming") ?? stored.upcomingWords,
    windowSize: num("words") ?? stored.windowSize,
    minNewKanaWords: num("newwords") ?? stored.minNewKanaWords,
    intervalMs: num("interval") ?? stored.intervalMs,
    requireBackspace: bool("bs") ?? stored.requireBackspace,
  });
}

export function saveSettings(s: Settings): void {
  try {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(s));
  } catch {
    // 保存できない環境では黙って進む
  }
}
