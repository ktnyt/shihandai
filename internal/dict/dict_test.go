// lesson が dict を使うため、循環を避けて外部テストパッケージにする。
package dict_test

import (
	"testing"

	"github.com/ktnyt/shihandai/internal/curriculum"
	"github.com/ktnyt/shihandai/internal/dict"
	"github.com/ktnyt/shihandai/internal/lesson"
)

func TestWordsLoaded(t *testing.T) {
	words := dict.Words()
	if len(words) < 10000 {
		t.Fatalf("辞書が小さすぎる: %d 語", len(words))
	}
}

func TestAllWordsTypable(t *testing.T) {
	universe := curriculum.For(curriculum.MaxLevel())
	for _, w := range dict.Words() {
		if _, ok := lesson.Segment(w, universe); !ok {
			t.Errorf("打てない語が辞書にある: %q", w)
		}
	}
}

func TestLevel1WordsExist(t *testing.T) {
	allowed := curriculum.For(1)
	count := 0
	for _, w := range dict.Words() {
		if _, ok := lesson.Segment(w, allowed); ok {
			count++
		}
	}
	if count < 10 {
		t.Fatalf("「あいなする」で打てる語が %d 語しかない", count)
	}
}
