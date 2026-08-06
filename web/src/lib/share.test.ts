import { describe, expect, it } from "vitest";
import { decodeShare, encodeShare, sanitizeState } from "./share";
import { DEFAULT_SETTINGS } from "./settings";
import type { State } from "./store";
import type { WordRecord } from "./drill";

function fullState(): State {
  const stats: State["stats"] = {};
  const kana = "あいうえおかきくけこさしすせそたちつてとなにぬねの";
  for (const u of kana) {
    stats[u] = {
      attempts: 123,
      errors: 4,
      recent: Array.from({ length: 12 }, (_, i) => i % 3 !== 0),
    };
  }
  const records: WordRecord[] = Array.from({ length: 200 }, (_, i) => ({
    success: i % 7 !== 0,
    units: 2 + (i % 4),
    keys: 3 + (i % 5),
    errors: i % 7 === 0 ? 1 : 0,
    typingMs: 800 + ((i * 37) % 900),
  }));
  return { level: 137, stats, records, newKanaWords: 42 };
}

describe("share", () => {
  it("エンコードとデコードで往復できる", async () => {
    const state = fullState();
    const encoded = await encodeShare(DEFAULT_SETTINGS, state);
    expect(encoded).toMatch(/^[A-Za-z0-9_-]+$/); // URLセーフ

    const decoded = await decodeShare(encoded);
    expect(decoded).not.toBeNull();
    expect(decoded!.settings).toEqual(DEFAULT_SETTINGS);
    expect(decoded!.state).toEqual(state);
  });

  it("フルサイズの進捗でもURLに載る長さに収まる", async () => {
    const encoded = await encodeShare(DEFAULT_SETTINGS, fullState());
    expect(encoded.length).toBeLessThan(8000);
  });

  it("壊れた入力は null", async () => {
    expect(await decodeShare("こわれている")).toBeNull();
    expect(await decodeShare("AAAA")).toBeNull();
    expect(await decodeShare("")).toBeNull();
  });
});

describe("sanitizeState", () => {
  it("壊れた要素を捨てて安全な形にする", () => {
    const state = sanitizeState({
      level: 3.7,
      newKanaWords: -5,
      stats: {
        あ: { attempts: 10, errors: 2, recent: [true, "x", false] },
        ながすぎるキー: { attempts: 1, errors: 0, recent: [] },
        い: null,
        う: { attempts: "many", errors: 0, recent: [] },
      },
      records: [
        { success: true, units: 2, keys: 3, errors: 0, typingMs: 900.6 },
        { success: 1, units: 2, keys: 3, errors: 0, typingMs: NaN },
        "garbage",
      ],
    });
    expect(state.level).toBe(4);
    expect(state.newKanaWords).toBe(0);
    expect(Object.keys(state.stats)).toEqual(["あ"]);
    expect(state.stats["あ"].recent).toEqual([true, false]);
    expect(state.records).toHaveLength(1);
    expect(state.records[0].typingMs).toBe(901);
  });

  it("配列や null はまるごと初期状態になる", () => {
    expect(sanitizeState(null).level).toBe(1);
    expect(sanitizeState([1, 2, 3]).records).toEqual([]);
  });
});
