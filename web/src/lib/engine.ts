// 薙刀式の同時押し判定エンジン。
//
// Go版 (ターミナル) はキーリリースを検出できずタイミングウィンドウで
// 近似していたが、ブラウザでは keydown/keyup が取れるので、QMK実装
// (eswai/qmk_firmware naginata_v15) の「リリースで確定」を忠実に移植する。
// 遅延がなく、連続シフト・連続濁音もそのまま動く。

import { count, has, type Key, type KeySet } from "./keys";
import { TABLE } from "./table";

export interface Emission {
  text: string;
}

// 連続シフトのとき押しっぱなしを引き継ぐキー (QMK の shifted lookup 相当)。
const F = 13, J = 16, V = 23, M = 26;
const SPACE = 30;
const STICKY: Key[] = [SPACE, F, J, V, M];

export class Engine {
  private buf: Key[] = []; // 押された順の未確定キー
  private held: KeySet = 0; // いま物理的に押されているキー
  presses = 0;

  press(key: Key): Emission[] {
    this.presses++;
    this.held |= 1 << key;

    // スペースも普通のキーとして組み合わせに参加させる (後入れシフト許容)。
    // Go版と同じ判断で、かなを打った直後のシフトも同時押しとして拾う。
    this.buf.push(key);
    const { n, complete } = this.candidates();
    if (n === 0 || (n === 1 && complete)) {
      return this.typeOnce();
    }
    return [];
  }

  release(key: Key): Emission[] {
    this.held &= ~(1 << key);
    if (this.buf.length > 0) {
      return this.typeOnce();
    }
    return [];
  }

  // 未確定のキーを捨てる。単語の切り替え時に呼ぶ。
  reset(): void {
    this.buf = [];
  }

  // 押しっぱなしの記録も消す。フォーカスが外れて keyup を
  // 取り逃がしたときに呼ぶ。
  clearHeld(): void {
    this.held = 0;
    this.buf = [];
  }

  private combOf(n: number): KeySet {
    let s = 0;
    for (let i = 0; i < n; i++) s |= 1 << this.buf[i];
    return s;
  }

  // バッファ全体を含む候補の数。complete は候補が一つでかつ
  // 全キーが押されていることを示す。
  private candidates(): { n: number; complete: boolean } {
    const comb = this.combOf(this.buf.length);
    let n = 0;
    let hit = 0;
    for (const e of TABLE) {
      if ((e.keys & comb) === comb) {
        n++;
        hit = e.keys;
      }
    }
    return { n, complete: n === 1 && this.buf.length >= count(hit) };
  }

  // バッファ先頭からの最長一致で1単位を確定する。
  // 連続シフト: 押しっぱなしのシフト・濁音キーを組み合わせに足して先に探す。
  // どの長さでも一致しなければ先頭の1キーを捨てる。
  private typeOnce(): Emission[] {
    for (let nt = this.buf.length; nt > 0; nt--) {
      const comb = this.combOf(nt);

      let sticky = comb;
      for (const k of STICKY) {
        if (has(this.held, k)) sticky |= 1 << k;
      }
      if (sticky !== comb) {
        const e = TABLE.find((e) => e.keys === sticky);
        if (e) {
          this.buf = this.buf.slice(nt);
          return [{ text: e.text }];
        }
      }
      const e = TABLE.find((e) => e.keys === comb);
      if (e) {
        this.buf = this.buf.slice(nt);
        return [{ text: e.text }];
      }
    }
    this.buf.shift();
    return [];
  }
}
