package lesson

import (
	"math/rand"
	"slices"
	"strings"
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
		word, err := g.Word(allowed, allowed[len(allowed)-1], nil, 0)
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

func TestWordNewestBias(t *testing.T) {
	cfg := DefaultConfig()
	cfg.NewestRatio = 1 // 常に新出かなを含む語を選ぶ
	cfg.WeakRatio = 0
	g := NewGenerator(cfg, rand.New(rand.NewSource(1)))

	for range 20 {
		word, err := g.Word(curriculum.For(1), "る", nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		if !slices.Contains(word, "る") {
			t.Errorf("新出かな「る」を含まない語が出た: %v", word)
		}
	}
}

func TestWordNewestFrequency(t *testing.T) {
	// 既定の設定でも新出かなを含む語が3割以上出る
	g := newGen(5)
	allowed := curriculum.For(40)
	newest := allowed[len(allowed)-1]
	hit := 0
	for range 200 {
		word, err := g.Word(allowed, newest, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		if slices.Contains(word, newest) {
			hit++
		}
	}
	if hit < 60 {
		t.Errorf("新出かな %q を含む語が %d/200 しか出ていない", newest, hit)
	}
}

func TestWordErrorWhenNoWords(t *testing.T) {
	g := newGen(1)
	if _, err := g.Word([]string{"っ"}, "っ", nil, 0); err == nil {
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
		word, err := g.Word(allowed, "", nil, 0)
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

func TestWordRespectsMaxLen(t *testing.T) {
	g := newGen(3)
	allowed := curriculum.For(30)
	for range 50 {
		word, err := g.Word(allowed, "", nil, 3)
		if err != nil {
			t.Fatal(err)
		}
		if len(word) > 3 {
			t.Fatalf("最大3文字のはずが %v が出た", word)
		}
	}
}

func TestWordNewestFallbackBeyondMaxLen(t *testing.T) {
	// 「ぢょ」を含む語は辞書では6文字が最短なので、新出かなを優先するときは
	// 長さ制限を超えてでも出す
	cfg := DefaultConfig()
	cfg.NewestRatio = 1
	cfg.WeakRatio = 0
	g := NewGenerator(cfg, rand.New(rand.NewSource(1)))
	allowed := curriculum.For(curriculum.MaxLevel())

	word, err := g.Word(allowed, "ぢょ", nil, 3)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Contains(word, "ぢょ") {
		t.Fatalf("新出かなを含まない語が出た: %v", word)
	}
	if len(word) <= 3 {
		t.Fatalf("長さ制限を超える語が出るはずが %v だった", word)
	}
}

func TestWordRandomPairAtTwo(t *testing.T) {
	// 2文字の段階は辞書を引かず、解放済みのかなを組み合わせる
	g := newGen(3)
	allowed := curriculum.For(30)
	for range 50 {
		word, err := g.Word(allowed, "", nil, 2)
		if err != nil {
			t.Fatal(err)
		}
		if len(word) != 2 {
			t.Fatalf("2文字のはずが %v が出た", word)
		}
		for _, u := range word {
			if !slices.Contains(allowed, u) {
				t.Fatalf("使えないかな %q が含まれる: %v", u, word)
			}
		}
	}
}

func TestWordRandomPairIgnoresDict(t *testing.T) {
	// 辞書にない組み合わせも出るので、レベル1でも25通りの多くが顔を出す
	g := newGen(11)
	allowed := curriculum.For(1)
	seen := map[string]bool{}
	for range 200 {
		word, err := g.Word(allowed, "", nil, 2)
		if err != nil {
			t.Fatal(err)
		}
		seen[strings.Join(word, "")] = true
	}
	if len(seen) < 20 {
		t.Errorf("200回で %d 種類しか出ていない (20種類以上出るべき)", len(seen))
	}
}

// dictPairs は allowed で打てる、辞書にある2文字語を集める。
func dictPairs(g *Generator, allowed []string) map[string]bool {
	set := newUnitSet(allowed)
	pairs := map[string]bool{}
	for _, w := range g.words {
		if units, ok := set.segment(w); ok && len(units) == 2 {
			pairs[strings.Join(units, "")] = true
		}
	}
	return pairs
}

func TestWordRandomPairBeyondTwoStage(t *testing.T) {
	// 長さの上限が2でない段階でも、2文字の出題は辞書に縛られない
	g := newGen(5)
	allowed := curriculum.For(30)
	inDict := dictPairs(g, allowed)

	pairs, offDict := 0, 0
	for range 300 {
		word, err := g.Word(allowed, "", nil, 5)
		if err != nil {
			t.Fatal(err)
		}
		if len(word) != 2 {
			continue
		}
		pairs++
		if !inDict[strings.Join(word, "")] {
			offDict++
		}
	}
	if pairs == 0 {
		t.Fatal("2文字の出題が1回もない")
	}
	if offDict == 0 {
		t.Errorf("2文字 %d 回すべてが辞書にある語だった", pairs)
	}
}

func TestWordRandomPairKeepsNewestBeyondTwoStage(t *testing.T) {
	// 組み合わせに置き換えても、新出かなを含む語だったことは保つ
	cfg := DefaultConfig()
	cfg.NewestRatio = 1
	cfg.WeakRatio = 0
	g := NewGenerator(cfg, rand.New(rand.NewSource(5)))
	allowed := curriculum.For(30)
	newest := allowed[len(allowed)-1]

	for range 200 {
		word, err := g.Word(allowed, newest, nil, 5)
		if err != nil {
			t.Fatal(err)
		}
		if !slices.Contains(word, newest) {
			t.Fatalf("新出かな %q を含まない語が出た: %v", newest, word)
		}
	}
}

func TestWordRandomPairNewestBias(t *testing.T) {
	cfg := DefaultConfig()
	cfg.NewestRatio = 1 // 常に新出かなを含める
	cfg.WeakRatio = 0
	g := NewGenerator(cfg, rand.New(rand.NewSource(1)))

	for range 20 {
		word, err := g.Word(curriculum.For(1), "る", nil, 2)
		if err != nil {
			t.Fatal(err)
		}
		if !slices.Contains(word, "る") {
			t.Errorf("新出かな「る」を含まない組み合わせが出た: %v", word)
		}
	}
}

func TestWordRandomPairWeakBias(t *testing.T) {
	cfg := DefaultConfig()
	cfg.NewestRatio = 0
	cfg.WeakRatio = 1 // 常に苦手かなを含める
	g := NewGenerator(cfg, rand.New(rand.NewSource(1)))

	for range 20 {
		word, err := g.Word(curriculum.For(1), "", []string{"す"}, 2)
		if err != nil {
			t.Fatal(err)
		}
		if !slices.Contains(word, "す") {
			t.Errorf("苦手かな「す」を含まない組み合わせが出た: %v", word)
		}
	}
}

func TestWordRandomPairSkipsUnavailableUnits(t *testing.T) {
	// 未解放のかなは newest や weak に混ざっても出題に使わない
	cfg := DefaultConfig()
	cfg.NewestRatio = 0.5
	cfg.WeakRatio = 0.5
	g := NewGenerator(cfg, rand.New(rand.NewSource(1)))
	allowed := curriculum.For(1)

	for range 50 {
		word, err := g.Word(allowed, "ぱ", []string{"ぴょ"}, 2)
		if err != nil {
			t.Fatal(err)
		}
		for _, u := range word {
			if !slices.Contains(allowed, u) {
				t.Fatalf("使えないかな %q が含まれる: %v", u, word)
			}
		}
	}
}

func TestWordVariety(t *testing.T) {
	// 偏りをゆるめたので、少ない語彙のレベル1でも200回で
	// それなりの種類の語が出る
	g := newGen(11)
	allowed := curriculum.For(1)
	seen := map[string]bool{}
	for range 200 {
		word, err := g.Word(allowed, "", nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		key := ""
		for _, u := range word {
			key += u
		}
		seen[key] = true
	}
	if len(seen) < 12 {
		t.Errorf("200回で %d 種類しか出ていない (12種類以上出るべき)", len(seen))
	}
}
