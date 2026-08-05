// 練習に使う単語辞書。Go版と同じ internal/dict/words.txt を読み込む。
import raw from "@dict/words.txt?raw";

let cached: string[] | null = null;

export function words(): string[] {
  if (cached === null) {
    cached = raw
      .split("\n")
      .map((line) => line.trim())
      .filter((line) => line !== "" && !line.startsWith("#"));
  }
  return cached;
}
