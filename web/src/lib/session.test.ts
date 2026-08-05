import { describe, expect, it } from "vitest";
import { Engine } from "./engine";
import { Drill, DEFAULT_DRILL_CONFIG } from "./drill";
import { Generator } from "./lesson";
import { Session } from "./session";
import { words } from "./dict";
import { chordFor } from "./table";
import { keysOf } from "./keys";

function seededRandom(seed: number): () => number {
  let s = seed;
  return () => {
    s = (s * 1103515245 + 12345) & 0x7fffffff;
    return s / 0x80000000;
  };
}

function makeSession() {
  const clock = { t: 0 };
  const session = new Session(
    new Engine(),
    new Drill({ ...DEFAULT_DRILL_CONFIG }, 1),
    new Generator(words(), undefined, seededRandom(1)),
    {
      intervalMs: 0,
      upcomingWords: 4,
      now: () => clock.t,
      schedule: () => {},
      onChange: () => {},
      onSave: () => {},
    },
  );
  return { session, clock };
}

// いまの単語を正しい打鍵で打ち切る。
function typeCurrentWord(session: Session) {
  for (const unit of [...session.drill.currentWord()]) {
    const keys = keysOf(chordFor(unit)!);
    for (const k of keys) session.keydown(k);
    for (const k of [...keys].reverse()) session.keyup(k);
  }
}

describe("Session", () => {
  it("ロード直後は開始待ちで、キー入力を受け付けない", () => {
    const { session } = makeSession();
    expect(session.state).toBe("ready");

    typeCurrentWord(session);
    expect(session.drill.currentPos()).toBe(0);
    expect(session.engine.presses).toBe(0);
  });

  it("start で計測が始まり、打ち切ると次の単語へ進む", () => {
    const { session, clock } = makeSession();
    clock.t = 5000;
    session.start();
    expect(session.state).toBe("typing");
    expect(session.drill.elapsedMs(5100)).toBe(100); // 開始時点から計測

    typeCurrentWord(session);
    expect(session.drill.successCount().total).toBe(1);
    expect(session.drill.currentPos()).toBe(0); // 次の単語が出ている
  });

  it("開始待ち中は一時停止に落ちない (フォーカス喪失対策)", () => {
    const { session } = makeSession();
    session.pause();
    expect(session.state).toBe("ready");
  });
});

describe("upcomingWords", () => {
  it("先読みの数が設定に追従する", () => {
    const { session } = makeSession();
    expect(session.upcoming.length).toBe(4);

    session.upcomingWords = 2;
    session.start();
    typeCurrentWord(session); // 次の単語で詰め直される
    expect(session.upcoming.length).toBe(2);
  });
});
