import { describe, expect, it } from "vitest";
import { Engine } from "./engine";
import { keysOf, KeySpace, type Key } from "./keys";
import { TABLE, chordFor } from "./table";

// キー添字 (keys.ts の KEYS 順)
const F = 13, G = 14, H = 15, J = 16, K = 17, M = 26, O = 8, I = 7, W = 1;
const SEMI = 19, V = 23, U = 6, Y = 5;

// 押して離す操作列を実行して確定した文字を集める。
// "+J" は押す、"-J" は離す。
function run(engine: Engine, ops: Array<["+" | "-", Key]>): string[] {
  const out: string[] = [];
  for (const [dir, key] of ops) {
    const ems = dir === "+" ? engine.press(key) : engine.release(key);
    out.push(...ems.map((e) => e.text));
  }
  return out;
}

// タップ (押してすぐ離す)。
function tap(...keys: Key[]): Array<["+" | "-", Key]> {
  const ops: Array<["+" | "-", Key]> = [];
  for (const k of keys) ops.push(["+", k]);
  for (const k of [...keys].reverse()) ops.push(["-", k]);
  return ops;
}

describe("Engine", () => {
  it("単打のあ", () => {
    expect(run(new Engine(), tap(J))).toEqual(["あ"]);
  });

  it("連続入力あいなする", () => {
    const e = new Engine();
    const got = [
      ...run(e, tap(J)),
      ...run(e, tap(K)),
      ...run(e, tap(M)),
      ...run(e, tap(O)),
      ...run(e, tap(I)),
    ];
    expect(got).toEqual(["あ", "い", "な", "す", "る"]);
  });

  it("同時押しの濁音が", () => {
    expect(run(new Engine(), tap(F, J))).toEqual(["が"]);
  });

  it("逆順の同時押しが", () => {
    expect(run(new Engine(), tap(J, F))).toEqual(["が"]);
  });

  it("シフトのの", () => {
    expect(run(new Engine(), tap(KeySpace, J))).toEqual(["の"]);
  });

  it("連続シフト: スペースを押しっぱなしで2文字", () => {
    const e = new Engine();
    const got = run(e, [
      ["+", KeySpace],
      ["+", J],
      ["-", J], // の
      ["+", K],
      ["-", K], // も
      ["-", KeySpace],
    ]);
    expect(got).toEqual(["の", "も"]);
  });

  it("連続濁音: Fを押しっぱなしでずを2回", () => {
    const e = new Engine();
    const got = run(e, [
      ["+", F],
      ["+", O],
      ["-", O],
      ["+", O],
      ["-", O],
      ["-", F],
    ]);
    expect(got).toEqual(["ず", "ず"]);
  });

  it("ロールオーバー: がの直後のあ", () => {
    const e = new Engine();
    const got = run(e, [
      ["+", F],
      ["+", J],
      ["-", F], // が確定
      ["-", J],
      ["+", J],
      ["-", J], // あ
    ]);
    expect(got).toEqual(["が", "あ"]);
  });

  it("拗音きゃと3キーの濁音拗音ぎゃ", () => {
    expect(run(new Engine(), tap(W, H))).toEqual(["きゃ"]);
    expect(run(new Engine(), tap(J, W, H))).toEqual(["ぎゃ"]);
  });

  it("外来音ふぁ", () => {
    expect(run(new Engine(), tap(V, SEMI, J))).toEqual(["ふぁ"]);
  });

  it("スペース単打は空白", () => {
    expect(run(new Engine(), tap(KeySpace))).toEqual([" "]);
  });

  it("かなのないキーは捨てる", () => {
    expect(run(new Engine(), tap(Y))).toEqual([]);
  });

  it("バックスペース", () => {
    expect(run(new Engine(), tap(U))).toEqual(["\b"]);
  });

  it("促音っは単打で確定", () => {
    expect(run(new Engine(), tap(G))).toEqual(["っ"]);
  });

  it("reset は未確定のキーを捨てる", () => {
    const e = new Engine();
    e.press(F);
    e.reset();
    const got = [...run(e, [["-", F]]), ...run(e, tap(J))];
    expect(got).toEqual(["あ"]);
  });

  it("テーブルの全エントリがタップで確定できる", () => {
    const seen = new Set<string>();
    for (const entry of TABLE) {
      if (seen.has(entry.text)) continue; // 別名は正規の打鍵だけ
      seen.add(entry.text);
      const e = new Engine();
      const got = run(e, tap(...keysOf(chordFor(entry.text)!)));
      expect(got, entry.text).toEqual([entry.text]);
    }
  });
});
