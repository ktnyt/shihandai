// 画面から独立した練習セッションの進行役。Go版 tui.Model のロジック部分の移植。

import { Engine } from "./engine";
import { Drill, type WordResult } from "./drill";
import type { Generator } from "./lesson";
import { recentAccuracy } from "./drill";
import type { Key } from "./keys";

export type SessionState = "typing" | "waiting" | "paused" | "leveledUp";

// 先読みしておく単語の数。
const QUEUE_LEN = 4;

export interface SessionOptions {
  intervalMs: number; // 単語と単語の間の入力を受け付けない時間
  now: () => number;
  schedule: (fn: () => void, ms: number) => void;
  onChange: () => void;
  onSave: () => void;
}

export class Session {
  state: SessionState = "typing";
  upcoming: string[][] = [];
  kanaAdded = false; // 直近の昇格でかなが増えた
  message = "";
  flash = "";
  lastResult: WordResult | null = null;
  intervalMs: number; // 設定から変えられるよう公開する

  private queueLevel = 0;
  private waitToken = 0;

  constructor(
    public engine: Engine,
    public drill: Drill,
    private gen: Generator,
    private opts: SessionOptions,
  ) {
    this.intervalMs = opts.intervalMs;
    this.newWord(opts.now());
  }

  keydown(key: Key): void {
    if (this.state !== "typing") return;
    this.handleEmissions(this.engine.press(key));
    this.opts.onChange();
  }

  keyup(key: Key): void {
    if (this.state !== "typing") {
      this.engine.release(key);
      return;
    }
    this.handleEmissions(this.engine.release(key));
    this.opts.onChange();
  }

  // 一時停止。単語を隠して計測を止める。
  pause(): void {
    if (this.state === "paused" || this.state === "leveledUp") return;
    this.state = "paused";
    this.engine.clearHeld();
    this.flash = "";
    this.opts.onChange();
  }

  // 再開。同じ単語を最初から出題し直し、計測もやり直す。
  // インターバル中に止めた場合は次の単語へ進む。
  resume(): void {
    if (this.state !== "paused") return;
    this.engine.reset();
    this.flash = "";
    this.state = "typing";
    this.newWord(this.opts.now());
    this.opts.onChange();
  }

  // レベルアップ画面から練習に戻る。
  continueLevelUp(): void {
    if (this.state !== "leveledUp") return;
    this.state = "typing";
    this.message = "";
    this.newWord(this.opts.now());
    this.opts.onChange();
  }

  private handleEmissions(ems: { text: string }[]): void {
    const now = this.opts.now();
    for (const em of ems) {
      const result = this.drill.input(em.text);
      switch (result) {
        case "advance":
          this.flash = "";
          break;
        case "error":
          this.flash = `ミス: ${printable(em.text)}`;
          break;
        case "wordDone": {
          const out = this.drill.finishWord(now);
          this.lastResult = out;
          this.message = resultMessage(out);
          this.opts.onSave();
          if (out.promoted) {
            this.state = "leveledUp";
            this.kanaAdded = out.kanaAdded;
            this.engine.reset();
            return;
          }
          if (this.intervalMs > 0) {
            // 打ち終わりの巻き込みを防ぐため、少し置いてから次を出す
            this.state = "waiting";
            this.engine.reset();
            const token = ++this.waitToken;
            this.opts.schedule(() => {
              if (this.state === "waiting" && this.waitToken === token) {
                this.state = "typing";
                this.newWord(this.opts.now());
                this.opts.onChange();
              }
            }, this.intervalMs);
            return;
          }
          this.newWord(now);
          return;
        }
        case "ignored":
          break;
      }
    }
  }

  // 先読みキューの先頭を出題し、キューを補充する。
  private newWord(now: number): void {
    const allowed = this.drill.allowed();
    this.drill.setNewKanaSupply(
      this.gen.countWithUnit(allowed, this.drill.newest()),
    );
    if (this.queueLevel !== this.drill.level) {
      this.upcoming = [];
      this.queueLevel = this.drill.level;
    }
    this.fillQueue();
    const word = this.upcoming.shift()!;
    this.fillQueue();

    this.engine.reset();
    this.drill.startWord(word, now);
    this.flash = "";
  }

  private fillQueue(): void {
    while (this.upcoming.length < QUEUE_LEN) {
      this.upcoming.push(
        this.gen.word(
          this.drill.allowed(),
          this.focusUnits(),
          this.drill.stage().maxLen,
        ),
      );
    }
  }

  // 優先して出題したいかな (新出と苦手)。
  private focusUnits(): string[] {
    const allowed = this.drill.allowed();
    const focus = allowed.slice(-2);
    const weak = Object.entries(this.drill.stats)
      .filter(([u, s]) => allowed.includes(u) && s.recent.length > 0)
      .map(([u, s]) => [u, recentAccuracy(s)] as const)
      .filter(([, acc]) => acc < 1)
      .sort((a, b) => a[1] - b[1])
      .slice(0, 3)
      .map(([u]) => u);
    return [...focus, ...weak];
  }
}

function resultMessage(out: WordResult): string {
  if (out.demoted) {
    return `「${out.weakUnit}」の正答率が下がったのでレベルダウン`;
  }
  if (out.promoted) {
    return "速度とミス率が基準をみたした! レベルアップ";
  }
  if (out.success) {
    return `成功 ${(out.durationMs / 1000).toFixed(1)}s`;
  }
  return `失敗 (ミス ${out.errors})`;
}

function printable(text: string): string {
  switch (text) {
    case " ":
      return "␣";
    case "\b":
      return "BS";
    case "\n":
      return "⏎";
    default:
      return text;
  }
}
