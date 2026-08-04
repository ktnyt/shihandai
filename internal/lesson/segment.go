// Package lesson は練習する単語列を辞書から組み立てる。
package lesson

// Segment は text を allowed の単位（拗音は2文字で1単位）に分割する。
// 最長一致で分割し、分割できない文字があれば ok=false を返す。
func Segment(text string, allowed []string) (units []string, ok bool) {
	return newUnitSet(allowed).segment(text)
}

// unitSet は同じ allowed で繰り返し分割するときにマップを使い回す。
type unitSet struct {
	set    map[string]bool
	maxLen int
}

func newUnitSet(allowed []string) unitSet {
	s := unitSet{set: make(map[string]bool, len(allowed))}
	for _, u := range allowed {
		s.set[u] = true
		if n := len([]rune(u)); n > s.maxLen {
			s.maxLen = n
		}
	}
	return s
}

func (s unitSet) segment(text string) (units []string, ok bool) {
	runes := []rune(text)
	for i := 0; i < len(runes); {
		matched := false
		for n := min(s.maxLen, len(runes)-i); n > 0; n-- {
			if s.set[string(runes[i:i+n])] {
				units = append(units, string(runes[i:i+n]))
				i += n
				matched = true
				break
			}
		}
		if !matched {
			return nil, false
		}
	}
	return units, true
}
