package sentence

import (
	"context"
	"fmt"
	"math/rand"
	"strings"
)

// Line は練習1行分の例文。
type Line struct {
	Units  []string // 打鍵単位に分割済みの本文
	Source string   // "llm" または "wordbank"
}

// Text は本文を返す。
func (l Line) Text() string { return strings.Join(l.Units, "") }

// TextGenerator は生の文章を1つ生成する。
// hints には allowed だけで打てる単語の例を渡す。
type TextGenerator interface {
	Generate(ctx context.Context, allowed, hints []string) (string, error)
}

// Config は例文生成の調整項目。
type Config struct {
	Retries  int // LLM生成の再試行回数
	MinUnits int // 例文の最小単位数
	MaxUnits int // 例文の最大単位数
}

// DefaultConfig は既定値を返す。
func DefaultConfig() Config {
	return Config{Retries: 5, MinUnits: 6, MaxUnits: 40}
}

// Generator はLLM生成と単語バンクを組み合わせた例文生成器。
type Generator struct {
	LLM  TextGenerator // nil なら常に単語バンクを使う
	Cfg  Config
	Rand *rand.Rand
}

// Generate は allowed のかなだけで打てる例文を1行生成する。
// LLMの出力が検証を通らなければ再試行し、だめなら単語バンクから組み立てる。
func (g *Generator) Generate(ctx context.Context, allowed []string) (Line, error) {
	if g.LLM != nil {
		hints := g.hintWords(allowed, 12)
		for range max(g.Cfg.Retries, 1) {
			if ctx.Err() != nil {
				break
			}
			text, err := g.LLM.Generate(ctx, allowed, hints)
			if err != nil {
				break // 接続エラー等はフォールバックに切り替える
			}
			units, ok := Segment(Normalize(text), allowed)
			if ok && len(units) >= g.Cfg.MinUnits && len(units) <= g.Cfg.MaxUnits {
				return Line{Units: units, Source: "llm"}, nil
			}
		}
	}
	return g.fromWordbank(allowed)
}

// hintWords は allowed だけで打てる単語を最大 n 個選ぶ。
func (g *Generator) hintWords(allowed []string, n int) []string {
	var words []string
	for _, w := range wordbank {
		if _, ok := Segment(w, allowed); ok {
			words = append(words, w)
		}
	}
	g.Rand.Shuffle(len(words), func(i, j int) { words[i], words[j] = words[j], words[i] })
	return words[:min(n, len(words))]
}

// fromWordbank は単語バンクから打てる語を選んでつなげる。
func (g *Generator) fromWordbank(allowed []string) (Line, error) {
	type entry struct{ units []string }
	var candidates []entry
	for _, w := range wordbank {
		if units, ok := Segment(w, allowed); ok {
			candidates = append(candidates, entry{units})
		}
	}
	if len(candidates) == 0 {
		return Line{}, fmt.Errorf("使えるかな %v で打てる語が単語バンクにない", allowed)
	}

	// 長すぎる行は練習しにくいので上限の半分程度に抑える
	target := min(
		g.Cfg.MinUnits+g.Rand.Intn(g.Cfg.MaxUnits-g.Cfg.MinUnits+1),
		g.Cfg.MaxUnits/2)

	var units []string
	for len(units) < target {
		w := candidates[g.Rand.Intn(len(candidates))]
		if len(units) > 0 && len(units)+len(w.units) > g.Cfg.MaxUnits {
			break
		}
		units = append(units, w.units...)
	}
	return Line{Units: units, Source: "wordbank"}, nil
}
