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

func TestForGrowsByOne(t *testing.T) {
	for level := 1; level < MaxLevel(); level++ {
		a, b := For(level), For(level+1)
		if len(b) != len(a)+1 {
			t.Fatalf("レベル %d→%d でかなが %d→%d に増えた (1ずつ増えるべき)",
				level, level+1, len(a), len(b))
		}
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
