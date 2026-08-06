import { useEffect, useMemo, useReducer, useRef, useState } from "preact/hooks";
import { Engine } from "./lib/engine";
import { Drill, DEFAULT_DRILL_CONFIG, recentAccuracy } from "./lib/drill";
import { Generator } from "./lib/lesson";
import { words } from "./lib/dict";
import { Session } from "./lib/session";
import { chordFor } from "./lib/table";
import { chordLabel, keyFromCode, KeySpace } from "./lib/keys";
import { groupOf, maxLevel } from "./lib/curriculum";
import * as store from "./lib/store";
import { encodeShare } from "./lib/share";
import { SoundPlayer, SOUND_TYPES } from "./lib/sound";
import {
  loadSettings,
  matchPreset,
  PRESETS,
  sanitize,
  saveSettings,
  type PresetSettings,
  type Settings,
} from "./lib/settings";

const KPM_METER_MAX = 180;

// 設定を練習中のセッションに反映する。
function applySettings(session: Session, sound: SoundPlayer, s: Settings): void {
  sound.enabled = s.soundEnabled;
  sound.type = s.soundType;
  const cfg = session.drill.cfg;
  cfg.targetKPM = s.targetKPM;
  cfg.maxMissRate = s.maxMissRate;
  cfg.windowSize = s.windowSize;
  cfg.minNewKanaWords = s.minNewKanaWords;
  cfg.requireBackspace = s.requireBackspace;
  session.intervalMs = s.intervalMs;
  session.upcomingWords = s.upcomingWords;
}

function createSession(
  settings: Settings,
  sound: SoundPlayer,
  onChange: () => void,
  onSaved: () => void,
): Session {
  sound.enabled = settings.soundEnabled;
  sound.type = settings.soundType;
  const cfg = { ...DEFAULT_DRILL_CONFIG };
  cfg.targetKPM = settings.targetKPM;
  cfg.maxMissRate = settings.maxMissRate;
  cfg.windowSize = settings.windowSize;
  cfg.minNewKanaWords = settings.minNewKanaWords;
  cfg.requireBackspace = settings.requireBackspace;

  const state = store.load();
  const drill = new Drill(cfg, state.level, state.stats);
  drill.setProgress(state.records, state.newKanaWords);

  const session = new Session(
    new Engine(),
    drill,
    new Generator(words()),
    {
      intervalMs: settings.intervalMs,
      upcomingWords: settings.upcomingWords,
      now: () => performance.now(),
      schedule: (fn, ms) => void setTimeout(fn, ms),
      onChange,
      onSave: () => {
        const { records, newKanaWords } = drill.progress();
        store.save({ level: drill.level, stats: drill.stats, records, newKanaWords });
        onSaved();
      },
      onEvent: (event) => {
        // 打鍵音だけで練習の流れを邪魔しない。鳴らすのはレベルアップのみ
        if (event === "promoted") sound.levelup();
      },
    },
  );
  return session;
}

export function App() {
  const [, bump] = useReducer((x: number) => x + 1, 0);
  const [settings, setSettings] = useState(() => loadSettings());
  const [settingsOpen, setSettingsOpen] = useState(false);
  const syncUrlRef = useRef<() => void>(() => {});
  const sound = useMemo(() => new SoundPlayer(), []);
  const session = useMemo(
    () =>
      createSession(loadSettings(), sound, () => bump(0), () => syncUrlRef.current()),
    [],
  );
  const drill = session.drill;

  // 進捗が保存されるたびにアドレスバーのURLを最新の共有リンクにする。
  // 打鍵のじゃまをしないよう少し遅らせてまとめる
  const settingsRef = useRef(settings);
  const urlTimer = useRef<ReturnType<typeof setTimeout> | undefined>(undefined);
  const syncUrl = () => {
    clearTimeout(urlTimer.current);
    urlTimer.current = setTimeout(async () => {
      const { records, newKanaWords } = drill.progress();
      const encoded = await encodeShare(settingsRef.current, {
        level: drill.level,
        stats: drill.stats,
        records,
        newKanaWords,
      });
      history.replaceState(
        null,
        "",
        `${location.pathname}${location.search}#p=${encoded}`,
      );
    }, 500);
  };
  syncUrlRef.current = syncUrl;

  const openSettings = () => {
    session.pause();
    setSettingsOpen(true);
  };
  const closeSettings = () => setSettingsOpen(false);
  const changeSettings = (next: Settings) => {
    setSettings(next);
    settingsRef.current = next;
    saveSettings(next);
    applySettings(session, sound, next);
    sound.warm(); // 種類が変わっていたら読み込み直す
    syncUrl();
    bump(0);
  };

  useEffect(() => {
    const onKeydown = (e: KeyboardEvent) => {
      if (settingsOpen) {
        if (e.key === "Escape") setSettingsOpen(false);
        return; // パネルの入力を邪魔しない
      }
      if (e.metaKey || e.ctrlKey || e.altKey) return;
      if (e.key === "Escape") {
        if (session.state === "paused" || session.state === "ready") {
          // 停止中の Esc は設定を開く
          openSettings();
        } else {
          session.pause();
        }
        return;
      }
      const key = keyFromCode(e.code);
      if (key === undefined) return;
      e.preventDefault();
      if (e.repeat) return;
      if (key === KeySpace && session.state === "ready") {
        sound.warm(); // 最初の操作でサンプルを読み込む
        session.start();
        return;
      }
      if (key === KeySpace && session.state === "paused") {
        session.resume();
        return;
      }
      if (key === KeySpace && session.state === "leveledUp") {
        session.continueLevelUp();
        return;
      }
      if (session.state === "typing") sound.key(key === KeySpace);
      session.keydown(key);
    };
    const onKeyup = (e: KeyboardEvent) => {
      const key = keyFromCode(e.code);
      if (key === undefined) return;
      if (session.state === "typing") sound.release(key === KeySpace);
      session.keyup(key);
    };
    // フォーカスが外れると keyup を取り逃がすので一時停止する
    const onBlur = () => session.pause();
    window.addEventListener("keydown", onKeydown);
    window.addEventListener("keyup", onKeyup);
    window.addEventListener("blur", onBlur);
    return () => {
      window.removeEventListener("keydown", onKeydown);
      window.removeEventListener("keyup", onKeyup);
      window.removeEventListener("blur", onBlur);
    };
  }, [session, settingsOpen]);

  // 経過時間の表示を進めるための再描画
  useEffect(() => {
    const timer = setInterval(() => {
      if (session.state === "typing") bump(0);
    }, 100);
    return () => clearInterval(timer);
  }, [session]);

  const allowed = drill.allowed();
  const newest = drill.newest();
  const stage = drill.stage();
  const { successes, total } = drill.successCount();
  const kpm = drill.windowKPM();
  const missRate = drill.missRate();
  const cfg = drill.cfg;

  const missOK = total > 0 && missRate <= cfg.maxMissRate;
  const kpmOK = kpm >= cfg.targetKPM;

  const onReset = () => {
    if (confirm("進捗を消して最初からやり直しますか？")) {
      // URLに残った古い進捗も消してからリロードする
      clearTimeout(urlTimer.current);
      store.reset();
      history.replaceState(null, "", location.pathname + location.search);
      location.reload();
    }
  };

  const onShare = async () => {
    session.pause();
    const { records, newKanaWords } = drill.progress();
    const encoded = await encodeShare(settings, {
      level: drill.level,
      stats: drill.stats,
      records,
      newKanaWords,
    });
    const url = `${location.origin}${location.pathname}${location.search}#p=${encoded}`;
    history.replaceState(null, "", `${location.pathname}${location.search}#p=${encoded}`);
    try {
      await navigator.clipboard.writeText(url);
      session.message = "共有リンクをコピーした (設定と進捗が入っている)";
    } catch {
      prompt("このURLをコピーしてください", url);
    }
    bump(0);
  };

  return (
    <main class="app">
      <header>
        <h1>
          shihandai <span class="sub">— 薙刀式タイピング練習</span>
        </h1>
        <div class="meta">
          レベル {drill.level}/{maxLevel()}　かな {allowed.length}文字　ながさ{" "}
          {stage.maxLen > 0 ? `${stage.maxLen}文字まで` : "せいげんなし"}
          　いまの段階: {groupOf(newest)}
        </div>
        <div class="kana-list">
          使えるかな: {allowed.join(" ")}
        </div>
      </header>

      {session.state === "leveledUp" ? (
        <section class="levelup">
          <div class="levelup-title">レベルアップ! レベル {drill.level}</div>
          {session.kanaAdded ? (
            <div class="levelup-body">
              あたらしいかな: <span class="unit current">{newest}</span>{" "}
              <span class="hint">[{chordLabel(chordFor(newest) ?? 0)}]</span>
            </div>
          ) : (
            <div class="levelup-body">
              {stage.maxLen > 0
                ? `ながさ${stage.maxLen}文字までの語がでるようになった`
                : "ながさのせいげんがなくなった"}
            </div>
          )}
          <div class="faint">Space ではじめる</div>
        </section>
      ) : (
        <section class="play">
          <WordStream session={session} />
          <HintLine session={session} />
          <div class="timer">
            {session.state === "typing"
              ? `${(drill.elapsedMs(performance.now()) / 1000).toFixed(1)}s`
              : "--.-s"}
            　ミス {drill.currentErrors()}
          </div>
        </section>
      )}

      <section class="gauges">
        <div class="bar">
          <div
            class={`fill ${missOK ? "ok" : "pending"}`}
            style={{ width: pct(successes, cfg.windowSize) }}
          />
          <div class="fill bad" style={{ width: pct(total - successes, cfg.windowSize) }} />
        </div>
        <div class="bar meter">
          <div
            class={`fill ${kpmOK ? "ok" : "pending"}`}
            style={{ width: pct(Math.min(kpm, KPM_METER_MAX), KPM_METER_MAX) }}
          />
          <div
            class="marker"
            style={{ left: pct(cfg.targetKPM, KPM_METER_MAX) }}
          />
        </div>
        <div class="stats">
          kpm {kpm.toFixed(0)}/{cfg.targetKPM}　ミス率 {(missRate * 100).toFixed(1)}%/
          {(cfg.maxMissRate * 100).toFixed(1)}%　直近 {total}/{cfg.windowSize} 語
        </div>
        <div class="faint">
          「{newest}」を含む語 {drill.newKanaWords()}/{drill.gateTarget()}
        </div>
      </section>

      <section class="messages">
        <div class="flash">{session.flash}</div>
        <div class="notice">{session.message}</div>
        <div class="faint">{weakLabel(session)}</div>
      </section>

      <footer>
        <span class="faint">
          Esc で一時停止 (停止中はもう一度で設定)　IME はオフ (直接入力) にして使う
        </span>
        <span class="footer-buttons">
          <button class="ghost" onClick={onShare}>
            共有
          </button>
          <button class="ghost" onClick={openSettings}>
            設定
          </button>
          <button class="ghost danger" onClick={onReset}>
            進捗リセット
          </button>
        </span>
      </footer>

      {settingsOpen && (
        <SettingsPanel
          settings={settings}
          onChange={changeSettings}
          onClose={closeSettings}
        />
      )}
    </main>
  );
}

interface FieldSpec {
  key: Exclude<keyof PresetSettings, "requireBackspace">;
  label: string;
  note: string;
  min: number;
  max: number;
  step: number;
  // 表示値と内部値の変換 (ミス率は%で見せる)
  toView?: (v: number) => number;
  fromView?: (v: number) => number;
}

const FIELDS: FieldSpec[] = [
  { key: "targetKPM", label: "目標打鍵速度 (kpm)", note: "昇格に必要な速度", min: 30, max: 600, step: 5 },
  {
    key: "maxMissRate", label: "ミス率の上限 (%)", note: "これ以下なら昇格できる",
    min: 0.1, max: 50, step: 0.1,
    toView: (v) => Math.round(v * 1000) / 10, fromView: (v) => v / 100,
  },
  { key: "upcomingWords", label: "先に見える単語数", note: "いま打つ単語の右に見える先読みの数", min: 0, max: 10, step: 1 },
  { key: "windowSize", label: "判定に使う直近の単語数", note: "この語数分の kpm とミス率で昇格を判定", min: 10, max: 500, step: 10 },
  { key: "minNewKanaWords", label: "新出かなの語数", note: "昇格までに打つ、新出かなを含む語の最低数", min: 0, max: 300, step: 5 },
  { key: "intervalMs", label: "単語間インターバル (ms)", note: "次の単語が出るまで入力を受け付けない時間", min: 0, max: 3000, step: 50 },
];

function SettingsPanel({
  settings,
  onChange,
  onClose,
}: {
  settings: Settings;
  onChange: (s: Settings) => void;
  onClose: () => void;
}) {
  const update = (key: keyof Settings, viewValue: number, spec: FieldSpec) => {
    const value = spec.fromView ? spec.fromView(viewValue) : viewValue;
    onChange(sanitize({ ...settings, [key]: value }));
  };
  const activePreset = matchPreset(settings);

  return (
    <div class="overlay" onClick={onClose}>
      <div class="panel" onClick={(e) => e.stopPropagation()}>
        <h2>設定</h2>
        <div class="presets">
          {PRESETS.map((p) => (
            <button
              key={p.name}
              class={`preset ${activePreset === p.name ? "active" : ""}`}
              onClick={() =>
                onChange({
                  ...p.settings,
                  soundEnabled: settings.soundEnabled,
                  soundType: settings.soundType,
                })
              }
            >
              {p.name}
            </button>
          ))}
          {activePreset === null && <span class="preset custom">カスタム</span>}
        </div>
        {FIELDS.map((f) => (
          <label key={f.key} class="field">
            <span class="field-label">
              {f.label}
              <span class="field-note">{f.note}</span>
            </span>
            <input
              type="number"
              min={f.min}
              max={f.max}
              step={f.step}
              value={f.toView ? f.toView(settings[f.key]) : settings[f.key]}
              onChange={(e) =>
                update(f.key, Number((e.target as HTMLInputElement).value), f)
              }
            />
          </label>
        ))}
        <label class="field">
          <span class="field-label">
            ミス時のバックスペース修正
            <span class="field-note">
              オンだとミスの後に BS (Uキー単打) で消すまで進めない
            </span>
          </span>
          <input
            type="checkbox"
            checked={settings.requireBackspace}
            onChange={(e) =>
              onChange(
                sanitize({
                  ...settings,
                  requireBackspace: (e.target as HTMLInputElement).checked,
                }),
              )
            }
          />
        </label>
        <label class="field">
          <span class="field-label">
            タイプ音
            <span class="field-note">打鍵に合わせて音を鳴らす</span>
          </span>
          <input
            type="checkbox"
            checked={settings.soundEnabled}
            onChange={(e) =>
              onChange(
                sanitize({
                  ...settings,
                  soundEnabled: (e.target as HTMLInputElement).checked,
                }),
              )
            }
          />
        </label>
        <label class="field">
          <span class="field-label">
            音の種類
            <span class="field-note">打てばそのまま試し聴きできる</span>
          </span>
          <select
            value={settings.soundType}
            disabled={!settings.soundEnabled}
            onChange={(e) =>
              onChange(
                sanitize({
                  ...settings,
                  soundType: (e.target as HTMLSelectElement)
                    .value as Settings["soundType"],
                }),
              )
            }
          >
            {SOUND_TYPES.map((t) => (
              <option key={t.value} value={t.value}>
                {t.label}
              </option>
            ))}
          </select>
        </label>
        <div class="panel-actions">
          <button class="primary" onClick={onClose}>
            閉じる (Esc)
          </button>
        </div>
        <p class="faint small">
          変更はすぐ保存され、この端末の次回起動にも引き継がれる。
        </p>
      </div>
    </div>
  );
}

function WordStream({ session }: { session: Session }) {
  const drill = session.drill;
  const word = drill.currentWord();
  const pos = drill.currentPos();
  const paused = session.state === "paused" || session.state === "ready";

  return (
    <div class="word-stream">
      <span class="word">
        {word.map((u, i) => (
          <span
            key={i}
            class={
              paused
                ? "unit masked"
                : i < pos
                  ? "unit done"
                  : i === pos
                    ? "unit current"
                    : "unit todo"
            }
          >
            {paused ? "●".repeat([...u].length) : u}
          </span>
        ))}
      </span>
      {session.upcoming.map((w, i) => (
        <span key={`u${i}`} class="word upcoming">
          {paused ? "●".repeat([...w.join("")].length) : w.join("")}
        </span>
      ))}
    </div>
  );
}

function HintLine({ session }: { session: Session }) {
  if (session.state === "ready") {
    return (
      <div class="hint-line">
        <span class="notice">Space ではじめる</span>{" "}
        <span class="faint">(IMEはオフにしておく)</span>
      </div>
    );
  }
  if (session.state === "paused") {
    return (
      <div class="hint-line">
        <span class="notice">一時停止中</span>{" "}
        <span class="faint">(Space で再開)</span>
      </div>
    );
  }
  if (session.state === "waiting") {
    return <div class="hint-line faint">つぎの単語へ…</div>;
  }
  const expected = session.drill.expected();
  const chord = expected ? chordFor(expected) : undefined;
  return (
    <div class="hint-line">
      {expected && chord !== undefined && (
        <>
          つぎ: <span class="unit current">{expected}</span>{" "}
          <span class="hint">[{chordLabel(chord)}]</span>
        </>
      )}
    </div>
  );
}

function weakLabel(session: Session): string {
  const drill = session.drill;
  const allowed = drill.allowed();
  const items = Object.entries(drill.stats)
    .filter(([u, s]) => allowed.includes(u) && s.recent.length > 0)
    .map(([u, s]) => [u, recentAccuracy(s)] as const)
    .filter(([, acc]) => acc < 1)
    .sort((a, b) => a[1] - b[1] || (a[0] < b[0] ? -1 : 1))
    .slice(0, 3);
  if (items.length === 0) return "";
  return "にがて: " + items.map(([u, acc]) => `${u} ${(acc * 100).toFixed(0)}%`).join("  ");
}

function pct(value: number, total: number): string {
  if (total <= 0) return "0%";
  return `${Math.min((value / total) * 100, 100)}%`;
}
