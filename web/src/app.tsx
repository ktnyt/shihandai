import { useEffect, useMemo, useReducer, useState } from "preact/hooks";
import { Engine } from "./lib/engine";
import { Drill, DEFAULT_DRILL_CONFIG, recentAccuracy } from "./lib/drill";
import { Generator } from "./lib/lesson";
import { words } from "./lib/dict";
import { Session } from "./lib/session";
import { chordFor } from "./lib/table";
import { chordLabel, keyFromCode, KeySpace } from "./lib/keys";
import { groupOf, maxLevel } from "./lib/curriculum";
import * as store from "./lib/store";
import {
  DEFAULT_SETTINGS,
  loadSettings,
  sanitize,
  saveSettings,
  type Settings,
} from "./lib/settings";

const KPM_METER_MAX = 180;

// 設定を練習中のセッションに反映する。
function applySettings(session: Session, s: Settings): void {
  const cfg = session.drill.cfg;
  cfg.targetKPM = s.targetKPM;
  cfg.maxMissRate = s.maxMissRate;
  cfg.reactionBudgetMs = s.reactionBudgetMs;
  cfg.windowSize = s.windowSize;
  cfg.minNewKanaWords = s.minNewKanaWords;
  session.intervalMs = s.intervalMs;
}

function createSession(settings: Settings, onChange: () => void): Session {
  const cfg = { ...DEFAULT_DRILL_CONFIG };
  cfg.targetKPM = settings.targetKPM;
  cfg.maxMissRate = settings.maxMissRate;
  cfg.reactionBudgetMs = settings.reactionBudgetMs;
  cfg.windowSize = settings.windowSize;
  cfg.minNewKanaWords = settings.minNewKanaWords;

  const state = store.load();
  const drill = new Drill(cfg, state.level, state.stats);
  drill.setProgress(state.records, state.newKanaWords);

  const session = new Session(
    new Engine(),
    drill,
    new Generator(words()),
    {
      intervalMs: settings.intervalMs,
      now: () => performance.now(),
      schedule: (fn, ms) => void setTimeout(fn, ms),
      onChange,
      onSave: () => {
        const { records, newKanaWords } = drill.progress();
        store.save({ level: drill.level, stats: drill.stats, records, newKanaWords });
      },
    },
  );
  return session;
}

export function App() {
  const [, bump] = useReducer((x: number) => x + 1, 0);
  const [settings, setSettings] = useState(() => loadSettings());
  const [settingsOpen, setSettingsOpen] = useState(false);
  const session = useMemo(() => createSession(loadSettings(), () => bump(0)), []);
  const drill = session.drill;

  const openSettings = () => {
    session.pause();
    setSettingsOpen(true);
  };
  const closeSettings = () => setSettingsOpen(false);
  const changeSettings = (next: Settings) => {
    setSettings(next);
    saveSettings(next);
    applySettings(session, next);
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
        session.pause();
        return;
      }
      const key = keyFromCode(e.code);
      if (key === undefined) return;
      e.preventDefault();
      if (e.repeat) return;
      if (key === KeySpace && session.state === "ready") {
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
      session.keydown(key);
    };
    const onKeyup = (e: KeyboardEvent) => {
      const key = keyFromCode(e.code);
      if (key === undefined) return;
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
      store.reset();
      location.reload();
    }
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
          {groupOf(newest)}
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
              ながさ {stage.maxLen > 0 ? `${stage.maxLen}文字まで` : "せいげんなし"}{" "}
              の語がでるようになった
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
          Esc で一時停止・IMEはオフ (直接入力) にして使う
        </span>
        <span class="footer-buttons">
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
  key: keyof Settings;
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
  { key: "targetKPM", label: "目標打鍵速度 (kpm)", note: "昇格に必要な速度。制限時間にも使う", min: 30, max: 400, step: 5 },
  {
    key: "maxMissRate", label: "ミス率の上限 (%)", note: "これ以下なら昇格できる",
    min: 0.5, max: 50, step: 0.5,
    toView: (v) => Math.round(v * 1000) / 10, fromView: (v) => v / 100,
  },
  { key: "reactionBudgetMs", label: "反応の猶予 (ms)", note: "kpm の計算で経過時間から引く", min: 0, max: 3000, step: 50 },
  { key: "windowSize", label: "判定に使う単語数", note: "この窓の kpm とミス率で昇格を判定", min: 10, max: 500, step: 10 },
  { key: "minNewKanaWords", label: "新出かなの語数", note: "昇格までに打つ、新しいかなを含む語の回数", min: 0, max: 300, step: 5 },
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

  return (
    <div class="overlay" onClick={onClose}>
      <div class="panel" onClick={(e) => e.stopPropagation()}>
        <h2>設定</h2>
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
        <div class="panel-actions">
          <button class="ghost" onClick={() => onChange({ ...DEFAULT_SETTINGS })}>
            既定に戻す
          </button>
          <button class="primary" onClick={onClose}>
            閉じる (Esc)
          </button>
        </div>
        <p class="faint small">
          変更はすぐ保存され、この端末の次回起動にも引き継がれます。
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
