// タイプ音。Web Audio API で合成する (音声ファイルなし、遅延ほぼゼロ)。
// AudioContext はユーザー操作 (keydown) の中で初期化する必要がある。

export type SoundType = "mech" | "typewriter" | "pop";

export const SOUND_TYPES: { value: SoundType; label: string }[] = [
  { value: "mech", label: "メカニカル" },
  { value: "typewriter", label: "タイプライター" },
  { value: "pop", label: "ぽこぽこ" },
];

export class SoundPlayer {
  enabled = true;
  type: SoundType = "mech";

  private ctx: AudioContext | null = null;
  private noiseBuffer: AudioBuffer | null = null;

  // AudioContext を用意する。使えない環境では null。
  private ensure(): AudioContext | null {
    if (!this.enabled) return null;
    if (typeof AudioContext === "undefined") return null;
    if (!this.ctx) {
      this.ctx = new AudioContext();
      const len = Math.floor(this.ctx.sampleRate * 0.1);
      this.noiseBuffer = this.ctx.createBuffer(1, len, this.ctx.sampleRate);
      const data = this.noiseBuffer.getChannelData(0);
      for (let i = 0; i < len; i++) data[i] = Math.random() * 2 - 1;
    }
    if (this.ctx.state === "suspended") void this.ctx.resume();
    return this.ctx;
  }

  private envelope(
    ctx: AudioContext,
    peak: number,
    decay: number,
  ): GainNode {
    const gain = ctx.createGain();
    const t = ctx.currentTime;
    gain.gain.setValueAtTime(peak, t);
    gain.gain.exponentialRampToValueAtTime(0.001, t + decay);
    gain.connect(ctx.destination);
    return gain;
  }

  private noise(
    ctx: AudioContext,
    filterType: BiquadFilterType,
    freq: number,
    peak: number,
    decay: number,
  ): void {
    if (!this.noiseBuffer) return;
    const src = ctx.createBufferSource();
    src.buffer = this.noiseBuffer;
    const filter = ctx.createBiquadFilter();
    filter.type = filterType;
    filter.frequency.value = freq;
    src.connect(filter);
    filter.connect(this.envelope(ctx, peak, decay));
    src.start();
    src.stop(ctx.currentTime + decay);
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

  // 打鍵音。種類ごとに音色を変え、毎回わずかに揺らして機械っぽさを消す。
  key(): void {
    const ctx = this.ensure();
    if (!ctx) return;
    const jitter = 0.9 + Math.random() * 0.2;
    switch (this.type) {
      case "mech":
        this.noise(ctx, "bandpass", 1800 * jitter, 0.35, 0.05);
        this.tone(ctx, "square", 150 * jitter, 70, 0.1, 0.035);
        break;
      case "typewriter":
        this.noise(ctx, "highpass", 2600 * jitter, 0.4, 0.03);
        this.tone(ctx, "sine", 420 * jitter, 380, 0.08, 0.012);
        break;
      case "pop":
        this.tone(ctx, "sine", 520 * jitter, 240, 0.3, 0.07);
        break;
    }
  }

  // ミス音。低い濁った音。
  error(): void {
    const ctx = this.ensure();
    if (!ctx) return;
    this.tone(ctx, "square", 160, 110, 0.12, 0.14);
    this.tone(ctx, "square", 122, 84, 0.12, 0.14);
  }

  // 単語を打ち切った音。短い上昇二音。
  done(): void {
    const ctx = this.ensure();
    if (!ctx) return;
    this.tone(ctx, "sine", 880, 880, 0.1, 0.06);
    this.tone(ctx, "sine", 1175, 1175, 0.1, 0.09, 0.06);
  }

  // レベルアップのアルペジオ。
  levelup(): void {
    const ctx = this.ensure();
    if (!ctx) return;
    const notes = [523, 659, 784, 1047];
    notes.forEach((f, i) => this.tone(ctx, "triangle", f, f, 0.12, 0.18, i * 0.09));
  }
}
