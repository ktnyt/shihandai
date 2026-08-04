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

func TestWordAtEveryLevel(t *testing.T) {
	g := newGen(1)
	for level := 1; level <= curriculum.MaxLevel(); level++ {
		allowed := curriculum.For(level)
		word, err := g.Word(allowed, allowed[len(allowed)-1:])
		if err != nil {
			t.Fatalf("レベル %d で単語が選べない: %v", level, err)
		}
		if len(word) == 0 {
			t.Fatalf("レベル %d の単語が空", level)
		}
		for _, u := range word {
			if !slices.Contains(allowed, u) {
				t.Fatalf("レベル %d の単語に使えないかな %q が含まれる: %v", level, u, word)
			}
		}
	}
}

func TestWordFocusBias(t *testing.T) {
	cfg := DefaultConfig()
	cfg.FocusRatio = 1 // 常に focus を含む語を選ぶ
	g := NewGenerator(cfg, rand.New(rand.NewSource(1)))

	focus := []string{"る"}
	for range 20 {
		word, err := g.Word(curriculum.For(1), focus)
		if err != nil {
			t.Fatal(err)
		}
		if !containsAny(word, focus) {
			t.Errorf("focus %v を含まない語が出た: %v", focus, word)
		}
	}
}

func TestWordErrorWhenNoWords(t *testing.T) {
	g := newGen(1)
	if _, err := g.Word([]string{"っ"}, nil); err == nil {
		t.Fatal("打てる語がないのにエラーにならない")
	}
}

func TestWordAvoidsImmediateRepeat(t *testing.T) {
	// 完全な保証はない（選び直しは1回だけ）が、豊富な語彙では
	// 同じ語が続かないことがほとんどであることを確かめる
	g := newGen(7)
	allowed := curriculum.For(30)
	repeats := 0
	var last []string
	for range 100 {
		word, err := g.Word(allowed, nil)
		if err != nil {
			t.Fatal(err)
		}
		if slices.Equal(word, last) {
			repeats++
		}
		last = word
	}
	if repeats > 5 {
		t.Errorf("同じ語の連続が %d/100 回もある", repeats)
	}
}
