// 設定と進捗をURLで共有するためのエンコード。
// JSON → deflate-raw 圧縮 → base64url。URLのハッシュフラグメントに載せる。

import { sanitize, type Settings } from "./settings";
import type { State } from "./store";
import type { UnitStat, WordRecord } from "./drill";

interface SharePayload {
  v: 1;
  settings: Settings;
  state: State;
}

async function pipe(
  bytes: Uint8Array,
  stream: CompressionStream | DecompressionStream,
): Promise<Uint8Array> {
  const blob = new Blob([bytes as BlobPart]);
  const compressed = blob.stream().pipeThrough(stream);
  return new Uint8Array(await new Response(compressed).arrayBuffer());
}

function toBase64Url(bytes: Uint8Array): string {
  let bin = "";
  for (const b of bytes) bin += String.fromCharCode(b);
  return btoa(bin).replaceAll("+", "-").replaceAll("/", "_").replace(/=+$/, "");
}

function fromBase64Url(s: string): Uint8Array {
  const b64 = s.replaceAll("-", "+").replaceAll("_", "/");
  const bin = atob(b64);
  const bytes = new Uint8Array(bin.length);
  for (let i = 0; i < bin.length; i++) bytes[i] = bin.charCodeAt(i);
  return bytes;
}

export async function encodeShare(
  settings: Settings,
  state: State,
): Promise<string> {
  const payload: SharePayload = { v: 1, settings, state };
  const raw = new TextEncoder().encode(JSON.stringify(payload));
  const compressed = await pipe(raw, new CompressionStream("deflate-raw"));
  return toBase64Url(compressed);
}

// 共有リンクを復元する。壊れていたら null。
export async function decodeShare(
  encoded: string,
): Promise<{ settings: Settings; state: State } | null> {
  try {
    const compressed = fromBase64Url(encoded);
    const raw = await pipe(compressed, new DecompressionStream("deflate-raw"));
    const parsed = JSON.parse(new TextDecoder().decode(raw)) as SharePayload;
    if (parsed.v !== 1) return null;
    return {
      settings: sanitize(parsed.settings ?? {}),
      state: sanitizeState(parsed.state),
    };
  } catch {
    return null;
  }
}

const MAX_RECORDS = 500;
const MAX_RECENT = 12;

function int(v: unknown, min: number, max: number): number | null {
  if (typeof v !== "number" || !Number.isFinite(v)) return null;
  return Math.min(Math.max(Math.round(v), min), max);
}

// 共有されてきた進捗を型と範囲で検証する。壊れた要素は捨てる。
export function sanitizeState(raw: unknown): State {
  const state: State = { level: 1, stats: {}, records: [], newKanaWords: 0 };
  if (typeof raw !== "object" || raw === null) return state;
  const r = raw as Record<string, unknown>;

  state.level = int(r.level, 1, 100_000) ?? 1;
  state.newKanaWords = int(r.newKanaWords, 0, 100_000) ?? 0;

  if (typeof r.stats === "object" && r.stats !== null) {
    for (const [unit, value] of Object.entries(r.stats as Record<string, unknown>)) {
      if (typeof unit !== "string" || unit.length === 0 || unit.length > 4) continue;
      if (typeof value !== "object" || value === null) continue;
      const v = value as Record<string, unknown>;
      const attempts = int(v.attempts, 0, 10_000_000);
      const errors = int(v.errors, 0, 10_000_000);
      if (attempts === null || errors === null) continue;
      const recent = Array.isArray(v.recent)
        ? v.recent.filter((b): b is boolean => typeof b === "boolean").slice(-MAX_RECENT)
        : [];
      const stat: UnitStat = { attempts, errors, recent };
      state.stats[unit] = stat;
    }
  }

  if (Array.isArray(r.records)) {
    for (const value of r.records.slice(-MAX_RECORDS)) {
      if (typeof value !== "object" || value === null) continue;
      const v = value as Record<string, unknown>;
      const units = int(v.units, 0, 100);
      const keys = int(v.keys, 0, 1000);
      const errors = int(v.errors, 0, 1000);
      const typingMs = int(v.typingMs, 1, 3_600_000);
      if (units === null || keys === null || errors === null || typingMs === null) {
        continue;
      }
      const rec: WordRecord = {
        success: v.success === true,
        units,
        keys,
        errors,
        typingMs,
      };
      state.records.push(rec);
    }
  }
  return state;
}
