// 画面から独立した練習セッションの進行役。Go版 tui.Model のロジック部分の移植。

import { Engine } from "./engine";
import { Drill, type WordResult } from "./drill";
import type { Generator } from "./lesson";
import { recentAccuracy } from "./drill";
import type { Key } from "./keys";

export type SessionState =
  | "ready" // 開始待ち。Space で計測が始まる
  | "typing"
  | "waiting"
  | "paused"
  | "leveledUp";

export type SessionEvent = "error" | "wordDone" | "promoted" | "demoted";

export interface SessionOptions {
  intervalMs: number; // 単語と単語の間の入力を受け付けない時間
  upcomingWords: number; // 先に見える単語の数
  now: () => number;
  schedule: (fn: () => void, ms: number) => void;
  onChange: () => void;
  onSave: () => void;
  onEvent?: (event: SessionEvent) => void; // 効果音などの演出用
}

export class Session {
  state: SessionState = "ready";
  upcoming: string[][] = [];
  kanaAdded = false; // 直近の昇格でかなが増えた
  message = "";
  flash = "";
  lastResult: WordResult | null = null;
  intervalMs: number; // 設定から変えられるよう公開する
  upcomingWords: number; // 同上
  repeat = false; // くりかえし練習中 (成績は記録しない)

  private practiceWord: string[] | null = null;
  private lastWord: string[] | null = null; // 直前に打ち終えた単語
  private queueLevel = 0;
  private waitToken = 0;

  constructor(
    public engine: Engine,
    public drill: Drill,
    private gen: Generator,
    private opts: SessionOptions,
  ) {
    this.intervalMs = opts.intervalMs;
    this.upcomingWords = opts.upcomingWords;
    this.newWord(opts.now());
  }

  // 開始待ちから練習を始める。出題と計測はここから。
  start(): void {
    if (this.state !== "ready") return;
    this.state = "typing";
    this.engine.reset();
    this.drill.startWord(this.drill.currentWord(), this.opts.now());
    this.opts.onChange();
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
    if (this.state !== "typing" && this.state !== "waiting") return;
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
    if (this.repeat) {
      // くりかえし練習はミスを記録せず同じ単語を出し直す
      this.restartPractice(this.opts.now());
      this.opts.onChange();
      return;
    }
    // 打ちかけを破棄して次の単語へ。ミスした位置は記録に残す
    this.drill.abandonWord();
    this.newWord(this.opts.now());
    this.opts.onChange();
  }

  // くりかえし練習の切り替え。直前に打ち終えた単語 (なければいまの単語) を
  // 何度でも打ち直せる。この間の成績は一切記録しない。
  toggleRepeat(): void {
    if (this.state !== "typing" && this.state !== "waiting") return;
    const now = this.opts.now();
    this.waitToken++; // インターバル待ちがあれば無効にする
    if (!this.repeat) {
      const word = this.lastWord ?? this.drill.currentWord();
      if (word.length === 0) return;
      // 打ちかけの通常の単語はミスだけ記録して破棄する
      if (this.state === "typing") this.drill.abandonWord();
      this.practiceWord = [...word];
      this.repeat = true;
      this.state = "typing";
      this.message = "";
      this.restartPractice(now);
    } else {
      // くりかえし中のミスは記録しない (abandonWord を呼ばない)
      this.repeat = false;
      this.practiceWord = null;
      this.state = "typing";
      this.message = "";
      this.newWord(now);
    }
    this.opts.onChange();
  }

  private restartPractice(now: number): void {
    this.engine.reset();
    this.drill.startWord(this.practiceWord!, now);
    this.flash = "";
  }

  // レベルアップ画面から練習に戻る。
  continueLevelUp(): void {
    if (this.state !== "leveledUp") return;
    this.state = "typing";
    this.message = "";
    this.newWord(this.opts.now());
    this.opts.onChange();
  }

  private handleEmissions(ems: { text: string; keys: number }[]): void {
    const now = this.opts.now();
    for (const em of ems) {
      const result = this.drill.input(em.text, em.keys);
      switch (result) {
        case "advance":
          this.flash = "";
          break;
        case "error":
          this.flash = this.drill.needsBackspace()
            ? `ミス: ${printable(em.text)} (BSで消してから打ち直してください)`
            : `ミス: ${printable(em.text)}`;
          this.opts.onEvent?.("error");
          break;
        case "rollover":
          this.flash = `ミス: ${printable(em.text)} (同時押し・ノーカウント)`;
          break;
        case "corrected":
          this.flash = "";
          break;
        case "blocked":
          // 修正待ちのまま。フラッシュは出したままにする
          break;
        case "wordDone": {
          if (this.repeat) {
            // くりかえし練習は記録せず、同じ単語を出し直す
            const errors = this.drill.currentErrors();
            const secs = (this.drill.elapsedMs(now) / 1000).toFixed(1);
            this.message =
              errors === 0
                ? `成功 ${secs}s (くりかえし)`
                : `ミス ${errors} (くりかえし)`;
            if (this.intervalMs > 0) {
              this.state = "waiting";
              this.engine.reset();
              const token = ++this.waitToken;
              this.opts.schedule(() => {
                if (this.state === "waiting" && this.waitToken === token) {
                  this.state = "typing";
                  this.restartPractice(this.opts.now());
                  this.opts.onChange();
                }
              }, this.intervalMs);
              return;
            }
            this.restartPractice(now);
            return;
          }
          const out = this.drill.finishWord(now);
          this.lastWord = [...this.drill.currentWord()];
          this.lastResult = out;
          this.message = resultMessage(out);
          this.opts.onEvent?.(
            out.promoted ? "promoted" : out.demoted ? "demoted" : "wordDone",
          );
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
    this.upcoming = this.upcoming.slice(0, Math.max(this.upcomingWords, 0));
    while (this.upcoming.length < this.upcomingWords) {
      this.upcoming.push(
        this.gen.word(
          this.drill.allowed(),
          this.drill.newest(),
          this.weakUnits(),
          this.drill.stage().maxLen,
        ),
      );
    }
  }

  // 苦手なかな (直近正答率の低い順に3つ)。
  private weakUnits(): string[] {
    const allowed = this.drill.allowed();
    return Object.entries(this.drill.stats)
      .filter(([u, s]) => allowed.includes(u) && s.recent.length > 0)
      .map(([u, s]) => [u, recentAccuracy(s)] as const)
      .filter(([, acc]) => acc < 1)
      .sort((a, b) => a[1] - b[1])
      .slice(0, 3)
      .map(([u]) => u);
  }
}

function resultMessage(out: WordResult): string {
  if (out.demoted) {
    return `「${out.weakUnit}」の正答率が下がったため、レベルダウンしました`;
  }
  if (out.promoted) {
    return "昇格基準を満たしました!";
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
