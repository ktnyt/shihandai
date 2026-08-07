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

// 辞書を使わずに組み合わせを作る長さ。
// 辞書にある2文字の語は数が限られていて同じ語ばかり出るので、この長さは
// 意味のない組み合わせも出して、かなの運指そのものを練習する。
const RANDOM_PAIR_LEN = 2;

function containsAny(units: string[], targets: string[]): boolean {
  return units.some((u) => targets.includes(u));
}

// 辞書から選んだ語のうち、組み合わせに置き換えても残すかなを返す。
function keptUnit(
  units: string[],
  newest: string | null,
  weak: string[],
): string | null {
  if (newest !== null && units.includes(newest)) return newest;
  return units.find((u) => weak.includes(u)) ?? null;
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
  // ただし2文字の語だけは辞書を引かず、かなをランダムに組み合わせる。
  word(
    allowed: string[],
    newest: string | null,
    weak: string[],
    maxLen: number,
  ): string[] {
    if (allowed.length === 0) {
      throw new Error("使えるかながない");
    }
    // 2文字までの段階は、辞書を引かずに毎回かなを組み合わせる
    if (maxLen === RANDOM_PAIR_LEN) {
      return this.randomPair(allowed, this.forcedUnit(allowed, newest, weak));
    }

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
    // 選ばれたのが2文字なら、辞書にない組み合わせにも広げる。
    // 選ばれた語が持っていた新出かなや苦手かなは組み合わせにも残す
    if (w.length === RANDOM_PAIR_LEN) {
      return this.randomPair(allowed, keptUnit(w, newest, weak));
    }
    this.remember(w);
    return w;
  }

  // allowed のかなを2つ並べる。辞書にある語かどうかは問わない。
  // forced が空でなければ、どちらか片方をそのかなにする。
  private randomPair(allowed: string[], forced: string | null): string[] {
    const at = (n: number) => Math.floor(this.rand() * n);

    // 作れる組み合わせの数。片方を固定すると一気に減るので、
    // 記憶をその半分までに縮めて選び直しが空回りするのを防ぐ
    const variants =
      forced !== null ? 2 * allowed.length : allowed.length * allowed.length;
    const limit = Math.min(RECENT_MEMORY, Math.floor(variants / 2));

    let w: string[] = [];
    for (let attempt = 0; attempt < 8; attempt++) {
      w = [allowed[at(allowed.length)], allowed[at(allowed.length)]];
      if (forced !== null) w[at(2)] = forced;
      if (!this.recentlyShown(w, limit)) break;
    }
    this.remember(w);
    return w;
  }

  // 組み合わせに必ず入れるかなを選ぶ。新出かなを newestRatio、苦手かなを
  // weakRatio の割合で選び、残りは null を返して2つともランダムに任せる。
  // 未解放のかなが混ざっていたら使わない。
  private forcedUnit(
    allowed: string[],
    newest: string | null,
    weak: string[],
  ): string | null {
    const r = this.rand();
    if (r < this.cfg.newestRatio && newest !== null && allowed.includes(newest))
      return newest;
    if (r < this.cfg.newestRatio + this.cfg.weakRatio) {
      const avail = weak.filter((u) => allowed.includes(u));
      if (avail.length > 0) return avail[Math.floor(this.rand() * avail.length)];
    }
    return null;
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
