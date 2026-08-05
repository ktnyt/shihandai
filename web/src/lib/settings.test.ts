import { describe, expect, it } from "vitest";
import { DEFAULT_SETTINGS, sanitize } from "./settings";

describe("sanitize", () => {
  it("空の入力は既定値になる", () => {
    expect(sanitize({})).toEqual(DEFAULT_SETTINGS);
  });

  it("範囲外の値はクランプされる", () => {
    const s = sanitize({ targetKPM: 10000, maxMissRate: -1, windowSize: 3 });
    expect(s.targetKPM).toBe(400);
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
