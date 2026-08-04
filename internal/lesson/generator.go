package lesson

import (
	"fmt"
	"math/rand"
	"slices"
	"strings"
	"time"

	"github.com/ktnyt/shihandai/internal/dict"
)

// Line は練習1行分の単語列。
type Line struct {
	Words [][]string // 単語ごとの打鍵単位
}

// Units は行全体の打鍵単位を返す。単語の区切りに打鍵は要らない。
func (l Line) Units() []string {
	var units []string
	for _, w := range l.Words {
		units = append(units, w...)
	}
	return units
}

// Text は本文を返す。
func (l Line) Text() string {
	var b strings.Builder
	for _, w := range l.Words {
		b.WriteString(strings.Join(w, ""))
	}
	return b.String()
}

// Config は行の組み立ての調整項目。
type Config struct {
	MinUnits   int     // 1行の最小単位数
	MaxUnits   int     // 1行の最大単位数
	FocusRatio float64 // 新出・苦手のかなを含む語を優先する割合
	Skew       float64 // 頻度への偏り。大きいほど高頻度語に寄る
}

// DefaultConfig は既定値を返す。
func DefaultConfig() Config {
	return Config{MinUnits: 10, MaxUnits: 24, FocusRatio: 0.5, Skew: 6}
}

// Generator は辞書から練習行を組み立てる。
type Generator struct {
	Cfg   Config
	Rand  *rand.Rand
	words []string // 頻度順
}

// NewGenerator は埋め込み辞書を使う Generator を作る。
func NewGenerator(cfg Config, rnd *rand.Rand) *Generator {
	if rnd == nil {
		rnd = rand.New(rand.NewSource(time.Now().UnixNano()))
	}
	if cfg.MinUnits < 1 || cfg.MaxUnits < cfg.MinUnits {
		cfg = DefaultConfig()
	}
	if cfg.Skew <= 0 {
		cfg.Skew = DefaultConfig().Skew
	}
	return &Generator{Cfg: cfg, Rand: rnd, words: dict.Words()}
}

// Generate は allowed のかなだけで打てる単語を並べて1行作る。
// focus には新しく覚えるかなや苦手なかなを渡す。それを含む語が優先される。
func (g *Generator) Generate(allowed, focus []string) (Line, error) {
	set := newUnitSet(allowed)

	// 頻度順を保ったまま、打てる語だけに絞る
	var candidates [][]string
	var focused [][]string
	for _, w := range g.words {
		units, ok := set.segment(w)
		if !ok {
			continue
		}
		candidates = append(candidates, units)
		if containsAny(units, focus) {
			focused = append(focused, units)
		}
	}
	if len(candidates) == 0 {
		return Line{}, fmt.Errorf("使えるかな %v で打てる語が辞書にない", allowed)
	}

	target := g.Cfg.MinUnits + g.Rand.Intn(g.Cfg.MaxUnits-g.Cfg.MinUnits+1)
	var line Line
	total := 0
	for total < target {
		pool := candidates
		if len(focused) > 0 && g.Rand.Float64() < g.Cfg.FocusRatio {
			pool = focused
		}
		w := pool[g.pick(len(pool))]
		// 同じ語が続くと練習にならないので選び直す（1回だけ）
		if n := len(line.Words); n > 0 && slices.Equal(line.Words[n-1], w) {
			w = pool[g.pick(len(pool))]
		}
		if total > 0 && total+len(w) > g.Cfg.MaxUnits {
			break
		}
		line.Words = append(line.Words, w)
		total += len(w)
	}
	return line, nil
}

// pick は頻度上位に偏った添字を選ぶ。
func (g *Generator) pick(n int) int {
	idx := int(g.Rand.ExpFloat64() / g.Cfg.Skew * float64(n))
	if idx >= n {
		idx = n - 1
	}
	return idx
}

func containsAny(units, targets []string) bool {
	for _, u := range units {
		if slices.Contains(targets, u) {
			return true
		}
	}
	return false
}
