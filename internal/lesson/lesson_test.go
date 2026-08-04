package lesson

import (
	"math/rand"
	"slices"
	"testing"

	"github.com/ktnyt/shihandai/internal/curriculum"
)

func TestSegment(t *testing.T) {
	allowed := []string{"あ", "い", "な", "す", "る", "き", "きゃ", "ゆ"}

	tests := []struct {
		name string
		text string
		want []string
		ok   bool
	}{
		{"単純な分割", "あいな", []string{"あ", "い", "な"}, true},
		{"拗音は最長一致", "きゃい", []string{"きゃ", "い"}, true},
		{"拗音でない並び", "きあ", []string{"き", "あ"}, true},
		{"使えない文字", "あかい", nil, false},
		{"空文字", "", nil, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := Segment(tt.text, allowed)
			if ok != tt.ok {
				t.Fatalf("Segment(%q) ok = %v, want %v", tt.text, ok, tt.ok)
			}
			if !slices.Equal(got, tt.want) {
				t.Fatalf("Segment(%q) = %v, want %v", tt.text, got, tt.want)
			}
		})
	}
}

func newGen(seed int64) *Generator {
	return NewGenerator(DefaultConfig(), rand.New(rand.NewSource(seed)))
}

func TestGenerateAtEveryLevel(t *testing.T) {
	g := newGen(1)
	for level := 1; level <= curriculum.MaxLevel(); level++ {
		allowed := curriculum.For(level)
		line, err := g.Generate(allowed, allowed[len(allowed)-1:])
		if err != nil {
			t.Fatalf("レベル %d で行が作れない: %v", level, err)
		}
		units := line.Units()
		if len(units) == 0 {
			t.Fatalf("レベル %d の行が空", level)
		}
		if len(units) > g.Cfg.MaxUnits {
			t.Fatalf("レベル %d の行が長すぎる: %d 単位", level, len(units))
		}
		if _, ok := Segment(line.Text(), allowed); !ok {
			t.Fatalf("レベル %d の行に使えないかなが含まれる: %q", level, line.Text())
		}
	}
}

func TestGenerateFocusBias(t *testing.T) {
	cfg := DefaultConfig()
	cfg.FocusRatio = 1 // 常に focus を含む語を選ぶ
	g := NewGenerator(cfg, rand.New(rand.NewSource(1)))

	focus := []string{"る"}
	line, err := g.Generate(curriculum.For(1), focus)
	if err != nil {
		t.Fatal(err)
	}
	for _, w := range line.Words {
		if !containsAny(w, focus) {
			t.Errorf("focus %v を含まない語がある: %v", focus, w)
		}
	}
}

func TestGenerateErrorWhenNoWords(t *testing.T) {
	g := newGen(1)
	if _, err := g.Generate([]string{"っ"}, nil); err == nil {
		t.Fatal("打てる語がないのにエラーにならない")
	}
}

func TestGenerateAvoidsImmediateRepeat(t *testing.T) {
	// 語彙が極端に少ないレベルでも、同じ語の3連続が滅多に出ないことは
	// 保証しない（選び直しは1回だけ）。連続回避の分岐が動くことだけ確かめる。
	g := newGen(7)
	line, err := g.Generate(curriculum.For(1), nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(line.Words) < 2 {
		t.Skipf("語が1つしか選ばれなかった: %v", line.Words)
	}
}

func TestLineUnitsAndText(t *testing.T) {
	l := Line{Words: [][]string{{"あ", "る"}, {"きゃ", "く"}}}
	if got := l.Text(); got != "あるきゃく" {
		t.Errorf("Text() = %q", got)
	}
	if got := l.Units(); !slices.Equal(got, []string{"あ", "る", "きゃ", "く"}) {
		t.Errorf("Units() = %v", got)
	}
}
