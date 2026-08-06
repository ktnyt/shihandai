// 練習セッションの進行と昇格・降格の判定。Go版 internal/drill の移植。
//
// 出題は1単語ずつ。単語が表示された瞬間から打ち終わるまでを計測する。
// 昇格は直近の窓の打鍵速度 (kpm) とミス率の両方が基準を満たしたとき。

import { count } from "./keys";
import { chordFor } from "./table";
import { maxLevel, stageFor, unitsFor, type Stage } from "./curriculum";

// 1つのかなの直近正答率を測る試行数。
const RECENT_WINDOW = 12;

export interface UnitStat {
  attempts: number;
  errors: number;
  recent: boolean[];
}

export function recentAccuracy(s: UnitStat): number {
  if (s.recent.length === 0) return 1;
  return s.recent.filter(Boolean).length / s.recent.length;
}

export interface DrillConfig {
  targetKPM: number; // 昇格に必要な打鍵速度 (keys per minute)
  maxMissRate: number; // 昇格できるミス率の上限
  windowSize: number; // 判定に使う直近の単語数
  minNewKanaWords: number; // 昇格までに打つ、新出かなを含む語の最低数
  requireBackspace: boolean; // ミス時にバックスペース修正が必要か
  demoteAccuracy: number; // これを下回ると降格するかなの直近正答率
  minAttempts: number; // 降格判定に必要な直近試行数
}

export const DEFAULT_DRILL_CONFIG: DrillConfig = {
  targetKPM: 120,
  maxMissRate: 0.05,
  windowSize: 100,
  minNewKanaWords: 50,
  requireBackspace: false,
  demoteAccuracy: 0.7,
  minAttempts: RECENT_WINDOW,
};

export interface WordRecord {
  success: boolean;
  units: number; // 正しく打ったかなの数
  keys: number; // 打鍵数 (同時押しは複数と数える)
  errors: number; // ミス入力の数
  typingMs: number; // 表示から打ち終わるまでの時間
}

export type InputResult =
  | "ignored"
  | "advance"
  | "error"
  | "wordDone"
  | "blocked" // バックスペース修正待ちの間のかな入力
  | "corrected"; // バックスペースで修正した

export interface WordResult {
  success: boolean;
  durationMs: number;
  errors: number;
  promoted: boolean;
  kanaAdded: boolean; // 昇格でかなが増えた。false の昇格は長さの解放
  demoted: boolean;
  weakUnit: string; // 降格の原因になったかな
}

export class Drill {
  level: number;
  stats: Record<string, UnitStat>;

  private word: string[] = [];
  private pos = 0;
  private wordErrors = 0;
  private pendingError = false; // バックスペース修正待ち
  private shownAt: number | null = null;
  private records: WordRecord[] = [];
  private newKanaWordCount = 0;

  constructor(
    public cfg: DrillConfig,
    level: number,
    stats: Record<string, UnitStat> = {},
  ) {
    this.level = Math.min(Math.max(level, 1), maxLevel());
    this.stats = stats;
    // 手で編集された進捗由来の null に耐える
    for (const k of Object.keys(this.stats)) {
      if (!this.stats[k]) delete this.stats[k];
    }
  }

  allowed(): string[] {
    return unitsFor(this.level);
  }

  stage(): Stage {
    return stageFor(this.level);
  }

  newest(): string {
    const a = this.allowed();
    return a[a.length - 1];
  }

  newKanaWords(): number {
    return this.newKanaWordCount;
  }

  // 最初の5文字の段階では全部が新出なので、ゲートはかけない。
  // それ以外は設定値そのまま (自動で緩めたりはしない)。
  gateTarget(): number {
    if (this.allowed().length === unitsFor(1).length) return 0;
    return this.cfg.minNewKanaWords;
  }

  progress(): { records: WordRecord[]; newKanaWords: number } {
    return { records: this.records, newKanaWords: this.newKanaWordCount };
  }

  setProgress(records: WordRecord[], newKanaWords: number): void {
    this.records = records.slice(-this.cfg.windowSize);
    this.newKanaWordCount = Math.max(newKanaWords, 0);
  }

  startWord(units: string[], nowMs: number): void {
    this.word = units;
    this.pos = 0;
    this.wordErrors = 0;
    this.pendingError = false;
    this.shownAt = nowMs;
  }

  currentWord(): string[] {
    return this.word;
  }

  currentPos(): number {
    return this.pos;
  }

  currentErrors(): number {
    return this.wordErrors;
  }

  expected(): string {
    return this.pos < this.word.length ? this.word[this.pos] : "";
  }

  // バックスペース修正待ちかどうか。
  needsBackspace(): boolean {
    return this.pendingError;
  }

  elapsedMs(nowMs: number): number {
    return this.shownAt === null ? 0 : nowMs - this.shownAt;
  }

  private wordKeys(): number {
    let keys = 0;
    for (const u of this.word) {
      const chord = chordFor(u);
      keys += chord === undefined ? 1 : count(chord);
    }
    return keys;
  }

  input(text: string): InputResult {
    if (this.pos >= this.word.length) return "ignored";

    // 修正必須モードでは、ミスの後にバックスペースを打つまで進めない
    if (this.pendingError) {
      if (text === "\b") {
        this.pendingError = false;
        return "corrected";
      }
      if (text === " " || text === "\n" || text === "") return "ignored";
      return "blocked";
    }

    if (text === " " || text === "\b" || text === "\n" || text === "") {
      return "ignored";
    }
    const expected = this.word[this.pos];
    if (text === expected) {
      this.record(expected, true);
      this.pos++;
      return this.pos >= this.word.length ? "wordDone" : "advance";
    }
    this.record(expected, false);
    this.wordErrors++;
    if (this.cfg.requireBackspace) {
      this.pendingError = true;
    }
    return "error";
  }

  private stat(unit: string): UnitStat {
    let s = this.stats[unit];
    if (!s) {
      s = { attempts: 0, errors: 0, recent: [] };
      this.stats[unit] = s;
    }
    return s;
  }

  private record(unit: string, ok: boolean): void {
    const s = this.stat(unit);
    s.attempts++;
    if (!ok) s.errors++;
    s.recent.push(ok);
    if (s.recent.length > RECENT_WINDOW) {
      s.recent = s.recent.slice(-RECENT_WINDOW);
    }
  }

  successCount(): { successes: number; total: number } {
    return {
      successes: this.records.filter((r) => r.success).length,
      total: this.records.length,
    };
  }

  // 直近の窓の打鍵速度 (keys per minute)。
  // 表示から打ち終わるまでの素の時間で計算する。
  windowKPM(): number {
    let keys = 0;
    let typingMs = 0;
    for (const r of this.records) {
      keys += r.keys;
      typingMs += r.typingMs;
    }
    if (typingMs <= 0) return 0;
    return keys / (typingMs / 60000);
  }

  // 直近の窓のミス率。打ったかな (正解とミスの合計) のうちミスの割合。
  missRate(): number {
    let units = 0;
    let errors = 0;
    for (const r of this.records) {
      units += r.units;
      errors += r.errors;
    }
    if (units + errors === 0) return 0;
    return errors / (units + errors);
  }

  finishWord(nowMs: number): WordResult {
    const out: WordResult = {
      success: this.wordErrors === 0,
      durationMs: this.elapsedMs(nowMs),
      errors: this.wordErrors,
      promoted: false,
      kanaAdded: false,
      demoted: false,
      weakUnit: "",
    };

    // 一瞬で打ち終えた語で速度が発散しないよう下限を置く。
    // ms は整数に丸める (保存と共有URLを短くするため)
    const typingMs = Math.max(Math.round(out.durationMs), 10);
    this.records.push({
      success: out.success,
      units: this.word.length,
      keys: this.wordKeys(),
      errors: out.errors,
      typingMs,
    });
    this.records = this.records.slice(-this.cfg.windowSize);
    if (this.word.includes(this.newest())) {
      this.newKanaWordCount++;
    }

    // 正答率が下がったかなが出たら1つ降格する。
    // 覚えている最中のいちばん新しいかなは対象外。
    if (this.level > 1) {
      const allowed = this.allowed();
      for (const unit of allowed.slice(0, -1)) {
        const s = this.stats[unit];
        if (!s || s.recent.length < this.cfg.minAttempts) continue;
        if (recentAccuracy(s) < this.cfg.demoteAccuracy) {
          this.level--;
          out.demoted = true;
          out.weakUnit = unit;
          this.resetProgress();
          return out;
        }
      }
    }

    // 窓が埋まっていて、打鍵速度とミス率の両方が基準を満たし、
    // 新出かなを含む語も十分に打っていたらレベルアップ
    if (
      this.records.length >= this.cfg.windowSize &&
      this.windowKPM() >= this.cfg.targetKPM &&
      this.missRate() <= this.cfg.maxMissRate &&
      this.newKanaWordCount >= this.gateTarget() &&
      this.level < maxLevel()
    ) {
      const before = this.allowed().length;
      this.level++;
      out.promoted = true;
      out.kanaAdded = this.allowed().length > before;
      this.records = [];
      this.newKanaWordCount = 0;
    }
    return out;
  }

  private resetProgress(): void {
    this.records = [];
    this.newKanaWordCount = 0;
    for (const s of Object.values(this.stats)) {
      s.recent = [];
    }
  }
}
