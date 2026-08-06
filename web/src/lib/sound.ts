// タイプ音。実録のキースイッチ音 (kbsim, MIT License) を再生する。
// レベルアップの通知音だけは Web Audio で合成する。
// AudioContext はユーザー操作 (keydown) の中で初期化する必要がある。

export type SoundType = "mech" | "typewriter" | "pop";

export const SOUND_TYPES: { value: SoundType; label: string }[] = [
  { value: "mech", label: "メカニカル (茶軸)" },
  { value: "typewriter", label: "タイプライター (座屈バネ)" },
  { value: "pop", label: "スコスコ (静電容量)" },
];

// assets/sounds/<type>/<name>.mp3 を URL として取り込む
const sampleUrls = import.meta.glob("../assets/sounds/*/*.mp3", {
  eager: true,
  query: "?url",
  import: "default",
}) as Record<string, string>;

function urlsFor(type: SoundType, prefix: string): string[] {
  return Object.entries(sampleUrls)
    .filter(([path]) => path.includes(`/${type}/`) && path.includes(`/${prefix}`))
    .map(([, url]) => url)
    .sort();
}

export class SoundPlayer {
  enabled = true;
  type: SoundType = "mech";

  private ctx: AudioContext | null = null;
  private cache = new Map<string, AudioBuffer>();
  private loading = new Set<string>();

  // AudioContext を用意する。使えない環境では null。
  private ensure(): AudioContext | null {
    if (!this.enabled) return null;
    if (typeof AudioContext === "undefined") return null;
    if (!this.ctx) this.ctx = new AudioContext();
    if (this.ctx.state === "suspended") void this.ctx.resume();
    return this.ctx;
  }

  // 現在の種類のサンプルを読み込んでおく。
  warm(): void {
    const ctx = this.ensure();
    if (!ctx) return;
    for (const url of [
      ...urlsFor(this.type, "press_"),
      ...urlsFor(this.type, "release_"),
    ]) {
      void this.load(ctx, url);
    }
  }

  private async load(ctx: AudioContext, url: string): Promise<AudioBuffer | null> {
    const cached = this.cache.get(url);
    if (cached) return cached;
    if (this.loading.has(url)) return null;
    this.loading.add(url);
    try {
      const res = await fetch(url);
      const buf = await ctx.decodeAudioData(await res.arrayBuffer());
      this.cache.set(url, buf);
      return buf;
    } catch {
      return null; // 読み込めなくても練習は続ける
    } finally {
      this.loading.delete(url);
    }
  }

  private playSample(urls: string[], gain: number): void {
    const ctx = this.ensure();
    if (!ctx || urls.length === 0) return;
    const url = urls[Math.floor(Math.random() * urls.length)];
    const buf = this.cache.get(url);
    if (!buf) {
      // 未ロードなら裏で読み込む。次の打鍵から鳴る
      void this.load(ctx, url);
      return;
    }
    const src = ctx.createBufferSource();
    src.buffer = buf;
    // 毎回わずかにピッチを揺らして機械っぽさを消す
    src.playbackRate.value = 0.96 + Math.random() * 0.08;
    const g = ctx.createGain();
    g.gain.value = gain;
    src.connect(g);
    g.connect(ctx.destination);
    src.start();
  }

  // 打鍵音。スペース (シフト) はスペースバーの音を使う。
  key(isSpace = false): void {
    const urls = isSpace
      ? urlsFor(this.type, "press_SPACE")
      : urlsFor(this.type, "press_GENERIC");
    this.playSample(urls, 1);
  }

  // 離鍵音。押下より控えめに鳴らす。
  release(isSpace = false): void {
    const urls = isSpace
      ? urlsFor(this.type, "release_SPACE")
      : urlsFor(this.type, "release_GENERIC");
    this.playSample(urls, 0.6);
  }

  private tone(
    ctx: AudioContext,
    shape: OscillatorType,
    from: number,
    to: number,
    peak: number,
    decay: number,
    delay = 0,
  ): void {
    const osc = ctx.createOscillator();
    osc.type = shape;
    const t = ctx.currentTime + delay;
    osc.frequency.setValueAtTime(from, t);
    if (to !== from) osc.frequency.exponentialRampToValueAtTime(to, t + decay);
    const gain = ctx.createGain();
    gain.gain.setValueAtTime(peak, t);
    gain.gain.exponentialRampToValueAtTime(0.001, t + decay);
    osc.connect(gain);
    gain.connect(ctx.destination);
    osc.start(t);
    osc.stop(t + decay);
  }

  // レベルアップのアルペジオ。
  levelup(): void {
    const ctx = this.ensure();
    if (!ctx) return;
    const notes = [523, 659, 784, 1047];
    notes.forEach((f, i) => this.tone(ctx, "triangle", f, f, 0.12, 0.18, i * 0.09));
  }
}
