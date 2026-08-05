import { describe, expect, it } from "vitest";
import { DEFAULT_DRILL_CONFIG, Drill, type DrillConfig } from "./drill";
import { maxLevel, stageFor, unitsFor } from "./curriculum";

function cfgWith(overrides: Partial<DrillConfig>): DrillConfig {
  return { ...DEFAULT_DRILL_CONFIG, ...overrides };
}

// units の単語を出題し、正しく打ち切って結果を返す。
function typeWord(d: Drill, units: string[], elapsedMs: number) {
  d.startWord(units, 0);
  for (const u of units) d.input(u);
  return d.finishWord(elapsedMs);
}

describe("curriculum", () => {
  it("レベル1はあいなするの5文字", () => {
    expect(unitsFor(1)).toEqual(["あ", "い", "な", "す", "る"]);
  });

  it("かな1文字ごとに長さ2→3→4→5の4段階を踏む", () => {
    expect(stageFor(1)).toMatchObject({ maxLen: 2 });
    expect(stageFor(4)).toMatchObject({ maxLen: 5 });
    expect(stageFor(5).units.length).toBe(6);
    expect(stageFor(5).maxLen).toBe(2);
  });

  it("最終レベルで長さ全開放", () => {
    expect(stageFor(maxLevel()).maxLen).toBe(0);
  });
});

describe("Drill", () => {
  it("正しい入力で進み、間違いは進まない", () => {
    const d = new Drill(cfgWith({}), 1);
    d.startWord(["あ", "い"], 0);
    expect(d.input("い")).toBe("error");
    expect(d.currentPos()).toBe(0);
    expect(d.input("あ")).toBe("advance");
    expect(d.input("い")).toBe("wordDone");
  });

  it("成功はミスの有無だけで決まる", () => {
    const d = new Drill(cfgWith({}), 1);
    expect(typeWord(d, ["あ", "い"], 60_000).success).toBe(true); // 遅くても成功
    d.startWord(["あ", "い"], 0);
    d.input("い"); // ミス
    d.input("あ");
    d.input("い");
    expect(d.finishWord(1000).success).toBe(false);
  });

  it("kpm は反応の猶予を引いて計算する", () => {
    const d = new Drill(cfgWith({}), 1);
    // 「ある」= 2打鍵、経過1.5秒 - 猶予0.5秒 = 1秒 → 120kpm
    typeWord(d, ["あ", "る"], 1500);
    expect(d.windowKPM()).toBeCloseTo(120);
  });

  it("kpm とミス率の両方を満たすと昇格しカウンターが空になる", () => {
    const d = new Drill(cfgWith({ windowSize: 20, minNewKanaWords: 10 }), 1);
    let out;
    for (let i = 0; i < 20; i++) out = typeWord(d, ["あ", "る"], 900);
    expect(out!.promoted).toBe(true);
    expect(out!.kanaAdded).toBe(false); // レベル1→2は長さの解放
    expect(d.level).toBe(2);
    expect(d.successCount().total).toBe(0);
    expect(d.newKanaWords()).toBe(0);
  });

  it("kpm 不足では昇格しない", () => {
    const d = new Drill(cfgWith({ windowSize: 10, minNewKanaWords: 0 }), 1);
    for (let i = 0; i < 20; i++) {
      expect(typeWord(d, ["あ", "る"], 3000).promoted).toBe(false);
    }
    expect(d.level).toBe(1);
  });

  it("ミス率が高いと昇格しない", () => {
    const d = new Drill(cfgWith({ windowSize: 10, minNewKanaWords: 0 }), 1);
    for (let i = 0; i < 20; i++) {
      d.startWord(["あ", "る"], 0);
      d.input("い"); // ミス
      d.input("あ");
      d.input("る");
      expect(d.finishWord(900).promoted).toBe(false);
    }
    expect(d.level).toBe(1);
  });

  it("新出かなを含む語が足りないと昇格しない", () => {
    const d = new Drill(cfgWith({ windowSize: 5, minNewKanaWords: 10 }), 1);
    for (let i = 0; i < 30; i++) {
      // 「あい」は新出かな「る」を含まない
      expect(typeWord(d, ["あ", "い"], 900).promoted).toBe(false);
    }
    let out;
    for (let i = 0; i < 10; i++) out = typeWord(d, ["あ", "る"], 900);
    expect(out!.promoted).toBe(true);
  });

  it("語彙が薄いかなはゲートが緩む", () => {
    const d = new Drill(cfgWith({ minNewKanaWords: 50 }), 1);
    d.setNewKanaSupply(4);
    expect(d.gateTarget()).toBe(20);
  });

  it("レベル4→5の昇格でかなが増えて2文字語に戻る", () => {
    const d = new Drill(cfgWith({ windowSize: 2, minNewKanaWords: 0 }), 4);
    let out;
    for (let i = 0; i < 2; i++) out = typeWord(d, ["あ", "る"], 900);
    expect(out!.promoted).toBe(true);
    expect(out!.kanaAdded).toBe(true);
    expect(d.allowed().length).toBe(6);
    expect(d.stage().maxLen).toBe(2);
  });

  it("古いかなの正答率低下で降格し、連鎖しない", () => {
    const d = new Drill(cfgWith({}), 5);
    let out;
    for (let i = 0; i < 6; i++) {
      d.startWord(["あ"], 0);
      d.input("い"); // ミス
      d.input("あ");
      out = d.finishWord(1000);
    }
    expect(out!.demoted).toBe(true);
    expect(out!.weakUnit).toBe("あ");
    expect(d.level).toBe(4);
    expect(typeWord(d, ["あ", "い"], 1000).demoted).toBe(false);
  });

  it("新出かなのミスでは降格しない", () => {
    const d = new Drill(cfgWith({}), 5);
    const allowed = d.allowed();
    const newest = allowed[allowed.length - 1];
    for (let i = 0; i < 12; i++) {
      d.startWord([newest], 0);
      d.input(allowed[0]); // ミス
      d.input(newest);
      expect(d.finishWord(1000).demoted).toBe(false);
    }
    expect(d.level).toBe(5);
  });

  it("進捗の保存と復元", () => {
    const d = new Drill(cfgWith({ windowSize: 5 }), 1);
    typeWord(d, ["あ", "る"], 900);
    const { records, newKanaWords } = d.progress();
    const d2 = new Drill(cfgWith({ windowSize: 5 }), 1);
    d2.setProgress(records, newKanaWords);
    expect(d2.successCount()).toEqual({ successes: 1, total: 1 });
    expect(d2.newKanaWords()).toBe(1);
  });
});
