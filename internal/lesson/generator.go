package lesson

import (
	"fmt"
	"math/rand"
	"slices"
	"strings"
	"time"

	"github.com/ktnyt/shihandai/internal/dict"
)

// Config は出題の調整項目。
type Config struct {
	NewestRatio float64 // 新出かなを含む語を出す割合
	WeakRatio   float64 // 苦手かなを含む語を出す割合
	Skew        float64 // 頻度への偏り。大きいほど高頻度語に寄る
}

// DefaultConfig は既定値を返す。
// 新出かなは練習の主役なので、4割は新出かなを含む語にする。
// Skew 2 は「上位半分の語が約6割を占める」程度のゆるい偏り。
func DefaultConfig() Config {
	return Config{NewestRatio: 0.4, WeakRatio: 0.2, Skew: 2}
}

// recentMemory は連発を抑えるために覚えておく、直近に出した語の数。
const recentMemory = 20

// randomPairLen は辞書を使わずに組み合わせを作る長さ。
// 辞書にある2文字の語は数が限られていて同じ語ばかり出るので、この長さは
// 意味のない組み合わせも出して、かなの運指そのものを練習する。
const randomPairLen = 2

// Generator は辞書から練習する単語を選ぶ。
type Generator struct {
	Cfg    Config
	Rand   *rand.Rand
	words  []string // 頻度順
	recent []string // 直近に出した語。しばらく出にくくする
}

// NewGenerator は埋め込み辞書を使う Generator を作る。
func NewGenerator(cfg Config, rnd *rand.Rand) *Generator {
	if rnd == nil {
		rnd = rand.New(rand.NewSource(time.Now().UnixNano()))
	}
	if cfg.Skew <= 0 {
		cfg.Skew = DefaultConfig().Skew
	}
	if cfg.NewestRatio < 0 || cfg.NewestRatio > 1 {
		cfg.NewestRatio = DefaultConfig().NewestRatio
	}
	if cfg.WeakRatio < 0 || cfg.NewestRatio+cfg.WeakRatio > 1 {
		cfg.WeakRatio = DefaultConfig().WeakRatio
	}
	return &Generator{Cfg: cfg, Rand: rnd, words: dict.Words()}
}

// Word は allowed のかなだけで打てる単語を1つ選ぶ。
// maxLen は単語の最大文字数（単位数）で、0 なら無制限。
// newest は新しく覚えているかな、weak は苦手なかな。新出かなを含む語を
// NewestRatio、苦手かなを含む語を WeakRatio の割合で優先して出す。
// ただし2文字の語だけは辞書を引かず、かなをランダムに組み合わせる。
func (g *Generator) Word(allowed []string, newest string, weak []string, maxLen int) ([]string, error) {
	if len(allowed) == 0 {
		return nil, fmt.Errorf("使えるかながない")
	}
	// 2文字までの段階は、辞書を引かずに毎回かなを組み合わせる
	if maxLen == randomPairLen {
		return g.randomPair(allowed, g.forcedUnit(allowed, newest, weak)), nil
	}

	set := newUnitSet(allowed)

	// 頻度順を保ったまま、打てる語だけに絞る
	var candidates [][]string
	var newestPool [][]string
	var newestLong [][]string // 長さ超過だが新出かなを含む語（緊急用）
	var weakPool [][]string
	for _, w := range g.words {
		units, ok := set.segment(w)
		if !ok {
			continue
		}
		hasNewest := newest != "" && slices.Contains(units, newest)
		if maxLen > 0 && len(units) > maxLen {
			if hasNewest {
				newestLong = append(newestLong, units)
			}
			continue
		}
		candidates = append(candidates, units)
		if hasNewest {
			newestPool = append(newestPool, units)
		} else if containsAny(units, weak) {
			weakPool = append(weakPool, units)
		}
	}
	// 長さの範囲内に新出かなを含む語がなければ、範囲を超えても出す。
	// 新出かなの練習が長さ制限で詰まないようにするため
	if len(newestPool) == 0 {
		newestPool = newestLong
	}
	if len(candidates) == 0 {
		candidates = newestPool
	}
	if len(candidates) == 0 {
		return nil, fmt.Errorf("使えるかな %v で打てる語が辞書にない", allowed)
	}

	pool := candidates
	r := g.Rand.Float64()
	switch {
	case r < g.Cfg.NewestRatio && len(newestPool) > 0:
		pool = newestPool
	case r < g.Cfg.NewestRatio+g.Cfg.WeakRatio && len(weakPool) > 0:
		pool = weakPool
	}

	// 直近に出した語は選び直す。語彙が少ないときは記憶を短くして
	// 選べなくなるのを防ぎ、それでもだめなら重複を受け入れる
	limit := min(recentMemory, len(pool)/2)
	var w []string
	for range 8 {
		w = pool[g.pick(len(pool))]
		if !g.recentlyShown(w, limit) {
			break
		}
	}
	// 選ばれたのが2文字なら、辞書にない組み合わせにも広げる。
	// 選ばれた語が持っていた新出かなや苦手かなは組み合わせにも残す
	if len(w) == randomPairLen {
		return g.randomPair(allowed, keptUnit(w, newest, weak)), nil
	}
	g.remember(w)
	return w, nil
}

// randomPair は allowed のかなを2つ並べる。辞書にある語かどうかは問わない。
// forced が空でなければ、どちらか片方をそのかなにする。
func (g *Generator) randomPair(allowed []string, forced string) []string {
	// 作れる組み合わせの数。片方を固定すると一気に減るので、
	// 記憶をその半分までに縮めて選び直しが空回りするのを防ぐ
	variants := len(allowed) * len(allowed)
	if forced != "" {
		variants = 2 * len(allowed)
	}
	limit := min(recentMemory, variants/2)

	var w []string
	for range 8 {
		w = []string{
			allowed[g.Rand.Intn(len(allowed))],
			allowed[g.Rand.Intn(len(allowed))],
		}
		if forced != "" {
			w[g.Rand.Intn(2)] = forced
		}
		if !g.recentlyShown(w, limit) {
			break
		}
	}
	g.remember(w)
	return w
}

// forcedUnit は組み合わせに必ず入れるかなを選ぶ。新出かなを NewestRatio、
// 苦手かなを WeakRatio の割合で選び、残りは空を返して2つともランダムに任せる。
// 未解放のかなが混ざっていたら使わない。
func (g *Generator) forcedUnit(allowed []string, newest string, weak []string) string {
	switch r := g.Rand.Float64(); {
	case r < g.Cfg.NewestRatio && slices.Contains(allowed, newest):
		return newest
	case r < g.Cfg.NewestRatio+g.Cfg.WeakRatio:
		avail := slices.DeleteFunc(slices.Clone(weak), func(u string) bool {
			return !slices.Contains(allowed, u)
		})
		if len(avail) > 0 {
			return avail[g.Rand.Intn(len(avail))]
		}
	}
	return ""
}

// keptUnit は辞書から選んだ語のうち、組み合わせに置き換えても残すかなを返す。
func keptUnit(units []string, newest string, weak []string) string {
	if newest != "" && slices.Contains(units, newest) {
		return newest
	}
	for _, u := range units {
		if slices.Contains(weak, u) {
			return u
		}
	}
	return ""
}

func (g *Generator) recentlyShown(units []string, limit int) bool {
	recent := g.recent
	if len(recent) > limit {
		recent = recent[len(recent)-limit:]
	}
	return slices.Contains(recent, strings.Join(units, ""))
}

func (g *Generator) remember(units []string) {
	g.recent = append(g.recent, strings.Join(units, ""))
	if len(g.recent) > recentMemory {
		g.recent = g.recent[len(g.recent)-recentMemory:]
	}
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
