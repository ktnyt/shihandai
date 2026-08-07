package lesson

import (
	"fmt"
	"math/rand"
	"slices"
	"strings"
	"time"

	"github.com/ktnyt/shihandai/internal/curriculum"
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

// Generator は辞書から練習する課題を選ぶ。
type Generator struct {
	Cfg   Config
	Rand  *rand.Rand
	words []string // 頻度順
	all   []string // 配列にあるかな。断片の切れ目を決めるのに使う

	units [][]string         // 語をかな単位に分けたもの
	grams map[int][][]string // 長さごとの断片。辞書全体から集める
	key   string             // avail を作ったときの条件
	avail map[int][][]string // いま使えるかなで打てる断片

	recent []string // 直近に出した課題。しばらく出にくくする
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
	return &Generator{
		Cfg:   cfg,
		Rand:  rnd,
		words: dict.Words(),
		all:   curriculum.All(),
		grams: map[int][][]string{},
	}
}

// Word は allowed のかなだけで打てる課題を1つ選ぶ。
// maxLen は課題の長さの上限（拗音は2文字で1単位）で、0 なら辞書の単語を
// そのまま出す。newest は新出かな、weak は苦手なかな。新出かなを含む課題を
// NewestRatio、苦手かなを含む課題を WeakRatio の割合で優先して出す。
func (g *Generator) Word(allowed []string, newest string, weak []string, maxLen int) ([]string, error) {
	if len(allowed) == 0 {
		return nil, fmt.Errorf("使えるかながない")
	}
	if maxLen > 0 {
		return g.fromGrams(allowed, newest, weak, maxLen), nil
	}
	return g.fromWords(allowed, newest, weak)
}

// fromGrams は辞書に現れるかなの並びから課題を選ぶ。単語として成り立たない
// 断片でもよい。2文字はかなを組み合わせて作り、3文字以上は辞書の断片から
// 一様に選ぶ。長さそのものを重みにして、長い断片ほど当たりやすくする。
func (g *Generator) fromGrams(allowed []string, newest string, weak []string, maxLen int) []string {
	forced := g.forcedUnit(allowed, newest, weak)
	pools := g.available(allowed, maxLen)
	if forced != "" {
		pools = withUnit(pools, forced)
	}

	// 2文字はかなを組み合わせて作るので、絞り込んでも必ず出せる
	length := g.pickLength(pools, maxLen)
	if length == randomPairLen {
		return g.randomPair(allowed, forced)
	}

	pool := pools[length]
	limit := min(recentMemory, len(pool)/2)
	var w []string
	for range 8 {
		w = pool[g.Rand.Intn(len(pool))]
		if !g.recentlyShown(w, limit) {
			break
		}
	}
	g.remember(w)
	return w
}

// pickLength は出す課題の長さを選ぶ。断片がない長さは飛ばす。
func (g *Generator) pickLength(pools map[int][][]string, maxLen int) int {
	total := 0
	for n := randomPairLen; n <= maxLen; n++ {
		if n == randomPairLen || len(pools[n]) > 0 {
			total += n
		}
	}
	r := g.Rand.Intn(total)
	for n := randomPairLen; n <= maxLen; n++ {
		if n != randomPairLen && len(pools[n]) == 0 {
			continue
		}
		if r < n {
			return n
		}
		r -= n
	}
	return randomPairLen
}

// available はいま使えるかなだけで打てる断片を長さごとに集める。
// 同じレベルのあいだは結果が変わらないので、条件が変わったときだけ作り直す。
func (g *Generator) available(allowed []string, maxLen int) map[int][][]string {
	key := fmt.Sprintf("%d\x00%s", maxLen, strings.Join(allowed, "\x00"))
	if g.key == key {
		return g.avail
	}
	set := make(map[string]bool, len(allowed))
	for _, u := range allowed {
		set[u] = true
	}
	avail := make(map[int][][]string, maxLen)
	for n := randomPairLen + 1; n <= maxLen; n++ {
		for _, gram := range g.gramsOf(n) {
			if typable(gram, set) {
				avail[n] = append(avail[n], gram)
			}
		}
	}
	g.key, g.avail = key, avail
	return avail
}

// gramsOf は辞書に現れる長さ n のかなの並びを重複なく集める。
func (g *Generator) gramsOf(n int) [][]string {
	if got, ok := g.grams[n]; ok {
		return got
	}
	seen := make(map[string]bool)
	var out [][]string
	for _, units := range g.wordUnits() {
		for i := 0; i+n <= len(units); i++ {
			gram := units[i : i+n : i+n]
			key := strings.Join(gram, "\x00")
			if seen[key] {
				continue
			}
			seen[key] = true
			out = append(out, gram)
		}
	}
	g.grams[n] = out
	return out
}

// wordUnits は辞書の語をかな単位に分けて返す。拗音の切れ目は配列のかな
// 全体で決めるので、使えるかなが増えても分けかたは変わらない。
func (g *Generator) wordUnits() [][]string {
	if g.units != nil {
		return g.units
	}
	set := newUnitSet(g.all)
	units := make([][]string, 0, len(g.words))
	for _, w := range g.words {
		if u, ok := set.segment(w); ok {
			units = append(units, u)
		}
	}
	g.units = units
	return units
}

// withUnit は指定したかなを含む断片だけに絞る。
func withUnit(pools map[int][][]string, unit string) map[int][][]string {
	out := make(map[int][][]string, len(pools))
	for n, pool := range pools {
		for _, gram := range pool {
			if slices.Contains(gram, unit) {
				out[n] = append(out[n], gram)
			}
		}
	}
	return out
}

func typable(gram []string, set map[string]bool) bool {
	for _, u := range gram {
		if !set[u] {
			return false
		}
	}
	return true
}

// fromWords は辞書の単語をそのまま選ぶ。長さの制限がない最終段階で使う。
// ただし選ばれたのが2文字なら、辞書にない組み合わせにも広げる。
func (g *Generator) fromWords(allowed []string, newest string, weak []string) ([]string, error) {
	set := newUnitSet(allowed)

	// 頻度順を保ったまま、打てる語だけに絞る
	var candidates [][]string
	var newestPool [][]string
	var weakPool [][]string
	for _, w := range g.words {
		units, ok := set.segment(w)
		if !ok {
			continue
		}
		candidates = append(candidates, units)
		if newest != "" && slices.Contains(units, newest) {
			newestPool = append(newestPool, units)
		} else if containsAny(units, weak) {
			weakPool = append(weakPool, units)
		}
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
