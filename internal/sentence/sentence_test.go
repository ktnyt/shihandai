package sentence

import (
	"context"
	"errors"
	"math/rand"
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
			if len(got) != len(tt.want) {
				t.Fatalf("Segment(%q) = %v, want %v", tt.text, got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Fatalf("Segment(%q) = %v, want %v", tt.text, got, tt.want)
				}
			}
		})
	}
}

func TestNormalize(t *testing.T) {
	tests := []struct{ in, want string }{
		{"ある。", "ある"},
		{"あい、する！", "あいする"},
		{"アイス", "あいす"},
		{"ある ない\n", "あるない"},
	}
	for _, tt := range tests {
		if got := Normalize(tt.in); got != tt.want {
			t.Errorf("Normalize(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

// fakeLLM は決められた応答を順に返す。
type fakeLLM struct {
	responses []string
	err       error
	calls     int
}

func (f *fakeLLM) Generate(ctx context.Context, allowed, hints []string) (string, error) {
	if f.err != nil {
		return "", f.err
	}
	i := min(f.calls, len(f.responses)-1)
	f.calls++
	return f.responses[i], nil
}

func newGen(llm TextGenerator) *Generator {
	return &Generator{
		LLM:  llm,
		Cfg:  DefaultConfig(),
		Rand: rand.New(rand.NewSource(1)),
	}
}

func TestGeneratorUsesValidLLMOutput(t *testing.T) {
	allowed := []string{"あ", "い", "な", "す", "る"}
	g := newGen(&fakeLLM{responses: []string{"あいするすなある。"}})

	line, err := g.Generate(context.Background(), allowed)
	if err != nil {
		t.Fatal(err)
	}
	if line.Source != "llm" {
		t.Errorf("Source = %q, want llm", line.Source)
	}
	if line.Text() != "あいするすなある" {
		t.Errorf("Text() = %q", line.Text())
	}
}

func TestGeneratorRetriesInvalidOutput(t *testing.T) {
	allowed := []string{"あ", "い", "な", "す", "る"}
	llm := &fakeLLM{responses: []string{"漢字が混ざる", "あるないするいな"}}
	g := newGen(llm)

	line, err := g.Generate(context.Background(), allowed)
	if err != nil {
		t.Fatal(err)
	}
	if line.Source != "llm" || llm.calls != 2 {
		t.Errorf("Source = %q, calls = %d, want llm after 2 calls", line.Source, llm.calls)
	}
}

func TestGeneratorFallsBackToWordbank(t *testing.T) {
	allowed := []string{"あ", "い", "な", "す", "る"}

	tests := []struct {
		name string
		llm  TextGenerator
	}{
		{"生成が全部不正", &fakeLLM{responses: []string{"ここは漢字だ"}}},
		{"接続エラー", &fakeLLM{err: errors.New("connection refused")}},
		{"LLMなし", nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := newGen(tt.llm)
			line, err := g.Generate(context.Background(), allowed)
			if err != nil {
				t.Fatal(err)
			}
			if line.Source != "wordbank" {
				t.Errorf("Source = %q, want wordbank", line.Source)
			}
			if len(line.Units) < 4 {
				t.Errorf("単語バンクの行が短すぎる: %v", line.Units)
			}
			if _, ok := Segment(line.Text(), allowed); !ok {
				t.Errorf("単語バンクの行に使えないかなが含まれる: %q", line.Text())
			}
		})
	}
}

func TestWordbankAvailableAtEveryLevel(t *testing.T) {
	// どのレベルでも単語バンクから例文が作れることを確かめる
	g := newGen(nil)
	for level := 1; level <= curriculum.MaxLevel(); level++ {
		line, err := g.Generate(context.Background(), curriculum.For(level))
		if err != nil {
			t.Fatalf("レベル %d で例文が作れない: %v", level, err)
		}
		if len(line.Units) == 0 {
			t.Fatalf("レベル %d の例文が空", level)
		}
	}
}
