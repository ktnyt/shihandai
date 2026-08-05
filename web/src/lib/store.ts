// 進捗の保存。localStorage に JSON で置く。

import type { UnitStat, WordRecord } from "./drill";

const STORAGE_KEY = "shihandai/state/v1";

export interface State {
  level: number;
  stats: Record<string, UnitStat>;
  records: WordRecord[];
  newKanaWords: number;
}

export function load(): State {
  const initial: State = { level: 1, stats: {}, records: [], newKanaWords: 0 };
  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    if (!raw) return initial;
    const parsed = JSON.parse(raw) as Partial<State>;
    return {
      level: typeof parsed.level === "number" && parsed.level >= 1 ? parsed.level : 1,
      stats: parsed.stats ?? {},
      records: parsed.records ?? [],
      newKanaWords: parsed.newKanaWords ?? 0,
    };
  } catch {
    return initial;
  }
}

export function save(state: State): void {
  try {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(state));
  } catch {
    // 保存できない環境 (プライベートモード等) では黙って進む
  }
}

export function reset(): void {
  try {
    localStorage.removeItem(STORAGE_KEY);
  } catch {
    // 消せなくても致命的ではない
  }
}
