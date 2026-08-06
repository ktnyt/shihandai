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

	// かなの成績は単語を打ち終えたときに記録される
	d.Input("あ")
	d.Input("い")
	d.FinishWord(time.Unix(1, 0))
	if s := d.Stats["あ"]; s == nil || s.Errors != 1 || s.Attempts != 1 {
		t.Errorf("Stats[あ] = %+v, want 1 attempt / 1 error", s)
	}
	if s := d.Stats["い"]; s == nil || s.Errors != 0 || s.Attempts != 1 {
		t.Errorf("Stats[い] = %+v, want 1 attempt / 0 error", s)
	}
}

func TestConsecutiveMissesCountOncePerWord(t *testing.T) {
	// 打鍵漏れからの連続入力で同じ位置を何度ミスしても、
	// かなの成績には1回の失敗としてしか残らない
	d := New(DefaultConfig(), 1, nil)
	d.StartWord([]string{"あ", "い"}, time.Unix(0, 0))
	for range 5 {
		d.Input("い") // ミス連打
	}
	d.Input("あ")
	d.Input("い")
	out := d.FinishWord(time.Unix(1, 0))

	if out.Errors != 5 {
		t.Errorf("WordResult.Errors = %d, want 5 (昇格側のミス率はタップ単位のまま)", out.Errors)
	}
	s := d.Stats["あ"]
	if s == nil || s.Errors != 1 || len(s.Recent) != 1 || s.Recent[0] {
		t.Errorf("Stats[あ] = %+v, want 1回の失敗だけ", s)
	}
}

func TestAbandonWordRecordsMisses(t *testing.T) {
	// 打ちかけの単語を捨てても、ミスした位置は失敗として残る
	d := New(DefaultConfig(), 1, nil)
	d.StartWord([]string{"あ", "い"}, time.Unix(0, 0))
	d.Input("い") // ミス
	d.AbandonWord()

	if s := d.Stats["あ"]; s == nil || s.Errors != 1 || len(s.Recent) != 1 {
		t.Errorf("Stats[あ] = %+v, want 1回の失敗", s)
	}
	if s := d.Stats["い"]; s != nil && s.Attempts != 0 {
		t.Errorf("打っていない位置が記録された: %+v", s)
	}

	// もう一度呼んでも二重記録しない
	d.AbandonWord()
	if s := d.Stats["あ"]; s.Errors != 1 {
		t.Errorf("二重記録された: %+v", s)
	}
}

func TestFinishThenAbandonDoesNotDoubleCount(t *testing.T) {
	// 打ち終わった後に破棄が呼ばれても二重記録しない
	d := New(DefaultConfig(), 1, nil)
	d.StartWord([]string{"あ"}, time.Unix(0, 0))
	d.Input("い") // ミス
	d.Input("あ")
	d.FinishWord(time.Unix(1, 0))
	d.AbandonWord()

	if s := d.Stats["あ"]; s.Attempts != 1 || s.Errors != 1 {
		t.Errorf("Stats[あ] = %+v, want 1 attempt / 1 error", s)
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

func TestFinishWordSuccessAndFailure(t *testing.T) {
	tests := []struct {
		name    string
		elapsed time.Duration
		miss    bool
		want    bool
	}{
		{"ノーミスは成功", time.Second, false, true},
		{"遅くてもノーミスなら成功", 10 * time.Second, false, true},
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

	// 「ある」は2打鍵。0.9秒 (133kpm) でノーミスなら
	// kpm もミス率も基準を満たす
	var out WordResult
	for range cfg.WindowSize {
		out = typeWord(d, []string{"あ", "る"}, 900*time.Millisecond)
	}
	if !out.Promoted {
		t.Fatalf("昇格しなかった: %+v (kpm=%.0f miss=%.2f)", out, d.WindowKPM(), d.MissRate())
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

	// 3秒 (2打鍵で 40kpm) はノーミスでも遅すぎる
	for range cfg.WindowSize * 2 {
		if out := typeWord(d, []string{"あ", "る"}, 3*time.Second); out.Promoted {
			t.Fatal("kpm 不足なのに昇格した")
		}
	}
	if got := d.WindowKPM(); got >= cfg.TargetKPM {
		t.Fatalf("前提が崩れた: WindowKPM = %.0f", got)
	}

	// 速い語で窓が入れ替われば昇格する
	promoted := false
	for range cfg.WindowSize {
		if out := typeWord(d, []string{"あ", "る"}, 800*time.Millisecond); out.Promoted {
			promoted = true
			break
		}
	}
	if !promoted {
		t.Fatalf("速くなったのに昇格しない: kpm=%.0f", d.WindowKPM())
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
		if out := d.FinishWord(start.Add(900 * time.Millisecond)); out.Promoted {
			t.Fatal("ミス率が高いのに昇格した")
		}
	}
	if got := d.MissRate(); got <= cfg.MaxMissRate {
		t.Fatalf("前提が崩れた: MissRate = %.2f", got)
	}

	// ノーミスの語で窓が入れ替われば昇格する
	promoted := false
	for range cfg.WindowSize {
		if out := typeWord(d, []string{"あ", "る"}, 900*time.Millisecond); out.Promoted {
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
	d := New(cfg, 5, nil) // レベル5から新出かな「ん」のゲートがかかる

	// 「あい」は新出かな「ん」を含まないので、いくら成功しても昇格しない
	for range 30 {
		if out := typeWord(d, []string{"あ", "い"}, time.Second); out.Promoted {
			t.Fatal("新出かなの語が足りないのに昇格した")
		}
	}
	if d.Level != 5 {
		t.Fatalf("Level = %d, want 5", d.Level)
	}

	// 「ん」を含む語を10語打てば昇格できる
	var out WordResult
	for range cfg.MinNewKanaWords {
		out = typeWord(d, []string{"あ", "ん"}, 900*time.Millisecond)
	}
	if !out.Promoted {
		t.Fatalf("ゲートを満たしたのに昇格しない: %+v (gate=%d)", out, d.NewKanaWords())
	}
}

func TestNoGateOnInitialKana(t *testing.T) {
	// 最初の5文字の段階 (レベル1〜4) は全部が新出なのでゲートなし
	cfg := DefaultConfig()
	cfg.WindowSize = 5
	cfg.MinNewKanaWords = 50
	d := New(cfg, 1, nil)
	if d.GateTarget() != 0 {
		t.Fatalf("GateTarget = %d, want 0", d.GateTarget())
	}

	// 「る」を含まない語だけでも昇格できる
	var out WordResult
	for range cfg.WindowSize {
		out = typeWord(d, []string{"あ", "い"}, 900*time.Millisecond)
	}
	if !out.Promoted {
		t.Fatalf("最初の段階なのにゲートで止まった: %+v", out)
	}
}

func TestPromoteKanaAddedEveryFourLevels(t *testing.T) {
	cfg := DefaultConfig()
	cfg.WindowSize = 2
	cfg.MinNewKanaWords = 0
	d := New(cfg, 4, nil) // 次の昇格でかなが増える

	var out WordResult
	for range cfg.WindowSize {
		out = typeWord(d, []string{"あ", "る"}, 900*time.Millisecond)
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

	// 「あ」をミスした単語を重ねる。記録は単語単位なので、
	// MinAttempts 語で降格判定に届く
	start := time.Unix(0, 0)
	var out WordResult
	for range cfg.MinAttempts {
		d.StartWord([]string{"あ"}, start)
		d.Input("い") // ミス
		d.Input("あ")
		out = d.FinishWord(start.Add(time.Second))
	}

	if !out.Demoted {
		t.Fatalf("全語ミスなのに降格しなかった: %+v", out)
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

func TestSlowWordsDoNotDemote(t *testing.T) {
	cfg := DefaultConfig()
	d := New(cfg, 5, nil)

	// どれだけ遅くても、打ち間違いがなければ成功のままで降格もしない
	for range cfg.MinAttempts * 3 {
		out := typeWord(d, []string{"あ", "い"}, 10*time.Second)
		if !out.Success {
			t.Fatal("ノーミスなのに失敗になった")
		}
		if out.Demoted {
			t.Fatal("遅いだけで降格した")
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
	if got := d.WindowKPM(); got != 120 {
		t.Errorf("WindowKPM = %.0f, want 120 (2打鍵/1秒 × 5語)", got)
	}
}

func TestWindowKPMSubtractsReaction(t *testing.T) {
	cfg := DefaultConfig()
	d := New(cfg, 1, nil)

	// 経過1.5秒のうち反応の猶予0.5秒を引いた1秒が打鍵時間になる
	typeWord(d, []string{"あ", "る"}, 1500*time.Millisecond)
	if got := d.WindowKPM(); got != 120 {
		t.Errorf("WindowKPM = %.0f, want 120 (2打鍵/(1.5s-0.5s))", got)
	}
}
