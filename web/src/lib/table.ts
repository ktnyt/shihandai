// 薙刀式v15のかな変換テーブル。Go版 internal/naginata/table.go の移植。
// 元は eswai/qmk_firmware users/naginata_v15/naginata_v15.c の ngmap。

import { set, type KeySet } from "./keys";

export interface Entry {
  keys: KeySet;
  text: string;
}

export const TEXT_BACKSPACE = "\b";
export const TEXT_ENTER = "\n";

// キー添字 (keys.ts の KEYS 順)
const Q = 0, W = 1, E = 2, R = 3;
const U = 6, I = 7, O = 8, P = 9;
const A = 10, S = 11, D = 12, F = 13, G = 14, H = 15, J = 16, K = 17, L = 18, SEMI = 19;
const Z = 20, X = 21, C = 22, V = 23, B = 24, N = 25, M = 26, COMMA = 27, DOT = 28, SLASH = 29;
const SPACE = 30;

export const TABLE: Entry[] = [
  // 清音
  { keys: set(J), text: "あ" },
  { keys: set(K), text: "い" },
  { keys: set(L), text: "う" },
  { keys: set(SPACE, O), text: "え" },
  { keys: set(SPACE, N), text: "お" },
  { keys: set(F), text: "か" },
  { keys: set(W), text: "き" },
  { keys: set(H), text: "く" },
  { keys: set(S), text: "け" },
  { keys: set(V), text: "こ" },
  { keys: set(SPACE, U), text: "さ" },
  { keys: set(R), text: "し" },
  { keys: set(O), text: "す" },
  { keys: set(SPACE, A), text: "せ" },
  { keys: set(B), text: "そ" },
  { keys: set(N), text: "た" },
  { keys: set(SPACE, G), text: "ち" },
  { keys: set(SPACE, L), text: "つ" },
  { keys: set(E), text: "て" },
  { keys: set(D), text: "と" },
  { keys: set(M), text: "な" },
  { keys: set(SPACE, D), text: "に" },
  { keys: set(SPACE, W), text: "ぬ" },
  { keys: set(SPACE, R), text: "ね" },
  { keys: set(SPACE, J), text: "の" },
  { keys: set(C), text: "は" },
  { keys: set(X), text: "ひ" },
  { keys: set(SPACE, X), text: "ひ" },
  { keys: set(SPACE, SEMI), text: "ふ" },
  { keys: set(P), text: "へ" },
  { keys: set(Z), text: "ほ" },
  { keys: set(SPACE, Z), text: "ほ" },
  { keys: set(SPACE, F), text: "ま" },
  { keys: set(SPACE, B), text: "み" },
  { keys: set(SPACE, COMMA), text: "む" },
  { keys: set(SPACE, S), text: "め" },
  { keys: set(SPACE, K), text: "も" },
  { keys: set(SPACE, H), text: "や" },
  { keys: set(SPACE, P), text: "ゆ" },
  { keys: set(SPACE, I), text: "よ" },
  { keys: set(DOT), text: "ら" },
  { keys: set(SPACE, E), text: "り" },
  { keys: set(I), text: "る" },
  { keys: set(SLASH), text: "れ" },
  { keys: set(SPACE, SLASH), text: "れ" },
  { keys: set(A), text: "ろ" },
  { keys: set(SPACE, DOT), text: "わ" },
  { keys: set(SPACE, C), text: "を" },
  { keys: set(COMMA), text: "ん" },
  { keys: set(SEMI), text: "ー" },

  // 濁音
  { keys: set(J, F), text: "が" },
  { keys: set(J, W), text: "ぎ" },
  { keys: set(F, H), text: "ぐ" },
  { keys: set(J, S), text: "げ" },
  { keys: set(J, V), text: "ご" },
  { keys: set(F, U), text: "ざ" },
  { keys: set(J, R), text: "じ" },
  { keys: set(F, O), text: "ず" },
  { keys: set(J, A), text: "ぜ" },
  { keys: set(J, B), text: "ぞ" },
  { keys: set(F, N), text: "だ" },
  { keys: set(J, G), text: "ぢ" },
  { keys: set(F, L), text: "づ" },
  { keys: set(J, E), text: "で" },
  { keys: set(J, D), text: "ど" },
  { keys: set(J, C), text: "ば" },
  { keys: set(J, X), text: "び" },
  { keys: set(F, SEMI), text: "ぶ" },
  { keys: set(F, P), text: "べ" },
  { keys: set(J, Z), text: "ぼ" },
  { keys: set(Q), text: "ゔ" },

  // 半濁音
  { keys: set(M, C), text: "ぱ" },
  { keys: set(M, X), text: "ぴ" },
  { keys: set(V, SEMI), text: "ぷ" },
  { keys: set(V, P), text: "ぺ" },
  { keys: set(M, Z), text: "ぽ" },

  // 小書き
  { keys: set(Q, H), text: "ゃ" },
  { keys: set(Q, P), text: "ゅ" },
  { keys: set(Q, I), text: "ょ" },
  { keys: set(Q, J), text: "ぁ" },
  { keys: set(Q, K), text: "ぃ" },
  { keys: set(Q, L), text: "ぅ" },
  { keys: set(Q, O), text: "ぇ" },
  { keys: set(Q, N), text: "ぉ" },
  { keys: set(Q, DOT), text: "ゎ" },
  { keys: set(G), text: "っ" },
  { keys: set(Q, S), text: "ヶ" },
  { keys: set(Q, F), text: "ヵ" },

  // 清音拗音 濁音拗音 半濁拗音
  { keys: set(R, H), text: "しゃ" },
  { keys: set(R, P), text: "しゅ" },
  { keys: set(R, I), text: "しょ" },
  { keys: set(J, R, H), text: "じゃ" },
  { keys: set(J, R, P), text: "じゅ" },
  { keys: set(J, R, I), text: "じょ" },
  { keys: set(W, H), text: "きゃ" },
  { keys: set(W, P), text: "きゅ" },
  { keys: set(W, I), text: "きょ" },
  { keys: set(J, W, H), text: "ぎゃ" },
  { keys: set(J, W, P), text: "ぎゅ" },
  { keys: set(J, W, I), text: "ぎょ" },
  { keys: set(G, H), text: "ちゃ" },
  { keys: set(G, P), text: "ちゅ" },
  { keys: set(G, I), text: "ちょ" },
  { keys: set(J, G, H), text: "ぢゃ" },
  { keys: set(J, G, P), text: "ぢゅ" },
  { keys: set(J, G, I), text: "ぢょ" },
  { keys: set(D, H), text: "にゃ" },
  { keys: set(D, P), text: "にゅ" },
  { keys: set(D, I), text: "にょ" },
  { keys: set(X, H), text: "ひゃ" },
  { keys: set(X, P), text: "ひゅ" },
  { keys: set(X, I), text: "ひょ" },
  { keys: set(J, X, H), text: "びゃ" },
  { keys: set(J, X, P), text: "びゅ" },
  { keys: set(J, X, I), text: "びょ" },
  { keys: set(M, X, H), text: "ぴゃ" },
  { keys: set(M, X, P), text: "ぴゅ" },
  { keys: set(M, X, I), text: "ぴょ" },
  { keys: set(B, H), text: "みゃ" },
  { keys: set(B, P), text: "みゅ" },
  { keys: set(B, I), text: "みょ" },
  { keys: set(E, H), text: "りゃ" },
  { keys: set(E, P), text: "りゅ" },
  { keys: set(E, I), text: "りょ" },

  // 清音外来音 濁音外来音
  { keys: set(M, E, K), text: "てぃ" },
  { keys: set(M, E, P), text: "てゅ" },
  { keys: set(J, E, K), text: "でぃ" },
  { keys: set(J, E, P), text: "でゅ" },
  { keys: set(M, D, L), text: "とぅ" },
  { keys: set(J, D, L), text: "どぅ" },
  { keys: set(M, R, O), text: "しぇ" },
  { keys: set(M, G, O), text: "ちぇ" },
  { keys: set(J, R, O), text: "じぇ" },
  { keys: set(J, G, O), text: "ぢぇ" },
  { keys: set(V, SEMI, J), text: "ふぁ" },
  { keys: set(V, SEMI, K), text: "ふぃ" },
  { keys: set(V, SEMI, O), text: "ふぇ" },
  { keys: set(V, SEMI, N), text: "ふぉ" },
  { keys: set(V, SEMI, P), text: "ふゅ" },
  { keys: set(V, K, O), text: "いぇ" },
  { keys: set(V, L, K), text: "うぃ" },
  { keys: set(V, L, O), text: "うぇ" },
  { keys: set(V, L, N), text: "うぉ" },
  { keys: set(M, Q, J), text: "ゔぁ" },
  { keys: set(M, Q, K), text: "ゔぃ" },
  { keys: set(M, Q, O), text: "ゔぇ" },
  { keys: set(M, Q, N), text: "ゔぉ" },
  { keys: set(M, Q, P), text: "ゔゅ" },
  { keys: set(V, H, J), text: "くぁ" },
  { keys: set(V, H, K), text: "くぃ" },
  { keys: set(V, H, O), text: "くぇ" },
  { keys: set(V, H, N), text: "くぉ" },
  { keys: set(V, H, DOT), text: "くゎ" },
  { keys: set(F, H, J), text: "ぐぁ" },
  { keys: set(F, H, K), text: "ぐぃ" },
  { keys: set(F, H, O), text: "ぐぇ" },
  { keys: set(F, H, N), text: "ぐぉ" },
  { keys: set(F, H, DOT), text: "ぐゎ" },
  { keys: set(V, L, J), text: "つぁ" },

  // 記号と制御
  { keys: set(SPACE), text: " " },
  { keys: set(SPACE, V), text: "、" },
  { keys: set(SPACE, M), text: "。" },
  { keys: set(U), text: TEXT_BACKSPACE },
  { keys: set(V, M), text: TEXT_ENTER },
];

// text を入力する組み合わせ。別名は最初に定義されたもの (正規の打鍵)。
const chordIndex = new Map<string, KeySet>();
for (const e of TABLE) {
  if (!chordIndex.has(e.text)) chordIndex.set(e.text, e.keys);
}

export function chordFor(text: string): KeySet | undefined {
  return chordIndex.get(text);
}
