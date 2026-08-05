package curriculum

import (
	"testing"

	"github.com/ktnyt/shihandai/internal/naginata"
)

func TestForStartsWithAinasuru(t *testing.T) {
	got := For(1)
	want := []string{"あ", "い", "な", "す", "る"}
	if len(got) != len(want) {
		t.Fatalf("For(1) = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("For(1) = %v, want %v", got, want)
		}
	}
}

func TestStageLengthProgression(t *testing.T) {
	// かな1文字ごとに 2→3→4→5 文字の4段階を踏む
	tests := []struct {
		level    int
		kana     int
		maxLen   int
	}{
		{1, 5, 2},
		{2, 5, 3},
		{3, 5, 4},
		{4, 5, 5},
		{5, 6, 2}, // 新しいかなが入って2文字に戻る
		{6, 6, 3},
		{9, 7, 2},
	}
	for _, tt := range tests {
		st := StageFor(tt.level)
		if len(st.Units) != tt.kana || st.MaxLen != tt.maxLen {
			t.Errorf("StageFor(%d) = %d文字/最大%d, want %d文字/最大%d",
				tt.level, len(st.Units), st.MaxLen, tt.kana, tt.maxLen)
		}
	}
}

func TestStageKanaGrowsEveryFourLevels(t *testing.T) {
	for level := 1; level < MaxLevel(); level++ {
		a, b := len(For(level)), len(For(level+1))
		wantGrow := level%4 == 0
		if wantGrow && b != a+1 {
			t.Fatalf("レベル %d→%d でかなが %d→%d (1増えるべき)", level, level+1, a, b)
		}
		if !wantGrow && b != a {
			t.Fatalf("レベル %d→%d でかなが %d→%d (変わらないべき)", level, level+1, a, b)
		}
	}
}

func TestFinalStageUnlocksAllLengths(t *testing.T) {
	st := StageFor(MaxLevel())
	if st.MaxLen != 0 {
		t.Errorf("最終レベルの MaxLen = %d, want 0 (全開放)", st.MaxLen)
	}
	if len(st.Units) != len(For(MaxLevel())) {
		t.Errorf("最終レベルで全かなが解放されていない")
	}
}

func TestForClampsRange(t *testing.T) {
	if len(For(0)) != 5 {
		t.Errorf("For(0) はレベル1と同じであるべき")
	}
	if len(For(MaxLevel()+10)) != len(For(MaxLevel())) {
		t.Errorf("For(MaxLevel()+10) は最大レベルと同じであるべき")
	}
}

func TestAllUnitsTypable(t *testing.T) {
	seen := map[string]bool{}
	for _, g := range Groups {
		for _, u := range g.Units {
			if seen[u] {
				t.Errorf("かな %q が重複している", u)
			}
			seen[u] = true
			if _, ok := naginata.ChordFor(u); !ok {
				t.Errorf("かな %q の打鍵が配列テーブルにない", u)
			}
		}
	}
}
