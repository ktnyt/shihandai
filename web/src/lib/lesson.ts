// 練習する単語の選択。Go版 internal/lesson の移植。

export interface UnitSet {
  set: Set<string>;
  maxLen: number;
}

export function newUnitSet(allowed: string[]): UnitSet {
  const set = new Set(allowed);
  let maxLen = 0;
  for (const u of allowed) maxLen = Math.max(maxLen, [...u].length);
  return { set, maxLen };
}

// text を allowed の単位 (拗音は2文字で1単位) に最長一致で分割する。
export function segmentWith(s: UnitSet, text: string): string[] | null {
  const runes = [...text];
  const units: string[] = [];
  let i = 0;
  while (i < runes.length) {
    let matched = false;
    for (let n = Math.min(s.maxLen, runes.length - i); n > 0; n--) {
      const candidate = runes.slice(i, i + n).join("");
      if (s.set.has(candidate)) {
        units.push(candidate);
        i += n;
        matched = true;
        break;
      }
    }
    if (!matched) return null;
  }
  return units;
}

export function segment(text: string, allowed: string[]): string[] | null {
  return segmentWith(newUnitSet(allowed), text);
}

export interface GeneratorConfig {
  newestRatio: number; // 新出かなを含む語を出す割合
  weakRatio: number; // 苦手かなを含む語を出す割合
  skew: number; // 頻度への偏り。大きいほど高頻度語に寄る
}

// 新出かなは練習の主役なので、4割は新出かなを含む語にする。
export const DEFAULT_GENERATOR_CONFIG: GeneratorConfig = {
  newestRatio: 0.4,
  weakRatio: 0.2,
  skew: 2,
};

// 連発を抑えるために覚えておく、直近に出した語の数。
const RECENT_MEMORY = 20;

function containsAny(units: string[], targets: string[]): boolean {
  return units.some((u) => targets.includes(u));
}

// 指数分布 (平均1) の乱数。
function expRandom(rand: () => number): number {
  return -Math.log(1 - rand());
}

export class Generator {
  private recent: string[] = [];

  constructor(
    private words: string[], // 頻度順
    private cfg: GeneratorConfig = DEFAULT_GENERATOR_CONFIG,
    private rand: () => number = Math.random,
  ) {}

  // allowed のかなだけで打てる単語を1つ選ぶ。maxLen は最大文字数 (0で無制限)。
  // newest は新しく覚えているかな、weak は苦手なかな。新出かなを含む語を
  // newestRatio、苦手かなを含む語を weakRatio の割合で優先して出す。
  // 長さの範囲に新出かなを含む語がなければ、範囲を超えても出す。
  word(
    allowed: string[],
    newest: string | null,
    weak: string[],
    maxLen: number,
  ): string[] {
    const set = newUnitSet(allowed);

    const candidates: string[][] = [];
    const newestPool: string[][] = [];
    const newestLong: string[][] = [];
    const weakPool: string[][] = [];
    for (const w of this.words) {
      const units = segmentWith(set, w);
      if (units === null) continue;
      const hasNewest = newest !== null && units.includes(newest);
      if (maxLen > 0 && units.length > maxLen) {
        if (hasNewest) newestLong.push(units);
        continue;
      }
      candidates.push(units);
      if (hasNewest) newestPool.push(units);
      else if (containsAny(units, weak)) weakPool.push(units);
    }
    const newestAny = newestPool.length > 0 ? newestPool : newestLong;
    let pool = candidates.length > 0 ? candidates : newestAny;
    if (pool.length === 0) {
      throw new Error(`使えるかな [${allowed}] で打てる語が辞書にない`);
    }
    const r = this.rand();
    if (r < this.cfg.newestRatio && newestAny.length > 0) {
      pool = newestAny;
    } else if (
      r < this.cfg.newestRatio + this.cfg.weakRatio &&
      weakPool.length > 0
    ) {
      pool = weakPool;
    }

    // 直近に出した語は選び直す。語彙が少ないときは記憶を短くする
    const limit = Math.min(RECENT_MEMORY, Math.floor(pool.length / 2));
    let w = pool[this.pick(pool.length)];
    for (let attempt = 0; attempt < 8; attempt++) {
      if (!this.recentlyShown(w, limit)) break;
      w = pool[this.pick(pool.length)];
    }
    this.remember(w);
    return w;
  }

  private pick(n: number): number {
    const idx = Math.floor((expRandom(this.rand) / this.cfg.skew) * n);
    return Math.min(idx, n - 1);
  }

  private recentlyShown(units: string[], limit: number): boolean {
    if (limit <= 0) return false;
    const key = units.join("");
    return this.recent.slice(-limit).includes(key);
  }

  private remember(units: string[]): void {
    this.recent.push(units.join(""));
    if (this.recent.length > RECENT_MEMORY) {
      this.recent = this.recent.slice(-RECENT_MEMORY);
    }
  }
}
