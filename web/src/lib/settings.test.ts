import { describe, expect, it } from "vitest";
import { DEFAULT_SETTINGS, matchPreset, PRESETS, sanitize } from "./settings";

describe("sanitize", () => {
  it("空の入力は既定値になる", () => {
    expect(sanitize({})).toEqual(DEFAULT_SETTINGS);
  });

  it("範囲外の値はクランプされる", () => {
    const s = sanitize({ targetKPM: 10000, maxMissRate: -1, windowSize: 3 });
    expect(s.targetKPM).toBe(600);
    expect(s.maxMissRate).toBe(0.001);
    expect(s.windowSize).toBe(10);
  });

  it("NaN は既定値になる", () => {
    expect(sanitize({ targetKPM: NaN }).targetKPM).toBe(
      DEFAULT_SETTINGS.targetKPM,
    );
  });

  it("小数の単語数は丸められる", () => {
    expect(sanitize({ windowSize: 55.7 }).windowSize).toBe(56);
  });
});

describe("presets", () => {
  it("既定は初心者モード", () => {
    expect(matchPreset(DEFAULT_SETTINGS)).toBe("初心者");
  });

  it("プリセットは sanitize で変化しない (共有時の互換性)", () => {
    for (const p of PRESETS) {
      expect(sanitize({ ...p.settings }), p.name).toEqual(p.settings);
      expect(matchPreset(p.settings)).toBe(p.name);
    }
  });

  it("値をいじるとカスタム扱いになる", () => {
    expect(matchPreset({ ...DEFAULT_SETTINGS, targetKPM: 61 })).toBeNull();
  });
});
