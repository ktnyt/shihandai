// 薙刀式で使う物理キー (QWERTY配列上の位置)。
// Go版 internal/naginata/key.go の移植。

export const KEYS = [
  "Q", "W", "E", "R", "T", "Y", "U", "I", "O", "P",
  "A", "S", "D", "F", "G", "H", "J", "K", "L", ";",
  "Z", "X", "C", "V", "B", "N", "M", ",", ".", "/",
  "Space",
] as const;

export type Key = number; // KEYS の添字
export const KeySpace: Key = 30;

export function keyLabel(k: Key): string {
  return KEYS[k] ?? "?";
}

// KeySet はキーの組み合わせのビットマスク (31ビットで足りる)。
export type KeySet = number;

export function set(...keys: Key[]): KeySet {
  let s = 0;
  for (const k of keys) s |= 1 << k;
  return s;
}

export function has(s: KeySet, k: Key): boolean {
  return (s & (1 << k)) !== 0;
}

export function count(s: KeySet): number {
  let c = 0;
  while (s > 0) {
    c += s & 1;
    s >>>= 1;
  }
  return c;
}

export function keysOf(s: KeySet): Key[] {
  const keys: Key[] = [];
  for (let k = 0; k < KEYS.length; k++) if (has(s, k)) keys.push(k);
  return keys;
}

// "F+J" のような表示名。Space は先頭に置く。
export function chordLabel(s: KeySet): string {
  const parts: string[] = [];
  if (has(s, KeySpace)) parts.push("Space");
  for (let k = 0; k < KEYS.length; k++) {
    if (k !== KeySpace && has(s, k)) parts.push(keyLabel(k));
  }
  return parts.join("+");
}

// KeyboardEvent.code から薙刀式のキーへ。物理位置基準なので
// OS側のレイアウトに依存しない。
const codeToKey = new Map<string, Key>();
for (let k = 0; k < KEYS.length; k++) {
  const label = KEYS[k];
  if (label === "Space") codeToKey.set("Space", k);
  else if (label === ";") codeToKey.set("Semicolon", k);
  else if (label === ",") codeToKey.set("Comma", k);
  else if (label === ".") codeToKey.set("Period", k);
  else if (label === "/") codeToKey.set("Slash", k);
  else codeToKey.set(`Key${label}`, k);
}

export function keyFromCode(code: string): Key | undefined {
  return codeToKey.get(code);
}
