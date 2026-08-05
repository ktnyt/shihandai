package drill

import (
	"testing"
	"time"

	"github.com/ktnyt/shihandai/internal/curriculum"
)

// typeWord は units の単語を出題し、正しく打ち切って結果を返す。
// elapsed は表示から打ち終わりまでの時間。
func typeWord(d *Drill, units []string, elapsed time.Duration) WordResult {
	start := time.Unix(0, 0)
	d.StartWord(units, start)
	for _, u := range units {
		d.Input(u)
	}
	return d.FinishWord(start.Add(elapsed))
}

func TestInputAdvancesOnMatch(t *testing.T) {
	d := New(DefaultConfig(), 1, nil)
	d.StartWord([]string{"あ", "い"}, time.Unix(0, 0))

	if got := d.Input("あ"); got != ResultAdvance {
		t.Fatalf("Input(あ) = %v, want ResultAdvance", got)
	}
	if got := d.Input("い"); got != ResultWordDone {
		t.Fatalf("Input(い) = %v, want ResultWordDone", got)
	}
}

func TestInputErrorDoesNotAdvance(t *testing.T) {
	d := New(DefaultConfig(), 1, nil)
	d.StartWord([]string{"あ", "い"}, time.Unix(0, 0))

	if got := d.Input("い"); got != ResultError {
		t.Fatalf("Input(い) = %v, want ResultError", got)
	}
	if d.Pos() != 0 {
		t.Errorf("Pos() = %d, want 0", d.Pos())
	}
	if d.WordErrors() != 1 {
		t.Errorf("WordErrors() = %d, want 1", d.WordErrors())
	}
	// ミスは打つべきだったかなに記録される
	if s := d.Stats["あ"]; s == nil || s.Errors != 1 {
		t.Errorf("Stats[あ] = %+v, want 1 error", s)
	}
}

func TestInputIgnoresControlText(t *testing.T) {
	d := New(DefaultConfig(), 1, nil)
	d.StartWord([]string{"あ"}, time.Unix(0, 0))
	for _, text := range []string{" ", "\b", "\n"} {
		if got := d.Input(text); got != ResultIgnored {
			t.Errorf("Input(%q) = %v, want ResultIgnored", text, got)
		}
	}
}

func TestThresholdScalesWithKeys(t *testing.T) {
	d := New(DefaultConfig(), 1, nil)

	// あ(J)+る(I) = 2打鍵。120kpm なら打鍵に1秒、猶予1秒で計2秒
	d.StartWord([]string{"あ", "る"}, time.Unix(0, 0))
	if got := d.Threshold(); got != 2*time.Second {
		t.Errorf("Threshold(ある) = %v, want 2s", got)
	}

	// が(F+J) は2打鍵で1文字
	d.StartWord([]string{"が"}, time.Unix(0, 0))
	if got := d.Threshold(); got != 2*time.Second {
		t.Errorf("Threshold(が) = %v, want 2s", got)
	}
}

func TestFinishWordSuccessAndFailure(t *testing.T) {
	tests := []struct {
		name    string
		elapsed time.Duration
		miss    bool
		want    bool
	}{
		{"時間内ノーミスは成功", time.Second, false, true},
		{"時間超過は失敗", 10 * time.Second, false, false},
		{"ミスがあると失敗", time.Second, true, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := New(DefaultConfig(), 1, nil)
			d.StartWord([]string{"あ", "い"}, time.Unix(0, 0))
			if tt.miss {
				d.Input("い")
			}
			d.Input("あ")
			d.Input("い")
			out := d.FinishWord(time.Unix(0, 0).Add(tt.elapsed))
			if out.Success != tt.want {
				t.Fatalf("Success = %v, want %v (%+v)", out.Success, tt.want, out)
			}
		})
	}
}

func TestPromoteWhenBothThresholdsMet(t *testing.T) {
	cfg := DefaultConfig()
	cfg.WindowSize = 20
	cfg.MinNewKanaWords = 10
	d := New(cfg, 1, nil)

	// 「ある」は2打鍵。1.5秒 (打鍵時間0.5秒 → 4kps) でノーミスなら
	// kps もミス率も基準を満たす
	var out WordResult
	for range cfg.WindowSize {
		out = typeWord(d, []string{"あ", "る"}, 1500*time.Millisecond)
	}
	if !out.Promoted {
		t.Fatalf("昇格しなかった: %+v (kps=%.2f miss=%.2f)", out, d.WindowKPS(), d.MissRate())
	}
	if out.KanaAdded {
		t.Errorf("レベル1→2はかな追加ではなく長さの解放のはず: %+v", out)
	}
	if d.Level != 2 {
		t.Errorf("Level = %d, want 2", d.Level)
	}
	if _, total := d.SuccessCount(); total != 0 {
		t.Errorf("昇格後にカウンターが残っている: %d", total)
	}
	if d.NewKanaWords() != 0 {
		t.Errorf("昇格後にゲートのカウンターが残っている: %d", d.NewKanaWords())
	}
}

func TestNoPromoteWhenKPSTooLow(t *testing.T) {
	cfg := DefaultConfig()
	cfg.WindowSize = 10
	cfg.MinNewKanaWords = 0
	d := New(cfg, 1, nil)

	// 3秒 (打鍵時間2秒 → 2打鍵で 1kps) はノーミスでも遅すぎる
	for range cfg.WindowSize * 2 {
		if out := typeWord(d, []string{"あ", "る"}, 3*time.Second); out.Promoted {
			t.Fatal("kps 不足なのに昇格した")
		}
	}
	if got := d.WindowKPS(); got >= cfg.TargetKPS {
		t.Fatalf("前提が崩れた: WindowKPS = %.2f", got)
	}

	// 速い語で窓が入れ替われば昇格する
	promoted := false
	for range cfg.WindowSize {
		if out := typeWord(d, []string{"あ", "る"}, 1200*time.Millisecond); out.Promoted {
			promoted = true
			break
		}
	}
	if !promoted {
		t.Fatalf("速くなったのに昇格しない: kps=%.2f", d.WindowKPS())
	}
}

func TestNoPromoteWhenMissRateTooHigh(t *testing.T) {
	cfg := DefaultConfig()
	cfg.WindowSize = 10
	cfg.MinNewKanaWords = 0
	d := New(cfg, 1, nil)

	// 速くても毎語1ミスでは ミス率 1/3 で昇格できない
	start := time.Unix(0, 0)
	for range cfg.WindowSize * 2 {
		d.StartWord([]string{"あ", "る"}, start)
		d.Input("い") // ミス
		d.Input("あ")
		d.Input("る")
		if out := d.FinishWord(start.Add(1200 * time.Millisecond)); out.Promoted {
			t.Fatal("ミス率が高いのに昇格した")
		}
	}
	if got := d.MissRate(); got <= cfg.MaxMissRate {
		t.Fatalf("前提が崩れた: MissRate = %.2f", got)
	}

	// ノーミスの語で窓が入れ替われば昇格する
	promoted := false
	for range cfg.WindowSize {
		if out := typeWord(d, []string{"あ", "る"}, 1200*time.Millisecond); out.Promoted {
			promoted = true
			break
		}
	}
	if !promoted {
		t.Fatalf("ミスが直ったのに昇格しない: miss=%.2f", d.MissRate())
	}
}

func TestNoPromoteWithoutEnoughNewKanaWords(t *testing.T) {
	cfg := DefaultConfig()
	cfg.WindowSize = 5
	cfg.MinNewKanaWords = 10
	d := New(cfg, 1, nil)

	// 「あい」は新出かな「る」を含まないので、いくら成功しても昇格しない
	for range 30 {
		if out := typeWord(d, []string{"あ", "い"}, time.Second); out.Promoted {
			t.Fatal("新出かなの語が足りないのに昇格した")
		}
	}
	if d.Level != 1 {
		t.Fatalf("Level = %d, want 1", d.Level)
	}

	// 「る」を含む語を10語打てば昇格できる
	var out WordResult
	for range cfg.MinNewKanaWords {
		out = typeWord(d, []string{"あ", "る"}, 1200*time.Millisecond)
	}
	if !out.Promoted {
		t.Fatalf("ゲートを満たしたのに昇格しない: %+v (gate=%d)", out, d.NewKanaWords())
	}
}

func TestPromoteKanaAddedEveryFourLevels(t *testing.T) {
	cfg := DefaultConfig()
	cfg.WindowSize = 2
	cfg.MinNewKanaWords = 0
	d := New(cfg, 4, nil) // 次の昇格でかなが増える

	var out WordResult
	for range cfg.WindowSize {
		out = typeWord(d, []string{"あ", "る"}, time.Second)
	}
	if !out.Promoted || !out.KanaAdded {
		t.Fatalf("レベル4→5でかな追加になっていない: %+v", out)
	}
	if got := len(d.Allowed()); got != 6 {
		t.Errorf("かな %d文字, want 6", got)
	}
	if st := d.Stage(); st.MaxLen != 2 {
		t.Errorf("かな追加後は2文字語に戻るべき: MaxLen = %d", st.MaxLen)
	}
}

func TestNoPromoteBeforeWindowFilled(t *testing.T) {
	cfg := DefaultConfig()
	cfg.WindowSize = 20
	cfg.MinNewKanaWords = 0
	d := New(cfg, 1, nil)
	for range cfg.WindowSize - 1 {
		if out := typeWord(d, []string{"あ", "る"}, time.Second); out.Promoted {
			t.Fatal("窓が埋まる前に昇格した")
		}
	}
	if d.Level != 1 {
		t.Fatalf("Level = %d, want 1", d.Level)
	}
}

func TestDemoteOnLowAccuracy(t *testing.T) {
	cfg := DefaultConfig()
	d := New(cfg, 5, nil)

	// 「あ」を直近で大量にミスさせる。1語につき試行が2回（ミスと打ち直し）
	// 記録されるので、MinAttempts/2 語で降格判定に届く
	start := time.Unix(0, 0)
	var out WordResult
	for range cfg.MinAttempts / 2 {
		d.StartWord([]string{"あ"}, start)
		d.Input("い") // ミス
		d.Input("あ")
		out = d.FinishWord(start.Add(time.Second))
	}

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
	if out := typeWord(d, []string{"あ", "い"}, time.Second); out.Demoted {
		t.Errorf("降格が連鎖した: %+v", out)
	}
}

func TestNoDemoteOnNewestKana(t *testing.T) {
	cfg := DefaultConfig()
	d := New(cfg, 5, nil)
	allowed := d.Allowed()
	newest := allowed[len(allowed)-1]
	old := allowed[0]

	// 覚えている最中のいちばん新しいかなをいくらミスしても降格しない
	start := time.Unix(0, 0)
	for range cfg.MinAttempts {
		d.StartWord([]string{newest}, start)
		d.Input(old) // ミス
		d.Input(newest)
		if out := d.FinishWord(start.Add(time.Second)); out.Demoted {
			t.Fatalf("新出かな %q のミスで降格した", newest)
		}
	}
	if d.Level != 5 {
		t.Fatalf("Level = %d, want 5", d.Level)
	}
}

func TestTimeoutFailureDoesNotDemote(t *testing.T) {
	cfg := DefaultConfig()
	d := New(cfg, 5, nil)

	// 時間超過の失敗をいくら重ねても、打ち間違いがなければ降格しない
	for range cfg.MinAttempts * 3 {
		out := typeWord(d, []string{"あ", "い"}, 10*time.Second)
		if out.Success {
			t.Fatal("前提が崩れた: 時間超過なのに成功になった")
		}
		if out.Demoted {
			t.Fatal("時間超過で降格した")
		}
	}
	if d.Level != 5 {
		t.Fatalf("Level = %d, want 5", d.Level)
	}
}

func TestNoDemoteBelowLevel1(t *testing.T) {
	cfg := DefaultConfig()
	d := New(cfg, 1, nil)
	start := time.Unix(0, 0)
	var out WordResult
	for range cfg.MinAttempts {
		d.StartWord([]string{"あ"}, start)
		d.Input("い")
		d.Input("あ")
		out = d.FinishWord(start.Add(time.Second))
	}
	if out.Demoted || d.Level != 1 {
		t.Fatalf("レベル1から降格した: %+v, Level = %d", out, d.Level)
	}
}

func TestNoPromoteAboveMaxLevel(t *testing.T) {
	cfg := DefaultConfig()
	cfg.WindowSize = 2
	cfg.MinNewKanaWords = 0
	d := New(cfg, curriculum.MaxLevel(), nil)
	for range cfg.WindowSize + 1 {
		typeWord(d, []string{"あ", "い"}, time.Second)
	}
	if d.Level != curriculum.MaxLevel() {
		t.Fatalf("最大レベルを超えた: Level = %d", d.Level)
	}
}

func TestElapsedMeasuresFromDisplay(t *testing.T) {
	d := New(DefaultConfig(), 1, nil)
	start := time.Unix(0, 0)
	d.StartWord([]string{"あ"}, start)
	if got := d.Elapsed(start.Add(1500 * time.Millisecond)); got != 1500*time.Millisecond {
		t.Errorf("Elapsed = %v, want 1.5s", got)
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

func TestProgressRoundtrip(t *testing.T) {
	cfg := DefaultConfig()
	cfg.WindowSize = 5
	d := New(cfg, 1, nil)
	rec := func(ok bool) WordRecord {
		return WordRecord{Success: ok, Units: 2, Keys: 2, Errors: 0, Typing: time.Second}
	}
	d.SetProgress([]WordRecord{
		rec(true), rec(true), rec(false), rec(true), rec(true), rec(true), rec(true),
	}, 30)

	records, newKana := d.Progress()
	if len(records) != cfg.WindowSize {
		t.Errorf("窓の大きさに切り詰められていない: %d", len(records))
	}
	if newKana != 30 {
		t.Errorf("NewKanaWords = %d, want 30", newKana)
	}
	if successes, total := d.SuccessCount(); total != 5 || successes != 4 {
		t.Errorf("SuccessCount = %d/%d, want 4/5", successes, total)
	}
	if got := d.WindowKPS(); got != 2 {
		t.Errorf("WindowKPS = %.2f, want 2 (2打鍵/1秒 × 5語)", got)
	}
}

func TestWindowKPSSubtractsReaction(t *testing.T) {
	cfg := DefaultConfig()
	d := New(cfg, 1, nil)

	// 経過2秒のうち反応の猶予1秒を引いた1秒が打鍵時間になる
	typeWord(d, []string{"あ", "る"}, 2*time.Second)
	if got := d.WindowKPS(); got != 2 {
		t.Errorf("WindowKPS = %.2f, want 2 (2打鍵/(2s-1s))", got)
	}
}
