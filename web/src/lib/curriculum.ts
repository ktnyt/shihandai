// 練習するかなの拡張順序。Go版 internal/curriculum の移植。

export interface Group {
  name: string;
  units: string[];
}

export const GROUPS: Group[] = [
  { name: "はじめの5文字", units: ["あ", "い", "な", "す", "る"] },
  {
    name: "ベース清音",
    units: [
      "ん", "し", "て", "か", "と", "た", "う", "く", "こ", "は",
      "き", "ら", "れ", "そ", "っ", "ろ", "け", "ほ", "ひ", "へ", "ー",
    ],
  },
  {
    name: "ベース濁音",
    units: [
      "で", "が", "だ", "じ", "ど", "ず", "ば", "ぐ", "ご", "げ",
      "ぞ", "び", "ぼ", "べ", "ぎ",
    ],
  },
  {
    name: "基本拗音",
    units: [
      "しょ", "しゅ", "しゃ", "きょ", "きゅ", "きゃ", "ちょ", "ちゃ", "ちゅ",
      "りょ", "りゅ", "りゃ", "ひょ", "ひゃ", "ひゅ", "にょ", "にゅ", "にゃ",
      "みょ", "みゃ", "みゅ",
    ],
  },
  {
    name: "シフト清音",
    units: [
      "の", "に", "を", "も", "ま", "り", "お", "え", "さ", "つ", "せ",
      "よ", "わ", "ち", "や", "め", "み", "ゆ", "ね", "ふ", "む", "ぬ",
    ],
  },
  { name: "シフト由来の濁音", units: ["ぶ", "ざ", "ぜ", "づ", "ぢ"] },
  { name: "半濁音", units: ["ぱ", "ぽ", "ぴ", "ぷ", "ぺ"] },
  {
    name: "濁音・半濁音の拗音",
    units: [
      "じょ", "じゅ", "じゃ", "ぎょ", "ぎゃ", "ぎゅ", "びょ", "びゃ", "びゅ",
      "ぴょ", "ぴゃ", "ぴゅ", "ぢゃ", "ぢゅ", "ぢょ",
    ],
  },
  {
    name: "外来音",
    units: [
      "ゔ", "ふぁ", "ふぃ", "ふぇ", "ふぉ", "てぃ", "でぃ", "うぃ", "うぇ", "うぉ",
      "しぇ", "ちぇ", "じぇ", "とぅ", "どぅ", "てゅ", "でゅ", "ふゅ", "つぁ",
      "ゔぁ", "ゔぃ", "ゔぇ", "ゔぉ",
    ],
  },
];

const flattened: string[] = GROUPS.flatMap((g) => g.units);

const INITIAL_COUNT = 5;

// かな1文字ごとに踏む単語の長さの段階。
const LENGTH_STEPS = [2, 3, 4, 5];

export interface Stage {
  units: string[];
  maxLen: number; // 出題する単語の最大文字数。0 なら無制限
}

export function maxLevel(): number {
  return (flattened.length - INITIAL_COUNT + 1) * LENGTH_STEPS.length;
}

export function stageFor(level: number): Stage {
  level = Math.min(Math.max(level, 1), maxLevel());
  const kanaIdx = Math.floor((level - 1) / LENGTH_STEPS.length);
  const step = (level - 1) % LENGTH_STEPS.length;
  const units = flattened.slice(0, INITIAL_COUNT + kanaIdx);
  let maxLen = LENGTH_STEPS[step];
  if (level === maxLevel()) maxLen = 0; // 最後まで来たら長さは全開放
  return { units, maxLen };
}

export function unitsFor(level: number): string[] {
  return stageFor(level).units;
}

export function groupOf(unit: string): string {
  for (const g of GROUPS) {
    if (g.units.includes(unit)) return g.name;
  }
  return "";
}
