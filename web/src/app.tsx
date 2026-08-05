import { useEffect, useMemo, useReducer } from "preact/hooks";
import { Engine } from "./lib/engine";
import { Drill, DEFAULT_DRILL_CONFIG, recentAccuracy } from "./lib/drill";
import { Generator } from "./lib/lesson";
import { words } from "./lib/dict";
import { Session } from "./lib/session";
import { chordFor } from "./lib/table";
import { chordLabel, keyFromCode, KeySpace } from "./lib/keys";
import { groupOf, maxLevel } from "./lib/curriculum";
import * as store from "./lib/store";

const KPM_METER_MAX = 180;
const INTERVAL_MS = 500;

function createSession(onChange: () => void): Session {
  const params = new URLSearchParams(location.search);
  const cfg = { ...DEFAULT_DRILL_CONFIG };
  const kpm = Number(params.get("kpm"));
  if (kpm > 0) cfg.targetKPM = kpm;
  const missRate = Number(params.get("missrate"));
  if (missRate > 0) cfg.maxMissRate = missRate;

  const state = store.load();
  const drill = new Drill(cfg, state.level, state.stats);
  drill.setProgress(state.records, state.newKanaWords);

  const session = new Session(
    new Engine(),
    drill,
    new Generator(words()),
    {
      intervalMs: INTERVAL_MS,
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
  const session = useMemo(() => createSession(() => bump(0)), []);
  const drill = session.drill;

  useEffect(() => {
    const onKeydown = (e: KeyboardEvent) => {
      if (e.metaKey || e.ctrlKey || e.altKey) return;
      if (e.key === "Escape") {
        session.pause();
        return;
      }
      const key = keyFromCode(e.code);
      if (key === undefined) return;
      e.preventDefault();
      if (e.repeat) return;
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
  }, [session]);

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
        <button class="reset" onClick={onReset}>
          進捗リセット
        </button>
      </footer>
    </main>
  );
}

function WordStream({ session }: { session: Session }) {
  const drill = session.drill;
  const word = drill.currentWord();
  const pos = drill.currentPos();
  const paused = session.state === "paused";

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
