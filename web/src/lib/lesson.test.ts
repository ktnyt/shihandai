import { describe, expect, it } from "vitest";
import { Generator, segment } from "./lesson";
import { unitsFor, maxLevel } from "./curriculum";
import { words } from "./dict";

// テスト用の決定的な乱数。
function seededRandom(seed: number): () => number {
  let s = seed;
  return () => {
    s = (s * 1103515245 + 12345) & 0x7fffffff;
    return s / 0x80000000;
  };
}

describe("segment", () => {
  const allowed = ["あ", "い", "な", "す", "る", "き", "きゃ", "ゆ"];

  it("単純な分割と最長一致", () => {
    expect(segment("あいな", allowed)).toEqual(["あ", "い", "な"]);
    expect(segment("きゃい", allowed)).toEqual(["きゃ", "い"]);
    expect(segment("きあ", allowed)).toEqual(["き", "あ"]);
  });

  it("使えない文字は分割できない", () => {
    expect(segment("あかい", allowed)).toBeNull();
  });
});

describe("dict", () => {
  it("辞書が読み込める", () => {
    expect(words().length).toBeGreaterThan(10000);
  });

  it("レベル1で打てる語が十分ある", () => {
    const allowed = unitsFor(1);
    const count = words().filter((w) => segment(w, allowed) !== null).length;
    expect(count).toBeGreaterThanOrEqual(10);
  });
});

describe("Generator", () => {
  it("どのレベルでも単語が選べる", () => {
    const g = new Generator(words(), undefined, seededRandom(1));
    // 全レベルは重いので代表的なレベルを確かめる
    for (const level of [1, 5, 50, 200, maxLevel()]) {
      const allowed = unitsFor(level);
      const word = g.word(allowed, allowed.slice(-1), 0);
      expect(word.length).toBeGreaterThan(0);
      for (const u of word) expect(allowed).toContain(u);
    }
  });

  it("最大文字数を守る", () => {
    const g = new Generator(words(), undefined, seededRandom(3));
    const allowed = unitsFor(30);
    for (let i = 0; i < 50; i++) {
      expect(g.word(allowed, [], 2).length).toBeLessThanOrEqual(2);
    }
  });

  it("focus を含む語がないときは長さを超えて出す", () => {
    const g = new Generator(
      words(),
      { focusRatio: 1, skew: 2 },
      seededRandom(1),
    );
    const allowed = unitsFor(maxLevel());
    const word = g.word(allowed, ["ぴゃ"], 2);
    expect(word).toContain("ぴゃ");
  });

  it("打てる語がなければ例外", () => {
    const g = new Generator(words(), undefined, seededRandom(1));
    expect(() => g.word(["っ"], [], 0)).toThrow();
  });

  it("countWithUnit は unit を含む語を数える", () => {
    const g = new Generator(words(), undefined, seededRandom(1));
    expect(g.countWithUnit(unitsFor(1), "る")).toBeGreaterThan(0);
    expect(g.countWithUnit(["っ"], "っ")).toBe(0);
  });
});
