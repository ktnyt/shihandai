package drill

import (
	"testing"
	"time"

	"github.com/ktnyt/shihandai/internal/curriculum"
)

// typeLine は units の行を interval 間隔で正しく打ち切り、結果を返す。
// keys は1単位1打鍵とみなす。
func typeLine(d *Drill, units []string, interval time.Duration) Outcome {
	d.StartLine(units)
	now := time.Unix(0, 0)
	keys := 0
	for _, u := range units {
		keys++
		d.MarkKeydown(now, keys)
		d.Input(u)
		now = now.Add(interval)
	}
	return d.FinishLine(now.Add(-interval), keys)
}

func TestInputAdvancesOnMatch(t *testing.T) {
	d := New(DefaultConfig(), 1, nil)
	d.StartLine([]string{"あ", "い"})

	if got := d.Input("あ"); got != ResultAdvance {
		t.Fatalf("Input(あ) = %v, want ResultAdvance", got)
	}
	if got := d.Input("い"); got != ResultLineDone {
		t.Fatalf("Input(い) = %v, want ResultLineDone", got)
	}
}

func TestInputErrorDoesNotAdvance(t *testing.T) {
	d := New(DefaultConfig(), 1, nil)
	d.StartLine([]string{"あ", "い"})

	if got := d.Input("い"); got != ResultError {
		t.Fatalf("Input(い) = %v, want ResultError", got)
	}
	if d.Pos() != 0 {
		t.Errorf("Pos() = %d, want 0", d.Pos())
	}
	if d.LineErrors() != 1 {
		t.Errorf("LineErrors() = %d, want 1", d.LineErrors())
	}
	// ミスは打つべきだったかなに記録される
	if s := d.Stats["あ"]; s == nil || s.Errors != 1 {
		t.Errorf("Stats[あ] = %+v, want 1 error", s)
	}
}

func TestInputIgnoresControlText(t *testing.T) {
	d := New(DefaultConfig(), 1, nil)
	d.StartLine([]string{"あ"})
	for _, text := range []string{" ", "\b", "\n"} {
		if got := d.Input(text); got != ResultIgnored {
			t.Errorf("Input(%q) = %v, want ResultIgnored", text, got)
		}
	}
}

func TestPromoteOnFastCleanLine(t *testing.T) {
	d := New(DefaultConfig(), 1, nil)
	// 1打/250ms = 240kpm でノーミス
	out := typeLine(d, []string{"あ", "い", "な", "す", "る", "あ", "い", "な"}, 250*time.Millisecond)

	if !out.Promoted {
		t.Fatalf("昇格しなかった: %+v", out)
	}
	if d.Level != 2 {
		t.Errorf("Level = %d, want 2", d.Level)
	}
}

func TestNoPromoteWhenSlow(t *testing.T) {
	d := New(DefaultConfig(), 1, nil)
	// 1打/1s = 60kpm
	out := typeLine(d, []string{"あ", "い", "な", "す", "る"}, time.Second)

	if out.Promoted {
		t.Fatalf("60kpm で昇格してはいけない: %+v", out)
	}
}

func TestNoPromoteWithErrors(t *testing.T) {
	d := New(DefaultConfig(), 1, nil)
	d.StartLine([]string{"あ", "い"})
	now := time.Unix(0, 0)
	d.MarkKeydown(now, 1)
	d.Input("い") // ミス
	d.Input("あ")
	d.Input("い")
	out := d.FinishLine(now.Add(500*time.Millisecond), 3)

	if out.Promoted {
		t.Fatalf("ミスがあるのに昇格した: %+v", out)
	}
	if out.Errors != 1 {
		t.Errorf("Errors = %d, want 1", out.Errors)
	}
}

func TestDemoteOnLowAccuracy(t *testing.T) {
	cfg := DefaultConfig()
	d := New(cfg, 5, nil)

	// 「あ」を直近で大量にミスさせる
	d.StartLine(makeLine("あ", cfg.MinAttempts))
	for range cfg.MinAttempts {
		d.Input("い") // 全部ミス
		d.Input("あ") // 打ち直して進む
	}
	out := d.FinishLine(time.Unix(60, 0), cfg.MinAttempts*2)

	if !out.Demoted {
		t.Fatalf("正答率50%%で降格しなかった: %+v", out)
	}
	if out.WeakUnit != "あ" {
		t.Errorf("WeakUnit = %q, want あ", out.WeakUnit)
	}
	if d.Level != 4 {
		t.Errorf("Level = %d, want 4", d.Level)
	}
	// 降格後は直近の記録が捨てられ、連鎖降格しない
	out = typeLine(d, []string{"あ", "い"}, time.Second)
	if out.Demoted {
		t.Errorf("降格が連鎖した: %+v", out)
	}
}

func TestNoDemoteBelowLevel1(t *testing.T) {
	cfg := DefaultConfig()
	d := New(cfg, 1, nil)
	d.StartLine(makeLine("あ", cfg.MinAttempts))
	for range cfg.MinAttempts {
		d.Input("い")
		d.Input("あ")
	}
	out := d.FinishLine(time.Unix(60, 0), cfg.MinAttempts*2)
	if out.Demoted || d.Level != 1 {
		t.Fatalf("レベル1から降格した: %+v, Level = %d", out, d.Level)
	}
}

func TestNoPromoteAboveMaxLevel(t *testing.T) {
	d := New(DefaultConfig(), curriculum.MaxLevel(), nil)
	out := typeLine(d, []string{"あ", "い", "な", "す"}, 100*time.Millisecond)
	if out.Promoted || d.Level != curriculum.MaxLevel() {
		t.Fatalf("最大レベルを超えた: %+v, Level = %d", out, d.Level)
	}
}

func TestKPM(t *testing.T) {
	d := New(DefaultConfig(), 1, nil)
	d.StartLine([]string{"あ", "い"})
	start := time.Unix(0, 0)
	d.MarkKeydown(start, 1)

	// 30秒で60打 → 120kpm (最初の1打も含む)
	if got := d.KPM(start.Add(30*time.Second), 60); got != 120 {
		t.Errorf("KPM = %v, want 120", got)
	}
}

func TestRecentAccuracyWindow(t *testing.T) {
	s := &UnitStat{}
	for range recentWindow {
		s.record(false)
	}
	for range recentWindow {
		s.record(true)
	}
	if got := s.RecentAccuracy(); got != 1 {
		t.Errorf("RecentAccuracy() = %v, want 1 (古いミスは窓の外)", got)
	}
	if s.Attempts != recentWindow*2 {
		t.Errorf("Attempts = %d, want %d", s.Attempts, recentWindow*2)
	}
}

func makeLine(unit string, n int) []string {
	line := make([]string, n)
	for i := range line {
		line[i] = unit
	}
	return line
}
