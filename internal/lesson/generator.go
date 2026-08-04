package lesson

import (
	"fmt"
	"math/rand"
	"slices"
	"time"

	"github.com/ktnyt/shihandai/internal/dict"
)

// Config は出題の調整項目。
type Config struct {
	FocusRatio float64 // 新出・苦手のかなを含む語を優先する割合
	Skew       float64 // 頻度への偏り。大きいほど高頻度語に寄る
}

// DefaultConfig は既定値を返す。
func DefaultConfig() Config {
	return Config{FocusRatio: 0.5, Skew: 6}
}

// Generator は辞書から練習する単語を選ぶ。
type Generator struct {
	Cfg   Config
	Rand  *rand.Rand
	words []string // 頻度順
	last  []string // 直前に出した単語。連続を避ける
}

// NewGenerator は埋め込み辞書を使う Generator を作る。
func NewGenerator(cfg Config, rnd *rand.Rand) *Generator {
	if rnd == nil {
		rnd = rand.New(rand.NewSource(time.Now().UnixNano()))
	}
	if cfg.Skew <= 0 {
		cfg.Skew = DefaultConfig().Skew
	}
	if cfg.FocusRatio < 0 || cfg.FocusRatio > 1 {
		cfg.FocusRatio = DefaultConfig().FocusRatio
	}
	return &Generator{Cfg: cfg, Rand: rnd, words: dict.Words()}
}

// Word は allowed のかなだけで打てる単語を1つ選ぶ。
// focus には新しく覚えるかなや苦手なかなを渡す。それを含む語が優先される。
func (g *Generator) Word(allowed, focus []string) ([]string, error) {
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
		return nil, fmt.Errorf("使えるかな %v で打てる語が辞書にない", allowed)
	}

	pool := candidates
	if len(focused) > 0 && g.Rand.Float64() < g.Cfg.FocusRatio {
		pool = focused
	}
	w := pool[g.pick(len(pool))]
	// 同じ語が続くと練習にならないので選び直す（1回だけ）
	if slices.Equal(w, g.last) {
		w = pool[g.pick(len(pool))]
	}
	g.last = w
	return w, nil
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
